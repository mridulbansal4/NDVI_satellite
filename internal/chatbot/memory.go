package chatbot

import (
	"sync"
	"time"
)

// Message is one conversation turn.
type Message struct {
	Role    string `json:"role"` // "user" | "assistant"
	Content string `json:"content"`
}

// Memory is the in-process, per-session chat history (§10.11).
//
// Mutex-guarded, exactly as the Python is. History is capped at
// CHATBOT_MAX_HISTORY and trimmed from the FRONT one message at a time, so an
// odd cap can leave the history starting on an assistant turn — that is the
// existing behaviour and is preserved.
//
// This is deliberately process-local: the Python keeps it in a module-level
// dict, so a restart clears every conversation and a second replica has its
// own. Making it shared would be a behaviour change.
//
// The Python's per-session cap bounded each conversation but nothing bounded
// the NUMBER of conversations, and session_id comes straight off the request —
// so the map grew forever, both under abuse and under ordinary use (the
// frontend mints a fresh UUID per field selection). This adds the same TTL +
// LRU bound internal/pipeline's ExprCache already uses. Eviction is invisible
// to a live conversation: an evicted session simply starts again with no
// history, which is exactly what a process restart already does.
type Memory struct {
	mu       sync.Mutex
	sessions map[string]*session
	max      int

	ttl         time.Duration
	maxSessions int
	counter     uint64
	// now is injectable so the eviction tests do not sleep.
	now func() time.Time
}

type session struct {
	messages []Message
	// order is a monotonic stamp for LRU, bumped on every read AND write: a
	// session being actively read is live even if the user has not sent
	// anything for a while.
	order    uint64
	lastSeen time.Time
}

// Defaults used when the caller passes zero, so a zero value stays usable.
const (
	defaultMaxHistory  = 20
	defaultTTL         = 2 * time.Hour
	defaultMaxSessions = 1000
)

// NewMemory returns a memory capped at max messages per session, with the
// default session TTL and session-count bound.
func NewMemory(max int) *Memory {
	return NewMemoryWithLimits(max, defaultTTL, defaultMaxSessions)
}

// NewMemoryWithLimits returns a memory with explicit retention bounds.
func NewMemoryWithLimits(max int, ttl time.Duration, maxSessions int) *Memory {
	if max <= 0 {
		max = defaultMaxHistory
	}
	if ttl <= 0 {
		ttl = defaultTTL
	}
	if maxSessions <= 0 {
		maxSessions = defaultMaxSessions
	}
	return &Memory{
		sessions:    make(map[string]*session),
		max:         max,
		ttl:         ttl,
		maxSessions: maxSessions,
	}
}

// History returns a copy of the session's messages.
//
// The copy matters: routes.py fetches history BEFORE appending the new user
// message, and the caller must not be able to mutate the stored slice.
func (m *Memory) History(sessionID string) []Message {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[sessionID]
	if !ok {
		return []Message{}
	}
	if m.expired(s) {
		delete(m.sessions, sessionID)
		return []Message{}
	}
	m.touch(s)

	out := make([]Message, len(s.messages))
	copy(out, s.messages)
	return out
}

// Append adds a message, trimming from the front to stay within the cap.
func (m *Memory) Append(sessionID, role, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[sessionID]
	if !ok || m.expired(s) {
		s = &session{}
		m.sessions[sessionID] = s
	}
	s.messages = append(s.messages, Message{Role: role, Content: content})
	for len(s.messages) > m.max {
		s.messages = s.messages[1:]
	}
	m.touch(s)
	m.evictLocked()
}

// Clear drops a session. Clearing an unknown session is a no-op, as in Python.
func (m *Memory) Clear(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
}

// Sessions reports the number of resident sessions, for diagnostics and tests.
func (m *Memory) Sessions() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions)
}

func (m *Memory) clock() time.Time {
	if m.now != nil {
		return m.now()
	}
	return time.Now()
}

func (m *Memory) touch(s *session) {
	m.counter++
	s.order = m.counter
	s.lastSeen = m.clock()
}

func (m *Memory) expired(s *session) bool {
	return m.clock().Sub(s.lastSeen) > m.ttl
}

// evictLocked drops idle sessions first, then the least-recently-used ones
// until the map is back within bounds.
func (m *Memory) evictLocked() {
	for id, s := range m.sessions {
		if m.expired(s) {
			delete(m.sessions, id)
		}
	}
	for len(m.sessions) > m.maxSessions {
		var oldestID string
		oldest := ^uint64(0)
		for id, s := range m.sessions {
			if s.order < oldest {
				oldest, oldestID = s.order, id
			}
		}
		if oldestID == "" {
			return
		}
		delete(m.sessions, oldestID)
	}
}

package chatbot

import "sync"

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
type Memory struct {
	mu       sync.Mutex
	sessions map[string][]Message
	max      int
}

// NewMemory returns a memory capped at max messages per session.
func NewMemory(max int) *Memory {
	if max <= 0 {
		max = 20
	}
	return &Memory{sessions: make(map[string][]Message), max: max}
}

// History returns a copy of the session's messages.
//
// The copy matters: routes.py fetches history BEFORE appending the new user
// message, and the caller must not be able to mutate the stored slice.
func (m *Memory) History(sessionID string) []Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	src := m.sessions[sessionID]
	out := make([]Message, len(src))
	copy(out, src)
	return out
}

// Append adds a message, trimming from the front to stay within the cap.
func (m *Memory) Append(sessionID, role, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msgs := append(m.sessions[sessionID], Message{Role: role, Content: content})
	for len(msgs) > m.max {
		msgs = msgs[1:]
	}
	m.sessions[sessionID] = msgs
}

// Clear drops a session. Clearing an unknown session is a no-op, as in Python.
func (m *Memory) Clear(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
}

// Sessions reports the number of live sessions, for diagnostics.
func (m *Memory) Sessions() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions)
}

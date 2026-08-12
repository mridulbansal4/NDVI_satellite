package chatbot

import (
	"fmt"
	"testing"
	"time"
)

func TestMemoryTrimsHistoryFromTheFront(t *testing.T) {
	m := NewMemory(3)
	for i := 0; i < 5; i++ {
		m.Append("s", "user", fmt.Sprintf("m%d", i))
	}
	got := m.History("s")
	if len(got) != 3 {
		t.Fatalf("history length: got %d, want 3", len(got))
	}
	// Front-trimmed, so the OLDEST are gone and order is preserved.
	if got[0].Content != "m2" || got[2].Content != "m4" {
		t.Fatalf("expected m2..m4, got %s..%s", got[0].Content, got[2].Content)
	}
}

func TestMemoryHistoryIsACopy(t *testing.T) {
	m := NewMemory(5)
	m.Append("s", "user", "original")

	got := m.History("s")
	got[0].Content = "mutated"

	if again := m.History("s"); again[0].Content != "original" {
		t.Fatal("History must return a copy; the caller mutated stored state")
	}
}

func TestMemoryEvictsIdleSessionsPastTTL(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	now := base
	m := NewMemoryWithLimits(20, time.Hour, 100)
	m.now = func() time.Time { return now }

	m.Append("stale", "user", "hello")
	if m.Sessions() != 1 {
		t.Fatalf("sessions after first append: got %d, want 1", m.Sessions())
	}

	// Past the TTL, the session is gone and reads start clean.
	now = base.Add(2 * time.Hour)
	if got := m.History("stale"); len(got) != 0 {
		t.Fatalf("an idle session past its TTL should read empty, got %d messages", len(got))
	}
	if m.Sessions() != 0 {
		t.Fatalf("expired session should have been dropped, %d remain", m.Sessions())
	}
}

func TestMemoryActivityKeepsASessionAlive(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	now := base
	m := NewMemoryWithLimits(20, time.Hour, 100)
	m.now = func() time.Time { return now }

	m.Append("live", "user", "one")
	// Read at 45 minutes: still inside the TTL, and the read refreshes it.
	now = base.Add(45 * time.Minute)
	if got := m.History("live"); len(got) != 1 {
		t.Fatalf("session inside its TTL should survive, got %d messages", len(got))
	}
	// 45 minutes after THAT is 90 minutes from creation but only 45 from last use.
	now = base.Add(90 * time.Minute)
	if got := m.History("live"); len(got) != 1 {
		t.Fatal("a session read recently must not expire on wall-clock age alone")
	}
}

func TestMemoryBoundsSessionCountByLRU(t *testing.T) {
	m := NewMemoryWithLimits(20, time.Hour, 3)
	for i := 0; i < 10; i++ {
		m.Append(fmt.Sprintf("s%d", i), "user", "hi")
	}
	if got := m.Sessions(); got > 3 {
		t.Fatalf("session count should be bounded at 3, got %d", got)
	}
	// The most recent survives; the first is long gone.
	if len(m.History("s9")) != 1 {
		t.Error("the most recently used session should have been retained")
	}
	if len(m.History("s0")) != 0 {
		t.Error("the least recently used session should have been evicted")
	}
}

func TestMemoryClearIsANoOpForUnknownSession(t *testing.T) {
	m := NewMemory(5)
	m.Clear("never-existed") // must not panic
	if m.Sessions() != 0 {
		t.Fatal("clearing an unknown session must not create one")
	}
}

func TestNewMemoryRejectsNonPositiveLimits(t *testing.T) {
	m := NewMemoryWithLimits(0, 0, 0)
	for i := 0; i < 50; i++ {
		m.Append("s", "user", "x")
	}
	if got := len(m.History("s")); got != defaultMaxHistory {
		t.Fatalf("zero cap should fall back to the default %d, got %d", defaultMaxHistory, got)
	}
}

package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/SanTiwari07/NDVI_satellite/internal/gee/eeexpr"
)

// Sentinel keys used by the hover-sampling endpoint.
//
// §7.7 / K6: /api/sample samples whatever image the most recent /api/analyze
// left behind — process-global, shared across all users, lost on restart. The
// storage is replaced here by a keyed, TTL'd, bounded cache (objective O4), but
// the SHARED "__last__" semantics are deliberately preserved, because the
// frontend sends no field identifier on the sample call and changing that would
// require touching the frontend (forbidden by §0.2).
const (
	LastKey      = "__last__"
	LastRadarKey = "__last_radar__"
)

// CachedAnalysis is what /api/sample needs to sample a point without re-running
// the pipeline: the indexed image graph and the geometry it was built for.
type CachedAnalysis struct {
	Image    eeexpr.Image
	Geometry eeexpr.Geometry
	StoredAt time.Time
}

type cacheEntry struct {
	value   CachedAnalysis
	expires time.Time
	// order is a monotonically increasing stamp used for LRU eviction. A
	// counter rather than a timestamp so that two writes in the same
	// millisecond still order deterministically.
	order uint64
}

// ExprCache stores serialised EE expression graphs (not results), keyed by a
// hash of the inputs that produced them, with a TTL and a bounded size.
type ExprCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
	maxSize int
	counter uint64
}

// NewExprCache returns a cache with the configured TTL and bound.
func NewExprCache(ttl time.Duration, maxEntries int) *ExprCache {
	if maxEntries <= 0 {
		maxEntries = 256
	}
	return &ExprCache{
		entries: make(map[string]*cacheEntry, maxEntries),
		ttl:     ttl,
		maxSize: maxEntries,
	}
}

// CacheKey derives a stable key from the inputs that produced an analysis:
// sha256 of canonical-JSON(geometry) + "|" + date + "|" + pipeline tag.
func CacheKey(geometry json.RawMessage, date, tag string) string {
	h := sha256.New()
	h.Write(canonicalJSON(geometry))
	h.Write([]byte("|" + date + "|" + tag))
	return hex.EncodeToString(h.Sum(nil))
}

// canonicalJSON re-marshals so that key order and whitespace do not change the
// hash. A body that will not parse is hashed as-is rather than dropped.
func canonicalJSON(raw json.RawMessage) []byte {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw
	}
	out, err := json.Marshal(v)
	if err != nil {
		return raw
	}
	return out
}

// Put stores a value under every supplied key. Callers store under both the
// derived key and the appropriate sentinel.
func (c *ExprCache) Put(value CachedAnalysis, keys ...string) {
	if len(keys) == 0 {
		return
	}
	now := time.Now()
	value.StoredAt = now

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, k := range keys {
		c.counter++
		c.entries[k] = &cacheEntry{
			value:   value,
			expires: now.Add(c.ttl),
			order:   c.counter,
		}
	}
	c.evictLocked(now)
}

// Get returns the value if present and unexpired.
func (c *ExprCache) Get(key string) (CachedAnalysis, bool) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return CachedAnalysis{}, false
	}
	if time.Now().After(e.expires) {
		c.mu.Lock()
		// Re-check under the write lock: another goroutine may have refreshed it.
		if cur, still := c.entries[key]; still && time.Now().After(cur.expires) {
			delete(c.entries, key)
		}
		c.mu.Unlock()
		return CachedAnalysis{}, false
	}
	return e.value, true
}

// Len reports the number of live entries, for tests and diagnostics.
func (c *ExprCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// evictLocked drops expired entries first, then the least-recently-written
// ones until the cache is back within bounds. The sentinel keys are evicted
// like any other entry: /api/sample answering 404 after a long idle period is
// correct, and matches a restarted Python process.
func (c *ExprCache) evictLocked(now time.Time) {
	for k, e := range c.entries {
		if now.After(e.expires) {
			delete(c.entries, k)
		}
	}
	for len(c.entries) > c.maxSize {
		var oldestKey string
		oldest := ^uint64(0)
		for k, e := range c.entries {
			if e.order < oldest {
				oldest, oldestKey = e.order, k
			}
		}
		if oldestKey == "" {
			return
		}
		delete(c.entries, oldestKey)
	}
}

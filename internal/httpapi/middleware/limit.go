package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// BodyLimit caps how many bytes a request body may carry.
//
// ReadTimeout bounds how LONG a client may spend sending, not how much it may
// send, and the analysis and chatbot routes take no credentials — so without a
// byte cap a handful of concurrent large POSTs is a cheap out-of-memory kill.
// The handlers read the body with io.ReadAll; MaxBytesReader turns an
// over-limit body into a read error there, which they already treat as an
// absent body.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes > 0 && c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}

// InFlight bounds how many requests may occupy an expensive handler at once.
//
// One /api/analyze is a dozen-plus Earth Engine round trips — a scene count, an
// auto-coarsening grid search, a computeFeatures over up to 2000 cells, three
// statistics calls and seven map creations. Unbounded concurrency there
// exhausts the project's Earth Engine quota, which takes the platform down for
// every user, long before it exhausts this process.
//
// Over-limit requests WAIT rather than fail: a cold analysis legitimately takes
// half a minute and the frontend proxy allows 300s, so queueing is kinder than
// a 503 the UI would surface as an outage. A cancelled client releases its slot.
func InFlight(limit int) gin.HandlerFunc {
	if limit <= 0 {
		return func(c *gin.Context) { c.Next() }
	}
	sem := make(chan struct{}, limit)
	return func(c *gin.Context) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			c.Next()
		case <-c.Request.Context().Done():
			c.AbortWithStatus(http.StatusRequestTimeout)
		}
	}
}

// RateLimiter is a per-client-IP token bucket.
//
// Every unauthenticated route in this service spends real money on each call —
// Earth Engine quota, Gemini tokens, an SMS message, or 32 MB of scrypt — so
// the cap exists to stop automated abuse, not to shape normal traffic. The
// defaults are deliberately generous: the contract suite replays 74 cases back
// to back from one address and must not trip them.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket

	// refill tokens per second, derived from the per-minute setting.
	refill float64
	burst  float64
	// now is injectable so tests need not sleep.
	now func() time.Time

	lastSweep time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter allows perMin sustained requests per client with a burst
// allowance on top. A non-positive perMin disables limiting entirely.
func NewRateLimiter(perMin, burst int) *RateLimiter {
	if burst <= 0 {
		burst = perMin
	}
	return &RateLimiter{
		buckets: make(map[string]*bucket),
		refill:  float64(perMin) / 60.0,
		burst:   float64(burst),
	}
}

func (r *RateLimiter) clock() time.Time {
	if r.now != nil {
		return r.now()
	}
	return time.Now()
}

// Allow reports whether a request from key may proceed, consuming a token.
func (r *RateLimiter) Allow(key string) bool {
	if r.refill <= 0 {
		return true
	}
	now := r.clock()

	r.mu.Lock()
	defer r.mu.Unlock()

	b, ok := r.buckets[key]
	if !ok {
		b = &bucket{tokens: r.burst, last: now}
		r.buckets[key] = b
	}
	// Refill for elapsed time, capped at the burst size.
	if elapsed := now.Sub(b.last).Seconds(); elapsed > 0 {
		b.tokens += elapsed * r.refill
		if b.tokens > r.burst {
			b.tokens = r.burst
		}
		b.last = now
	}
	r.sweepLocked(now)

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// sweepLocked drops buckets that have been full and idle long enough that
// forgetting them changes nothing, so the map cannot grow without bound under
// a rotating source address.
func (r *RateLimiter) sweepLocked(now time.Time) {
	const sweepEvery = 5 * time.Minute
	if now.Sub(r.lastSweep) < sweepEvery {
		return
	}
	r.lastSweep = now
	for k, b := range r.buckets {
		if now.Sub(b.last) > sweepEvery && b.tokens >= r.burst {
			delete(r.buckets, k)
		}
	}
}

// Middleware rejects over-limit callers with 429 and a Retry-After hint.
//
// The envelope is {"error": …}, matching every other ad-hoc route error in this
// service rather than inventing a fourth shape.
func (r *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !r.Allow(clientKey(c)) {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				gin.H{"error": "Too many requests. Please slow down and try again."})
			return
		}
		c.Next()
	}
}

// clientKey identifies the caller for rate-limiting purposes.
//
// gin's ClientIP already honours the trusted-proxy configuration, so a
// spoofed X-Forwarded-For does not get a fresh bucket unless the deployment
// has explicitly trusted that proxy. The port is stripped so a client opening
// many connections shares one bucket.
func clientKey(c *gin.Context) string {
	ip := c.ClientIP()
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}
	return ip
}

package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiterAllowsBurstThenBlocks(t *testing.T) {
	r := NewRateLimiter(60, 10)
	base := time.Unix(1_700_000_000, 0)
	r.now = func() time.Time { return base }

	for i := 0; i < 10; i++ {
		if !r.Allow("1.2.3.4") {
			t.Fatalf("request %d should be inside the burst allowance", i+1)
		}
	}
	if r.Allow("1.2.3.4") {
		t.Fatal("the 11th request should exceed a burst of 10")
	}
	// A different client has its own bucket.
	if !r.Allow("5.6.7.8") {
		t.Fatal("a second client must not be affected by the first client's bucket")
	}
}

func TestRateLimiterRefillsOverTime(t *testing.T) {
	r := NewRateLimiter(60, 2) // 1 token/second
	base := time.Unix(1_700_000_000, 0)
	now := base
	r.now = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		if !r.Allow("a") {
			t.Fatalf("burst of 2 should permit request %d", i+1)
		}
	}
	if r.Allow("a") {
		t.Fatal("third immediate request should be blocked")
	}

	now = base.Add(1500 * time.Millisecond) // one whole token back
	if !r.Allow("a") {
		t.Fatal("a token should have refilled after 1.5s at 1/s")
	}
}

func TestRateLimiterDisabledWhenRateNonPositive(t *testing.T) {
	r := NewRateLimiter(0, 0)
	for i := 0; i < 100; i++ {
		if !r.Allow("a") {
			t.Fatal("a non-positive rate must disable limiting entirely")
		}
	}
}

func TestRateLimiterMiddlewareReturns429WithErrorEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewRateLimiter(60, 1)
	base := time.Unix(1_700_000_000, 0)
	r.now = func() time.Time { return base }

	eng := gin.New()
	eng.Use(r.Middleware())
	eng.GET("/x", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	first := httptest.NewRecorder()
	eng.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/x", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first request: got %d, want 200", first.Code)
	}

	second := httptest.NewRecorder()
	eng.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/x", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: got %d, want 429", second.Code)
	}
	// The envelope must be the existing {"error": …} shape, not a fourth one.
	if body := second.Body.String(); !strings.Contains(body, `"error"`) {
		t.Fatalf("429 body should use the {\"error\": …} envelope, got %s", body)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Error("429 should carry a Retry-After hint")
	}
}

func TestRateLimiterIsConcurrencySafe(t *testing.T) {
	r := NewRateLimiter(60_000, 500)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				r.Allow("shared")
			}
		}()
	}
	wg.Wait() // -race is what actually asserts here
}

// The handlers read with io.ReadAll, so that is what the cap has to stop —
// a single short Read succeeds by design and only the read PAST the limit fails.
func TestBodyLimitStopsIoReadAllAtTheCap(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var gotBytes int
	var gotErr error

	eng := gin.New()
	eng.Use(BodyLimit(16))
	eng.POST("/x", func(c *gin.Context) {
		raw, err := io.ReadAll(c.Request.Body)
		gotBytes, gotErr = len(raw), err
		c.JSON(http.StatusOK, gin.H{"read": len(raw)})
	})

	big := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(strings.Repeat("x", 4096)))
	eng.ServeHTTP(httptest.NewRecorder(), big)

	if gotErr == nil {
		t.Fatal("io.ReadAll should fail once the body exceeds the cap")
	}
	if gotBytes > 16 {
		t.Fatalf("read %d bytes past a 16-byte cap", gotBytes)
	}
}

func TestBodyLimitPassesSmallBodyThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var got string
	eng := gin.New()
	eng.Use(BodyLimit(1 << 20))
	eng.POST("/x", func(c *gin.Context) {
		raw, err := io.ReadAll(c.Request.Body)
		if err != nil {
			t.Errorf("a small body must read cleanly: %v", err)
		}
		got = string(raw)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	rec := httptest.NewRecorder()
	eng.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"a":1}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("a small body must pass through untouched, got %d", rec.Code)
	}
	if got != `{"a":1}` {
		t.Fatalf("body was altered: got %q", got)
	}
}

func TestBodyLimitZeroIsPassthrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	eng := gin.New()
	eng.Use(BodyLimit(0))
	eng.POST("/x", func(c *gin.Context) {
		raw, err := io.ReadAll(c.Request.Body)
		if err != nil || len(raw) != 4096 {
			t.Errorf("a non-positive cap must not gate the body: n=%d err=%v", len(raw), err)
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	eng.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(strings.Repeat("x", 4096))))
}

func TestInFlightBoundsConcurrency(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var mu sync.Mutex
	current, peak := 0, 0

	eng := gin.New()
	eng.Use(InFlight(2))
	eng.GET("/x", func(c *gin.Context) {
		mu.Lock()
		current++
		if current > peak {
			peak = current
		}
		mu.Unlock()

		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		current--
		mu.Unlock()
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			eng.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
			if rec.Code != http.StatusOK {
				t.Errorf("queued request should still succeed, got %d", rec.Code)
			}
		}()
	}
	wg.Wait()

	if peak > 2 {
		t.Fatalf("InFlight(2) allowed %d concurrent handlers", peak)
	}
}

func TestInFlightZeroIsPassthrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	eng := gin.New()
	eng.Use(InFlight(0))
	eng.GET("/x", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	rec := httptest.NewRecorder()
	eng.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("a non-positive limit must not gate anything, got %d", rec.Code)
	}
}

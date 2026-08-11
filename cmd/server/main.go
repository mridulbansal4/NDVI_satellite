// Command server is the Go replacement for backend/app.py.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §3.1, §5.5, §13.3, §13.10.
//
// Wiring only: config → deps → router → ListenAndServe.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/SanTiwari07/NDVI_satellite/internal/config"
	"github.com/SanTiwari07/NDVI_satellite/internal/httpapi"
	"github.com/SanTiwari07/NDVI_satellite/internal/logging"
)

func main() {
	var (
		healthcheck = flag.Bool("healthcheck", false,
			"probe /health on the configured port and exit (container HEALTHCHECK)")
		port = flag.Int("port", 0, "override FLASK_PORT")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	if *port != 0 {
		cfg.Port = *port
	}

	if *healthcheck {
		os.Exit(probeHealth(cfg.Port))
	}

	log := logging.New(cfg.LogLevel, cfg.LogFile)
	initLog := logging.Named(log, "Init")

	deps := httpapi.Deps{
		Cfg:           cfg,
		Log:           log,
		GEEReady:      &atomic.Bool{},
		FirebaseReady: &atomic.Bool{},
	}

	// Startup probes run ONCE in a goroutine, not per-request. The Python code
	// does this in @app.before_request, which is a workaround for Flask's
	// lifecycle and is deliberately not reproduced (§5.5). Neither probe may
	// prevent startup: /health, /auth/* and /dashboard must work without GEE.
	go func() {
		if cfg.GEEProjectID == "" {
			initLog.Error("GEE_PROJECT_ID is not set. " +
				"Create a .env file with:\n    GEE_PROJECT_ID=your-cloud-project-id\n" +
				"Find your project ID at https://console.cloud.google.com")
			return
		}
		// Phase 2 replaces this with a real value:compute connectivity probe.
		initLog.Warn("GEE client not wired yet (Phase 2) — analysis routes will answer 503")
	}()

	go func() {
		// Phase 5 wires Firebase Admin. firebase_ready=false is the normal dev
		// state and must not break anything (§13.7).
		initLog.Info("Firebase not wired yet (Phase 5) — firebase_ready=false")
	}()

	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.Port),
		Handler: httpapi.New(deps),
		// A cold /api/analyze on a large polygon can exceed 60s: grid
		// auto-coarsening round-trips, seven maps calls, and a large
		// computeFeatures. Vite proxies /api with a 300s timeout, so the write
		// timeout must not be shorter (§3.1, §13.3).
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      300 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		initLog.Info("Starting Satellite Agronomy Intelligence Platform Backend — port " +
			strconv.Itoa(cfg.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			initLog.Error("listen failed: " + err.Error())
			os.Exit(1)
		}
	}()

	// Graceful shutdown with a 30s drain so in-flight analyses finish (§13.10).
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	initLog.Info("shutdown signal received, draining for up to 30s")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		initLog.Error("graceful shutdown failed: " + err.Error())
	}
	initLog.Info("stopped")
}

func probeHealth(port int) int {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/health")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

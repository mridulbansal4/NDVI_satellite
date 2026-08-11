// Package logging reproduces the Python backend's log format so existing
// log-grep habits keep working.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §9.4.
//
// Python:  LOG_FORMAT = "%(asctime)s [%(levelname)s] %(name)s — %(message)s"
//          LOG_DATE   = "%Y-%m-%d %H:%M:%S"
// Output:  2026-08-11 22:48:03 [INFO] app — Analysis complete
//
// Note the separator is an em dash (U+2014), not a hyphen.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

const timeLayout = "2006-01-02 15:04:05"

// handler writes records in the Python logging format. The bracketed subsystem
// prefixes the Python code uses ([DB], [Auth], [Firestore], [VI Engine], …)
// are part of the message text and are preserved verbatim at the call sites.
type handler struct {
	mu    *sync.Mutex
	w     io.Writer
	level slog.Level
	name  string
	attrs []slog.Attr
}

// New returns a logger writing the Python format to both stdout and logFile.
// A logFile that cannot be opened is not fatal — stdout alone is used.
func New(level string, logFile string) *slog.Logger {
	var lv slog.Level
	if err := lv.UnmarshalText([]byte(strings.ToUpper(level))); err != nil {
		lv = slog.LevelInfo
	}

	var w io.Writer = os.Stdout
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			w = io.MultiWriter(os.Stdout, f)
		} else {
			fmt.Fprintf(os.Stderr, "logging: cannot open %s: %v (stdout only)\n", logFile, err)
		}
	}

	return slog.New(&handler{mu: &sync.Mutex{}, w: w, level: lv, name: "app"})
}

// Named returns a logger tagged with a subsystem name, the analogue of
// Python's logging.getLogger(__name__).
func Named(l *slog.Logger, name string) *slog.Logger {
	if h, ok := l.Handler().(*handler); ok {
		clone := *h
		clone.name = name
		return slog.New(&clone)
	}
	return l
}

func (h *handler) Enabled(_ context.Context, l slog.Level) bool { return l >= h.level }

func (h *handler) Handle(_ context.Context, r slog.Record) error {
	var sb strings.Builder
	sb.WriteString(r.Time.Format(timeLayout))
	sb.WriteString(" [")
	sb.WriteString(levelName(r.Level))
	sb.WriteString("] ")
	sb.WriteString(h.name)
	sb.WriteString(" — ") // U+2014, matching LOG_FORMAT
	sb.WriteString(r.Message)

	for _, a := range h.attrs {
		writeAttr(&sb, a)
	}
	r.Attrs(func(a slog.Attr) bool {
		writeAttr(&sb, a)
		return true
	})
	sb.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, sb.String())
	return err
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &clone
}

func (h *handler) WithGroup(name string) slog.Handler {
	clone := *h
	if name != "" {
		clone.name = h.name + "." + name
	}
	return &clone
}

func writeAttr(sb *strings.Builder, a slog.Attr) {
	if a.Equal(slog.Attr{}) {
		return
	}
	fmt.Fprintf(sb, " %s=%v", a.Key, a.Value.Any())
}

// levelName maps slog levels onto Python's logging level names. Python has no
// slog.LevelDebug-4 style offsets, so anything below INFO reads as DEBUG.
func levelName(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "ERROR"
	case l >= slog.LevelWarn:
		return "WARNING" // Python prints WARNING, not WARN
	case l >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}

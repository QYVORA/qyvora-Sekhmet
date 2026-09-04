// Package logger provides structured, leveled logging for sekhmet. Normal
// terminal UX is separate from debug logging and the machine-readable event
// stream: debug logs are never printed into the fuzzing HUD.
package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

type Level int

const (
	LevelSilent Level = iota
	LevelError
	LevelWarn
	LevelInfo
	LevelDebug
)

var levelNames = map[Level]string{
	LevelError: "ERROR",
	LevelWarn:  "WARN",
	LevelInfo:  "INFO",
	LevelDebug: "DEBUG",
}

// ParseLevel resolves a configured log level name.
func ParseLevel(s string) Level {
	switch strings.ToLower(s) {
	case "silent":
		return LevelSilent
	case "error":
		return LevelError
	case "warn":
		return LevelWarn
	case "debug":
		return LevelDebug
	default:
		return LevelInfo
	}
}

// Logger is a leveled, concurrency-safe logger.
type Logger struct {
	mu      sync.Mutex
	w       io.Writer
	level   Level
	verbose bool
	quiet   bool
}

// New returns a logger writing to stderr at the info level.
func New() *Logger {
	return &Logger{w: os.Stderr, level: LevelInfo}
}

// SetWriter sets the output writer.
func (l *Logger) SetWriter(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.w = w
}

// SetLevel sets the active threshold level.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetVerbose toggles verbose output.
func (l *Logger) SetVerbose(v bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.verbose = v
}

// SetQuiet toggles quiet output.
func (l *Logger) SetQuiet(q bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.quiet = q
}

func (l *Logger) Errorf(format string, args ...any) { l.log(LevelError, format, args...) }
func (l *Logger) Warnf(format string, args ...any)  { l.log(LevelWarn, format, args...) }
func (l *Logger) Infof(format string, args ...any)  { l.log(LevelInfo, format, args...) }
func (l *Logger) Debugf(format string, args ...any) { l.log(LevelDebug, format, args...) }

func (l *Logger) log(level Level, format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if level > l.level {
		return
	}
	if level < LevelWarn && l.quiet && !l.verbose {
		return
	}
	name, ok := levelNames[level]
	if !ok {
		name = "INFO"
	}
	fmt.Fprintf(l.w, "[sekhmet][%s] %s\n", name, fmt.Sprintf(format, args...))
}

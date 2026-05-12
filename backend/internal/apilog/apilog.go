package apilog

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// Entry is one line retained for the in-memory debug console.
type Entry struct {
	Time  string `json:"time"`
	Level string `json:"level"`
	Msg   string `json:"msg"`
}

const defaultCap = 250

var ring = &buffer{cap: defaultCap}

type buffer struct {
	mu  sync.Mutex
	cap int
	buf []Entry
}

func (b *buffer) add(level, msg string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	e := Entry{
		Time:  time.Now().UTC().Format(time.RFC3339),
		Level: level,
		Msg:   msg,
	}
	if len(b.buf) >= b.cap {
		b.buf = b.buf[1:]
	}
	b.buf = append(b.buf, e)
}

func (b *buffer) snapshot(limit int) []Entry {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(b.buf)
	if limit > 0 && limit < n {
		n = limit
	}
	out := make([]Entry, n)
	copy(out, b.buf[len(b.buf)-n:])
	return out
}

func (b *buffer) clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = b.buf[:0]
}

// Info logs to stderr (standard logger) and the ring buffer.
func Info(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("[api] %s", msg)
	ring.add("info", msg)
}

// Warn logs a warning line.
func Warn(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("[api] WARN %s", msg)
	ring.add("warn", msg)
}

// Error logs an error line.
func Error(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("[api] ERROR %s", msg)
	ring.add("error", msg)
}

// Snapshot returns up to limit newest entries (oldest first). limit <= 0 means all retained.
func Snapshot(limit int) []Entry {
	return ring.snapshot(limit)
}

// Clear removes buffered entries.
func Clear() {
	ring.clear()
}

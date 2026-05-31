// Package logger provides a file-based logger for debugging.
// Writes to ~/.config/prwatch/prwatch.log by default.
package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

// L is the package-level logger. nil until Init is called (safe to check).
var L *log.Logger

// Init opens (or creates) the log file and configures L.
// Returns a closer the caller must defer.
func Init(path string) (io.Closer, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, err
	}
	L = log.New(f, "", log.LstdFlags|log.Lmsgprefix)
	L.Printf("[prwatch] ---- session start ----\n")
	return f, nil
}

// DefaultPath returns the default log file path.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "prwatch", "prwatch.log")
}

// Logf writes a formatted line if L is initialised.
func Logf(format string, args ...any) {
	if L != nil {
		L.Printf(format, args...)
	}
}

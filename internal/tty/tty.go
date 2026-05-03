// Package tty wraps terminal-detection and width queries so the rest of the
// codebase does not depend on golang.org/x/term directly.
package tty

import (
	"io"
	"os"

	"golang.org/x/term"
)

// IsTerminal reports whether w writes to a terminal. Returns false for any
// writer that is not an *os.File or whose underlying file descriptor is not
// a terminal (pipes, redirects, in-memory buffers, etc.).
func IsTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// Width returns the column width of the terminal w writes to. ok is false
// when w is not an *os.File or its file descriptor is not a terminal, in
// which case callers should fall back to a sensible default.
func Width(w io.Writer) (cols int, ok bool) {
	f, isFile := w.(*os.File)
	if !isFile {
		return 0, false
	}
	cols, _, err := term.GetSize(int(f.Fd()))
	if err != nil {
		return 0, false
	}
	return cols, true
}

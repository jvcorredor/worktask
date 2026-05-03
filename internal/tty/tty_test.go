package tty

import (
	"bytes"
	"os"
	"testing"
)

func TestIsTerminal_pipeIsNotTerminal(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	if IsTerminal(w) {
		t.Fatalf("IsTerminal returned true for an os.Pipe writer; want false")
	}
}

func TestIsTerminal_nonFileWriterIsNotTerminal(t *testing.T) {
	if IsTerminal(&bytes.Buffer{}) {
		t.Fatalf("IsTerminal returned true for a *bytes.Buffer; want false")
	}
}

func TestWidth_pipeReturnsNotOK(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	cols, ok := Width(w)
	if ok {
		t.Fatalf("Width returned ok=true for an os.Pipe writer; cols=%d", cols)
	}
}

func TestWidth_nonFileWriterReturnsNotOK(t *testing.T) {
	if cols, ok := Width(&bytes.Buffer{}); ok {
		t.Fatalf("Width returned ok=true for a *bytes.Buffer; cols=%d", cols)
	}
}

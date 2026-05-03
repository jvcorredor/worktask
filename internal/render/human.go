package render

import (
	"fmt"
	"io"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/jvcorredor/worktask/internal/task"
	"github.com/jvcorredor/worktask/internal/tty"
)

const (
	humanShowMaxWrap     = 120
	humanShowFallbackCol = 80
)

// HumanShow writes a single task in human-format. When styled is false the
// raw markdown bytes (frontmatter included) are written verbatim, preserving
// the pipe-mode contract callers rely on for editing and round-tripping.
// When styled is true the body is rendered through glamour with a one-line
// metadata strip above it.
func HumanShow(w io.Writer, t task.Task, raw []byte, styled bool) error {
	if !styled {
		_, err := w.Write(raw)
		return err
	}
	return humanShowStyled(w, t)
}

func humanShowStyled(w io.Writer, t task.Task) error {
	if _, err := fmt.Fprintln(w, metadataStrip(t)); err != nil {
		return err
	}
	wrap := wrapWidth(w)
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(wrap),
	)
	if err != nil {
		return fmt.Errorf("render: glamour: %w", err)
	}
	out, err := r.Render(t.Body)
	if err != nil {
		return fmt.Errorf("render: glamour render: %w", err)
	}
	_, err = io.WriteString(w, out)
	return err
}

func metadataStrip(t task.Task) string {
	status := "open"
	if !t.Completed.IsZero() {
		status = "closed"
	}
	line := fmt.Sprintf("%s · %s · %s", t.ID, t.Created.Format("2006-01-02"), status)
	return lipgloss.NewStyle().Faint(true).Render(line)
}

func wrapWidth(w io.Writer) int {
	cols, ok := tty.Width(w)
	if !ok {
		return humanShowFallbackCol
	}
	if cols > humanShowMaxWrap {
		return humanShowMaxWrap
	}
	return cols
}

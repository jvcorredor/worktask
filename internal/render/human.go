package render

import (
	"fmt"
	"io"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/jvcorredor/worktask/internal/task"
	"github.com/jvcorredor/worktask/internal/tty"
)

const (
	humanShowMaxWrap     = 120
	humanShowFallbackCol = 80

	humanListIDWidth        = 8
	humanListDateWidth      = 10
	humanListSeparatorWidth = 2
	humanListDescFallback   = 80
	humanListMinDescBudget  = 1
)

// humanListAccent is the adaptive accent colour used for IDs in the
// styled list and candidate renderers. Soft indigo on dark, deeper
// indigo on light. lipgloss.AdaptiveColor picks at runtime based on
// terminal background.
var humanListAccent = lipgloss.AdaptiveColor{Light: "#5A4FCF", Dark: "#A29BFE"}

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

// humanListStyled renders tasks as a borderless lipgloss table with a
// header row. IDs use the accent colour; dates are faint date-only;
// descriptions are hard-truncated with `…` to fit the terminal width
// budget. NO_COLOR is honoured natively by lipgloss.
func humanListStyled(w io.Writer, tasks []task.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	descBudget := humanListDescBudget(w)

	idStyle := lipgloss.NewStyle().Foreground(humanListAccent)
	faintStyle := lipgloss.NewStyle().Faint(true)
	headerStyle := lipgloss.NewStyle().Bold(true).Faint(true)

	t := table.New().
		Border(lipgloss.HiddenBorder()).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		BorderRow(false).
		BorderHeader(false).
		Wrap(false).
		Headers("ID", "CREATED", "DESCRIPTION").
		StyleFunc(func(row, col int) lipgloss.Style {
			padLeft := 0
			padRight := 2
			if col == 2 {
				padRight = 0
			}
			base := lipgloss.NewStyle().Padding(0, padRight, 0, padLeft)
			switch {
			case row == table.HeaderRow:
				return base.Inherit(headerStyle)
			case col == 0:
				return base.Inherit(idStyle)
			case col == 1:
				return base.Inherit(faintStyle)
			default:
				return base
			}
		})

	for _, tk := range tasks {
		t.Row(tk.ID, tk.Created.Format("2006-01-02"), truncateRunes(description(tk), descBudget))
	}

	_, err := fmt.Fprintln(w, t.Render())
	return err
}

// humanListDescBudget returns the column width available to the
// description after subtracting the fixed-width ID and date columns
// plus their separators. Falls back to 80 when terminal width
// detection fails.
func humanListDescBudget(w io.Writer) int {
	cols, ok := tty.Width(w)
	if !ok {
		return humanListDescFallback
	}
	overhead := humanListIDWidth + humanListSeparatorWidth + humanListDateWidth + humanListSeparatorWidth
	budget := cols - overhead
	if budget < humanListMinDescBudget {
		return humanListMinDescBudget
	}
	return budget
}

// truncateRunes returns s shortened to width display runes, replacing
// the trailing rune with `…` when truncation occurs. Returns "" when
// width <= 0 and "…" when width == 1.
func truncateRunes(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	return string(runes[:width-1]) + "…"
}

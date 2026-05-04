package render

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/jvcorredor/bytheway/internal/store"
)

// TestZGenerateGoldenFiles is a one-shot helper enabled with BTW_GOLDEN=1
// to (re)generate golden testdata files for new render shapes.
func TestZGenerateGoldenFiles(t *testing.T) {
	if os.Getenv("BTW_GOLDEN") != "1" {
		t.Skip("set BTW_GOLDEN=1 to regenerate golden files")
	}
	emit := func(name string, fn func() ([]byte, error)) {
		t.Helper()
		b, err := fn()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if err := os.WriteFile("testdata/"+name, b, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	emit("research_event_task_started.json", func() ([]byte, error) {
		return JSONResearchEvent(ResearchEvent{
			Type:        "task_started",
			ID:          "abcdef12",
			Description: "buy milk",
			Started:     time.Date(2026, 5, 1, 14, 0, 0, 0, time.UTC),
		})
	})
	emit("research_event_task_done.json", func() ([]byte, error) {
		return JSONResearchEvent(ResearchEvent{
			Type:    "task_done",
			ID:      "abcdef12",
			Status:  "findings",
			Summary: "deploy lock",
		})
	})
	emit("research_event_task_failed.json", func() ([]byte, error) {
		return JSONResearchEvent(ResearchEvent{
			Type:  "task_failed",
			ID:    "abcdef12",
			Error: "context deadline exceeded",
		})
	})
	emit("tag_ls.json", func() ([]byte, error) {
		return JSONTagList([]store.TagCount{
			{Name: "bug", Count: 3},
			{Name: "infra", Count: 5},
			{Name: "urgent", Count: 2},
		})
	})
	emit("tag_ls_human_unstyled.txt", func() ([]byte, error) {
		var buf bytes.Buffer
		err := HumanTagList(&buf, []store.TagCount{
			{Name: "bug", Count: 3},
			{Name: "infra", Count: 5},
			{Name: "urgent", Count: 2},
		}, false)
		return buf.Bytes(), err
	})

	emit("research_summary.json", func() ([]byte, error) {
		return JSONResearchSummary(ResearchSummary{
			BatchID:  "2026-05-01T14:00:00Z",
			Started:  time.Date(2026, 5, 1, 14, 0, 0, 0, time.UTC),
			Finished: time.Date(2026, 5, 1, 14, 8, 30, 0, time.UTC),
			Results: []ResearchSummaryResult{
				{ID: "abcdef12", Description: "buy milk", Status: "findings", Summary: "deploy lock", LogPath: "research-logs/abcdef12_2026-05-01T14-00-00.jsonl", Tokens: 1234},
				{ID: "fedcba98", Description: "investigate dashboard", Status: "clarify", Summary: "what is X", LogPath: "research-logs/fedcba98_2026-05-01T14-01-00.jsonl", Tokens: 500},
				{ID: "11111111", Description: "datadog hang", Status: "failed", Summary: "claude exited 1", LogPath: "research-logs/11111111_2026-05-01T14-02-00.jsonl", Error: "exit status 1"},
			},
			Totals: ResearchSummaryTotals{Findings: 1, Clarify: 1, Failed: 1, Skipped: 4},
		})
	})
}

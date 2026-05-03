package render

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jvcorredor/worktask/internal/store"
	"github.com/jvcorredor/worktask/internal/task"
)

func TestJSONList_matchesSnapshot(t *testing.T) {
	tasks := []task.Task{
		{
			ID:      "11111111",
			Created: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC),
			Body:    "first task\nbody continues here\n",
		},
		{
			ID:      "22222222",
			Created: time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC),
			Body:    "second task\n",
		},
	}

	got, err := JSONList(tasks)
	if err != nil {
		t.Fatalf("JSONList: %v", err)
	}

	want, err := os.ReadFile("testdata/list.json")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}

	if string(got) != string(want) {
		t.Errorf("JSON output does not match snapshot.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestJSONErrorAmbiguous_matchesSnapshot(t *testing.T) {
	candidates := []store.Candidate{
		{ID: "11111111", Description: "buy milk"},
		{ID: "22222222", Description: "buy milkshake"},
	}
	got, err := JSONErrorAmbiguous("milk", candidates)
	if err != nil {
		t.Fatalf("JSONErrorAmbiguous: %v", err)
	}
	want, err := os.ReadFile("testdata/error_ambiguous.json")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("JSON output does not match snapshot.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestJSONErrorNoMatch_matchesSnapshot(t *testing.T) {
	openTasks := []task.Task{
		{
			ID:      "11111111",
			Created: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC),
			Body:    "first task\n",
		},
		{
			ID:      "22222222",
			Created: time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC),
			Body:    "second task\n",
		},
	}
	got, err := JSONErrorNoMatch("zzz", openTasks)
	if err != nil {
		t.Fatalf("JSONErrorNoMatch: %v", err)
	}
	want, err := os.ReadFile("testdata/error_no_match.json")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("JSON output does not match snapshot.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestJSONList_mixedOpenAndClosed(t *testing.T) {
	tasks := []task.Task{
		{
			ID:      "11111111",
			Created: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC),
			Body:    "open task\n",
		},
		{
			ID:        "22222222",
			Created:   time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC),
			Completed: time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC),
			Body:      "shipped feature\n",
		},
	}

	got, err := JSONList(tasks)
	if err != nil {
		t.Fatalf("JSONList: %v", err)
	}

	want, err := os.ReadFile("testdata/list_mixed.json")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}

	if string(got) != string(want) {
		t.Errorf("JSON output does not match snapshot.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestJSONResearchRun_findingsMatchesSnapshot(t *testing.T) {
	got, err := JSONResearchRun(ResearchRun{
		ID:          "abcdef12",
		Description: "buy milk",
		Status:      "findings",
		Summary:     "deploy lock",
		LogPath:     "research-logs/abcdef12_2026-05-01T14-00-00.jsonl",
	})
	if err != nil {
		t.Fatalf("JSONResearchRun: %v", err)
	}
	want, err := os.ReadFile("testdata/research_run_findings.json")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("JSON output does not match snapshot.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestJSONResearchRun_failedIncludesError(t *testing.T) {
	got, err := JSONResearchRun(ResearchRun{
		ID:          "abcdef12",
		Description: "buy milk",
		Status:      "failed",
		Summary:     "claude exited 1",
		LogPath:     "research-logs/abcdef12_2026-05-01T14-00-00.jsonl",
		Error:       "exec: claude: signal: killed",
	})
	if err != nil {
		t.Fatalf("JSONResearchRun: %v", err)
	}
	want, err := os.ReadFile("testdata/research_run_failed.json")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("JSON output does not match snapshot.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestJSONResearchEvent_taskStartedMatchesSnapshot(t *testing.T) {
	got, err := JSONResearchEvent(ResearchEvent{
		Type:        "task_started",
		ID:          "abcdef12",
		Description: "buy milk",
		Started:     time.Date(2026, 5, 1, 14, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("JSONResearchEvent: %v", err)
	}
	want, err := os.ReadFile("testdata/research_event_task_started.json")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("JSON output does not match snapshot.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestJSONResearchEvent_taskDoneMatchesSnapshot(t *testing.T) {
	got, err := JSONResearchEvent(ResearchEvent{
		Type:    "task_done",
		ID:      "abcdef12",
		Status:  "findings",
		Summary: "deploy lock",
	})
	if err != nil {
		t.Fatalf("JSONResearchEvent: %v", err)
	}
	want, err := os.ReadFile("testdata/research_event_task_done.json")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("JSON output does not match snapshot.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestJSONResearchEvent_taskFailedMatchesSnapshot(t *testing.T) {
	got, err := JSONResearchEvent(ResearchEvent{
		Type:  "task_failed",
		ID:    "abcdef12",
		Error: "context deadline exceeded",
	})
	if err != nil {
		t.Fatalf("JSONResearchEvent: %v", err)
	}
	want, err := os.ReadFile("testdata/research_event_task_failed.json")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("JSON output does not match snapshot.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestJSONResearchSummary_matchesSnapshot(t *testing.T) {
	got, err := JSONResearchSummary(ResearchSummary{
		BatchID:  "2026-05-01T14:00:00Z",
		Started:  time.Date(2026, 5, 1, 14, 0, 0, 0, time.UTC),
		Finished: time.Date(2026, 5, 1, 14, 8, 30, 0, time.UTC),
		Results: []ResearchSummaryResult{
			{
				ID:          "abcdef12",
				Description: "buy milk",
				Status:      "findings",
				Summary:     "deploy lock",
				LogPath:     "research-logs/abcdef12_2026-05-01T14-00-00.jsonl",
				Tokens:      1234,
			},
			{
				ID:          "fedcba98",
				Description: "investigate dashboard",
				Status:      "clarify",
				Summary:     "what is X",
				LogPath:     "research-logs/fedcba98_2026-05-01T14-01-00.jsonl",
				Tokens:      500,
			},
			{
				ID:          "11111111",
				Description: "datadog hang",
				Status:      "failed",
				Summary:     "claude exited 1",
				LogPath:     "research-logs/11111111_2026-05-01T14-02-00.jsonl",
				Error:       "exit status 1",
			},
		},
		Totals: ResearchSummaryTotals{
			Findings: 1,
			Clarify:  1,
			Failed:   1,
			Skipped:  4,
		},
	})
	if err != nil {
		t.Fatalf("JSONResearchSummary: %v", err)
	}
	want, err := os.ReadFile("testdata/research_summary.json")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("JSON output does not match snapshot.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestHumanShow_unstyledOpenTaskPassesRawThrough(t *testing.T) {
	raw, err := os.ReadFile("testdata/show_open.md")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	tk, err := task.Decode(raw)
	if err != nil {
		t.Fatalf("task.Decode: %v", err)
	}

	var buf bytes.Buffer
	if err := HumanShow(&buf, tk, raw, "/some/path.md", false); err != nil {
		t.Fatalf("HumanShow: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), raw) {
		t.Errorf("HumanShow unstyled output does not match raw input.\n--- got ---\n%s\n--- want ---\n%s", buf.Bytes(), raw)
	}
}

func TestHumanShow_unstyledClosedTaskPassesRawThrough(t *testing.T) {
	raw, err := os.ReadFile("testdata/show_closed.md")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	tk, err := task.Decode(raw)
	if err != nil {
		t.Fatalf("task.Decode: %v", err)
	}
	if tk.Completed.IsZero() {
		t.Fatalf("snapshot must represent a closed task; Completed is zero")
	}

	var buf bytes.Buffer
	if err := HumanShow(&buf, tk, raw, "/some/path.md", false); err != nil {
		t.Fatalf("HumanShow: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), raw) {
		t.Errorf("HumanShow unstyled output does not match raw input.\n--- got ---\n%s\n--- want ---\n%s", buf.Bytes(), raw)
	}
}

func TestHumanCandidates_unstyledSingleMatchesSnapshot(t *testing.T) {
	candidates := []store.Candidate{
		{ID: "abcdef12", Description: "buy milk"},
	}

	var buf bytes.Buffer
	if err := HumanCandidates(&buf, candidates, false); err != nil {
		t.Fatalf("HumanCandidates: %v", err)
	}

	want, err := os.ReadFile("testdata/candidates_human_unstyled_single.txt")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("unstyled HumanCandidates does not match snapshot.\n--- got ---\n%s\n--- want ---\n%s", buf.Bytes(), want)
	}
}

func TestHumanCandidates_unstyledTruncatesLongDescription(t *testing.T) {
	candidates := []store.Candidate{
		{ID: "11111111", Description: strings.Repeat("x", 70)},
	}

	var buf bytes.Buffer
	if err := HumanCandidates(&buf, candidates, false); err != nil {
		t.Fatalf("HumanCandidates: %v", err)
	}

	want, err := os.ReadFile("testdata/candidates_human_unstyled_truncated.txt")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("unstyled HumanCandidates does not match snapshot.\n--- got ---\n%s\n--- want ---\n%s", buf.Bytes(), want)
	}
}

func TestHumanCandidates_unstyledMultipleMatchesSnapshot(t *testing.T) {
	candidates := []store.Candidate{
		{ID: "11111111", Description: "buy milk"},
		{ID: "22222222", Description: "buy milkshake"},
	}

	var buf bytes.Buffer
	if err := HumanCandidates(&buf, candidates, false); err != nil {
		t.Fatalf("HumanCandidates: %v", err)
	}

	want, err := os.ReadFile("testdata/candidates_human_unstyled.txt")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("unstyled HumanCandidates does not match snapshot.\n--- got ---\n%s\n--- want ---\n%s", buf.Bytes(), want)
	}
}

func TestHumanList_styledIncludesTagsColumn(t *testing.T) {
	tasks := []task.Task{
		{
			ID:      "11111111",
			Created: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC),
			Tags:    []string{"bug", "urgent"},
			Body:    "first task\n",
		},
		{
			ID:      "22222222",
			Created: time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC),
			Body:    "no tags task\n",
		},
	}

	var buf bytes.Buffer
	if err := HumanList(&buf, tasks, true); err != nil {
		t.Fatalf("HumanList styled: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "TAGS") {
		t.Errorf("styled HumanList must include TAGS header, got:\n%s", out)
	}
	if !strings.Contains(out, "[bug, urgent]") {
		t.Errorf("styled HumanList must show [bug, urgent] for tagged task, got:\n%s", out)
	}
	if !strings.Contains(out, "[]") {
		t.Errorf("styled HumanList must show [] for task with no tags, got:\n%s", out)
	}
}

func TestHumanList_unstyledIncludesTagsColumn(t *testing.T) {
	tasks := []task.Task{
		{
			ID:      "11111111",
			Created: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC),
			Tags:    []string{"bug", "urgent"},
			Body:    "first task\n",
		},
		{
			ID:      "22222222",
			Created: time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC),
			Body:    "no tags task\n",
		},
	}

	var buf bytes.Buffer
	if err := HumanList(&buf, tasks, false); err != nil {
		t.Fatalf("HumanList: %v", err)
	}

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d:\n%s", len(lines), out)
	}

	if !strings.Contains(lines[0], "[bug, urgent]") {
		t.Errorf("first line should contain [bug, urgent], got %q", lines[0])
	}
	if !strings.Contains(lines[1], "[]") {
		t.Errorf("second line should contain [] for empty tags, got %q", lines[1])
	}
	if !strings.Contains(lines[1], "no tags task") {
		t.Errorf("second line should still contain description, got %q", lines[1])
	}
}

func TestHumanList_unstyledEmptyList(t *testing.T) {
	var buf bytes.Buffer
	if err := HumanList(&buf, []task.Task{}, false); err != nil {
		t.Fatalf("HumanList: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for empty list, got %q", buf.String())
	}
}

func TestHumanList_unstyledMixedOpenClosedMatchesSnapshot(t *testing.T) {
	tasks := []task.Task{
		{
			ID:      "11111111",
			Created: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC),
			Body:    "first task\nbody continues here\n",
		},
		{
			ID:        "22222222",
			Created:   time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC),
			Completed: time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC),
			Body:      "shipped feature\n",
		},
	}

	var buf bytes.Buffer
	if err := HumanList(&buf, tasks, false); err != nil {
		t.Fatalf("HumanList: %v", err)
	}

	want, err := os.ReadFile("testdata/list_human_unstyled.txt")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("unstyled HumanList does not match snapshot.\n--- got ---\n%s\n--- want ---\n%s", buf.Bytes(), want)
	}
}

func TestHumanList_unstyledLongDescriptionsNotTruncated(t *testing.T) {
	long := strings.Repeat("very long description ", 20)
	tasks := []task.Task{
		{
			ID:      "abcdef12",
			Created: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC),
			Body:    long + "\n",
		},
	}

	var buf bytes.Buffer
	if err := HumanList(&buf, tasks, false); err != nil {
		t.Fatalf("HumanList: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, long) {
		t.Errorf("expected long description preserved verbatim in unstyled output, got %q", out)
	}
	if strings.Count(out, "\n") != 1 {
		t.Errorf("expected exactly one row (one newline) in unstyled output, got %d:\n%s", strings.Count(out, "\n"), out)
	}
	if strings.ContainsRune(out, '…') {
		t.Errorf("unstyled output must not truncate with ellipsis, got %q", out)
	}
}

func TestHumanShow_styledMetadataStripShowsTagsWhenPresent(t *testing.T) {
	tk := task.Task{
		ID:      "abcdef12",
		Created: time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC),
		Tags:    []string{"bug", "urgent"},
		Body:    "fix login\n",
	}

	var buf bytes.Buffer
	if err := HumanShow(&buf, tk, []byte("---\nid: abcdef12\n---\nfix login\n"), "/tasks/open/abc.md", true); err != nil {
		t.Fatalf("HumanShow styled: %v", err)
	}

	out := buf.String()
	lines := strings.Split(out, "\n")

	if !strings.Contains(lines[0], "[bug, urgent]") {
		t.Errorf("metadata strip must contain [bug, urgent] when tags present, got %q", lines[0])
	}
}

func TestHumanShow_styledMetadataStripOmitsTagsWhenEmpty(t *testing.T) {
	tk := task.Task{
		ID:      "abcdef12",
		Created: time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC),
		Body:    "buy milk\n",
	}

	var buf bytes.Buffer
	if err := HumanShow(&buf, tk, []byte("---\nid: abcdef12\n---\nbuy milk\n"), "/tasks/open/abc.md", true); err != nil {
		t.Fatalf("HumanShow styled: %v", err)
	}

	out := buf.String()
	lines := strings.Split(out, "\n")

	if strings.Contains(lines[0], "[") || strings.Contains(lines[0], "]") {
		t.Errorf("metadata strip must not contain bracket segment when tags empty, got %q", lines[0])
	}
}

func TestHumanShow_styledIncludesFaintPathLineBetweenMetadataAndBody(t *testing.T) {
	raw, err := os.ReadFile("testdata/show_open.md")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	tk, err := task.Decode(raw)
	if err != nil {
		t.Fatalf("task.Decode: %v", err)
	}

	taskPath := "/tasks/open/2026-04-29T11-30_abcdef12_buy-milk.md"

	var buf bytes.Buffer
	if err := HumanShow(&buf, tk, raw, taskPath, true); err != nil {
		t.Fatalf("HumanShow styled: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, taskPath) {
		t.Errorf("styled HumanShow output must contain the path %q, got:\n%s", taskPath, out)
	}

	lines := strings.Split(out, "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines (metadata, path, body), got %d:\n%s", len(lines), out)
	}

	if !strings.Contains(lines[1], taskPath) {
		t.Errorf("second line must contain the path; got line %q", lines[1])
	}

	if strings.Contains(lines[0], taskPath) {
		t.Errorf("first line (metadata strip) must not contain the path; got %q", lines[0])
	}
}

func TestHumanTagList_unstyledSpaceSeparatesEntries(t *testing.T) {
	tagCounts := []store.TagCount{
		{Name: "bug", Count: 3},
		{Name: "infra", Count: 5},
		{Name: "urgent", Count: 2},
	}
	var buf bytes.Buffer
	if err := HumanTagList(&buf, tagCounts, false); err != nil {
		t.Fatalf("HumanTagList: %v", err)
	}
	want, err := os.ReadFile("testdata/tag_ls_human_unstyled.txt")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("HumanTagList unstyled does not match snapshot.\n--- got ---\n%q\n--- want ---\n%q", buf.String(), string(want))
	}
}

func TestHumanTagList_emptyEmitsBlankLine(t *testing.T) {
	var buf bytes.Buffer
	if err := HumanTagList(&buf, nil, false); err != nil {
		t.Fatalf("HumanTagList: %v", err)
	}
	if buf.String() != "\n" {
		t.Errorf("empty HumanTagList = %q; want %q", buf.String(), "\n")
	}
}

func TestHumanTagList_styledMatchesUnstyled(t *testing.T) {
	tagCounts := []store.TagCount{
		{Name: "bug", Count: 3},
		{Name: "infra", Count: 5},
	}
	var styledBuf, unstyledBuf bytes.Buffer
	if err := HumanTagList(&styledBuf, tagCounts, true); err != nil {
		t.Fatalf("HumanTagList styled: %v", err)
	}
	if err := HumanTagList(&unstyledBuf, tagCounts, false); err != nil {
		t.Fatalf("HumanTagList unstyled: %v", err)
	}
	if !bytes.Equal(styledBuf.Bytes(), unstyledBuf.Bytes()) {
		t.Errorf("styled and unstyled outputs must be byte-identical for tag list.\n--- styled ---\n%q\n--- unstyled ---\n%q", styledBuf.String(), unstyledBuf.String())
	}
}

func TestJSONTagList_emptyEmitsTagsEmptyArray(t *testing.T) {
	got, err := JSONTagList(nil)
	if err != nil {
		t.Fatalf("JSONTagList: %v", err)
	}
	want := "{\n  \"tags\": []\n}\n"
	if string(got) != want {
		t.Errorf("empty JSONTagList:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestJSONTagList_matchesSnapshot(t *testing.T) {
	tagCounts := []store.TagCount{
		{Name: "bug", Count: 3},
		{Name: "infra", Count: 5},
		{Name: "urgent", Count: 2},
	}
	got, err := JSONTagList(tagCounts)
	if err != nil {
		t.Fatalf("JSONTagList: %v", err)
	}
	want, err := os.ReadFile("testdata/tag_ls.json")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("JSON output does not match snapshot.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestJSONShow_matchesSnapshot(t *testing.T) {
	tk := task.Task{
		ID:      "abcdef12",
		Created: time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC),
		Body:    "buy milk\nremember the brand\n",
	}

	got, err := JSONShow(tk, "/tasks/open/2026-04-29T11-30_abcdef12_buy-milk.md")
	if err != nil {
		t.Fatalf("JSONShow: %v", err)
	}
	want, err := os.ReadFile("testdata/show.json")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("JSON output does not match snapshot.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestJSONShow_researchedMatchesSnapshot(t *testing.T) {
	tk := task.Task{
		ID:              "abcdef12",
		Created:         time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC),
		LastResearched:  time.Date(2026, 5, 1, 14, 0, 0, 0, time.UTC),
		LastResearchLog: "/tasks/research-logs/abcdef12_2026-05-01T14-00-00.jsonl",
		Body:            "buy milk\nremember the brand\n",
	}

	got, err := JSONShow(tk, "/tasks/open/2026-04-29T11-30_abcdef12_buy-milk.md")
	if err != nil {
		t.Fatalf("JSONShow: %v", err)
	}
	want, err := os.ReadFile("testdata/show_researched.json")
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("JSON output does not match snapshot.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

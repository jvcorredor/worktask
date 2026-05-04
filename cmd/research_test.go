package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jvcorredor/bytheway/internal/research/runner"
	"github.com/jvcorredor/bytheway/internal/task"
)

// TestResearchCmd_batchForwardsCoordinatorOutputsIntoRender verifies the
// cmd shell stitches research.Coordinator outputs into the existing
// render.JSONResearchEvent / render.JSONResearchSummary functions
// correctly: per-task task_started/task_done/task_failed lines arrive on
// stdout, and the final summary line carries the right totals and
// per-task results. The real claude binary is never invoked — the test
// injects a stub research.RunFunc via the cmd-level seam.
func TestResearchCmd_batchForwardsCoordinatorOutputsIntoRender(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))

	openDir := filepath.Join(tmp, "worktask", "open")
	if err := os.MkdirAll(openDir, 0o755); err != nil {
		t.Fatalf("mkdir open: %v", err)
	}
	writeOpenTask(t, openDir, "aaaaaaaa", "alpha buy milk")
	writeOpenTask(t, openDir, "bbbbbbbb", "beta debug login")

	prevRun := researchRunFunc
	t.Cleanup(func() { researchRunFunc = prevRun })
	researchRunFunc = func(_ context.Context, in runner.RunInput) (runner.Result, error) {
		switch {
		case strings.Contains(in.Prompt, "alpha"):
			return runner.Result{
				Status:  "findings",
				Summary: "alpha summary",
				Body:    "alpha body",
				Tokens:  42,
			}, nil
		case strings.Contains(in.Prompt, "beta"):
			return runner.Result{}, errors.New("beta blew up")
		}
		t.Fatalf("unexpected prompt: %q", in.Prompt)
		return runner.Result{}, nil
	}

	prevConcurrency, prevAll := researchConcurrency, researchAll
	t.Cleanup(func() {
		researchConcurrency = prevConcurrency
		researchAll = prevAll
	})

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"research", "--all", "--concurrency=1"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute research: %v (stderr=%s)", err, stderr.String())
	}

	out := stdout.String()
	if out == "" {
		t.Fatalf("research --all produced no stdout")
	}

	// Each event is a single newline-terminated JSON object; the final
	// summary is a multi-line indented JSON object. json.Decoder handles
	// both shapes via successive Decode calls.
	dec := json.NewDecoder(strings.NewReader(out))
	var docs []map[string]any
	for {
		var m map[string]any
		err := dec.Decode(&m)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("decode JSON doc: %v\nremaining=%q", err, out)
		}
		docs = append(docs, m)
	}
	if len(docs) < 2 {
		t.Fatalf("expected at least one event + summary; got %d docs:\n%s", len(docs), out)
	}

	summary := docs[len(docs)-1]
	events := docs[:len(docs)-1]

	got := map[string]map[string]map[string]any{} // id -> event-type -> doc
	for _, ev := range events {
		evType, _ := ev["event"].(string)
		id, _ := ev["id"].(string)
		if got[id] == nil {
			got[id] = map[string]map[string]any{}
		}
		got[id][evType] = ev
	}

	if _, ok := got["aaaaaaaa"]["task_started"]; !ok {
		t.Errorf("missing task_started for alpha (got=%v)", events)
	}
	if done, ok := got["aaaaaaaa"]["task_done"]; !ok {
		t.Errorf("missing task_done for alpha (got=%v)", events)
	} else {
		if done["status"] != "findings" {
			t.Errorf("alpha task_done.status = %v; want findings", done["status"])
		}
		if done["summary"] != "alpha summary" {
			t.Errorf("alpha task_done.summary = %v; want alpha summary", done["summary"])
		}
	}

	if _, ok := got["bbbbbbbb"]["task_started"]; !ok {
		t.Errorf("missing task_started for beta (got=%v)", events)
	}
	if failed, ok := got["bbbbbbbb"]["task_failed"]; !ok {
		t.Errorf("missing task_failed for beta (got=%v)", events)
	} else if errStr, _ := failed["error"].(string); !strings.Contains(errStr, "beta blew up") {
		t.Errorf("beta task_failed.error = %q; want it to contain runner error", errStr)
	}

	totals, _ := summary["totals"].(map[string]any)
	if got, want := numField(totals, "findings"), 1.0; got != want {
		t.Errorf("totals.findings = %v; want %v", got, want)
	}
	if got, want := numField(totals, "failed"), 1.0; got != want {
		t.Errorf("totals.failed = %v; want %v", got, want)
	}
	if results, _ := summary["results"].([]any); len(results) != 2 {
		t.Errorf("len(summary.results) = %d; want 2", len(results))
	}
}

func writeOpenTask(t *testing.T, openDir, id, body string) {
	t.Helper()
	tk := task.Task{
		ID:      id,
		Created: time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC),
		Body:    body + "\n",
	}
	raw, err := task.Encode(tk)
	if err != nil {
		t.Fatalf("task.Encode: %v", err)
	}
	if err := os.WriteFile(filepath.Join(openDir, id+".md"), raw, 0o644); err != nil {
		t.Fatalf("write task file: %v", err)
	}
}

func numField(m map[string]any, k string) float64 {
	if m == nil {
		return -1
	}
	v, ok := m[k]
	if !ok {
		return -1
	}
	f, _ := v.(float64)
	return f
}

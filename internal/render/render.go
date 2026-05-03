// Package render formats tasks for human and JSON output.
//
// JSON list schema (stable):
//
//	[
//	  {
//	    "id":          string,  // 8-char lowercase hex
//	    "created":     string,  // RFC 3339 timestamp
//	    "description": string,  // first body line
//	    "completed":   string   // RFC 3339; omitted when task is open
//	  },
//	  ...
//	]
//
// JSON show schema (stable): same fields as the list element plus "body"
// (full task body, including the canonical first line as "description").
//
// JSON research-run schema (stable): per-task summary line emitted on stdout
// when `worktask research <hash>` completes.
//
//	{
//	  "id":          string,  // 8-char lowercase hex
//	  "description": string,  // first body line of the researched task
//	  "status":      string,  // "findings" | "clarify" | "failed"
//	  "summary":     string,  // one-line summary suitable for worklog and parent-session display
//	  "log_path":    string,  // relative path under tasks dir to the stream-json log file
//	  "error":       string   // omitted on success; populated when status is "failed"
//	}
package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jvcorredor/worktask/internal/store"
	"github.com/jvcorredor/worktask/internal/task"
)

type jsonTask struct {
	ID          string `json:"id"`
	Created     string `json:"created"`
	Description string `json:"description"`
	Completed   string `json:"completed,omitempty"`
}

type jsonShowTask struct {
	jsonTask
	Body string `json:"body"`
}

type jsonMatch struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type jsonErrAmbiguous struct {
	Error    string      `json:"error"`
	Fragment string      `json:"fragment"`
	Matches  []jsonMatch `json:"matches"`
}

type jsonErrNoMatch struct {
	Error     string     `json:"error"`
	Fragment  string     `json:"fragment"`
	OpenTasks []jsonTask `json:"open_tasks"`
}

// JSONErrorNoMatch renders the no-match error envelope: an object with
// error="no_match", the offending fragment, and the current open tasks
// for the user to choose from.
func JSONErrorNoMatch(fragment string, openTasks []task.Task) ([]byte, error) {
	tasks := make([]jsonTask, 0, len(openTasks))
	for _, t := range openTasks {
		tasks = append(tasks, toJSONTask(t))
	}
	return marshalIndent(jsonErrNoMatch{Error: "no_match", Fragment: fragment, OpenTasks: tasks})
}

// JSONErrorAmbiguous renders the ambiguous-fragment error envelope: an
// object with error="ambiguous", the offending fragment, and the
// candidate tasks the fragment matched.
func JSONErrorAmbiguous(fragment string, candidates []store.Candidate) ([]byte, error) {
	matches := make([]jsonMatch, 0, len(candidates))
	for _, c := range candidates {
		matches = append(matches, jsonMatch{ID: c.ID, Description: c.Description})
	}
	return marshalIndent(jsonErrAmbiguous{Error: "ambiguous", Fragment: fragment, Matches: matches})
}

// ResearchRun is the input shape for JSONResearchRun. Stable.
type ResearchRun struct {
	ID          string
	Description string
	Status      string
	Summary     string
	LogPath     string
	Error       string
}

type jsonResearchRun struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Summary     string `json:"summary"`
	LogPath     string `json:"log_path"`
	Error       string `json:"error,omitempty"`
}

// JSONResearchRun renders the per-task summary line emitted on stdout
// when `worktask research <hash>` completes. The output schema is
// described in the package documentation.
func JSONResearchRun(r ResearchRun) ([]byte, error) {
	return marshalIndent(jsonResearchRun(r))
}

// ResearchEvent is one streaming progress event emitted as a single JSON
// line on stdout while a batch runs.
type ResearchEvent struct {
	Type        string // "task_started" | "task_done" | "task_failed"
	ID          string
	Description string    // populated on task_started
	Started     time.Time // populated on task_started
	Status      string    // populated on task_done
	Summary     string    // populated on task_done
	Error       string    // populated on task_failed
}

type jsonResearchEvent struct {
	Event       string `json:"event"`
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
	Started     string `json:"started,omitempty"`
	Status      string `json:"status,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Error       string `json:"error,omitempty"`
}

// JSONResearchEvent renders one streaming progress event as a
// newline-terminated single-line JSON object suitable for stdout
// streaming during a research batch.
func JSONResearchEvent(e ResearchEvent) ([]byte, error) {
	out := jsonResearchEvent{
		Event:       e.Type,
		ID:          e.ID,
		Description: e.Description,
		Status:      e.Status,
		Summary:     e.Summary,
		Error:       e.Error,
	}
	if !e.Started.IsZero() {
		out.Started = e.Started.UTC().Format(time.RFC3339)
	}
	return marshalLine(out)
}

// ResearchSummary is the final per-batch summary written as the last
// stdout line when a sweep completes.
type ResearchSummary struct {
	BatchID  string
	Started  time.Time
	Finished time.Time
	Results  []ResearchSummaryResult
	Totals   ResearchSummaryTotals
}

// ResearchSummaryResult is one row of the per-batch summary.
type ResearchSummaryResult struct {
	ID          string
	Description string
	Status      string
	Summary     string
	LogPath     string
	Tokens      int
	Error       string
}

// ResearchSummaryTotals counts per-status outcomes plus the skipped count.
type ResearchSummaryTotals struct {
	Findings int
	Clarify  int
	Failed   int
	Skipped  int
}

type jsonResearchSummary struct {
	BatchID  string                      `json:"batch_id"`
	Started  string                      `json:"started"`
	Finished string                      `json:"finished"`
	Results  []jsonResearchSummaryResult `json:"results"`
	Totals   jsonResearchSummaryTotals   `json:"totals"`
}

type jsonResearchSummaryResult struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Summary     string `json:"summary"`
	LogPath     string `json:"log_path"`
	Tokens      int    `json:"tokens"`
	Error       string `json:"error,omitempty"`
}

type jsonResearchSummaryTotals struct {
	Findings int `json:"findings"`
	Clarify  int `json:"clarify"`
	Failed   int `json:"failed"`
	Skipped  int `json:"skipped"`
}

// JSONResearchSummary renders the final per-batch summary written as
// the last stdout line when a research sweep completes. Timestamps are
// emitted as RFC 3339 in UTC.
func JSONResearchSummary(s ResearchSummary) ([]byte, error) {
	results := make([]jsonResearchSummaryResult, 0, len(s.Results))
	for _, r := range s.Results {
		results = append(results, jsonResearchSummaryResult(r))
	}
	return marshalIndent(jsonResearchSummary{
		BatchID:  s.BatchID,
		Started:  s.Started.UTC().Format(time.RFC3339),
		Finished: s.Finished.UTC().Format(time.RFC3339),
		Results:  results,
		Totals: jsonResearchSummaryTotals{
			Findings: s.Totals.Findings,
			Clarify:  s.Totals.Clarify,
			Failed:   s.Totals.Failed,
			Skipped:  s.Totals.Skipped,
		},
	})
}

// marshalLine encodes v as a single-line JSON document terminated by '\n',
// suitable for streaming-event output where each line is a self-contained
// JSON object.
func marshalLine(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("render: encode: %w", err)
	}
	return buf.Bytes(), nil
}

// JSONShow renders a single task in the show schema described in the
// package documentation: the list-element fields plus the full body.
func JSONShow(t task.Task) ([]byte, error) {
	return marshalIndent(jsonShowTask{jsonTask: toJSONTask(t), Body: t.Body})
}

// JSONList renders tasks as the stable list-element schema described
// in the package documentation, in the order given.
func JSONList(tasks []task.Task) ([]byte, error) {
	out := make([]jsonTask, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, toJSONTask(t))
	}
	return marshalIndent(out)
}

func marshalIndent(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("render: encode: %w", err)
	}
	return buf.Bytes(), nil
}

func toJSONTask(t task.Task) jsonTask {
	out := jsonTask{
		ID:          t.ID,
		Created:     t.Created.Format(time.RFC3339),
		Description: description(t),
	}
	if !t.Completed.IsZero() {
		out.Completed = t.Completed.Format(time.RFC3339)
	}
	return out
}

func description(t task.Task) string {
	body := strings.TrimRight(t.Body, "\n")
	if i := strings.IndexByte(body, '\n'); i >= 0 {
		return body[:i]
	}
	return body
}

// HumanCandidates writes the human-readable disambiguation listing for
// candidates to w. When styled is false the rendering is byte-identical
// to today's plain output: "  <id>  <description>\n" per row, with the
// description truncated to 60 columns with a trailing ellipsis. When
// styled is true callers get the same layout with IDs in an accent
// colour. Returns the first write error encountered.
func HumanCandidates(w io.Writer, candidates []store.Candidate, styled bool) error {
	if styled {
		return humanCandidatesStyled(w, candidates)
	}
	const max = 60
	for _, c := range candidates {
		desc := c.Description
		if len(desc) > max {
			desc = desc[:max-1] + "…"
		}
		if _, err := fmt.Fprintf(w, "  %s  %s\n", c.ID, desc); err != nil {
			return err
		}
	}
	return nil
}

// HumanList writes one line per task to w. When styled is false the
// rendering is byte-identical to today's plain output:
// "<id>  <yyyy-mm-dd hh:mm>  <description>" per row, no header.
// When styled is true callers get a borderless lipgloss table with a
// header row, accent-coloured IDs, faint dates (date-only), and
// descriptions hard-truncated to fit the terminal width. Returns the
// first write error encountered.
func HumanList(w io.Writer, tasks []task.Task, styled bool) error {
	if styled {
		return humanListStyled(w, tasks)
	}
	for _, t := range tasks {
		if _, err := fmt.Fprintf(w, "%s  %s  %s\n", t.ID, t.Created.Format("2006-01-02 15:04"), description(t)); err != nil {
			return err
		}
	}
	return nil
}

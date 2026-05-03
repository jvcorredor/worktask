package task

import (
	"os"
	"testing"
	"time"
)

func TestRoundTrip_openTask(t *testing.T) {
	created := time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC)
	original := Task{
		ID:      "abcdef12",
		Created: created,
		Body:    "buy milk\n",
	}
	data, err := Encode(original)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.ID != original.ID {
		t.Errorf("ID = %q; want %q", got.ID, original.ID)
	}
	if !got.Created.Equal(original.Created) {
		t.Errorf("Created = %v; want %v", got.Created, original.Created)
	}
	if !got.Completed.IsZero() {
		t.Errorf("Completed = %v; want zero", got.Completed)
	}
	if got.Body != original.Body {
		t.Errorf("Body = %q; want %q", got.Body, original.Body)
	}
}

func TestRoundTrip_researchFieldsSet(t *testing.T) {
	created := time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC)
	lastResearched := time.Date(2026, 5, 1, 14, 0, 0, 0, time.UTC)
	original := Task{
		ID:              "abcdef12",
		Created:         created,
		LastResearched:  lastResearched,
		LastResearchLog: "research-logs/abcdef12_2026-05-01T14-00.jsonl",
		Body:            "buy milk\n",
	}
	data, err := Encode(original)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !got.LastResearched.Equal(original.LastResearched) {
		t.Errorf("LastResearched = %v; want %v", got.LastResearched, original.LastResearched)
	}
	if got.LastResearchLog != original.LastResearchLog {
		t.Errorf("LastResearchLog = %q; want %q", got.LastResearchLog, original.LastResearchLog)
	}
}

func TestEncode_researchFieldsWireFormat(t *testing.T) {
	tk := Task{
		ID:              "abcdef12",
		Created:         time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC),
		LastResearched:  time.Date(2026, 5, 1, 14, 0, 0, 0, time.UTC),
		LastResearchLog: "research-logs/abcdef12_2026-05-01T14-00.jsonl",
		Body:            "buy milk\n",
	}
	got, err := Encode(tk)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	want := "---\nid: abcdef12\ncreated: 2026-04-29T11:30:00Z\n" +
		"last_researched: 2026-05-01T14:00:00Z\n" +
		"last_research_log: research-logs/abcdef12_2026-05-01T14-00.jsonl\n" +
		"---\nbuy milk\n"
	if string(got) != want {
		t.Errorf("Encode bytes:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestDecode_legacyFixtureLeavesResearchFieldsZero(t *testing.T) {
	fixture, err := os.ReadFile("testdata/open_task.md")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	got, err := Decode(fixture)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !got.LastResearched.IsZero() {
		t.Errorf("LastResearched = %v; want zero", got.LastResearched)
	}
	if got.LastResearchLog != "" {
		t.Errorf("LastResearchLog = %q; want empty", got.LastResearchLog)
	}
}

func TestFormatStability_openTask(t *testing.T) {
	fixture, err := os.ReadFile("testdata/open_task.md")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	want := Task{
		ID:      "abcdef12",
		Created: time.Date(2026, 4, 29, 11, 30, 0, 0, time.UTC),
		Body:    "buy milk\n",
	}

	got, err := Decode(fixture)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.ID != want.ID || !got.Created.Equal(want.Created) || got.Body != want.Body || !got.Completed.IsZero() {
		t.Errorf("Decode(fixture) = %+v; want %+v", got, want)
	}

	encoded, err := Encode(want)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if string(encoded) != string(fixture) {
		t.Errorf("Encode(want) bytes do not match fixture.\n--- got ---\n%s\n--- want ---\n%s", encoded, fixture)
	}
}

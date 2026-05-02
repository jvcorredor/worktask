package runner

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type fakeExec struct {
	gotCmd   Command
	stdout   string
	err      error
	closeErr error // simulates the cmdReader.Close() result, e.g. "runner: claude exited 1: <stderr tail>"
}

type fakeReadCloser struct {
	io.Reader
	closeErr error
}

func (f *fakeReadCloser) Close() error { return f.closeErr }

func (f *fakeExec) Run(_ context.Context, cmd Command) (io.ReadCloser, error) {
	f.gotCmd = cmd
	if f.err != nil {
		return nil, f.err
	}
	return &fakeReadCloser{Reader: strings.NewReader(f.stdout), closeErr: f.closeErr}, nil
}

func TestBuildCommand_usesClaudeWithStreamJsonAndBypassPermissions(t *testing.T) {
	cmd := BuildCommand(RunInput{
		Prompt:  "render this task",
		LogPath: "/var/data/worktask/research-logs/abcdef12_2026-05-01T14-00.jsonl",
	})

	if cmd.Bin != "claude" {
		t.Errorf("Bin = %q; want %q", cmd.Bin, "claude")
	}
	wantArgs := []string{
		"-p",
		"--output-format=stream-json",
		"--verbose",
		"--permission-mode=bypassPermissions",
		"--model=" + DefaultModel,
	}
	for _, want := range wantArgs {
		if !contains(cmd.Args, want) {
			t.Errorf("Args missing %q; got %v", want, cmd.Args)
		}
	}
	if cmd.Stdin != "render this task" {
		t.Errorf("Stdin = %q; want %q", cmd.Stdin, "render this task")
	}
}

func TestBuildCommand_modelOverrideWinsOverDefault(t *testing.T) {
	cmd := BuildCommand(RunInput{
		Prompt: "x",
		Model:  "claude-haiku-4-5-20251001",
	})

	if !contains(cmd.Args, "--model=claude-haiku-4-5-20251001") {
		t.Errorf("Args missing --model override; got %v", cmd.Args)
	}
	if contains(cmd.Args, "--model="+DefaultModel) {
		t.Errorf("Args still contains the default model alongside the override; got %v", cmd.Args)
	}
}

func TestBuildCommand_defaultModelIsSonnet(t *testing.T) {
	if DefaultModel != "claude-sonnet-4-6" {
		t.Errorf("DefaultModel = %q; want claude-sonnet-4-6 (sonnet is the right default for retrieval+summarization)", DefaultModel)
	}
}

func TestBuildCommand_allowedToolsIsExactlyTheVanillaBaseline(t *testing.T) {
	// The shipped binary's baseline allowlist is the universal,
	// install-agnostic set of read-only built-ins. User MCPs and Bash(gh:*)
	// patterns are config-owned (loaded with validation in a follow-up
	// slice), not in-source.
	cmd := BuildCommand(RunInput{Prompt: "x"})

	allowed := allowedToolsFlag(cmd.Args)
	if allowed == "" {
		t.Fatalf("Args missing --allowed-tools flag; got %v", cmd.Args)
	}
	got := strings.Split(allowed, ",")

	want := []string{"Read", "Glob", "Grep", "WebSearch", "WebFetch"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("baseline allowed-tools mismatch:\n got: %v\nwant: %v", got, want)
	}
}

func TestBuildCommand_allowedToolsExcludesBuiltInWrites(t *testing.T) {
	// Safety net against accidental additions to the baseline. The shipped
	// binary must never grant the agent write-capable built-ins, and must
	// never grant unrestricted Bash (bare `Bash`) or the wildcard escape
	// hatch `Bash(*)`. MCP write methods and Bash(gh ...) write verbs are
	// not enumerated here — they are config-owned and rejected at config
	// load by the validator in a follow-up slice.
	cmd := BuildCommand(RunInput{Prompt: "x"})

	allowed := allowedToolsFlag(cmd.Args)
	tools := strings.Split(allowed, ",")
	set := make(map[string]bool, len(tools))
	for _, name := range tools {
		set[name] = true
	}

	mustNotHave := []string{
		"Bash",
		"Bash(*)",
		"Edit",
		"Write",
		"NotebookEdit",
		"TodoWrite",
	}
	for _, name := range mustNotHave {
		if set[name] {
			t.Errorf("baseline allowed-tools must NOT include write-capable %q", name)
		}
	}
}

func TestParseResult_findingsExtractsFinalFencedBlock(t *testing.T) {
	output := "I read the slack thread and the linked jira.\n" +
		"\n" +
		"```json\n" +
		`{"status":"findings","summary":"thread is about deploy lock","body":"## context\nfoo\n","questions":[],"sources":[{"kind":"slack","ref":"https://slack.com/x/y"}]}` + "\n" +
		"```\n"

	got, err := ParseResult(output)
	if err != nil {
		t.Fatalf("ParseResult: %v", err)
	}
	if got.Status != "findings" {
		t.Errorf("Status = %q; want %q", got.Status, "findings")
	}
	if got.Summary != "thread is about deploy lock" {
		t.Errorf("Summary = %q; want %q", got.Summary, "thread is about deploy lock")
	}
	if got.Body != "## context\nfoo\n" {
		t.Errorf("Body = %q", got.Body)
	}
	if len(got.Sources) != 1 || got.Sources[0].Kind != "slack" || got.Sources[0].Ref != "https://slack.com/x/y" {
		t.Errorf("Sources = %+v", got.Sources)
	}
}

func TestParseResult_clarifyExtractsQuestions(t *testing.T) {
	output := "Task body is too vague to anchor on.\n" +
		"\n" +
		"```json\n" +
		`{"status":"clarify","summary":"need more context","body":"","questions":["which jira project","which slack channel"],"sources":[]}` + "\n" +
		"```\n"

	got, err := ParseResult(output)
	if err != nil {
		t.Fatalf("ParseResult: %v", err)
	}
	if got.Status != "clarify" {
		t.Errorf("Status = %q; want %q", got.Status, "clarify")
	}
	if len(got.Questions) != 2 {
		t.Fatalf("len(Questions) = %d; want 2 (got %v)", len(got.Questions), got.Questions)
	}
	if got.Questions[0] != "which jira project" {
		t.Errorf("Questions[0] = %q", got.Questions[0])
	}
}

func TestParseResult_noFencedBlockReturnsError(t *testing.T) {
	output := "I have findings but I forgot to wrap them in a fenced JSON block.\n"
	if _, err := ParseResult(output); err == nil {
		t.Fatalf("ParseResult: want error, got nil")
	}
}

func TestParseResult_malformedJsonReturnsError(t *testing.T) {
	output := "```json\n{not valid json\n```\n"
	if _, err := ParseResult(output); err == nil {
		t.Fatalf("ParseResult: want error, got nil")
	}
}

func TestParseResult_extraProseAroundFencedBlockIsIgnored(t *testing.T) {
	output := "First I want to think out loud about the task.\n" +
		"There are several angles.\n" +
		"\n" +
		"```json\n" +
		`{"status":"findings","summary":"x","body":"y","questions":[],"sources":[]}` + "\n" +
		"```\n" +
		"\n" +
		"Postscript prose that the parser must ignore.\n"

	got, err := ParseResult(output)
	if err != nil {
		t.Fatalf("ParseResult: %v", err)
	}
	if got.Status != "findings" {
		t.Errorf("Status = %q; want findings", got.Status)
	}
	if got.Summary != "x" {
		t.Errorf("Summary = %q; want x", got.Summary)
	}
}

func TestParseResult_bodyContainingBacktickFenceDoesNotTruncate(t *testing.T) {
	// Regression: agents quote real-world tool output (terraform errors,
	// shell sessions) inside the body field as a markdown code block. The
	// ``` characters appear literally inside the JSON string value; a naive
	// strings.Index for the close fence matches them and truncates payload
	// mid-JSON. The close-fence search must find the LAST ``` in the block.
	bodyWithFence := "SSH fails:\n\n```\nError: Failed to download module\n```\n"
	payload := `{"status":"findings","summary":"dependabot ssh","body":` +
		strconv.Quote(bodyWithFence) +
		`,"questions":[],"sources":[]}`
	output := "Findings:\n\n```json\n" + payload + "\n```\n"

	got, err := ParseResult(output)
	if err != nil {
		t.Fatalf("ParseResult: %v", err)
	}
	if got.Status != "findings" {
		t.Errorf("Status = %q; want findings", got.Status)
	}
	if got.Body != bodyWithFence {
		t.Errorf("Body = %q; want %q (inner ``` must round-trip verbatim)", got.Body, bodyWithFence)
	}
}

func TestParseResult_picksLastFencedBlockWhenMultiple(t *testing.T) {
	output := "Earlier draft I'm discarding:\n" +
		"```json\n" +
		`{"status":"clarify","summary":"DRAFT","body":"","questions":[],"sources":[]}` + "\n" +
		"```\n" +
		"\n" +
		"Final answer:\n" +
		"```json\n" +
		`{"status":"findings","summary":"FINAL","body":"y","questions":[],"sources":[]}` + "\n" +
		"```\n"

	got, err := ParseResult(output)
	if err != nil {
		t.Fatalf("ParseResult: %v", err)
	}
	if got.Summary != "FINAL" {
		t.Errorf("Summary = %q; want FINAL (parser must pick the LAST fenced block)", got.Summary)
	}
}

func TestRun_extractsTokenUsageFromStreamResultEvent(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "research-logs", "abcdef12.jsonl")

	stream := strings.Join([]string{
		`{"type":"system","subtype":"init"}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"` + "```" + `json\n{\"status\":\"findings\",\"summary\":\"x\",\"body\":\"\",\"questions\":[],\"sources\":[]}\n` + "```" + `\n"}]}}`,
		`{"type":"result","subtype":"success","usage":{"input_tokens":1500,"output_tokens":480,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}`,
		"",
	}, "\n")

	got, err := Run(context.Background(), RunInput{Prompt: "x", LogPath: logPath}, &fakeExec{stdout: stream})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.Tokens != 1980 {
		t.Errorf("Tokens = %d; want 1980 (1500 input + 480 output)", got.Tokens)
	}
}

func TestRun_propagatesChildExitErrorWithStderrTail(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "research-logs", "abcdef12.jsonl")

	// Mirrors the slice-03 smoke test: claude rejected the args, exited 1
	// with a stderr message, and the runner saw zero stdout. The child
	// exit error (with its stderr tail) is what callers need to see.
	exitErr := errors.New("runner: claude exited 1: Error: When using --print, --output-format=stream-json requires --verbose")
	fx := &fakeExec{stdout: "", closeErr: exitErr}

	_, err := Run(context.Background(), RunInput{Prompt: "x", LogPath: logPath}, fx)
	if err == nil {
		t.Fatalf("Run: want error, got nil")
	}
	if !strings.Contains(err.Error(), "claude exited 1") {
		t.Errorf("error = %q; want one containing the child exit code", err.Error())
	}
	if !strings.Contains(err.Error(), "requires --verbose") {
		t.Errorf("error = %q; want one containing the stderr tail", err.Error())
	}
	if strings.Contains(err.Error(), "no output") || strings.Contains(err.Error(), "fenced block") {
		t.Errorf("error = %q; the exec-layer error must outrank the empty-stream and parse messages", err.Error())
	}
}

func TestRun_streamWithoutAssistantTextReturnsNoOutputError(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "research-logs", "abcdef12.jsonl")

	// Child emitted some stream events but never produced an assistant
	// text block — so assembled output is empty. Falling through to the
	// parser would surface "no fenced block", which mis-blames the parser
	// for the agent's silence.
	stream := strings.Join([]string{
		`{"type":"system","subtype":"init"}`,
		`{"type":"result","subtype":"success","usage":{"input_tokens":10,"output_tokens":0}}`,
		"",
	}, "\n")

	_, err := Run(context.Background(), RunInput{Prompt: "x", LogPath: logPath}, &fakeExec{stdout: stream})
	if err == nil {
		t.Fatalf("Run: want error, got nil")
	}
	if !strings.Contains(err.Error(), "no output") {
		t.Errorf("error = %q; want one mentioning 'no output' when the agent emits stream events but no assistant text", err.Error())
	}
	if strings.Contains(err.Error(), "fenced block") {
		t.Errorf("error = %q; should not delegate to the parser when the assistant said nothing", err.Error())
	}
}

func TestRun_emptyStreamReturnsDistinctNoOutputError(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "research-logs", "abcdef12.jsonl")

	// Child wrote nothing to stdout and exited cleanly. The runner must
	// distinguish this ("agent emitted no output") from the parser's
	// "no fenced block" complaint, which would mislead callers about the
	// root cause when the real failure is upstream of the parser.
	got, err := Run(context.Background(), RunInput{Prompt: "x", LogPath: logPath}, &fakeExec{stdout: ""})
	if err == nil {
		t.Fatalf("Run: want error, got result %+v", got)
	}
	if !strings.Contains(err.Error(), "no output") {
		t.Errorf("error = %q; want one mentioning 'no output'", err.Error())
	}
	if strings.Contains(err.Error(), "fenced block") {
		t.Errorf("error = %q; should NOT delegate to the parser's no-fenced-block message", err.Error())
	}
}

func TestRun_writesStreamToLogAndReturnsParsedResult(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "research-logs", "abcdef12_2026-05-01T14-00.jsonl")

	stream := strings.Join([]string{
		`{"type":"system","subtype":"init"}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"I read the slack thread.\n\n"}]}}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"` + "```" + `json\n{\"status\":\"findings\",\"summary\":\"deploy lock\",\"body\":\"## context\\nfoo\\n\",\"questions\":[],\"sources\":[]}\n` + "```" + `\n"}]}}`,
		`{"type":"result","subtype":"success"}`,
		"",
	}, "\n")

	fx := &fakeExec{stdout: stream}
	got, err := Run(context.Background(), RunInput{Prompt: "render this", LogPath: logPath}, fx)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got.Status != "findings" || got.Summary != "deploy lock" {
		t.Errorf("Result = %+v; want findings/'deploy lock'", got)
	}

	logged, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if string(logged) != stream {
		t.Errorf("log contents:\n--- got ---\n%s\n--- want ---\n%s", logged, stream)
	}

	if fx.gotCmd.Bin != "claude" {
		t.Errorf("exec received Bin=%q; want claude", fx.gotCmd.Bin)
	}
	if fx.gotCmd.Stdin != "render this" {
		t.Errorf("exec received Stdin=%q; want %q", fx.gotCmd.Stdin, "render this")
	}
}

func contains(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

func allowedToolsFlag(args []string) string {
	const prefix = "--allowed-tools="
	for _, a := range args {
		if strings.HasPrefix(a, prefix) {
			return strings.TrimPrefix(a, prefix)
		}
	}
	return ""
}

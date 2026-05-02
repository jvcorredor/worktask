// Package runner spawns the headless claude agent that performs research,
// tees its stream-json output to a log file, and parses the agent's final
// fenced JSON block into a typed Result.
//
// The exec layer is exposed via BuildCommand for test inspection; the real
// claude binary is never invoked by unit tests.
package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DefaultModel is the model the research agent uses when no override is
// supplied. Sonnet is the right default for retrieval+summarization across
// MCP tools; the caller's interactive default (often Opus) would burn the
// usage limit on every batch sweep without a meaningful quality gain here.
const DefaultModel = "claude-sonnet-4-6"

// RunInput is the input to a single research run.
type RunInput struct {
	// Prompt is the rendered prompt sent to the agent on stdin.
	Prompt string
	// LogPath is the absolute path the stream-json log will be written to.
	LogPath string
	// Model overrides the model the headless agent runs as. Empty means
	// DefaultModel.
	Model string
}

// Command is a description of the child process to spawn. It is returned
// by BuildCommand so that tests can verify the argv construction without
// executing claude.
type Command struct {
	Bin   string
	Args  []string
	Stdin string
}

// allowedTools is the install-agnostic baseline allowlist passed to claude
// via --allowed-tools. It contains only built-in read-only tools that every
// install has — no MCPs, no Bash patterns. The shipped binary cannot assume
// any particular MCP server is installed in the user's environment, so
// install-specific tool wiring (Slack/Jira/Datadog MCPs, Bash(gh:*) patterns,
// etc.) is config-owned: users opt in via worktask config, which a
// follow-up slice will load with validation that rejects bare Bash and any
// non-`mcp__*`/non-`Bash(...)` entry.
//
// Adding a tool here means it must be safe for every install, with no
// runtime configuration. The bar is "read-only built-in." Any tool that
// requires user-side setup belongs in config, not here.
var allowedTools = []string{
	"Read",
	"Glob",
	"Grep",
	"WebSearch",
	"WebFetch",
}

// Source cites where an agent claim came from.
type Source struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}

// Result is the parsed output of one research run.
type Result struct {
	Status    string   `json:"status"`
	Summary   string   `json:"summary"`
	Body      string   `json:"body"`
	Questions []string `json:"questions"`
	Sources   []Source `json:"sources"`
	Tokens    int      `json:"-"` // sum of input+output tokens from the stream-json result event, not the agent's fenced JSON block
}

// ParseResult extracts the final ```json fenced block from the agent's
// textual output and decodes it into a Result.
func ParseResult(output string) (Result, error) {
	const open = "```json"
	const closeFence = "```"

	lastOpen := strings.LastIndex(output, open)
	if lastOpen < 0 {
		return Result{}, fmt.Errorf("runner: no ```json fenced block in output")
	}
	rest := output[lastOpen+len(open):]
	rest = strings.TrimLeft(rest, "\r\n")

	// LastIndex (not Index): agents quote tool output inside `body` as a
	// markdown code block, so ``` characters appear inside the JSON string
	// value. The actual close fence is always the last ``` in the block.
	closeIdx := strings.LastIndex(rest, closeFence)
	if closeIdx < 0 {
		return Result{}, fmt.Errorf("runner: unclosed ```json fenced block")
	}
	payload := rest[:closeIdx]

	var r Result
	if err := json.Unmarshal([]byte(payload), &r); err != nil {
		return Result{}, fmt.Errorf("runner: decode result json: %w", err)
	}
	return r, nil
}

// Exec is the injection point for the child-process runner. The real
// implementation spawns claude; tests inject a fake.
type Exec interface {
	Run(ctx context.Context, cmd Command) (io.ReadCloser, error)
}

// Run spawns claude via the injected Exec, tees the stream-json stdout to
// LogPath, parses the agent's textual response into a Result, and returns it.
func Run(ctx context.Context, in RunInput, exec Exec) (Result, error) {
	cmd := BuildCommand(in)

	if err := os.MkdirAll(filepath.Dir(in.LogPath), 0o755); err != nil {
		return Result{}, fmt.Errorf("runner: mkdir log dir: %w", err)
	}
	logFile, err := os.Create(in.LogPath)
	if err != nil {
		return Result{}, fmt.Errorf("runner: create log: %w", err)
	}
	defer logFile.Close()

	stdout, err := exec.Run(ctx, cmd)
	if err != nil {
		return Result{}, fmt.Errorf("runner: exec: %w", err)
	}

	var assembled strings.Builder
	var tokens int
	var lines int
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		lines++
		line := scanner.Bytes()
		if _, werr := logFile.Write(append(append([]byte{}, line...), '\n')); werr != nil {
			_ = stdout.Close()
			return Result{}, fmt.Errorf("runner: write log: %w", werr)
		}
		text := extractAssistantText(line)
		if text != "" {
			assembled.WriteString(text)
		}
		if t := extractTokens(line); t > 0 {
			tokens += t
		}
	}
	if serr := scanner.Err(); serr != nil {
		_ = stdout.Close()
		return Result{}, fmt.Errorf("runner: scan stdout: %w", serr)
	}

	// Close reaps the child and surfaces a non-zero exit (with stderr tail)
	// before we get to lines/parse checks; the exit error is the most
	// diagnostic signal we have when the child rejected its args.
	if cerr := stdout.Close(); cerr != nil {
		return Result{}, cerr
	}

	if lines == 0 || assembled.Len() == 0 {
		return Result{}, fmt.Errorf("runner: agent emitted no output")
	}

	result, err := ParseResult(assembled.String())
	if err != nil {
		return Result{}, err
	}
	result.Tokens = tokens
	return result, nil
}

type streamEvent struct {
	Type    string `json:"type"`
	Message struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
}

type resultEvent struct {
	Type  string `json:"type"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func extractTokens(line []byte) int {
	if len(line) == 0 {
		return 0
	}
	var ev resultEvent
	if err := json.Unmarshal(line, &ev); err != nil {
		return 0
	}
	if ev.Type != "result" {
		return 0
	}
	return ev.Usage.InputTokens + ev.Usage.OutputTokens
}

func extractAssistantText(line []byte) string {
	if len(line) == 0 {
		return ""
	}
	var ev streamEvent
	if err := json.Unmarshal(line, &ev); err != nil {
		return ""
	}
	if ev.Type != "assistant" {
		return ""
	}
	var b strings.Builder
	for _, c := range ev.Message.Content {
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	return b.String()
}

// BuildCommand returns the exec spec the runner will use to spawn claude.
// Exposed for test inspection.
func BuildCommand(in RunInput) Command {
	model := in.Model
	if model == "" {
		model = DefaultModel
	}
	return Command{
		Bin: "claude",
		Args: []string{
			"-p",
			"--output-format=stream-json",
			// --verbose is required: claude rejects -p with stream-json output unless verbose is set.
			"--verbose",
			"--permission-mode=bypassPermissions",
			"--model=" + model,
			"--allowed-tools=" + strings.Join(allowedTools, ","),
		},
		Stdin: in.Prompt,
	}
}

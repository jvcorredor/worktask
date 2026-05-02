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

// allowedTools is the explicit read-only allowlist passed to claude via
// --allowed-tools. Adding or removing a tool is a code review step, not
// an emergent behavior. Disallowed by omission: Edit, Write, NotebookEdit,
// TodoWrite, any MCP write method, all Gmail/HubSpot/Calendar tools.
//
// Bash is allowed only under explicit subcommand patterns (e.g.
// Bash(gh issue view:*)); bare Bash is denied. The argv-pattern check
// in claude is the safety boundary, so any new Bash entry must be a
// prefix that no write-capable subcommand can match.
//
// MCP tool names must be fully qualified as mcp__<server>__<tool>. The
// bare-name form is silently denied even with --permission-mode=bypassPermissions.
var allowedTools = []string{
	// Built-in read-only.
	"Read",
	"Glob",
	"Grep",
	"WebSearch",
	"WebFetch",

	// Atlassian MCP, read-only subset.
	"mcp__claude_ai_Atlassian__getJiraIssue",
	"mcp__claude_ai_Atlassian__searchJiraIssuesUsingJql",
	"mcp__claude_ai_Atlassian__getJiraIssueRemoteIssueLinks",
	"mcp__claude_ai_Atlassian__getConfluencePage",
	"mcp__claude_ai_Atlassian__searchConfluenceUsingCql",
	"mcp__claude_ai_Atlassian__getConfluencePageDescendants",
	"mcp__claude_ai_Atlassian__getConfluencePageFooterComments",
	"mcp__claude_ai_Atlassian__getConfluencePageInlineComments",
	"mcp__claude_ai_Atlassian__getConfluenceCommentChildren",
	"mcp__claude_ai_Atlassian__getPagesInConfluenceSpace",
	"mcp__claude_ai_Atlassian__getConfluenceSpaces",
	"mcp__claude_ai_Atlassian__getIssueLinkTypes",
	"mcp__claude_ai_Atlassian__getJiraProjectIssueTypesMetadata",
	"mcp__claude_ai_Atlassian__getJiraIssueTypeMetaWithFields",
	"mcp__claude_ai_Atlassian__getTransitionsForJiraIssue",
	"mcp__claude_ai_Atlassian__getVisibleJiraProjects",
	"mcp__claude_ai_Atlassian__lookupJiraAccountId",
	"mcp__claude_ai_Atlassian__atlassianUserInfo",
	"mcp__claude_ai_Atlassian__getAccessibleAtlassianResources",
	"mcp__claude_ai_Atlassian__search",
	"mcp__claude_ai_Atlassian__fetch",

	// Datadog MCP — entirely read-only by design.
	"mcp__claude_ai_Datadog__aggregate_events",
	"mcp__claude_ai_Datadog__aggregate_rum_events",
	"mcp__claude_ai_Datadog__aggregate_spans",
	"mcp__claude_ai_Datadog__analyze_datadog_logs",
	"mcp__claude_ai_Datadog__get_datadog_dashboard",
	"mcp__claude_ai_Datadog__get_datadog_incident",
	"mcp__claude_ai_Datadog__get_datadog_metric",
	"mcp__claude_ai_Datadog__get_datadog_metric_context",
	"mcp__claude_ai_Datadog__get_datadog_notebook",
	"mcp__claude_ai_Datadog__get_datadog_trace",
	"mcp__claude_ai_Datadog__search_datadog_dashboards",
	"mcp__claude_ai_Datadog__search_datadog_events",
	"mcp__claude_ai_Datadog__search_datadog_hosts",
	"mcp__claude_ai_Datadog__search_datadog_incidents",
	"mcp__claude_ai_Datadog__search_datadog_logs",
	"mcp__claude_ai_Datadog__search_datadog_metrics",
	"mcp__claude_ai_Datadog__search_datadog_metrics_v2",
	"mcp__claude_ai_Datadog__search_datadog_monitors",
	"mcp__claude_ai_Datadog__search_datadog_notebooks",
	"mcp__claude_ai_Datadog__search_datadog_rum_events",
	"mcp__claude_ai_Datadog__search_datadog_service_dependencies",
	"mcp__claude_ai_Datadog__search_datadog_services",
	"mcp__claude_ai_Datadog__search_datadog_spans",

	// Slack MCP, read-only subset.
	"mcp__claude_ai_Slack__slack_read_channel",
	"mcp__claude_ai_Slack__slack_read_thread",
	"mcp__claude_ai_Slack__slack_read_canvas",
	"mcp__claude_ai_Slack__slack_read_user_profile",
	"mcp__claude_ai_Slack__slack_search_channels",
	"mcp__claude_ai_Slack__slack_search_public",
	"mcp__claude_ai_Slack__slack_search_public_and_private",
	"mcp__claude_ai_Slack__slack_search_users",

	// Unblocked MCP — entirely read-only by design.
	"mcp__unblocked__context_research",
	"mcp__unblocked__context_get_urls",

	// context7 MCP — library / framework / SDK / CLI documentation
	// lookups. Read-only by design.
	"mcp__context7__resolve-library-id",
	"mcp__context7__query-docs",

	// gh CLI, read-only verbs only. The agent inherits the parent
	// process's gh auth state. gh's write subcommands (create, edit,
	// close, merge, comment, delete, review, lock, pin, develop) live
	// under different verbs, so the prefixes below cannot match them.
	//
	// gh api is intentionally NOT on this list. It can issue arbitrary
	// HTTP methods (gh api -X POST/PATCH/DELETE, or -f field=value which
	// auto-switches to POST), so the read/write boundary would collapse
	// to the gh auth token's scopes rather than the argv pattern.
	"Bash(gh issue view:*)",
	"Bash(gh issue list:*)",
	"Bash(gh pr view:*)",
	"Bash(gh pr list:*)",
	"Bash(gh pr diff:*)",
	"Bash(gh search:*)",
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

package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/jvcorredor/worktask/internal/config"
	"github.com/jvcorredor/worktask/internal/render"
	researchlog "github.com/jvcorredor/worktask/internal/research/log"
	"github.com/jvcorredor/worktask/internal/research/pool"
	"github.com/jvcorredor/worktask/internal/research/prompt"
	"github.com/jvcorredor/worktask/internal/research/runner"
	"github.com/jvcorredor/worktask/internal/research/writeback"
	"github.com/jvcorredor/worktask/internal/store"
)

var (
	researchConcurrency int
	researchAll         bool
	researchStale       time.Duration
	researchTimeout     time.Duration
	researchModel       string
)

const (
	defaultResearchConcurrency = 3
	defaultResearchTimeout     = 10 * time.Minute
)

var researchCmd = &cobra.Command{
	Use:   "research [<fragment>]",
	Short: "Research one task or sweep every open task via headless claude agents",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		s := store.New(cfg.TasksDir)

		if len(args) == 1 {
			return runSingleResearch(cmd, cfg, s, args[0])
		}
		return runBatchResearch(cmd, cfg, s)
	},
}

func init() {
	researchCmd.Flags().IntVar(&researchConcurrency, "concurrency", defaultResearchConcurrency, "max concurrent in-flight research runs")
	researchCmd.Flags().BoolVar(&researchAll, "all", false, "force re-research of every open task, ignoring last_researched")
	researchCmd.Flags().DurationVar(&researchStale, "stale", 0, "re-research tasks whose last_researched is older than this duration")
	researchCmd.Flags().DurationVar(&researchTimeout, "timeout", defaultResearchTimeout, "per-task hard timeout")
	researchCmd.Flags().StringVar(&researchModel, "model", "", "claude model id for the research agent; overrides config.research_model")
	rootCmd.AddCommand(researchCmd)
}

func runSingleResearch(cmd *cobra.Command, cfg config.Config, s *store.Store, fragment string) error {
	t, _, _, err := s.Get(fragment)
	if err != nil {
		return handleResolveError(cmd, s, fragment, err)
	}

	tmpl, err := prompt.Load(cfg.ResearchPromptPath)
	if err != nil {
		return err
	}
	rendered, err := prompt.Render(tmpl, t)
	if err != nil {
		return err
	}

	runStart := time.Now().UTC()
	logPath := researchlog.Path(cfg.TasksDir, t.ID, runStart)
	logPathRel, err := filepath.Rel(cfg.TasksDir, logPath)
	if err != nil {
		logPathRel = logPath
	}

	result, runErr := runner.Run(context.Background(), runner.RunInput{
		Prompt:     rendered,
		LogPath:    logPath,
		Model:      resolveResearchModel(cfg),
		ExtraTools: cfg.ResearchExtraTools,
	}, runner.OSExec{})
	if runErr != nil {
		result = runner.Result{
			Status:  "failed",
			Summary: runErr.Error(),
		}
	}

	if err := writeback.Apply(s, writeback.Input{
		Fragment:       t.ID,
		Result:         result,
		RunStart:       runStart,
		LogPath:        logPath,
		LogPathRel:     logPathRel,
		WorkingLogPath: cfg.WorkingLogPath,
	}); err != nil {
		return err
	}

	summary, err := render.JSONResearchRun(render.ResearchRun{
		ID:          t.ID,
		Description: firstLineOfBody(t.Body),
		Status:      result.Status,
		Summary:     result.Summary,
		LogPath:     logPathRel,
		Error:       errString(runErr),
	})
	if err != nil {
		return err
	}
	if _, err := cmd.OutOrStdout().Write(summary); err != nil {
		return err
	}
	if result.Status == "failed" {
		return errExit
	}
	return nil
}

// itemMeta keeps the per-task absolute log path next to the queue. The pool
// only echoes back the relative path it was given (via Item.LogPath) for
// output, so the absolute path must be looked up in cmd/ for runner +
// writeback callbacks.
type itemMeta struct {
	logPath    string
	logPathRel string
	runStart   time.Time
}

func runBatchResearch(cmd *cobra.Command, cfg config.Config, s *store.Store) error {
	openTasks, err := s.List(store.FilterOpen, 0, "")
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	toResearch, skipped := partitionResearchCandidates(openTasks, skipFlags{
		all:   researchAll,
		stale: researchStale,
	}, now)

	tmpl, err := prompt.Load(cfg.ResearchPromptPath)
	if err != nil {
		return err
	}

	items := make([]pool.Item, 0, len(toResearch))
	meta := make(map[string]itemMeta, len(toResearch))
	for _, t := range toResearch {
		rendered, err := prompt.Render(tmpl, t)
		if err != nil {
			return err
		}
		runStart := time.Now().UTC()
		logPath := researchlog.Path(cfg.TasksDir, t.ID, runStart)
		logPathRel, err := filepath.Rel(cfg.TasksDir, logPath)
		if err != nil {
			logPathRel = logPath
		}
		items = append(items, pool.Item{
			Task:    t,
			Prompt:  rendered,
			LogPath: logPathRel,
		})
		meta[t.ID] = itemMeta{logPath: logPath, logPathRel: logPathRel, runStart: runStart}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	model := resolveResearchModel(cfg)
	extraTools := cfg.ResearchExtraTools
	runFn := func(ctx context.Context, item pool.Item) (runner.Result, error) {
		return runner.Run(ctx, runner.RunInput{
			Prompt:     item.Prompt,
			LogPath:    meta[item.Task.ID].logPath,
			Model:      model,
			ExtraTools: extraTools,
		}, runner.OSExec{})
	}

	res := pool.Run(ctx, items, runFn, pool.Opts{
		Concurrency: researchConcurrency,
		Timeout:     researchTimeout,
	})

	stdout := cmd.OutOrStdout()
	for ev := range res.Events {
		emitProgressEvent(stdout, ev)
		if ev.Type == "task_done" {
			m := meta[ev.Task.ID]
			_ = writeback.Apply(s, writeback.Input{
				Fragment:       ev.Task.ID,
				Result:         ev.Result,
				RunStart:       m.runStart,
				LogPath:        m.logPath,
				LogPathRel:     m.logPathRel,
				WorkingLogPath: cfg.WorkingLogPath,
			})
		}
	}
	summary := <-res.Done

	final := render.ResearchSummary{
		BatchID:  summary.BatchID,
		Started:  summary.Started,
		Finished: summary.Finished,
		Totals: render.ResearchSummaryTotals{
			Findings: summary.Totals.Findings,
			Clarify:  summary.Totals.Clarify,
			Failed:   summary.Totals.Failed,
			Skipped:  len(skipped),
		},
	}
	for _, r := range summary.Results {
		final.Results = append(final.Results, render.ResearchSummaryResult{
			ID:          r.ID,
			Description: r.Description,
			Status:      r.Status,
			Summary:     r.Summary,
			LogPath:     r.LogPath,
			Tokens:      r.Tokens,
			Error:       r.Error,
		})
	}
	out, err := render.JSONResearchSummary(final)
	if err != nil {
		return err
	}
	_, err = stdout.Write(out)
	return err
}

func emitProgressEvent(stdout io.Writer, ev pool.Event) {
	re := render.ResearchEvent{Type: ev.Type, ID: ev.Task.ID}
	switch ev.Type {
	case "task_started":
		re.Description = firstLineOfBody(ev.Task.Body)
		re.Started = ev.Started
	case "task_done":
		re.Status = ev.Result.Status
		re.Summary = ev.Result.Summary
	case "task_failed":
		re.Error = errString(ev.Err)
	}
	line, err := render.JSONResearchEvent(re)
	if err != nil {
		return
	}
	_, _ = stdout.Write(line)
}

func firstLineOfBody(body string) string {
	for i := 0; i < len(body); i++ {
		if body[i] == '\n' {
			return body[:i]
		}
	}
	return body
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprint(err)
}

// resolveResearchModel picks the model id to pass to the runner.
// Precedence: --model flag > config.research_model > runner default.
func resolveResearchModel(cfg config.Config) string {
	if researchModel != "" {
		return researchModel
	}
	return cfg.ResearchModel
}

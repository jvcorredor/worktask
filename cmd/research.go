package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/jvcorredor/worktask/internal/config"
	"github.com/jvcorredor/worktask/internal/render"
	"github.com/jvcorredor/worktask/internal/research"
	"github.com/jvcorredor/worktask/internal/research/prompt"
	"github.com/jvcorredor/worktask/internal/store"
)

var (
	researchConcurrency int
	researchAll         bool
	researchStale       time.Duration
	researchTimeout     time.Duration
	researchModel       string
	researchTag         string
)

const (
	defaultResearchConcurrency = 3
	defaultResearchTimeout     = 10 * time.Minute
)

// researchRunFunc is the seam the cmd shell hands to research.New. The
// production wiring runs the headless claude binary via the runner
// subpackage; tests substitute an in-memory stub to avoid invoking
// claude.
var researchRunFunc research.RunFunc = research.DefaultRun

var researchCmd = &cobra.Command{
	Use:   "research [<fragment>]",
	Short: "Research one task or sweep every open task via headless claude agents",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 && cmd.Flags().Changed("tag") {
			return fmt.Errorf("--tag is not valid in single-task mode (drop the fragment to sweep all tagged tasks)")
		}
		normalizedTag, err := validateResearchTag(researchTag)
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		s := store.New(cfg.TasksDir)

		tmpl, err := prompt.Load(cfg.ResearchPromptPath)
		if err != nil {
			return err
		}

		coord, err := research.New(s, researchRunFunc, research.Config{
			TasksDir:       cfg.TasksDir,
			WorkingLogPath: cfg.WorkingLogPath,
			Model:          resolveResearchModel(cfg),
			ExtraTools:     cfg.ResearchExtraTools,
		}, tmpl)
		if err != nil {
			return err
		}

		if len(args) == 1 {
			return runSingleResearch(cmd, s, coord, args[0])
		}
		return runBatchResearch(cmd, s, coord, normalizedTag)
	},
}

func init() {
	researchCmd.Flags().IntVar(&researchConcurrency, "concurrency", defaultResearchConcurrency, "max concurrent in-flight research runs")
	researchCmd.Flags().BoolVar(&researchAll, "all", false, "force re-research of every open task, ignoring last_researched")
	researchCmd.Flags().DurationVar(&researchStale, "stale", 0, "re-research tasks whose last_researched is older than this duration")
	researchCmd.Flags().DurationVar(&researchTimeout, "timeout", defaultResearchTimeout, "per-task hard timeout")
	researchCmd.Flags().StringVar(&researchModel, "model", "", "claude model id for the research agent; overrides config.research_model")
	researchCmd.Flags().StringVar(&researchTag, "tag", "", "in batch mode, restrict the sweep to open tasks carrying this tag (exact match)")
	rootCmd.AddCommand(researchCmd)
}

func runSingleResearch(cmd *cobra.Command, s *store.Store, coord *research.Coordinator, fragment string) error {
	t, _, _, err := s.Get(fragment)
	if err != nil {
		return handleResolveError(cmd, s, fragment, err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	out, err := coord.RunOne(ctx, t)
	if err != nil {
		return err
	}

	line, err := render.JSONResearchRun(render.ResearchRun{
		ID:          out.ID,
		Description: out.Description,
		Status:      out.Status,
		Summary:     out.Summary,
		LogPath:     out.LogPath,
		Error:       out.Error,
	})
	if err != nil {
		return err
	}
	if _, err := cmd.OutOrStdout().Write(line); err != nil {
		return err
	}
	if out.Status == "failed" {
		return errExit
	}
	return nil
}

func runBatchResearch(cmd *cobra.Command, s *store.Store, coord *research.Coordinator, tagFilter string) error {
	openTasks, err := s.List(store.FilterOpen, 0, tagFilter)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	toResearch, skipped := partitionResearchCandidates(openTasks, skipFlags{
		all:   researchAll,
		stale: researchStale,
	}, now)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	events, done, err := coord.RunBatch(ctx, toResearch, len(skipped), research.BatchOpts{
		Concurrency: researchConcurrency,
		Timeout:     researchTimeout,
	})
	if err != nil {
		return err
	}

	stdout := cmd.OutOrStdout()
	for ev := range events {
		emitProgressEvent(stdout, ev)
	}
	summary := <-done

	final := render.ResearchSummary{
		BatchID:  summary.BatchID,
		Started:  summary.Started,
		Finished: summary.Finished,
		Totals: render.ResearchSummaryTotals{
			Findings: summary.Totals.Findings,
			Clarify:  summary.Totals.Clarify,
			Failed:   summary.Totals.Failed,
			Skipped:  summary.Totals.Skipped,
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

func emitProgressEvent(stdout io.Writer, ev research.Event) {
	re := render.ResearchEvent{
		Type:        ev.Type,
		ID:          ev.ID,
		Description: ev.Description,
		Started:     ev.Started,
		Status:      ev.Status,
		Summary:     ev.Summary,
		Error:       errString(ev.Err),
	}
	line, err := render.JSONResearchEvent(re)
	if err != nil {
		return
	}
	_, _ = stdout.Write(line)
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

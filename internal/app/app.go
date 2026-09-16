package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/tdawn0-0/git-analyzer/internal/config"
	"github.com/tdawn0-0/git-analyzer/internal/model"
	"github.com/tdawn0-0/git-analyzer/internal/workspace"
)

// Options holds CLI flags for analysis.
type Options struct {
	Since    string
	Until    string
	Author   string
	Repo     string
	Branch   string
	MaxDepth int
	Jobs     int
	Exclude  []string
}

// NewRootCommand builds the Cobra root command for git-workstats.
func NewRootCommand() *cobra.Command {
	opts := &Options{
		MaxDepth: config.DefaultMaxDepth,
		Jobs:     workspace.DefaultJobs(),
	}

	cmd := &cobra.Command{
		Use:   "git-workstats [path]",
		Short: "Local multi-repo Git engineering activity analysis",
		Long:  "Analyze Change Intensity across local Git repositories. Not an employee performance tool.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			return Run(cmd.Context(), root, *opts, cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVar(&opts.Since, "since", "", "Only commits after this date (git --since)")
	cmd.Flags().StringVar(&opts.Until, "until", "", "Only commits before this date (git --until)")
	cmd.Flags().StringVar(&opts.Author, "author", "", "Filter by author (name, email, or config identity)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Limit to repository name")
	cmd.Flags().StringVar(&opts.Branch, "branch", "", "Analyze this branch when present (else skip repo)")
	cmd.Flags().IntVar(&opts.MaxDepth, "max-depth", config.DefaultMaxDepth, "Max discovery depth")
	cmd.Flags().IntVar(&opts.Jobs, "jobs", workspace.DefaultJobs(), "Max concurrent repository analyzers")
	cmd.Flags().StringSliceVar(&opts.Exclude, "exclude", nil, "Extra directory names/globs to skip during discovery")

	return cmd
}

// Run discovers, analyzes, aggregates, and prints a text summary (Phase 2 smoke output).
func Run(ctx context.Context, root string, opts Options, out io.Writer) error {
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	cfg, err := workspace.LoadConfigNearRoot(abs)
	if err != nil {
		return err
	}

	analyzer := workspace.NewAnalyzer()
	stats, _, err := analyzer.Run(ctx, abs, workspace.AnalyzeOptions{
		Since:    opts.Since,
		Until:    opts.Until,
		Author:   opts.Author,
		Repo:     opts.Repo,
		Branch:   opts.Branch,
		MaxDepth: opts.MaxDepth,
		Jobs:     opts.Jobs,
		Exclude:  opts.Exclude,
		Config:   cfg,
	})
	if err != nil {
		return err
	}

	PrintSummary(out, stats)
	return nil
}

// PrintSummary writes a human-readable analysis summary.
func PrintSummary(out io.Writer, stats model.WorkspaceStats) {
	fmt.Fprintf(out, "git-workstats — workspace %s\n", stats.Root)
	fmt.Fprintf(out, "repositories: %d  developers: %d  changes: %d\n",
		len(stats.Repositories), len(stats.Developers), len(stats.Changes))

	complete, skipped, errored := 0, 0, 0
	for _, r := range stats.Repositories {
		switch r.Status {
		case model.StatusComplete:
			complete++
		case model.StatusSkipped:
			skipped++
			fmt.Fprintf(out, "  SKIP %s (%s)\n", r.Repository.Name, r.Error)
		case model.StatusError:
			errored++
			fmt.Fprintf(out, "  ERR  %s (%s)\n", r.Repository.Name, r.Error)
		}
	}
	fmt.Fprintf(out, "status: complete=%d skipped=%d error=%d\n", complete, skipped, errored)

	if len(stats.Developers) > 0 {
		fmt.Fprintln(out, "\nDevelopers (by Change Intensity):")
		limit := 10
		if len(stats.Developers) < limit {
			limit = len(stats.Developers)
		}
		for i := 0; i < limit; i++ {
			d := stats.Developers[i]
			fmt.Fprintf(out, "  %-20s  changes=%-4d  score=%.2f  avg=%.2f  +%d/-%d  repos=%d  cross-module=%d\n",
				trim(d.Developer, 20), d.ChangeCount, d.TotalScore, d.AverageScore,
				d.AddedLines, d.DeletedLines, len(d.RepositoryIDs), d.CrossModuleChanges)
		}
	}

	if len(stats.Timeline) > 0 {
		fmt.Fprintln(out, "\nDaily timeline:")
		start := stats.Timeline[0].Date.Format("2006-01-02")
		end := stats.Timeline[len(stats.Timeline)-1].Date.Format("2006-01-02")
		fmt.Fprintf(out, "  range %s → %s (%d days)\n", start, end, len(stats.Timeline))
		shown := 0
		for i := len(stats.Timeline) - 1; i >= 0 && shown < 5; i-- {
			b := stats.Timeline[i]
			if b.ChangeCount == 0 {
				continue
			}
			fmt.Fprintf(out, "  %s  score=%.2f  +%d/-%d  changes=%d\n",
				b.Date.Format("2006-01-02"), b.TotalScore, b.AddedLines, b.DeletedLines, b.ChangeCount)
			shown++
		}
	}
	fmt.Fprintf(out, "\nFinished at %s\n", time.Now().Format(time.RFC3339))
}

func trim(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

// Execute runs the root command with process context.
func Execute() {
	cmd := NewRootCommand()
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

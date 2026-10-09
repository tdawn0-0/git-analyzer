package tui

import (
	"context"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tdawn0-0/git-analyzer/internal/config"
	"github.com/tdawn0-0/git-analyzer/internal/git"
	"github.com/tdawn0-0/git-analyzer/internal/model"
	"github.com/tdawn0-0/git-analyzer/internal/workspace"
)

// WorkspaceDiscoveredMsg is emitted after repository discovery.
type WorkspaceDiscoveredMsg struct {
	Root  string
	Repos []model.Repository
	Cfg   config.Config
}

// RepositoryAnalyzedMsg reports a checkout starting or finishing analysis.
type RepositoryAnalyzedMsg struct {
	Result git.RepoResult
	Index  int
	Total  int
}

// AnalysisFinishedMsg carries the final aggregated stats.
type AnalysisFinishedMsg struct {
	Stats model.WorkspaceStats
}

// AnalysisErrorMsg is a fatal analysis error.
type AnalysisErrorMsg struct {
	Err error
}

func scanWorkspaceCmd(ctx context.Context, opts Options) tea.Cmd {
	return func() tea.Msg {
		abs, err := filepath.Abs(opts.Root)
		if err != nil {
			return AnalysisErrorMsg{Err: err}
		}
		cfg := opts.Config
		if cfg.Version == 0 && len(cfg.Types) == 0 {
			loaded, err := workspace.LoadConfigNearRoot(abs)
			if err != nil {
				return AnalysisErrorMsg{Err: err}
			}
			cfg = loaded
		}
		if opts.MaxDepth > 0 {
			cfg.Workspace.MaxDepth = opts.MaxDepth
		}
		if len(opts.Exclude) > 0 {
			cfg.Workspace.Exclude = append(append([]string(nil), cfg.Workspace.Exclude...), opts.Exclude...)
		}
		sc := workspace.NewScanner()
		ws, err := sc.Discover(ctx, abs, cfg)
		if err != nil {
			return AnalysisErrorMsg{Err: err}
		}
		repos := filterRepos(ws.Repositories, opts.Repo)
		return WorkspaceDiscoveredMsg{Root: abs, Repos: repos, Cfg: cfg}
	}
}

func filterRepos(repos []model.Repository, needle string) []model.Repository {
	if needle == "" {
		return repos
	}
	var out []model.Repository
	n := strings.ToLower(needle)
	for _, r := range repos {
		if strings.EqualFold(r.Name, needle) || strings.Contains(strings.ToLower(r.Name), n) {
			out = append(out, r)
		}
	}
	return out
}

func startAnalysisCmd(ctx context.Context, root string, repos []model.Repository, cfg config.Config, opts Options) tea.Cmd {
	return func() tea.Msg { return analyzeRepositories(ctx, root, repos, cfg, opts, nil) }
}

func analyzeRepositories(ctx context.Context, root string, repos []model.Repository, cfg config.Config, opts Options, onProgress func(workspace.RepositoryProgress)) tea.Msg {
	stats, _, err := workspace.AnalyzeRepositories(ctx, root, repos, workspace.AnalyzeOptions{
		Since: opts.Since, Until: opts.Until, Author: opts.Author, Branch: opts.Branch, Jobs: opts.Jobs, Config: cfg, OnProgress: onProgress,
	})
	if err != nil {
		return AnalysisErrorMsg{Err: err}
	}
	return AnalysisFinishedMsg{Stats: stats}
}

// Stream worker events through one ordered queue, including the final result.
// Bubble Tea commands wait for one event at a time without blocking Update.
func streamAnalysisCmd(ctx context.Context, root string, repos []model.Repository, cfg config.Config, opts Options, events chan<- tea.Msg) tea.Cmd {
	return func() tea.Msg {
		defer close(events)
		send := func(msg tea.Msg) {
			select {
			case events <- msg:
			case <-ctx.Done():
			}
		}
		final := analyzeRepositories(ctx, root, repos, cfg, opts, func(progress workspace.RepositoryProgress) {
			send(RepositoryAnalyzedMsg{Index: progress.Index, Total: len(repos), Result: progress.Result})
		})
		send(final)
		return nil
	}
}

func waitAnalysisEventCmd(ctx context.Context, events <-chan tea.Msg) tea.Cmd {
	if events == nil {
		return nil
	}
	return func() tea.Msg {
		select {
		case msg, ok := <-events:
			if !ok {
				return nil
			}
			return msg
		case <-ctx.Done():
			return AnalysisErrorMsg{Err: ctx.Err()}
		}
	}
}

func statusGlyph(st model.RepositoryStatus) string {
	switch st {
	case model.StatusComplete:
		return "✓"
	case model.StatusAnalyzing:
		return "…"
	case model.StatusError:
		return "ERR"
	case model.StatusSkipped:
		return "SKIP"
	default:
		return "·"
	}
}

func progressBar(done, total, width int) string {
	if total <= 0 {
		total = 1
	}
	if width < 4 {
		width = 4
	}
	inner := width - 2
	filled := done * inner / total
	if filled > inner {
		filled = inner
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", inner-filled)
	return "[" + bar + "]"
}

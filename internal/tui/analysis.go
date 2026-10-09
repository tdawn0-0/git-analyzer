package tui

import (
	"context"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tdawn0-0/git-analyzer/internal/aggregate"
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

// RepositoryAnalyzedMsg is emitted when one repository finishes (optional progressive path).
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
	return func() tea.Msg {
		jobs := opts.Jobs
		if jobs <= 0 {
			jobs = workspace.DefaultJobs()
		}
		gitOpts := git.AnalyzeOptions{
			Since:  opts.Since,
			Until:  opts.Until,
			Branch: opts.Branch,
		}
		if opts.Author != "" {
			gitOpts.Author = opts.Author
		}

		results := workspace.Map(ctx, jobs, repos, func(ctx context.Context, repo model.Repository) git.RepoResult {
			res := git.AnalyzeRepository(ctx, repo, cfg, gitOpts)
			if opts.Author != "" {
				res.Changes = git.FilterChangesByAuthor(res.Changes, opts.Author)
			}
			return res
		})

		if err := ctx.Err(); err != nil {
			return AnalysisErrorMsg{Err: err}
		}
		return AnalysisFinishedMsg{Stats: aggregate.Workspace(root, results)}
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

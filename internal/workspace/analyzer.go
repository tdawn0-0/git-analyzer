package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdawn0-0/git-analyzer/internal/aggregate"
	"github.com/tdawn0-0/git-analyzer/internal/config"
	"github.com/tdawn0-0/git-analyzer/internal/git"
	"github.com/tdawn0-0/git-analyzer/internal/model"
)

// AnalyzeOptions configures workspace discovery + multi-repo analysis.
type AnalyzeOptions struct {
	Since    string
	Until    string
	Author   string
	Repo     string // filter by repository name (substring / exact)
	Branch   string
	MaxDepth int
	Jobs     int
	Exclude  []string
	Config   config.Config
}

// Analyzer discovers repositories and analyzes them concurrently.
type Analyzer struct {
	Scanner *Scanner
}

// NewAnalyzer returns an Analyzer with the default scanner.
func NewAnalyzer() *Analyzer {
	return &Analyzer{Scanner: NewScanner()}
}

// Run discovers repos under root, analyzes each in a worker pool, and aggregates.
func (a *Analyzer) Run(ctx context.Context, root string, opts AnalyzeOptions) (model.WorkspaceStats, []git.RepoResult, error) {
	if a.Scanner == nil {
		a.Scanner = NewScanner()
	}
	cfg := opts.Config
	if cfg.Version == 0 && len(cfg.Types) == 0 {
		cfg = config.Defaults()
	}
	if opts.MaxDepth > 0 {
		cfg.Workspace.MaxDepth = opts.MaxDepth
	}
	if len(opts.Exclude) > 0 {
		cfg.Workspace.Exclude = append(append([]string(nil), cfg.Workspace.Exclude...), opts.Exclude...)
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return model.WorkspaceStats{}, nil, err
	}

	ws, err := a.Scanner.Discover(ctx, absRoot, cfg)
	if err != nil {
		return model.WorkspaceStats{}, nil, err
	}

	repos := ws.Repositories
	if opts.Repo != "" {
		repos = filterReposByName(repos, opts.Repo)
	}

	return AnalyzeRepositories(ctx, absRoot, repos, opts)
}

// AnalyzeRepositories is the shared analysis pipeline for text and TUI sessions.
// Filter identities after mailmap/config resolution so canonical names and aliases
// cannot be removed prematurely by git log's raw-author filter.
func AnalyzeRepositories(ctx context.Context, root string, repos []model.Repository, opts AnalyzeOptions) (model.WorkspaceStats, []git.RepoResult, error) {
	cfg := opts.Config
	if cfg.Version == 0 && len(cfg.Types) == 0 {
		cfg = config.Defaults()
	}
	gitOpts := git.AnalyzeOptions{Since: opts.Since, Until: opts.Until, Branch: opts.Branch}
	results := Map(ctx, opts.Jobs, repos, func(ctx context.Context, repo model.Repository) git.RepoResult {
		res := git.AnalyzeRepository(ctx, repo, cfg, gitOpts)
		res.Changes = git.FilterChangesByAuthor(res.Changes, opts.Author)
		return res
	})
	if err := ctx.Err(); err != nil {
		return model.WorkspaceStats{}, results, err
	}
	return aggregate.Workspace(root, results), results, nil
}

func filterReposByName(repos []model.Repository, needle string) []model.Repository {
	n := strings.ToLower(needle)
	var out []model.Repository
	for _, r := range repos {
		if strings.EqualFold(r.Name, needle) || strings.Contains(strings.ToLower(r.Name), n) {
			out = append(out, r)
		}
	}
	return out
}

// LoadConfigNearRoot loads .workstats.yml from root if present, else defaults.
func LoadConfigNearRoot(root string) (config.Config, error) {
	candidates := []string{
		filepath.Join(root, ".workstats.yml"),
		filepath.Join(root, ".workstats.yaml"),
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return config.Load(p)
		}
	}
	return config.Defaults(), nil
}

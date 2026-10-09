package workspace

import (
	"context"

	"github.com/tdawn0-0/git-analyzer/internal/config"
	"github.com/tdawn0-0/git-analyzer/internal/git"
	"github.com/tdawn0-0/git-analyzer/internal/model"
)

// Scanner discovers repositories for a workspace root using git.RepositoryScanner.
type Scanner struct {
	Git git.RepositoryScanner
}

// NewScanner returns a workspace scanner with the default git FS scanner.
func NewScanner() *Scanner {
	return &Scanner{Git: git.NewRepositoryScanner()}
}

// Discover walks root and returns a Workspace populated with repositories.
func (s *Scanner) Discover(ctx context.Context, root string, cfg config.Config) (model.Workspace, error) {
	if s.Git == nil {
		s.Git = git.NewRepositoryScanner()
	}
	opts := git.ScanOptions{
		MaxDepth: cfg.Workspace.MaxDepth,
		Exclude:  cfg.Workspace.Exclude,
	}
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = config.DefaultMaxDepth
	}
	if opts.Exclude == nil {
		opts.Exclude = append([]string(nil), config.DefaultExcludeDirs...)
	}
	repos, err := s.Git.Scan(ctx, root, opts)
	if err != nil {
		return model.Workspace{}, err
	}
	return model.Workspace{Root: root, Repositories: repos}, nil
}

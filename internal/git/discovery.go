package git

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdawn0-0/git-analyzer/internal/config"
	"github.com/tdawn0-0/git-analyzer/internal/model"
)

// ScanOptions controls recursive Git repository discovery.
type ScanOptions struct {
	MaxDepth      int      // default 6; depth 0 is the root itself
	Exclude       []string // directory basenames and/or path globs
	FollowSymlink bool     // default false — do not follow directory symlinks
}

// RepositoryScanner discovers Git repositories under a filesystem root.
type RepositoryScanner interface {
	Scan(ctx context.Context, root string, opts ScanOptions) ([]model.Repository, error)
}

// FSScanner walks the filesystem and confirms candidates with git rev-parse.
type FSScanner struct{}

// NewRepositoryScanner returns the default filesystem-backed scanner.
func NewRepositoryScanner() RepositoryScanner {
	return FSScanner{}
}

// Scan discovers repositories under root.
// If root itself is a Git repository, it is included (along with any nested repos
// found within MaxDepth, unless root's .git causes early handling — nested are still found).
func (FSScanner) Scan(ctx context.Context, root string, opts ScanOptions) ([]model.Repository, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(absRoot)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &fs.PathError{Op: "scan", Path: absRoot, Err: fs.ErrInvalid}
	}

	opts = normalizeOptions(opts)
	exclude := buildExcludeMatcher(opts.Exclude)

	seen := map[string]struct{}{}
	var repos []model.Repository

	add := func(dir string) error {
		repo, err := NewRepository(ctx, dir)
		if err != nil {
			return nil // not a usable repo; skip
		}
		if _, ok := seen[repo.Path]; ok {
			return nil
		}
		seen[repo.Path] = struct{}{}
		repos = append(repos, repo)
		return nil
	}

	// Depth: root = 0. MaxDepth limits how deep we descend relative to root.
	var walk func(dir string, depth int) error
	walk = func(dir string, depth int) error {
		if err := ctx.Err(); err != nil {
			return err
		}

		if isGitRepoPath(dir) {
			_ = add(dir)
			// Continue walking to find nested repositories (do not enter .git).
		}

		if depth >= opts.MaxDepth {
			return nil
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil // unreadable directory — skip
		}
		for _, ent := range entries {
			name := ent.Name()
			if name == ".git" {
				continue
			}
			full := filepath.Join(dir, name)

			// Use Lstat semantics via DirEntry — do not follow symlinks by default.
			isSymlink := ent.Type()&fs.ModeSymlink != 0
			isDir := ent.IsDir()

			if isSymlink {
				if !opts.FollowSymlink {
					continue
				}
				target, err := os.Stat(full)
				if err != nil || !target.IsDir() {
					continue
				}
				isDir = true
			}

			if !isDir {
				continue
			}

			rel, err := filepath.Rel(absRoot, full)
			if err != nil {
				rel = full
			}
			if exclude.match(name, filepath.ToSlash(rel)) {
				continue
			}

			if err := walk(full, depth+1); err != nil {
				return err
			}
		}
		return nil
	}

	if err := walk(absRoot, 0); err != nil {
		return nil, err
	}
	return repos, nil
}

func normalizeOptions(opts ScanOptions) ScanOptions {
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = config.DefaultMaxDepth
	}
	if opts.Exclude == nil {
		opts.Exclude = append([]string(nil), config.DefaultExcludeDirs...)
	}
	return opts
}

func isGitRepoPath(dir string) bool {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Lstat(gitPath)
	if err != nil {
		return false
	}
	// Directory (normal) or file (worktree / gitdir pointer).
	return info.IsDir() || info.Mode().IsRegular()
}

type excludeMatcher struct {
	basenames map[string]struct{}
	globs     []string
}

func buildExcludeMatcher(patterns []string) excludeMatcher {
	m := excludeMatcher{basenames: map[string]struct{}{}}
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		slash := filepath.ToSlash(p)
		// Bare directory name (no slash, no glob metachar) → basename skip.
		if !strings.ContainsAny(slash, "*/?[") {
			m.basenames[slash] = struct{}{}
			continue
		}
		// "**/name/**" or "**/name" → also treat basename specially for SkipDir.
		if strings.HasPrefix(slash, "**/") {
			rest := strings.TrimPrefix(slash, "**/")
			rest = strings.TrimSuffix(rest, "/**")
			rest = strings.TrimSuffix(rest, "/**/")
			if rest != "" && !strings.ContainsAny(rest, "*/?[") {
				m.basenames[rest] = struct{}{}
			}
		}
		m.globs = append(m.globs, slash)
	}
	return m
}

func (m excludeMatcher) match(basename, relSlash string) bool {
	if _, ok := m.basenames[basename]; ok {
		return true
	}
	for _, g := range m.globs {
		if ok, _ := filepath.Match(g, relSlash); ok {
			return true
		}
		// Also try matching basename against trailing segment of glob.
		if ok, _ := filepath.Match(g, basename); ok {
			return true
		}
	}
	return false
}

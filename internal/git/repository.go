package git

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"

	"github.com/tdawn0-0/git-analyzer/internal/model"
)

// RepositoryID returns remote.origin.url when set, otherwise a hash of the absolute path.
func RepositoryID(ctx context.Context, absPath string) (id, remote string) {
	remote = RemoteOriginURL(ctx, absPath)
	if remote != "" {
		return remote, remote
	}
	sum := sha256.Sum256([]byte(absPath))
	return "path:" + hex.EncodeToString(sum[:16]), ""
}

// NewRepository builds a model.Repository for an absolute repository root.
func NewRepository(ctx context.Context, absPath string) (model.Repository, error) {
	absPath = filepath.Clean(absPath)
	if !filepath.IsAbs(absPath) {
		var err error
		absPath, err = filepath.Abs(absPath)
		if err != nil {
			return model.Repository{}, err
		}
	}
	top, err := ShowTopLevel(ctx, absPath)
	if err != nil {
		return model.Repository{}, err
	}
	top = filepath.Clean(top)
	id, remote := RepositoryID(ctx, top)
	return model.Repository{
		ID:     id,
		Name:   filepath.Base(top),
		Path:   top,
		Remote: remote,
	}, nil
}

// DisambiguateNames returns display names that append a relative path suffix on collisions.
func DisambiguateNames(repos []model.Repository, workspaceRoot string) map[string]string {
	byName := map[string][]model.Repository{}
	for _, r := range repos {
		byName[r.Name] = append(byName[r.Name], r)
	}
	out := make(map[string]string, len(repos))
	root := filepath.Clean(workspaceRoot)
	for name, group := range byName {
		if len(group) == 1 {
			out[group[0].Path] = name
			continue
		}
		for _, r := range group {
			rel, err := filepath.Rel(root, r.Path)
			if err != nil {
				rel = r.Path
			}
			rel = filepath.ToSlash(rel)
			out[r.Path] = name + " (" + strings.TrimPrefix(rel, "./") + ")"
		}
	}
	return out
}

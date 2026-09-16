package aggregate

import (
	"github.com/tdawn0-0/git-analyzer/internal/git"
	"github.com/tdawn0-0/git-analyzer/internal/model"
)

// Workspace builds WorkspaceStats from root, repo results, and all changes.
func Workspace(root string, results []git.RepoResult) model.WorkspaceStats {
	var changes []model.ChangeUnit
	for _, r := range results {
		changes = append(changes, r.Changes...)
	}
	return model.WorkspaceStats{
		Root:         root,
		Repositories: Repositories(results),
		Developers:   Developers(changes),
		Changes:      changes,
		Timeline:     Timeline(changes),
	}
}

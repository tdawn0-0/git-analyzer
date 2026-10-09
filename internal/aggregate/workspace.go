package aggregate

import (
	"github.com/tdawn0-0/git-analyzer/internal/git"
	"github.com/tdawn0-0/git-analyzer/internal/model"
)

// Workspace builds WorkspaceStats from root, repo results, and all changes.
func Workspace(root string, results []git.RepoResult) model.WorkspaceStats {
	results = mergeRepositories(results)
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

// Merge local checkouts of the same logical repository. Shared commits count once;
// branch-specific commits remain visible. Independent repositories stay separate.
func mergeRepositories(results []git.RepoResult) []git.RepoResult {
	out := make([]git.RepoResult, 0, len(results))
	indices := map[string]int{}
	seen := map[string]map[string]bool{}
	for _, result := range results {
		id := result.Repository.ID
		if id == "" {
			id = "path:" + result.Repository.Path
		}
		i, ok := indices[id]
		if !ok {
			i = len(out)
			indices[id] = i
			entry := result
			entry.Changes = nil
			out = append(out, entry)
			seen[id] = map[string]bool{}
		} else if result.Status == model.StatusComplete && out[i].Status != model.StatusComplete {
			out[i].Status = result.Status
			out[i].Err = nil
			out[i].Message = ""
		}
		for _, change := range result.Changes {
			if change.Hash != "" && seen[id][change.Hash] {
				continue
			}
			seen[id][change.Hash] = true
			out[i].Changes = append(out[i].Changes, change)
		}
	}
	return out
}

package aggregate

import (
	"sort"

	"github.com/tdawn0-0/git-analyzer/internal/git"
	"github.com/tdawn0-0/git-analyzer/internal/model"
)

// Repositories builds RepositoryStats from per-repo analysis results.
func Repositories(results []git.RepoResult) []model.RepositoryStats {
	out := make([]model.RepositoryStats, 0, len(results))
	for _, r := range results {
		st := model.RepositoryStats{
			Repository:       r.Repository,
			Status:           r.Status,
			TypeDistribution: map[model.ChangeType]int{},
			Error:            r.Message,
		}
		devSet := map[string]struct{}{}
		for _, c := range r.Changes {
			st.ChangeCount++
			st.TotalScore += c.Score.Final
			st.AddedLines += c.AddedLines
			st.DeletedLines += c.DeletedLines
			st.TypeDistribution[c.Type]++
			name := c.Author.Name
			if name == "" {
				name = c.Author.Email
			}
			if name != "" {
				devSet[name] = struct{}{}
			}
		}
		devs := make([]string, 0, len(devSet))
		for d := range devSet {
			devs = append(devs, d)
		}
		sort.Strings(devs)
		st.Developers = devs
		out = append(out, st)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Repository.Name < out[j].Repository.Name
	})
	return out
}

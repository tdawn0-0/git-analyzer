package aggregate

import (
	"sort"

	"github.com/tdawn0-0/git-analyzer/internal/model"
)

// Developers builds DeveloperStats from ChangeUnits, merging by Author.Name.
func Developers(changes []model.ChangeUnit) []model.DeveloperStats {
	by := map[string]*model.DeveloperStats{}
	repoSeen := map[string]map[string]struct{}{}

	for _, c := range changes {
		key := c.Author.Name
		if key == "" {
			key = c.Author.Email
		}
		if key == "" {
			key = "unknown"
		}
		d, ok := by[key]
		if !ok {
			d = &model.DeveloperStats{
				Developer:        key,
				ModulesTouched:   map[string]struct{}{},
				TypeDistribution: map[model.ChangeType]int{},
			}
			by[key] = d
			repoSeen[key] = map[string]struct{}{}
		}
		d.ChangeCount++
		d.TotalScore += c.Score.Final
		d.AddedLines += c.AddedLines
		d.DeletedLines += c.DeletedLines
		if c.RepositoryID != "" {
			repoSeen[key][c.RepositoryID] = struct{}{}
		}
		for _, m := range c.Modules {
			d.ModulesTouched[m] = struct{}{}
		}
		if len(c.Modules) > 1 {
			d.CrossModuleChanges++
		}
		d.TypeDistribution[c.Type]++
		switch c.Type {
		case model.ChangeTypeFeat:
			d.FeatureCount++
		case model.ChangeTypeFix:
			d.FixCount++
		case model.ChangeTypeRefactor:
			d.RefactorCount++
		case model.ChangeTypePerf:
			d.PerfCount++
		case model.ChangeTypeTest:
			d.TestCount++
		case model.ChangeTypeDocs:
			d.DocsCount++
		case model.ChangeTypeChore:
			d.ChoreCount++
		}
	}

	out := make([]model.DeveloperStats, 0, len(by))
	for key, d := range by {
		d.NetLines = d.AddedLines - d.DeletedLines
		if d.ChangeCount > 0 {
			d.AverageScore = d.TotalScore / float64(d.ChangeCount)
		}
		ids := make([]string, 0, len(repoSeen[key]))
		for id := range repoSeen[key] {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		d.RepositoryIDs = ids
		out = append(out, *d)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TotalScore == out[j].TotalScore {
			return out[i].Developer < out[j].Developer
		}
		return out[i].TotalScore > out[j].TotalScore
	})
	return out
}

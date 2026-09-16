package tui

import (
	"sort"

	"github.com/tdawn0-0/git-analyzer/internal/model"
)

func sortDevelopers(devs []model.DeveloperStats, mode SortMode) {
	sort.SliceStable(devs, func(i, j int) bool {
		a, b := devs[i], devs[j]
		switch mode {
		case SortName:
			return a.Developer < b.Developer
		case SortChanges:
			if a.ChangeCount == b.ChangeCount {
				return a.Developer < b.Developer
			}
			return a.ChangeCount > b.ChangeCount
		case SortAdded:
			if a.AddedLines == b.AddedLines {
				return a.Developer < b.Developer
			}
			return a.AddedLines > b.AddedLines
		case SortDeleted:
			if a.DeletedLines == b.DeletedLines {
				return a.Developer < b.Developer
			}
			return a.DeletedLines > b.DeletedLines
		default: // SortActivity
			if a.TotalScore == b.TotalScore {
				return a.Developer < b.Developer
			}
			return a.TotalScore > b.TotalScore
		}
	})
}

func sortRepositories(repos []model.RepositoryStats, mode SortMode) {
	sort.SliceStable(repos, func(i, j int) bool {
		a, b := repos[i], repos[j]
		switch mode {
		case SortName:
			return a.Repository.Name < b.Repository.Name
		case SortChanges:
			if a.ChangeCount == b.ChangeCount {
				return a.Repository.Name < b.Repository.Name
			}
			return a.ChangeCount > b.ChangeCount
		case SortAdded:
			if a.AddedLines == b.AddedLines {
				return a.Repository.Name < b.Repository.Name
			}
			return a.AddedLines > b.AddedLines
		case SortDeleted:
			if a.DeletedLines == b.DeletedLines {
				return a.Repository.Name < b.Repository.Name
			}
			return a.DeletedLines > b.DeletedLines
		default:
			if a.TotalScore == b.TotalScore {
				return a.Repository.Name < b.Repository.Name
			}
			return a.TotalScore > b.TotalScore
		}
	})
}

func cycleSort(m SortMode) SortMode {
	return (m + 1) % 5
}

func sortLabel(m SortMode) string {
	switch m {
	case SortName:
		return "Name"
	case SortChanges:
		return "Changes"
	case SortAdded:
		return "Added"
	case SortDeleted:
		return "Deleted"
	default:
		return "Activity"
	}
}

package aggregate_test

import (
	"testing"
	"time"

	"github.com/tdawn0-0/git-analyzer/internal/aggregate"
	"github.com/tdawn0-0/git-analyzer/internal/git"
	"github.com/tdawn0-0/git-analyzer/internal/model"
)

func cu(dev, email, repoID string, day time.Time, typ model.ChangeType, score float64, add, del int, modules []string) model.ChangeUnit {
	return model.ChangeUnit{
		RepositoryID: repoID,
		Author:       model.Author{Name: dev, Email: email},
		Timestamp:    day,
		Type:         typ,
		AddedLines:   add,
		DeletedLines: del,
		Modules:      modules,
		Score:        model.ScoreBreakdown{Final: score},
		Message:      string(typ) + ": x",
		Hash:         "h",
	}
}

func TestDevelopersCrossRepoIdentityAndTypes(t *testing.T) {
	day := time.Date(2026, 9, 10, 15, 0, 0, 0, time.UTC)
	changes := []model.ChangeUnit{
		cu("Yeu", "dev@example.com", "repo-a", day, model.ChangeTypeFeat, 10, 5, 1, []string{"api"}),
		cu("Yeu", "dev2@example.com", "repo-b", day, model.ChangeTypeFix, 4, 2, 0, []string{"api", "db"}),
		cu("Alice", "alice@ex.com", "repo-a", day, model.ChangeTypeDocs, 1, 3, 0, []string{"docs"}),
	}
	// Identity merge is expected to happen before aggregation (Author.Name already canonical).
	devs := aggregate.Developers(changes)
	if len(devs) != 2 {
		t.Fatalf("devs=%d %+v", len(devs), devs)
	}
	var yeu *model.DeveloperStats
	for i := range devs {
		if devs[i].Developer == "Yeu" {
			yeu = &devs[i]
		}
	}
	if yeu == nil {
		t.Fatal("Yeu missing")
	}
	if yeu.ChangeCount != 2 || yeu.TotalScore != 14 || yeu.AverageScore != 7 {
		t.Fatalf("%+v", yeu)
	}
	if len(yeu.RepositoryIDs) != 2 {
		t.Fatalf("repos=%v", yeu.RepositoryIDs)
	}
	if yeu.CrossModuleChanges != 1 {
		t.Fatalf("cross=%d", yeu.CrossModuleChanges)
	}
	if yeu.FeatureCount != 1 || yeu.FixCount != 1 {
		t.Fatalf("types feat=%d fix=%d dist=%v", yeu.FeatureCount, yeu.FixCount, yeu.TypeDistribution)
	}
	if yeu.NetLines != (5+2)-(1+0) {
		t.Fatalf("net=%d", yeu.NetLines)
	}
}

func TestTimelineDailyGrouping(t *testing.T) {
	d1 := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
	changes := []model.ChangeUnit{
		cu("A", "a@x", "r", d1, model.ChangeTypeFeat, 2.5, 10, 1, nil),
		cu("A", "a@x", "r", d1, model.ChangeTypeFix, 1.5, 4, 2, nil),
		cu("B", "b@x", "r", d2, model.ChangeTypeFeat, 3, 7, 0, nil),
	}
	tl := aggregate.Timeline(changes)
	if len(tl) != 3 { // Sep 1,2,3 with zero day in middle
		t.Fatalf("len=%d %+v", len(tl), tl)
	}
	if tl[0].Date.Format("2006-01-02") != "2026-09-01" || tl[0].TotalScore != 4 || tl[0].AddedLines != 14 {
		t.Fatalf("day0=%+v", tl[0])
	}
	if tl[1].ChangeCount != 0 || tl[1].TotalScore != 0 {
		t.Fatalf("gap day should be zero: %+v", tl[1])
	}
	if tl[2].TotalScore != 3 || tl[2].DeletedLines != 0 {
		t.Fatalf("day2=%+v", tl[2])
	}
}

func TestWorkspaceAggregation(t *testing.T) {
	day := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	results := []git.RepoResult{
		{
			Repository: model.Repository{ID: "r1", Name: "one", Path: "/one"},
			Status:     model.StatusComplete,
			Changes: []model.ChangeUnit{
				cu("Yeu", "a@x", "r1", day, model.ChangeTypeFeat, 5, 1, 0, []string{"m"}),
			},
		},
		{
			Repository: model.Repository{ID: "r2", Name: "two", Path: "/two"},
			Status:     model.StatusSkipped,
			Message:    "branch unavailable",
		},
	}
	ws := aggregate.Workspace("/ws", results)
	if len(ws.Repositories) != 2 || len(ws.Developers) != 1 || len(ws.Changes) != 1 {
		t.Fatalf("%+v", ws)
	}
	var skipped bool
	for _, r := range ws.Repositories {
		if r.Status == model.StatusSkipped && r.Error == "branch unavailable" {
			skipped = true
		}
	}
	if !skipped {
		t.Fatal("expected skipped repo preserved")
	}
}

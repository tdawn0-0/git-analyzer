package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tdawn0-0/git-analyzer/internal/model"
)

func sampleStats() model.WorkspaceStats {
	day := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	return model.WorkspaceStats{
		Root: "/ws",
		Repositories: []model.RepositoryStats{{
			Repository: model.Repository{ID: "r1", Name: "svc", Path: "/ws/svc"},
			Status:     model.StatusComplete,
			ChangeCount: 1,
			TotalScore: 5,
			TypeDistribution: map[model.ChangeType]int{model.ChangeTypeFeat: 1},
		}},
		Developers: []model.DeveloperStats{{
			Developer:        "Yeu",
			RepositoryIDs:    []string{"r1"},
			ChangeCount:      1,
			TotalScore:       5,
			AverageScore:     5,
			AddedLines:       10,
			TypeDistribution: map[model.ChangeType]int{model.ChangeTypeFeat: 1, model.ChangeTypeFix: 2},
			FeatureCount:     1,
			FixCount:         2,
		}},
		Changes: []model.ChangeUnit{{
			RepositoryID: "r1",
			Hash:         "abcdef123456",
			Author:       model.Author{Name: "Yeu", Email: "y@x.com"},
			Timestamp:    day,
			Message:      "feat: thing",
			AddedLines:   10,
			DeletedLines: 2,
			ChangedFiles: 3,
			Modules:      []string{"api"},
			Languages:    []string{"Go"},
			Layers:       []string{"backend"},
			Type:         model.ChangeTypeFeat,
			Score: model.ScoreBreakdown{
				SizeScore: 3.7, ComplexityFactor: 1.1, ModuleFactor: 1.1,
				TypeFactor: 1.15, QualityFactor: 1.0, Final: 5.0,
			},
		}},
		Timeline: []model.DailyBucket{{
			Date: day, TotalScore: 5, AddedLines: 10, DeletedLines: 2, ChangeCount: 1,
		}},
	}
}

func TestModelNavigationAndExplain(t *testing.T) {
	m := New(context.Background(), Options{Root: "/ws"})
	m.Width, m.Height = 140, 40
	m.Loading = false
	stats := sampleStats()
	m.Workspace = &stats

	// Enter developer view
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mod := m2.(Model)
	if mod.Active != ViewDeveloper {
		t.Fatalf("view=%v", mod.Active)
	}

	view := mod.View()
	if !strings.Contains(view, "Developer:") {
		t.Fatalf("missing developer header: %s", view)
	}
	if strings.Contains(strings.ToLower(view), "winner") || strings.Contains(strings.ToLower(view), "most productive") {
		t.Fatal("forbidden performance language")
	}

	// Explain
	mod.SelectedChange = 0
	m3, _ := mod.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	mod = m3.(Model)
	if mod.Active != ViewExplain {
		t.Fatalf("expected explain, got %v", mod.Active)
	}
	exp := mod.View()
	for _, needle := range []string{
		"Change Intensity Explanation",
		"SizeScore",
		"Complexity",
		"ModuleFactor",
		"TypeFactor",
		"QualityFactor",
		"FINAL",
	} {
		if !strings.Contains(exp, needle) {
			t.Fatalf("explain missing %q in:\n%s", needle, exp)
		}
	}

	// Trend toggle
	m4, _ := mod.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	mod = m4.(Model)
	_ = mod

	// Help
	m5, _ := mod.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	mod = m5.(Model)
	if !strings.Contains(mod.View(), "Keys") {
		t.Fatal("help missing")
	}
}

func TestLayoutBreakpoints(t *testing.T) {
	stats := sampleStats()
	for _, w := range []int{60, 100, 140} {
		m := New(context.Background(), Options{Root: "/ws"})
		m.Loading = false
		m.Workspace = &stats
		m.Width, m.Height = w, 40
		out := m.View()
		if out == "" {
			t.Fatalf("empty view at width %d", w)
		}
		if !strings.Contains(out, "git-workstats") {
			t.Fatalf("header missing at width %d", w)
		}
	}
}

func TestLoadingView(t *testing.T) {
	m := New(context.Background(), Options{Root: "/ws"})
	m.Width, m.Height = 80, 24
	m.phase = "discovering"
	out := m.View()
	if !strings.Contains(out, "Discovering repositories") {
		t.Fatalf("%s", out)
	}
	m.phase = "analyzing"
	m.repos = []model.Repository{{Name: "a"}, {Name: "b"}}
	out = m.View()
	if !strings.Contains(out, "Analyzing repositories") {
		t.Fatalf("%s", out)
	}
}

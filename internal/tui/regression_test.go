package tui

import (
	"context"
	"fmt"
	"github.com/tdawn0-0/git-analyzer/internal/config"
	"github.com/tdawn0-0/git-analyzer/internal/git"
	"github.com/tdawn0-0/git-analyzer/internal/workspace"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tdawn0-0/git-analyzer/internal/model"
)

func regressionModel() Model {
	m := New(context.Background(), Options{Root: "/ws"})
	m.Loading = false
	m.Width, m.Height = 100, 24
	ws := sampleStats()
	ws.Repositories = append(ws.Repositories, model.RepositoryStats{Repository: model.Repository{ID: "r2", Name: "other"}})
	ws.Changes = append([]model.ChangeUnit{{RepositoryID: "r2", Hash: "other", Author: model.Author{Name: "Other"}, Message: "other commit"}}, ws.Changes...)
	m.Workspace = &ws
	return m
}

func press(m Model, key string) Model {
	typ := tea.KeyRunes
	switch key {
	case "enter":
		typ = tea.KeyEnter
	case "pgdown":
		typ = tea.KeyPgDown
	case "pgup":
		typ = tea.KeyPgUp
	}
	next, _ := m.Update(tea.KeyMsg{Type: typ, Runes: []rune(key)})
	return next.(Model)
}

func TestDetailNavigationPreservesSelection(t *testing.T) {
	for _, detail := range []string{"d", "r"} {
		t.Run(detail, func(t *testing.T) {
			m := press(regressionModel(), detail)
			m = press(m, "enter")
			if cs := m.filteredChanges(); len(cs) != 1 || cs[0].Hash != "abcdef123456" {
				t.Fatalf("detail scope lost: %+v", cs)
			}
			m = press(m, "e")
			c, ok := m.selectedChange()
			if !ok || c.Hash != "abcdef123456" {
				t.Fatalf("explaining wrong change: %+v", c)
			}
			m = press(m, "h")
			m = press(m, "h")
			want := ViewDeveloper
			if detail == "r" {
				want = ViewRepository
			}
			if m.Active != want {
				t.Fatalf("back went to %v, want %v", m.Active, want)
			}
		})
	}
}

func TestExplainCanScrollToFinal(t *testing.T) {
	m := press(press(regressionModel(), "d"), "e")
	for i := 0; i < 4; i++ {
		m = press(m, "pgdown")
	}
	if !strings.Contains(m.View(), "FINAL") {
		t.Fatal("final score inaccessible at 24 rows")
	}
	m = press(m, "pgup")
	m = press(m, "h")
	if m.Active != ViewDeveloper {
		t.Fatalf("explain back lost origin: %v", m.Active)
	}
}

func TestChangeListScrollsWithSelection(t *testing.T) {
	m := regressionModel()
	m.Workspace.Changes = nil
	for i := 0; i < 50; i++ {
		m.Workspace.Changes = append(m.Workspace.Changes, model.ChangeUnit{RepositoryID: "r1", Hash: fmt.Sprintf("%07d", i), Author: model.Author{Name: "Yeu"}, Message: fmt.Sprintf("commit-%02d", i)})
	}
	m = press(m, "r")
	m = press(m, "enter")
	for i := 0; i < 49; i++ {
		m = press(m, "j")
	}
	if !strings.Contains(m.View(), "commit-49") {
		t.Fatal("selected commit is below visible list")
	}
}

func TestTUIAuthorFilterMatchesText(t *testing.T) {
	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Raw Author", "GIT_AUTHOR_EMAIL=raw@example.test", "GIT_COMMITTER_NAME=Raw Author", "GIT_COMMITTER_EMAIL=raw@example.test")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git: %v %s", err, out)
		}
	}
	run("init", "-q")
	if err := os.WriteFile(filepath.Join(root, "hello.go"), []byte("package main\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-qm", "feat: hello")
	cfg := config.Defaults()
	cfg.Authors = map[string][]string{"Canonical": {"raw@example.test"}}
	repo, err := git.NewRepository(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, author := range []string{"Canonical", "canonical", "Canon", "raw@example.test"} {
		text, _, err := workspace.NewAnalyzer().Run(context.Background(), root, workspace.AnalyzeOptions{Config: cfg, Author: author})
		if err != nil {
			t.Fatal(err)
		}
		msg := startAnalysisCmd(context.Background(), root, []model.Repository{repo}, cfg, Options{Author: author})()
		finished, ok := msg.(AnalysisFinishedMsg)
		if !ok || len(finished.Stats.Changes) != 1 || len(text.Changes) != 1 {
			t.Fatalf("author %q: TUI=%+v text=%+v", author, msg, text)
		}
	}
}

func TestExplainScrollClampsAtBottom(t *testing.T) {
	m := press(press(regressionModel(), "d"), "e")
	for i := 0; i < 10; i++ {
		m = press(m, "pgdown")
	}
	bottom := m.scrollOffset
	if bottom <= 0 {
		t.Fatal("expected scrollable explanation")
	}
	m = press(m, "k")
	if m.scrollOffset != bottom-1 {
		t.Fatalf("up did not move from bottom: %d -> %d", bottom, m.scrollOffset)
	}
}

func TestDeveloperScopeDoesNotBecomeRepositoryScope(t *testing.T) {
	m := regressionModel()
	m.Workspace.Changes[0].Author.Name = "Yeu"
	m = press(press(m, "d"), "enter")
	if len(m.filteredChanges()) != 2 {
		t.Fatal("developer cross-repository changes lost")
	}
	m = press(m, "j")
	expected, _ := m.selectedChange()
	m = press(m, "e")
	actual, _ := m.selectedChange()
	if expected.Hash != actual.Hash {
		t.Fatalf("selected change changed: %s -> %s", expected.Hash, actual.Hash)
	}
}

func TestWorkspaceListsKeepSelectionVisible(t *testing.T) {
	for _, width := range []int{60, 80, 100, 140} {
		for _, focus := range []Focus{FocusDevelopers, FocusRepositories} {
			t.Run(fmt.Sprintf("width-%d-focus-%d", width, focus), func(t *testing.T) {
				m := regressionModel()
				m.Width = width
				m.Workspace.Developers = nil
				m.Workspace.Repositories = nil
				for i := 0; i < 50; i++ {
					m.Workspace.Developers = append(m.Workspace.Developers, model.DeveloperStats{Developer: fmt.Sprintf("dev-%02d", i)})
					m.Workspace.Repositories = append(m.Workspace.Repositories, model.RepositoryStats{Repository: model.Repository{ID: fmt.Sprintf("r%d", i), Name: fmt.Sprintf("repo-%02d", i)}})
				}
				m.Focus = focus
				for i := 0; i < 49; i++ {
					m = press(m, "j")
				}
				want := "dev-49"
				if focus == FocusRepositories {
					want = "repo-49"
				}
				if !strings.Contains(m.View(), want) {
					t.Fatalf("selected %s hidden at width %d", want, width)
				}
			})
		}
	}
}

func TestChangeListPageKeysMoveSelection(t *testing.T) {
	m := regressionModel()
	m.Workspace.Changes = nil
	for i := 0; i < 50; i++ {
		m.Workspace.Changes = append(m.Workspace.Changes, model.ChangeUnit{RepositoryID: "r1", Hash: fmt.Sprintf("%07d", i), Author: model.Author{Name: "Yeu"}, Message: fmt.Sprintf("commit-%02d", i)})
	}
	m = press(press(m, "r"), "enter")
	m = press(m, "pgdown")
	if m.SelectedChange <= 0 {
		t.Fatal("page down did not move selection")
	}
	selected, _ := m.selectedChange()
	if !strings.Contains(m.View(), selected.Message) {
		t.Fatal("paged selection hidden")
	}
	m = press(m, "pgup")
	if m.SelectedChange != 0 {
		t.Fatalf("page up failed: %d", m.SelectedChange)
	}
}

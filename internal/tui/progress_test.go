package tui

import (
	"context"
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tdawn0-0/git-analyzer/internal/config"
	"strings"
	"testing"

	"github.com/tdawn0-0/git-analyzer/internal/git"
	"github.com/tdawn0-0/git-analyzer/internal/model"
)

func TestAnalysisProgressUpdatesBeforeFinish(t *testing.T) {
	m := New(context.Background(), Options{Root: "/ws"})
	m.phase = "analyzing"
	m.repos = []model.Repository{{ID: "a", Name: "alpha"}, {ID: "b", Name: "beta"}, {ID: "c", Name: "gamma"}}
	next, _ := m.Update(RepositoryAnalyzedMsg{Index: 1, Total: 3, Result: git.RepoResult{Repository: m.repos[1], Status: model.StatusComplete}})
	m = next.(Model)
	if !m.Loading || m.analyzedCount != 1 {
		t.Fatalf("no intermediate progress: loading=%v done=%d", m.Loading, m.analyzedCount)
	}
	if !strings.Contains(m.View(), "1 / 3") || !strings.Contains(m.View(), "beta  Complete") {
		t.Fatalf("progress not rendered: %s", m.View())
	}
	// Worker completions may arrive out of order and must remain attached to their repo.
	next, _ = m.Update(RepositoryAnalyzedMsg{Index: 0, Total: 3, Result: git.RepoResult{Repository: m.repos[0], Status: model.StatusError, Message: "unavailable"}})
	m = next.(Model)
	if m.analyzedCount != 2 || !strings.Contains(m.View(), "alpha  Error") {
		t.Fatalf("out-of-order progress lost: %s", m.View())
	}
	// A duplicate event must not count twice.
	next, _ = m.Update(RepositoryAnalyzedMsg{Index: 0, Total: 3, Result: git.RepoResult{Repository: m.repos[0], Status: model.StatusError}})
	if next.(Model).analyzedCount != 2 {
		t.Fatal("duplicate progress counted twice")
	}
}

func TestAnalysisStreamOrdersProgressBeforeFinal(t *testing.T) {
	for _, count := range []int{0, 3} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			repos := make([]model.Repository, count)
			for i := range repos {
				repos[i] = model.Repository{ID: fmt.Sprint(i), Name: fmt.Sprint(i), Path: t.TempDir()}
			}
			events := make(chan tea.Msg, 2*count+1)
			done := make(chan struct{})
			go func() {
				defer close(done)
				streamAnalysisCmd(ctx, "/ws", repos, config.Defaults(), Options{Branch: "missing", Jobs: 2}, events)()
			}()
			started, finished := map[int]bool{}, map[int]bool{}
			for {
				msg := waitAnalysisEventCmd(ctx, events)()
				switch msg := msg.(type) {
				case RepositoryAnalyzedMsg:
					if msg.Result.Status == model.StatusAnalyzing {
						started[msg.Index] = true
					} else {
						if !started[msg.Index] {
							t.Fatal("completion arrived before start")
						}
						finished[msg.Index] = true
					}
				case AnalysisFinishedMsg:
					if len(started) != count || len(finished) != count {
						t.Fatalf("final arrived before progress: started=%d finished=%d", len(started), len(finished))
					}
					<-done
					if msg := waitAnalysisEventCmd(ctx, events)(); msg != nil {
						t.Fatalf("event after final: %+v", msg)
					}
					return
				default:
					t.Fatalf("unexpected stream event: %+v", msg)
				}
			}
		})
	}
}

func TestWaitingForProgressCanBeCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan tea.Msg)
	cancel()
	msg := waitAnalysisEventCmd(ctx, events)()
	err, ok := msg.(AnalysisErrorMsg)
	if !ok || !errors.Is(err.Err, context.Canceled) {
		t.Fatalf("cancel did not stop waiter: %+v", msg)
	}
}

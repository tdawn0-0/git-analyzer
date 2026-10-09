package tui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/tdawn0-0/git-analyzer/internal/config"
	"github.com/tdawn0-0/git-analyzer/internal/git"
	"github.com/tdawn0-0/git-analyzer/internal/model"
)

// View is the primary navigation screen.
type View int

const (
	ViewWorkspace View = iota
	ViewRepository
	ViewDeveloper
	ViewChanges
	ViewExplain
)

// Focus is the active panel within a view.
type Focus int

const (
	FocusDevelopers Focus = iota
	FocusRepositories
	FocusChanges
	FocusActivity
	FocusProfile
)

// SortMode controls list ordering (Activity = Change Intensity).
type SortMode int

const (
	SortActivity SortMode = iota
	SortName
	SortChanges
	SortAdded
	SortDeleted
)

// TrendMode selects which chart series to show.
type TrendMode int

const (
	TrendActivity TrendMode = iota
	TrendCodeChange
)

// Options configures analysis for the TUI session.
type Options struct {
	Root     string
	Since    string
	Until    string
	Author   string
	Repo     string
	Branch   string
	MaxDepth int
	Jobs     int
	Exclude  []string
	Config   config.Config
}

// Model is the Bubble Tea root model.
type changeScope struct {
	repositoryID string
	developer    string
}

type navigationFrame struct {
	selectedDeveloper  int
	selectedRepository int
	active             View
	focus              Focus
	scope              changeScope
	filter             string
	selectedChange     int
	scrollOffset       int
}

type Model struct {
	Width  int
	Height int

	Active View

	Workspace *model.WorkspaceStats

	SelectedDeveloper  int
	SelectedRepository int
	SelectedChange     int

	Focus Focus

	Loading bool
	Err     error

	opts   Options
	ctx    context.Context
	cancel context.CancelFunc

	phase          string // discovering | analyzing | ready
	repos          []model.Repository
	repoResults    []git.RepoResult
	analyzedCount  int
	analysisEvents <-chan tea.Msg
	showHelp       bool
	filter         string
	filtering      bool
	sort           SortMode
	trend          TrendMode
	scope          changeScope
	history        []navigationFrame
	scrollOffset   int
	statusHint     string
}

// New creates a TUI model. Analysis starts in Init via tea.Cmd.
func New(parent context.Context, opts Options) Model {
	ctx, cancel := context.WithCancel(parent)
	return Model{
		Active:     ViewWorkspace,
		Focus:      FocusDevelopers,
		Loading:    true,
		phase:      "discovering",
		opts:       opts,
		ctx:        ctx,
		cancel:     cancel,
		sort:       SortActivity,
		trend:      TrendActivity,
		statusHint: defaultStatusHint,
	}
}

const defaultStatusHint = "j/k move · Enter open · Tab focus · e explain · t trend · ? help · q quit"

// Init starts workspace discovery.
func (m Model) Init() tea.Cmd {
	return scanWorkspaceCmd(m.ctx, m.opts)
}

// Run launches the Bubble Tea program with alt screen.
func Run(ctx context.Context, opts Options) error {
	m := New(ctx, opts)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err := p.Run()
	return err
}

func (m Model) filteredDevelopers() []model.DeveloperStats {
	if m.Workspace == nil {
		return nil
	}
	devs := append([]model.DeveloperStats(nil), m.Workspace.Developers...)
	if m.filter != "" {
		q := strings.ToLower(m.filter)
		var out []model.DeveloperStats
		for _, d := range devs {
			if strings.Contains(strings.ToLower(d.Developer), q) {
				out = append(out, d)
			}
		}
		devs = out
	}
	sortDevelopers(devs, m.sort)
	return devs
}

func (m Model) filteredRepositories() []model.RepositoryStats {
	if m.Workspace == nil {
		return nil
	}
	repos := append([]model.RepositoryStats(nil), m.Workspace.Repositories...)
	if m.filter != "" {
		q := strings.ToLower(m.filter)
		var out []model.RepositoryStats
		for _, r := range repos {
			if strings.Contains(strings.ToLower(r.Repository.Name), q) {
				out = append(out, r)
			}
		}
		repos = out
	}
	sortRepositories(repos, m.sort)
	return repos
}

func (m Model) filteredChanges() []model.ChangeUnit {
	if m.Workspace == nil {
		return nil
	}
	changes := m.Workspace.Changes
	switch m.Active {
	case ViewRepository:
		repos := m.filteredRepositories()
		if len(repos) == 0 {
			return nil
		}
		idx := clampIndex(m.SelectedRepository, len(repos))
		id := repos[idx].Repository.ID
		var out []model.ChangeUnit
		for _, c := range changes {
			if c.RepositoryID == id {
				out = append(out, c)
			}
		}
		changes = out
	case ViewDeveloper:
		devs := m.filteredDevelopers()
		if len(devs) == 0 {
			return nil
		}
		idx := clampIndex(m.SelectedDeveloper, len(devs))
		name := devs[idx].Developer
		var out []model.ChangeUnit
		for _, c := range changes {
			if c.Author.Name == name {
				out = append(out, c)
			}
		}
		changes = out
	case ViewChanges, ViewExplain:
		var out []model.ChangeUnit
		for _, c := range changes {
			if m.scope.repositoryID != "" && c.RepositoryID != m.scope.repositoryID {
				continue
			}
			if m.scope.developer != "" && c.Author.Name != m.scope.developer {
				continue
			}
			out = append(out, c)
		}
		changes = out
	}
	if m.filter != "" && (m.Active == ViewChanges || m.Active == ViewExplain) {
		q := strings.ToLower(m.filter)
		var out []model.ChangeUnit
		for _, c := range changes {
			if strings.Contains(strings.ToLower(c.Message), q) ||
				strings.Contains(strings.ToLower(c.Hash), q) ||
				strings.Contains(strings.ToLower(c.Author.Name), q) {
				out = append(out, c)
			}
		}
		changes = out
	}
	return changes
}

func (m Model) selectedChange() (model.ChangeUnit, bool) {
	changes := m.filteredChanges()
	if len(changes) == 0 {
		return model.ChangeUnit{}, false
	}
	return changes[clampIndex(m.SelectedChange, len(changes))], true
}

func clampIndex(i, n int) int {
	if n <= 0 {
		return 0
	}
	if i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}

func (m Model) currentScope() changeScope {
	switch m.Active {
	case ViewRepository:
		repos := m.filteredRepositories()
		if len(repos) > 0 {
			return changeScope{repositoryID: repos[clampIndex(m.SelectedRepository, len(repos))].Repository.ID}
		}
	case ViewDeveloper:
		devs := m.filteredDevelopers()
		if len(devs) > 0 {
			return changeScope{developer: devs[clampIndex(m.SelectedDeveloper, len(devs))].Developer}
		}
	case ViewChanges, ViewExplain:
		return m.scope
	}
	return changeScope{}
}

func (m Model) navigate(view View) Model {
	frame := navigationFrame{active: m.Active, focus: m.Focus, scope: m.scope, filter: m.filter, selectedChange: m.SelectedChange, scrollOffset: m.scrollOffset, selectedDeveloper: m.SelectedDeveloper, selectedRepository: m.SelectedRepository}
	scope := m.currentScope()
	// Copy before appending: Bubble Tea models are values and must not share a mutable stack.
	m.history = append(append([]navigationFrame(nil), m.history...), frame)
	m.Active = view
	m.scrollOffset = 0
	if view == ViewChanges || view == ViewExplain {
		m.scope = scope
		if frame.active != ViewChanges && frame.active != ViewExplain {
			m.filter = ""
		}
	} else {
		m.scope = changeScope{}
		m.SelectedChange = 0
	}
	m.Focus = FocusChanges
	return m
}

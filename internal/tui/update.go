package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tdawn0-0/git-analyzer/internal/git"
	"github.com/tdawn0-0/git-analyzer/internal/model"
)

// Update handles Bubble Tea messages. Never calls git directly.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case WorkspaceDiscoveredMsg:
		m.phase = "analyzing"
		m.repos = msg.Repos
		m.opts.Config = msg.Cfg
		m.statusHint = "Analyzing repositories…"
		root := msg.Root
		repos := msg.Repos
		cfg := msg.Cfg
		opts := m.opts
		m.repoResults = make([]git.RepoResult, len(repos))
		m.analyzedCount = 0
		// At most two events per checkout plus one final result. The bounded queue
		// lets worker callbacks finish even if the UI is quitting.
		events := make(chan tea.Msg, 2*len(repos)+1)
		m.analysisEvents = events
		return m, tea.Batch(streamAnalysisCmd(m.ctx, root, repos, cfg, opts, events), waitAnalysisEventCmd(m.ctx, events))

	case RepositoryAnalyzedMsg:
		if !m.Loading || msg.Index < 0 || msg.Index >= len(m.repos) {
			return m, nil
		}
		if len(m.repoResults) != len(m.repos) {
			m.repoResults = make([]git.RepoResult, len(m.repos))
		}
		previous := m.repoResults[msg.Index].Status
		if !repositoryFinished(previous) {
			m.repoResults[msg.Index] = msg.Result
			if repositoryFinished(msg.Result.Status) {
				m.analyzedCount++
			}
		}
		return m, waitAnalysisEventCmd(m.ctx, m.analysisEvents)

	case AnalysisFinishedMsg:
		m.Loading = false
		m.phase = "ready"
		ws := msg.Stats
		m.Workspace = &ws
		m.repoResults = nil
		m.analyzedCount = len(m.repos)
		m.analysisEvents = nil
		m.SelectedDeveloper = 0
		m.SelectedRepository = 0
		m.SelectedChange = 0
		m.statusHint = defaultStatusHint
		return m, nil

	case AnalysisErrorMsg:
		m.Loading = false
		m.Err = msg.Err
		m.analysisEvents = nil
		m.phase = "ready"
		m.statusHint = "Analysis error — press q to quit"
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filtering {
		return m.handleFilterInput(msg)
	}
	if m.showHelp {
		switch msg.String() {
		case "?", "esc", "q", "h":
			m.showHelp = false
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c":
		if m.cancel != nil {
			m.cancel()
		}
		if m.Loading {
			m.Loading = false
			m.statusHint = "Cancelled"
			return m, nil
		}
		return m, tea.Quit
	case "q":
		if m.cancel != nil {
			m.cancel()
		}
		return m, tea.Quit
	case "?":
		m.showHelp = true
		return m, nil
	case "/":
		m.filtering = true
		m.filter = ""
		m.statusHint = "Filter: (enter to apply, esc cancel)"
		return m, nil
	case "s":
		m.sort = cycleSort(m.sort)
		m.statusHint = "Sort: " + sortLabel(m.sort)
		return m, nil
	case "t":
		if m.trend == TrendActivity {
			m.trend = TrendCodeChange
			m.statusHint = "Trend: Code Change (+/−)"
		} else {
			m.trend = TrendActivity
			m.statusHint = "Trend: Change Intensity"
		}
		return m, nil
	case "tab":
		m.Focus = (m.Focus + 1) % 5
		return m, nil
	case "d":
		return m.navigate(ViewDeveloper), nil
	case "r":
		return m.navigate(ViewRepository), nil
	case "e":
		if m.Active != ViewExplain {
			if _, ok := m.selectedChange(); ok {
				m = m.navigate(ViewExplain)
			}
		}
		return m, nil
	case "pgdown", "ctrl+d":
		m.pageMove(1)
		return m, nil
	case "pgup", "ctrl+u":
		m.pageMove(-1)
		return m, nil
	case "home":
		m.scrollOffset = 0
		return m, nil
	case "end":
		vp := m.bodyViewport()
		vp.GotoBottom()
		m.scrollOffset = vp.YOffset
		return m, nil
	case "h", "left", "esc", "backspace":
		return m.goBack(), nil
	case "l", "right", "enter":
		return m.enterSelection(), nil
	case "j", "down":
		if m.Active == ViewExplain {
			m.scrollBy(1)
		} else {
			m.moveSelection(1)
			m.scrollOffset = 0
		}
		return m, nil
	case "k", "up":
		if m.Active == ViewExplain {
			m.scrollBy(-1)
		} else {
			m.moveSelection(-1)
			m.scrollOffset = 0
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.SelectedDeveloper, m.SelectedRepository, m.SelectedChange, m.scrollOffset = 0, 0, 0, 0
	switch msg.String() {
	case "esc", "ctrl+c":
		m.filtering = false
		m.filter = ""
		m.statusHint = defaultStatusHint
	case "enter":
		m.filtering = false
		m.statusHint = defaultStatusHint
		if m.filter != "" {
			m.statusHint = "Filter: " + m.filter
		}
	case "backspace":
		if len(m.filter) > 0 {
			runes := []rune(m.filter)
			m.filter = string(runes[:len(runes)-1])
		}
	default:
		if len(msg.Runes) == 1 && msg.Type == tea.KeyRunes {
			m.filter += string(msg.Runes)
		} else if len(msg.String()) == 1 {
			m.filter += msg.String()
		}
	}
	return m, nil
}

func (m Model) goBack() Model {
	if len(m.history) == 0 {
		return m
	}
	frame := m.history[len(m.history)-1]
	m.history = m.history[:len(m.history)-1]
	m.Active, m.Focus, m.scope, m.filter = frame.active, frame.focus, frame.scope, frame.filter
	m.SelectedChange, m.scrollOffset = frame.selectedChange, frame.scrollOffset
	m.SelectedDeveloper, m.SelectedRepository = frame.selectedDeveloper, frame.selectedRepository
	return m
}

func (m Model) enterSelection() Model {
	switch m.Active {
	case ViewWorkspace:
		if m.Focus == FocusRepositories {
			return m.navigate(ViewRepository)
		}
		return m.navigate(ViewDeveloper)
	case ViewDeveloper, ViewRepository:
		return m.navigate(ViewChanges)
	case ViewChanges:
		if _, ok := m.selectedChange(); ok {
			return m.navigate(ViewExplain)
		}
	}
	return m
}

func (m *Model) moveSelection(delta int) {
	switch m.Active {
	case ViewWorkspace:
		switch m.Focus {
		case FocusRepositories:
			n := len(m.filteredRepositories())
			if n == 0 {
				return
			}
			m.SelectedRepository = clampIndex(m.SelectedRepository+delta, n)
		default:
			n := len(m.filteredDevelopers())
			if n == 0 {
				return
			}
			m.SelectedDeveloper = clampIndex(m.SelectedDeveloper+delta, n)
		}
	case ViewDeveloper:
		if m.Focus == FocusRepositories {
			n := len(m.filteredRepositories())
			if n == 0 {
				return
			}
			m.SelectedRepository = clampIndex(m.SelectedRepository+delta, n)
		} else {
			n := len(m.filteredChanges())
			if n == 0 {
				return
			}
			m.SelectedChange = clampIndex(m.SelectedChange+delta, n)
		}
	case ViewRepository, ViewChanges:
		n := len(m.filteredChanges())
		if n == 0 {
			return
		}
		m.SelectedChange = clampIndex(m.SelectedChange+delta, n)
	case ViewExplain:
		n := len(m.filteredChanges())
		if n == 0 {
			return
		}
		m.SelectedChange = clampIndex(m.SelectedChange+delta, n)
	}
}

func (m *Model) pageMove(direction int) {
	rows := max(m.bodyHeight()-5, 1)
	if m.Active == ViewExplain || (m.Active == ViewWorkspace && (m.Focus == FocusActivity || m.Focus == FocusProfile)) {
		m.scrollBy(direction * max(m.bodyHeight()-1, 1))
		return
	}
	if m.Active == ViewWorkspace || m.Active == ViewDeveloper {
		rows = max(m.bodyHeight()/2-4, 1)
	}
	m.moveSelection(direction * rows)
	m.scrollOffset = 0
}

func repositoryFinished(status model.RepositoryStatus) bool {
	return status == model.StatusComplete || status == model.StatusError || status == model.StatusSkipped
}

package tui

import (
	tea "github.com/charmbracelet/bubbletea"
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
		return m, startAnalysisCmd(m.ctx, root, repos, cfg, opts)

	case AnalysisFinishedMsg:
		m.Loading = false
		m.phase = "ready"
		ws := msg.Stats
		m.Workspace = &ws
		m.repoResults = nil
		m.analyzedCount = len(ws.Repositories)
		m.SelectedDeveloper = 0
		m.SelectedRepository = 0
		m.SelectedChange = 0
		m.statusHint = defaultStatusHint
		return m, nil

	case AnalysisErrorMsg:
		m.Loading = false
		m.Err = msg.Err
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
		m.Active = ViewDeveloper
		m.Focus = FocusChanges
		return m, nil
	case "r":
		m.Active = ViewRepository
		m.Focus = FocusChanges
		return m, nil
	case "e":
		if _, ok := m.selectedChange(); ok {
			m.Active = ViewExplain
		} else if m.Active == ViewWorkspace || m.Active == ViewDeveloper || m.Active == ViewRepository {
			// Prefer changes context: jump to changes then explain if possible
			changes := m.filteredChanges()
			if len(changes) > 0 {
				m.Active = ViewExplain
			}
		}
		return m, nil
	case "h", "left", "esc", "backspace":
		return m.goBack(), nil
	case "l", "right", "enter":
		return m.enterSelection(), nil
	case "j", "down":
		m.moveSelection(1)
		return m, nil
	case "k", "up":
		m.moveSelection(-1)
		return m, nil
	}
	return m, nil
}

func (m Model) handleFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			m.filter = m.filter[:len(m.filter)-1]
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
	switch m.Active {
	case ViewExplain:
		m.Active = ViewChanges
	case ViewChanges:
		if m.Focus == FocusDevelopers {
			m.Active = ViewDeveloper
		} else {
			m.Active = ViewRepository
		}
	case ViewDeveloper, ViewRepository:
		m.Active = ViewWorkspace
		m.Focus = FocusDevelopers
	}
	return m
}

func (m Model) enterSelection() Model {
	switch m.Active {
	case ViewWorkspace:
		switch m.Focus {
		case FocusDevelopers:
			m.Active = ViewDeveloper
			m.Focus = FocusChanges
		case FocusRepositories:
			m.Active = ViewRepository
			m.Focus = FocusChanges
		default:
			m.Active = ViewDeveloper
		}
	case ViewDeveloper, ViewRepository:
		m.Active = ViewChanges
	case ViewChanges:
		if _, ok := m.selectedChange(); ok {
			m.Active = ViewExplain
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

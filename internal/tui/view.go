package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/tdawn0-0/git-analyzer/internal/model"
)

// View renders the current screen. Never calls git.
func (m Model) View() string {
	if m.Width == 0 {
		m.Width = 100
	}
	if m.Height == 0 {
		m.Height = 30
	}

	if m.Loading {
		return m.viewLoading()
	}
	if m.Err != nil {
		return styleErr().Render("ERR "+m.Err.Error()) + "\n" + styleMuted().Render("q quit")
	}
	if m.showHelp {
		return m.viewHelp()
	}

	var body string
	switch m.Active {
	case ViewRepository:
		body = m.viewRepository()
	case ViewDeveloper:
		body = m.viewDeveloper()
	case ViewChanges:
		body = m.viewChanges()
	case ViewExplain:
		body = m.viewExplain()
	default:
		body = m.viewWorkspace()
	}

	header := m.viewHeader()
	status := m.viewStatus()
	// Fit content into terminal height.
	avail := m.Height - lipgloss.Height(header) - lipgloss.Height(status) - 1
	if avail < 5 {
		avail = 5
	}
	body = lipgloss.NewStyle().Height(avail).MaxHeight(avail).Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, status)
}

func (m Model) viewHeader() string {
	root := m.opts.Root
	repos, devs, changes := 0, 0, 0
	if m.Workspace != nil {
		root = m.Workspace.Root
		repos = len(m.Workspace.Repositories)
		devs = len(m.Workspace.Developers)
		changes = len(m.Workspace.Changes)
	}
	rangeLabel := ""
	if m.opts.Since != "" || m.opts.Until != "" {
		rangeLabel = fmt.Sprintf(" · %s → %s", dash(m.opts.Since), dash(m.opts.Until))
	}
	title := fmt.Sprintf("git-workstats · %s · %d repos · %d developers · %d changes%s",
		root, repos, devs, changes, rangeLabel)
	viewName := map[View]string{
		ViewWorkspace:  "Workspace",
		ViewRepository: "Repository",
		ViewDeveloper:  "Developer",
		ViewChanges:    "Changes",
		ViewExplain:    "Explain",
	}[m.Active]
	line := styleTitle().Render(title) + "  " + styleAccent().Render("["+viewName+"]")
	return line
}

func dash(s string) string {
	if s == "" {
		return "…"
	}
	return s
}

func (m Model) viewStatus() string {
	hint := m.statusHint
	if m.filtering {
		hint = "Filter> " + m.filter + "█"
	}
	extra := fmt.Sprintf("sort=%s · trend=%s", sortLabel(m.sort), trendLabel(m.trend))
	return styleStatusBar().Width(m.Width).Render(hint + "  ·  " + extra)
}

func trendLabel(t TrendMode) string {
	if t == TrendCodeChange {
		return "code"
	}
	return "intensity"
}

func (m Model) viewLoading() string {
	var b strings.Builder
	b.WriteString(styleTitle().Render("git-workstats") + "\n\n")
	switch m.phase {
	case "discovering":
		b.WriteString("Discovering repositories...\n")
	default:
		total := len(m.repos)
		done := m.analyzedCount
		b.WriteString(fmt.Sprintf("Analyzing repositories  %s  %d / %d\n\n", progressBar(done, max(total, 1), 24), done, total))
		limit := 12
		for i, r := range m.repos {
			if i >= limit {
				b.WriteString(styleMuted().Render(fmt.Sprintf("  … %d more", len(m.repos)-limit)) + "\n")
				break
			}
			st := "waiting"
			glyph := "·"
			if i < len(m.repoResults) {
				glyph = statusGlyph(m.repoResults[i].Status)
				st = string(m.repoResults[i].Status)
				if m.repoResults[i].Message != "" {
					st += " (" + m.repoResults[i].Message + ")"
				}
			} else if m.phase == "analyzing" {
				st = "queued"
			}
			line := fmt.Sprintf("  %s  %s  %s", glyph, r.Name, st)
			if glyph == "ERR" {
				b.WriteString(styleErr().Render(line) + "\n")
			} else if glyph == "✓" {
				b.WriteString(styleOK().Render(line) + "\n")
			} else {
				b.WriteString(styleMuted().Render(line) + "\n")
			}
		}
	}
	b.WriteString("\n" + styleMuted().Render("Ctrl+C cancel"))
	return b.String()
}

func (m Model) viewHelp() string {
	body := `Keys
  j/k or ↑/↓     move selection
  h/← or Esc     back
  l/→ or Enter   open / drill down
  Tab            cycle panel focus
  e              Score Explain (Change Intensity)
  r / d          Repository / Developer view
  t              toggle trend (Intensity ↔ Code Change)
  s              cycle sort (Activity, Name, Changes, Added, Deleted)
  /              filter
  ?              help
  q / Ctrl+C     quit (Ctrl+C also cancels loading)

Wording
  Change Intensity is engineering activity — not employee performance.
  Sort by Activity = total Change Intensity.`
	return styleHelp().Width(min(m.Width-4, 72)).Render(body)
}

func (m Model) viewWorkspace() string {
	devs := renderDeveloperList(m.filteredDevelopers(), m.SelectedDeveloper, m.Focus == FocusDevelopers)
	repos := renderRepositoryList(m.filteredRepositories(), m.SelectedRepository, m.Focus == FocusRepositories)
	chart := renderTrendChart(m.timelineBuckets(), m.trend, panelChartWidth(m.Width), panelChartHeight(m.Height))
	profile := renderProfileBars(m.workspaceTypeCounts(), panelChartWidth(m.Width))

	mode := layoutFor(m.Width)
	switch mode {
	case layoutFour:
		halfW := (m.Width / 2) - 2
		halfH := (m.Height / 2) - 4
		if halfH < 6 {
			halfH = 6
		}
		left := lipgloss.JoinVertical(lipgloss.Left,
			stylePanel(m.Focus == FocusDevelopers, halfW, halfH).Render(styleHeader().Render("Developers (Activity)")+"\n"+devs),
			stylePanel(m.Focus == FocusRepositories, halfW, halfH).Render(styleHeader().Render("Repositories")+"\n"+repos),
		)
		right := lipgloss.JoinVertical(lipgloss.Left,
			stylePanel(m.Focus == FocusActivity, halfW, halfH).Render(styleHeader().Render(trendTitle(m.trend))+"\n"+chart),
			stylePanel(m.Focus == FocusProfile, halfW, halfH).Render(styleHeader().Render("Engineering Profile")+"\n"+profile),
		)
		return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	case layoutDual:
		colW := (m.Width / 2) - 2
		h := m.Height - 6
		left := stylePanel(true, colW, h).Render(styleHeader().Render("Developers")+"\n"+devs+"\n\n"+styleHeader().Render("Repositories")+"\n"+repos)
		right := stylePanel(false, colW, h).Render(styleHeader().Render(trendTitle(m.trend))+"\n"+chart+"\n\n"+styleHeader().Render("Engineering Profile")+"\n"+profile)
		return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	default:
		return strings.Join([]string{
			styleHeader().Render("Developers"), devs,
			styleHeader().Render("Repositories"), repos,
			styleHeader().Render(trendTitle(m.trend)), chart,
			styleHeader().Render("Engineering Profile"), profile,
		}, "\n")
	}
}

func (m Model) viewRepository() string {
	repos := m.filteredRepositories()
	name := "(none)"
	var selected model.RepositoryStats
	if len(repos) > 0 {
		selected = repos[clampIndex(m.SelectedRepository, len(repos))]
		name = selected.Repository.Name
		if selected.Status != model.StatusComplete {
			name += " [" + string(selected.Status)
			if selected.Error != "" {
				name += ": " + selected.Error
			}
			name += "]"
		}
	}
	changes := renderChangeList(m.filteredChanges(), m.SelectedChange, true)
	chart := renderTrendChart(m.timelineBuckets(), m.trend, max(m.Width/2-4, 40), 10)
	types := renderProfileBars(selected.TypeDistribution, max(m.Width/2-4, 40))
	top := styleHeader().Render("Repository: "+name) + "\n"
	return top + lipgloss.JoinHorizontal(lipgloss.Top,
		stylePanel(true, max(m.Width/2-2, 30), max(m.Height-8, 12)).Render(styleHeader().Render("Changes")+"\n"+changes),
		stylePanel(false, max(m.Width/2-2, 30), max(m.Height-8, 12)).Render(styleHeader().Render(trendTitle(m.trend))+"\n"+chart+"\n"+styleHeader().Render("Types")+"\n"+types),
	)
}

func (m Model) viewDeveloper() string {
	devs := m.filteredDevelopers()
	name := "(none)"
	var selected model.DeveloperStats
	if len(devs) > 0 {
		selected = devs[clampIndex(m.SelectedDeveloper, len(devs))]
		name = selected.Developer
	}
	changes := renderChangeList(m.filteredChanges(), m.SelectedChange, true)
	chart := renderTrendChart(m.timelineBuckets(), m.trend, max(m.Width-6, 40), 10)
	profile := renderProfileBars(selected.TypeDistribution, max(m.Width-6, 40))
	meta := fmt.Sprintf("Activity=%.2f  avg=%.2f  changes=%d  +%d/-%d  repos=%d  cross-module=%d",
		selected.TotalScore, selected.AverageScore, selected.ChangeCount,
		selected.AddedLines, selected.DeletedLines, len(selected.RepositoryIDs), selected.CrossModuleChanges)
	return styleHeader().Render("Developer: "+name) + "\n" + styleMuted().Render(meta) + "\n" +
		lipgloss.JoinVertical(lipgloss.Left,
			stylePanel(true, m.Width-2, max(m.Height/2-4, 8)).Render(styleHeader().Render("Change Units")+"\n"+changes),
			stylePanel(false, m.Width-2, max(m.Height/2-4, 8)).Render(styleHeader().Render(trendTitle(m.trend))+"\n"+chart+"\n"+profile),
		)
}

func (m Model) viewChanges() string {
	changes := renderChangeList(m.filteredChanges(), m.SelectedChange, true)
	return styleHeader().Render("Changes") + "\n" +
		styleMuted().Render("Enter/e → Change Intensity Explanation") + "\n" +
		stylePanel(true, m.Width-2, max(m.Height-6, 10)).Render(changes)
}

func (m Model) viewExplain() string {
	c, ok := m.selectedChange()
	if !ok {
		return styleMuted().Render("No change selected. Open a change list and press e.")
	}
	repoName := c.RepositoryID
	if m.Workspace != nil {
		for _, r := range m.Workspace.Repositories {
			if r.Repository.ID == c.RepositoryID {
				repoName = r.Repository.Name
				break
			}
		}
	}
	return renderExplain(c, repoName, m.Width)
}

func (m Model) timelineBuckets() []model.DailyBucket {
	if m.Workspace == nil {
		return nil
	}
	switch m.Active {
	case ViewDeveloper:
		devs := m.filteredDevelopers()
		if len(devs) == 0 {
			return m.Workspace.Timeline
		}
		name := devs[clampIndex(m.SelectedDeveloper, len(devs))].Developer
		var changes []model.ChangeUnit
		for _, c := range m.Workspace.Changes {
			if c.Author.Name == name {
				changes = append(changes, c)
			}
		}
		return timelineFromChanges(changes)
	case ViewRepository:
		repos := m.filteredRepositories()
		if len(repos) == 0 {
			return m.Workspace.Timeline
		}
		id := repos[clampIndex(m.SelectedRepository, len(repos))].Repository.ID
		var changes []model.ChangeUnit
		for _, c := range m.Workspace.Changes {
			if c.RepositoryID == id {
				changes = append(changes, c)
			}
		}
		return timelineFromChanges(changes)
	default:
		return m.Workspace.Timeline
	}
}

func (m Model) workspaceTypeCounts() map[model.ChangeType]int {
	out := map[model.ChangeType]int{}
	if m.Workspace == nil {
		return out
	}
	for _, c := range m.Workspace.Changes {
		out[c.Type]++
	}
	return out
}

func trendTitle(t TrendMode) string {
	if t == TrendCodeChange {
		return "Code Change Trend (+/−)"
	}
	return "Activity Trend (Change Intensity)"
}

func panelChartWidth(termW int) int {
	w := termW/2 - 6
	if w < 30 {
		w = max(termW-6, 20)
	}
	return w
}

func panelChartHeight(termH int) int {
	h := termH/2 - 8
	if h < 6 {
		h = 6
	}
	if h > 14 {
		h = 14
	}
	return h
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

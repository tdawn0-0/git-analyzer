package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/tdawn0-0/git-analyzer/internal/aggregate"
	"github.com/tdawn0-0/git-analyzer/internal/model"
)

type tableColumn struct {
	label   string
	width   int
	numeric bool
}

type tableRow struct {
	values []string
	style  lipgloss.Style
}

func renderTable(columns []tableColumn, rows []tableRow, selected, width int, empty string) string {
	format := func(values []string) string {
		cells := make([]string, len(columns))
		for i, col := range columns {
			cell := ansi.Truncate(values[i], max(col.width, 1), "…")
			padding := strings.Repeat(" ", max(col.width-lipgloss.Width(cell), 0))
			if col.numeric {
				cell = padding + cell
			} else {
				cell += padding
			}
			cells[i] = cell
		}
		return strings.Join(cells, " ")
	}
	labels := make([]string, len(columns))
	for i, col := range columns {
		labels[i] = col.label
	}
	lines := []string{styleHeader().Render(fitLines("  "+format(labels), width))}
	if len(rows) == 0 {
		return lines[0] + "\n" + styleMuted().Render(empty)
	}
	for i, row := range rows {
		cursor := "  "
		if i == selected {
			cursor = "> "
		}
		lines = append(lines, row.style.Render(fitLines(cursor+format(row.values), width)))
	}
	return strings.Join(lines, "\n")
}

func renderDeveloperList(devs []model.DeveloperStats, selected int, focused bool, width int) string {
	columns := []tableColumn{{"Developer", max(width-20, 9), false}, {"Intensity", 9, true}, {"Commits", 7, true}}
	if width >= 54 {
		columns[0].width = width - 36
		columns = append(columns, tableColumn{"Added", 7, true}, tableColumn{"Deleted", 7, true})
	}
	rows := make([]tableRow, 0, len(devs))
	for i, d := range devs {
		values := []string{d.Developer, fmt.Sprintf("%.2f", d.TotalScore), fmt.Sprint(d.ChangeCount)}
		if len(columns) > 3 {
			values = append(values, fmt.Sprint(d.AddedLines), fmt.Sprint(d.DeletedLines))
		}
		style := lipgloss.NewStyle()
		if i == selected && focused {
			style = styleAccent()
		}
		rows = append(rows, tableRow{values, style})
	}
	return renderTable(columns, rows, selected, width, "(no developers)")
}

func renderRepositoryList(repos []model.RepositoryStats, selected int, focused bool, width int) string {
	columns := []tableColumn{{"State", 5, false}, {"Repo", max(width-26, 4), false}, {"Intensity", 9, true}, {"Commits", 7, true}}
	rows := make([]tableRow, 0, len(repos))
	for i, r := range repos {
		state := statusGlyph(r.Status)
		if r.Status == model.StatusComplete {
			state = "OK"
		}
		style := styleMuted()
		switch {
		case r.Status == model.StatusError:
			style = styleErr()
		case i == selected && focused:
			style = styleAccent()
		case r.Status == model.StatusComplete:
			style = styleOK()
		}
		rows = append(rows, tableRow{[]string{state, r.Repository.Name, fmt.Sprintf("%.2f", r.TotalScore), fmt.Sprint(r.ChangeCount)}, style})
	}
	return renderTable(columns, rows, selected, width, "(no repositories)")
}

func renderChangeList(changes []model.ChangeUnit, selected int, focused bool, width int) string {
	columns := []tableColumn{{"Commit", 7, false}}
	if width >= 58 {
		columns = append(columns, tableColumn{"Date", 5, false}, tableColumn{"Type", 8, false})
	}
	columns = append(columns, tableColumn{"Intensity", 9, true})
	if width >= 86 {
		columns = append(columns, tableColumn{"Added", 6, true}, tableColumn{"Deleted", 7, true})
	}
	used := 2 + len(columns) // selection marker and gaps, including the message column
	for _, col := range columns {
		used += col.width
	}
	columns = append(columns, tableColumn{"Message", max(width-used, 7), false})
	rows := make([]tableRow, 0, len(changes))
	for i, c := range changes {
		hash := c.Hash
		if len(hash) > 7 {
			hash = hash[:7]
		}
		values := []string{hash}
		if width >= 58 {
			day := ""
			if !c.Timestamp.IsZero() {
				day = c.Timestamp.UTC().Format("01-02")
			}
			values = append(values, day, string(c.Type))
		}
		values = append(values, fmt.Sprintf("%.2f", c.Score.Final))
		if width >= 86 {
			values = append(values, fmt.Sprint(c.AddedLines), fmt.Sprint(c.DeletedLines))
		}
		values = append(values, c.Message)
		style := lipgloss.NewStyle()
		if i == selected && focused {
			style = styleAccent()
		}
		rows = append(rows, tableRow{values, style})
	}
	return renderTable(columns, rows, selected, width, "(no changes)")
}

func renderProfileBars(dist map[model.ChangeType]int, width int) string {
	order := []struct {
		label string
		key   model.ChangeType
	}{
		{"Feature", model.ChangeTypeFeat},
		{"Fix", model.ChangeTypeFix},
		{"Refactor", model.ChangeTypeRefactor},
		{"Performance", model.ChangeTypePerf},
		{"Test", model.ChangeTypeTest},
		{"Docs", model.ChangeTypeDocs},
		{"Chore", model.ChangeTypeChore},
	}
	maxN := 1
	for _, o := range order {
		if dist[o.key] > maxN {
			maxN = dist[o.key]
		}
	}
	barW := width - 22
	if barW < 8 {
		barW = 8
	}
	var b strings.Builder
	for _, o := range order {
		n := 0
		if dist != nil {
			n = dist[o.key]
		}
		filled := n * barW / maxN
		if n > 0 && filled == 0 {
			filled = 1
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
		line := fmt.Sprintf("%-12s %s  %d", o.label, bar, n)
		b.WriteString(line + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 1 {
		return string(runes[:max(n, 0)])
	}
	return string(runes[:n-1]) + "…"
}

func timelineFromChanges(changes []model.ChangeUnit) []model.DailyBucket {
	return aggregate.Timeline(changes)
}

// Keep the selected row visible without allocating state for each panel.
func listWindow(total, selected, rows int) (int, int) {
	rows = max(rows, 1)
	start := max(clampIndex(selected, total)-rows+1, 0)
	return start, min(start+rows, total)
}

func (m Model) renderChanges(rows, width int) string {
	changes := m.filteredChanges()
	start, end := listWindow(len(changes), m.SelectedChange, max(rows-1, 1))
	return fitLines(renderChangeList(changes[start:end], m.SelectedChange-start, true, width), width)
}

func (m Model) renderDevelopers(rows, width int) string {
	devs := m.filteredDevelopers()
	start, end := listWindow(len(devs), m.SelectedDeveloper, max(rows-1, 1))
	return fitLines(renderDeveloperList(devs[start:end], m.SelectedDeveloper-start, m.Focus == FocusDevelopers, width), width)
}

func (m Model) renderRepositories(rows, width int) string {
	repos := m.filteredRepositories()
	start, end := listWindow(len(repos), m.SelectedRepository, max(rows-1, 1))
	return fitLines(renderRepositoryList(repos[start:end], m.SelectedRepository-start, m.Focus == FocusRepositories, width), width)
}

func fitLines(content string, width int) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, max(width, 1), "…")
	}
	return strings.Join(lines, "\n")
}

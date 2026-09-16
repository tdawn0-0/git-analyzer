package tui

import (
	"fmt"
	"strings"

	"github.com/tdawn0-0/git-analyzer/internal/aggregate"
	"github.com/tdawn0-0/git-analyzer/internal/model"
)

func renderDeveloperList(devs []model.DeveloperStats, selected int, focused bool) string {
	if len(devs) == 0 {
		return styleMuted().Render("(no developers)")
	}
	var b strings.Builder
	for i, d := range devs {
		cursor := "  "
		if i == selected {
			cursor = "> "
		}
		line := fmt.Sprintf("%s%-16s  %6.2f  c=%-4d  +%-5d/-%-5d",
			cursor, truncate(d.Developer, 16), d.TotalScore, d.ChangeCount, d.AddedLines, d.DeletedLines)
		if i == selected && focused {
			b.WriteString(styleAccent().Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderRepositoryList(repos []model.RepositoryStats, selected int, focused bool) string {
	if len(repos) == 0 {
		return styleMuted().Render("(no repositories)")
	}
	var b strings.Builder
	for i, r := range repos {
		cursor := "  "
		if i == selected {
			cursor = "> "
		}
		glyph := statusGlyph(r.Status)
		line := fmt.Sprintf("%s%s %-14s  %6.2f  c=%-4d",
			cursor, glyph, truncate(r.Repository.Name, 14), r.TotalScore, r.ChangeCount)
		switch {
		case r.Status == model.StatusError:
			b.WriteString(styleErr().Render(line) + "\n")
		case i == selected && focused:
			b.WriteString(styleAccent().Render(line) + "\n")
		case r.Status == model.StatusComplete:
			b.WriteString(styleOK().Render(line) + "\n")
		default:
			b.WriteString(styleMuted().Render(line) + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderChangeList(changes []model.ChangeUnit, selected int, focused bool) string {
	if len(changes) == 0 {
		return styleMuted().Render("(no changes)")
	}
	var b strings.Builder
	for i, c := range changes {
		cursor := "  "
		if i == selected {
			cursor = "> "
		}
		hash := c.Hash
		if len(hash) > 7 {
			hash = hash[:7]
		}
		day := ""
		if !c.Timestamp.IsZero() {
			day = c.Timestamp.UTC().Format("01-02")
		}
		line := fmt.Sprintf("%s%s  %s  %-8s  %6.2f  +%-4d/-%-4d  %s",
			cursor, hash, day, c.Type, c.Score.Final, c.AddedLines, c.DeletedLines, truncate(c.Message, 40))
		if i == selected && focused {
			b.WriteString(styleAccent().Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
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
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

func timelineFromChanges(changes []model.ChangeUnit) []model.DailyBucket {
	return aggregate.Timeline(changes)
}

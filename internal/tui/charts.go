package tui

import (
	"fmt"
	"strings"

	"github.com/NimbleMarkets/ntcharts/canvas/runes"
	"github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	"github.com/charmbracelet/lipgloss"

	"github.com/tdawn0-0/git-analyzer/internal/model"
)

func renderTrendChart(buckets []model.DailyBucket, mode TrendMode, width, height int) string {
	if width < 20 {
		width = 20
	}
	if height < 5 {
		height = 5
	}
	if len(buckets) == 0 {
		return styleMuted().Render("(no daily activity yet)")
	}

	chart := timeserieslinechart.New(width, height)
	chart.AxisStyle = lipgloss.NewStyle().Foreground(colorMuted)
	chart.LabelStyle = lipgloss.NewStyle().Foreground(colorMuted)
	chart.XLabelFormatter = timeserieslinechart.DateTimeLabelFormatter()
	chart.SetLineStyle(runes.ThinLineStyle)

	minT, maxT := buckets[0].Date, buckets[len(buckets)-1].Date
	maxY := 1.0
	for _, b := range buckets {
		if mode == TrendActivity {
			if b.TotalScore > maxY {
				maxY = b.TotalScore
			}
		} else {
			if float64(b.AddedLines) > maxY {
				maxY = float64(b.AddedLines)
			}
			if float64(b.DeletedLines) > maxY {
				maxY = float64(b.DeletedLines)
			}
		}
	}
	chart.SetTimeRange(minT, maxT)
	chart.SetViewTimeRange(minT, maxT)
	chart.SetYRange(0, maxY*1.1+0.01)
	chart.SetViewYRange(0, maxY*1.1+0.01)

	if mode == TrendActivity {
		chart.SetStyle(lipgloss.NewStyle().Foreground(colorAccent))
		for _, b := range buckets {
			chart.Push(timeserieslinechart.TimePoint{Time: b.Date, Value: b.TotalScore})
		}
		chart.Draw()
	} else {
		chart.SetDataSetStyle("added", lipgloss.NewStyle().Foreground(colorAdded))
		chart.SetDataSetStyle("deleted", lipgloss.NewStyle().Foreground(colorDeleted))
		for _, b := range buckets {
			chart.PushDataSet("added", timeserieslinechart.TimePoint{Time: b.Date, Value: float64(b.AddedLines)})
			chart.PushDataSet("deleted", timeserieslinechart.TimePoint{Time: b.Date, Value: float64(b.DeletedLines)})
		}
		chart.DrawAll()
	}

	legend := ""
	if mode == TrendCodeChange {
		legend = styleOK().Render("+ added") + "  " + styleErr().Render("− deleted") + "\n"
	} else {
		first, last := buckets[0].Date.Format("Jan 02"), buckets[len(buckets)-1].Date.Format("Jan 02")
		legend = styleMuted().Render(fmt.Sprintf("%s → %s  (%d days)", first, last, len(buckets))) + "\n"
	}
	return legend + chart.View()
}

// sparklineFallback unused but kept for narrow terminals if ntcharts fails — simple ASCII.
func sparklineFallback(values []float64, width int) string {
	if len(values) == 0 {
		return ""
	}
	blocks := []rune("▁▂▃▄▅▆▇█")
	maxV := 0.0
	for _, v := range values {
		if v > maxV {
			maxV = v
		}
	}
	if maxV == 0 {
		return strings.Repeat("▁", min(width, len(values)))
	}
	step := 1
	if len(values) > width && width > 0 {
		step = len(values) / width
		if step < 1 {
			step = 1
		}
	}
	var b strings.Builder
	for i := 0; i < len(values); i += step {
		idx := int(values[i] / maxV * float64(len(blocks)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		b.WriteRune(blocks[idx])
	}
	return b.String()
}

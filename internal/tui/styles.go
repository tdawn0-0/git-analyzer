package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	colorMuted   = lipgloss.Color("245")
	colorAccent  = lipgloss.Color("39")  // blue — Activity, not "winner"
	colorOK      = lipgloss.Color("78")
	colorErr     = lipgloss.Color("203")
	colorBorder  = lipgloss.Color("238")
	colorTitle   = lipgloss.Color("255")
	colorFocus   = lipgloss.Color("81")
	colorHeader  = lipgloss.Color("250")
	colorAdded   = lipgloss.Color("114")
	colorDeleted = lipgloss.Color("210")
)

func styleHeader() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(colorHeader)
}

func styleTitle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(colorTitle)
}

func styleMuted() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colorMuted)
}

func styleAccent() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colorAccent)
}

func styleOK() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colorOK)
}

func styleErr() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colorErr)
}

func stylePanel(focused bool, width, height int) lipgloss.Style {
	border := lipgloss.NormalBorder()
	fg := colorBorder
	if focused {
		fg = colorFocus
	}
	s := lipgloss.NewStyle().
		Border(border).
		BorderForeground(fg).
		Width(width).
		Height(height).
		Padding(0, 1)
	return s
}

func styleStatusBar() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colorMuted)
}

func styleHelp() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(1, 2)
}

// layoutMode from terminal width.
type layoutMode int

const (
	layoutFour layoutMode = iota
	layoutDual
	layoutStack
)

func layoutFor(width int) layoutMode {
	switch {
	case width >= 120:
		return layoutFour
	case width >= 80:
		return layoutDual
	default:
		return layoutStack
	}
}

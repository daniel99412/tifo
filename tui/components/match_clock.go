package components

import (
	"tifo/internal/domain"

	"github.com/charmbracelet/lipgloss"
)

type MatchClockData struct {
	Minute       string
	WaterBreak   bool
	IsLive       bool
	IsFinished   bool
	Period       int
	HasPenalties bool
}

// RenderClock returns the full clock string styled for the detail view.
func RenderClock(d MatchClockData) string {
	if d.Minute != "" {
		s := d.Minute
		if d.WaterBreak {
			s += " H2O"
		}
		return clockLiveIndicator.Render("●") + " " + clockLiveMinute.Render(s)
	}
	return clockStatus.Render("programado")
}

// RenderClockWithLabel is like RenderClock but allows a custom status label.
func RenderClockWithLabel(d MatchClockData, label string) string {
	if d.Minute != "" {
		s := d.Minute
		if d.WaterBreak {
			s += " H2O"
		}
		return clockLiveIndicator.Render("●") + " " + clockLiveMinute.Render(s)
	}
	if label == "" {
		label = "programado"
	}
	return clockStatus.Render(label)
}

// ClockParts holds individual styled pieces for assembling a list row.
type ClockParts struct {
	Indicator string // ● (styled) or blank
	Time      string // minute (styled) or kickoff time (styled)
	Status    string // FT / HT / ET / PEN (styled) or blank
}

// BuildClockParts returns styled components for the match list row.
func BuildClockParts(d MatchClockData, kickoffTime string) ClockParts {
	var p ClockParts

	if d.IsLive {
		p.Indicator = clockLiveIndicator.Render("●")
		minute := d.Minute
		if minute == "" {
			minute = "--:--"
		}
		if d.WaterBreak {
			minute += " H2O"
		}
		p.Time = clockLiveMinute.Render(minute)
	} else {
		p.Time = clockStatus.Render(kickoffTime)
	}

	if d.HasPenalties {
		p.Status = clockScore.Render("PEN")
	} else if d.IsFinished {
		if d.Period >= domain.PeriodETFirstHalf {
			p.Status = clockScore.Render("ET")
		} else {
			p.Status = clockScore.Render("FT")
		}
	} else if d.IsLive && (d.Minute == "HT" || d.Minute == "Descanso") {
		p.Status = clockScore.Render("HT")
	}

	return p
}

// Shared styles for the clock across both views.
var (
	clockLiveIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	clockLiveMinute    = lipgloss.NewStyle().Foreground(lipgloss.Color("215"))
	clockStatus        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	clockScore         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
)

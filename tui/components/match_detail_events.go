package components

import (
	"fmt"
	"tifo/internal/domain"

	"github.com/charmbracelet/lipgloss"
)

func (md *MatchDetail) renderEvents(width, height int) string {
	if md.Details == nil || len(md.Details.Events.Items) == 0 {
		return mdInfoStyle.Render("sin eventos")
	}

	periods := md.availablePeriods()

	var bodyLines []string

	if len(periods) > 1 {
		bodyLines = append(bodyLines, md.renderPeriodSubtabs(periods, width))
		bodyLines = append(bodyLines, "")
	}

	var eventLines []string
	selPeriod := md.Details.SelectedPeriod
	for _, ev := range md.Details.Events.Items {
		if selPeriod > 0 && selPeriod != domain.PeriodAll && ev.Period != selPeriod {
			continue
		}
		timeCell := mdTimeStyle.Render(ev.Minute)
		typeCell := md.eventTypeCell(ev)
		descCell := md.eventDesc(ev)
		row := lipgloss.JoinHorizontal(lipgloss.Top, timeCell, typeCell, descCell)
		eventLines = append(eventLines, row)
	}
	if len(eventLines) == 0 {
		eventLines = append(eventLines, mdInfoStyle.Render("sin eventos en este periodo"))
	}

	eventColW := width - 24
	if eventColW < 30 {
		eventColW = 30
	}
	legendW := 22

	headerH := len(bodyLines)
	scrollH := height - headerH
	if scrollH < 1 {
		scrollH = 1
	}
	eventsBody := lipgloss.JoinVertical(lipgloss.Top, eventLines...)
	scrollPart := md.applyScroll(eventsBody, eventColW, scrollH)

	bodyLines = append(bodyLines, scrollPart)
	left := lipgloss.NewStyle().Width(eventColW).Render(lipgloss.JoinVertical(lipgloss.Top, bodyLines...))

	var legLines []string
	legLines = append(legLines, mdSectionHeader.Render("Simbología"))
	for _, s := range md.symbolLegend() {
		legLines = append(legLines, lipgloss.NewStyle().Width(legendW).Foreground(lipgloss.Color("240")).Render(s))
	}
	right := lipgloss.JoinVertical(lipgloss.Top, legLines...)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (md MatchDetail) symbolLegend() []string {
	legend := []struct{ key, label string }{
		{"GOL", "Gol"}, {"CAR", "Tarjeta"}, {"SUB", "Sustitución"},
		{"SHO", "Tiro"}, {"PEN", "Penal"}, {"PAR", "Penal atajado"},
		{"VAR", "VAR"}, {"REV", "Revisión"}, {"AT", "Añadido"},
		{"HT", "Descanso"}, {"FT", "Final"}, {"KO", "Inicio"},
		{"S2", "2do tiempo"}, {"H2O", "Hidratación"}, {"LES", "Lesión"},
		{"PAU", "Pausa"}, {"CONT", "Continúa"},
		{"AET", "Tiempo extra"}, {"PEN", "Penales"},
	}
	var out []string
	for _, l := range legend {
		sym := mdTypeStyle.Render(l.key)
		out = append(out, sym+"  "+l.label)
	}
	return out
}

func (md MatchDetail) eventTypeCell(ev EventItem) string {
	switch ev.EventType {
	case "Goal":
		return mdTypeStyle.Render("GOL")
	case "Card":
		return mdTypeStyle.Render("CAR")
	case "Substitution":
		return mdTypeStyle.Render("SUB")
	case "Half":
		str := "HT"
		if ev.HalfStr == "FT" {
			str = "FT"
		}
		return mdTypeStyle.Render(str)
	case "FT":
		return mdTypeStyle.Render("FT")
	case "AddedTime":
		return mdTypeStyle.Render("AT")
	case "PenaltyAwarded":
		return mdTypeStyle.Render("PEN")
	case "MissedPenalty":
		return mdTypeStyle.Render("PEN")
	case "SavedPenalty":
		return mdTypeStyle.Render("PAR")
	case "OwnGoal":
		return mdTypeStyle.Render("GOL")
	case "InjuryTime":
		return mdTypeStyle.Render("LES")
	case "Yellow":
		return mdTypeStyle.Render("CAR")
	case "Red":
		return mdTypeStyle.Render("CAR")
	case "InternationalDuty":
		return mdTypeStyle.Render("SEL")
	case "WaterBreak", "CoolingBreak", "DrinkBreak":
		return mdTypeStyle.Render("H2O")
	case "VAR":
		return mdTypeStyle.Render("VAR")
	case "VideoReview":
		return mdTypeStyle.Render("REV")
	case "Shot":
		return mdTypeStyle.Render("SHO")
	case "KO":
		return mdTypeStyle.Render("KO")
	case "S2":
		return mdTypeStyle.Render("S2")
	case "HT":
		return mdTypeStyle.Render("HT")
	case "Pausa":
		return mdTypeStyle.Render("PAU")
	case "Continúa":
		return mdTypeStyle.Render("CONT")
	case "AET":
		return mdTypeStyle.Render("AET")
	case "AET_S2":
		return mdTypeStyle.Render("AET")
	case "Penales":
		return mdTypeStyle.Render("PEN")
	default:
		short := ev.EventType
		if len(short) > 5 {
			short = short[:5]
		}
		return mdTypeStyle.Render(short)
	}
}

func (md MatchDetail) eventDesc(ev EventItem) string {
	var parts []string

	switch ev.EventType {
	case "Goal":
		if ev.OwnGoal {
			parts = append(parts, mdRedStyle.Render("AG"))
		} else {
			parts = append(parts, mdGoalStyle.Render("G"))
		}
		parts = append(parts, " [")
		parts = append(parts, fmt.Sprintf("%d-%d", ev.HomeScore, ev.AwayScore))
		parts = append(parts, "] ")
		parts = append(parts, ev.Player)
		if ev.GoalDesc != "" {
			parts = append(parts, " (")
			parts = append(parts, ev.GoalDesc)
			parts = append(parts, ")")
		}

	case "PenaltyAwarded":
		parts = append(parts, mdRedStyle.Render("P"))
		parts = append(parts, " Penal")

	case "MissedPenalty":
		parts = append(parts, fmt.Sprintf("[%d-%d] Penal fallado — ", ev.HomeScore, ev.AwayScore))
		parts = append(parts, ev.Player)

	case "SavedPenalty":
		parts = append(parts, fmt.Sprintf("[%d-%d] Penal atajado — ", ev.HomeScore, ev.AwayScore))
		parts = append(parts, ev.Player)

	case "OwnGoal":
		parts = append(parts, mdRedStyle.Render("AG"))
		parts = append(parts, fmt.Sprintf(" [%d-%d] ", ev.HomeScore, ev.AwayScore))
		parts = append(parts, ev.Player)

	case "Card":
		if ev.CardType == "Red" || ev.CardType == "red" {
			parts = append(parts, mdRedStyle.Render("R"))
		} else {
			parts = append(parts, mdYellowStyle.Render("!"))
		}
		if ev.Player != "" {
			parts = append(parts, " ")
			parts = append(parts, ev.Player)
		}

	case "Yellow":
		parts = append(parts, mdYellowStyle.Render("!"))
		if ev.Player != "" {
			parts = append(parts, " ")
			parts = append(parts, ev.Player)
		}

	case "Red":
		parts = append(parts, mdRedStyle.Render("R"))
		if ev.Player != "" {
			parts = append(parts, " ")
			parts = append(parts, ev.Player)
		}

	case "Substitution":
		parts = append(parts, mdSubOutStyle.Render("↓"))
		parts = append(parts, " ")
		parts = append(parts, ev.SubOut)
		parts = append(parts, "  ")
		parts = append(parts, mdSubInStyle.Render("↑"))
		parts = append(parts, " ")
		parts = append(parts, ev.SubIn)

	case "Half":
		if ev.HalfStr == "FT" {
			parts = append(parts, "Final del partido")
		} else {
			parts = append(parts, "Descanso")
		}

	case "FT":
		parts = append(parts, "Final del partido")

	case "AddedTime":
		if ev.AddedTime > 0 {
			parts = append(parts, fmt.Sprintf("%d' añadido", ev.AddedTime))
		} else {
			parts = append(parts, "Tiempo añadido")
		}

	case "InjuryTime":
		parts = append(parts, "Lesión: ")
		parts = append(parts, ev.Player)

	case "InternationalDuty":
		parts = append(parts, "Fecha FIFA: ")
		parts = append(parts, ev.Player)

	case "WaterBreak", "CoolingBreak", "DrinkBreak":
		parts = append(parts, "Pausa de hidratación")

	case "VAR":
		parts = append(parts, mdYellowStyle.Render("VAR"))
		if ev.Detail != "" {
			parts = append(parts, " — ")
			parts = append(parts, ev.Detail)
		}

	case "VideoReview":
		parts = append(parts, mdYellowStyle.Render("VAR"))
		if ev.Detail != "" {
			parts = append(parts, " — ")
			parts = append(parts, ev.Detail)
		}

	case "Shot":
		switch ev.ShotDesc {
		case "gol":
			parts = append(parts, mdShotGoalStyle.Render("S"))
		case "atajado":
			parts = append(parts, mdShotStyle.Render("S"))
		default:
			parts = append(parts, mdShotMissStyle.Render("S"))
		}
		parts = append(parts, " ")
		parts = append(parts, ev.Player)
		if ev.ShotDesc != "" {
			parts = append(parts, " (")
			parts = append(parts, ev.ShotDesc)
			parts = append(parts, ")")
		}

	case "KO":
		if ev.Detail != "" {
			parts = append(parts, ev.Detail)
		} else {
			parts = append(parts, "Inicio del partido")
		}

	case "HT":
		if ev.Detail != "" {
			parts = append(parts, ev.Detail)
		} else {
			parts = append(parts, "Descanso")
		}

	case "S2":
		if ev.Detail != "" {
			parts = append(parts, ev.Detail)
		} else {
			parts = append(parts, "Inicio 2do tiempo")
		}

	case "Pausa":
		parts = append(parts, mdYellowStyle.Render("⏸"))
		parts = append(parts, " ")
		if ev.Detail != "" {
			parts = append(parts, ev.Detail)
		} else {
			parts = append(parts, "Pausa")
		}

	case "Continúa":
		parts = append(parts, mdGoalStyle.Render("▶"))

	default:
		if ev.Player != "" {
			parts = append(parts, ev.Player)
		} else if ev.Detail != "" {
			parts = append(parts, ev.Detail)
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

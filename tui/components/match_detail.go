package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	mdTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Align(lipgloss.Center)

	mdScoreStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("39")).
			Padding(0, 1).
			Align(lipgloss.Center)

	mdTeamStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Align(lipgloss.Center)

	mdInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	mdBackStyle = lipgloss.NewStyle().
			PaddingTop(1).
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Align(lipgloss.Center)

	mdSectionHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("39")).
				PaddingBottom(1)

	mdStatHome = lipgloss.NewStyle().
			Width(8).
			Align(lipgloss.Right).
			Foreground(lipgloss.Color("255"))

	mdStatLabel = lipgloss.NewStyle().
			Width(20).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("240"))

	mdStatAway = lipgloss.NewStyle().
			Width(8).
			Align(lipgloss.Left).
			Foreground(lipgloss.Color("255"))

	mdPlayerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255"))

	mdEventStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	mdInjuryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	mdGoalStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("46"))

	mdYellowStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("220"))

	mdRedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196"))

	mdSubOutStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	mdSubInStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46"))

	mdShotStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214"))

	mdShotMissStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196"))

	mdShotGoalStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("46"))

	mdTimeStyle = lipgloss.NewStyle().
			Width(6).
			Foreground(lipgloss.Color("240"))

	mdTypeStyle = lipgloss.NewStyle().
			Width(5).
			Foreground(lipgloss.Color("240"))
)

type MatchDetail struct {
	HomeName    string
	AwayName    string
	Score       string
	Status      string
	DateTime    string
	MatchID     string
	HomeScore   string
	AwayScore   string
	Tabs        Tabs
	Details     *MatchDetailData
	ScrollOff   int
	Error       string
}

type MatchDetailData struct {
	Stats     []StatCategory
	Lineup    LineupData
	Events    EventData
	H2H       H2HData
	Injuries  InjuriesData
}

type StatCategory struct {
	Title string
	Stats []StatRow
}

type StatRow struct {
	Label string
	Home  string
	Away  string
}

type LineupData struct {
	HomeFormation string
	AwayFormation string
	HomeStarters  []PlayerLineup
	HomeSubs      []PlayerLineup
	AwayStarters  []PlayerLineup
	AwaySubs      []PlayerLineup
	HomeCoach     string
	AwayCoach     string
}

type PlayerLineup struct {
	Name    string
	Number  string
	PosName string
}

type EventData struct {
	Items     []EventItem
	ExtraInfo *MatchExtraInfo
}

type MatchExtraInfo struct {
	Venue       string
	Attendance  int
	Referee     string
	Weather     string
	Broadcasts  []string
	ESPNStatus  string
	HomeColor   string
	AwayColor   string
	HomeAltColor string
	AwayAltColor string
}

type EventItem struct {
	Minute       string
	EventType    string
	Player       string
	Team         string
	HomeScore    int
	AwayScore    int
	CardType     string
	IsHome       bool
	Detail       string
	SubOut       string
	SubIn        string
	AddedTime    int
	GoalDesc     string
	HalfStr      string
	OwnGoal      bool
	ShotDesc     string
	SortTime     int
	SortOverload int
}

type H2HData struct {
	HomeWins  int
	Draws     int
	AwayWins  int
}

type InjuriesData struct {
	Home []InjuryPlayer
	Away []InjuryPlayer
}

type InjuryPlayer struct {
	Name   string
	Type   string
	Return string
}

func NewMatchDetail(home, away, score, status, dateTime, matchID string) MatchDetail {
	homeScore := ""
	awayScore := ""
	if score != "" {
		parts := strings.Split(score, "-")
		if len(parts) == 2 {
			homeScore = strings.TrimSpace(parts[0])
			awayScore = strings.TrimSpace(parts[1])
		}
	}
	return MatchDetail{
		HomeName:  home,
		AwayName:  away,
		Score:     score,
		Status:    status,
		DateTime:  dateTime,
		MatchID:   matchID,
		HomeScore: homeScore,
		AwayScore: awayScore,
		Tabs:      NewTabs([]string{"Alineaciones", "Eventos", "Estadísticas", "H2H", "Lesiones"}),
	}
}

func (md *MatchDetail) ScrollUp() {
	if md.ScrollOff > 0 {
		md.ScrollOff--
	}
}

func (md *MatchDetail) ScrollDown() {
	md.ScrollOff++
}

func (md *MatchDetail) SetError(err string) {
	md.Error = err
}

func (md *MatchDetail) Render(width, height int) string {
	if width < 5 || height < 5 {
		return ""
	}

	pad := 2
	width -= pad * 2
	if width < 10 {
		width = 10
	}

	var lines []string

	// Title always visible at top
	lines = append(lines, mdTitleStyle.Render("Detalle del Partido"))
	lines = append(lines, "")
	scoreCell := ""
	if md.HomeScore != "" && md.AwayScore != "" {
		scoreCell = mdScoreStyle.Render(fmt.Sprintf(" %s - %s ", md.HomeScore, md.AwayScore))
	} else if md.Score != "" {
		scoreCell = mdScoreStyle.Render(fmt.Sprintf(" %s ", md.Score))
	} else {
		scoreCell = mdScoreStyle.Render(" vs ")
	}

	homeHeaderColor := lipgloss.Color("255")
	awayHeaderColor := lipgloss.Color("255")
	if md.Details != nil && md.Details.Events.ExtraInfo != nil {
		if md.Details.Events.ExtraInfo.HomeColor != "" {
			c := md.Details.Events.ExtraInfo.HomeColor
			if !strings.HasPrefix(c, "#") { c = "#" + c }
			homeHeaderColor = lipgloss.Color(c)
		}
		if md.Details.Events.ExtraInfo.AwayColor != "" {
			c := md.Details.Events.ExtraInfo.AwayColor
			if !strings.HasPrefix(c, "#") { c = "#" + c }
			awayHeaderColor = lipgloss.Color(c)
		}
	}

	bulletHome := lipgloss.NewStyle().Foreground(homeHeaderColor).Render("●")
	bulletAway := lipgloss.NewStyle().Foreground(awayHeaderColor).Render("●")

	homeTeamStyle := lipgloss.NewStyle().Bold(true).Foreground(homeHeaderColor)
	awayTeamStyle := lipgloss.NewStyle().Bold(true).Foreground(awayHeaderColor)

	homeCol := homeTeamStyle.Render(bulletHome + " " + md.HomeName)
	awayCol := awayTeamStyle.Render(md.AwayName + " " + bulletAway)

	matchup := lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(
		lipgloss.JoinHorizontal(lipgloss.Center,
			homeCol, "  ", scoreCell, "  ", awayCol,
		),
	)
	lines = append(lines, matchup)

	statusLabel := md.Status
	if statusLabel == "" {
		statusLabel = "programado"
	}
	lines = append(lines, mdInfoStyle.Render(statusLabel))
	lines = append(lines, mdInfoStyle.Render(md.DateTime))

	// Extra info (venue, attendance, referee, TV)
	if md.Details != nil && md.Details.Events.ExtraInfo != nil {
		info := md.Details.Events.ExtraInfo
		if info.ESPNStatus != "" {
			lines = append(lines, mdInfoStyle.Render(info.ESPNStatus))
		}
		if info.Venue != "" {
			lines = append(lines, mdInfoStyle.Render(fmt.Sprintf("Venue: %s", info.Venue)))
		}
		if info.Attendance > 0 {
			lines = append(lines, mdInfoStyle.Render(fmt.Sprintf("Asistencia: %d", info.Attendance)))
		}
		if info.Referee != "" {
			lines = append(lines, mdInfoStyle.Render(fmt.Sprintf("Árbitro: %s", info.Referee)))
		}
		if len(info.Broadcasts) > 0 {
			lines = append(lines, mdInfoStyle.Render(fmt.Sprintf("TV: %s", strings.Join(info.Broadcasts, ", "))))
		}
	}
	lines = append(lines, "")

	// Tabs
	lines = append(lines, md.Tabs.Render(width))
	lines = append(lines, "")

	// Tab content
	contentH := height - len(lines) - 2
	content := md.renderTabContent(width, contentH)
	lines = append(lines, content)

	lines = append(lines, mdBackStyle.Render("←/→ tabs · u/d scroll · esc volver"))

	body := lipgloss.JoinVertical(lipgloss.Top, lines...)
	return lipgloss.NewStyle().PaddingLeft(pad).PaddingRight(pad).Render(body)
}

func (md *MatchDetail) renderTabContent(width, height int) string {
	if md.Error != "" {
		return mdInfoStyle.Render(fmt.Sprintf("error: %s", md.Error))
	}
	if md.Details == nil {
		return mdInfoStyle.Render("cargando detalles...")
	}

	switch md.Tabs.Active() {
	case 0:
		return md.renderLineup(width, height)
	case 1:
		return md.renderEvents(width, height)
	case 2:
		return md.renderStats(width, height)
	case 3:
		return md.renderH2H(width, height)
	case 4:
		return md.renderInjuries(width, height)
	}
	return ""
}

func parseFloatVal(s string) (float64, bool) {
	if s == "" || s == "<nil>" || s == "-" {
		return 0, false
	}
	var v float64
	if _, err := fmt.Sscanf(s, "%f", &v); err == nil {
		return v, true
	}
	return 0, false
}

func statBar(leftVal, rightVal string, barW int) string {
	lv, lOK := parseFloatVal(leftVal)
	rv, rOK := parseFloatVal(rightVal)

	if barW < 6 {
		barW = 6
	}

	if !lOK && !rOK {
		return strings.Repeat("░", barW)
	}

	max := lv
	if rv > max {
		max = rv
	}
	if max == 0 {
		max = 1
	}

	fill := int(lv / max * float64(barW))
	if fill < 0 {
		fill = 0
	}
	if fill > barW {
		fill = barW
	}

	return strings.Repeat("█", fill) + strings.Repeat("░", barW-fill)
}

func displayVal(s string) string {
	if s == "" || s == "<nil>" {
		return "—"
	}
	return s
}

func (md *MatchDetail) renderStats(width, height int) string {
	if md.Details == nil || len(md.Details.Stats) == 0 {
		return mdInfoStyle.Render("sin estadísticas")
	}

	homeColor := lipgloss.Color("255")
	awayColor := lipgloss.Color("255")
	if md.Details.Events.ExtraInfo != nil {
		if md.Details.Events.ExtraInfo.HomeColor != "" {
			c := md.Details.Events.ExtraInfo.HomeColor
			if !strings.HasPrefix(c, "#") { c = "#" + c }
			homeColor = lipgloss.Color(c)
		}
		if md.Details.Events.ExtraInfo.AwayColor != "" {
			c := md.Details.Events.ExtraInfo.AwayColor
			if !strings.HasPrefix(c, "#") { c = "#" + c }
			awayColor = lipgloss.Color(c)
		}
	}

	maxStatW := (width - 30)
	if maxStatW < 10 {
		maxStatW = 10
	}
	if maxStatW > 40 {
		maxStatW = 40
	}
	barW := maxStatW

	homeNumStyle := lipgloss.NewStyle().Width(6).Align(lipgloss.Right).Bold(true).Foreground(homeColor)
	awayNumStyle := lipgloss.NewStyle().Width(6).Align(lipgloss.Left).Bold(true).Foreground(awayColor)

	var lines []string
	for _, cat := range md.Details.Stats {
		lines = append(lines, mdSectionHeader.Render(cat.Title))
		for _, stat := range cat.Stats {
			labelWidth := barW + 12
			lines = append(lines, mdInfoStyle.Width(labelWidth).Align(lipgloss.Center).Render(stat.Label))
			bar := statBar(stat.Home, stat.Away, barW)
			barColor := lipgloss.Color("240")
			homeVal := displayVal(stat.Home)
			awayVal := displayVal(stat.Away)
			// Color the bar with the higher team's color
			if h, hOK := parseFloatVal(stat.Home); hOK {
				if a, aOK := parseFloatVal(stat.Away); aOK {
					if h >= a {
						barColor = homeColor
					} else {
						barColor = awayColor
					}
				} else {
					barColor = homeColor
				}
			} else if _, aOK := parseFloatVal(stat.Away); aOK {
				barColor = awayColor
			}
			row := lipgloss.JoinHorizontal(lipgloss.Top,
				homeNumStyle.Render(homeVal),
				lipgloss.NewStyle().Width(barW).Align(lipgloss.Center).Foreground(barColor).Render(bar),
				awayNumStyle.Render(awayVal),
			)
			lines = append(lines, row)
		}
		lines = append(lines, "")
	}

	body := lipgloss.JoinVertical(lipgloss.Top, lines...)
	centered := lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(body)
	return md.applyScroll(centered, width, height)
}

func (md *MatchDetail) renderLineup(width, height int) string {
	if md.Details == nil {
		return mdInfoStyle.Render("sin alineaciones")
	}

	lu := md.Details.Lineup

	homeW := width * 20 / 100
	centerW := width * 60 / 100
	awayW := width * 20 / 100
	if homeW < 10 { homeW = 10 }
	if awayW < 10 { awayW = 10 }

	homeLines := md.teamColumn(lu.HomeFormation, lu.HomeCoach, lu.HomeStarters, lu.HomeSubs, homeW)
	awayLines := md.teamColumn(lu.AwayFormation, lu.AwayCoach, lu.AwayStarters, lu.AwaySubs, awayW)

	maxLen := len(homeLines)
	if len(awayLines) > maxLen {
		maxLen = len(awayLines)
	}

	var rendered []string
	for i := 0; i < maxLen; i++ {
		h := ""
		a := ""
		if i < len(homeLines) {
			h = homeLines[i]
		}
		if i < len(awayLines) {
			a = awayLines[i]
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top,
			h,
			lipgloss.NewStyle().Width(centerW).Render(""),
			a,
		)
		rendered = append(rendered, row)
	}

	body := lipgloss.JoinVertical(lipgloss.Top, rendered...)
	return md.applyScroll(body, width, height)
}

func (md MatchDetail) teamColumn(formation, coach string, starters, subs []PlayerLineup, colW int) []string {
	var lines []string

	lines = append(lines, lipgloss.NewStyle().Width(colW).Align(lipgloss.Center).
		Bold(true).Foreground(lipgloss.Color("39")).Render(fmt.Sprintf("[%s]", formation)))
	lines = append(lines, "")

	lines = append(lines, lipgloss.NewStyle().Width(colW).
		Foreground(lipgloss.Color("240")).Render(fmt.Sprintf("DT: %s", coach)))
	lines = append(lines, "")

	posW := 3
	for _, p := range starters {
		if len(p.PosName) > posW { posW = len(p.PosName) }
	}
	for _, p := range subs {
		if len(p.PosName) > posW { posW = len(p.PosName) }
	}
	nameW := colW - 8 - posW
	if nameW < 3 { nameW = 3 }

	lines = append(lines, lipgloss.NewStyle().Width(colW).Bold(true).
		Foreground(lipgloss.Color("255")).Render("Titulares"))
	lines = append(lines, "")

	for _, p := range starters {
		line := lipgloss.NewStyle().Width(colW).
			Foreground(lipgloss.Color("255")).Render(fmt.Sprintf("  %2s  %-*s  %-*s", p.Number, nameW, p.Name, posW, p.PosName))
		lines = append(lines, line)
	}

	lines = append(lines, "")
	sep := strings.Repeat("─", colW)
	lines = append(lines, lipgloss.NewStyle().Width(colW).
		Foreground(lipgloss.Color("236")).Render(sep))
	lines = append(lines, "")

	lines = append(lines, lipgloss.NewStyle().Width(colW).Bold(true).
		Foreground(lipgloss.Color("240")).Render("Suplentes"))
	lines = append(lines, "")

	for _, p := range subs {
		line := lipgloss.NewStyle().Width(colW).
			Foreground(lipgloss.Color("240")).Render(fmt.Sprintf("  %2s  %-*s  %-*s", p.Number, nameW, p.Name, posW, p.PosName))
		lines = append(lines, line)
	}

	return lines
}

func (md *MatchDetail) renderEvents(width, height int) string {
	if md.Details == nil || len(md.Details.Events.Items) == 0 {
		return mdInfoStyle.Render("sin eventos")
	}

	var eventLines []string
	for _, ev := range md.Details.Events.Items {
		timeCell := mdTimeStyle.Render(ev.Minute)
		typeCell := md.eventTypeCell(ev)
		descCell := md.eventDesc(ev)
		row := lipgloss.JoinHorizontal(lipgloss.Top, timeCell, typeCell, descCell)
		eventLines = append(eventLines, row)
	}

	eventColW := width - 24
	if eventColW < 30 {
		eventColW = 30
	}
	legendW := 22

	eventsBody := lipgloss.JoinVertical(lipgloss.Top, eventLines...)
	scrollPart := md.applyScroll(eventsBody, eventColW, height)

	// Left: events
	left := lipgloss.NewStyle().Width(eventColW).Render(scrollPart)

	// Right: symbol legend
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
		parts = append(parts, "Revisión VAR")

	case "VideoReview":
		parts = append(parts, "Revisión de video")

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
		parts = append(parts, " ")
		if ev.Detail != "" {
			parts = append(parts, ev.Detail)
		} else {
			parts = append(parts, "Se reanuda")
		}

	default:
		if ev.Player != "" {
			parts = append(parts, ev.Player)
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (md *MatchDetail) renderH2H(width, height int) string {
	if md.Details == nil {
		return mdInfoStyle.Render("sin datos H2H")
	}

	h2h := md.Details.H2H
	var lines []string

	lines = append(lines, mdSectionHeader.Render("Enfrentamientos directos"))
	summary := fmt.Sprintf("  %s: %d  ·  Empates: %d  ·  %s: %d",
		md.HomeName, h2h.HomeWins, h2h.Draws, md.AwayName, h2h.AwayWins)
	lines = append(lines, mdPlayerStyle.Render(summary))

	body := lipgloss.JoinVertical(lipgloss.Top, lines...)
	return md.applyScroll(body, width, height)
}

func (md *MatchDetail) renderInjuries(width, height int) string {
	if md.Details == nil {
		return mdInfoStyle.Render("sin datos de lesiones")
	}

	inf := md.Details.Injuries
	colW := (width - 4) / 2
	if colW < 15 {
		colW = 15
	}

	homeLines := md.injuryColumn(md.HomeName, inf.Home, colW)
	awayLines := md.injuryColumn(md.AwayName, inf.Away, colW)

	if len(homeLines) == 0 && len(awayLines) == 0 {
		return mdInfoStyle.Render("sin jugadores lesionados")
	}

	maxLen := len(homeLines)
	if len(awayLines) > maxLen {
		maxLen = len(awayLines)
	}

	var rendered []string
	for i := 0; i < maxLen; i++ {
		h := ""
		a := ""
		if i < len(homeLines) {
			h = homeLines[i]
		}
		if i < len(awayLines) {
			a = awayLines[i]
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, h, lipgloss.NewStyle().Width(4).Render(""), a)
		rendered = append(rendered, row)
	}

	body := lipgloss.JoinVertical(lipgloss.Top, rendered...)
	return md.applyScroll(body, width, height)
}

func (md MatchDetail) injuryColumn(teamName string, players []InjuryPlayer, colW int) []string {
	if len(players) == 0 {
		return nil
	}

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Width(colW).Bold(true).
		Foreground(lipgloss.Color("255")).Render(teamName))
	lines = append(lines, "")

	for _, p := range players {
		tag := p.Type
		if tag == "" {
			tag = "lesión"
		}
		ret := p.Return
		if ret == "" {
			ret = "?"
		}
		line := lipgloss.NewStyle().Width(colW).
			Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("  %s (%s) — %s", p.Name, tag, ret))
		lines = append(lines, line)
	}

	return lines
}

func (md *MatchDetail) applyScroll(body string, width, height int) string {
	allLines := strings.Split(body, "\n")
	total := len(allLines)

	maxScroll := total - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if md.ScrollOff > maxScroll {
		md.ScrollOff = maxScroll
	}

	start := md.ScrollOff
	end := start + height
	if end > total {
		end = total
	}

	visible := allLines[start:end]
	result := strings.Join(visible, "\n")

	remaining := height - len(visible)
	if remaining > 0 {
		result += "\n" + strings.Repeat("\n", remaining)
	}

	ScrollIndicator := ""
	if maxScroll > 0 {
		pct := 0
		if maxScroll > 0 {
			pct = md.ScrollOff * 100 / maxScroll
		}
		ScrollIndicator = mdInfoStyle.Render(fmt.Sprintf("  [%d%%]", pct))
	}

	return result + ScrollIndicator
}

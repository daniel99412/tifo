package components

import (
	"fmt"
	"strings"

	"tifo/internal/domain"

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

	mdLiveIndicatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))

	mdLiveMinuteStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("215"))
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
	PenScore    string
	PenShootout []domain.PenShot
	Minute      string
	WaterBreak  bool
	Tabs        Tabs
	Details     *MatchDetailData
	ScrollOff   int
	PlayerStats *PlayerStatsView
	Error       string
}

type MatchDetailData struct {
	StatsByPeriod map[int][]StatCategory
	Momentum      []MomentumPoint
	SelectedPeriod int
	Lineup   LineupData
	Events   EventData
	H2H      H2HData
	Injuries InjuriesData
	PlayerStats []domain.PlayerStatItem
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

type MomentumPoint struct {
	Minute int
	Value  int
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
	Name      string
	Number    string
	PosName   string
	IsCaptain bool
	SubOut    bool
	SubIn     bool
	CardType  string
}

type EventData struct {
	Items     []EventItem
	ExtraInfo *MatchExtraInfo
}

type MatchExtraInfo struct {
	Venue         string
	VenueCapacity int
	Surface       string
	Attendance    int
	Referee       string
	Weather       string
	Broadcasts    []string
	ESPNStatus    string
	HomeColor     string
	AwayColor     string
	HomeAltColor  string
	AwayAltColor  string
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
	Period       int
	SortTime     int
	SortOverload int
}

type H2HData struct {
	HomeWins   int
	Draws      int
	AwayWins   int
	Matches    []H2HMatchItem
	HomeForm   []H2HFormEvent
	AwayForm   []H2HFormEvent
	HomeRecord string
	AwayRecord string
}

type H2HMatchItem struct {
	Date        string
	HomeTeam    string
	AwayTeam    string
	Score       string
	Competition string
}

type H2HFormEvent struct {
	Opponent string
	Score    string
	Result   string
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
		HomeScore:   homeScore,
		AwayScore:   awayScore,
		Tabs:        NewTabs([]string{"Alineaciones", "Eventos", "Estadísticas", "Jugadores", "H2H", "Lesiones"}),
		PlayerStats: NewPlayerStatsView(),
	}
}

func (md *MatchDetail) ScrollUp(step int) {
	md.ScrollOff -= step
	if md.ScrollOff < 0 {
		md.ScrollOff = 0
	}
	if md.PlayerStats != nil {
		md.PlayerStats.ScrollUp(step)
	}
}

func (md *MatchDetail) ScrollDown(step int) {
	md.ScrollOff += step
	if md.PlayerStats != nil {
		md.PlayerStats.ScrollDown(step)
	}
}

func (md *MatchDetail) SetError(err string) {
	md.Error = err
}

// ChangePeriodPrev moves to the previous available stats period.
func (md *MatchDetail) ChangePeriodPrev() {
	if md.Details == nil {
		return
	}
	periods := md.availablePeriods()
	if len(periods) < 2 {
		return
	}
	cur := indexOf(periods, md.Details.SelectedPeriod)
	if cur < 0 {
		md.Details.SelectedPeriod = periods[0]
		return
	}
	cur--
	if cur < 0 {
		cur = len(periods) - 1
	}
	md.Details.SelectedPeriod = periods[cur]
	md.ScrollOff = 0
	if md.PlayerStats != nil {
		md.PlayerStats.Reset()
	}
}

// ChangePeriodNext moves to the next available stats period.
func (md *MatchDetail) ChangePeriodNext() {
	if md.Details == nil {
		return
	}
	periods := md.availablePeriods()
	if len(periods) < 2 {
		return
	}
	cur := indexOf(periods, md.Details.SelectedPeriod)
	if cur < 0 {
		md.Details.SelectedPeriod = periods[0]
		return
	}
	cur++
	if cur >= len(periods) {
		cur = 0
	}
	md.Details.SelectedPeriod = periods[cur]
	md.ScrollOff = 0
	if md.PlayerStats != nil {
		md.PlayerStats.Reset()
	}
}

// SelectPeriodByIndex selects the nth available period (1-based).
func (md *MatchDetail) SelectPeriodByIndex(n int) {
	if md.Details == nil {
		return
	}
	periods := md.availablePeriods()
	if n < 1 || n > len(periods) {
		return
	}
	md.Details.SelectedPeriod = periods[n-1]
	md.ScrollOff = 0
	if md.PlayerStats != nil {
		md.PlayerStats.Reset()
	}
}

func indexOf(slice []int, val int) int {
	for i, v := range slice {
		if v == val {
			return i
		}
	}
	return -1
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

	lines = append(lines, mdTitleStyle.Render("Detalle del Partido"))
	lines = append(lines, "")
	scoreCell := ""
	if md.HomeScore != "" && md.AwayScore != "" {
		scoreText := fmt.Sprintf(" %s - %s ", md.HomeScore, md.AwayScore)
		if len(md.PenShootout) > 0 {
			livePen := formatPenShots(md.PenShootout)
			scoreText += fmt.Sprintf("(%s) ", livePen)
		} else if md.PenScore != "" {
			penParts := strings.Split(md.PenScore, "-")
			if len(penParts) == 2 {
				scoreText += fmt.Sprintf("(%s - %s) ", strings.TrimSpace(penParts[0]), strings.TrimSpace(penParts[1]))
			}
		}
		scoreCell = mdScoreStyle.Render(scoreText)
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
			if !strings.HasPrefix(c, "#") {
				c = "#" + c
			}
			homeHeaderColor = lipgloss.Color(c)
		}
		if md.Details.Events.ExtraInfo.AwayColor != "" {
			c := md.Details.Events.ExtraInfo.AwayColor
			if !strings.HasPrefix(c, "#") {
				c = "#" + c
			}
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

	if md.Minute != "" {
		minuteStr := md.Minute
		if md.WaterBreak {
			minuteStr += " H2O"
		}
		lines = append(lines, fmt.Sprintf("  %s %s",
			mdLiveIndicatorStyle.Render("●"),
			mdLiveMinuteStyle.Render(minuteStr)))
	} else {
		statusLabel := md.Status
		if statusLabel == "" {
			statusLabel = "programado"
		}
		lines = append(lines, mdInfoStyle.Render(statusLabel))
	}
	lines = append(lines, mdInfoStyle.Render(md.DateTime))

	if md.Details != nil && md.Details.Events.ExtraInfo != nil {
		info := md.Details.Events.ExtraInfo
		if info.ESPNStatus != "" {
			lines = append(lines, mdInfoStyle.Render(info.ESPNStatus))
		}
		if info.Venue != "" {
			venueLine := info.Venue
			if info.VenueCapacity > 0 {
				venueLine = fmt.Sprintf("%s — Cap. %d", venueLine, info.VenueCapacity)
			}
			if info.Surface != "" && info.Surface != "grass" {
				venueLine = fmt.Sprintf("%s · %s", venueLine, info.Surface)
			}
			lines = append(lines, mdInfoStyle.Render(venueLine))
		}
		if info.Attendance > 0 {
			lines = append(lines, mdInfoStyle.Render(fmt.Sprintf("Asistencia: %d", info.Attendance)))
		}
		if info.Referee != "" {
			lines = append(lines, mdInfoStyle.Render(fmt.Sprintf("Árbitro: %s", info.Referee)))
		}
		if info.Weather != "" {
			lines = append(lines, mdInfoStyle.Render(fmt.Sprintf("Clima: %s", info.Weather)))
		}
		if len(info.Broadcasts) > 0 {
			lines = append(lines, mdInfoStyle.Render(fmt.Sprintf("TV: %s", strings.Join(info.Broadcasts, ", "))))
		}
	}
	lines = append(lines, "")

	lines = append(lines, md.Tabs.Render(width))
	lines = append(lines, "")

	contentH := height - len(lines) - 2
	content := md.renderTabContent(width, contentH)
	lines = append(lines, content)

	lines = append(lines, mdBackStyle.Render("←/→ tabs · u/d scroll · a/s periodo · esc volver"))

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
		if md.Details == nil || md.PlayerStats == nil {
			return mdInfoStyle.Render("sin estadisticas de jugadores")
		}
		homeColor, awayColor := md.teamColors()
		return md.PlayerStats.Render(
			md.Details.PlayerStats,
			md.HomeName, md.AwayName,
			homeColor, awayColor,
			width, height,
		)
	case 4:
		return md.renderH2H(width, height)
	case 5:
		return md.renderInjuries(width, height)
	}
	return ""
}

func (md *MatchDetail) teamColors() (homeColor, awayColor lipgloss.Color) {
	homeColor = lipgloss.Color("39")
	awayColor = lipgloss.Color("196")
	if md.Details != nil && md.Details.Events.ExtraInfo != nil {
		if c := md.Details.Events.ExtraInfo.HomeColor; c != "" {
			if !strings.HasPrefix(c, "#") {
				c = "#" + c
			}
			homeColor = lipgloss.Color(c)
		}
		if c := md.Details.Events.ExtraInfo.AwayColor; c != "" {
			if !strings.HasPrefix(c, "#") {
				c = "#" + c
			}
			awayColor = lipgloss.Color(c)
		}
	}
	return
}

func (md *MatchDetail) applyScroll(body string, width, height int) string {
	return applyColumnScroll(body, width, height, &md.ScrollOff)
}

func applyColumnScroll(body string, width, height int, scrollOff *int) string {
	allLines := strings.Split(body, "\n")
	total := len(allLines)

	maxScroll := total - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if *scrollOff > maxScroll {
		*scrollOff = maxScroll
	}

	start := *scrollOff
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

	scrollIndicator := ""
	if maxScroll > 0 {
		pct := *scrollOff * 100 / maxScroll
		scrollIndicator = mdInfoStyle.Render(fmt.Sprintf("  [%d%%]", pct))
	}

	return result + scrollIndicator
}

func formatPenShots(shots []domain.PenShot) string {
	if len(shots) == 0 {
		return ""
	}
	const maxRounds = 5
	homeChars := make([]bool, 0, maxRounds)
	awayChars := make([]bool, 0, maxRounds)
	for _, s := range shots {
		if s.Team == domain.SideHome {
			homeChars = append(homeChars, s.Scored)
		} else {
			awayChars = append(awayChars, s.Scored)
		}
	}
	homeStr := formatPenSide(homeChars, maxRounds)
	awayStr := formatPenSide(awayChars, maxRounds)
	return fmt.Sprintf("%s : %s", homeStr, awayStr)
}

func formatPenSide(results []bool, maxRounds int) string {
	var sb strings.Builder
	for _, scored := range results {
		if scored {
			sb.WriteString("✓ ")
		} else {
			sb.WriteString("X ")
		}
	}
	if len(results) < maxRounds {
		for i := 0; i < maxRounds-len(results); i++ {
			sb.WriteString("〇 ")
		}
	}
	return strings.TrimRight(sb.String(), " ")
}

package tui

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"tifo/internal/domain"
	"tifo/internal/enrich"
	"tifo/internal/persistence/sqlite"
	fotmobProvider "tifo/internal/providers/fotmob"
	espnProvider "tifo/internal/providers/espn"
	fotmob "tifo/fotmob"
	espn "tifo/espn"
	"tifo/internal/providers"
	"tifo/internal/resolver"
	"tifo/internal/services"
	"tifo/ipapi"
	"tifo/tui/components"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Align(lipgloss.Center).
			PaddingTop(1).
			PaddingBottom(1)

	separatorStyle = lipgloss.NewStyle().
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("236"))

	footerStyle = lipgloss.NewStyle().
			Width(80).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("240")).
			Italic(true)

	loadingStyle = lipgloss.NewStyle().
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("240"))

	matchStyle = lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1).
			Foreground(lipgloss.Color("255"))

	matchTimeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	matchScoreStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255"))

	matchCursorStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("39"))

	leagueHeaderStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Bold(true).
				Foreground(lipgloss.Color("63"))

	emptyStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color("240"))

	vsStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
)

type Model struct {
	svc       *services.MatchService
	width     int
	height    int
	ipapi     *ipapi.Client
	location  *ipapi.Location
	leftList  components.LeagueList
	ready     bool
	err       error
	leagues   []domain.Competition
	matches   []domain.Match

	loadingMatch bool
	matchScroll  int
	matchIdx     int
	selDate       time.Time
	calendar      components.Calendar
	showCalendar  bool

	selectedMatch   *domain.Match
	matchDetails    *domain.MatchDetails
	loadingDetail   bool
	detailErr       string
	detailTab       int
	detailScrollOff int
	espnStatus      string
}

func New() Model {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return Model{
		ipapi:    ipapi.NewClient(),
		leftList: components.NewLeagueList(nil),
		selDate:  today,
		calendar: components.NewCalendar(today),
	}
}

func (m Model) locale() string {
	if m.location != nil {
		return m.location.Locale()
	}
	return "es-419"
}

func (m Model) country() string {
	if m.location != nil && m.location.CountryCode != "" {
		return m.location.CountryCode
	}
	return "MEX"
}

func buildService() *services.MatchService {
	db, err := sqlite.OpenMappingDB("tifo_mappings.db")
	if err != nil {
		log.Printf("[tifo] mapping db: %v (proceeding without cache)", err)
	}

	mr := resolver.NewMatchResolver(db)
	tr := resolver.NewTeamResolver(db)
	cr := resolver.NewCompetitionResolver(db)

	oldFotmob := fotmob.NewService()
	oldESPN := espn.NewService()

	fp := fotmobProvider.NewProvider(oldFotmob, mr, tr, cr)
	ep := espnProvider.NewProvider(oldESPN)

	return services.NewMatchService(fp, []providers.Provider{ep}, enrich.DefaultMergeConfig())
}

func (m Model) Init() tea.Cmd {
	f, err := tea.LogToFile("tifo.log", "tifo")
	if err != nil {
		fmt.Println("could not open log file:", err)
	} else {
		log.Println("=== tifo started ===")
		_ = f
	}
	return tea.Batch(
		func() tea.Msg { return initSvcMsg{svc: buildService()} },
		fetchIPLocation(m.ipapi),
	)
}

// Messages
type initSvcMsg struct {
	svc *services.MatchService
	err error
}

type locationMsg struct {
	loc *ipapi.Location
	err error
}

type leaguesMsg struct {
	leagues []domain.Competition
	err     error
}

type matchesMsg struct {
	matches  []domain.Match
	leagueID int
	err      error
}

type detailsMsg struct {
	details *domain.MatchDetails
	err     error
}

// Commands
func fetchIPLocation(c *ipapi.Client) tea.Cmd {
	return func() tea.Msg {
		loc, err := c.GetLocation()
		if err != nil {
			return locationMsg{err: err}
		}
		return locationMsg{loc: loc}
	}
}

func fetchLeagues(svc *services.MatchService, locale, country string) tea.Cmd {
	return func() tea.Msg {
		leagues, err := svc.Leagues(nil, locale, country)
		return leaguesMsg{leagues: leagues, err: err}
	}
}

func fetchMatches(svc *services.MatchService, fotmobID string) tea.Cmd {
	return func() tea.Msg {
		matches, err := svc.LeagueMatches(nil, fotmobID)
		return matchesMsg{matches: matches, err: err}
	}
}

func fetchMatchDetails(svc *services.MatchService, matchID string, ctx services.MatchContext) tea.Cmd {
	return func() tea.Msg {
		log.Printf("[TUI] fetchDetails match=%s league=%q time=%v home=%q away=%q",
			matchID, ctx.LeagueName, ctx.UTCTime, ctx.HomeTeam, ctx.AwayTeam)
		details, err := svc.MatchDetails(nil, matchID, ctx)
		return detailsMsg{details: details, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case initSvcMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.svc = msg.svc
		if m.location != nil || m.err != nil {
			return m, fetchLeagues(m.svc, m.locale(), m.country())
		}
		return m, nil

	case locationMsg:
		if msg.err == nil {
			m.location = msg.loc
		} else {
			log.Printf("[TUI] location error: %v (using default)", msg.err)
		}
		if m.svc != nil {
			return m, fetchLeagues(m.svc, m.locale(), m.country())
		}
		return m, nil

	case leaguesMsg:
		m.ready = true
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.leagues = msg.leagues
		items := make([]components.LeagueItem, 0, len(msg.leagues))
		for _, l := range msg.leagues {
			items = append(items, components.LeagueItem{
				TIFOID:       l.TIFOID,
				Name:         l.Name,
				OriginalName: l.OriginalName,
				Country:      l.Country,
			})
		}
		m.leftList.SetLeagues(items)
		if len(msg.leagues) > 0 {
			m.loadingMatch = true
			if id, ok := msg.leagues[0].ExternalIDs.Get("fotmob"); ok {
				return m, fetchMatches(m.svc, id)
			}
		}

	case matchesMsg:
		m.loadingMatch = false
		if msg.err == nil {
			m.matches = msg.matches
			m.matchScroll = 0
		}

	case detailsMsg:
		m.loadingDetail = false
		if msg.err != nil {
			m.detailErr = msg.err.Error()
			m.matchDetails = nil
			log.Printf("[TUI] detail error: %s", m.detailErr)
		} else {
			m.detailErr = ""
			m.matchDetails = msg.details
		}

	case tea.KeyMsg:
		if m.showCalendar {
			return m.updateCalendar(msg)
		}
		if m.selectedMatch != nil {
			return m.updateDetail(msg)
		}
		return m.updateBrowse(msg)
	}

	return m, nil
}

func (m Model) updateCalendar(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.showCalendar = false
	case "enter":
		m.selDate = m.calendar.Date()
		m.showCalendar = false
		m.calendar.SetDate(m.selDate)
	case "up":
		m.calendar.CursorUp()
	case "down":
		m.calendar.CursorDown()
	case "left":
		m.calendar.CursorLeft()
	case "right":
		m.calendar.CursorRight()
	}
	return m, nil
}

func (m Model) updateDetail(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.selectedMatch = nil
		m.matchDetails = nil
	case "left", "h":
		if m.detailTab > 0 {
			m.detailTab--
		} else {
			m.detailTab = 4
		}
	case "right", "l":
		if m.detailTab < 4 {
			m.detailTab++
		} else {
			m.detailTab = 0
		}
	case "u":
		m.detailScrollOff -= 3
		if m.detailScrollOff < 0 {
			m.detailScrollOff = 0
		}
	case "d":
		m.detailScrollOff += 3
	}
	return m, nil
}

func (m Model) updateBrowse(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Batch(tea.ClearScreen, tea.Quit)

	case "enter":
		matches := m.filteredMatches()
		if len(matches) == 0 {
			return m, nil
		}
		idx := m.matchIdx
		if idx < 0 {
			idx = 0
		}
		if idx >= len(matches) {
			idx = len(matches) - 1
		}

		m.selectedMatch = &matches[idx]
		m.detailTab = 0
		m.detailScrollOff = 0
		m.matchDetails = nil
		m.detailErr = ""
		m.loadingDetail = true

		// Build match context for enrichment
		ctx := services.MatchContext{}
		if sel := m.leftList.Selected(); sel != nil && m.svc != nil {
			if id, ok := matches[idx].ExternalIDs.Get("fotmob"); ok {
				ctx.HomeTeam = matches[idx].Home.Name
				ctx.AwayTeam = matches[idx].Away.Name
				ctx.UTCTime = matches[idx].Status.Kickoff
				ctx.LeagueName = sel.OriginalName

				return m, fetchMatchDetails(m.svc, id, ctx)
			}
		}

	case "c":
		m.showCalendar = true

	case "up", "k":
		m.leftList.Up()
		sel := m.leftList.Selected()
		if sel != nil && len(m.leagues) > m.leftList.Cursor() {
			if id, ok := m.leagues[m.leftList.Cursor()].ExternalIDs.Get("fotmob"); ok {
				m.loadingMatch = true
				return m, fetchMatches(m.svc, id)
			}
		}

	case "down", "j":
		m.leftList.Down()
		sel := m.leftList.Selected()
		if sel != nil && len(m.leagues) > m.leftList.Cursor() {
			if id, ok := m.leagues[m.leftList.Cursor()].ExternalIDs.Get("fotmob"); ok {
				m.loadingMatch = true
				return m, fetchMatches(m.svc, id)
			}
		}

	case "n":
		if m.matchIdx < len(m.matches)-1 {
			m.matchIdx++
		}
	case "p":
		if m.matchIdx > 0 {
			m.matchIdx--
		}
	case "left":
		m.selDate = m.selDate.AddDate(0, 0, -1)
		m.calendar.SetDate(m.selDate)
	case "right":
		m.selDate = m.selDate.AddDate(0, 0, 1)
		m.calendar.SetDate(m.selDate)
	}

	return m, nil
}

func (m Model) filteredMatches() []domain.Match {
	dateStr := m.selDate.Format("2006-01-02")
	var out []domain.Match
	for _, match := range m.matches {
		if t := match.Status.Kickoff; !t.IsZero() {
			// Convert UTC kickoff to local time before comparing dates
			if t.In(time.Local).Format("2006-01-02") == dateStr {
				out = append(out, match)
			}
		} else if strings.HasPrefix(match.Status.UTCTime, dateStr) {
			out = append(out, match)
		}
	}
	return out
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "cargando..."
	}

	sepW := 1
	leftW := (m.width - sepW*2) * 2 / 10
	rightW := (m.width - sepW*2) * 2 / 10
	centerW := m.width - leftW - rightW - sepW*2

	separator := separatorStyle.
		Width(sepW).
		Height(m.height - 1).
		Render("│")

	// Left sidebar
	leftTitle := titleStyle.Width(leftW).Render("LIGAS")
	var leftBody string
	if !m.ready {
		leftBody = loadingStyle.Width(leftW).Height(m.height - 3).Render("cargando...")
	} else {
		leftBody = m.leftList.Render(leftW, m.height-3)
	}
	leftView := lipgloss.JoinVertical(lipgloss.Top, leftTitle, leftBody)

	// Center panel
	dateNavStyle := lipgloss.NewStyle().
		Width(centerW).
		Bold(true).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("39"))

	dateStr := m.selDate.Format("Mon 2006-01-02")
	today := time.Now()
	isToday := m.selDate.Year() == today.Year() && m.selDate.YearDay() == today.YearDay()
	dateNav := fmt.Sprintf("  < %s >", dateStr)
	if isToday {
		dateNav = fmt.Sprintf("  < %s • Today >", dateStr)
	}

	dateView := dateNavStyle.Render(dateNav)

	filtered := m.filteredMatches()

	var centerBody string
	if m.loadingMatch {
		centerBody = loadingStyle.Width(centerW).Height(m.height - 4).Render("cargando...")
	} else if len(m.matches) == 0 {
		centerBody = emptyStyle.Width(centerW).Height(m.height - 4).
			Render("selecciona una liga")
	} else if len(filtered) == 0 {
		centerBody = emptyStyle.Width(centerW).Height(m.height - 4).
			Render(fmt.Sprintf("sin partidos el %s", m.selDate.Format("2006-01-02")))
	} else {
		var lines []string
		sel := m.leftList.Selected()
		if sel != nil {
			lines = append(lines, leagueHeaderStyle.Render(sel.Name))
		}

		availHeight := m.height - 4
		headerLines := len(lines)
		maxMatches := availHeight - headerLines - 1
		if maxMatches < 0 {
			maxMatches = 0
		}

		if m.matchScroll > len(filtered) {
			m.matchScroll = len(filtered)
		}
		start := m.matchScroll
		end := start + maxMatches
		if end > len(filtered) {
			end = len(filtered)
		}

		shown := filtered[start:end]
		for i, match := range shown {
			matchIdx := start + i
			if matchIdx == m.matchIdx && m.selectedMatch == nil {
				lines = append(lines, matchCursorStyle.Render("▸ ")+formatMatch(match))
			} else {
				lines = append(lines, "  "+formatMatch(match))
			}
		}

		if len(filtered) > end {
			lines = append(lines, emptyStyle.Render(fmt.Sprintf("  ▼ %d más", len(filtered)-end)))
		}

		centerBody = lipgloss.JoinVertical(lipgloss.Top, lines...)

		remaining := availHeight - lipgloss.Height(centerBody)
		if remaining > 0 {
			centerBody += "\n" + lipgloss.NewStyle().Width(centerW).Height(remaining).Render("")
		}
	}

	centerHeader := lipgloss.JoinVertical(lipgloss.Top, titleStyle.Width(centerW).Render("PARTIDOS"), dateView)
	centerView := lipgloss.JoinVertical(lipgloss.Top, centerHeader, centerBody)

	// Right sidebar
	rightTitle := titleStyle.Width(rightW).Render("INFO")
	var rightLines []string
	if m.location != nil {
		rightLines = append(rightLines,
			emptyStyle.Render(fmt.Sprintf("%s", m.location.CountryName)),
			emptyStyle.Render(fmt.Sprintf("   %s", m.location.City)),
		)
	}
	rightBody := components.NewSidebar("", rightLines).Render(rightW, m.height-3)
	rightView := lipgloss.JoinVertical(lipgloss.Top, rightTitle, rightBody)

	mainRow := lipgloss.JoinHorizontal(lipgloss.Top,
		leftView, separator, centerView, separator, rightView,
	)

	footer := footerStyle.
		Width(m.width).
		Render("↑/k ↓/j ligas · n/p partidos · ←/→ día · c calendario · ↩ detalle · u/d scroll · q salir")

	mainView := lipgloss.JoinVertical(lipgloss.Top, mainRow, footer)

	if m.showCalendar {
		calView := lipgloss.NewStyle().Width(centerW).Align(lipgloss.Center).
			Render(m.calendar.Render(23, 10))
		centerView = lipgloss.JoinVertical(lipgloss.Top, centerHeader, calView)

		mainRow = lipgloss.JoinHorizontal(lipgloss.Top,
			leftView, separator, centerView, separator, rightView,
		)
		mainView = lipgloss.JoinVertical(lipgloss.Top, mainRow, footer)
	} else if m.selectedMatch != nil {
		md := m.buildDetailView()
		detailView := md.Render(m.width, m.height-2)
		m.detailScrollOff = md.ScrollOff
		mainView = lipgloss.JoinVertical(lipgloss.Top, detailView, footer)
	}

	return mainView
}

func (m Model) buildDetailView() *components.MatchDetail {
	match := m.selectedMatch

	md := components.NewMatchDetail(
		match.Home.Name,
		match.Away.Name,
		match.Status.ScoreStr,
		statusLabel(match.Status),
		formatTime(match.Status),
		"",
	)

	md.Tabs = components.NewTabs([]string{"Alineaciones", "Eventos", "Estadísticas", "H2H", "Lesiones"})
	for i := 0; i < m.detailTab; i++ {
		md.Tabs.Right()
	}
	md.ScrollOff = m.detailScrollOff

	if m.matchDetails != nil {
		d := m.matchDetails
		md.Details = buildFromDomain(d, m.espnStatus)
	} else if m.loadingDetail {
		md.Details = nil
	} else if m.detailErr != "" {
		md.SetError(m.detailErr)
	}

	return &md
}

func buildFromDomain(d *domain.MatchDetails, espnStatus string) *components.MatchDetailData {
	data := &components.MatchDetailData{}

	// Stats
	for _, cat := range d.Statistics {
		sc := components.StatCategory{Title: cat.Title}
		for _, s := range cat.Stats {
			sc.Stats = append(sc.Stats, components.StatRow{
				Label: s.Label,
				Home:  s.Home,
				Away:  s.Away,
			})
		}
		data.Stats = append(data.Stats, sc)
	}

	// Lineups
	if d.Lineups != nil {
		data.Lineup = components.LineupData{
			HomeFormation: d.Lineups.HomeFormation,
			AwayFormation: d.Lineups.AwayFormation,
			HomeCoach:     d.Lineups.HomeCoach,
			AwayCoach:     d.Lineups.AwayCoach,
		}
		for _, p := range d.Lineups.HomeStarters {
			data.Lineup.HomeStarters = append(data.Lineup.HomeStarters, components.PlayerLineup{
				Name: p.Name, Number: p.Number,
			})
		}
		for _, p := range d.Lineups.HomeSubs {
			data.Lineup.HomeSubs = append(data.Lineup.HomeSubs, components.PlayerLineup{
				Name: p.Name, Number: p.Number,
			})
		}
		for _, p := range d.Lineups.AwayStarters {
			data.Lineup.AwayStarters = append(data.Lineup.AwayStarters, components.PlayerLineup{
				Name: p.Name, Number: p.Number,
			})
		}
		for _, p := range d.Lineups.AwaySubs {
			data.Lineup.AwaySubs = append(data.Lineup.AwaySubs, components.PlayerLineup{
				Name: p.Name, Number: p.Number,
			})
		}
	}

	// Events
	for _, ev := range d.Events {
		player := ""
		if ev.Player != nil {
			player = ev.Player.Name
		}
		detail := ev.Detail
		subOut, subIn := "", ""
		if ev.SubOut != nil {
			subOut = ev.SubOut.Name
		}
		if ev.SubIn != nil {
			subIn = ev.SubIn.Name
		}
		team := d.Match.Away
		if ev.Team == domain.SideHome {
			team = d.Match.Home
		}

		minute := fmt.Sprintf("%d", ev.Minute)
		if ev.SortOverload > 0 {
			minute = fmt.Sprintf("%d+%d", ev.Minute, ev.SortOverload)
		}

		data.Events.Items = append(data.Events.Items, components.EventItem{
			Minute:       minute,
			EventType:    string(ev.EventType),
			Player:       player,
			Team:         team,
			HomeScore:    ev.HomeScore,
			AwayScore:    ev.AwayScore,
			CardType:     ev.CardType,
			IsHome:       ev.Team == domain.SideHome,
			Detail:       detail,
			SubOut:       subOut,
			SubIn:        subIn,
			AddedTime:    ev.AddedTime,
			GoalDesc:     ev.GoalDesc,
			HalfStr:      ev.HalfStr,
			OwnGoal:      ev.OwnGoal,
			ShotDesc:     ev.ShotDesc,
			SortTime:     ev.SortTime,
			SortOverload: ev.SortOverload,
		})
	}

	// Sort events
	sort.Slice(data.Events.Items, func(i, j int) bool {
		ti, tj := data.Events.Items[i].SortTime, data.Events.Items[j].SortTime
		if ti == tj {
			return data.Events.Items[i].SortOverload < data.Events.Items[j].SortOverload
		}
		return ti < tj
	})

	// H2H
	if d.H2H != nil {
		data.H2H = components.H2HData{
			HomeWins: d.H2H.HomeWins,
			Draws:    d.H2H.Draws,
			AwayWins: d.H2H.AwayWins,
		}
	}

	// Injuries
	for _, inj := range d.Injuries {
		item := components.InjuryPlayer{
			Name: inj.Player.Name, Type: inj.Type, Return: inj.Return,
		}
		if inj.Team == domain.SideHome {
			data.Injuries.Home = append(data.Injuries.Home, item)
		} else {
			data.Injuries.Away = append(data.Injuries.Away, item)
		}
	}

	// Extra info
	data.Events.ExtraInfo = &components.MatchExtraInfo{
		Venue:       d.ExtraInfo.Venue,
		Attendance:  d.ExtraInfo.Attendance,
		Referee:     d.ExtraInfo.Referee,
		Weather:     d.ExtraInfo.Weather,
		Broadcasts:  d.ExtraInfo.Broadcasts,
		HomeColor:   d.ExtraInfo.HomeColor,
		AwayColor:   d.ExtraInfo.AwayColor,
		ESPNStatus:  espnStatus,
	}

	return data
}

func formatMatch(m domain.Match) string {
	ko := m.Status.Kickoff
	timeStr := "--:--"
	if !ko.IsZero() {
		timeStr = ko.In(time.Local).Format("15:04")
	}

	home := m.Home.Name
	away := m.Away.Name

	if m.HomeScore != nil && m.AwayScore != nil {
		return fmt.Sprintf("%s %s  %s vs %s",
			matchTimeStyle.Render(timeStr),
			matchScoreStyle.Render(fmt.Sprintf("%d-%d", *m.HomeScore, *m.AwayScore)),
			matchStyle.Render(home),
			matchStyle.Render(away))
	}

	return fmt.Sprintf("%s  %s vs %s",
		matchTimeStyle.Render(timeStr),
		matchStyle.Render(home),
		matchStyle.Render(away))
}

func statusLabel(s domain.MatchStatus) string {
	switch s.State {
	case domain.MatchScheduled:
		return "Programado"
	case domain.MatchLive:
		if s.Detail != "" {
			return s.Detail
		}
		return "En vivo"
	case domain.MatchFinished:
		return "Finalizado"
	case domain.MatchPostponed:
		return "Postergado"
	default:
		return string(s.State)
	}
}

func formatTime(s domain.MatchStatus) string {
	if t := s.Kickoff; !t.IsZero() {
		return t.In(time.Local).Format("01-02 15:04")
	}
	if len(s.UTCTime) >= 16 {
		return s.UTCTime[5:10] + " " + s.UTCTime[11:16]
	}
	return s.UTCTime
}

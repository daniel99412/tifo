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

	liveIndicatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))

	liveMinuteStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("215"))
)

const uiTickInterval = 5 * time.Second
const dataRefreshInterval = 30 * time.Second

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

	selectedMatch *domain.Match
	detailView    *components.MatchDetail
	matchDetails  *domain.MatchDetails
	loadingDetail bool
	detailErr     string
	espnStatus    string
	lastDataRefresh time.Time
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
		tickCmd(),
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

type tickMsg struct{}

// Commands
func tickCmd() tea.Cmd {
	return tea.Tick(uiTickInterval, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

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
			m.lastDataRefresh = time.Now()
			oldID := ""
			if m.selectedMatch != nil {
				if id, ok := m.selectedMatch.ExternalIDs.Get("fotmob"); ok {
					oldID = id
				}
			}
			m.matches = msg.matches
			m.matchScroll = 0
			m.matchIdx = 0
			if oldID != "" {
				found := false
				for i := range m.matches {
					if id, ok := m.matches[i].ExternalIDs.Get("fotmob"); ok && id == oldID {
						m.selectedMatch = &m.matches[i]
						found = true
						break
					}
				}
				if !found {
					m.selectedMatch = nil
					m.detailView = nil
					m.matchDetails = nil
				}
			}
		}

	case detailsMsg:
		m.loadingDetail = false
		if msg.err != nil {
			m.detailErr = msg.err.Error()
			m.matchDetails = nil
			log.Printf("[TUI] detail error: %s", m.detailErr)
			if m.detailView != nil {
				m.detailView.SetError(msg.err.Error())
			}
		} else {
			m.detailErr = ""
			m.matchDetails = msg.details
			m.lastDataRefresh = time.Now()
			if m.detailView != nil {
				m.detailView.Details = buildFromDomain(msg.details, m.espnStatus)
				if msg.details.Match.Score != "" {
					m.detailView.Score = msg.details.Match.Score
					parts := strings.Split(msg.details.Match.Score, "-")
					if len(parts) == 2 {
						m.detailView.HomeScore = strings.TrimSpace(parts[0])
						m.detailView.AwayScore = strings.TrimSpace(parts[1])
					}
				}
				m.detailView.WaterBreak = isWaterBreakActive(msg.details.Events)
				if isHalfTime(msg.details.Events) {
					m.detailView.Minute = "HT"
				}
				// If FT detected in events, mark match as finished
				if m.selectedMatch != nil && isMatchFinished(*m.selectedMatch, msg.details.Events) {
					m.selectedMatch.Status.State = domain.MatchFinished
					m.detailView.Minute = ""
					m.detailView.Status = statusLabel(m.selectedMatch.Status)
					m.detailView.WaterBreak = false
				}
			}
		}

	case tickMsg:
		cmds := []tea.Cmd{tickCmd()}
		if m.svc == nil {
			return m, tea.Batch(cmds...)
		}

		// Always update detail view minute/score immediately (local, no fetch)
		if m.selectedMatch != nil && m.detailView != nil {
			if isMatchLive(*m.selectedMatch) {
				minute := m.selectedMatch.Status.Detail
				if minute == "" || minute == "En vivo" {
					minute = computeMatchMinute(m.selectedMatch.Status.Kickoff)
				}
				m.detailView.Minute = minute
				if m.selectedMatch.HomeScore != nil && m.selectedMatch.AwayScore != nil {
					score := fmt.Sprintf("%d-%d", *m.selectedMatch.HomeScore, *m.selectedMatch.AwayScore)
					m.detailView.Score = score
					m.detailView.HomeScore = fmt.Sprintf("%d", *m.selectedMatch.HomeScore)
					m.detailView.AwayScore = fmt.Sprintf("%d", *m.selectedMatch.AwayScore)
				}
			} else {
				m.detailView.Minute = ""
				m.detailView.WaterBreak = false
				m.detailView.Status = statusLabel(m.selectedMatch.Status)
			}
		}

		// Only fetch from API every dataRefreshInterval
		now := time.Now()
		if now.Sub(m.lastDataRefresh) >= dataRefreshInterval {
			m.lastDataRefresh = now

			// Auto-refresh match list on today
			if !m.loadingMatch {
				today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				if m.selDate.Equal(today) {
					if sel := m.leftList.Selected(); sel != nil && len(m.leagues) > m.leftList.Cursor() {
						if id, ok := m.leagues[m.leftList.Cursor()].ExternalIDs.Get("fotmob"); ok {
							m.loadingMatch = true
							cmds = append(cmds, fetchMatches(m.svc, id))
						}
					}
				}
			}

			// Auto-refresh match details when in detail view on a live match
			if m.selectedMatch != nil && !m.loadingDetail && isMatchLive(*m.selectedMatch) {
				if id, ok := m.selectedMatch.ExternalIDs.Get("fotmob"); ok {
					ctx := services.MatchContext{}
					if sel := m.leftList.Selected(); sel != nil {
						ctx.HomeTeam = m.selectedMatch.Home.Name
						ctx.AwayTeam = m.selectedMatch.Away.Name
						ctx.UTCTime = m.selectedMatch.Status.Kickoff
						ctx.LeagueName = sel.OriginalName
					}
					m.loadingDetail = true
					cmds = append(cmds, fetchMatchDetails(m.svc, id, ctx))
				}
			}
		}

		return m, tea.Batch(cmds...)

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
		m.detailView = nil
		m.matchDetails = nil
	case "left", "h":
		if m.detailView != nil {
			m.detailView.Tabs.Left()
		}
	case "right", "l":
		if m.detailView != nil {
			m.detailView.Tabs.Right()
		}
	case "u":
		if m.detailView != nil {
			m.detailView.ScrollOff -= 3
			if m.detailView.ScrollOff < 0 {
				m.detailView.ScrollOff = 0
			}
		}
	case "d":
		if m.detailView != nil {
			m.detailView.ScrollOff += 3
		}
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
		mdVal := components.NewMatchDetail(
			matches[idx].Home.Name,
			matches[idx].Away.Name,
			matches[idx].Status.ScoreStr,
			statusLabel(matches[idx].Status),
			formatTime(matches[idx].Status),
			"",
		)
		if isMatchLive(matches[idx]) {
			minute := matches[idx].Status.Detail
			if minute == "" || minute == "En vivo" {
				minute = computeMatchMinute(matches[idx].Status.Kickoff)
			}
			mdVal.Minute = minute
		}
		mdVal.Tabs = components.NewTabs([]string{"Alineaciones", "Eventos", "Estadísticas", "H2H", "Lesiones"})
		m.detailView = &mdVal
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
		filtered := m.filteredMatches()
		if m.matchIdx < len(filtered)-1 {
			m.matchIdx++
		}
	case "p":
		if m.matchIdx > 0 {
			m.matchIdx--
		}
	case "left":
		m.selDate = m.selDate.AddDate(0, 0, -1)
		m.calendar.SetDate(m.selDate)
		m.matchIdx = 0
		m.matchScroll = 0
	case "right":
		m.selDate = m.selDate.AddDate(0, 0, 1)
		m.calendar.SetDate(m.selDate)
		m.matchIdx = 0
		m.matchScroll = 0
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
	} else if m.detailView != nil {
		detailView := m.detailView.Render(m.width, m.height-2)
		mainView = lipgloss.JoinVertical(lipgloss.Top, detailView, footer)
	}

	return mainView
}

func posAbbr(posID int, posName string) string {
	if posName != "" {
		return strings.ReplaceAll(posName, "-", "")
	}
	switch posID {
	case 0, 1: return "POR"
	case 2: return "DFC"
	case 3: return "MC"
	case 4: return "DC"
	default: return ""
	}
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
		mapPlayer := func(p domain.PlayerRef) components.PlayerLineup {
			return components.PlayerLineup{Name: p.Name, Number: p.Number, PosName: posAbbr(p.PosID, p.PosName)}
		}
		sortByPos := func(ps []domain.PlayerRef) []components.PlayerLineup {
			sorted := make([]domain.PlayerRef, len(ps))
			copy(sorted, ps)
			sort.Slice(sorted, func(i, j int) bool { return sorted[i].PosID < sorted[j].PosID })
			out := make([]components.PlayerLineup, len(sorted))
			for i, p := range sorted { out[i] = mapPlayer(p) }
			return out
		}
		data.Lineup.HomeStarters = sortByPos(d.Lineups.HomeStarters)
		data.Lineup.HomeSubs = sortByPos(d.Lineups.HomeSubs)
		data.Lineup.AwayStarters = sortByPos(d.Lineups.AwayStarters)
		data.Lineup.AwaySubs = sortByPos(d.Lineups.AwaySubs)
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

func isMatchFinished(m domain.Match, events []domain.MatchEvent) bool {
	if m.Status.State == domain.MatchFinished {
		return true
	}
	for _, ev := range events {
		if ev.EventType == domain.EvFT {
			return true
		}
	}
	if !m.Status.Kickoff.IsZero() && time.Since(m.Status.Kickoff) > 120*time.Minute {
		return true
	}
	return false
}

func isMatchLive(m domain.Match) bool {
	if isMatchFinished(m, nil) {
		return false
	}
	switch m.Status.State {
	case domain.MatchLive:
		return true
	case domain.MatchScheduled:
		if !m.Status.Kickoff.IsZero() {
			elapsed := time.Since(m.Status.Kickoff)
			if elapsed > 0 && elapsed < 120*time.Minute {
				return true
			}
		}
		if m.HomeScore != nil && m.AwayScore != nil && !m.Status.Kickoff.IsZero() {
			if time.Since(m.Status.Kickoff) > 0 {
				return true
			}
		}
	}
	return false
}

func computeMatchMinute(ko time.Time) string {
	if ko.IsZero() {
		return ""
	}
	totalSec := int(time.Since(ko).Seconds())
	em := totalSec / 60
	es := totalSec % 60

	switch {
	case em < 45:
		return fmt.Sprintf("%d:%02d", em, es)
	case em < 48:
		return fmt.Sprintf("45:%02d", es)
	case em < 63:
		return "HT"
	default:
		sm := em - 15
		if sm < 90 {
			return fmt.Sprintf("%d:%02d", sm, es)
		}
		return fmt.Sprintf("90:%02d", es)
	}
}

func isWaterBreakActive(events []domain.MatchEvent) bool {
	if len(events) == 0 {
		return false
	}
	sorted := make([]domain.MatchEvent, len(events))
	copy(sorted, events)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].SortTime == sorted[j].SortTime {
			return sorted[i].SortOverload < sorted[j].SortOverload
		}
		return sorted[i].SortTime < sorted[j].SortTime
	})
	lastPause := -1
	lastResume := -1
	for i, ev := range sorted {
		switch ev.EventType {
		case domain.EvWaterBreak, "CoolingBreak", "DrinkBreak", domain.EvPausa:
			lastPause = i
		case domain.EvContinua:
			lastResume = i
		}
	}
	return lastPause > lastResume
}

func isHalfTime(events []domain.MatchEvent) bool {
	if len(events) == 0 {
		return false
	}
	sorted := make([]domain.MatchEvent, len(events))
	copy(sorted, events)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].SortTime == sorted[j].SortTime {
			return sorted[i].SortOverload < sorted[j].SortOverload
		}
		return sorted[i].SortTime < sorted[j].SortTime
	})
	lastHT := -1
	lastS2 := -1
	for i, ev := range sorted {
		if ev.EventType == domain.EvHalf && ev.HalfStr == "HT" {
			lastHT = i
		}
		if ev.EventType == domain.EvS2 {
			lastS2 = i
		}
	}
	return lastHT > lastS2
}

func padRight(s string, w int) string {
	if d := w - lipgloss.Width(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}

func padCenter(s string, w int) string {
	sw := lipgloss.Width(s)
	if d := w - sw; d > 0 {
		l := d / 2
		return strings.Repeat(" ", l) + s + strings.Repeat(" ", d-l)
	}
	return s
}

func formatMatch(m domain.Match) string {
	ko := m.Status.Kickoff
	timeStr := "--:--"
	if !ko.IsZero() {
		timeStr = ko.In(time.Local).Format("15:04")
	}

	// Col 1: marcador en vivo
	var col1 string
	if isMatchLive(m) {
		col1 = liveIndicatorStyle.Render("●")
	} else {
		col1 = " "
	}

	// Col 2: horario / minuto de juego
	var col2 string
	if isMatchLive(m) {
		minute := m.Status.Detail
		if minute == "" || minute == "En vivo" {
			minute = computeMatchMinute(ko)
		}
		col2 = liveMinuteStyle.Render(minute)
	} else {
		col2 = matchTimeStyle.Render(timeStr)
	}

	// Col 3: equipo local
	col3 := matchStyle.Render(m.Home.Name)

	// Col 4: vs / resultado
	var col4 string
	if m.HomeScore != nil && m.AwayScore != nil {
		col4 = matchScoreStyle.Render(fmt.Sprintf("%d : %d", *m.HomeScore, *m.AwayScore))
	} else if isMatchLive(m) {
		col4 = matchScoreStyle.Render("0 : 0")
		log.Printf("[score] live match %q %q state=%s scoreStr=%q homeScore=%v awayScore=%v",
			m.Home.Name, m.Away.Name, m.Status.State, m.Status.ScoreStr,
			m.HomeScore, m.AwayScore)
	} else {
		col4 = vsStyle.Render("vs")
	}

	// Col 5: equipo visitante
	col5 := matchStyle.Render(m.Away.Name)

	const c1W = 2
	const c2W = 6
	const c3W = 18
	const c4W = 7

	return padRight(col1, c1W) +
		padRight(col2, c2W) +
		padRight(col3, c3W) +
		padCenter(col4, c4W) +
		col5
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

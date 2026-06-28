package tui

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"tifo/enricher"
	"tifo/espn"
	"tifo/fotmob"
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
	width        int
	height       int
	fotmob       *fotmob.Service
	ipapi        *ipapi.Client
	location     *ipapi.Location
	leftList     components.LeagueList
	rightSidebar components.Sidebar
	ready        bool
	err          error
	matches      []fotmob.LeagueMatch
	matchLeague  int
	loadingMatch bool
	matchScroll  int
	matchIdx     int
	selDate       time.Time
	calendar      components.Calendar
	showCalendar  bool
	selectedMatch     *fotmob.LeagueMatch
	matchDetails      *fotmob.MatchDetailsResponse
	matchEnrich       *enricher.EnrichedMatch
	loadingDetail     bool
	detailErr         string
	detailTab         int
	detailScrollOff   int
	espn              *espn.Service
	loadingESPN       bool
	espnErr           string
	pendingESPN       *espn.EnrichData
}

func New() Model {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return Model{
		fotmob:       fotmob.NewService(),
		espn:         espn.NewService(),
		ipapi:        ipapi.NewClient(),
		leftList:     components.NewLeagueList(nil),
		rightSidebar: components.NewSidebar("", []string{}),
		selDate:      today,
		calendar:     components.NewCalendar(today),
	}
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
		fetchLocation(m.ipapi),
		fetchLeagues(m.fotmob, "es-419", "MEX"),
	)
}

func fetchLocation(c *ipapi.Client) tea.Cmd {
	return func() tea.Msg {
		loc, err := c.GetLocation()
		if err != nil {
			return locationMsg{err: err}
		}
		return locationMsg{loc: loc}
	}
}

func fetchLeagues(svc *fotmob.Service, locale, country string) tea.Cmd {
	return func() tea.Msg {
		leagues, err := svc.GetPopularLeagues(locale, country)
		if err != nil {
			return leaguesMsg{err: err}
		}
		return leaguesMsg{leagues: leagues}
	}
}

func fetchMatches(svc *fotmob.Service, leagueID int) tea.Cmd {
	return func() tea.Msg {
		matches, err := svc.LeagueMatches(leagueID)
		if err != nil {
			return matchesMsg{err: err}
		}
		return matchesMsg{matches: matches, leagueID: leagueID}
	}
}

func fetchESPN(svc *espn.Service, matchID string, leagueName string, utcTime time.Time, homeTeam, awayTeam string) tea.Cmd {
	return func() tea.Msg {
		log.Printf("[TUI] fetchESPN match=%s league=%q time=%s home=%q away=%q", matchID, leagueName, utcTime.Format(time.RFC3339), homeTeam, awayTeam)
		fid := 0
		fmt.Sscanf(matchID, "%d", &fid)
		data, err := svc.FetchMatch(fid, leagueName, utcTime, homeTeam, awayTeam)
		if err != nil {
			log.Printf("[TUI] fetchESPN error: %v", err)
			return espnMsg{err: err}
		}
		log.Printf("[TUI] fetchESPN success: eventID=%s league=%s home=%s away=%s", data.EventID, data.LeagueSlug, data.HomeTeam, data.AwayTeam)
		return espnMsg{data: data}
	}
}

func fetchMatchDetails(svc *fotmob.Service, matchID string) tea.Cmd {
	return func() tea.Msg {
		details, err := svc.MatchDetailsByID(matchID)
		if err != nil {
			return detailsMsg{err: err}
		}
		return detailsMsg{details: details}
	}
}

type locationMsg struct {
	loc *ipapi.Location
	err error
}

type leaguesMsg struct {
	leagues []fotmob.GroupedLeague
	err     error
}

type matchesMsg struct {
	matches  []fotmob.LeagueMatch
	leagueID int
	err      error
}

type detailsMsg struct {
	details *fotmob.MatchDetailsResponse
	err     error
}

type espnMsg struct {
	data *espn.EnrichData
	err  error
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case locationMsg:
		if msg.err == nil {
			m.location = msg.loc
			return m, fetchLeagues(m.fotmob, msg.loc.Locale(), msg.loc.CountryCode)
		}

	case leaguesMsg:
		m.ready = true
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.leftList.SetLeagues(msg.leagues)
		if len(msg.leagues) > 0 {
			m.loadingMatch = true
			return m, fetchMatches(m.fotmob, msg.leagues[0].ID)
		}

	case matchesMsg:
		m.loadingMatch = false
		if msg.err == nil {
			m.matches = msg.matches
			m.matchLeague = msg.leagueID
			m.matchScroll = 0
		}

	case detailsMsg:
		m.loadingDetail = false
		if msg.err != nil {
			m.detailErr = msg.err.Error()
			m.matchDetails = nil
			log.Printf("[TUI] fotmob detail error: %s", m.detailErr)
		} else {
			m.detailErr = ""
			m.matchDetails = msg.details
			log.Printf("[TUI] fotmob detail loaded: stats=%d events=%d", len(msg.details.Content.Stats.Periods.All.Stats), len(msg.details.Content.MatchFacts.Events.Events))
			if m.pendingESPN != nil {
				m.matchEnrich = enricher.Enrich(msg.details, m.pendingESPN)
				log.Printf("[TUI] enriched from pending ESPN data")
				m.pendingESPN = nil
			}
		}

	case espnMsg:
		m.loadingESPN = false
		if msg.err != nil {
			m.espnErr = msg.err.Error()
			log.Printf("[TUI] espn error: %s", m.espnErr)
		} else {
			m.espnErr = ""
			log.Printf("[TUI] espn success, fotmob ready=%v", m.matchDetails != nil)
			if m.matchDetails != nil {
				m.matchEnrich = enricher.Enrich(m.matchDetails, msg.data)
				log.Printf("[TUI] enriched: venue=%q att=%d ref=%q weather=%q colors=%s/%s",
					m.matchEnrich.Venue, m.matchEnrich.Attendance, m.matchEnrich.Referee,
					m.matchEnrich.Weather, m.matchEnrich.HomeColor, m.matchEnrich.AwayColor)
				if m.matchEnrich.ESPNStats != nil {
					log.Printf("[TUI] ESPN stats: %d mappings applied", len(m.matchEnrich.ESPNStats))
				}
				m.loadingDetail = false
			} else {
				m.pendingESPN = msg.data
			}
		}

	case tea.KeyMsg:
		if m.showCalendar {
			switch msg.String() {
			case "esc":
				m.showCalendar = false
			case "enter":
				m.selDate = m.calendar.Date()
				m.showCalendar = false
				m.calendar.SetDate(m.selDate)
				return m, nil
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

		if m.selectedMatch != nil {
			switch msg.String() {
			case "esc", "backspace":
				m.selectedMatch = nil
				m.matchDetails = nil
				m.matchEnrich = nil
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

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Batch(tea.ClearScreen, tea.Quit)
		case "enter":
			if matches := m.filteredMatches(); len(matches) > 0 {
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
				m.matchEnrich = nil
				m.pendingESPN = nil
				m.espnErr = ""
				m.detailErr = ""
				m.loadingDetail = true
				m.loadingESPN = true

				cmds := []tea.Cmd{fetchMatchDetails(m.fotmob, matches[idx].ID)}

				if sel := m.leftList.Selected(); sel != nil {
					utcTime, err := parseUTCTime(matches[idx].Status.UTCTime)
					if err == nil {
						leagueName := sel.OriginalName
						if leagueName == "" {
							leagueName = sel.Name
						}
						cmds = append(cmds, fetchESPN(m.espn, matches[idx].ID, leagueName, utcTime,
							matches[idx].Home.Name, matches[idx].Away.Name))
					}
				}

				return m, tea.Batch(cmds...)
			}
		case "c":
			m.showCalendar = true
		case "up", "k":
			m.leftList.Up()
			if sel := m.leftList.Selected(); sel != nil && sel.ID != m.matchLeague {
				m.loadingMatch = true
				m.matchIdx = 0
				return m, fetchMatches(m.fotmob, sel.ID)
			}
		case "down", "j":
			m.leftList.Down()
			if sel := m.leftList.Selected(); sel != nil && sel.ID != m.matchLeague {
				m.loadingMatch = true
				m.matchIdx = 0
				return m, fetchMatches(m.fotmob, sel.ID)
			}
		case "n":
			if matches := m.filteredMatches(); len(matches) > 0 {
				m.matchIdx++
				if m.matchIdx >= len(matches) {
					m.matchIdx = 0
				}
			}
		case "p":
			if matches := m.filteredMatches(); len(matches) > 0 {
				m.matchIdx--
				if m.matchIdx < 0 {
					m.matchIdx = len(matches) - 1
				}
			}
		case "left", "h":
			m.selDate = m.selDate.AddDate(0, 0, -1)
			m.calendar.SetDate(m.selDate)
			m.matchIdx = 0
		case "right", "l":
			m.selDate = m.selDate.AddDate(0, 0, 1)
			m.calendar.SetDate(m.selDate)
			m.matchIdx = 0
		case "d":
			m.matchScroll += 5
		case "u":
			m.matchScroll -= 5
			if m.matchScroll < 0 {
				m.matchScroll = 0
			}
		}
	}
	return m, nil
}

func formatMatch(m fotmob.LeagueMatch) string {
	var sb strings.Builder

	home := m.Home.Name
	away := m.Away.Name

	if m.Status.ScoreStr != "" {
		sb.WriteString(matchStyle.Render(fmt.Sprintf("%-22s", home)))
		sb.WriteString(matchScoreStyle.Render(fmt.Sprintf(" %s ", m.Status.ScoreStr)))
		sb.WriteString(matchStyle.Render(away))
	} else {
		timeStr := formatTime(m.Status.UTCTime)
		sb.WriteString(matchStyle.Render(fmt.Sprintf("%-22s", home)))
		sb.WriteString(vsStyle.Render(" vs "))
		sb.WriteString(matchStyle.Render(fmt.Sprintf("%-22s", away)))
		sb.WriteString(matchTimeStyle.Render(" " + timeStr))
	}

	return sb.String()
}

func statusLabel(s fotmob.MatchStatus) string {
	if s.Finished {
		return "Finalizado"
	}
	if s.Started {
		return "En vivo"
	}
	if s.Cancelled {
		return "Cancelado"
	}
	return "Programado"
}

func buildDetailData(d *fotmob.MatchDetailsResponse, enrich *enricher.EnrichedMatch, espnLoading bool, espnErr string) *components.MatchDetailData {
	if d == nil {
		return nil
	}

	data := &components.MatchDetailData{}

	// Stats — FotMob primary, ESPN fill for nulls
	for _, cat := range d.Content.Stats.Periods.All.Stats {
		sc := components.StatCategory{Title: cat.Title}
		for _, s := range cat.Stats {
			homeVal := ""
			if len(s.Stats) > 0 && s.Stats[0] != nil {
				homeVal = fmt.Sprintf("%v", s.Stats[0])
			}
			awayVal := ""
			if len(s.Stats) > 1 && s.Stats[1] != nil {
				awayVal = fmt.Sprintf("%v", s.Stats[1])
			}

			// Fill from ESPN if FotMob has null
			if homeVal == "" || awayVal == "" {
				if enrich != nil && enrich.ESPNStats != nil {
					if e, ok := enrich.ESPNStats[s.Key]; ok {
						if homeVal == "" {
							homeVal = e[0]
						}
						if awayVal == "" {
							awayVal = e[1]
						}
					}
				}
			}

			sc.Stats = append(sc.Stats, components.StatRow{
				Label: s.Title,
				Home:  homeVal,
				Away:  awayVal,
			})
		}
		data.Stats = append(data.Stats, sc)
	}

	// Lineup
	lu := &data.Lineup
	lu.HomeFormation = d.Content.Lineup.HomeTeam.Formation
	lu.AwayFormation = d.Content.Lineup.AwayTeam.Formation
	lu.HomeCoach = d.Content.Lineup.HomeTeam.Coach.Name
	lu.AwayCoach = d.Content.Lineup.AwayTeam.Coach.Name
	for _, p := range d.Content.Lineup.HomeTeam.Starters {
		lu.HomeStarters = append(lu.HomeStarters, components.PlayerLineup{
			Name:   p.Name,
			Number: p.ShirtNumber,
		})
	}
	for _, p := range d.Content.Lineup.HomeTeam.Subs {
		lu.HomeSubs = append(lu.HomeSubs, components.PlayerLineup{
			Name:   p.Name,
			Number: p.ShirtNumber,
		})
	}
	for _, p := range d.Content.Lineup.AwayTeam.Starters {
		lu.AwayStarters = append(lu.AwayStarters, components.PlayerLineup{
			Name:   p.Name,
			Number: p.ShirtNumber,
		})
	}
	for _, p := range d.Content.Lineup.AwayTeam.Subs {
		lu.AwaySubs = append(lu.AwaySubs, components.PlayerLineup{
			Name:   p.Name,
			Number: p.ShirtNumber,
		})
	}

	// Events - all events from matchFacts + shotmap sorted by time
	var evItems []components.EventItem

	for _, ev := range d.Content.MatchFacts.Events.Events {
		team := ""
		if ev.IsHome {
			team = d.General.HomeTeam.Name
		} else {
			team = d.General.AwayTeam.Name
		}
		player := ""
		if ev.Player != nil {
			player = ev.Player.Name
		}
		minute := fmt.Sprintf("%d", ev.Time)
		overload := 0
		if ev.OverloadTime != nil && *ev.OverloadTime > 0 {
			minute = fmt.Sprintf("%d+%d", ev.Time, *ev.OverloadTime)
			overload = *ev.OverloadTime
		}
		detail := ""
		subOut := ""
		subIn := ""
		if len(ev.Swap) >= 2 {
			subOut = ev.Swap[1].Name
			subIn = ev.Swap[0].Name
			detail = fmt.Sprintf("↓ %s · ↑ %s", subOut, subIn)
		}
		if ev.InjuredPlayerOut {
			if detail != "" {
				detail += " (lesión)"
			} else {
				detail = "lesión"
			}
		}
		addedTime := 0
		if ev.MinutesAddedInput > 0 {
			addedTime = ev.MinutesAddedInput
		}
		cardType := ev.Card
		goalDesc := ev.GoalDescription
		halfStr := ev.HalfStrShort
		ownGoal := ev.OwnGoal != nil

		evItems = append(evItems, components.EventItem{
			Minute:     minute,
			EventType:  ev.Type,
			Player:     player,
			Team:       team,
			HomeScore:  ev.HomeScore,
			AwayScore:  ev.AwayScore,
			CardType:   cardType,
			IsHome:     ev.IsHome,
			Detail:     detail,
			SubOut:     subOut,
			SubIn:      subIn,
			AddedTime:  addedTime,
			GoalDesc:   goalDesc,
			HalfStr:    halfStr,
			OwnGoal:    ownGoal,
			SortTime:   ev.Time,
			SortOverload: overload,
		})
	}

	// Add shotmap events
	for _, s := range d.Content.Shotmap.Shots {
		minute := fmt.Sprintf("%d", s.Min)
		overload := 0
		if s.MinAdded != nil && *s.MinAdded > 0 {
			minute = fmt.Sprintf("%d+%d", s.Min, *s.MinAdded)
			overload = *s.MinAdded
		}
		isHome := teamIDMatch(d.General.HomeTeam.ID, s.TeamID)
		team := d.General.AwayTeam.Name
		if isHome {
			team = d.General.HomeTeam.Name
		}
		desc := ""
		switch s.EventType {
		case "Goal":
			desc = "gol"
		case "AttemptSaved":
			desc = "atajado"
		case "Miss":
			desc = "falló"
		default:
			desc = s.EventType
		}
		evItems = append(evItems, components.EventItem{
			Minute:       minute,
			EventType:    "Shot",
			Player:       s.PlayerName,
			Team:         team,
			IsHome:       isHome,
			ShotDesc:     desc,
			SortTime:     s.Min,
			SortOverload: overload,
		})
	}

	// Inject ESPN extra events
	if enrich != nil {
		for _, ee := range enrich.ExtraEvents {
			evItems = append(evItems, components.EventItem{
				Minute:       fmt.Sprintf("%d", ee.Minute),
				EventType:    ee.EventType,
				Detail:       ee.Description,
				Team:         ee.TeamSide,
				SortTime:     ee.Minute,
				SortOverload: ee.AddedTime,
			})
		}
	}

	// Fix Half/FT events to account for added time (match ends at 90+X, not 90)
	for i, ev := range evItems {
		if ev.EventType == "Half" && (ev.HalfStr == "FT" || ev.HalfStr == "HT") && ev.SortOverload == 0 {
			for _, at := range evItems {
				if at.EventType == "AddedTime" && at.SortTime == ev.SortTime && at.AddedTime > 0 {
					evItems[i].SortOverload = at.AddedTime
					evItems[i].Minute = fmt.Sprintf("%d+%d", ev.SortTime, at.AddedTime)
					break
				}
			}
		}
	}

	sort.Slice(evItems, func(i, j int) bool {
		ti := evItems[i].SortTime
		tj := evItems[j].SortTime
		if ti == tj {
			return evItems[i].SortOverload < evItems[j].SortOverload
		}
		return ti < tj
	})
	data.Events.Items = evItems

	// H2H
	if len(d.Content.H2H.Summary) >= 3 {
		data.H2H = components.H2HData{
			HomeWins: d.Content.H2H.Summary[0],
			Draws:    d.Content.H2H.Summary[1],
			AwayWins: d.Content.H2H.Summary[2],
		}
	}

	// Injuries
	for _, p := range d.Content.Lineup.HomeTeam.Unavailable {
		data.Injuries.Home = append(data.Injuries.Home, components.InjuryPlayer{
			Name:   p.Name,
			Type:   p.Unavailability.Type,
			Return: p.Unavailability.ExpectedReturn,
		})
	}
	for _, p := range d.Content.Lineup.AwayTeam.Unavailable {
		data.Injuries.Away = append(data.Injuries.Away, components.InjuryPlayer{
			Name:   p.Name,
			Type:   p.Unavailability.Type,
			Return: p.Unavailability.ExpectedReturn,
		})
	}

	// Team colors: FotMob primary, ESPN fallback
	extraInfo := &components.MatchExtraInfo{}
	extraInfo.HomeColor = d.General.TeamColors.LightMode.Home
	extraInfo.AwayColor = d.General.TeamColors.LightMode.Away
	if extraInfo.HomeColor == "" {
		extraInfo.HomeColor = d.General.TeamColors.DarkMode.Home
	}
	if extraInfo.AwayColor == "" {
		extraInfo.AwayColor = d.General.TeamColors.DarkMode.Away
	}

	if espnLoading {
		extraInfo.ESPNStatus = "ESPN: cargando..."
	}
	if espnErr != "" {
		extraInfo.ESPNStatus = fmt.Sprintf("ESPN: %s", espnErr)
	}
	if enrich != nil {
		extraInfo.Venue = enrich.Venue
		extraInfo.Attendance = enrich.Attendance
		extraInfo.Referee = enrich.Referee
		extraInfo.Weather = enrich.Weather
		extraInfo.Broadcasts = enrich.Broadcasts
		if extraInfo.HomeColor == "" {
			extraInfo.HomeColor = enrich.HomeColor
		}
		if extraInfo.AwayColor == "" {
			extraInfo.AwayColor = enrich.AwayColor
		}
		extraInfo.HomeAltColor = enrich.HomeAltColor
		extraInfo.AwayAltColor = enrich.AwayAltColor
		extraInfo.ESPNStatus = ""
	}
	data.Events.ExtraInfo = extraInfo

	return data
}

func teamIDMatch(id interface{}, teamID int) bool {
	switch v := id.(type) {
	case int:
		return v == teamID
	case float64:
		return int(v) == teamID
	case string:
		var n int
		_, err := fmt.Sscanf(v, "%d", &n)
		return err == nil && n == teamID
	}
	return false
}

func formatTime(utcTime string) string {
	// try to parse known time layouts and convert to local time
	if t, err := parseUTCTime(utcTime); err == nil {
		local := t.In(time.Local)
		return local.Format("01-02 15:04")
	}

	if len(utcTime) >= 16 {
		return utcTime[5:10] + " " + utcTime[11:16]
	}
	return utcTime
}

func matchDate(m fotmob.LeagueMatch) string {
	if t, err := parseUTCTime(m.Status.UTCTime); err == nil {
		return t.In(time.Local).Format("2006-01-02")
	}
	if len(m.Status.UTCTime) >= 10 {
		return m.Status.UTCTime[:10]
	}
	return ""
}

func filterMatches(matches []fotmob.LeagueMatch, date time.Time) []fotmob.LeagueMatch {
	dateStr := date.Format("2006-01-02")
	var out []fotmob.LeagueMatch
	for _, m := range matches {
		if matchDate(m) == dateStr {
			out = append(out, m)
		}
	}
	return out
}

func (m Model) filteredMatches() []fotmob.LeagueMatch {
	return filterMatches(m.matches, m.selDate)
}

func parseUTCTime(utc string) (time.Time, error) {
	// try common layouts (RFC3339 is preferred)
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, utc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized time format")
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
	} else {
		dateNav = fmt.Sprintf("  < %s >", dateStr)
	}
	dateView := dateNavStyle.Render(dateNav)

	filtered := filterMatches(m.matches, m.selDate)

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
			emptyStyle.Render(fmt.Sprintf("📍 %s", m.location.CountryName)),
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
		md := components.NewMatchDetail(
			m.selectedMatch.Home.Name,
			m.selectedMatch.Away.Name,
			m.selectedMatch.Status.ScoreStr,
			statusLabel(m.selectedMatch.Status),
			formatTime(m.selectedMatch.Status.UTCTime),
			m.selectedMatch.ID,
		)
		md.Tabs = components.NewTabs([]string{"Alineaciones", "Eventos", "Estadísticas", "H2H", "Lesiones"})
		for i := 0; i < m.detailTab; i++ {
			md.Tabs.Right()
		}
		md.ScrollOff = m.detailScrollOff

		if m.matchDetails != nil {
			md.Details = buildDetailData(m.matchDetails, m.matchEnrich, m.loadingESPN, m.espnErr)
		} else if m.loadingDetail {
			md.Details = nil
		} else if m.detailErr != "" {
			md.SetError(m.detailErr)
		} else {
			// still loading fotmob details
		}

		detailView := md.Render(m.width, m.height-2)
		mainView = lipgloss.JoinVertical(lipgloss.Top, detailView, footer)
	}

	return mainView
}

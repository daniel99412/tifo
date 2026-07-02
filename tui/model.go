package tui

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	espn "tifo/espn"
	fotmob "tifo/fotmob"
	"tifo/internal/domain"
	"tifo/internal/persistence/sqlite"
	"tifo/internal/providers"
	espnProvider "tifo/internal/providers/espn"
	fotmobProvider "tifo/internal/providers/fotmob"
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

const dataRefreshInterval = 5 * time.Second

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
	detailMinute  int
	detailUpdated time.Time
	batchDone     bool
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

	return services.NewMatchService(fp, []providers.Provider{ep})
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
		renderTickCmd(),
		dataTickCmd(),
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
	batch   bool // true = from background batch, false = from manual Enter
}

type renderTickMsg struct{}
type dataTickMsg struct{}

// Commands
func renderTickCmd() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return renderTickMsg{}
	})
}

func dataTickCmd() tea.Cmd {
	return tea.Tick(dataRefreshInterval, func(t time.Time) tea.Msg {
		return dataTickMsg{}
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
		return detailsMsg{details: details, err: err, batch: false}
	}
}

// Worker pool semaphore for batch fetching — max 3 concurrent HTTP calls.
var batchSem = make(chan struct{}, 3)

// fetchBatchDetails is like fetchMatchDetails but uses MatchDetailsLight (no ESPN enrichment) and marks as batch.
func fetchBatchDetails(svc *services.MatchService, matchID string) tea.Cmd {
	return func() tea.Msg {
		batchSem <- struct{}{}
		defer func() { <-batchSem }()

		log.Printf("[batch] fetching details for match %s", matchID)
		details, err := svc.MatchDetailsLight(nil, matchID)
		return detailsMsg{details: details, err: err, batch: true}
	}
}

// batchFetchEnrichCmd returns a tea.Cmd that fetches light details for all live and finished matches
// with a worker pool limited to 3 concurrent calls.
func batchFetchEnrichCmd(svc *services.MatchService, matches []domain.Match) tea.Cmd {
	var cmds []tea.Cmd
	for _, m := range matches {
		if m.Status.State != domain.MatchLive && m.Status.State != domain.MatchFinished {
			continue
		}
		id, ok := m.ExternalIDs.Get("fotmob")
		if !ok || id == "" {
			continue
		}
		cmds = append(cmds, fetchBatchDetails(svc, id))
	}
	if len(cmds) == 0 {
		return nil
	}
	log.Printf("[batch] queued %d batch detail fetches (concurrency limited to 3)", len(cmds))
	return tea.Batch(cmds...)
}

// selectMatch sets selectedMatch from filtered matches and creates detail view, returning fetch commands.
func (m *Model) selectMatch(idx int, matches []domain.Match) []tea.Cmd {
	m.selectedMatch = nil
	if matchID, ok := matches[idx].ExternalIDs.Get("fotmob"); ok {
		for j := range m.matches {
			if mid, ok2 := m.matches[j].ExternalIDs.Get("fotmob"); ok2 && mid == matchID {
				m.selectedMatch = &m.matches[j]
				break
			}
		}
	}
	if m.selectedMatch == nil {
		m.selectedMatch = &matches[idx]
	}

	selMatch := m.selectedMatch
	mdVal := components.NewMatchDetail(
		selMatch.Home.Name,
		selMatch.Away.Name,
		selMatch.Status.ScoreStr,
		statusLabel(selMatch.Status),
		formatTime(selMatch.Status),
		"",
	)
	if isMatchLive(*selMatch) {
		minute := selMatch.Status.Detail
		if minute == "" || minute == "En vivo" {
			minute = computeMatchMinute(selMatch.Status.Kickoff, selMatch.Status.FirstHalfAddedTime)
		}
		mdVal.Minute = minute
	}
	mdVal.Tabs = components.NewTabs([]string{"Alineaciones", "Eventos", "Estadísticas", "H2H", "Lesiones"})
	m.detailView = &mdVal
	m.matchDetails = nil
	m.detailErr = ""
	m.loadingDetail = true

	if sel := m.leftList.Selected(); sel != nil && m.svc != nil {
		if id, ok := selMatch.ExternalIDs.Get("fotmob"); ok {
			ctx := services.MatchContext{
				HomeTeam:   selMatch.Home.Name,
				AwayTeam:   selMatch.Away.Name,
				UTCTime:    selMatch.Status.Kickoff,
				LeagueName: sel.OriginalName,
			}
			return []tea.Cmd{fetchMatchDetails(m.svc, id, ctx)}
		}
	}
	return nil
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
		if msg.err != nil {
			return m, nil
		}
		oldID := ""
		var savedPenScore *int
		var savedAwayPenScore *int
		var savedShootout []domain.PenShot
		if m.selectedMatch != nil {
			if id, ok := m.selectedMatch.ExternalIDs.Get("fotmob"); ok {
				oldID = id
			}
			savedPenScore = m.selectedMatch.HomePenScore
			savedAwayPenScore = m.selectedMatch.AwayPenScore
			savedShootout = m.selectedMatch.PenShootout
		}
		m.matches = msg.matches
		m.matchScroll = 0
		m.matchIdx = 0
		if oldID != "" {
			found := false
			for i := range m.matches {
				if id, ok := m.matches[i].ExternalIDs.Get("fotmob"); ok && id == oldID {
					m.selectedMatch = &m.matches[i]
					// Restore enriched penalty data from previous details fetch
					if savedPenScore != nil {
						m.selectedMatch.HomePenScore = savedPenScore
					}
					if savedAwayPenScore != nil {
						m.selectedMatch.AwayPenScore = savedAwayPenScore
					}
					if len(savedShootout) > 0 {
						m.selectedMatch.PenShootout = savedShootout
					}
					found = true
					log.Printf("[penalty] restored enriched data on matches refresh: homePen=%v awayPen=%v shootout=%d",
						savedPenScore, savedAwayPenScore, len(savedShootout))
					break
				}
			}
			if !found {
				m.selectedMatch = nil
				m.detailView = nil
				m.matchDetails = nil
			}
		}
		// Fire background batch (once per league load) to enrich matches with live time and penalty data.
		if !m.batchDone {
			m.batchDone = true
			return m, batchFetchEnrichCmd(m.svc, m.matches)
		}
		return m, nil

	case detailsMsg:
		m.loadingDetail = false
		if msg.err != nil {
			// Always log the error — don't propagate to detailView for batch errors
			if !msg.batch {
				m.detailErr = msg.err.Error()
				m.matchDetails = nil
				log.Printf("[TUI] detail error: %s", m.detailErr)
				if m.detailView != nil {
					m.detailView.SetError(msg.err.Error())
				}
			} else {
				log.Printf("[batch] error for match: %v", msg.err)
			}
			break
		}
		// For all details (batch or manual): update penalty data on the live m.matches entry.
		detailFotmobID, _ := msg.details.ExternalIDs.Get("fotmob")
		if detailFotmobID != "" {
			for i := range m.matches {
				if id, ok := m.matches[i].ExternalIDs.Get("fotmob"); ok && id == detailFotmobID {
					if msg.details.Match.PenScore != "" {
						penParts := strings.Split(msg.details.Match.PenScore, "-")
						if len(penParts) == 2 {
							if hps, err := strconv.Atoi(strings.TrimSpace(penParts[0])); err == nil {
								m.matches[i].HomePenScore = &hps
							}
							if aps, err := strconv.Atoi(strings.TrimSpace(penParts[1])); err == nil {
								m.matches[i].AwayPenScore = &aps
							}
						}
					}
					if len(msg.details.Match.PenShootout) > 0 {
						m.matches[i].PenShootout = msg.details.Match.PenShootout
					}
					log.Printf("[batch] updated m.matches[%d] with penalty data: homePen=%v awayPen=%v shootout=%d",
						i, m.matches[i].HomePenScore, m.matches[i].AwayPenScore, len(m.matches[i].PenShootout))

					// FirstHalfAddedTime from events
					for _, ev := range msg.details.Events {
						if ev.SortTime == 45 && ev.AddedTime > 0 {
							m.matches[i].Status.FirstHalfAddedTime = ev.AddedTime
							log.Printf("[clock] extracted FirstHalfAddedTime=%d for match %s", ev.AddedTime, detailFotmobID)
							break
						}
					}

					// If overall selectedMatch ref is stale, re-point it.
					if m.selectedMatch != nil {
						if sid, ok := m.selectedMatch.ExternalIDs.Get("fotmob"); ok && sid == detailFotmobID {
							m.selectedMatch = &m.matches[i]
						}
					}
					break
				}
			}
		}
		// Batch results stop here — no detail view updates.
		if msg.batch {
			break
		}
		// Manual (Enter) detail: update the detail view and selected match.
		if msg.err == nil {
			m.detailErr = ""
			m.matchDetails = msg.details
			// Use LiveMinute as primary source, events as fallback
			newMinute := msg.details.Match.LiveMinute
			if newMinute == 0 {
				newMinute = lastEventMinute(msg.details.Events)
			}
			// Only snap when the minute actually changed (avoids sawtooth)
			if newMinute > 0 && newMinute != m.detailMinute {
				m.detailMinute = newMinute
				m.detailUpdated = time.Now()
			}
			// Extract FirstHalfAddedTime
			for _, ev := range msg.details.Events {
				if ev.SortTime == 45 && ev.AddedTime > 0 {
					if m.selectedMatch != nil {
						m.selectedMatch.Status.FirstHalfAddedTime = ev.AddedTime
					}
					break
				}
			}
		if m.detailView != nil {
			oldPeriod := 0
			if m.detailView.Details != nil {
				oldPeriod = m.detailView.Details.SelectedPeriod
			}
			m.detailView.Details = buildFromDomain(msg.details, m.espnStatus)
			if _, ok := m.detailView.Details.StatsByPeriod[oldPeriod]; ok {
				m.detailView.Details.SelectedPeriod = oldPeriod
			}
if msg.details.Match.Score != "" {
					m.detailView.Score = msg.details.Match.Score
					parts := strings.Split(msg.details.Match.Score, "-")
					if len(parts) == 2 {
						hs := strings.TrimSpace(parts[0])
						as := strings.TrimSpace(parts[1])
						m.detailView.HomeScore = hs
						m.detailView.AwayScore = as
						if m.selectedMatch != nil {
							if hi, err := strconv.Atoi(hs); err == nil {
								m.selectedMatch.HomeScore = &hi
							}
							if ai, err := strconv.Atoi(as); err == nil {
								m.selectedMatch.AwayScore = &ai
							}
							m.selectedMatch.Status.ScoreStr = msg.details.Match.Score
						}
					}
				}
				if msg.details.Match.PenScore != "" {
					log.Printf("[penalty] PenScore=%q from details", msg.details.Match.PenScore)
					m.detailView.PenScore = msg.details.Match.PenScore
					penParts := strings.Split(msg.details.Match.PenScore, "-")
					if len(penParts) == 2 && m.selectedMatch != nil {
						if hps, err := strconv.Atoi(strings.TrimSpace(penParts[0])); err == nil {
							m.selectedMatch.HomePenScore = &hps
							log.Printf("[penalty] selectedMatch.HomePenScore=%d", hps)
						}
						if aps, err := strconv.Atoi(strings.TrimSpace(penParts[1])); err == nil {
							m.selectedMatch.AwayPenScore = &aps
							log.Printf("[penalty] selectedMatch.AwayPenScore=%d", aps)
						}
					} else {
						log.Printf("[penalty] PenScore split failed: parts=%v, selectedMatch=%v", penParts, m.selectedMatch != nil)
					}
				} else {
					log.Printf("[penalty] no PenScore in details MatchRef (PenScore=%q Score=%q)", msg.details.Match.PenScore, msg.details.Match.Score)
				}
				if len(msg.details.Match.PenShootout) > 0 && m.selectedMatch != nil {
					m.selectedMatch.PenShootout = msg.details.Match.PenShootout
					log.Printf("[penalty] selectedMatch.PenShootout=%d shots", len(msg.details.Match.PenShootout))
				} else {
					log.Printf("[penalty] no PenShootout in details (len=%d, selectedMatch=%v)", len(msg.details.Match.PenShootout), m.selectedMatch != nil)
				}
				m.detailView.WaterBreak = isWaterBreakActive(msg.details.Events)
				// Only override state from events if provider hasn't explicitly set it
				if m.selectedMatch != nil {
					if m.selectedMatch.Status.State == domain.MatchScheduled && isMatchFinished(*m.selectedMatch, msg.details.Events) {
						m.selectedMatch.Status.State = domain.MatchFinished
						m.detailView.Minute = ""
						m.detailView.Status = statusLabel(m.selectedMatch.Status)
						m.detailView.WaterBreak = false
					} else {
						log.Printf("[debug] match id=%s state=%s events=%d hasAET=%v hasFT=%v",
							m.selectedMatch.TIFOID, m.selectedMatch.Status.State, len(msg.details.Events),
							hasEventType(msg.details.Events, domain.EvAETStart, domain.EvAETS2, domain.EvPenShootout),
							hasEventType(msg.details.Events, domain.EvFT))
					}
				}
			}
		}

	case renderTickMsg:
		// Refresh UI-only state (clock, score from local snapshot) every 1s
		if m.selectedMatch != nil && m.detailView != nil {
			if isMatchLive(*m.selectedMatch) {
				if m.detailMinute > 0 && !m.detailUpdated.IsZero() {
					m.detailView.Minute = computeMatchMinuteFromEvents(m.selectedMatch.Status.Kickoff, m.detailUpdated, m.detailMinute, m.selectedMatch.Status.FirstHalfAddedTime)
					m.selectedMatch.Status.Detail = m.detailView.Minute
				} else {
					m.detailView.Minute = computeMatchMinute(m.selectedMatch.Status.Kickoff, m.selectedMatch.Status.FirstHalfAddedTime)
				}
				if m.matchDetails != nil && isHalfTime(m.matchDetails.Events) {
					m.detailView.Minute = "HT"
					m.selectedMatch.Status.Detail = "HT"
				}
			if m.selectedMatch.HomeScore != nil && m.selectedMatch.AwayScore != nil {
				m.detailView.Score = fmt.Sprintf("%d-%d", *m.selectedMatch.HomeScore, *m.selectedMatch.AwayScore)
				m.detailView.HomeScore = fmt.Sprintf("%d", *m.selectedMatch.HomeScore)
				m.detailView.AwayScore = fmt.Sprintf("%d", *m.selectedMatch.AwayScore)
			}
			if m.selectedMatch.HomePenScore != nil && m.selectedMatch.AwayPenScore != nil {
				m.detailView.PenScore = fmt.Sprintf("%d-%d", *m.selectedMatch.HomePenScore, *m.selectedMatch.AwayPenScore)
			}
			if len(m.selectedMatch.PenShootout) > 0 {
				m.detailView.PenShootout = m.selectedMatch.PenShootout
			}
			} else {
				m.detailView.Minute = ""
				m.detailView.WaterBreak = false
				m.detailView.Status = statusLabel(m.selectedMatch.Status)
			}
		}
		return m, renderTickCmd()

	case dataTickMsg:
		// Fetch fresh data from API every 30s
		cmds := []tea.Cmd{dataTickCmd()}
		if m.svc == nil {
			return m, tea.Batch(cmds...)
		}
		now := time.Now()

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
	case "a":
		if m.detailView != nil && m.detailView.Tabs.Active() == 2 {
			m.detailView.ChangePeriodPrev()
		}
	case "s":
		if m.detailView != nil && m.detailView.Tabs.Active() == 2 {
			m.detailView.ChangePeriodNext()
		}
	case "1", "2", "3", "4", "5":
		if m.detailView != nil && m.detailView.Tabs.Active() == 2 {
			var n int
			fmt.Sscanf(msg.String(), "%d", &n)
			m.detailView.SelectPeriodByIndex(n)
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

		cmds := m.selectMatch(idx, matches)
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case "c":
		m.showCalendar = true

	case "up", "k":
		m.leftList.Up()
		sel := m.leftList.Selected()
		if sel != nil && len(m.leagues) > m.leftList.Cursor() {
			if id, ok := m.leagues[m.leftList.Cursor()].ExternalIDs.Get("fotmob"); ok {
				m.loadingMatch = true
				m.batchDone = false
				return m, fetchMatches(m.svc, id)
			}
		}

	case "down", "j":
		m.leftList.Down()
		sel := m.leftList.Selected()
		if sel != nil && len(m.leagues) > m.leftList.Cursor() {
			if id, ok := m.leagues[m.leftList.Cursor()].ExternalIDs.Get("fotmob"); ok {
				m.loadingMatch = true
				m.batchDone = false
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
	case 1: return "POR"
	case 2: return "DFC"
	case 3: return "MC"
	case 4: return "DC"
	default: return ""
	}
}

func buildFromDomain(d *domain.MatchDetails, espnStatus string) *components.MatchDetailData {
	data := &components.MatchDetailData{}

	// Stats by period
	for period, cats := range d.StatsByPeriod {
		uiCats := make([]components.StatCategory, 0, len(cats))
		for _, cat := range cats {
			sc := components.StatCategory{Title: cat.Title}
			for _, s := range cat.Stats {
				sc.Stats = append(sc.Stats, components.StatRow{
					Label: s.Label,
					Home:  s.Home,
					Away:  s.Away,
				})
			}
			uiCats = append(uiCats, sc)
		}
		if data.StatsByPeriod == nil {
			data.StatsByPeriod = make(map[int][]components.StatCategory)
		}
		data.StatsByPeriod[period] = uiCats
	}
	data.SelectedPeriod = domain.PeriodAll

	// Momentum
	for _, mp := range d.Momentum {
		data.Momentum = append(data.Momentum, components.MomentumPoint{
			Minute: mp.Minute,
			Value:  mp.Value,
		})
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
		h2h := components.H2HData{
			HomeWins:   d.H2H.HomeWins,
			Draws:      d.H2H.Draws,
			AwayWins:   d.H2H.AwayWins,
			HomeRecord: d.H2H.HomeRecord,
			AwayRecord: d.H2H.AwayRecord,
		}
		for _, fe := range d.H2H.HomeForm {
			h2h.HomeForm = append(h2h.HomeForm, components.H2HFormEvent{
				Opponent: fe.Opponent,
				Score:    fe.Score,
				Result:   fe.Result,
			})
		}
		for _, fe := range d.H2H.AwayForm {
			h2h.AwayForm = append(h2h.AwayForm, components.H2HFormEvent{
				Opponent: fe.Opponent,
				Score:    fe.Score,
				Result:   fe.Result,
			})
		}
		for _, m := range d.H2H.Matches {
			date := ""
			if !m.Date.IsZero() {
				date = m.Date.In(time.Local).Format("02-01-2006")
			}
			score := fmt.Sprintf("%d : %d", m.HomeScore, m.AwayScore)
			h2h.Matches = append(h2h.Matches, components.H2HMatchItem{
				Date:        date,
				HomeTeam:    m.HomeTeam,
				AwayTeam:    m.AwayTeam,
				Score:       score,
				Competition: m.Competition,
			})
		}
		data.H2H = h2h
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
		Venue:         d.ExtraInfo.Venue,
		VenueCapacity: d.ExtraInfo.VenueCapacity,
		Surface:       d.ExtraInfo.Surface,
		Attendance:    d.ExtraInfo.Attendance,
		Referee:       d.ExtraInfo.Referee,
		Weather:       d.ExtraInfo.Weather,
		Broadcasts:    d.ExtraInfo.Broadcasts,
		HomeColor:     d.ExtraInfo.HomeColor,
		AwayColor:     d.ExtraInfo.AwayColor,
	}

	return data
}

func hasEventType(events []domain.MatchEvent, types ...domain.EventType) bool {
	for _, ev := range events {
		for _, t := range types {
			if ev.EventType == t {
				return true
			}
		}
	}
	return false
}

func isMatchFinished(m domain.Match, events []domain.MatchEvent) bool {
	if m.Status.State == domain.MatchFinished {
		return true
	}
	hasAETOrPen := false
	for _, ev := range events {
		if ev.EventType == domain.EvAETStart || ev.EventType == domain.EvAETS2 || ev.EventType == domain.EvPenShootout {
			hasAETOrPen = true
			break
		}
	}
	for _, ev := range events {
		if ev.EventType == domain.EvFT && !hasAETOrPen {
			return true
		}
	}
	if !m.Status.Kickoff.IsZero() && time.Since(m.Status.Kickoff) > 150*time.Minute {
		return true
	}
	return false
}

func isMatchLive(m domain.Match) bool {
	if isMatchFinished(m, nil) {
		return false
	}
	return m.Status.State == domain.MatchLive
}

func computeMatchMinute(ko time.Time, added int) string {
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
		return fmt.Sprintf("45+%d", es)
	case em < 60+added:
		return "HT"
	case em < 90+15+added:
		sm := em - 15 - added
		if sm < 90 {
			return fmt.Sprintf("%d:%02d", sm, es)
		}
		return fmt.Sprintf("90+%d", sm-90)
	default:
		return fmt.Sprintf("ET %d+%d", em-105, es)
	}
}

func computeMatchMinuteFromEvents(ko, lastUpdate time.Time, minute int, added int) string {
	if minute <= 0 {
		return computeMatchMinute(ko, added)
	}

	phaseSec := int(time.Since(lastUpdate).Seconds())
	em := minute + phaseSec/60
	es := phaseSec % 60

	switch {
	case em <= 45:
		if em == 45 && es > 0 {
			return fmt.Sprintf("45+%d", es)
		}
		return fmt.Sprintf("%d:%02d", em, es)
	case em < 60:
		return "HT"
	case em <= 90:
		if em == 90 && es > 0 {
			return fmt.Sprintf("90+%d", es)
		}
		return fmt.Sprintf("%d:%02d", em, es)
	case em <= 105:
		if em == 105 && es > 0 {
			return fmt.Sprintf("105+%d", es)
		}
		return fmt.Sprintf("%d", em)
	case em <= 120:
		if em == 120 && es > 0 {
			return fmt.Sprintf("120+%d", es)
		}
		return fmt.Sprintf("%d", em)
	default:
		return fmt.Sprintf("%d", em)
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

func lastEventMinute(events []domain.MatchEvent) int {
	max := 0
	for _, ev := range events {
		switch ev.EventType {
		case domain.EvHalf, domain.EvHT, domain.EvFT, domain.EvKO, domain.EvS2, domain.EvAddedTime, domain.EvPausa, domain.EvContinua:
			continue
		}
		m := ev.SortTime
		if ev.SortOverload > 0 && (ev.SortTime == 45 || ev.SortTime == 90 || ev.SortTime == 105) {
			m += ev.SortOverload
		}
		if m > max {
			max = m
		}
	}
	return max
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
	lastAET := -1
	lastEventMinute := 0
	for i, ev := range sorted {
		if ev.EventType == domain.EvHalf && ev.HalfStr == "HT" {
			lastHT = i
		}
		if ev.EventType == domain.EvS2 {
			lastS2 = i
		}
		if ev.EventType == domain.EvAETStart {
			lastAET = i
		}
		if ev.SortTime > lastEventMinute {
			lastEventMinute = ev.SortTime
		}
	}
	if lastHT == -1 {
		return false
	}
	if lastAET > lastHT {
		return false
	}
	if lastS2 > lastHT {
		return false
	}
	return lastEventMinute < 60
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

func truncate(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	runes := []rune(s)
	var out strings.Builder
	width := 0
	for _, r := range runes {
		rw := lipgloss.Width(string(r))
		if width+rw > w-1 {
			break
		}
		out.WriteRune(r)
		width += rw
	}
	out.WriteRune('…')
	return out.String()
}

func formatMatch(m domain.Match) string {
	ko := m.Status.Kickoff
	timeStr := "--:--"
	if !ko.IsZero() {
		timeStr = ko.In(time.Local).Format("15:04")
	}

	// Col 1: live indicator
	var col1 string
	if isMatchLive(m) {
		col1 = liveIndicatorStyle.Render("●")
	} else {
		col1 = " "
	}

	// Col 2: time / minute
	var col2 string
	if isMatchLive(m) {
		minute := m.Status.Detail
		if minute == "" || minute == "En vivo" {
			minute = computeMatchMinute(ko, m.Status.FirstHalfAddedTime)
		}
		col2 = liveMinuteStyle.Render(minute)
	} else {
		col2 = matchTimeStyle.Render(timeStr)
	}

	const c1W = 2
	const c2W = 6
	const c3W = 25
	const c4W = 3
	const c5W = 25
	const c6W = 5

	// Col 3: home team
	col3 := matchStyle.Render(truncate(m.Home.Name, c3W))

	// Col 4: vs (always)
	col4 := vsStyle.Render("vs")

	// Col 5: away team
	col5 := matchStyle.Render(truncate(m.Away.Name, c5W))

	// Col 6: info extra (FT / HT / ET / PEN / minute)
	var col6 string
	if m.HomePenScore != nil && m.AwayPenScore != nil {
		col6 = matchScoreStyle.Render("PEN")
	} else if m.Status.State == domain.MatchFinished {
		if m.Status.Period >= domain.PeriodETFirstHalf {
			col6 = matchScoreStyle.Render("ET")
		} else {
			col6 = matchScoreStyle.Render("FT")
		}
	} else if isMatchLive(m) {
		d := m.Status.Detail
		if d == "HT" || d == "Descanso" {
			col6 = matchScoreStyle.Render("HT")
		} else {
			col6 = matchStyle.Render(d)
		}
	}

	// Col 7: score + penalties
	col7 := formatPenScore(m)

	return padRight(col1, c1W) +
		padRight(col2, c2W) +
		padRight(col3, c3W) +
		padCenter(col4, c4W) +
		padRight(col5, c5W) +
		padRight(col6, c6W) +
		col7
}

func formatPenScore(m domain.Match) string {
	if len(m.PenShootout) > 0 {
		score := fmt.Sprintf("%d : %d", *m.HomeScore, *m.AwayScore)
		score += " (" + formatPenShots(m.PenShootout) + ")"
		log.Printf("[score] PenShootout display: %s", score)
		return matchScoreStyle.Render(score)
	}
	if m.HomePenScore != nil && m.AwayPenScore != nil && m.HomeScore != nil && m.AwayScore != nil {
		score := fmt.Sprintf("%d : %d (%d : %d)", *m.HomeScore, *m.AwayScore, *m.HomePenScore, *m.AwayPenScore)
		log.Printf("[score] penalties display: %s (match=%q)", score, m.Home.Name)
		return matchScoreStyle.Render(score)
	}
	if m.HomeScore != nil && m.AwayScore != nil {
		log.Printf("[score] regular display: %d:%d (match=%q, homePen=%v awayPen=%v)",
			*m.HomeScore, *m.AwayScore, m.Home.Name, m.HomePenScore, m.AwayPenScore)
		return matchScoreStyle.Render(fmt.Sprintf("%d : %d", *m.HomeScore, *m.AwayScore))
	}
	if isMatchLive(m) {
		log.Printf("[score] live match %q %q state=%s scoreStr=%q homeScore=%v awayScore=%v",
			m.Home.Name, m.Away.Name, m.Status.State, m.Status.ScoreStr,
			m.HomeScore, m.AwayScore)
		return matchScoreStyle.Render("0 : 0")
	}
	log.Printf("[score] no score for match %q (state=%s)", m.Home.Name, m.Status.State)
	return ""
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

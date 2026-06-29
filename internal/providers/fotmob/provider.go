package fotmob

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"tifo/internal/domain"
	"tifo/internal/resolver"
	oldFotmob "tifo/fotmob"
	"time"
)

// Provider implements providers.Provider using the FotMob API.
type Provider struct {
	svc      *oldFotmob.Service
	mr       *resolver.MatchResolver
	tr       *resolver.TeamResolver
	cr       *resolver.CompetitionResolver
	priority int
}

func NewProvider(svc *oldFotmob.Service, mr *resolver.MatchResolver, tr *resolver.TeamResolver, cr *resolver.CompetitionResolver) *Provider {
	return &Provider{
		svc:      svc,
		mr:       mr,
		tr:       tr,
		cr:       cr,
		priority: 0,
	}
}

func (p *Provider) Name() string      { return "fotmob" }
func (p *Provider) Priority() int     { return p.priority }
func (p *Provider) SetTimeout(d time.Duration) {}

func (p *Provider) Leagues(ctx context.Context, locale, country string) ([]domain.Competition, error) {
	oldLeagues, err := p.svc.GetPopularLeagues(locale, country)
	if err != nil {
		return nil, fmt.Errorf("fotmob leagues: %w", err)
	}

	out := make([]domain.Competition, 0, len(oldLeagues))
	for _, l := range oldLeagues {
		tifoID, err := p.cr.Resolve("fotmob", fmt.Sprintf("%d", l.ID))
		if err != nil {
			log.Printf("[fotmob] resolver comp: %v", err)
			continue
		}
		out = append(out, domain.Competition{
			TIFOID:       tifoID,
			ExternalIDs:  domain.ExternalIDs{{Provider: "fotmob", ID: fmt.Sprintf("%d", l.ID)}},
			Name:         l.Name,
			OriginalName: l.OriginalName,
			Country:      l.CCode,
		})
	}
	return out, nil
}

func (p *Provider) LeagueMatches(ctx context.Context, competitionID string) ([]domain.Match, error) {
	id, err := strconv.Atoi(competitionID)
	if err != nil {
		return nil, fmt.Errorf("invalid fotmob competition id: %s", competitionID)
	}

	oldMatches, err := p.svc.LeagueMatches(id)
	if err != nil {
		return nil, fmt.Errorf("fotmob league matches: %w", err)
	}

	out := make([]domain.Match, 0, len(oldMatches))
	for _, m := range oldMatches {
		dm, err := p.mapMatch(m)
		if err != nil {
			log.Printf("[fotmob] map match %s: %v", m.ID, err)
			continue
		}
		out = append(out, *dm)
	}
	return out, nil
}

func (p *Provider) MatchDetails(ctx context.Context, matchID string) (*domain.MatchDetails, error) {
	oldDetails, err := p.svc.MatchDetailsByID(matchID)
	if err != nil {
		return nil, fmt.Errorf("fotmob details: %w", err)
	}
	return p.mapMatchDetails(oldDetails)
}

func (p *Provider) mapMatch(m oldFotmob.LeagueMatch) (*domain.Match, error) {
	tifoID, err := p.mr.Resolve("fotmob", m.ID)
	if err != nil {
		return nil, err
	}

	homeTifoID, err := p.tr.Resolve("fotmob", fmt.Sprintf("%v", m.Home.ID))
	if err != nil {
		return nil, err
	}
	awayTifoID, err := p.tr.Resolve("fotmob", fmt.Sprintf("%v", m.Away.ID))
	if err != nil {
		return nil, err
	}

	ko := parseUTCTime(m.Status.UTCTime)
	homeScore := parseIntPtr(m.Status.ScoreStr, true)
	awayScore := parseIntPtr(m.Status.ScoreStr, false)

	log.Printf("[fotmob] mapMatch %q vs %q: started=%v finished=%v scoreStr=%q homeScore=%v awayScore=%v",
		m.Home.Name, m.Away.Name, m.Status.Started, m.Status.Finished,
		m.Status.ScoreStr, homeScore, awayScore)

	return &domain.Match{
		TIFOID:      tifoID,
		ExternalIDs: domain.ExternalIDs{{Provider: "fotmob", ID: m.ID}},
		Home: domain.TeamRef{
			TIFOID: homeTifoID,
			Name:   m.Home.Name,
		},
		Away: domain.TeamRef{
			TIFOID: awayTifoID,
			Name:   m.Away.Name,
		},
		HomeScore: homeScore,
		AwayScore: awayScore,
		Status: domain.MatchStatus{
			State:    mapFotmobStatus(m.Status.Started, m.Status.Finished),
			Detail:   func() string { if m.Status.Finished { return "Finalizado" }; if m.Status.Started { return "En vivo" }; return "" }(),
			ScoreStr: m.Status.ScoreStr,
			UTCTime:  m.Status.UTCTime,
			Kickoff:  ko,
		},
	}, nil
}

func (p *Provider) mapMatchDetails(d *oldFotmob.MatchDetailsResponse) (*domain.MatchDetails, error) {
	matchIDStr := d.General.MatchID
	tifoID, err := p.mr.Resolve("fotmob", matchIDStr)
	if err != nil {
		return nil, err
	}

	md := &domain.MatchDetails{
		TIFOID: tifoID,
		ExternalIDs: domain.ExternalIDs{
			{Provider: "fotmob", ID: matchIDStr},
		},
		Match: domain.MatchRef{
			TIFOID: tifoID,
			Home:   d.General.HomeTeam.Name,
			Away:   d.General.AwayTeam.Name,
		},
		ExtraInfo: domain.MatchExtraInfo{
			HomeColor: d.General.TeamColors.LightMode.Home,
			AwayColor: d.General.TeamColors.LightMode.Away,
		},
	}

	if len(d.Header.Teams) >= 2 {
		md.Match.Score = fmt.Sprintf("%d-%d", d.Header.Teams[0].Score, d.Header.Teams[1].Score)
	}

	// Lineups
	md.Lineups = p.mapLineups(d)

	// Events
	md.Events = p.mapEvents(d)

	// Statistics
	md.Statistics = p.mapStats(d)

	// H2H
	md.H2H = p.mapH2H(d)

	// Injuries
	md.Injuries = p.mapInjuries(d)

	// Shotmap
	md.ShotMap = p.mapShotmap(d)

	return md, nil
}

func (p *Provider) mapLineups(d *oldFotmob.MatchDetailsResponse) *domain.Lineups {
	if d.Content.Lineup.HomeTeam.Formation == "" && d.Content.Lineup.AwayTeam.Formation == "" {
		return nil
	}
	lu := &domain.Lineups{
		HomeFormation: d.Content.Lineup.HomeTeam.Formation,
		AwayFormation: d.Content.Lineup.AwayTeam.Formation,
		HomeCoach:     d.Content.Lineup.HomeTeam.Coach.Name,
		AwayCoach:     d.Content.Lineup.AwayTeam.Coach.Name,
	}
	for _, p := range d.Content.Lineup.HomeTeam.Starters {
		posName := p.Position
		if posName == "" { posName = p.Role }
		lu.HomeStarters = append(lu.HomeStarters, domain.PlayerRef{Name: p.Name, Number: p.ShirtNumber, PosID: p.PositionID, PosName: posName})
	}
	for _, p := range d.Content.Lineup.HomeTeam.Subs {
		posName := p.Position
		if posName == "" { posName = p.Role }
		lu.HomeSubs = append(lu.HomeSubs, domain.PlayerRef{Name: p.Name, Number: p.ShirtNumber, PosID: p.PositionID, PosName: posName})
	}
	for _, p := range d.Content.Lineup.AwayTeam.Starters {
		posName := p.Position
		if posName == "" { posName = p.Role }
		lu.AwayStarters = append(lu.AwayStarters, domain.PlayerRef{Name: p.Name, Number: p.ShirtNumber, PosID: p.PositionID, PosName: posName})
	}
	for _, p := range d.Content.Lineup.AwayTeam.Subs {
		posName := p.Position
		if posName == "" { posName = p.Role }
		lu.AwaySubs = append(lu.AwaySubs, domain.PlayerRef{Name: p.Name, Number: p.ShirtNumber, PosID: p.PositionID, PosName: posName})
	}
	return lu
}

func (p *Provider) mapEvents(d *oldFotmob.MatchDetailsResponse) []domain.MatchEvent {
	var events []domain.MatchEvent

	// Find max added time per half to place HT/FT after all stoppage-time events
	maxAddedHT, maxAddedFT := 0, 0
	for _, ev := range d.Content.MatchFacts.Events.Events {
		if ev.OverloadTime != nil && *ev.OverloadTime > 0 {
			if ev.Time == 45 && *ev.OverloadTime > maxAddedHT {
				maxAddedHT = *ev.OverloadTime
			}
			if ev.Time == 90 && *ev.OverloadTime > maxAddedFT {
				maxAddedFT = *ev.OverloadTime
			}
		}
	}

	for _, ev := range d.Content.MatchFacts.Events.Events {
		player := playerRef(ev.Player)
		minute := ev.Time
		overload := 0
		if ev.OverloadTime != nil && *ev.OverloadTime > 0 {
			overload = *ev.OverloadTime
		}
		if ev.Type == "Half" {
			if ev.HalfStrShort == "HT" {
				overload = maxAddedHT + 1
			} else if ev.HalfStrShort == "FT" {
				overload = maxAddedFT + 1
			}
		}
		eventType := domain.EventType(ev.Type)
		if ev.Type == "Half" && ev.HalfStrShort == "FT" {
			eventType = domain.EvFT
		}
		team := domain.SideHome
		if !ev.IsHome {
			team = domain.SideAway
		}
		detail := ""
		subOut, subIn := (*domain.PlayerRef)(nil), (*domain.PlayerRef)(nil)
		if len(ev.Swap) >= 2 {
			subOut = &domain.PlayerRef{Name: ev.Swap[1].Name}
			subIn = &domain.PlayerRef{Name: ev.Swap[0].Name}
		}
		addedTime := 0
		if ev.MinutesAddedInput > 0 {
			addedTime = ev.MinutesAddedInput
		}
		ownGoal := ev.OwnGoal != nil

		events = append(events, domain.MatchEvent{
			Minute:       minute,
			AddedTime:    addedTime,
			EventType:    eventType,
			Team:         team,
			Player:       player,
			HomeScore:    ev.HomeScore,
			AwayScore:    ev.AwayScore,
			CardType:     ev.Card,
			Detail:       detail,
			SubOut:       subOut,
			SubIn:        subIn,
			GoalDesc:     ev.GoalDescription,
			OwnGoal:      ownGoal,
			HalfStr:      ev.HalfStrShort,
			SortTime:     ev.Time,
			SortOverload: overload,
		})
	}

	// Add shotmap as Shot events
	for _, s := range d.Content.Shotmap.Shots {
		isHome := teamIDMatch(d.General.HomeTeam.ID, s.TeamID)
		team := domain.SideAway
		if isHome {
			team = domain.SideHome
		}
		overload := 0
		if s.MinAdded != nil && *s.MinAdded > 0 {
			overload = *s.MinAdded
		}
		shotDesc := s.EventType
		switch s.EventType {
		case "Goal":
			shotDesc = "gol"
		case "AttemptSaved":
			shotDesc = "atajado"
		case "Miss":
			shotDesc = "falló"
		}
		events = append(events, domain.MatchEvent{
			Minute:       s.Min,
			AddedTime:    overload,
			EventType:    domain.EvShot,
			Team:         team,
			Player:       &domain.PlayerRef{Name: s.PlayerName},
			ShotDesc:     shotDesc,
			SortTime:     s.Min,
			SortOverload: overload,
		})
	}

	sort.Slice(events, func(i, j int) bool {
		ti, tj := events[i].SortTime, events[j].SortTime
		if ti == tj {
			return events[i].SortOverload < events[j].SortOverload
		}
		return ti < tj
	})

	return events
}

func (p *Provider) mapStats(d *oldFotmob.MatchDetailsResponse) []domain.StatCategory {
	var out []domain.StatCategory
	for _, cat := range d.Content.Stats.Periods.All.Stats {
		sc := domain.StatCategory{Title: cat.Title, Key: cat.Key}
		for _, s := range cat.Stats {
			homeVal := ""
			if len(s.Stats) > 0 && s.Stats[0] != nil {
				homeVal = fmt.Sprintf("%v", s.Stats[0])
			}
			awayVal := ""
			if len(s.Stats) > 1 && s.Stats[1] != nil {
				awayVal = fmt.Sprintf("%v", s.Stats[1])
			}
			sc.Stats = append(sc.Stats, domain.StatRow{
				Label: s.Title,
				Key:   s.Key,
				Home:  homeVal,
				Away:  awayVal,
			})
		}
		out = append(out, sc)
	}
	return out
}

func (p *Provider) mapH2H(d *oldFotmob.MatchDetailsResponse) *domain.H2H {
	if len(d.Content.H2H.Summary) < 3 {
		return nil
	}
	out := &domain.H2H{
		HomeWins: d.Content.H2H.Summary[0],
		Draws:    d.Content.H2H.Summary[1],
		AwayWins: d.Content.H2H.Summary[2],
	}
	for _, m := range d.Content.H2H.Matches {
		hs, as := 0, 0
		if m.Status.ScoreStr != "" {
			parts := strings.Split(m.Status.ScoreStr, "-")
			if len(parts) == 2 {
				hs, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
				as, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
			}
		}
		comp := m.League.Name
		if comp == "Friendlies" {
			comp = "FRIENDLY"
		}
		out.Matches = append(out.Matches, domain.H2HMatchDetail{
			Date:        parseUTCTime(m.Status.UTCTime),
			HomeTeam:    m.Home.Name,
			AwayTeam:    m.Away.Name,
			HomeScore:   hs,
			AwayScore:   as,
			Competition: comp,
		})
	}
	return out
}

func (p *Provider) mapInjuries(d *oldFotmob.MatchDetailsResponse) []domain.InjuryItem {
	var out []domain.InjuryItem

	for _, pl := range d.Content.Lineup.HomeTeam.Unavailable {
		out = append(out, domain.InjuryItem{
			Player: domain.PlayerRef{Name: pl.Name},
			Type:   pl.Unavailability.Type,
			Return: pl.Unavailability.ExpectedReturn,
			Team:   domain.SideHome,
		})
	}
	for _, pl := range d.Content.Lineup.AwayTeam.Unavailable {
		out = append(out, domain.InjuryItem{
			Player: domain.PlayerRef{Name: pl.Name},
			Type:   pl.Unavailability.Type,
			Return: pl.Unavailability.ExpectedReturn,
			Team:   domain.SideAway,
		})
	}
	return out
}

func (p *Provider) mapShotmap(d *oldFotmob.MatchDetailsResponse) []domain.Shot {
	var out []domain.Shot
	for _, s := range d.Content.Shotmap.Shots {
		isHome := teamIDMatch(d.General.HomeTeam.ID, s.TeamID)
		team := domain.SideAway
		if isHome {
			team = domain.SideHome
		}
		addedTime := 0
		if s.MinAdded != nil {
			addedTime = *s.MinAdded
		}
		out = append(out, domain.Shot{
			Minute:    s.Min,
			AddedTime: addedTime,
			Player:    s.PlayerName,
			Team:      team,
			EventType: s.EventType,
		})
	}
	return out
}

func playerRef(ev *oldFotmob.EventPlayer) *domain.PlayerRef {
	if ev == nil {
		return nil
	}
	return &domain.PlayerRef{Name: ev.Name}
}

func parseUTCTime(utc string) time.Time {
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, utc); err == nil {
			return t
		}
	}
	return time.Time{}
}

func parseIntPtr(scoreStr string, home bool) *int {
	if scoreStr == "" {
		return nil
		}
	parts := strings.Split(scoreStr, "-")
	if len(parts) != 2 {
		return nil
	}
	val := parts[0]
	if !home {
		val = parts[1]
	}
	n, err := strconv.Atoi(strings.TrimSpace(val))
	if err != nil {
		return nil
	}
	return &n
}

func mapFotmobStatus(live, finished bool) domain.MatchState {
	switch {
	case finished:
		return domain.MatchFinished
	case live:
		return domain.MatchLive
	default:
		return domain.MatchScheduled
	}
}

func teamIDMatch(id interface{}, teamID int) bool {
	switch v := id.(type) {
	case int:
		return v == teamID
	case float64:
		return int(math.Round(v)) == teamID
	case string:
		n, err := strconv.Atoi(v)
		return err == nil && n == teamID
	}
	return false
}

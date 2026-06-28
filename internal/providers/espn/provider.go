package espn

import (
	"context"
	"fmt"
	"strconv"
	"tifo/internal/domain"
	"tifo/internal/resolver"
	oldESPN "tifo/espn"
	"time"
)

// Provider implements providers.Provider and providers.EnrichmentProvider.
type Provider struct {
	svc *oldESPN.Service
	mr  *resolver.MatchResolver
	tr  *resolver.TeamResolver
}

func NewProvider(svc *oldESPN.Service) *Provider {
	return &Provider{svc: svc}
}

func (p *Provider) Name() string  { return "espn" }
func (p *Provider) Priority() int { return 10 }

func (p *Provider) Leagues(_ context.Context, _, _ string) ([]domain.Competition, error) {
	return nil, nil
}

func (p *Provider) LeagueMatches(_ context.Context, _ string) ([]domain.Match, error) {
	return nil, nil
}

func (p *Provider) MatchDetails(_ context.Context, _ string) (*domain.MatchDetails, error) {
	return nil, fmt.Errorf("espn: MatchDetails not supported")
}

// EnrichMatch enriches FotMob match details with ESPN data.
func (p *Provider) EnrichMatch(matchID int, leagueName string, utcTime time.Time, homeTeam, awayTeam string, fotmobDetails *domain.MatchDetails) *domain.MatchDetails {
	if fotmobDetails == nil {
		return nil
	}

	data, err := p.svc.FetchMatch(matchID, leagueName, utcTime, homeTeam, awayTeam)
	if err != nil {
		return fotmobDetails
	}

	out := *fotmobDetails

	// Venue
	if data.Summary.GameInfo.Venue.FullName != "" {
		out.ExtraInfo.Venue = data.Summary.GameInfo.Venue.FullName
		city := data.Summary.GameInfo.Venue.Address.City
		country := data.Summary.GameInfo.Venue.Address.Country
		if city != "" && country != "" {
			out.ExtraInfo.Venue = fmt.Sprintf("%s, %s, %s", data.Summary.GameInfo.Venue.FullName, city, country)
		} else if city != "" {
			out.ExtraInfo.Venue = fmt.Sprintf("%s, %s", data.Summary.GameInfo.Venue.FullName, city)
		}
	}

	// Attendance
	out.ExtraInfo.Attendance = data.Summary.GameInfo.Attendance

	// Referee
	for _, off := range data.Summary.GameInfo.Officials {
		if off.Position.ID == "1" {
			out.ExtraInfo.Referee = off.DisplayName
			break
		}
	}

	// Weather
	if data.Summary.GameInfo.Weather.DisplayValue != "" {
		out.ExtraInfo.Weather = data.Summary.GameInfo.Weather.DisplayValue
	} else if data.Summary.GameInfo.Weather.Condition != "" {
		w := data.Summary.GameInfo.Weather.Condition
		temp := data.Summary.GameInfo.Weather.Temperature
		if temp == 0 {
			temp = data.Summary.GameInfo.Weather.High
		}
		if temp > 0 {
			w = fmt.Sprintf("%s, %d°C", w, temp)
		}
		out.ExtraInfo.Weather = w
	}

	// Broadcasts
	for _, b := range data.Summary.Broadcasts {
		if b.Media != nil && b.Media.Name != "" {
			out.ExtraInfo.Broadcasts = append(out.ExtraInfo.Broadcasts, b.Media.Name)
		}
	}

	// Team colors (fallback)
	for _, comp := range data.Summary.Header.Competitions {
		for _, c := range comp.Competitors {
			if c.HomeAway == "home" {
				if out.ExtraInfo.HomeColor == "" {
					out.ExtraInfo.HomeColor = c.Team.Color
				}
			} else {
				if out.ExtraInfo.AwayColor == "" {
					out.ExtraInfo.AwayColor = c.Team.Color
				}
			}
		}
	}

	// Extra events (kickoff, halftime, etc.)
	if fotmobDetails.Events == nil {
		fotmobDetails.Events = []domain.MatchEvent{}
	}
	out.Events = p.mapExtraEvents(data, fotmobDetails.Events, homeTeam, awayTeam)

	// Stats fill
	out.Statistics = p.mapStats(data, fotmobDetails.Statistics)

	return &out
}

func (p *Provider) mapExtraEvents(data *oldESPN.EnrichData, existing []domain.MatchEvent, homeTeam, awayTeam string) []domain.MatchEvent {
	hash := make(map[string]bool)
	for _, ev := range existing {
		key := fmt.Sprintf("%d:%d:%s", ev.Minute, ev.SortOverload, string(ev.EventType))
		hash[key] = true
	}

	for _, ke := range data.Summary.KeyEvents {
		typ := classify(ke)
		if typ == "" {
			continue
		}
		minute, added := parseClock(ke.Clock.DisplayValue)

		// Skip if already exists from FotMob
		key := fmt.Sprintf("%d:%d:%s", minute, added, string(typ))
		if hash[key] {
			continue
		}

		var teamSide string
		if ke.Team != nil {
			teamSide = ke.Team.DisplayName
		}

		desc := ke.Text
		if desc == "" {
			desc = typeLabel(typ)
		}
		if teamSide != "" && !stringsContains(desc, teamSide) {
			// Add team name to description
			desc = teamSide + " — " + desc
		}

		existing = append(existing, domain.MatchEvent{
			Minute:       minute,
			AddedTime:    added,
			EventType:    typ,
			Detail:       desc,
			SortTime:     minute,
			SortOverload: added,
		})
		hash[key] = true
	}

	return existing
}

func (p *Provider) mapStats(data *oldESPN.EnrichData, existing []domain.StatCategory) []domain.StatCategory {
	if len(data.Summary.Boxscore.Teams) < 2 {
		return existing
	}

	// Build ESPN stat map: key → [home, away]
	espnByName := make(map[string][2]string)
	homeStats := data.Summary.Boxscore.Teams[0].Statistics
	awayStats := data.Summary.Boxscore.Teams[1].Statistics
	for i, hs := range homeStats {
		hVal := normalizeESPNVal(hs.Name, hs.DisplayValue)
		aVal := ""
		if i < len(awayStats) && awayStats[i].Name == hs.Name {
			aVal = normalizeESPNVal(hs.Name, awayStats[i].DisplayValue)
		}
		espnByName[hs.Name] = [2]string{hVal, aVal}
	}

	// Map ESPN names → FotMob keys
	espnFotmobKey := map[string]string{
		"possessionPct":     "Ball possession",
		"totalShots":        "Total shots",
		"shotsOnTarget":     "Shots on target",
		"offsides":          "Offsides",
		"wonCorners":        "Corner kicks",
		"yellowCards":       "Yellow cards",
		"redCards":          "Red cards",
		"foulsCommitted":    "Fouls",
		"saves":             "Saves",
		"totalPasses":       "Total passes",
		"accuratePasses":    "Accurate passes",
		"passPct":           "Pass accuracy",
		"totalCrosses":      "Total crosses",
		"accurateCrosses":   "Accurate crosses",
		"aerialsWon":        "Aerials won",
		"totalAerials":      "Total aerials",
		"aerialPct":         "Aerial success",
		"totalLongBalls":    "Long balls",
		"accurateLongBalls": "Accurate long balls",
		"tacklesTotal":      "Tackles",
		"interceptions":     "Interceptions",
		"clearances":        "Clearances",
		"totalDuels":        "Duels",
		"duelsWonPct":       "Duels won",
		"goalKicks":         "Goal kicks",
		"throwIns":          "Throw-ins",
		"punches":           "Punches",
	}

	for espnName, fotmobKey := range espnFotmobKey {
		vals, ok := espnByName[espnName]
		if !ok {
			continue
		}
		// Find and fill matching stat category/row
		for ci, cat := range existing {
			for si, s := range cat.Stats {
				if s.Key == fotmobKey {
					if s.Home == "" {
						existing[ci].Stats[si].Home = vals[0]
						existing[ci].Stats[si].HomeProvider = "espn"
					}
					if s.Away == "" {
						existing[ci].Stats[si].Away = vals[1]
						existing[ci].Stats[si].AwayProvider = "espn"
					}
				}
			}
		}
	}

	return existing
}

func classify(ke oldESPN.KeyEvent) domain.EventType {
	switch ke.Type.Type {
	case "kickoff":
		return domain.EvKO
	case "halftime":
		return domain.EvHT
	case "start-2nd-half":
		return domain.EvS2
	case "start-delay":
		return domain.EvPausa
	case "end-delay":
		return domain.EvContinua
	default:
		return ""
	}
}

func parseClock(display string) (minute, added int) {
	if display == "" {
		return 0, 0
	}
	var m, a int
	n, err := fmt.Sscanf(display, "%d'+%d", &m, &a)
	if err == nil && n >= 1 {
		return m, a
	}
	n, err = fmt.Sscanf(display, "%d", &m)
	if err == nil && n == 1 {
		return m, 0
	}
	return 0, 0
}

func typeLabel(typ domain.EventType) string {
	switch typ {
	case domain.EvKO:
		return "Inicio del partido"
	case domain.EvS2:
		return "Inicio 2do tiempo"
	case domain.EvPausa:
		return "Pausa"
	case domain.EvContinua:
		return "Se reanuda"
	case domain.EvHT:
		return "Descanso"
	}
	return string(typ)
}

func normalizeESPNVal(name, val string) string {
	if val == "" {
		return val
	}
	pctStats := map[string]bool{
		"passPct": true, "shotPct": true, "crossPct": true,
		"longBallPct": true, "aerialPct": true, "duelsWonPct": true,
	}
	if pctStats[name] {
		var v float64
		if _, err := fmt.Sscanf(val, "%f", &v); err == nil && v < 1 {
			return strconv.Itoa(int(v * 100))
		}
	}
	return val
}

func stringsContains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

package enricher

import (
	"fmt"
	"strings"

	"tifo/espn"
	"tifo/fotmob"
)

type EnrichedMatch struct {
	Fotmob        *fotmob.MatchDetailsResponse
	ESPN          *espn.EnrichData
	ExtraEvents   []ExtraEvent
	Venue         string
	Attendance    int
	Referee       string
	Weather       string
	Broadcasts    []string
	HomeColor     string
	AwayColor     string
	HomeAltColor  string
	AwayAltColor  string
	ESPNStats     map[string][2]string // fotmob stat key → [home, away]
}

type ExtraEvent struct {
	Minute       int
	AddedTime    int
	Period       int
	EventType    string
	Description  string
	TeamSide     string
}

// fotmobStatKey maps ESPN stat names → FotMob stat keys
// usado para llenar stats nulas de FotMob con datos de ESPN
// ESPN % stats use 0-1 range, FotMob uses 0-100; multiply by 100 for display
var espnPctStats = map[string]bool{
	"passPct": true, "shotPct": true, "crossPct": true,
	"longBallPct": true, "aerialPct": true, "duelsWonPct": true,
}

var fotmobStatKey = map[string]string{
	"possessionPct":      "Ball possession",
	"totalShots":         "Total shots",
	"shotsOnTarget":      "Shots on target",
	"offsides":           "Offsides",
	"wonCorners":         "Corner kicks",
	"yellowCards":        "Yellow cards",
	"redCards":           "Red cards",
	"foulsCommitted":     "Fouls",
	"saves":              "Saves",
	"totalPasses":        "Total passes",
	"accuratePasses":     "Accurate passes",
	"passPct":            "Pass accuracy",
	"totalCrosses":       "Total crosses",
	"accurateCrosses":    "Accurate crosses",
	"crossPct":           "Cross accuracy",
	"totalLongBalls":     "Long balls",
	"accurateLongBalls":  "Accurate long balls",
	"longBallPct":        "Long ball accuracy",
	"aerialsWon":         "Aerials won",
	"totalAerials":       "Total aerials",
	"aerialPct":          "Aerial success",
	"tacklesTotal":       "Tackles",
	"interceptions":      "Interceptions",
	"clearances":         "Clearances",
	"totalDuels":         "Duels",
	"duelsWonPct":        "Duels won",
	"goalKicks":          "Goal kicks",
	"throwIns":           "Throw-ins",
	"punches":            "Punches",
}

const (
	EvKickoff    = "KO"
	EvDelayStart = "Pausa"
	EvDelayEnd   = "Continúa"
)

func Enrich(f *fotmob.MatchDetailsResponse, e *espn.EnrichData) *EnrichedMatch {
	em := &EnrichedMatch{
		Fotmob: f,
		ESPN:   e,
	}

	if e == nil || e.Summary == nil {
		return em
	}

	s := e.Summary

	if s.GameInfo.Venue.FullName != "" {
		em.Venue = s.GameInfo.Venue.FullName
		city := s.GameInfo.Venue.Address.City
		country := s.GameInfo.Venue.Address.Country
		if city != "" && country != "" {
			em.Venue = fmt.Sprintf("%s, %s, %s", s.GameInfo.Venue.FullName, city, country)
		} else if city != "" {
			em.Venue = fmt.Sprintf("%s, %s", s.GameInfo.Venue.FullName, city)
		}
	}

	em.Attendance = s.GameInfo.Attendance

	for _, off := range s.GameInfo.Officials {
		if em.Referee == "" && off.Position.ID == "1" {
			em.Referee = off.DisplayName
			break
		}
	}

	if s.GameInfo.Weather.DisplayValue != "" {
		em.Weather = s.GameInfo.Weather.DisplayValue
	} else if s.GameInfo.Weather.Condition != "" {
		w := s.GameInfo.Weather.Condition
		if s.GameInfo.Weather.Temperature > 0 || s.GameInfo.Weather.High > 0 {
			temp := s.GameInfo.Weather.Temperature
			if temp == 0 {
				temp = s.GameInfo.Weather.High
			}
			w = fmt.Sprintf("%s, %d°C", w, temp)
		}
		em.Weather = w
	}

	for _, b := range s.Broadcasts {
		if b.Media != nil && b.Media.Name != "" {
			em.Broadcasts = append(em.Broadcasts, b.Media.Name)
		}
	}

	for _, comp := range s.Header.Competitions {
		for _, c := range comp.Competitors {
			if c.HomeAway == "home" {
				em.HomeColor = c.Team.Color
				em.HomeAltColor = c.Team.AlternateColor
			} else {
				em.AwayColor = c.Team.Color
				em.AwayAltColor = c.Team.AlternateColor
			}
		}
	}

	em.extraEvents(s)
	em.parseStats(s)

	return em
}

func (em *EnrichedMatch) extraEvents(s *espn.SummaryResponse) {
	fotmobHalfSet := make(map[int]bool)
	if em.Fotmob != nil {
		for _, ev := range em.Fotmob.Content.MatchFacts.Events.Events {
			if ev.Type == "Half" {
				fotmobHalfSet[ev.Time] = true
			}
		}
	}

	for _, ke := range s.KeyEvents {
		typ := classifyESPNEvent(ke)
		if typ == "" {
			continue
		}

		isHalftime := ke.Type.Type == "halftime"
		isStart2nd := ke.Type.Type == "start-2nd-half"

		if isHalftime || isStart2nd {
			_, added := parseClock(ke.Clock.DisplayValue)
			timeMin := int(ke.Clock.Value / 60)
			if fotmobHalfSet[timeMin] || fotmobHalfSet[timeMin+added] {
				continue
			}
		}

		minute, added := parseClock(ke.Clock.DisplayValue)

		var teamSide string
		if ke.Team != nil {
			teamSide = ke.Team.DisplayName
		}

		desc := ke.Text
		if desc == "" {
			desc = typeLabel(typ, ke.Type.Type)
		}
		if teamSide != "" && !strings.Contains(desc, teamSide) {
			desc = fmt.Sprintf("%s — %s", teamSide, desc)
		}

		em.ExtraEvents = append(em.ExtraEvents, ExtraEvent{
			Minute:      minute,
			AddedTime:   added,
			Period:      ke.Period.Number,
			EventType:   typ,
			Description: desc,
			TeamSide:    teamSide,
		})
	}
}

func classifyESPNEvent(ke espn.KeyEvent) string {
	switch ke.Type.Type {
	case "kickoff":
		return EvKickoff
	case "halftime":
		return "HT"
	case "start-2nd-half":
		return "S2"
	case "start-delay":
		return EvDelayStart
	case "end-delay":
		return EvDelayEnd
	case "goal", "yellow-card", "red-card", "substitution":
		return ""
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

func typeLabel(typ, espnType string) string {
	switch espnType {
	case "kickoff":
		return "Inicio del partido"
	case "start-2nd-half":
		return "Inicio 2do tiempo"
	case "start-delay":
		return "Pausa"
	case "end-delay":
		return "Se reanuda"
	default:
		return typ
	}
}

// ShouldInject checks if an ESPN key event should be shown in the event list
// (it's not already in fotmob events and is one of the extra types)
func (em *EnrichedMatch) parseStats(s *espn.SummaryResponse) {
	if len(s.Boxscore.Teams) < 2 {
		return
	}

	espnByName := make(map[string][2]string)
	homeStats := s.Boxscore.Teams[0].Statistics
	awayStats := s.Boxscore.Teams[1].Statistics

	for i, hs := range homeStats {
		hVal := normalizeESPNVal(hs.Name, hs.DisplayValue)
		aVal := ""
		if i < len(awayStats) && awayStats[i].Name == hs.Name {
			aVal = normalizeESPNVal(hs.Name, awayStats[i].DisplayValue)
		}
		espnByName[hs.Name] = [2]string{hVal, aVal}
	}

	em.ESPNStats = make(map[string][2]string)
	for espnName, fotmobKey := range fotmobStatKey {
		if vals, ok := espnByName[espnName]; ok {
			em.ESPNStats[fotmobKey] = vals
		}
	}
}

// normalizeESPNVal converts ESPN values to FotMob-like format
func normalizeESPNVal(name, val string) string {
	if val == "" {
		return val
	}
	if espnPctStats[name] {
		var v float64
		if _, err := fmt.Sscanf(val, "%f", &v); err == nil && v < 1 {
			return fmt.Sprintf("%d", int(v*100))
		}
	}
	return val
}

func (em *EnrichedMatch) ShouldInject(espnEventType string) bool {
	for _, ev := range em.ExtraEvents {
		if ev.EventType == espnEventType {
			return true
		}
	}
	return false
}

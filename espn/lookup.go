package espn

import (
	"fmt"
	"log"
	"time"
)

type ResolvedEvent struct {
	ID         string
	LeagueSlug string
	HomeTeam   string
	AwayTeam   string
}

type LookupService struct {
	client *Client
	cache  *IDCache
}

func NewLookupService(client *Client, cache *IDCache) *LookupService {
	return &LookupService{client: client, cache: cache}
}

func parseESPNDate(s string) (time.Time, error) {
	layouts := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04Z",
		"2006-01-02T15:04:05.000Z",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("no se pudo parsear fecha ESPN: %s", s)
}

func (s *LookupService) Resolve(fotmobID int, leagueName string, utcTime time.Time, homeTeam, awayTeam string) (*ResolvedEvent, error) {
	if id, ok := s.cache.Get(fotmobID); ok {
		slug := FotmobLeagueToESPN(leagueName)
		log.Printf("[ESPN] cache hit fotmobID=%d → eventID=%s league=%s", fotmobID, id, slug)
		return &ResolvedEvent{ID: id, LeagueSlug: slug}, nil
	}

	log.Printf("[ESPN] Resolve fotmobID=%d league=%q utc=%s home=%q away=%q", fotmobID, leagueName, utcTime.Format(time.RFC3339), homeTeam, awayTeam)
	slug := FotmobLeagueToESPN(leagueName)
	if slug == "" {
		log.Printf("[ESPN] no mapping for league=%q", leagueName)
		return nil, fmt.Errorf("sin mapeo ESPN para: %s", leagueName)
	}
	log.Printf("[ESPN] league mapping: %q → %s", leagueName, slug)

	dates := []string{
		utcTime.AddDate(0, 0, -1).Format("20060102"),
		utcTime.Format("20060102"),
		utcTime.AddDate(0, 0, 1).Format("20060102"),
	}
	log.Printf("[ESPN] searching dates: %v", dates)

	var allEvents []ScoreboardEvent
	for _, d := range dates {
		sb, err := s.client.Scoreboard(slug, d)
		if err != nil {
			log.Printf("[ESPN] scoreboard %s/%s error: %v", slug, d, err)
			continue
		}
		log.Printf("[ESPN] scoreboard %s/%s → %d events", slug, d, len(sb.Events))
		allEvents = append(allEvents, sb.Events...)
	}
	log.Printf("[ESPN] total %d events across all dates", len(allEvents))

	ev := FindEventByMatch(allEvents, utcTime, homeTeam, awayTeam)
	if ev == nil {
		log.Printf("[ESPN] no match found for %s vs %s (normalized: %s vs %s)", homeTeam, awayTeam, NormalizeTeamName(homeTeam), NormalizeTeamName(awayTeam))
		// Log first 5 events for debugging
		for i, e := range allEvents {
			if i >= 5 {
				break
			}
			for _, comp := range e.Competitions {
				for _, c := range comp.Competitors {
					log.Printf("[ESPN] event %s: %s=%s %s=%s (normalized: %s vs %s)", e.ID, c.HomeAway, c.Team.DisplayName, c.HomeAway, c.Team.DisplayName, NormalizeTeamName(c.Team.DisplayName), NormalizeTeamName(c.Team.DisplayName))
				}
			}
		}
		return nil, fmt.Errorf("no se encontró evento ESPN para %s vs %s", homeTeam, awayTeam)
	}

	var homeTeamName, awayTeamName string
	for _, comp := range ev.Competitions {
		for _, c := range comp.Competitors {
			if c.HomeAway == "home" {
				homeTeamName = c.Team.DisplayName
			} else {
				awayTeamName = c.Team.DisplayName
			}
		}
	}

	log.Printf("[ESPN] found event %s: %s vs %s (ESPN names: %s vs %s)", ev.ID, homeTeam, awayTeam, homeTeamName, awayTeamName)
	s.cache.Set(fotmobID, ev.ID)
	return &ResolvedEvent{
		ID:         ev.ID,
		LeagueSlug: slug,
		HomeTeam:   homeTeamName,
		AwayTeam:   awayTeamName,
	}, nil
}

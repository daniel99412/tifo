package sofascore

import (
	"fmt"
	"log"
	"strings"
	"time"
)

type LookupService struct {
	client *Client
}

func NewLookupService(client *Client) *LookupService {
	return &LookupService{client: client}
}

func (s *LookupService) Resolve(homeTeam, awayTeam string, utcTime time.Time) (int, error) {
	homeID, err := s.findTeamID(homeTeam)
	if err != nil {
		return 0, fmt.Errorf("buscar equipo local %q: %w", homeTeam, err)
	}

	events, err := s.client.GetTeamLastEvents(homeID, 20)
	if err != nil {
		return 0, fmt.Errorf("eventos de %q: %w", homeTeam, err)
	}

	dateStr := utcTime.Format("2006-01-02")
	for _, ev := range events.Events {
		evDate := time.Unix(ev.StartTimestamp, 0).Format("2006-01-02")
		if evDate != dateStr {
			continue
		}

		opponent := ev.AwayTeam.Name
		if ev.HomeTeam.ID != homeID {
			opponent = ev.HomeTeam.Name
		}

		if strings.EqualFold(opponent, awayTeam) || strings.Contains(strings.ToLower(opponent), strings.ToLower(awayTeam)) {
			return ev.ID, nil
		}
	}

	return 0, fmt.Errorf("no se encontró evento Sofascore para %s vs %s en %s", homeTeam, awayTeam, dateStr)
}

func (s *LookupService) findTeamID(name string) (int, error) {
	results, err := s.client.SearchTeam(name)
	if err != nil {
		return 0, err
	}

	for _, r := range results {
		if r.Type == "team" && r.Entity.Slug != "" && !strings.Contains(r.Entity.Slug, "u21") && !strings.Contains(r.Entity.Slug, "u20") && !strings.Contains(r.Entity.Slug, "u19") && !strings.Contains(r.Entity.Slug, "u17") {
			nameLower := strings.ToLower(name)
			entityLower := strings.ToLower(r.Entity.Name)
			if strings.Contains(entityLower, nameLower) || strings.Contains(nameLower, entityLower) {
				log.Printf("[sofascore] encontrado teamID=%d name=%q para búsqueda %q", r.Entity.ID, r.Entity.Name, name)
				return r.Entity.ID, nil
			}
		}
	}

	return 0, fmt.Errorf("team %q no encontrado", name)
}

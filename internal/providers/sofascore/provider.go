package sofascore

import (
	"context"
	"fmt"
	"log"
	"tifo/internal/domain"
	"time"
)

type Provider struct {
	client *Client
	lookup *LookupService
}

func NewProvider() *Provider {
	c := NewClient()
	return &Provider{
		client: c,
		lookup: NewLookupService(c),
	}
}

func (p *Provider) Name() string                     { return "sofascore" }
func (p *Provider) Priority() int                    { return 20 }
func (p *Provider) Leagues(_ context.Context, _, _ string) ([]domain.Competition, error) {
	return nil, nil
}
func (p *Provider) LeagueMatches(_ context.Context, _ string) ([]domain.Match, error) {
	return nil, nil
}
func (p *Provider) MatchDetails(_ context.Context, _ string) (*domain.MatchDetails, error) {
	return nil, fmt.Errorf("sofascore: MatchDetails not supported")
}

func (p *Provider) EnrichMatch(matchID int, leagueName string, utcTime time.Time, homeTeam, awayTeam string, fotmobDetails *domain.MatchDetails) *domain.MatchDetails {
	if fotmobDetails == nil {
		return nil
	}

	eventID, err := p.lookup.Resolve(homeTeam, awayTeam, utcTime)
	if err != nil {
		log.Printf("[sofascore] lookup: %v", err)
		return fotmobDetails
	}

	incidents, err := p.client.GetMatchIncidents(eventID)
	if err != nil {
		log.Printf("[sofascore] incidents: %v", err)
		return fotmobDetails
	}

	out := *fotmobDetails

	if out.Events == nil {
		out.Events = []domain.MatchEvent{}
	}

	// Hash existing events by minute (without type — we override pause-like events)
	eventsByMinute := make(map[int]int)
	for i, ev := range out.Events {
		key := ev.Minute
		if ev.AddedTime > 0 {
			key = key*1000 + ev.AddedTime
		}
		eventsByMinute[key] = i
	}

	homeName := fotmobDetails.Match.Home
	awayName := fotmobDetails.Match.Away

	for _, inc := range incidents.Incidents {
		ev, ok := p.mapIncident(inc, homeName, awayName)
		if !ok {
			continue
		}

		key := ev.Minute
		if ev.AddedTime > 0 {
			key = key*1000 + ev.AddedTime
		}

		// If Sofascore has a VAR event and ESPN already put a pause/review at same minute, replace it
		if ev.EventType == domain.EvVAR || ev.EventType == domain.EvVideoReview {
			if idx, exists := eventsByMinute[key]; exists {
				existingType := out.Events[idx].EventType
				if existingType == domain.EvPausa || existingType == domain.EvVideoReview || existingType == domain.EvVAR || existingType == domain.EvInjury || existingType == domain.EvContinua {
					out.Events[idx] = ev
					continue
				}
			}
		}

		// Dedup by minute:type for non-VAR events
		if _, exists := eventsByMinute[key]; exists {
			continue
		}

		out.Events = append(out.Events, ev)
		eventsByMinute[key] = len(out.Events) - 1
	}

	// Update CONT descriptions that follow VAR with SofaScore's specific detail
	var lastVARDetail string
	for i := range out.Events {
		switch out.Events[i].EventType {
		case domain.EvVAR, domain.EvVideoReview:
			if out.Events[i].Detail != "" {
				lastVARDetail = out.Events[i].Detail
			}
		case domain.EvContinua:
			if lastVARDetail != "" {
				out.Events[i].Detail = "Se reanuda después de " + lastVARDetail
				lastVARDetail = ""
			}
		case domain.EvPausa, domain.EvInjury:
			lastVARDetail = ""
		}
	}

	return &out
}

func (p *Provider) mapIncident(inc Incident, homeName, awayName string) (domain.MatchEvent, bool) {
	minute := inc.Time
	if minute < 0 {
		minute = 0
	}

	period := domain.PeriodSecondHalf
	if minute <= 45 {
		period = domain.PeriodFirstHalf
	}

	team := domain.SideHome
	if inc.IsHome != nil && !*inc.IsHome {
		team = domain.SideAway
	}

	detail := ""
	eventType := domain.EventType("")
	addedTime := 0

	switch inc.IncidentType {
	case "varDecision":
		eventType = domain.EvVAR
		switch inc.IncidentClass {
		case "cardUpgrade":
			detail = "VAR: tarjeta subida a roja"
		case "penaltyNotAwarded":
			detail = "VAR: penal no concedido"
		case "penaltyAwarded":
			detail = "VAR: penal concedido"
		case "goal":
			eventType = domain.EvVideoReview
			detail = "VAR: gol confirmado"
		case "goalCancelled":
			detail = "VAR: gol anulado"
		case "offside":
			detail = "VAR: fuera de juego"
		default:
			detail = fmt.Sprintf("VAR: %s", inc.IncidentClass)
		}

	default:
		return domain.MatchEvent{}, false
	}

	if eventType == "" {
		return domain.MatchEvent{}, false
	}

	var player *domain.PlayerRef
	if inc.Player != nil && inc.Player.Name != "" {
		player = &domain.PlayerRef{
			Name: inc.Player.Name,
		}
	} else if inc.PlayerName != "" {
		player = &domain.PlayerRef{
			Name: inc.PlayerName,
		}
	}

	return domain.MatchEvent{
		Minute:       minute,
		AddedTime:    addedTime,
		EventType:    eventType,
		Team:         team,
		Player:       player,
		Detail:       detail,
		Period:       period,
		SortTime:     minute,
		SortOverload: addedTime,
	}, true
}

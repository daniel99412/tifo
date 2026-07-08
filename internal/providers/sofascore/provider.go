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
	log.Printf("[sofascore] found eventID=%d with %d incidents", eventID, len(incidents.Incidents))

	out := *fotmobDetails

	if out.Events == nil {
		out.Events = []domain.MatchEvent{}
	}

	homeName := fotmobDetails.Match.Home
	awayName := fotmobDetails.Match.Away

	for _, inc := range incidents.Incidents {
		ev, ok := p.mapIncident(inc, homeName, awayName)
		if !ok {
			continue
		}

		// Try to match against existing events
		matched := false
		for i := range out.Events {
			if domain.IdentityMatch(out.Events[i], ev) {
				domain.MergeEvents(&out.Events[i], &ev)
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		// No match — check dedup by minute:type
		dup := false
		for _, existing := range out.Events {
			if existing.Minute == ev.Minute && existing.EventType == ev.EventType {
				dup = true
				break
			}
		}
		if dup {
			continue
		}

		out.Events = append(out.Events, ev)
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
		var varConfirmed *bool
		if inc.Confirmed != nil {
			varConfirmed = inc.Confirmed
		}
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

		var player *domain.PlayerRef
		if inc.Player != nil && inc.Player.Name != "" {
			player = &domain.PlayerRef{Name: inc.Player.Name}
		} else if inc.PlayerName != "" {
			player = &domain.PlayerRef{Name: inc.PlayerName}
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
			VarClass:     inc.IncidentClass,
			VarConfirmed: varConfirmed,
		}, true

	case "substitution":
		var subOut, subIn *domain.PlayerRef
		if inc.PlayerOut != nil && inc.PlayerOut.Name != "" {
			subOut = &domain.PlayerRef{Name: inc.PlayerOut.Name}
		}
		if inc.PlayerIn != nil && inc.PlayerIn.Name != "" {
			subIn = &domain.PlayerRef{Name: inc.PlayerIn.Name}
		}
		return domain.MatchEvent{
			Minute:       minute,
			AddedTime:    addedTime,
			EventType:    domain.EvSubstitution,
			Team:         team,
			Period:       period,
			SortTime:     minute,
			SortOverload: addedTime,
			SubOut:       subOut,
			SubIn:        subIn,
			GoalDesc:     inc.IncidentClass, // "regular" or "injury"
		}, true

	case "goal":
		return domain.MatchEvent{
			Minute:       minute,
			AddedTime:    addedTime,
			EventType:    domain.EvGoal,
			Team:         team,
			Period:       period,
			SortTime:     minute,
			SortOverload: addedTime,
			GoalDesc:     inc.IncidentClass, // "regular" or "penalty"
		}, true

	case "card":
		cardType := ""
		switch inc.IncidentClass {
		case "yellow":
			cardType = "Yellow"
		case "red":
			cardType = "Red"
		default:
			cardType = inc.IncidentClass
		}
		var player *domain.PlayerRef
		if inc.Player != nil && inc.Player.Name != "" {
			player = &domain.PlayerRef{Name: inc.Player.Name}
		} else if inc.PlayerName != "" {
			player = &domain.PlayerRef{Name: inc.PlayerName}
		}
		return domain.MatchEvent{
			Minute:       minute,
			AddedTime:    addedTime,
			EventType:    domain.EvCard,
			Team:         team,
			Player:       player,
			Period:       period,
			SortTime:     minute,
			SortOverload: addedTime,
			CardType:     cardType,
		}, true

	case "injuryTime":
		return domain.MatchEvent{
			Minute:       minute,
			AddedTime:    inc.Length,
			EventType:    domain.EvAddedTime,
			Period:       period,
			SortTime:     minute,
			SortOverload: inc.Length,
		}, true

	default:
		return domain.MatchEvent{}, false
	}
}

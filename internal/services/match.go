package services

import (
	"context"
	"fmt"
	"log"
	"time"
	"tifo/internal/domain"
	"tifo/internal/enrich"
	"tifo/internal/providers"
)

// MatchContext provides context needed for enrichment lookups.
type MatchContext struct {
	LeagueName string
	UTCTime    time.Time
	HomeTeam   string
	AwayTeam   string
}

// MatchService coordinates providers, resolvers, and enrichers for match data.
type MatchService struct {
	primary   providers.Provider
	enrichers []providers.Provider
	enricher  *enrich.Enricher
}

// NewMatchService creates a MatchService with a primary provider and optional enrichers.
func NewMatchService(primary providers.Provider, enrichers []providers.Provider, enrichCfg enrich.MergeConfig) *MatchService {
	return &MatchService{
		primary:   primary,
		enrichers: enrichers,
		enricher:  enrich.NewEnricher(enrichCfg),
	}
}

// Leagues returns competitions from the primary provider.
func (s *MatchService) Leagues(ctx context.Context, locale, country string) ([]domain.Competition, error) {
	return s.primary.Leagues(ctx, locale, country)
}

// LeagueMatches returns matches from the primary provider.
func (s *MatchService) LeagueMatches(ctx context.Context, competitionID string) ([]domain.Match, error) {
	return s.primary.LeagueMatches(ctx, competitionID)
}

// MatchDetails fetches details from primary and enriches with all enrichment providers.
func (s *MatchService) MatchDetails(ctx context.Context, matchID string, ctxProvider ...MatchContext) (*domain.MatchDetails, error) {
	details, err := s.primary.MatchDetails(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("primary match details: %w", err)
	}

	var matchCtx MatchContext
	if len(ctxProvider) > 0 {
		matchCtx = ctxProvider[0]
	}

	for _, ep := range s.enrichers {
		if ep.Name() == s.primary.Name() {
			continue
		}
		enriched := s.applyEnrichment(ep, details, matchCtx)
		if enriched != nil {
			details = enriched
		}
	}

	return details, nil
}

func (s *MatchService) applyEnrichment(ep providers.Provider, details *domain.MatchDetails, ctx MatchContext) *domain.MatchDetails {
	espnProvider, ok := ep.(interface {
		EnrichMatch(matchID int, leagueName string, utcTime time.Time, homeTeam, awayTeam string, fotmobDetails *domain.MatchDetails) *domain.MatchDetails
	})
	if !ok {
		return nil
	}

	// Extract FotMob ID from external IDs
	fotmobID := 0
	if id, ok := details.ExternalIDs.Get("fotmob"); ok {
		if _, err := fmt.Sscanf(id, "%d", &fotmobID); err != nil {
			log.Printf("[service] parse fotmob id %q: %v", id, err)
		}
	}

	homeTeam := ctx.HomeTeam
	awayTeam := ctx.AwayTeam
	if homeTeam == "" {
		homeTeam = details.Match.Home
	}
	if awayTeam == "" {
		awayTeam = details.Match.Away
	}

	return espnProvider.EnrichMatch(fotmobID, ctx.LeagueName, ctx.UTCTime, homeTeam, awayTeam, details)
}

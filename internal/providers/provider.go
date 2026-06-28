package providers

import (
	"context"
	"time"
	"tifo/internal/domain"
)

// Provider is the interface every football data provider must implement.
type Provider interface {
	// Name returns a short unique key (e.g. "fotmob", "espn").
	Name() string

	// Priority returns the provider's priority (lower = preferred).
	Priority() int

	// Leagues returns all leagues available from this provider.
	Leagues(ctx context.Context, locale, country string) ([]domain.Competition, error)

	// LeagueMatches returns all matches for a competition.
	LeagueMatches(ctx context.Context, competitionID string) ([]domain.Match, error)

	// MatchDetails returns full details for a single match.
	MatchDetails(ctx context.Context, matchID string) (*domain.MatchDetails, error)
}

// MatchTimelineProvider is an optional interface for providers that supply timeline events.
type MatchTimelineProvider interface {
	Timeline(ctx context.Context, matchID string) ([]domain.MatchEvent, error)
}

// StatProvider is an optional interface for providers that supply statistics.
type StatProvider interface {
	Statistics(ctx context.Context, matchID string) ([]domain.StatCategory, error)
}

// LineupProvider is an optional interface for providers that supply lineups.
type LineupProvider interface {
	Lineups(ctx context.Context, matchID string) (*domain.Lineups, error)
}

// ShotMapProvider is an optional interface for providers that supply shot maps.
type ShotMapProvider interface {
	ShotMap(ctx context.Context, matchID string) ([]domain.Shot, error)
}

// EnrichmentProvider is an optional interface for providers that supply supplemental info.
type EnrichmentProvider interface {
	Enrich(ctx context.Context, matchID string, details *domain.MatchDetails) error
}

// ProviderOption configures a provider instance.
type ProviderOption func(p Provider)

// WithTimeout sets the HTTP client timeout for providers.
func WithTimeout(d time.Duration) ProviderOption {
	return func(p Provider) {
		if c, ok := p.(interface{ SetTimeout(time.Duration) }); ok {
			c.SetTimeout(d)
		}
	}
}

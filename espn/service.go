package espn

import "time"

type Service struct {
	client  *Client
	cache   *IDCache
	lookup  *LookupService
}

func NewService() *Service {
	c := NewClient()
	cache := NewIDCache()
	return &Service{
		client: c,
		cache:  cache,
		lookup: NewLookupService(c, cache),
	}
}

type EnrichData struct {
	EventID      string
	LeagueSlug   string
	HomeTeam     string
	AwayTeam     string
	Summary      *SummaryResponse
}

func (s *Service) FetchMatch(fotmobID int, leagueName string, utcTime time.Time, homeTeam, awayTeam string) (*EnrichData, error) {
	resolved, err := s.lookup.Resolve(fotmobID, leagueName, utcTime, homeTeam, awayTeam)
	if err != nil {
		return nil, err
	}

	summary, err := s.client.Summary(resolved.LeagueSlug, resolved.ID)
	if err != nil {
		return nil, err
	}

	return &EnrichData{
		EventID:    resolved.ID,
		LeagueSlug: resolved.LeagueSlug,
		HomeTeam:   resolved.HomeTeam,
		AwayTeam:   resolved.AwayTeam,
		Summary:    summary,
	}, nil
}

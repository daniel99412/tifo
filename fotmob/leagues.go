package fotmob

import "fmt"

type GroupedLeague struct {
	ID           int
	Name         string
	OriginalName string
	CCode        string
}

func (s *Service) GetPopularLeagues(locale, country string) ([]GroupedLeague, error) {
	resp, err := s.client.GetAllLeagues(locale, country)
	if err != nil {
		return nil, fmt.Errorf("obtener ligas populares: %w", err)
	}

	out := make([]GroupedLeague, 0, len(resp.Popular))
	for _, l := range resp.Popular {
		name := l.Name
		if l.LocalizedName != "" {
			name = l.LocalizedName
		}
		out = append(out, GroupedLeague{
			ID:           l.ID,
			Name:         name,
			OriginalName: l.Name,
			CCode:        l.CCode,
		})
	}
	return out, nil
}

func (s *Service) GetAllLeaguesByCountry(locale, country string) ([]GroupedLeague, error) {
	resp, err := s.client.GetAllLeagues(locale, country)
	if err != nil {
		return nil, fmt.Errorf("obtener ligas: %w", err)
	}

	seen := make(map[int]bool)
	var out []GroupedLeague

	add := func(items []AllLeagueItem) {
		for _, l := range items {
			if seen[l.ID] {
				continue
			}
			seen[l.ID] = true
			name := l.Name
			if l.LocalizedName != "" {
				name = l.LocalizedName
			}
			out = append(out, GroupedLeague{ID: l.ID, Name: name, OriginalName: l.Name, CCode: l.CCode})
		}
	}

	add(resp.Popular)
	for _, group := range resp.International {
		add(group.Leagues)
	}
	for _, group := range resp.Countries {
		if country == "" || group.CCode == country {
			add(group.Leagues)
		}
	}

	return out, nil
}

func (s *Service) GetLeagueNames() (map[int]string, error) {
	mapping, err := s.client.GetTranslationMapping("en")
	if err != nil {
		return nil, fmt.Errorf("obtener ligas: %w", err)
	}
	out := make(map[int]string)
	for idStr, name := range mapping.TournamentPrefixes {
		var id int
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
			continue
		}
		out[id] = name
	}
	return out, nil
}

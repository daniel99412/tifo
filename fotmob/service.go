package fotmob

import (
	"fmt"
	"time"
)

type Service struct {
	client *Client
}

func NewService() *Service {
	return &Service{client: NewClient()}
}

func (s *Service) TodaysMatches() ([]LeagueMatches, error) {
	date := time.Now().Format("20060102")
	resp, err := s.client.GetMatches(date)
	if err != nil {
		return nil, err
	}
	return resp.Leagues, nil
}

func (s *Service) MatchesByDate(date time.Time) ([]LeagueMatches, error) {
	resp, err := s.client.GetMatches(date.Format("20060102"))
	if err != nil {
		return nil, err
	}
	return resp.Leagues, nil
}

func (s *Service) LeagueStandings(leagueID int, season string) ([]LeagueTableEntry, error) {
	resp, err := s.client.GetLeague(fmt.Sprint(leagueID), season)
	if err != nil {
		return nil, err
	}
	if len(resp.Table) == 0 || len(resp.Table[0].Data.Table.All) == 0 {
		return nil, nil
	}
	return resp.Table[0].Data.Table.All, nil
}

func (s *Service) MatchDetails(matchID int) (*MatchDetailsResponse, error) {
	resp, err := s.client.GetMatchDetails(matchID)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) MatchDetailsByID(matchID string) (*MatchDetailsResponse, error) {
	resp, err := s.client.GetMatchDetailsByID(matchID)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) MatchDetailsPage(pageURL string) (*MatchDetailsResponse, error) {
	return s.client.GetMatchDetailsPage(pageURL)
}

func (s *Service) PlayerData(playerID int) (*PlayerDataResponse, error) {
	resp, err := s.client.GetPlayer(playerID)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) LeagueMatches(leagueID int) ([]LeagueMatch, error) {
	resp, err := s.client.GetLeaguePage(leagueID)
	if err != nil {
		return nil, err
	}
	return resp.PageProps.Overview.Matches.AllMatches, nil
}

func (s *Service) SearchTeams(term string) ([]SearchResult, error) {
	resp, err := s.client.Search(term)
	if err != nil {
		return nil, err
	}
	if !resp.Found {
		return nil, nil
	}
	return resp.Data, nil
}


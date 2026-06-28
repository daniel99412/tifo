package espn

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	siteBaseURL = "https://site.api.espn.com/apis/site/v2/sports/soccer"
)

type Client struct {
	http *http.Client
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) get(url string, dst any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("crear request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request falló: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("espn status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decodificar espn json: %w", err)
	}
	return nil
}

func (c *Client) Scoreboard(leagueSlug, date string) (*ScoreboardResponse, error) {
	url := fmt.Sprintf("%s/%s/scoreboard?dates=%s", siteBaseURL, leagueSlug, date)
	var resp ScoreboardResponse
	if err := c.get(url, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Summary(leagueSlug, eventID string) (*SummaryResponse, error) {
	url := fmt.Sprintf("%s/%s/summary?event=%s&lang=es", siteBaseURL, leagueSlug, eventID)
	var resp SummaryResponse
	if err := c.get(url, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

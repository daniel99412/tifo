package sofascore

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://api.sofascore.com/api/v1"

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

func (c *Client) GetMatchIncidents(eventID int) (*IncidentsResponse, error) {
	body, err := c.get(fmt.Sprintf("/event/%d/incidents", eventID))
	if err != nil {
		return nil, err
	}
	var resp IncidentsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decodificar incidents: %w", err)
	}
	return &resp, nil
}

func (c *Client) GetMatchDetail(eventID int) (*Event, error) {
	body, err := c.get(fmt.Sprintf("/event/%d", eventID))
	if err != nil {
		return nil, err
	}
	var resp EventResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decodificar event: %w", err)
	}
	return &resp.Event, nil
}

func (c *Client) SearchTeam(name string) ([]SearchResult, error) {
	body, err := c.get(fmt.Sprintf("/search/all?q=%s", name))
	if err != nil {
		return nil, err
	}
	var resp SearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decodificar search: %w", err)
	}
	return resp.Results, nil
}

func (c *Client) GetTeamLastEvents(teamID int, limit int) (*TeamEventsResponse, error) {
	body, err := c.get(fmt.Sprintf("/team/%d/events/last/%d", teamID, limit))
	if err != nil {
		return nil, err
	}
	var resp TeamEventsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decodificar team events: %w", err)
	}
	return &resp, nil
}

func (c *Client) get(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d for %s", resp.StatusCode, path)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("leer %s: %w", path, err)
	}
	return body, nil
}

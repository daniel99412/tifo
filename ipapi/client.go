package ipapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Location struct {
	IP          string `json:"ip"`
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name"`
	Languages   string `json:"languages"`
	City        string `json:"city"`
}

type Client struct {
	http *http.Client
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) GetLocation() (*Location, error) {
	req, err := http.NewRequest(http.MethodGet, "https://ipapi.co/json/", nil)
	if err != nil {
		return nil, fmt.Errorf("crear request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request falló: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var loc Location
	if err := json.NewDecoder(resp.Body).Decode(&loc); err != nil {
		return nil, fmt.Errorf("decodificar json: %w", err)
	}

	if loc.CountryCode == "" {
		return nil, fmt.Errorf("no se pudo determinar ubicación")
	}

	return &loc, nil
}

func (l *Location) Locale() string {
	if l.Languages == "" {
		return "en"
	}
	lang := strings.Split(l.Languages, ",")[0]
	lang = strings.Split(lang, "-")[0]
	return lang
}

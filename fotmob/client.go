package fotmob

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var nextDataRe = regexp.MustCompile(`<script id="__NEXT_DATA__" type="application/json">(.*?)</script>`)

const baseURL = "https://www.fotmob.com/api"

type Client struct {
	http    *http.Client
	buildID string
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) get(path string, params url.Values, dst any) error {
	u := baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("crear request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://www.fotmob.com/")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request falló: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decodificar json: %w", err)
	}

	return nil
}

func (c *Client) GetMatches(date string) (MatchesResponse, error) {
	var resp MatchesResponse
	err := c.get("/matches", url.Values{"date": {date}}, &resp)
	return resp, err
}

func (c *Client) GetMatchDetails(matchID int) (MatchDetailsResponse, error) {
	var resp MatchDetailsResponse
	err := c.get("/matchDetails", url.Values{"matchId": {fmt.Sprint(matchID)}}, &resp)
	return resp, err
}

func (c *Client) GetMatchDetailsByID(matchID string) (MatchDetailsResponse, error) {
	var resp MatchDetailsResponse
	err := c.get("/data/matchDetails", url.Values{"matchId": {matchID}}, &resp)
	return resp, err
}

func (c *Client) GetMatchDetailsPage(pageURL string) (*MatchDetailsResponse, error) {
	// strip fragment
	if idx := strings.IndexByte(pageURL, '#'); idx >= 0 {
		pageURL = pageURL[:idx]
	}

	body, err := c.fetchPage(pageURL)
	if err != nil {
		return nil, err
	}

	match := nextDataRe.FindSubmatch(body)
	if match == nil {
		return nil, fmt.Errorf("no se encontró __NEXT_DATA__")
	}

	var resp struct {
		Props struct {
			PageProps MatchDetailsResponse `json:"pageProps"`
		} `json:"props"`
	}
	if err := json.Unmarshal(match[1], &resp); err != nil {
		return nil, fmt.Errorf("decodificar next data: %w", err)
	}

	return &resp.Props.PageProps, nil
}

func (c *Client) GetLeague(leagueID, season string) (LeagueResponse, error) {
	var resp LeagueResponse
	params := url.Values{"id": {fmt.Sprint(leagueID)}}
	if season != "" {
		params.Set("season", season)
	}
	err := c.get("/leagues", params, &resp)
	return resp, err
}

func (c *Client) GetTeam(teamID int) (map[string]any, error) {
	var resp map[string]any
	err := c.get("/teams", url.Values{"id": {fmt.Sprint(teamID)}}, &resp)
	return resp, err
}

func (c *Client) GetPlayer(playerID int) (PlayerDataResponse, error) {
	var resp PlayerDataResponse
	err := c.get("/playerData", url.Values{"id": {fmt.Sprint(playerID)}}, &resp)
	return resp, err
}

func (c *Client) Search(term string) (SearchResponse, error) {
	var resp SearchResponse
	err := c.get("/searchData", url.Values{"term": {term}}, &resp)
	return resp, err
}

func (c *Client) fetchPage(path string) ([]byte, error) {
	u := "https://www.fotmob.com" + path
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("crear request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Referer", "https://www.fotmob.com/")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request falló: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("leer body: %w", err)
	}
	return body, nil
}

func (c *Client) GetLeaguePage(leagueID int) (*LeaguePageResponse, error) {
	body, err := c.fetchPage(fmt.Sprintf("/leagues/%d", leagueID))
	if err != nil {
		return nil, err
	}

	match := nextDataRe.FindSubmatch(body)
	if match == nil {
		return nil, fmt.Errorf("no se encontró __NEXT_DATA__")
	}

	var resp struct {
		Props struct {
			PageProps LeaguePageProps `json:"pageProps"`
		} `json:"props"`
	}
	if err := json.Unmarshal(match[1], &resp); err != nil {
		return nil, fmt.Errorf("decodificar next data: %w", err)
	}

	return &LeaguePageResponse{PageProps: resp.Props.PageProps}, nil
}

func (c *Client) GetAllLeagues(locale, country string) (AllLeaguesResponse, error) {
	var resp AllLeaguesResponse
	params := url.Values{}
	if locale != "" {
		params.Set("locale", locale)
	}
	if country != "" {
		params.Set("country", country)
	}
	err := c.get("/data/allLeagues", params, &resp)
	return resp, err
}

func (c *Client) GetTranslationMapping(locale string) (TranslationMapping, error) {
	var resp TranslationMapping
	if locale == "" {
		locale = "en"
	}
	err := c.get("/translationmapping", url.Values{"locale": {locale}}, &resp)
	return resp, err
}

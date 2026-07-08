package sofascore

type Incident struct {
	ID                 int      `json:"id"`
	Time               int      `json:"time"`
	IncidentType       string   `json:"incidentType"`
	IncidentClass      string   `json:"incidentClass"`
	Player             *Player  `json:"player,omitempty"`
	PlayerName         string   `json:"playerName,omitempty"`
	PlayerIn           *Player  `json:"playerIn,omitempty"`
	PlayerOut          *Player  `json:"playerOut,omitempty"`
	IsHome             *bool    `json:"isHome,omitempty"`
	Text               string   `json:"text,omitempty"`
	Confirmed          *bool    `json:"confirmed,omitempty"`
	Length             int      `json:"length,omitempty"`
	ReversedPeriodTime int      `json:"reversedPeriodTime"`
}

type Player struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	ShortName string `json:"shortName"`
}

type IncidentsResponse struct {
	Incidents []Incident `json:"incidents"`
}

type EventResponse struct {
	Event Event `json:"event"`
}

type Event struct {
	ID             int       `json:"id"`
	HomeTeam       TeamRef   `json:"homeTeam"`
	AwayTeam       TeamRef   `json:"awayTeam"`
	HomeScore      *Score    `json:"homeScore"`
	AwayScore      *Score    `json:"awayScore"`
	Status         Status    `json:"status"`
	StartTimestamp int64     `json:"startTimestamp"`
	Slug           string    `json:"slug"`
	WinnerCode     int       `json:"winnerCode"`
	Venue          *Venue    `json:"venue,omitempty"`
	Referee        *Referee  `json:"referee,omitempty"`
	Attendance     int       `json:"attendance"`
	AwayRedCards   int       `json:"awayRedCards"`
}

type TeamRef struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	NameCode string `json:"nameCode"`
}

type Score struct {
	Current    int `json:"current"`
	Display    int `json:"display"`
	Period1    int `json:"period1"`
	Period2    int `json:"period2"`
	Normaltime int `json:"normaltime"`
}

type Status struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

type Venue struct {
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
	City     struct {
		Name string `json:"name"`
	} `json:"city"`
}

type Referee struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

type SearchResult struct {
	Type   string       `json:"type"`
	Entity SearchEntity `json:"entity"`
	Score  float64      `json:"score"`
}

type SearchEntity struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type TeamEventsResponse struct {
	Events      []TeamEvent `json:"events"`
	HasNextPage bool        `json:"hasNextPage"`
}

type TeamEvent struct {
	ID             int     `json:"id"`
	HomeTeam       TeamRef `json:"homeTeam"`
	AwayTeam       TeamRef `json:"awayTeam"`
	HomeScore      *Score  `json:"homeScore"`
	AwayScore      *Score  `json:"awayScore"`
	Status         Status  `json:"status"`
	StartTimestamp int64   `json:"startTimestamp"`
	Slug           string  `json:"slug"`
}

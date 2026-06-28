package fotmob

import "encoding/json"

type League struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	CCode    string `json:"ccode"`
	ParentID int    `json:"parentLeagueId"`
}

type Team struct {
	ID    interface{} `json:"id"`
	Name  string      `json:"name"`
	Score *int        `json:"score"`
}

type MatchStatus struct {
	UTCTime   string `json:"utcTime"`
	Started   bool   `json:"started"`
	Finished  bool   `json:"finished"`
	Cancelled bool   `json:"cancelled"`
	ScoreStr  string `json:"scoreStr"`
}

type Match struct {
	ID       int         `json:"id"`
	LeagueID int         `json:"leagueId"`
	League   string      `json:"league"`
	Time     string      `json:"time"`
	Status   MatchStatus `json:"status"`
	Home     Team        `json:"home"`
	Away     Team        `json:"away"`
}

type LeagueMatches struct {
	League
	Matches []Match `json:"matches"`
}

type MatchesResponse struct {
	Leagues []LeagueMatches `json:"leagues"`
}

type LeagueTableEntry struct {
	Idx         int    `json:"idx"`
	Name        string `json:"name"`
	ID          int    `json:"id"`
	Played      int    `json:"played"`
	Wins        int    `json:"wins"`
	Draws       int    `json:"draws"`
	Losses      int    `json:"losses"`
	GoalConDiff int    `json:"goalConDiff"`
	Pts         int    `json:"pts"`
}

type LeagueTableData struct {
	All  []LeagueTableEntry `json:"all"`
	Home []LeagueTableEntry `json:"home"`
	Away []LeagueTableEntry `json:"away"`
}

type LeagueTable struct {
	Data struct {
		Table LeagueTableData `json:"table"`
	} `json:"data"`
}

type LeagueResponse struct {
	Details League        `json:"details"`
	Table   []LeagueTable `json:"table"`
}

type TeamColors struct {
	DarkMode  ColorPair `json:"darkMode"`
	LightMode ColorPair `json:"lightMode"`
}

type ColorPair struct {
	Home string `json:"home"`
	Away string `json:"away"`
}

type MatchDetailsGeneral struct {
	MatchID    string     `json:"matchId"`
	HomeTeam   Team       `json:"homeTeam"`
	AwayTeam   Team       `json:"awayTeam"`
	TeamColors TeamColors `json:"teamColors"`
}

type ShotmapItem struct {
	ID          int64  `json:"id"`
	EventType   string `json:"eventType"`
	TeamID      int    `json:"teamId"`
	PlayerID    int    `json:"playerId"`
	PlayerName  string `json:"playerName"`
	Min         int    `json:"min"`
	MinAdded    *int   `json:"minAdded"`
	IsBlocked   bool   `json:"isBlocked"`
	IsOnTarget  bool   `json:"isOnTarget"`
	Period      string `json:"period"`
	Situation   string `json:"situation"`
	ShotType    string `json:"shotType"`
}

type ShotmapData struct {
	Shots []ShotmapItem `json:"shots"`
}

type MatchDetailsContent struct {
	MatchFacts struct {
		Events struct {
			Events     []MatchEvent `json:"events"`
			EventTypes []string     `json:"eventTypes"`
		} `json:"events"`
	} `json:"matchFacts"`
	Stats struct {
		Periods struct {
			All struct {
				Stats []StatCategory `json:"stats"`
			} `json:"All"`
		} `json:"Periods"`
	} `json:"stats"`
	Lineup struct {
		HomeTeam LineupTeam `json:"homeTeam"`
		AwayTeam LineupTeam `json:"awayTeam"`
	} `json:"lineup"`
	H2H H2HWrapper `json:"h2h"`
	PlayerStats map[string]PlayerStat `json:"playerStats"`
	Shotmap ShotmapData `json:"shotmap"`
}

type MatchEvent struct {
	Type             string      `json:"type"`
	Time             int         `json:"time"`
	OverloadTime     *int        `json:"overloadTime"`
	Player           *EventPlayer `json:"player"`
	IsHome           bool        `json:"isHome"`
	HomeScore        int         `json:"homeScore"`
	AwayScore        int         `json:"awayScore"`
	Card             string      `json:"card"`
	CardDescription  string      `json:"cardDescription"`
	Swap             []EventSwap `json:"swap"`
	Assist           *EventPlayer `json:"assist"`
	MinutesAddedInput int        `json:"minutesAddedInput"`
	MinutesAddedStr  string      `json:"minutesAddedStr"`
	HalfStrShort     string      `json:"halfStrShort"`
	GoalDescription  string      `json:"goalDescription"`
	OwnGoal          interface{} `json:"ownGoal"`
	InjuredPlayerOut bool        `json:"injuredPlayerOut"`
}

type EventPlayer struct {
	ID         *int   `json:"id"`
	Name       string `json:"name"`
	ProfileURL string `json:"profileUrl"`
}

type EventSwap struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type StatCategory struct {
	Title  string `json:"title"`
	Key    string `json:"key"`
	Stats  []StatEntry `json:"stats"`
}

type StatEntry struct {
	Title  string      `json:"title"`
	Key    string      `json:"key"`
	Stats  []interface{} `json:"stats"`
}

type LineupTeam struct {
	ID        int            `json:"id"`
	Name      string         `json:"name"`
	Formation string         `json:"formation"`
	Starters  []LineupPlayer `json:"starters"`
	Subs      []LineupPlayer `json:"subs"`
	Coach     struct {
		Name string `json:"name"`
	} `json:"coach"`
	Unavailable []struct {
		Name           string `json:"name"`
		Unavailability struct {
			Type           string `json:"type"`
			ExpectedReturn string `json:"expectedReturn"`
		} `json:"unavailability"`
	} `json:"unavailable"`
}

type LineupPlayer struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ShirtNumber string `json:"shirtNumber"`
	PositionID  int    `json:"positionId"`
	Position    string `json:"position,omitempty"`
	Role        string `json:"role,omitempty"`
}

type H2HData struct {
	Summary []int      `json:"summary"`
	Matches []H2HMatch `json:"matches"`
}

type H2HWrapper struct {
	H2HData
}

func (h *H2HWrapper) UnmarshalJSON(data []byte) error {
	if string(data) == "false" {
		return nil
	}
	return json.Unmarshal(data, &h.H2HData)
}

type H2HMatch struct {
	Home   Team        `json:"home"`
	Away   Team        `json:"away"`
	Status MatchStatus `json:"status"`
}

type PlayerStat struct {
	Rating struct {
		Num string `json:"num"`
	} `json:"rating"`
}

type GoalEvent struct {
	Player struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"player"`
	TimeStr   interface{} `json:"timeStr"`
	Type      string      `json:"type"`
	HomeScore int         `json:"homeScore"`
	AwayScore int         `json:"awayScore"`
	IsHome    bool        `json:"isHome"`
}

type MatchDetailsResponse struct {
	General MatchDetailsGeneral `json:"general"`
	Header  struct {
		Status MatchStatus `json:"status"`
		Teams  []struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			Score    int    `json:"score"`
			RedCards int    `json:"numberOfRedCards"`
		} `json:"teams"`
		Events struct {
			HomeTeamGoals map[string][]GoalEvent `json:"homeTeamGoals"`
			AwayTeamGoals map[string][]GoalEvent `json:"awayTeamGoals"`
		} `json:"events"`
	} `json:"header"`
	Content MatchDetailsContent `json:"content"`
}

type PlayerInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TeamInfo struct {
	TeamID   int    `json:"teamId"`
	TeamName string `json:"teamName"`
	Role     string `json:"role"`
}

type PlayerRecentMatch struct {
	Rating struct {
		Num string `json:"num"`
	} `json:"rating"`
	MinutesPlayed int `json:"minutesPlayed"`
	Goals         int `json:"goals"`
	Assists       int `json:"assists"`
}

type PlayerDataResponse struct {
	ID              int                 `json:"id"`
	Name            string              `json:"name"`
	FirstName       string              `json:"firstName"`
	LastName        string              `json:"lastName"`
	PrimaryTeam     TeamInfo            `json:"primaryTeam"`
	PlayerInfo      []PlayerInfoItem    `json:"playerInformation"`
	RecentMatches   []PlayerRecentMatch `json:"recentMatches"`
}

type PlayerInfoItem struct {
	Title string `json:"title"`
	Value struct {
		Fallback string `json:"fallback"`
	} `json:"value"`
}

type SearchResult struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	CCode  string `json:"ccode"`
}

type SearchResponse struct {
	Found bool            `json:"found"`
	Query string          `json:"query"`
	Data  []SearchResult  `json:"data"`
}

type LeagueMatch struct {
	ID      string `json:"id"`
	Round   string `json:"round"`
	PageURL string `json:"pageUrl"`
	Home    struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"home"`
	Away struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"away"`
	Status struct {
		UTCTime   string `json:"utcTime"`
		Started   bool   `json:"started"`
		Finished  bool   `json:"finished"`
		Cancelled bool   `json:"cancelled"`
		ScoreStr  string `json:"scoreStr"`
	} `json:"status"`
}

type LeagueOverviewMatches struct {
	AllMatches []LeagueMatch `json:"allMatches"`
}

type LeagueOverview struct {
	Matches LeagueOverviewMatches `json:"matches"`
}

type LeaguePageProps struct {
	Details  League         `json:"details"`
	Overview LeagueOverview `json:"overview"`
	Fixtures struct {
		AllMatches []LeagueMatch `json:"allMatches"`
	} `json:"fixtures"`
}

type LeaguePageResponse struct {
	PageProps LeaguePageProps `json:"pageProps"`
}

type TranslationMapping struct {
	Language             string              `json:"Language"`
	CountryCodes         map[string]string   `json:"CountryCodes"`
	TournamentPrefixes   map[string]string   `json:"TournamentPrefixes"`
	TournamentTemplates  map[string]string   `json:"TournamentTemplates"`
	LeagueMapping        map[string]string   `json:"LeagueMapping"`
}

type AllLeagueItem struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	LocalizedName  string `json:"localizedName"`
	PageURL        string `json:"pageUrl"`
	CCode          string `json:"ccode"`
}

type AllLeagueCountry struct {
	CCode         string          `json:"ccode"`
	Name          string          `json:"name"`
	LocalizedName string          `json:"localizedName"`
	Leagues       []AllLeagueItem `json:"leagues"`
}

type AllLeaguesResponse struct {
	Popular       []AllLeagueItem   `json:"popular"`
	International []AllLeagueCountry `json:"international"`
	Countries     []AllLeagueCountry `json:"countries"`
}

type LeagueInfo struct {
	ID     int
	Name   string
	CCode  string
	Region string
}

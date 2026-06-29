package espn

import "encoding/json"

type ScoreboardResponse struct {
	Events []ScoreboardEvent `json:"events"`
}

type ScoreboardEvent struct {
	ID           string            `json:"id"`
	Date         string            `json:"date"`
	Name         string            `json:"name"`
	ShortName    string            `json:"shortName"`
	Competitions []ScoreboardComp  `json:"competitions"`
}

type ScoreboardComp struct {
	ID          string                `json:"id"`
	Date        string                `json:"date"`
	StartDate   string                `json:"startDate"`
	Attendance  int                   `json:"attendance"`
	Venue       *Venue                `json:"venue"`
	Competitors []ScoreboardCompetitor `json:"competitors"`
	Broadcasts  []ScoreboardBroadcast  `json:"broadcasts"`
	Status      Status                `json:"status"`
	Format      Format                `json:"format"`
}

type ScoreboardCompetitor struct {
	ID       string `json:"id"`
	HomeAway string `json:"homeAway"`
	Score    string `json:"score"`
	Winner   bool   `json:"winner"`
	Team     struct {
		ID           string `json:"id"`
		Abbreviation string `json:"abbreviation"`
		DisplayName  string `json:"displayName"`
		ShortName    string `json:"shortDisplayName"`
	} `json:"team"`
}

type ScoreboardBroadcast struct {
	Market string   `json:"market"`
	Names  []string `json:"names"`
}

type SummaryResponse struct {
	Header         SummaryHeader    `json:"header"`
	GameInfo       GameInfo         `json:"gameInfo"`
	KeyEvents      []KeyEvent       `json:"keyEvents"`
	Commentary     []CommentaryItem `json:"commentary"`
	Format         Format           `json:"format"`
	Boxscore       Boxscore         `json:"boxscore"`
	Broadcasts     []SummaryBroadcast `json:"broadcasts"`
	Odds           []Odds           `json:"odds"`
	Rosters        []SummaryRoster  `json:"rosters"`
	HeadToHeadGames []HeadToHeadTeamEvents `json:"headToHeadGames,omitempty"`
}

type SummaryHeader struct {
	ID           string         `json:"id"`
	Competitions []SummaryComp  `json:"competitions"`
}

type SummaryComp struct {
	ID           string              `json:"id"`
	Date         string              `json:"date"`
	Competitors  []SummaryCompetitor  `json:"competitors"`
}

type SummaryCompetitor struct {
	ID       string `json:"id"`
	HomeAway string `json:"homeAway"`
	Score    string `json:"score"`
	Winner   bool   `json:"winner"`
	Team     struct {
		ID             string `json:"id"`
		DisplayName    string `json:"displayName"`
		Abbreviation   string `json:"abbreviation"`
		Color          string `json:"color"`
		AlternateColor string `json:"alternateColor"`
	} `json:"team"`
	Linescores []struct {
		DisplayValue string `json:"displayValue"`
	} `json:"linescores"`
	Record []struct {
		Type        string `json:"type"`
		Summary     string `json:"summary"`
		DisplayValue string `json:"displayValue"`
	} `json:"record"`
}

type GameInfo struct {
	Venue      Venue      `json:"venue"`
	Attendance int        `json:"attendance"`
	Officials  []Official `json:"officials"`
	Weather    Weather    `json:"weather"`
}

type Venue struct {
	ID       string `json:"id"`
	FullName string `json:"fullName"`
	Address  struct {
		City    string `json:"city"`
		State   string `json:"state"`
		Country string `json:"country"`
	} `json:"address"`
	Capacity int  `json:"capacity"`
	Indoor   bool `json:"indoor"`
}

type Official struct {
	FullName    string `json:"fullName"`
	DisplayName string `json:"displayName"`
	Position    struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		ID          string `json:"id"`
	} `json:"position"`
	Order int `json:"order"`
}

type Weather struct {
	Temperature   int    `json:"temperature"`
	Condition     string `json:"condition"`
	High          int    `json:"high"`
	Low           int    `json:"low"`
	Humidity      int    `json:"humidity"`
	WindSpeed     int    `json:"windSpeed"`
	WindDirection string `json:"windDirection"`
	DisplayValue  string `json:"displayValue"`
}

type KeyEvent struct {
	ID          string     `json:"id"`
	Type        EventType  `json:"type"`
	Text        string     `json:"text"`
	ShortText   string     `json:"shortText"`
	Period      Period     `json:"period"`
	Clock       Clock      `json:"clock"`
	Team        *TeamRef   `json:"team"`
	ScoringPlay bool       `json:"scoringPlay"`
	Shootout    bool       `json:"shootout"`
	Sequence    int        `json:"sequenceNumber"`
}

type EventType struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Type     string `json:"type"`
}

type Period struct {
	Number int `json:"number"`
}

type Clock struct {
	Value       float64 `json:"value"`
	DisplayValue string `json:"displayValue"`
}

type TeamRef struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type CommentaryItem struct {
	Sequence int          `json:"sequence"`
	Time     Clock        `json:"time"`
	Text     string       `json:"text"`
	Play     *KeyEvent    `json:"play"`
}

type Format struct {
	Regulation Regulation `json:"regulation"`
}

type Regulation struct {
	Periods     int     `json:"periods"`
	DisplayName string  `json:"displayName"`
	Slug        string  `json:"slug"`
	Clock       float64 `json:"clock"`
}

type Boxscore struct {
	Form  interface{} `json:"form"`
	Teams []struct {
		Team struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"team"`
		Statistics []struct {
			Name         string `json:"name"`
			DisplayValue string `json:"displayValue"`
			Label        string `json:"label"`
		} `json:"statistics"`
	} `json:"teams"`
}

type SummaryBroadcast struct {
	Type   *BroadcastType `json:"type"`
	Market interface{}    `json:"market"`
	Media  *BroadcastMedia `json:"media"`
	Lang   string         `json:"lang"`
	Region string         `json:"region"`
}

type BroadcastType struct {
	ID        string `json:"id"`
	ShortName string `json:"shortName"`
	LongName  string `json:"longName"`
	Slug      string `json:"slug"`
}

type BroadcastMedia struct {
	CallLetters string `json:"callLetters"`
	Name        string `json:"name"`
	ShortName   string `json:"shortName"`
}

type Odds struct {
	Provider OddsProvider `json:"provider"`
	Details  string       `json:"details"`
}

type OddsProvider struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Priority int    `json:"priority"`
}

type Status struct {
	Clock     float64  `json:"clock"`
	Period    int      `json:"period"`
	Type      StatusType `json:"type"`
}

type StatusType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	State       string `json:"state"`
	Completed   bool   `json:"completed"`
	Description string `json:"description"`
	Detail      string `json:"detail"`
}

type PositionsResponse struct {
	Count    int              `json:"count"`
	PageIndex int             `json:"pageIndex"`
	PageSize  int             `json:"pageSize"`
	PageCount int             `json:"pageCount"`
	Items    []PositionRef    `json:"items"`
}

type PositionRef struct {
	Ref string `json:"$ref"`
}

type ESPNPosition struct {
	Ref          string          `json:"$ref,omitempty"`
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	DisplayName  string          `json:"displayName"`
	Abbreviation string          `json:"abbreviation"`
	Leaf         bool            `json:"leaf"`
	Parent       *PositionParent `json:"parent,omitempty"`
}

type PositionParent struct {
	Ref string `json:"$ref"`
}

type RosterResponse struct {
	Team     RosterTeam     `json:"team"`
	Athletes []RosterGroup  `json:"athletes"`
}

type RosterTeam struct {
	ID           string `json:"id"`
	Abbreviation string `json:"abbreviation"`
	DisplayName  string `json:"displayName"`
}

type RosterGroup struct {
	Position string         `json:"position"`
	Items    []RosterPlayer `json:"items"`
}

type RosterPlayer struct {
	ID          string         `json:"id"`
	DisplayName string         `json:"displayName"`
	FirstName   string         `json:"firstName"`
	LastName    string         `json:"lastName"`
	Jersey      string         `json:"jersey"`
	Position    RosterPosition `json:"position"`
}

type RosterPosition struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`

	DisplayName string `json:"displayName"`
}

type SummaryRoster struct {
	HomeAway  string                `json:"homeAway"`
	Winner    bool                  `json:"winner"`
	Team      SummaryRosterTeam     `json:"team"`
	Roster    []SummaryRosterPlayer `json:"roster"`
	Uniform   json.RawMessage       `json:"uniform"`
	Formation string                `json:"formation"`
}

type SummaryRosterTeam struct {
	ID           string `json:"id"`
	DisplayName  string `json:"displayName"`
	Abbreviation string `json:"abbreviation"`
}

type SummaryRosterPlayer struct {
	Active      bool                 `json:"active"`
	Starter     bool                 `json:"starter"`
	Jersey      string               `json:"jersey"`
	Athlete     SummaryAthlete       `json:"athlete"`
	Position    SummaryPosition      `json:"position"`
	FormationPlace string             `json:"formationPlace"`
}

type SummaryAthlete struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	FullName    string `json:"fullName"`
}

type SummaryPosition struct {
	Name         string `json:"name"`
	DisplayName  string `json:"displayName"`
	Abbreviation string `json:"abbreviation"`
}

type HeadToHeadTeamEvents struct {
	Team   HeadToHeadTeam   `json:"team"`
	Events []HeadToHeadGame `json:"events"`
}

type HeadToHeadTeam struct {
	ID           string `json:"id"`
	DisplayName  string `json:"displayName"`
	Abbreviation string `json:"abbreviation"`
}

type HeadToHeadGame struct {
	ID               string `json:"id"`
	Score            string `json:"score"`
	GameDate         string `json:"gameDate"`
	HomeTeamID       string `json:"homeTeamId"`
	AwayTeamID       string `json:"awayTeamId"`
	HomeTeamScore    string `json:"homeTeamScore"`
	AwayTeamScore    string `json:"awayTeamScore"`
	GameResult       string `json:"gameResult"`
	CompetitionName  string `json:"competitionName"`
	LeagueName       string `json:"leagueName"`
	AtVs             string `json:"atVs"`
	Opponent         struct {
		ID           string `json:"id"`
		DisplayName  string `json:"displayName"`
		Abbreviation string `json:"abbreviation"`
	} `json:"opponent"`
}

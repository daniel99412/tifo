package domain

import "time"

// Competition represents a league or tournament.
type Competition struct {
	TIFOID       TIFOID
	ExternalIDs  ExternalIDs
	Name         string
	OriginalName string
	Country      string
	Season       string
}

// CompetitionRef is a lightweight reference to a competition.
type CompetitionRef struct {
	TIFOID TIFOID
	Name   string
}

// Team represents a club or national team.
type Team struct {
	TIFOID      TIFOID
	ExternalIDs ExternalIDs
	Name        string
	ShortName   string
	Country     string
}

// TeamRef is a lightweight reference to a team.
type TeamRef struct {
	TIFOID TIFOID
	Name   string
	Short  string
}

// TeamColors contains home/away color pairs from providers.
type TeamColors struct {
	Home     string
	Away     string
	HomeAlt  string
	AwayAlt  string
}

// Player represents a person.
type Player struct {
	TIFOID      TIFOID
	ExternalIDs ExternalIDs
	Name        string
	Number      string
	Position    string
	Country     string
	PhotoURL    string
}

// PlayerRef is a lightweight reference.
type PlayerRef struct {
	TIFOID  TIFOID
	Name    string
	Number  string
	PosID   int
	PosName string
}

// MatchStatus describes the current state of a match.
type MatchStatus struct {
	State     MatchState
	Detail    string // e.g. "Half Time", "Full Time", "66'"
	Period    int
	Clock     string
	ScoreStr  string // e.g. "2-1"
	UTCTime   string // raw UTC string from provider
	Kickoff   time.Time
}

// MatchState enumerates possible match states.
type MatchState string

const (
	MatchScheduled  MatchState = "scheduled"
	MatchLive       MatchState = "live"
	MatchFinished   MatchState = "finished"
	MatchPostponed  MatchState = "postponed"
	MatchCancelled  MatchState = "cancelled"
	MatchAbandoned  MatchState = "abandoned"
)

// Match is the top-level match summary shown in lists.
type Match struct {
	TIFOID       TIFOID
	ExternalIDs  ExternalIDs
	LeagueID     int
	Competition  CompetitionRef
	Home         TeamRef
	Away         TeamRef
	HomeScore    *int
	AwayScore    *int
	Status       MatchStatus
}

// MatchDetails contains all in-depth match data.
type MatchDetails struct {
	TIFOID      TIFOID
	ExternalIDs ExternalIDs
	Match       MatchRef

	Lineups    *Lineups
	Events     []MatchEvent
	Statistics []StatCategory
	H2H        *H2H
	Injuries   []InjuryItem
	ShotMap    []Shot

	ExtraInfo MatchExtraInfo
}

// MatchRef is a lightweight match reference.
type MatchRef struct {
	TIFOID TIFOID
	Home   string
	Away   string
	Score  string
}

// MatchExtraInfo holds supplemental info from enrichment.
type MatchExtraInfo struct {
	Venue      string
	Attendance int
	Referee    string
	Weather    string
	Broadcasts []string
	HomeColor  string
	AwayColor  string
}

// Lineups contains formations, starters, subs, and coaches.
type Lineups struct {
	HomeFormation string
	AwayFormation string
	HomeStarters  []PlayerRef
	HomeSubs      []PlayerRef
	AwayStarters  []PlayerRef
	AwaySubs      []PlayerRef
	HomeCoach     string
	AwayCoach     string
}

// MatchEvent is a single event in the match timeline.
type MatchEvent struct {
	Minute       int
	AddedTime    int
	EventType    EventType
	Team         TeamSide
	Player       *PlayerRef
	Assist       *PlayerRef
	HomeScore    int
	AwayScore    int
	CardType     string
	Detail       string
	SubOut       *PlayerRef
	SubIn        *PlayerRef
	GoalDesc     string
	OwnGoal      bool
	ShotDesc     string
	HalfStr      string
	// Internal sorting
	SortTime     int
	SortOverload int
}

// EventType enumerates match event types.
type EventType string

const (
	EvGoal         EventType = "Goal"
	EvCard         EventType = "Card"
	EvYellow       EventType = "Yellow"
	EvRed          EventType = "Red"
	EvSubstitution EventType = "Substitution"
	EvHalf         EventType = "Half"
	EvAddedTime    EventType = "AddedTime"
	EvPenalty      EventType = "PenaltyAwarded"
	EvMissedPenalty EventType = "MissedPenalty"
	EvSavedPenalty  EventType = "SavedPenalty"
	EvOwnGoal      EventType = "OwnGoal"
	EvShot         EventType = "Shot"
	EvInjury       EventType = "InjuryTime"
	EvVAR          EventType = "VAR"
	EvVideoReview  EventType = "VideoReview"
	EvWaterBreak   EventType = "WaterBreak"
	EvKO           EventType = "KO"
	EvHT           EventType = "HT"
	EvS2           EventType = "S2"
	EvPausa        EventType = "Pausa"
	EvContinua     EventType = "Continúa"
	EvFT           EventType = "FT"
)

// TeamSide distinguishes home/away/neutral.
type TeamSide int

const (
	SideHome    TeamSide = 0
	SideAway    TeamSide = 1
	SideNeutral TeamSide = 2
)

// StatCategory groups related stats.
type StatCategory struct {
	Title string
	Key   string
	Stats []StatRow
}

// StatRow is a single stat with home/away values.
type StatRow struct {
	Label string
	Key   string
	Home  string
	Away  string
	// Provenance: which provider supplied this value
	HomeProvider string
	AwayProvider string
}

// H2H contains head-to-head aggregate data.
type H2HMatchDetail struct {
	Date        time.Time
	HomeTeam    string
	AwayTeam    string
	HomeScore   int
	AwayScore   int
	Competition string
}

type H2HFormEvent struct {
	Opponent string
	Score    string
	Result   string // W/D/L
}

type H2H struct {
	HomeWins int
	Draws    int
	AwayWins int
	Matches  []H2HMatchDetail

	// ESPN enrichment
	HomeForm   []H2HFormEvent
	AwayForm   []H2HFormEvent
	HomeRecord string
	AwayRecord string
}

// InjuryItem describes an unavailable player.
type InjuryItem struct {
	Player PlayerRef
	Type   string
	Return string
	Team   TeamSide
}

// Shot is a shot-map entry.
type Shot struct {
	Minute     int
	AddedTime  int
	Player     string
	Team       TeamSide
	X, Y       float64
	ExpectedGO float64
	EventType  string
}

// Venue describes a stadium.
type Venue struct {
	Name    string
	City    string
	Country string
}

// LeaguePage holds matches grouped for a competition on a date.
type LeaguePage struct {
	Competition CompetitionRef
	Matches     []Match
}

// ExtraEvent is an enrichment-only event (kickoff, delay, etc.).
type ExtraEvent struct {
	Minute      int
	AddedTime   int
	Period      int
	EventType   EventType
	Description string
	TeamSide    string
}

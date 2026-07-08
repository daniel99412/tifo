package domain

import (
	"sort"
	"strings"
)

// EventIdentity uniquely identifies a real match event across providers.
type EventIdentity struct {
	Minute   int
	Type     EventType // normalized domain type (EvGoal, EvCard, etc.)
	Team     TeamSide
	Player   string // player name involved
	CardType string // for cards: "Yellow" / "Red"
	SubOut   string // for substitutions: player going out
	SubIn    string // for substitutions: player coming in
}

// Identity returns the canonical identity of a match event.
func Identity(ev MatchEvent) EventIdentity {
	id := EventIdentity{
		Minute:   ev.Minute,
		Type:     classifyDomain(ev),
		Team:     ev.Team,
		Player:   "",
		CardType: ev.CardType,
		SubOut:   "",
		SubIn:    "",
	}
	if ev.Player != nil {
		id.Player = ev.Player.Name
	}
	if ev.SubOut != nil {
		id.SubOut = ev.SubOut.Name
	}
	if ev.SubIn != nil {
		id.SubIn = ev.SubIn.Name
	}
	if ev.EventType == EvVAR || ev.EventType == EvVideoReview {
		// VAR matches by minute only (no player/team needed)
		id.Team = SideHome
		id.Player = ""
	}
	if ev.EventType == EvPausa || ev.EventType == EvInjury || ev.EventType == EvContinua {
		id.Team = SideHome
		id.Player = ""
	}
	return id
}

// classifyDomain normalizes a MatchEvent to its canonical domain type,
// collapsing provider-specific variations.
func classifyDomain(ev MatchEvent) EventType {
	switch ev.EventType {
	case EvGoal, "goal", "goal---header", "penalty---scored":
		return EvGoal
	case EvCard, "yellow-card", "red-card":
		return EvCard
	case EvSubstitution, "substitution":
		return EvSubstitution
	case EvVAR, EvVideoReview, "var", "video-review", "varDecision":
		return EvVAR
	case EvPausa, "CoolingBreak", "DrinkBreak":
		return EvPausa
	case EvInjury:
		return EvInjury
	case EvContinua, "end-delay":
		return EvContinua
	case EvKO, "kickoff":
		return EvKO
	case EvHT, "halftime":
		return EvHT
	case EvS2, "start-2nd-half":
		return EvS2
	case EvFT, "end-regular-time":
		return EvFT
	case EvAddedTime, "injuryTime":
		return EvAddedTime
	case EvShot:
		return EvShot
	case EvHalf:
		if ev.HalfStr == "HT" || ev.HalfStr == "HalfTime" {
			return EvHT
		}
		if ev.HalfStr == "FT" || ev.HalfStr == "FullTime" {
			return EvFT
		}
		return EvHalf
	default:
		return ev.EventType
	}
}

// MergeEvents enriches base with any non-zero data from candidate.
// Base fields take priority (first provider wins), except for specific enrichment fields.
func MergeEvents(base, candidate *MatchEvent) {
	// Enrichment fields: candidate fills gaps that base doesn't have
	if base.GoalType == "" {
		base.GoalType = candidate.GoalType
	}
	if base.PauseType == "" {
		base.PauseType = candidate.PauseType
	}
	if base.DelayText == "" {
		base.DelayText = candidate.DelayText
	}
	if base.VarClass == "" {
		base.VarClass = candidate.VarClass
	}
	if base.VarConfirmed == nil && candidate.VarConfirmed != nil {
		base.VarConfirmed = candidate.VarConfirmed
	}
	if base.XG == nil && candidate.XG != nil {
		base.XG = candidate.XG
	}
	if base.AssistName == "" {
		base.AssistName = candidate.AssistName
	}
	if base.Detail == "" {
		base.Detail = candidate.Detail
	}
	if base.Player == nil && candidate.Player != nil {
		base.Player = candidate.Player
	}
	if base.CardType == "" {
		base.CardType = candidate.CardType
	}

	// For HT/FT, prefer the version with added time (more precise)
	if (base.EventType == EvHT || base.EventType == EvFT) &&
		base.AddedTime == 0 && candidate.AddedTime > 0 {
		base.AddedTime = candidate.AddedTime
		base.SortOverload = candidate.AddedTime
	}

	// For VAR/VideoReview, prefer SofaScore's detail over ESPN's generic fallback
	if (candidate.EventType == EvVAR || candidate.EventType == EvVideoReview) &&
		candidate.Detail != "" && !strings.HasPrefix(base.Detail, "VAR:") {
		base.Detail = candidate.Detail
	}

	// Prefer using EventType=EvVAR if one side has it
	if base.EventType != EvVAR && base.EventType != EvVideoReview &&
		(candidate.EventType == EvVAR || candidate.EventType == EvVideoReview) {
		base.EventType = candidate.EventType
	}
}

// IdentityMatch returns true if two events represent the same real event.
func IdentityMatch(a, b MatchEvent) bool {
	ai := Identity(a)
	bi := Identity(b)

	// Minute tolerance varies by type
	switch ai.Type {
	case EvHT, EvFT:
		diff := ai.Minute - bi.Minute
		if diff < 0 {
			diff = -diff
		}
		if diff > 3 {
			return false
		}
	case EvSubstitution:
		diff := ai.Minute - bi.Minute
		if diff < 0 {
			diff = -diff
		}
		if diff > 1 {
			return false
		}
	default:
		if ai.Minute != bi.Minute {
			return false
		}
	}

	if ai.Type != bi.Type {
		// Allow VAR ↔ VideoReview match
		if !(ai.Type == EvVAR && bi.Type == EvVAR) {
			return false
		}
	}
	if ai.Team != bi.Team {
		return false
	}

	switch ai.Type {
	case EvGoal, EvCard:
		return ai.Player != "" && ai.Player == bi.Player
	case EvSubstitution:
		// Primary: match by subOut
		if ai.SubOut != "" && ai.SubOut == bi.SubOut {
			return true
		}
		// Secondary: match by subIn + team
		if ai.SubIn != "" && ai.SubIn == bi.SubIn && ai.Team == bi.Team {
			return true
		}
		return false
	case EvVAR:
		return true // same minute = same VAR event
	case EvPausa, EvInjury, EvContinua, EvShot:
		return true // same minute + same type = same event
	default:
		return true
	}
}

// DedupEvents removes duplicate events and collapses same-minute same-type pairs.
// Run after all enrichment is complete.
func DedupEvents(events []MatchEvent) []MatchEvent {
	if len(events) == 0 {
		return events
	}

	// Sort by (period, sortTime, sortOverload)
	sorted := make([]MatchEvent, len(events))
	copy(sorted, events)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Period != sorted[j].Period {
			return sorted[i].Period < sorted[j].Period
		}
		if sorted[i].SortTime != sorted[j].SortTime {
			return sorted[i].SortTime < sorted[j].SortTime
		}
		if sorted[i].SortOverload != sorted[j].SortOverload {
			return sorted[i].SortOverload < sorted[j].SortOverload
		}
		// Pausa before Continua at same time
		pi := eventTypeOrder(sorted[i].EventType)
		pj := eventTypeOrder(sorted[j].EventType)
		return pi < pj
	})

	// Collapse duplicates
	type dedupKey struct {
		minute int
		etype  EventType
	}
	seen := make(map[dedupKey]bool)
	out := make([]MatchEvent, 0, len(sorted))

	for _, ev := range sorted {
		key := dedupKey{minute: ev.Minute, etype: classifyDomain(ev)}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, ev)
	}

	return out
}

func eventTypeOrder(et EventType) int {
	switch et {
	case EvPausa, EvWaterBreak, EvInjury:
		return 0
	case EvVAR:
		return 1
	case EvVideoReview:
		return 2
	case EvContinua:
		return 3
	default:
		return 5
	}
}

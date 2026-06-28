package enrich

import (
	"fmt"
	"tifo/internal/domain"
)

// MergeRule defines how to merge a specific field from two providers.
type MergeRule int

const (
	MergePreferPrimary MergeRule = iota
	MergePreferEnrichment
	MergeFillGaps
	MergeConcatenate
)

// MergeConfig defines per-field merge behaviour.
type MergeConfig struct {
	Venue       MergeRule
	Attendance  MergeRule
	Referee     MergeRule
	Weather     MergeRule
	Broadcasts  MergeRule
	TeamColors  MergeRule
	Stats       MergeRule
	Events      MergeRule
	Lineups     MergeRule
	H2H         MergeRule
	Injuries    MergeRule
	ShotMap     MergeRule
	ExtraEvents MergeRule
}

// DefaultMergeConfig returns the recommended merge rules.
func DefaultMergeConfig() MergeConfig {
	return MergeConfig{
		Venue:       MergePreferEnrichment,
		Attendance:  MergePreferEnrichment,
		Referee:     MergePreferEnrichment,
		Weather:     MergePreferEnrichment,
		Broadcasts:  MergeConcatenate,
		TeamColors:  MergePreferPrimary,
		Stats:       MergePreferPrimary,
		Events:      MergeConcatenate,
		Lineups:     MergePreferPrimary,
		H2H:         MergePreferPrimary,
		Injuries:    MergePreferPrimary,
		ShotMap:     MergePreferPrimary,
		ExtraEvents: MergeConcatenate,
	}
}

// Enricher merges data from a primary and enrichment provider into a single MatchDetails.
type Enricher struct {
	cfg MergeConfig
}

func NewEnricher(cfg MergeConfig) *Enricher {
	return &Enricher{cfg: cfg}
}

// Merge takes a primary MatchDetails (e.g. from FotMob) and enrichment data
// (e.g. from ESPN) and returns a unified result.
func (e *Enricher) Merge(primary, enrichment *domain.MatchDetails) *domain.MatchDetails {
	if primary == nil {
		return enrichment
	}
	if enrichment == nil {
		return primary
	}

	out := *primary

	// Venue
	if e.cfg.Venue == MergePreferEnrichment {
		if enrichment.ExtraInfo.Venue != "" {
			out.ExtraInfo.Venue = enrichment.ExtraInfo.Venue
		}
	} else if e.cfg.Venue == MergeFillGaps {
		if out.ExtraInfo.Venue == "" {
			out.ExtraInfo.Venue = enrichment.ExtraInfo.Venue
		}
	}

	// Attendance
	if e.cfg.Attendance == MergePreferEnrichment {
		if enrichment.ExtraInfo.Attendance > 0 {
			out.ExtraInfo.Attendance = enrichment.ExtraInfo.Attendance
		}
	} else if e.cfg.Attendance == MergeFillGaps {
		if out.ExtraInfo.Attendance == 0 {
			out.ExtraInfo.Attendance = enrichment.ExtraInfo.Attendance
		}
	}

	// Referee
	if e.cfg.Referee == MergePreferEnrichment {
		if enrichment.ExtraInfo.Referee != "" {
			out.ExtraInfo.Referee = enrichment.ExtraInfo.Referee
		}
	} else if e.cfg.Referee == MergeFillGaps {
		if out.ExtraInfo.Referee == "" {
			out.ExtraInfo.Referee = enrichment.ExtraInfo.Referee
		}
	}

	// Weather
	if e.cfg.Weather == MergePreferEnrichment {
		if enrichment.ExtraInfo.Weather != "" {
			out.ExtraInfo.Weather = enrichment.ExtraInfo.Weather
		}
	} else if e.cfg.Weather == MergeFillGaps {
		if out.ExtraInfo.Weather == "" {
			out.ExtraInfo.Weather = enrichment.ExtraInfo.Weather
		}
	}

	// Broadcasts
	if e.cfg.Broadcasts == MergeConcatenate {
		seen := make(map[string]bool, len(out.ExtraInfo.Broadcasts))
		for _, b := range out.ExtraInfo.Broadcasts {
			seen[b] = true
		}
		for _, b := range enrichment.ExtraInfo.Broadcasts {
			if !seen[b] {
				out.ExtraInfo.Broadcasts = append(out.ExtraInfo.Broadcasts, b)
			}
		}
	}

	// Team colors: prefer primary, fall back to enrichment
	if e.cfg.TeamColors == MergePreferPrimary {
		if out.ExtraInfo.HomeColor == "" {
			out.ExtraInfo.HomeColor = enrichment.ExtraInfo.HomeColor
		}
		if out.ExtraInfo.AwayColor == "" {
			out.ExtraInfo.AwayColor = enrichment.ExtraInfo.AwayColor
		}
	}

	// Stats: fill gaps from enrichment
	if e.cfg.Stats == MergeFillGaps || e.cfg.Stats == MergePreferPrimary {
		// Build enrichment stats map: key → [home, away]
		enrichMap := make(map[string][2]string)
		for _, cat := range enrichment.Statistics {
			for _, s := range cat.Stats {
				enrichMap[s.Key] = [2]string{s.Home, s.Away}
			}
		}
		// Fill gaps in primary stats
		for ci, cat := range out.Statistics {
			for si, s := range cat.Stats {
				if s.Home == "" || s.Away == "" {
					if e, ok := enrichMap[s.Key]; ok {
						if s.Home == "" {
							out.Statistics[ci].Stats[si].Home = e[0]
							out.Statistics[ci].Stats[si].HomeProvider = "espn"
						}
						if s.Away == "" {
							out.Statistics[ci].Stats[si].Away = e[1]
							out.Statistics[ci].Stats[si].AwayProvider = "espn"
						}
					}
				}
			}
		}
	}

	// Events: merge both, deduplicate
	if e.cfg.Events == MergeConcatenate {
		existing := make(map[string]bool)
		for _, ev := range out.Events {
			key := eventKey(ev)
			existing[key] = true
		}
		for _, ev := range enrichment.Events {
			key := eventKey(ev)
			if !existing[key] {
				out.Events = append(out.Events, ev)
				existing[key] = true
			}
		}
	}

	// Extra events from enrichment
	if enrichment.ExtraInfo.Broadcasts != nil {
		// ExtraInfo doesn't carry extra events — they're in Events with enrichment types
	}

	return &out
}

func eventKey(ev domain.MatchEvent) string {
	return fmt.Sprintf("%d:%d:%s:%s", ev.Minute, ev.AddedTime, ev.EventType, ev.Player)
}

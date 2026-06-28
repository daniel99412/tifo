package resolver

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
	"tifo/internal/domain"
	"tifo/internal/persistence/sqlite"
)

var tifoCounter int64

func nextTIFOID(prefix string) domain.TIFOID {
	n := atomic.AddInt64(&tifoCounter, 1)
	return domain.TIFOID(fmt.Sprintf("%s_%d", prefix, n))
}

// MatchResolver resolves different provider match IDs to a single TIFO ID.
type MatchResolver struct {
	db *sqlite.MappingDB
}

func NewMatchResolver(db *sqlite.MappingDB) *MatchResolver {
	return &MatchResolver{db: db}
}

// Resolve returns the TIFO ID for a provider match, creating one if needed.
func (r *MatchResolver) Resolve(provider, externalID string) (domain.TIFOID, error) {
	tifo, err := r.db.GetByExternalID("match", provider, externalID)
	if err != nil {
		return "", err
	}
	if tifo != "" {
		return domain.TIFOID(tifo), nil
	}

	id := nextTIFOID("match")
	if err := r.db.Set("match", provider, externalID, string(id), 1.0); err != nil {
		return "", err
	}
	return id, nil
}

// TeamResolver resolves team IDs.
type TeamResolver struct {
	db *sqlite.MappingDB
}

func NewTeamResolver(db *sqlite.MappingDB) *TeamResolver {
	return &TeamResolver{db: db}
}

func (r *TeamResolver) Resolve(provider, externalID string) (domain.TIFOID, error) {
	tifo, err := r.db.GetByExternalID("team", provider, externalID)
	if err != nil {
		return "", err
	}
	if tifo != "" {
		return domain.TIFOID(tifo), nil
	}

	id := nextTIFOID("team")
	if err := r.db.Set("team", provider, externalID, string(id), 1.0); err != nil {
		return "", err
	}
	return id, nil
}

// CompetitionResolver resolves competition IDs.
type CompetitionResolver struct {
	db *sqlite.MappingDB
}

func NewCompetitionResolver(db *sqlite.MappingDB) *CompetitionResolver {
	return &CompetitionResolver{db: db}
}

func (r *CompetitionResolver) Resolve(provider, externalID string) (domain.TIFOID, error) {
	tifo, err := r.db.GetByExternalID("competition", provider, externalID)
	if err != nil {
		return "", err
	}
	if tifo != "" {
		return domain.TIFOID(tifo), nil
	}

	id := nextTIFOID("comp")
	if err := r.db.Set("competition", provider, externalID, string(id), 1.0); err != nil {
		return "", err
	}
	return id, nil
}

// FuzzyTeamMatch normalizes and compares two team names.
func FuzzyTeamMatch(a, b string) bool {
	return normalizeTeam(a) == normalizeTeam(b)
}

func normalizeTeam(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	repl := []string{
		"ä", "ae", "ö", "oe", "ü", "ue", "é", "e", "è", "e",
		"ê", "e", "í", "i", "ó", "o", "ú", "u", "ñ", "n",
		"á", "a", "ç", "c", "'", "", ".", "", "-", " ",
		"fc ", "", "cf ", "", "ac ", "", "afc ", "",
		"  ", " ",
	}
	for i := 0; i < len(repl); i += 2 {
		s = strings.ReplaceAll(s, repl[i], repl[i+1])
	}
	return strings.TrimSpace(s)
}

// FuzzyMatchByDateTime attempts to find a match by teams, UTC time (±tolerance).
func FuzzyMatchByDateTime(home, away string, utcTime time.Time, match domain.Match, tolerance time.Duration) bool {
	diff := match.Status.Kickoff.Sub(utcTime)
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		return false
	}
	if FuzzyTeamMatch(home, match.Home.Name) && FuzzyTeamMatch(away, match.Away.Name) {
		return true
	}
	if FuzzyTeamMatch(home, match.Away.Name) && FuzzyTeamMatch(away, match.Home.Name) {
		return true
	}
	return false
}

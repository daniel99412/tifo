package espn

import (
	"testing"
	"tifo/internal/domain"
	oldESPN "tifo/espn"
)

func TestNamesMatch(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"aubrey maphosa modiba", "aubrey modiba", true},
		{"aubrey modiba", "aubrey maphosa modiba", true},
		{"jonathan david", "jonathan david", true},
		{"ronwen williams", "ronwen williams", true},
		{"dayne st clair", "dayne st clair", true},
		{"david", "david beckham", false},
		{"aubrey", "aubrey modiba", false},
		{"stephen eustaquio", "stephen eustaquio", true},
		{"moise bombito", "moise bombito", true},
		{"cristian romero", "cristian romero", true},
		{"cristian romero", "cristian pavon", false},
	}

	for _, tt := range tests {
		got := namesMatch(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("namesMatch(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestPositionEnrichmentWithMockRoster(t *testing.T) {
	players := []domain.PlayerRef{
		{Name: "Emiliano Martínez", Number: "1"},
		{Name: "Nahuel Molina", Number: "26"},
		{Name: "Cristian Romero", Number: "13"},
		{Name: "Nicolás Otamendi", Number: "19"},
	}

	rosterPos := map[string]string{
		"emiliano martinez": "G",
		"nahuel molina":     "RB",
		"cristian romero":   "CD-R",
		"nicolas otamendi":  "CD-L",
	}

	enrichPlayersFromRoster(players, rosterPos)

	for _, p := range players {
		t.Logf("  %s → %s", p.Name, p.PosName)
		if p.PosName == "" {
			t.Errorf("position empty for %s", p.Name)
		}
	}
}

func TestEnrichPositionsFuzzyMatch(t *testing.T) {
	players := []domain.PlayerRef{
		{Name: "Aubrey Maphosa Modiba", Number: "6"},
		{Name: "Ronwen Williams", Number: "1"},
		{Name: "Khuliso Mudau", Number: "20"},
	}

	rosterPos := map[string]string{
		"aubrey modiba":    "LB",
		"ronwen williams":  "G",
		"khuliso mudau":    "RB",
	}

	enrichPlayersFromRoster(players, rosterPos)

	for _, p := range players {
		t.Logf("  %s → %s", p.Name, p.PosName)
		if p.PosName == "" {
			t.Errorf("position empty for %s", p.Name)
		}
	}
}

func TestEnrichPositionsFromSummary(t *testing.T) {
	lineups := &domain.Lineups{
		HomeFormation: "4-3-3",
		HomeStarters: []domain.PlayerRef{
			{Name: "Saulo", Number: "85"},
			{Name: "Victor Ramos", Number: "33"},
			{Name: "Frazan", Number: "27"},
		},
		AwayStarters: []domain.PlayerRef{
			{Name: "Bruno D´Lucas Nascimento Lopes", Number: "1"},
			{Name: "Arthur Henrique", Number: "3"},
			{Name: "João Paulo", Number: "13"},
		},
	}
	details := &domain.MatchDetails{Lineups: lineups}

	rosters := []oldESPN.SummaryRoster{
		{
			HomeAway: "home",
			Roster: []oldESPN.SummaryRosterPlayer{
				{Athlete: oldESPN.SummaryAthlete{DisplayName: "Saulo"}, Position: oldESPN.SummaryPosition{Name: "Goalkeeper", Abbreviation: "G"}, Starter: true},
				{Athlete: oldESPN.SummaryAthlete{DisplayName: "Victor Ramos"}, Position: oldESPN.SummaryPosition{Name: "Center Defender", Abbreviation: "CD-L"}, Starter: true},
				{Athlete: oldESPN.SummaryAthlete{DisplayName: "Frazan"}, Position: oldESPN.SummaryPosition{Name: "Center Defender", Abbreviation: "CD-R"}, Starter: true},
			},
		},
		{
			HomeAway: "away",
			Roster: []oldESPN.SummaryRosterPlayer{
				{Athlete: oldESPN.SummaryAthlete{DisplayName: "Bruno D´Lucas Nascimento Lopes"}, Position: oldESPN.SummaryPosition{Name: "Goalkeeper", Abbreviation: "G"}, Starter: true},
				{Athlete: oldESPN.SummaryAthlete{DisplayName: "Arthur Henrique"}, Position: oldESPN.SummaryPosition{Name: "Center Defender", Abbreviation: "CD"}, Starter: true},
				{Athlete: oldESPN.SummaryAthlete{DisplayName: "João Paulo"}, Position: oldESPN.SummaryPosition{Name: "Center Defender", Abbreviation: "CD-L"}, Starter: true},
			},
		},
	}

	got := &Provider{}
	got.enrichPositions(details, rosters)

	t.Log("Home starters:")
	for _, p := range details.Lineups.HomeStarters {
		t.Logf("  %s → %s", p.Name, p.PosName)
		if p.PosName == "" {
			t.Errorf("position empty for home player %s", p.Name)
		}
	}
	t.Log("Away starters:")
	for _, p := range details.Lineups.AwayStarters {
		t.Logf("  %s → %s", p.Name, p.PosName)
		if p.PosName == "" {
			t.Errorf("position empty for away player %s", p.Name)
		}
	}
}

func TestNormalizePlayerName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Emiliano Martínez", "emiliano martinez"},
		{"Ángel Di María", "angel di maria"},
		{"Julián Álvarez", "julian alvarez"},
		{"Lionel Messi", "lionel messi"},
		{"  Cristian Romero  ", "cristian romero"},
		{"José", "jose"},
		{"João", "joao"},
		{"François", "francois"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizePlayerName(tt.input)
			if got != tt.want {
				t.Errorf("normalizePlayerName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

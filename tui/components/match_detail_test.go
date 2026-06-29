package components

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestParseFormation(t *testing.T) {
	tests := []struct {
		name      string
		formation string
		want      []int
		wantOK    bool
	}{
		{name: "four three three", formation: "4-3-3", want: []int{4, 3, 3}, wantOK: true},
		{name: "four two three one", formation: "4-2-3-1", want: []int{4, 2, 3, 1}, wantOK: true},
		{name: "invalid total", formation: "4-4-1", wantOK: false},
		{name: "empty", formation: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseFormation(tt.formation)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestFormationRows(t *testing.T) {
	starters := makePlayers(11)
	rows := formationRows("4-2-3-1", starters)
	wantSizes := []int{1, 4, 2, 3, 1}

	if len(rows) != len(wantSizes) {
		t.Fatalf("row count = %d, want %d", len(rows), len(wantSizes))
	}
	for i, want := range wantSizes {
		if len(rows[i]) != want {
			t.Fatalf("row %d size = %d, want %d", i, len(rows[i]), want)
		}
	}
	if rows[0][0].Name != "Player 1" {
		t.Fatalf("goalkeeper row starts with %q", rows[0][0].Name)
	}
}

func TestFormationRowsFallbackByPosition(t *testing.T) {
	rows := formationRows("", []PlayerLineup{
		{Name: "Keeper", PosName: "POR"},
		{Name: "Back", PosName: "DFC"},
		{Name: "Mid", PosName: "MC"},
		{Name: "Forward", PosName: "DC"},
	})

	if len(rows) != 4 {
		t.Fatalf("row count = %d, want 4", len(rows))
	}
	if rows[0][0].Name != "Keeper" || rows[3][0].Name != "Forward" {
		t.Fatalf("unexpected fallback rows: %#v", rows)
	}
}

func TestPlayerTokenTruncatesToWidth(t *testing.T) {
	token := playerToken(PlayerLineup{Name: "Alexanderson", Number: "10"}, 8)
	if lipgloss.Width(token) > 8 {
		t.Fatalf("token width = %d, want <= 8: %q", lipgloss.Width(token), token)
	}
	if !strings.HasSuffix(token, "…") {
		t.Fatalf("token %q should be truncated", token)
	}
}

func TestRenderPitchNarrowWidth(t *testing.T) {
	md := MatchDetail{
		Details: &MatchDetailData{
			Lineup: LineupData{
				HomeFormation: "4-3-3",
				AwayFormation: "4-4-2",
				HomeStarters:  makePlayers(11),
				AwayStarters:  makePlayers(11),
			},
		},
	}

	lines := md.renderPitch(24, md.Details.Lineup)
	if len(lines) == 0 {
		t.Fatal("expected pitch lines")
	}
	for i, line := range lines {
		if lipgloss.Width(line) > 24 {
			t.Fatalf("line %d width = %d, want <= 24: %q", i, lipgloss.Width(line), line)
		}
	}
}

func TestRenderPitchMinimumWidth(t *testing.T) {
	md := MatchDetail{
		Details: &MatchDetailData{
			Lineup: LineupData{
				HomeFormation: "3-5-2",
				AwayFormation: "3-5-2",
				HomeStarters:  makePlayers(11),
				AwayStarters:  makePlayers(11),
			},
		},
	}

	lines := md.renderPitch(18, md.Details.Lineup)
	if len(lines) == 0 {
		t.Fatal("expected pitch lines")
	}
	for i, line := range lines {
		if lipgloss.Width(line) > 18 {
			t.Fatalf("line %d width = %d, want <= 18: %q", i, lipgloss.Width(line), line)
		}
	}
}

func makePlayers(n int) []PlayerLineup {
	players := make([]PlayerLineup, n)
	for i := range players {
		players[i] = PlayerLineup{
			Name:   "Player " + strconv.Itoa(i+1),
			Number: strconv.Itoa(i + 1),
		}
	}
	return players
}

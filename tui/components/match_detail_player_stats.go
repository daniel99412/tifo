package components

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tifo/internal/domain"

	"github.com/charmbracelet/lipgloss"
)

var (
	psRatingStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("215"))

	psPlayerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255"))

	psStatLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	psStatValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	psGKStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("75"))
)

type PlayerStatsView struct {
	ScrollHome int
	ScrollAway int
	Focus      int // 0=home, 1=away
}

func NewPlayerStatsView() *PlayerStatsView {
	return &PlayerStatsView{}
}

func (psv *PlayerStatsView) ScrollUp(step int) {
	switch psv.Focus {
	case 0:
		psv.ScrollHome -= step
		if psv.ScrollHome < 0 {
			psv.ScrollHome = 0
		}
	case 1:
		psv.ScrollAway -= step
		if psv.ScrollAway < 0 {
			psv.ScrollAway = 0
		}
	}
}

func (psv *PlayerStatsView) ScrollDown(step int) {
	switch psv.Focus {
	case 0:
		psv.ScrollHome += step
	case 1:
		psv.ScrollAway += step
	}
}

func (psv *PlayerStatsView) FocusHome() {
	psv.Focus = 0
}

func (psv *PlayerStatsView) FocusAway() {
	psv.Focus = 1
}

func (psv *PlayerStatsView) Reset() {
	psv.ScrollHome = 0
	psv.ScrollAway = 0
}

func (psv *PlayerStatsView) Render(
	playerStats []domain.PlayerStatItem,
	homeName, awayName string,
	homeColor, awayColor lipgloss.Color,
	width, height int,
) string {
	if len(playerStats) == 0 {
		return mdInfoStyle.Render("sin estadísticas de jugadores")
	}

	var homePlayers, awayPlayers []domain.PlayerStatItem
	for _, ps := range playerStats {
		if ps.Team == domain.SideHome {
			homePlayers = append(homePlayers, ps)
		} else {
			awayPlayers = append(awayPlayers, ps)
		}
	}

	homePlayers = filterEmptyPlayerStats(homePlayers)
	awayPlayers = filterEmptyPlayerStats(awayPlayers)

	sortPlayersByRating(homePlayers)
	sortPlayersByRating(awayPlayers)

	colW := (width - 3) / 2
	if colW < 30 {
		colW = 30
	}

	homeHeaderStyle := lipgloss.NewStyle().Bold(true).Width(colW).Align(lipgloss.Center)
	if psv.Focus == 0 {
		homeHeaderStyle = homeHeaderStyle.
			Background(homeColor).
			Foreground(lipgloss.Color("255"))
	} else {
		homeHeaderStyle = homeHeaderStyle.Foreground(homeColor)
	}
	awayHeaderStyle := lipgloss.NewStyle().Bold(true).Width(colW).Align(lipgloss.Center)
	if psv.Focus == 1 {
		awayHeaderStyle = awayHeaderStyle.
			Background(awayColor).
			Foreground(lipgloss.Color("255"))
	} else {
		awayHeaderStyle = awayHeaderStyle.Foreground(awayColor)
	}

	homeHeader := homeHeaderStyle.Render(homeName)
	awayHeader := awayHeaderStyle.Render(awayName)

	homeBody := renderPlayerList(homePlayers, colW, homeColor)
	awayBody := renderPlayerList(awayPlayers, colW, awayColor)

	bodyH := height - 2
	homeScrolled := applyColumnScroll(homeBody, colW, bodyH, &psv.ScrollHome)
	awayScrolled := applyColumnScroll(awayBody, colW, bodyH, &psv.ScrollAway)

	homeScrolled = ensureHeight(homeScrolled, bodyH)
	awayScrolled = ensureHeight(awayScrolled, bodyH)

	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("236")).
		Width(1).
		Height(bodyH)

	homeCol := lipgloss.JoinVertical(lipgloss.Top, homeHeader, homeScrolled)
	awayCol := lipgloss.JoinVertical(lipgloss.Top, awayHeader, awayScrolled)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(colW).Render(homeCol),
		sepStyle.Render("│"),
		lipgloss.NewStyle().Width(colW).Render(awayCol),
	)
}

func renderPlayerList(players []domain.PlayerStatItem, width int, teamColor lipgloss.Color) string {
	var lines []string
	playerStyle := lipgloss.NewStyle().Bold(true).Foreground(teamColor)

	for _, p := range players {
		rating := p.Rating
		if rating == "" {
			rating = "-"
		}
		if !strings.Contains(rating, ".") {
			if _, err := fmt.Sscanf(rating, "%f", new(float64)); err == nil {
			}
		}

		icon := "●"
		if p.IsGK {
			icon = psGKStyle.Render("◆")
		}

		header := fmt.Sprintf("%s %s  %s",
			psPlayerStyle.Render(icon),
			playerStyle.Render(p.Player),
			psRatingStyle.Render(rating),
		)
		lines = append(lines, header)

		for _, s := range p.Stats {
			label := psStatLabelStyle.Render("  " + s.Label)
			val := psStatValueStyle.Render(s.Value)
			line := fmt.Sprintf("%-"+fmt.Sprintf("%d", width-2)+"s", lipgloss.JoinHorizontal(lipgloss.Top, label, "  ", val))
			lines = append(lines, truncatePS(line, width))
		}

		if len(p.Stats) > 0 {
			lines = append(lines, "")
		}
	}

	return lipgloss.JoinVertical(lipgloss.Top, lines...)
}

func ensureHeight(s string, h int) string {
	lines := strings.Split(s, "\n")
	if len(lines) < h {
		return s + strings.Repeat("\n", h-len(lines))
	}
	return s
}

func truncatePS(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	runes := []rune(s)
	var out strings.Builder
	width := 0
	for _, r := range runes {
		rw := lipgloss.Width(string(r))
		if width+rw > w-1 {
			break
		}
		out.WriteRune(r)
		width += rw
	}
	out.WriteRune('…')
	return out.String()
}

func sortPlayersByRating(players []domain.PlayerStatItem) {
	sort.SliceStable(players, func(i, j int) bool {
		ri, rj := players[i].Rating, players[j].Rating
		if ri == "" && rj == "" {
			return false
		}
		if ri == "" {
			return false
		}
		if rj == "" {
			return true
		}
		fi, erri := strconv.ParseFloat(ri, 64)
		fj, errj := strconv.ParseFloat(rj, 64)
		if erri != nil && errj != nil {
			return false
		}
		if erri != nil {
			return false
		}
		if errj != nil {
			return true
		}
		return fi > fj
	})
}

func filterEmptyPlayerStats(players []domain.PlayerStatItem) []domain.PlayerStatItem {
	var out []domain.PlayerStatItem
	for _, p := range players {
		if p.Rating != "" || len(p.Stats) > 0 {
			out = append(out, p)
		}
	}
	return out
}

package espn

import (
	"strings"
	"time"
)

type LeagueMapping struct {
	FotmobNames []string
	ESPNLeague  string
}

var leagueMappings = []LeagueMapping{
	{FotmobNames: []string{"World Cup", "fifa.world", "WC"}, ESPNLeague: "fifa.world"},
	{FotmobNames: []string{"Club Friendlies", "Friendly", "fifa.friendly"}, ESPNLeague: "fifa.friendly"},
	{FotmobNames: []string{"Premier League", "eng.1", "EPL"}, ESPNLeague: "eng.1"},
	{FotmobNames: []string{"Championship", "eng.2", "EFL Champ"}, ESPNLeague: "eng.2"},
	{FotmobNames: []string{"FA Cup", "eng.fa"}, ESPNLeague: "eng.fa"},
	{FotmobNames: []string{"La Liga", "esp.1", "LALIGA"}, ESPNLeague: "esp.1"},
	{FotmobNames: []string{"La Liga 2", "esp.2"}, ESPNLeague: "esp.2"},
	{FotmobNames: []string{"Bundesliga", "ger.1"}, ESPNLeague: "ger.1"},
	{FotmobNames: []string{"2. Bundesliga", "ger.2"}, ESPNLeague: "ger.2"},
	{FotmobNames: []string{"Serie A", "ita.1"}, ESPNLeague: "ita.1"},
	{FotmobNames: []string{"Ligue 1", "fra.1"}, ESPNLeague: "fra.1"},
	{FotmobNames: []string{"Eredivisie", "ned.1"}, ESPNLeague: "ned.1"},
	{FotmobNames: []string{"Primeira Liga", "por.1"}, ESPNLeague: "por.1"},
	{FotmobNames: []string{"MLS", "usa.1"}, ESPNLeague: "usa.1"},
	{FotmobNames: []string{"Liga MX", "mex.1", "Liga BBVA MX"}, ESPNLeague: "mex.1"},
	{FotmobNames: []string{"Liga Profesional", "arg.1"}, ESPNLeague: "arg.1"},
	{FotmobNames: []string{"Serie A", "bra.1"}, ESPNLeague: "bra.1"},
	{FotmobNames: []string{"EURO", "uefa.euro", "European Championship"}, ESPNLeague: "uefa.euro"},
	{FotmobNames: []string{"UEFA Champions League", "Champions League", "uefa.champions", "UCL"}, ESPNLeague: "uefa.champions"},
	{FotmobNames: []string{"UEFA Europa League", "Europa League", "uefa.europa"}, ESPNLeague: "uefa.europa"},
	{FotmobNames: []string{"UEFA Conference League", "Conference League", "uefa.europa.conf"}, ESPNLeague: "uefa.europa.conf"},
	{FotmobNames: []string{"Copa Libertadores", "conmebol.libertadores"}, ESPNLeague: "conmebol.libertadores"},
	{FotmobNames: []string{"Copa Sudamericana", "conmebol.sudamericana"}, ESPNLeague: "conmebol.sudamericana"},
	{FotmobNames: []string{"Recopa Sudamericana", "conmebol.recopa"}, ESPNLeague: "conmebol.recopa"},
	{FotmobNames: []string{"AFC Champions League", "AFC Champions", "afc.champions"}, ESPNLeague: "afc.champions"},
	{FotmobNames: []string{"Saudi Pro League", "ksa.1"}, ESPNLeague: "ksa.1"},
	{FotmobNames: []string{"J.League", "jpn.1"}, ESPNLeague: "jpn.1"},
	{FotmobNames: []string{"Liga Argentina", "arg.1"}, ESPNLeague: "arg.1"},
	{FotmobNames: []string{"Copa Chile", "chi.copa"}, ESPNLeague: "chi.copa"},
	{FotmobNames: []string{"Primera División", "arg.1"}, ESPNLeague: "arg.1"},
	{FotmobNames: []string{"Copa del Rey", "esp.copa"}, ESPNLeague: "esp.copa"},
	{FotmobNames: []string{"DFB-Pokal", "ger.dfb"}, ESPNLeague: "ger.dfb"},
	{FotmobNames: []string{"Copa MX", "mex.copa"}, ESPNLeague: "mex.copa"},
	{FotmobNames: []string{"Campeón de Campeones", "mex.campeon"}, ESPNLeague: "mex.campeon"},
	{FotmobNames: []string{"Supercopa MX", "mex.supercopa"}, ESPNLeague: "mex.supercopa"},
}

func FotmobLeagueToESPN(leagueName string) string {
	lower := strings.ToLower(leagueName)
	var best string
	bestLen := 0
	for _, m := range leagueMappings {
		for _, name := range m.FotmobNames {
			nlower := strings.ToLower(name)
			if strings.Contains(lower, nlower) && len(nlower) > bestLen {
				best = m.ESPNLeague
				bestLen = len(nlower)
			}
		}
	}
	return best
}

type TeamNameNormalizer struct {
	pairs [][2]string
}

var defaultNormalizer = buildNormalizer()

func buildNormalizer() TeamNameNormalizer {
	pairs := [][2]string{
		{"ä", "ae"}, {"ö", "oe"}, {"ü", "ue"}, {"é", "e"}, {"è", "e"},
		{"ê", "e"}, {"í", "i"}, {"ó", "o"}, {"ú", "u"}, {"ñ", "n"},
		{"á", "a"}, {"ç", "c"}, {"'", ""}, {".", ""}, {"-", " "},
		{"fc", ""}, {"cf", ""}, {"ac", ""}, {"afc", ""}, {"fc ", ""},
		{"  ", " "},
	}
	return TeamNameNormalizer{pairs: pairs}
}

func NormalizeTeamName(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, p := range defaultNormalizer.pairs {
		lower = strings.ReplaceAll(lower, p[0], p[1])
	}
	lower = strings.TrimSpace(lower)
	return lower
}

func FuzzyTeamMatch(a, b string) bool {
	return NormalizeTeamName(a) == NormalizeTeamName(b)
}

func FindEventByMatch(events []ScoreboardEvent, utcTime time.Time, homeTeam, awayTeam string) *ScoreboardEvent {
	for _, ev := range events {
		for _, comp := range ev.Competitions {
			var home, away string
			for _, c := range comp.Competitors {
				if c.HomeAway == "home" {
					home = c.Team.DisplayName
				} else {
					away = c.Team.DisplayName
				}
			}

			if FuzzyTeamMatch(home, homeTeam) && FuzzyTeamMatch(away, awayTeam) {
				ev := ev
				return &ev
			}
			if FuzzyTeamMatch(home, awayTeam) && FuzzyTeamMatch(away, homeTeam) {
				ev := ev
				return &ev
			}
		}
	}
	return nil
}

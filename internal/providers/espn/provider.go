package espn

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"tifo/internal/domain"
	"tifo/internal/resolver"
	oldESPN "tifo/espn"
	"time"
)

type Provider struct {
	svc *oldESPN.Service
	mr  *resolver.MatchResolver
	tr  *resolver.TeamResolver
}

func NewProvider(svc *oldESPN.Service) *Provider {
	return &Provider{svc: svc}
}

func (p *Provider) Name() string  { return "espn" }
func (p *Provider) Priority() int { return 10 }

func (p *Provider) Leagues(_ context.Context, _, _ string) ([]domain.Competition, error) {
	return nil, nil
}

func (p *Provider) LeagueMatches(_ context.Context, _ string) ([]domain.Match, error) {
	return nil, nil
}

func (p *Provider) MatchDetails(_ context.Context, _ string) (*domain.MatchDetails, error) {
	return nil, fmt.Errorf("espn: MatchDetails not supported")
}

func normalizePlayerName(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	repl := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u",
		"ü", "u", "ñ", "n", "ç", "c",
		"ä", "a", "ë", "e", "ö", "o", "ï", "i",
		"â", "a", "ê", "e", "ô", "o", "î", "i", "û", "u",
		"ã", "a", "õ", "o",
		".", "", "-", "", "'", "",
	)
	return strings.TrimSpace(repl.Replace(lower))
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func namesMatch(a, b string) bool {
	if a == b {
		return true
	}
	aTokens := strings.Fields(a)
	bTokens := strings.Fields(b)
	shorter, longer := aTokens, bTokens
	if len(shorter) > len(longer) {
		shorter, longer = longer, shorter
	}
	if len(shorter) < 2 {
		return false
	}
	if len(shorter) == len(longer) {
		for i := range shorter {
			dist := levenshtein(shorter[i], longer[i])
			if dist > 2 {
				log.Printf("[ESPN] namesMatch lev fail: %q vs %q → dist=%d > 2", shorter[i], longer[i], dist)
				return false
			}
		}
		log.Printf("[ESPN] namesMatch lev ok: a=%q b=%q", a, b)
		return true
	}
	longerSet := make(map[string]bool, len(longer))
	for _, t := range longer {
		longerSet[t] = true
	}
	for _, t := range shorter {
		if !longerSet[t] {
			return false
		}
	}
	return true
}

func enrichPlayersFromRoster(players []domain.PlayerRef, rosterNamePos map[string]string) {
	for i := range players {
		key := normalizePlayerName(players[i].Name)
		matched := false
		if posName, ok := rosterNamePos[key]; ok && posName != "" {
			players[i].PosName = posName
			log.Printf("[ESPN] player %q exact match → %s (key=%q)", players[i].Name, posName, key)
			continue
		}
		for espnKey, posName := range rosterNamePos {
			if posName != "" && namesMatch(key, espnKey) {
				players[i].PosName = posName
				log.Printf("[ESPN] player %q fuzzy match → %s (fotmobKey=%q, espnKey=%q)", players[i].Name, posName, key, espnKey)
				matched = true
				break
			}
		}
		if !matched {
			log.Printf("[ESPN] player %q NO match in roster (normalized=%q, roster=%v)", players[i].Name, key, mapKeys(rosterNamePos))
		}
	}
}

func mapKeys(m map[string]string) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (p *Provider) enrichPositions(fotmobDetails *domain.MatchDetails, rosters []oldESPN.SummaryRoster) {
	if fotmobDetails.Lineups == nil || len(rosters) == 0 {
		log.Printf("[ESPN] enrichPositions: no lineups or rosters")
		return
	}

	for _, r := range rosters {
		rosterPos := make(map[string]string)
		for _, player := range r.Roster {
			abbr := player.Position.Abbreviation
			if abbr == "" || abbr == "SUB" {
				continue
			}
			key := normalizePlayerName(player.Athlete.DisplayName)
			rosterPos[key] = abbr
		}

		isHome := r.HomeAway == "home"
		log.Printf("[ESPN] enrichPositions side=%s roster=%v", r.HomeAway, rosterPos)
		if isHome {
			enrichPlayersFromRoster(fotmobDetails.Lineups.HomeStarters, rosterPos)
			enrichPlayersFromRoster(fotmobDetails.Lineups.HomeSubs, rosterPos)
		} else {
			enrichPlayersFromRoster(fotmobDetails.Lineups.AwayStarters, rosterPos)
			enrichPlayersFromRoster(fotmobDetails.Lineups.AwaySubs, rosterPos)
		}
	}
}

func (p *Provider) EnrichMatch(matchID int, leagueName string, utcTime time.Time, homeTeam, awayTeam string, fotmobDetails *domain.MatchDetails) *domain.MatchDetails {
	if fotmobDetails == nil {
		return nil
	}

	data, err := p.svc.FetchMatch(matchID, leagueName, utcTime, homeTeam, awayTeam)
	if err != nil {
		return fotmobDetails
	}

	out := *fotmobDetails

	if data.Summary.GameInfo.Venue.FullName != "" {
		out.ExtraInfo.Venue = data.Summary.GameInfo.Venue.FullName
		city := data.Summary.GameInfo.Venue.Address.City
		country := data.Summary.GameInfo.Venue.Address.Country
		if city != "" && country != "" {
			out.ExtraInfo.Venue = fmt.Sprintf("%s, %s, %s", data.Summary.GameInfo.Venue.FullName, city, country)
		} else if city != "" {
			out.ExtraInfo.Venue = fmt.Sprintf("%s, %s", data.Summary.GameInfo.Venue.FullName, city)
		}
		if data.Summary.GameInfo.Venue.Capacity > 0 {
			out.ExtraInfo.VenueCapacity = data.Summary.GameInfo.Venue.Capacity
		}
	}

	if data.Summary.GameInfo.Attendance > 0 {
		out.ExtraInfo.Attendance = data.Summary.GameInfo.Attendance
	}

	for _, off := range data.Summary.GameInfo.Officials {
		if off.Position.ID == "1" {
			out.ExtraInfo.Referee = off.DisplayName
			break
		}
	}

	if data.Summary.GameInfo.Weather.DisplayValue != "" {
		out.ExtraInfo.Weather = data.Summary.GameInfo.Weather.DisplayValue
	} else if data.Summary.GameInfo.Weather.Condition != "" {
		w := data.Summary.GameInfo.Weather.Condition
		temp := data.Summary.GameInfo.Weather.Temperature
		if temp == 0 {
			temp = data.Summary.GameInfo.Weather.High
		}
		if temp > 0 {
			w = fmt.Sprintf("%s, %.0f°C", w, temp)
		}
		out.ExtraInfo.Weather = w
	}

	for _, b := range data.Summary.Broadcasts {
		if b.Media != nil && b.Media.Name != "" {
			out.ExtraInfo.Broadcasts = append(out.ExtraInfo.Broadcasts, b.Media.Name)
		}
	}

	var homeTeamName string
	for _, comp := range data.Summary.Header.Competitions {
		for _, c := range comp.Competitors {
			if c.HomeAway == "home" {
				homeTeamName = c.Team.DisplayName
				if out.ExtraInfo.HomeColor == "" {
					out.ExtraInfo.HomeColor = c.Team.Color
				}
				if c.ShootoutScore != nil {
					homePS := int(*c.ShootoutScore)
					out.Match.PenScore = fmt.Sprintf("%d", homePS)
					log.Printf("[espn] home shootoutScore=%.0f → PenScore=%s", *c.ShootoutScore, out.Match.PenScore)
				}
			} else {
				if out.ExtraInfo.AwayColor == "" {
					out.ExtraInfo.AwayColor = c.Team.Color
				}
				if c.ShootoutScore != nil {
					awayPS := int(*c.ShootoutScore)
					if out.Match.PenScore != "" {
						out.Match.PenScore = fmt.Sprintf("%s-%d", out.Match.PenScore, awayPS)
					} else {
						out.Match.PenScore = fmt.Sprintf("0-%d", awayPS)
					}
					log.Printf("[espn] away shootoutScore=%.0f → PenScore=%s", *c.ShootoutScore, out.Match.PenScore)
				}
			}
		}
	}

	// Individual penalty shots from shootout data
	if len(data.Summary.Shootout) > 0 {
		log.Printf("[espn] shootout data found: %d teams", len(data.Summary.Shootout))
		var shots []domain.PenShot
		for _, team := range data.Summary.Shootout {
			teamSide := domain.SideAway
			if team.Team == homeTeamName || team.Team == homeTeam {
				teamSide = domain.SideHome
			}
			log.Printf("[espn] shootout team=%q side=%v shots=%d", team.Team, teamSide, len(team.Shots))
			for _, s := range team.Shots {
				shots = append(shots, domain.PenShot{
					Team:   teamSide,
					Player: s.Player,
					Scored: s.DidScore,
				})
				log.Printf("[espn]   shot %d: player=%q scored=%v", s.ShotNumber, s.Player, s.DidScore)
			}
		}
		out.Match.PenShootout = shots
		if len(shots) > 0 {
			log.Printf("[espn] PenShootout set with %d shots", len(shots))
		}
	} else {
		log.Printf("[espn] no shootout data in response")
	}

	if fotmobDetails.Events == nil {
		fotmobDetails.Events = []domain.MatchEvent{}
	}
	out.Events = p.mapExtraEvents(data, fotmobDetails.Events, homeTeam, awayTeam)

	out.StatsByPeriod = p.mapStats(data, fotmobDetails.StatsByPeriod)

	p.enrichPositions(&out, data.Summary.Rosters)

	// H2H enrichment: form + record
	if out.H2H == nil {
		out.H2H = &domain.H2H{}
	}
	for _, comp := range data.Summary.Header.Competitions {
		for _, c := range comp.Competitors {
			rec := ""
			for _, r := range c.Record {
				if r.Type == "total" {
					rec = r.Summary
					break
				}
			}
			if c.HomeAway == "home" {
				out.H2H.HomeRecord = rec
			} else {
				out.H2H.AwayRecord = rec
			}
		}
	}
	if data.Summary.Boxscore.Form != nil {
		log.Printf("[espn] Boxscore.Form type=%T", data.Summary.Boxscore.Form)
		var entries []interface{}
		switch v := data.Summary.Boxscore.Form.(type) {
		case []interface{}:
			entries = v
		case map[string]interface{}:
			entries = []interface{}{v}
		default:
			log.Printf("[espn] Boxscore.Form tipo inesperado %T, omitiendo", data.Summary.Boxscore.Form)
		}
		for _, entry := range entries {
			em, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			teamRaw, _ := em["team"].(map[string]interface{})
			teamName, _ := teamRaw["displayName"].(string)
			if teamName == "" {
				continue
			}
			var formEvents []domain.H2HFormEvent
			eventsRaw, _ := em["events"].([]interface{})
			for _, evRaw := range eventsRaw {
				ev, _ := evRaw.(map[string]interface{})
				res, _ := ev["gameResult"].(string)
				score, _ := ev["score"].(string)
				oppRaw, _ := ev["opponent"].(map[string]interface{})
				oppName, _ := oppRaw["displayName"].(string)
				if res != "" {
					formEvents = append(formEvents, domain.H2HFormEvent{
						Opponent: oppName,
						Score:    score,
						Result:   res,
					})
				}
			}
			if len(formEvents) > 0 {
				if teamName == homeTeam || strings.Contains(homeTeam, teamName) || strings.Contains(teamName, homeTeam) {
					out.H2H.HomeForm = formEvents
				} else if teamName == awayTeam || strings.Contains(awayTeam, teamName) || strings.Contains(teamName, awayTeam) {
					out.H2H.AwayForm = formEvents
				}
			}
		}
	} else {
		log.Printf("[espn] Boxscore.Form es nil")
	}
	log.Printf("[espn] HeadToHeadGames count=%d", len(data.Summary.HeadToHeadGames))
	// H2H historical matches from ESPN headToHeadGames.
	// Deduplicate against existing FotMob matches by date + teams.
	existing := make(map[string]bool)
	for _, m := range out.H2H.Matches {
		key := h2hMatchKey(m)
		existing[key] = true
	}
	uniq := make(map[string]bool)
	for _, teamEntry := range data.Summary.HeadToHeadGames {
		for _, ev := range teamEntry.Events {
			if uniq[ev.ID] {
				continue
			}
			uniq[ev.ID] = true
			var date time.Time
			if t, err := time.Parse("2006-01-02T15:04Z", ev.GameDate); err == nil {
				date = t
			} else if t, err := time.Parse(time.RFC3339, ev.GameDate); err == nil {
				date = t
			}
			hs, _ := strconv.Atoi(ev.HomeTeamScore)
			as, _ := strconv.Atoi(ev.AwayTeamScore)
			homeName := teamEntry.Team.DisplayName
			awayName := ev.Opponent.DisplayName
			if ev.AtVs == "@" {
				homeName, awayName = awayName, homeName
			}
			candidate := domain.H2HMatchDetail{
				Date:        date,
				HomeTeam:    homeName,
				AwayTeam:    awayName,
				HomeScore:   hs,
				AwayScore:   as,
				Competition: ev.CompetitionName,
			}
			if existing[h2hMatchKey(candidate)] {
				continue
			}
			out.H2H.Matches = append(out.H2H.Matches, candidate)
		}
	}

	// Recompute H2H summary from actual matches using current match orientation.
		if len(out.H2H.Matches) > 0 {
		hw, d, aw := 0, 0, 0
		for _, m := range out.H2H.Matches {
			switch {
			case m.HomeScore > m.AwayScore:
				if teamMatch(m.HomeTeam, homeTeam) {
					hw++
				} else if teamMatch(m.HomeTeam, awayTeam) {
					aw++
				}
			case m.HomeScore < m.AwayScore:
				if teamMatch(m.AwayTeam, awayTeam) {
					aw++
				} else if teamMatch(m.AwayTeam, homeTeam) {
					hw++
				}
			default:
				d++
			}
		}
		out.H2H.HomeWins = hw
		out.H2H.Draws = d
		out.H2H.AwayWins = aw
	}

	return &out
}

func teamMatch(a, b string) bool {
	return strings.EqualFold(a, b) || strings.Contains(strings.ToLower(a), strings.ToLower(b)) || strings.Contains(strings.ToLower(b), strings.ToLower(a))
}

func (p *Provider) mapExtraEvents(data *oldESPN.EnrichData, existing []domain.MatchEvent, homeTeam, awayTeam string) []domain.MatchEvent {
	// Track the last delay type so end-delay knows if it follows an injury
	var lastDelayType domain.EventType
	delayTypeAtMinute := make(map[int]domain.EventType)
	espnCreated := make(map[string]bool) // track events created by ESPN in this pass

	for _, ke := range data.Summary.KeyEvents {
		typ := classify(ke)
		if typ == "" {
			continue
		}
		if typ == domain.EvVideoReview || typ == domain.EvVAR {
			log.Printf("[espn] VAR event detected: type=%q text=%q shortText=%q", ke.Type.Type, ke.Text, ke.ShortText)
		}
		minute, added := parseClock(ke.Clock.DisplayValue)

		// Determine if this is a matchable type (provider-shared) or unique ESPN type
		isMatchable := isMatchableType(ke.Type.Type, typ)

		if ke.Type.Type == "start-delay" {
			// Use consistent type for same-minute start-delay (paired team events)
			if t, ok := delayTypeAtMinute[minute]; ok {
				typ = t
			} else {
				if containsStr(strings.ToLower(ke.Text), "injury") {
					typ = domain.EvInjury
				}
				delayTypeAtMinute[minute] = typ
			}
			lastDelayType = typ
		} else if typ == domain.EvVAR || typ == domain.EvVideoReview {
			lastDelayType = typ
		}

		// Build candidate event
		desc := ke.Text
		if desc == "" {
			desc = typeLabel(typ)
		}
		pauseType := classifyPauseType(ke)

		// Empty-text start-delay at minute 0 is pre-match delay, not VAR
		if typ == domain.EvPausa && pauseType == "var" {
			pauseType = "other"
		}

		ev := domain.MatchEvent{
			Minute:       minute,
			AddedTime:    added,
			EventType:    typ,
			Period:       ke.Period.Number,
			Detail:       desc,
			SortTime:     minute,
			SortOverload: added,
			PauseType:    pauseType,
			DelayText:    ke.Text,
		}

		// Set GoalType for goals
		if ke.Type.Type == "goal---header" {
			ev.GoalType = "header"
		} else if ke.Type.Type == "penalty---scored" {
			ev.GoalType = "penalty"
		} else if ke.Type.Type == "goal" {
			ev.GoalType = "regular"
		}

		// Set CardType
		if ke.Type.Type == "yellow-card" {
			ev.CardType = "Yellow"
			ev.Player = extractPlayerFromText(ke.Text, "shown the")
		} else if ke.Type.Type == "red-card" {
			ev.CardType = "Red"
			ev.Player = extractPlayerFromText(ke.Text, "shown the")
		}

		// Set sub out/in for substitutions
		if ke.Type.Type == "substitution" {
			ev.SubOut, ev.SubIn = parseESPNSub(ke.Text)
		}

		// Extract player from injury delay text
		if typ == domain.EvInjury {
			ev.Player = extractPlayerFromText(ke.Text, "injury")
		}

		// Try to match against existing events
		if isMatchable {
			matched := false
			for i := range existing {
				if domain.IdentityMatch(existing[i], ev) {
					// Also check second-level match for cards: same card color
					if typ == domain.EvCard || typ == "yellow-card" || typ == "red-card" {
						if existing[i].CardType != "" && existing[i].CardType != ev.CardType {
							continue
						}
					}
					domain.MergeEvents(&existing[i], &ev)
					matched = true
					break
				}
			}
			if matched {
				continue
			}
		}

		// No match — check if this is a duplicate ESPN event (same minute + same type)
		dedupKey := fmt.Sprintf("%d:%s", minute, string(typ))
		if espnCreated[dedupKey] {
			continue
		}

		// Build final description
		switch typ {
		case domain.EvPausa:
			if ev.PauseType == "hydration" && !strings.Contains(desc, "drinks break") {
				desc = "Pausa de hidratación"
			}
			ev.Detail = desc
		case domain.EvInjury:
			ev.Detail = desc
		case domain.EvContinua:
			switch lastDelayType {
			case domain.EvInjury:
				desc = "Se reanuda después de lesión"
			case domain.EvVAR, domain.EvVideoReview:
				desc = "Se reanuda después de revisión VAR"
			default:
				desc = "Se reanuda después de pausa de hidratación"
			}
			ev.Detail = desc
		}

		existing = append(existing, ev)
		espnCreated[dedupKey] = true
	}

	return existing
}

func classifyPauseType(ke oldESPN.KeyEvent) string {
	if ke.Type.Type != "start-delay" {
		return ""
	}
	text := strings.ToLower(ke.Text)
	switch {
	case containsStr(text, "drink") || containsStr(text, "hydration") || containsStr(text, "agua"):
		return "hydration"
	case containsStr(text, "injury"):
		return "injury"
	default:
		return "var"
	}
}

func isMatchableType(espnType string, domainType domain.EventType) bool {
	switch espnType {
	case "goal", "goal---header", "penalty---scored":
		return true
	case "yellow-card", "red-card":
		return true
	case "substitution":
		return true
	case "halftime":
		return true
	case "end-regular-time":
		return true
	}
	return false
}

func parseESPNSub(text string) (out, in *domain.PlayerRef) {
	if !containsStr(text, "replaces") {
		return nil, nil
	}
	// Format: "Substitution, Team. PlayerIn replaces PlayerOut."
	parts := strings.SplitN(text, "replaces", 2)
	if len(parts) != 2 {
		return nil, nil
	}
	inName := extractPlayerName(parts[0])
	outName := extractPlayerName(parts[1])
	if inName != "" {
		in = &domain.PlayerRef{Name: inName}
	}
	if outName != "" {
		out = &domain.PlayerRef{Name: outName}
	}
	return
}

func extractPlayerFromText(text, after string) *domain.PlayerRef {
	if text == "" {
		return nil
	}
	// Cards: "Declan Rice (England) is shown the yellow card for a bad foul."
	if idx := strings.Index(text, "("); idx > 0 {
		// Try to find the player name after "injury " keyword
		before := text[:idx]
		if injIdx := strings.LastIndex(before, "injury "); injIdx >= 0 {
			name := strings.TrimSpace(before[injIdx+7:])
			return &domain.PlayerRef{Name: name}
		}
		name := strings.TrimSpace(before)
		return &domain.PlayerRef{Name: name}
	}
	return nil
}
func extractPlayerName(s string) string {
	// Find last word before the period
	s = strings.TrimSpace(s)
	if idx := strings.LastIndex(s, "."); idx >= 0 {
		s = s[idx+1:]
	}
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ".")
	s = strings.TrimSuffix(s, ")")
	return strings.TrimSpace(s)
}

func (p *Provider) mapStats(data *oldESPN.EnrichData, existing domain.StatsByPeriod) domain.StatsByPeriod {
	if len(data.Summary.Boxscore.Teams) < 2 {
		return existing
	}
	if existing == nil {
		existing = domain.StatsByPeriod{}
	}

	espnByName := make(map[string][2]string)
	homeStats := data.Summary.Boxscore.Teams[0].Statistics
	awayStats := data.Summary.Boxscore.Teams[1].Statistics
	for i, hs := range homeStats {
		hVal := normalizeESPNVal(hs.Name, hs.DisplayValue)
		aVal := ""
		if i < len(awayStats) && awayStats[i].Name == hs.Name {
			aVal = normalizeESPNVal(hs.Name, awayStats[i].DisplayValue)
		}
		espnByName[hs.Name] = [2]string{hVal, aVal}
	}

	espnFotmobKey := map[string]string{
		"possessionPct":     "Ball possession",
		"totalShots":        "Total shots",
		"shotsOnTarget":     "Shots on target",
		"offsides":          "Offsides",
		"wonCorners":        "Corner kicks",
		"yellowCards":       "Yellow cards",
		"redCards":          "Red cards",
		"foulsCommitted":    "Fouls",
		"saves":             "Saves",
		"totalPasses":       "Total passes",
		"accuratePasses":    "Accurate passes",
		"passPct":           "Pass accuracy",
		"totalCrosses":      "Total crosses",
		"accurateCrosses":   "Accurate crosses",
		"aerialsWon":        "Aerials won",
		"totalAerials":      "Total aerials",
		"aerialPct":         "Aerial success",
		"totalLongBalls":    "Long balls",
		"accurateLongBalls": "Accurate long balls",
		"tacklesTotal":      "Tackles",
		"interceptions":     "Interceptions",
		"clearances":        "Clearances",
		"totalDuels":        "Duels",
		"duelsWonPct":       "Duels won",
		"goalKicks":         "Goal kicks",
		"throwIns":          "Throw-ins",
		"punches":           "Punches",
	}

	// ESPN only provides full-match stats, so fill gaps in PeriodAll.
	allCats, ok := existing[domain.PeriodAll]
	if !ok {
		return existing
	}
	for espnName, fotmobKey := range espnFotmobKey {
		vals, ok := espnByName[espnName]
		if !ok {
			continue
		}
		for ci, cat := range allCats {
			for si, s := range cat.Stats {
				if s.Key == fotmobKey {
					if s.Home == "" {
						allCats[ci].Stats[si].Home = vals[0]
						allCats[ci].Stats[si].HomeProvider = "espn"
					}
					if s.Away == "" {
						allCats[ci].Stats[si].Away = vals[1]
						allCats[ci].Stats[si].AwayProvider = "espn"
					}
				}
			}
		}
	}
	existing[domain.PeriodAll] = allCats
	return existing
}

func classify(ke oldESPN.KeyEvent) domain.EventType {
	switch ke.Type.Type {
	case "kickoff":
		return domain.EvKO
	case "halftime":
		return domain.EvHT
	case "start-2nd-half":
		return domain.EvS2
	case "start-delay":
		text := strings.ToLower(ke.Text)
		switch {
		case containsStr(text, "injury"):
			return domain.EvInjury
		case containsStr(text, "drink") || containsStr(text, "hydration") || containsStr(text, "agua"):
			return domain.EvPausa
		case text == "" || containsVARorVideo(ke.Text):
			// Empty text start-delay at minute 0 = pre-match delay (rain, etc), not VAR
			if text == "" {
				minute, _ := parseClock(ke.Clock.DisplayValue)
				if minute <= 0 {
					return domain.EvPausa
				}
			}
			return domain.EvVideoReview
		default:
			return domain.EvPausa
		}
	case "end-delay":
		return domain.EvContinua
	case "start-extra-time":
		if ke.Shootout {
			return domain.EvPenShootout
		}
		return domain.EvAETStart
	case "video-review":
		return domain.EvVideoReview
	case "var":
		return domain.EvVAR
	default:
		if ke.Text != "" && containsVARorVideo(ke.Text) {
			return domain.EvVideoReview
		}
		return ""
	}
}

func containsVARorVideo(text string) bool {
	lower := strings.ToLower(text)
	// Must be surrounded by spaces or at boundaries to avoid matching player names
	// like "Álvarez", "Varela", "Vargas" etc.
	return strings.Contains(lower, " var ") || strings.HasPrefix(lower, "var ") ||
		strings.Contains(lower, " review ") || strings.HasPrefix(lower, "review ")
}

func parseClock(display string) (minute, added int) {
	if display == "" {
		return 0, 0
	}
	var m, a int
	n, err := fmt.Sscanf(display, "%d'+%d", &m, &a)
	if err == nil && n >= 1 {
		return m, a
	}
	n, err = fmt.Sscanf(display, "%d", &m)
	if err == nil && n == 1 {
		return m, 0
	}
	return 0, 0
}

func typeLabel(typ domain.EventType) string {
	switch typ {
	case domain.EvKO:
		return "Inicio del partido"
	case domain.EvS2:
		return "Inicio 2do tiempo"
	case domain.EvPausa:
		return "Pausa"
	case domain.EvInjury:
		return "Lesión"
	case domain.EvContinua:
		return "Se reanuda"
	case domain.EvHT:
		return "Descanso"
	}
	return string(typ)
}

func normalizeESPNVal(name, val string) string {
	if val == "" {
		return val
	}
	pctStats := map[string]bool{
		"passPct": true, "shotPct": true, "crossPct": true,
		"longBallPct": true, "aerialPct": true, "duelsWonPct": true,
	}
	if pctStats[name] {
		var v float64
		if _, err := fmt.Sscanf(val, "%f", &v); err == nil && v < 1 {
			return strconv.Itoa(int(v * 100))
		}
	}
	return val
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func h2hMatchKey(m domain.H2HMatchDetail) string {
	dateStr := ""
	if !m.Date.IsZero() {
		dateStr = m.Date.Format("2006-01-02")
	}
	return dateStr + "|" + strings.ToLower(m.HomeTeam) + "|" + strings.ToLower(m.AwayTeam)
}

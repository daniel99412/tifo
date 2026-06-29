# Graph Report - .  (2026-06-28)

## Corpus Check
- Corpus is ~21,604 words - fits in a single context window. You may not need a graph.

## Summary
- 376 nodes · 576 edges · 16 communities detected
- Extraction: 79% EXTRACTED · 21% INFERRED · 0% AMBIGUOUS · INFERRED: 122 edges (avg confidence: 0.8)
- Token cost: 25,000 input · 1,800 output

## Community Hubs (Navigation)
- [[_COMMUNITY_FotMob Types|FotMob Types]]
- [[_COMMUNITY_TUI Model & State|TUI Model & State]]
- [[_COMMUNITY_Service Layer & Enrichment|Service Layer & Enrichment]]
- [[_COMMUNITY_Match Detail UI Components|Match Detail UI Components]]
- [[_COMMUNITY_HTTP Client Layer|HTTP Client Layer]]
- [[_COMMUNITY_FotMob Data Fetching|FotMob Data Fetching]]
- [[_COMMUNITY_ESPN Response Types|ESPN Response Types]]
- [[_COMMUNITY_Domain Models|Domain Models]]
- [[_COMMUNITY_App Entry & Navigation|App Entry & Navigation]]
- [[_COMMUNITY_FotMob Provider Adapter|FotMob Provider Adapter]]
- [[_COMMUNITY_ESPN Provider Adapter|ESPN Provider Adapter]]
- [[_COMMUNITY_Persistence & Resolver|Persistence & Resolver]]
- [[_COMMUNITY_Provider Interfaces|Provider Interfaces]]
- [[_COMMUNITY_ESPN API Reference|ESPN API Reference]]
- [[_COMMUNITY_Center Layout Component|Center Layout Component]]
- [[_COMMUNITY_League Data|League Data]]

## God Nodes (most connected - your core abstractions)
1. `MatchDetail` - 17 edges
2. `Provider` - 15 edges
3. `Client` - 14 edges
4. `Calendar` - 13 edges
5. `Service` - 13 edges
6. `Model` - 11 edges
7. `buildService()` - 10 edges
8. `Provider` - 9 edges
9. `Format` - 9 edges
10. `New()` - 7 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `New()`  [INFERRED]
  main.go → tui/model.go
- `New()` --calls--> `NewClient()`  [INFERRED]
  tui/model.go → espn/client.go
- `buildService()` --calls--> `OpenMappingDB()`  [INFERRED]
  tui/model.go → internal/persistence/sqlite/mapping.go
- `buildService()` --calls--> `NewMatchResolver()`  [INFERRED]
  tui/model.go → internal/resolver/resolver.go
- `buildService()` --calls--> `NewTeamResolver()`  [INFERRED]
  tui/model.go → internal/resolver/resolver.go

## Hyperedges (group relationships)
- **Known Issues and Workarounds Pattern** — espn_api_complete_known_issues, espn_api_complete_site_api, espn_api_complete_core_api_v2 [EXTRACTED 0.90]

## Communities

### Community 0 - "FotMob Types"
Cohesion: 0.04
Nodes (45): AllLeagueCountry, AllLeagueItem, AllLeaguesResponse, ColorPair, EventPlayer, EventSwap, GoalEvent, H2HData (+37 more)

### Community 1 - "TUI Model & State"
Cohesion: 0.07
Nodes (19): LeagueList, Sidebar, Tabs, Format, NewMatchDetail(), buildFromDomain(), fetchLeagues(), formatMatch() (+11 more)

### Community 2 - "Service Layer & Enrichment"
Cohesion: 0.09
Nodes (22): ExternalID, ExternalIDs, TIFOID, Enricher, MergeConfig, MergeRule, DefaultMergeConfig(), eventKey() (+14 more)

### Community 3 - "Match Detail UI Components"
Cohesion: 0.13
Nodes (15): EventData, EventItem, H2HData, InjuriesData, InjuryPlayer, LineupData, MatchDetail, MatchDetailData (+7 more)

### Community 4 - "HTTP Client Layer"
Cohesion: 0.09
Nodes (17): NewIDCache(), NewClient(), Client, EnrichData, IDCache, LeagueMapping, LookupService, ResolvedEvent (+9 more)

### Community 5 - "FotMob Data Fetching"
Cohesion: 0.12
Nodes (3): Client, Service, fetchMatchDetails()

### Community 6 - "ESPN Response Types"
Cohesion: 0.07
Nodes (27): Boxscore, BroadcastMedia, BroadcastType, Clock, CommentaryItem, GameInfo, KeyEvent, Odds (+19 more)

### Community 7 - "Domain Models"
Cohesion: 0.08
Nodes (25): Competition, CompetitionRef, EventType, ExtraEvent, H2H, InjuryItem, LeaguePage, Lineups (+17 more)

### Community 8 - "App Entry & Navigation"
Cohesion: 0.15
Nodes (7): NewCalendar(), Calendar, LeagueItem, Location, NewLeagueList(), main(), New()

### Community 9 - "FotMob Provider Adapter"
Cohesion: 0.15
Nodes (9): EventType, LeagueMatches, Provider, fetchMatches(), mapFotmobStatus(), parseIntPtr(), parseUTCTime(), playerRef() (+1 more)

### Community 10 - "ESPN Provider Adapter"
Cohesion: 0.13
Nodes (9): Provider, classify(), containsStr(), normalizeESPNVal(), parseClock(), stringsContains(), typeLabel(), MatchContext (+1 more)

### Community 11 - "Persistence & Resolver"
Cohesion: 0.18
Nodes (5): Client, OpenMappingDB(), fetchIPLocation(), EntityMap, MappingDB

### Community 12 - "Provider Interfaces"
Cohesion: 0.2
Nodes (8): WithTimeout(), EnrichmentProvider, LineupProvider, MatchTimelineProvider, Provider, ProviderOption, ShotMapProvider, StatProvider

### Community 13 - "ESPN API Reference"
Cohesion: 0.27
Nodes (10): ESPN API Reference, CDN Game Package (cdn.espn.com/core), Common API v3 (site.web.api.espn.com/common/v3), Core API v2 (sports.core.api.espn.com/v2), Known Issues and Workarounds, Now API (now.core.api.espn.com/v1), Scoreboard Response Schema, Site API (site.api.espn.com) (+2 more)

### Community 14 - "Center Layout Component"
Cohesion: 0.5
Nodes (1): Center

### Community 15 - "League Data"
Cohesion: 1.0
Nodes (1): GroupedLeague

## Knowledge Gaps
- **132 isolated node(s):** `initSvcMsg`, `locationMsg`, `leaguesMsg`, `matchesMsg`, `detailsMsg` (+127 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `League Data`** (2 nodes): `GroupedLeague`, `leagues.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Format` connect `TUI Model & State` to `HTTP Client Layer`, `FotMob Data Fetching`, `ESPN Response Types`?**
  _High betweenness centrality (0.216) - this node is a cross-community bridge._
- **Why does `LeagueMatches` connect `FotMob Provider Adapter` to `FotMob Types`, `ESPN Provider Adapter`?**
  _High betweenness centrality (0.202) - this node is a cross-community bridge._
- **Why does `fetchMatches()` connect `FotMob Provider Adapter` to `TUI Model & State`?**
  _High betweenness centrality (0.138) - this node is a cross-community bridge._
- **What connects `initSvcMsg`, `locationMsg`, `leaguesMsg` to the rest of the system?**
  _132 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `FotMob Types` be split into smaller, more focused modules?**
  _Cohesion score 0.04 - nodes in this community are weakly interconnected._
- **Should `TUI Model & State` be split into smaller, more focused modules?**
  _Cohesion score 0.07 - nodes in this community are weakly interconnected._
- **Should `Service Layer & Enrichment` be split into smaller, more focused modules?**
  _Cohesion score 0.09 - nodes in this community are weakly interconnected._
# Plan: Pre-resolución cruzada de IDs + persistencia unificada

## Arquitectura

### Tabla única con JSON blob

Nueva tabla en SQLite que reemplaza el modelo de filas separadas:

```sql
CREATE TABLE IF NOT EXISTS entity_mappings (
    tifo_id     TEXT NOT NULL,
    entity_type TEXT NOT NULL,  -- "tournament", "match", "team", "player"
    name        TEXT NOT NULL DEFAULT '',
    ids_json    TEXT NOT NULL DEFAULT '{}',  -- {"fotmob":"17","sofascore":"17","espn":"16"}
    updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (entity_type, tifo_id)
);
```

Ejemplo de fila:
```
tifo_id: "fifa_world_cup_2026"
entity_type: "tournament"
name: "FIFA World Cup 2026"
ids_json: {"fotmob":"894789","sofascore":"16","espn":"fifa.world"}
```

### Modelo Go

```go
// internal/domain/mapping.go
type EntityMapping struct {
    TIFOID     string            `json:"tifoId"`
    EntityType string            `json:"type"`
    Name       string            `json:"name,omitempty"`
    IDs        map[string]string `json:"ids"` // provider → externalID
}
```

## Pipeline completo

### Fase 0: App Start — Restaurar desde DB

```
Al cargar la lista de partidos:
  Para cada match, cargar EntityMapping desde DB
  → Poblar match.ExternalIDs con los IDs persistidos
  → Si todos los IDs existen en DB → no hacer HTTP
  → Si falta algún ID → resolver en batch
```

### Fase 1: Batch — Resolver torneos primero

```
Después de cargar Leagues() de FotMob:
  Para cada liga:
    → Si ya existe en DB con todos los IDs → skip
    → Si no:
        ESPN: scoreboard → match por nombre → eventID
        SofaScore: /unique-tournament/{id}/scheduled-events/{date} → tournamentID
    → Guardar en DB + en memoria (Competition.ExternalIDs)
```

### Fase 2: Batch — Resolver matches

```
Después de cargar LeagueMatches() de FotMob:
  Para cada match:
    → Si ya existe en DB con todos los IDs → poblar ExternalIDs desde DB
    → Si no:
        1. Obtener tournamentID desde la liga (ya resuelto)
        2. ESPN: scoreboard por slug+date → FuzzyTeamMatch → eventID
        3. SofaScore: /unique-tournament/{id}/scheduled-events/{date} → FuzzyTeamMatch → eventID
    → Guardar en DB + en memoria (match.ExternalIDs)
```

### Fase 3: On-demand (usuario abre detalle)

```
MatchDetails(fotmobID):
  1. FotMob.MatchDetails() → eventos base
  2. applyEnrichment(ESPN):
     → match.ExternalIDs.Get("espn") → usar directo (sin HTTP extra)
     → ESPN.EnrichMatch(eventID, ...)
  3. applyEnrichment(SofaScore):
     → match.ExternalIDs.Get("sofascore") → usar directo (sin HTTP extra)
     → SofaScore.EnrichMatch(eventID, ...)
  4. DedupEvents()
```

## Interfaz MatchResolver

```go
// internal/domain/resolver.go
type MatchResolver interface {
    ResolveMatches(matches []Match, leagues []Competition, date time.Time) error
}
```

Implementada por:
- `espn.Provider` (usa scoreboard existente + FuzzyTeamMatch)
- `sofascore.Provider` (usa nuevo endpoint scheduled-events)

## EntityMappingDB (nueva capa de persistencia)

```go
// internal/persistence/sqlite/mapping.go — NUEVOS métodos

// GetEntityMapping carga el mapping completo de una entidad desde DB
func (m *MappingDB) GetEntityMapping(entityType, tifoID string) (*domain.EntityMapping, error)

// SetEntityMapping guarda/actualiza el mapping completo de una entidad
func (m *MappingDB) SetEntityMapping(em *domain.EntityMapping) error

// GetAllEntityMappings carga todos los mappings de un tipo
func (m *MappingDB) GetAllEntityMappings(entityType string) ([]domain.EntityMapping, error)
```

## Provider updates

### ESPN Provider

```go
func (p *Provider) ResolveMatches(matches []Match, leagues []Competition, date time.Time) error {
    // 1. Agrupar matches por league (para no repetir scoreboards)
    // 2. Para cada league, mapear a ESPN slug (FotmobLeagueToESPN)
    // 3. Fetch scoreboard para slug + date
    // 4. Match por FuzzyTeamMatch → eventID
    // 5. match.ExternalIDs.Set("espn", eventID)
    // 6. DB.SetEntityMapping (vía callback o referencia)
}
```

### SofaScore Provider

```go
type Provider struct {
    client     *Client
    lookup     *LookupService
    db         *MappingDB  // NUEVO: referencia a DB para guardar mappings
    tournamentCache map[int]int // fotmobLeagueID → sofascoreTournamentID
}

func (p *Provider) ResolveMatches(matches []Match, leagues []Competition, date time.Time) error {
    // 1. Para cada league, mapear a SofaScore tournamentID
    //    (desde mapping estático o desde API de torneos)
    // 2. Fetch /unique-tournament/{id}/scheduled-events/{date}
    // 3. Match por FuzzyTeamMatch → eventID
    // 4. match.ExternalIDs.Set("sofascore", eventID)
    // 5. Persistir en DB
}
```

Nuevo endpoint en cliente:
```go
func (c *Client) GetScheduledEvents(tournamentID int, date string) (*ScheduledEventsResponse, error) {
    return c.get(fmt.Sprintf("/unique-tournament/%d/scheduled-events/%s", tournamentID, date))
}
```

## Modelo EntityMapping → Match/Competition sync

Cuando se cargan mappings desde DB:

```go
// RestoreFromDB carga mappings y los aplica a los modelos en memoria
func RestoreFromDB(db *MappingDB, matches []Match, leagues []Competition) error {
    // 1. Cargar todos los tournament mappings
    for _, em := range leagues {
        mapping, _ := db.GetEntityMapping("tournament", em.TIFOID)
        if mapping != nil {
            for k, v := range mapping.IDs {
                em.ExternalIDs = em.ExternalIDs.Set(k, v)
            }
        }
    }
    // 2. Cargar todos los match mappings
    for _, m := range matches {
        mapping, _ := db.GetEntityMapping("match", m.TIFOID)
        if mapping != nil {
            for k, v := range mapping.IDs {
                m.ExternalIDs = m.ExternalIDs.Set(k, v)
            }
        }
    }
}
```

## Escalabilidad

Agregar un nuevo provider (ej: `statsperform`):

1. Implementar `EnrichMatch` (on-demand)
2. Implementar `MatchResolver.ResolveMatches` (batch)
3. En el batch, resolver IDs y guardarlos en el `EntityMapping`
4. El on-demand usa `ExternalIDs.Get("statsperform")` → ya resuelto

Sin cambios en la DB, sin cambios en el pipeline batch, sin cambios en el servicio de match.

## Archivos

| Archivo | Cambio |
|---------|--------|
| `internal/domain/mapping.go` | **NUEVO**: `EntityMapping` struct |
| `internal/domain/resolver.go` | **NUEVO**: `MatchResolver` interface |
| `internal/persistence/sqlite/mapping.go` | **MODIFICAR**: migración nueva tabla, métodos GetEntityMapping/SetEntityMapping |
| `internal/providers/sofascore/types.go` | **MODIFICAR**: agregar `ScheduledEventsResponse`, `EventSummary` |
| `internal/providers/sofascore/mapping.go` | **NUEVO**: `FotmobLeagueToSofaScore()` |
| `internal/providers/sofascore/client.go` | **MODIFICAR**: agregar `GetScheduledEvents()` |
| `internal/providers/sofascore/provider.go` | **MODIFICAR**: agregar cache, `ResolveMatches()`, refactor `EnrichMatch()` |
| `internal/providers/espn/provider.go` | **MODIFICAR**: agregar `ResolveMatches()`, guardar en ExternalIDs |
| `internal/services/match.go` | **MODIFICAR**: agregar `InitResolvers()`, `ResolveMatches()`, `RestoreFromDB()` |
| `espn/mapping.go` | **SIN CAMBIOS** (FuzzyTeamMatch ya existe) |
| `tui/model.go` | **MODIFICAR**: trigger batch resolution + restore desde DB |

## Orden de implementación

1. `domain/mapping.go` + `domain/resolver.go` — interfaces y tipos
2. `sqlite/mapping.go` — nueva tabla + métodos CRUD
3. `sofascore/types.go` + `client.go` — endpoint scheduled-events
4. `sofascore/mapping.go` — league-to-tournament mapping
5. `sofascore/provider.go` — cache, ResolveMatches, refactor EnrichMatch
6. `espn/provider.go` — ResolveMatches, guardar ExternalIDs
7. `services/match.go` — InitResolvers, ResolveMatches, RestoreFromDB
8. `tui/model.go` — integrar batch resolution

# Plan: Sistema unificado de eventos (3 providers)

## Objetivo

Cada **evento real del partido** → **1 solo `MatchEvent`** enriquecido con datos de los 3 providers.

## 1. Modelo de dominio (`internal/domain/models.go`)

Agregar campos al `MatchEvent`:

```go
type MatchEvent struct {
    // existentes...
    Minute, AddedTime, EventType, Team, Player *PlayerRef
    Assist *PlayerRef, HomeScore, AwayScore int
    CardType string, Detail string
    SubOut, SubIn *PlayerRef
    GoalDesc string, OwnGoal bool, ShotDesc string, HalfStr string
    Period int, SortTime int, SortOverload int

    // NUEVOS — enrichment cross-provider

    // ESPN
    GoalType  string // "header", "penalty", "regular"
    PauseType string // "hydration", "injury", "var", ""
    DelayText string // texto original del delay

    // SofaScore
    VarClass     string // "cardUpgrade", "penaltyNotAwarded", etc.
    VarConfirmed *bool  // nil | true | false

    // FotMob
    XG         *float64
    AssistName string
}
```

## 2. Servicio de matching (`internal/domain/merge.go` — nuevo)

```go
// MatchResult indica qué hacer
type MatchResult int
const (
    MatchIdentical   // mismo evento → mergear campos
    MatchNew         // no existe → crear
    MatchDuplicate   // duplicado intra-provider → ignorar
)

// MergeEvents fusiona campos de candidate en base, campo por campo.
// No sobreescribe campos ya poblados a menos que candidate tenga mejor data.
func MergeEvents(base, candidate *MatchEvent)

// ClassifyEvent determina el tipo de evento real sin importar el provider.
// Ej: ESPN type="goal---header" + FotMob type="Goal" → ambos son "Goal" con GoalType="header"
func ClassifyEvent(ev MatchEvent) EventType

// EventIdentity define qué hace único a un evento real para matching.
type EventIdentity struct {
    Minute    int
    EventType EventType  // dominio unificado
    Team      TeamSide
    Player    string     // nombre del jugador implicado
    CardType  string     // para tarjetas
    SubOut    string     // para sustituciones
}
func Identity(ev MatchEvent) EventIdentity
```

### Reglas de matching

| Tipo | Identity key |
|------|-------------|
| **Goal** | `(minute, TeamSide, Player.Name)` |
| **Card** | `(minute, TeamSide, Player.Name, CardType)` |
| **Substitution** | `(minute, TeamSide, SubOut.Name)` |
| **VAR** | `(minute, "")` — match por minuto exacto entre ESPN y SofaScore |
| **HT/FT/Half** | `(minute, EventType)` |
| **Pausa/Injury** | `(minute, EventType)` — si 2 eventos con mismo minute+EvPausa → `MatchDuplicate` |
| **Shot** | `(minute, TeamSide, Player.Name, EventType)` |
| **KO/S2** | `(minute, EventType)` |
| **AddedTime** | `(minute, EventType)` — match FotMob AddedTime ↔ SofaScore injuryTime |

## 3. Refactor ESPN (`internal/providers/espn/provider.go`)

### `mapExtraEvents` cambia de:

**Antes**: Crea eventos nuevos y los agrega a la lista. No matchea contra existentes.

**Después**: 
1. Itera `keyEvents` de ESPN
2. Para cada evento, intenta `match` contra la lista de eventos existente (FotMob)
3. Si hay match → `MergeEvents()` para enriquecer (GoalType en Goals, Detail en Cards, etc.)
4. Si NO hay match → crea evento nuevo solo si es de tipo que solo ESPN conoce:
   - `kickoff` → `EvKO`
   - `start-2nd-half` → `EvS2`
   - `start-delay`+drink → `EvPausa` con `PauseType="hydration"`
   - `start-delay`+injury → `EvInjury` con `PauseType="injury"`
   - `start-delay` vacío → `EvVideoReview` con `PauseType="var"`
   - `end-delay` → `EvContinua`
   - `halftime` → `EvHT` (solo si FotMob no puso Half+HT)
   - `end-regular-time` → `EvFT` (solo si FotMob no puso Half+FT)
5. **Dedup intra-ESPN**: mismo minuto + mismo tipo clasificado → 1 solo evento (colapsa pares por equipo)

### Matching específico por tipo ESPN → evento existente:

```python
# goal type en ESPN
"goal" → Match con FotMob Goal, mergear GoalType="regular"
"goal---header" → Match con FotMob Goal, mergear GoalType="header"
"penalty---scored" → Match con FotMob Goal, mergear GoalType="penalty"

# card
"yellow-card" → Match con FotMob Card (Yellow), mergear Detail
"red-card" → Match con FotMob Card (Red), mergear Detail

# substitution
"substitution" → Match con FotMob Substitution por minuto+equipo+subOut, mergear Detail

# half
"halftime" → Match con FotMob Half+HT si existe
"end-regular-time" → Match con FotMob Half+FT si existe
```

## 4. Refactor SofaScore (`internal/providers/sofascore/provider.go`)

### `EnrichMatch` cambia de:

**Antes**: Reemplaza eventos ESPN por minuto exacto. Solo maneja `varDecision`.

**Después**:
1. Itera `incidents` de SofaScore
2. Para cada incidente, intenta `match` contra la lista de eventos existente
3. Si hay match → `MergeEvents()` para enriquecer:
   - `varDecision` → mergea `VarClass`, `VarConfirmed`, `Player` en evento VAR existente
   - `substitution` → mergea `incidentClass` (`"regular"`/`"injury"`) en sub existente
   - `goal` → mergea `incidentClass` (redundante con GoalType de ESPN)
4. Si NO hay match → crea evento nuevo:
   - `varDecision` → `EvVAR` con `VarClass` y `VarConfirmed` (cuando ESPN no detectó el VAR)
   - Otros tipos (card, goal, sub) → se ignoran porque FotMob ya los cubre mejor

### `mapIncident` se expande para manejar más tipos:

```go
switch inc.IncidentType {
case "varDecision":
    // ya existe, agregar VarConfirmed
case "substitution":
    // nuevo: extraer incidentClass ("regular"/"injury") y mergear en sub existente
    // devolver MatchEvent con EventType=EvSubstitution
case "goal":
    // nuevo: confirmar que existe el goal, incidentClass para tipo
case "card":
    // nuevo: confirmar que existe la card
default:
    return domain.MatchEvent{}, false
}
```

## 5. Post-proceso global de dedup (`internal/services/match.go` o nuevo paso)

Después de que todos los enrichers corren, antes de devolver:

```
fn dedupEvents(events []MatchEvent) []MatchEvent:
    1. Ordenar por (period, sortTime, sortOverload, eventTypePriority)
    2. Recorrer y colapsar duplicados intra-minuto:
       - Mismo minute + mismo EventType → mergear, mantener 1
       - Especial: EvPausa + EvPausa → 1
       - Especial: EvVAR + EvVideoReview → mergear en 1 (priorizar VarClass de SofaScore)
       - Especial: EvContinua + EvContinua → 1
    3. Verificar pares start/end:
       - Si un start-delay no tiene end-delay después de X minutos → eliminar
       - Si un end-delay no tiene start-delay antes → eliminar
    4. Reordenar final
```

Añadir llamado en `MatchDetails` de `internal/services/match.go`:

```go
for _, ep := range s.enrichers {
    // ... enrichment existente ...
}

// NUEVO: post-proceso global
details.Events = dedupEvents(details.Events)
```

## 6. UI (`tui/components/match_detail_events.go`)

Actualizar `eventDesc()` y `eventTypeCell()` para usar los nuevos campos de enrichment:

```go
// En vez de solo Detail, usar campos específicos
case domain.EvVAR:
    if ev.VarClass != "" {
        return "VAR: " + varClassLabel(ev.VarClass)
    }
    return "VAR"
case domain.EvGoal:
    if ev.GoalType != "" {
        return fmt.Sprintf("Gol [%s]", ev.GoalType)
    }
    // ... resto igual
case domain.EvPausa:
    if ev.PauseType == "hydration" {
        return "Pausa de hidratación"
    }
    // ...
```

Función helper:

```go
func varClassLabel(class string) string {
    switch class {
    case "cardUpgrade":       return "Tarjeta subida a roja"
    case "penaltyNotAwarded": return "Penal no concedido"
    case "penaltyAwarded":    return "Penal concedido"
    case "goal":              return "Gol confirmado"
    case "goalCancelled":     return "Gol anulado"
    case "offside":           return "Fuera de juego"
    default:                  return class
    }
}
```

## 7. Orden de implementación

| Paso | Archivo | Cambio |
|------|---------|--------|
| 1 | `internal/domain/models.go` | Agregar campos nuevos a `MatchEvent` |
| 2 | `internal/domain/merge.go` | Crear `Identity()`, `MergeEvents()`, `ClassifyEvent()`, reglas de matching |
| 3 | `internal/providers/espn/provider.go` | Refactor `mapExtraEvents`: matchear existentes + enriquecer + crear huérfanos + dedup pares ESPN |
| 4 | `internal/providers/sofascore/provider.go` | Refactor `EnrichMatch` + `mapIncident`: matchear existentes + enriquecer + crear huérfanos. Agregar manejo de `substitution`, `goal`, `card` |
| 5 | `internal/services/match.go` | Agregar `dedupEvents()` como post-paso |
| 6 | `tui/components/match_detail_events.go` | Actualizar UI para usar campos nuevos |
| 7 | `go build ./...` + `go vet ./...` | Verificar compilación |

## 8. Casos borde cubiertos

| Escenario | Comportamiento |
|-----------|---------------|
| ESPN envía 2 `start-delay` same minute (equipo A + B) | Dedup → 1 evento |
| ESPN `start-delay` vacío + SofaScore `varDecision` same minute | Merge → 1 evento VAR con VarClass de SofaScore |
| FotMob Goal + ESPN goal same minute+player | Merge GoalType en el evento Goal existente |
| FotMob Goal + SofaScore goal same minute | Merge incidentClass en el evento Goal existente |
| FotMob Sub + SofaScore sub same minute+subOut | Merge incidentClass ("injury"/"regular") |
| ESPN `halftime` + FotMob Half+HT same minute | Merge → 1 evento HT |
| Solo ESPN tiene `kickoff`/`start-2nd-half` | Se crean como eventos nuevos |
| SofaScore substitution+injury sin FotMob Sub match | Se crea evento Sub con incidentClass="injury" |
| VAR sin `end-delay` (SofaScore no lo manda) | El CONT de ESPN se mantiene |

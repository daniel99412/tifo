# Plan: Enriquecimiento unificado de eventos (3 providers)

## Filosofía

Cada **evento real del partido** (un gol, una tarjeta, una pausa, un VAR) debe ser **un solo `MatchEvent`** en el dominio. Cada provider aporta los campos que conoce:

- **FotMob**: estructura base (jugadores, scores, asistencias, shotmap, xG, tipo de tarjeta)
- **ESPN**: periodos (KO, S2, HT, FT), pausas (hidratación, lesión, VAR), descripciones textuales
- **SofaScore**: clasificación VAR (`cardUpgrade`, `penaltyNotAwarded`, etc.), `confirmed`, detalle de jugador en VAR, tipo de sustitución (`regular`/`injury`)

## Pipeline propuesto

```
FotMob ──► eventos base (Goal, Card, Sub, Half, AddedTime, Shot)
              │
              ▼
ESPN ────► matching + enrichment + creación de eventos faltantes
              │
              ▼
SofaScore ► matching + enrichment (VAR, sub tipo)
              │
              ▼
         Lista final de MatchEvent únicos y enriquecidos
```

Cada etapa:
1. Toma la lista de eventos actual
2. Para cada evento del provider, busca un match en la lista existente
3. Si hay match → **enriquece** el evento existente (no lo reemplaza)
4. Si no hay match → **crea** el evento nuevo

## Matching: cómo identificar el mismo evento real

La función `matchEvent(base, candidate)` recibe el evento base (de la lista) y el candidato (del provider actual). Retorna `true` si representan el mismo evento real.

### Reglas por tipo:

| Tipo | Match por |
|------|-----------|
| **Goal** | mismo minuto (±0), mismo equipo, mismo jugador |
| **Card** | mismo minuto (±0), mismo equipo, mismo jugador, mismo color |
| **Sub** | mismo minuto (±0), mismo equipo, mismo `subOut` |
| **Half/FT** | mismo minuto, mismo tipo |
| **VAR** | `start-delay` vacío (ESPN) ≈ `varDecision` (SofaScore) en mismo minuto (±0) |
| **Pausa hidratación** | `start-delay`+drink (ESPN) → único provider, se crea |
| **Lesión** | `start-delay`+injury (ESPN) → único provider, se crea |
| **KO / S2** | ESPN → único provider, se crea |
| **Shot** | FotMob → único provider, se crea |
| **AddedTime** | FotMob → se crea, SofaScore `injuryTime` enriquece con `length` |

## Modelo de dominio unificado

Se agregan campos a `MatchEvent` en `internal/domain/models.go`:

```go
type MatchEvent struct {
    // Existing fields...
    
    // Nuevos campos para enrichment
    
    // De ESPN
    PauseType     string   // "hydration", "injury", "var", "" 
    GoalType      string   // "header", "penalty", "regular"
    DelayText     string   // texto original del start-delay/end-delay
    
    // De SofaScore  
    VarClass      string   // "cardUpgrade", "penaltyNotAwarded", etc.
    VarConfirmed  *bool    // nil si no hay dato, true/false si SofaScore lo tiene
    
    // De FotMob
    XG            *float64
    AssistName    string
}
```

## Detalle de enrichment por tipo de evento

### Goal
```
Base (FotMob):  minute, team, player, homeScore, awayScore, assistStr, xG
ESPN:           match por minuto+equipo+player → agrega GoalType ("header"/"penalty"/"regular")
SofaScore:      match por minuto+equipo+player → agrega incidentClass ("regular"/"penalty")
```

### Card
```
Base (FotMob):  minute, team, player, cardType (Yellow/Red)
ESPN:           match por minuto+equipo+player → agrega texto descriptivo
SofaScore:      match por minuto+equipo+player → (redundante, misma info)
```

### Substitution
```
Base (FotMob):  minute, team, subOut, subIn
ESPN:           match por minuto+equipo+subOut → agrega texto
SofaScore:      match por minuto+equipo+subOut → agrega incidentClass ("regular"/"injury")
```

### VAR
```
ESPN crea:      minute, PauseType="var" (desde start-delay vacío)
SofaScore:      match por minuto (±0) → agrega VarClass, VarConfirmed, player
Además:
  - ESPN start-delay vacío + end-delay posterior → el Cont (Continúa) se enlaza al VAR
  - SofaScore no tiene end-delay, así que el CONT se mantiene de ESPN
```

### Pausa hidratación / Lesión (solo ESPN)
```
ESPN crea:      minute, PauseType="hydration"/"injury", DelayText, 
                + EvContinua asociado con minute del end-delay
```

### KO / S2 / HT / FT
```
KO, S2:         solo ESPN → se crean
HT:             ESPN (halftime) + FotMob (Half+HT) → match por minuto+tipo, se enriquece
FT:             ESPN (end-regular-time) + FotMob (Half+FT) → match por minuto+tipo
                + SofaScore (period+FT) exact match
```

### AddedTime
```
FotMob crea:    minute, addedTime (minutesAddedInput)
SofaScore:      match por minuto → enriquece con length (injuryTime)
```

## Orden de implementación

1. **Agregar campos nuevos a `MatchEvent`** en `internal/domain/models.go`
2. **Crear función `mergeEvents(base, candidate)`** que aplica las reglas de matching y mergea campos
3. **Refactor `espn/provider.go` `mapExtraEvents`**: en lugar de solo agregar, también matchear contra eventos existentes y enriquecer (goalType en Goals, etc.)
4. **Refactor `sofascore/provider.go` `EnrichMatch`**: matchear contra eventos existentes y enriquecer (varClass en VAR, incidentClass en subs), en lugar de reemplazar
5. **Mantener creación de eventos únicos**: KO, S2, pausas (ESPN), shots (FotMob) se crean si no existen

## Archivos a modificar

| Archivo | Cambio |
|---------|--------|
| `internal/domain/models.go` | Agregar campos PauseType, GoalType, DelayText, VarClass, VarConfirmed, XG, AssistName |
| `internal/providers/espn/provider.go` | `mapExtraEvents`: matchear y enriquecer goals/cards/subs, crear pausas/periodos |
| `internal/providers/sofascore/provider.go` | `EnrichMatch`: matchear y enriquecer VAR/subs, no reemplazar |
| `tui/components/match_detail_events.go` | Actualizar `eventDesc()` para usar los nuevos campos |
| `tui/model.go` | Si hay referencias a campos viejos, actualizar |

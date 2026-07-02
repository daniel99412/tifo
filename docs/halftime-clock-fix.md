# Halftime Clock Resume Fix

## Análisis de payloads

### ESPN (`keyEvents[]`)
- **Tiene "Start 2nd Half"**: `type.id="82"`, `type.type="start-2nd-half"`, `period.number=2`
- `classify()` en `espn/provider.go:573` lo mapea a `EvS2` correctamente
- `parseClock("45'")` → `minute=45, added=0`
- Se agrega al slice de eventos con `SortTime=45, SortOverload=0`

### FotMob (`content.matchFacts.events.events[]`)
- **NO tiene S2**. La lista va directo de `Half("HT")` en min 45 a `Substitution` en min 46
- HT se mapea con `SortTime=45, SortOverload=maxAddedHT+1` (ej. 4 si hubo 3 min añadidos)
- `header.status.liveTime.short` cambia de `"HT"` → `"46'"` al arrancar 2do tiempo
- `parseLiveMinute("HT")` → 0, `parseLiveMinute("46'")` → 46

---

## Bugs encontrados

### Bug #1: S2 se ordena ANTES de HT (desplazamiento)

ESPN pone S2 con `SortOverload=0`. FotMob pone HT con `SortOverload = maxAddedHT + 1` (4).
Al ordenar por `(SortTime, SortOverload)`:

```
[0] Evento 45+0          SortOverload=0
[1] AddedTime 45+3       SortOverload=3
[2] EvS2 (ESPN)          SortOverload=0   ← ¡ANTES de HT!
[3] EvHalf+HT (FotMob)   SortOverload=4
[4] Evento min 46        SortOverload=0
```

`isHalfTime` en `tui/model.go:1367` compara **índices**: `lastS2=2`, `lastHT=3`.
`2 > 3` → **false** → `isHalfTime` retorna true incluso con S2 presente.

**Causa**: ESPN no conoce el tiempo añadido del primer tiempo. El S2 se inserta
con `SortOverload=0` y los eventos 45+X lo desplazan antes de HT.

### Bug #2: Sin ESPN, FotMob nunca envía S2

Sin el evento S2, `lastS2` siempre es -1. `isHalfTime` nunca retorna false
(solo lo hace con `lastEventMinute >= 60`, que es demasiado tarde).

---

## Plan de fix

### Paso 1: Corregir SortOverload de S2 en ESPN enricher

**Archivo**: `internal/providers/espn/provider.go` — `mapExtraEvents()` (~L454)

Al mapear `EvS2`, buscar el `SortOverload` máximo entre los eventos existentes
en minuto 45, y poner S2 con `+1` para que quede después de HT:

```go
minute, added := parseClock(ke.Clock.DisplayValue)

// EvS2 debe ordenarse DESPUÉS del HT, no antes
if typ == domain.EvS2 {
    maxOverload := 0
    for _, ev := range existing {
        if ev.SortTime == 45 && ev.SortOverload > maxOverload {
            maxOverload = ev.SortOverload
        }
    }
    added = maxOverload + 1
}
```

Resultado: `HT(45, 4) < S2(45, 5) < Eventos(46, 0)` → `lastS2 > lastHT` = true.

### Paso 2: Agregar campo `Period` a `MatchEvent`

**Archivo**: `internal/domain/models.go` — struct `MatchEvent`

```go
type MatchEvent struct {
    // ... campos existentes ...
    Period       int       // MatchPeriod: 1=FirstHalf, 2=SecondHalf, 3=ET1, 4=ET2
}
```

#### Mapeo desde ESPN (`espn/provider.go`)
Usar `ke.Period.Number`:
- `1` → `domain.PeriodFirstHalf`
- `2` → `domain.PeriodSecondHalf`

#### Mapeo desde FotMob (`fotmob/provider.go`)
Inferir del minuto del evento:
- `time <= 45` → `PeriodFirstHalf`
- `46 <= time <= 90` → `PeriodSecondHalf`
- `91 <= time <= 105` → `PeriodETFirstHalf`
- `106 <= time <= 120` → `PeriodETSecondHalf`

#### Usar en `isHalfTime` (`tui/model.go`)
Reemplazar el chequeo `lastEventMinute < 60` por:
```go
// Si hay eventos en PeriodSecondHalf después de HT, ya pasó el medio tiempo
for i := lastHT + 1; i < len(sorted); i++ {
    if sorted[i].Period >= domain.PeriodSecondHalf {
        return false
    }
}
```

### Paso 3: Mantener guard existente como safety net

**Archivo**: `tui/model.go` ~L632

El check `m.detailMinute <= 45` se queda como respaldo para casos donde
ni Period ni S2 están disponibles (FotMob sin ESPN, partido recién arrancando
segundo tiempo sin eventos todavía):

```go
if m.matchDetails != nil && isHalfTime(m.matchDetails.Events) && m.detailMinute <= 45 {
    m.detailView.Minute = "HT"
}
```

---

## Fixes ya aplicados (no se tocan)

| Ubicación | Cambio | Propósito |
|-----------|--------|-----------|
| `tui/model.go:632` | `isHalfTime(...) && m.detailMinute <= 45` | No sobreescribir a HT si API reporta minuto >45 |
| `tui/model.go:496` | `LiveMinute > 45` antes que eventos HT | Listado de partidos muestra minuto real post-HT |

---

## Archivos a modificar

| Archivo | Qué se cambia |
|---------|---------------|
| `internal/domain/models.go` | Agregar `Period int` a `MatchEvent` |
| `internal/providers/espn/provider.go` | Ajustar `SortOverload` de EvS2; setear `Period` desde `ke.Period.Number` |
| `internal/providers/fotmob/provider.go` | Setear `Period` inferido del minuto del evento |
| `tui/model.go` | Actualizar `isHalfTime` para usar `Period` en vez de `lastEventMinute < 60` |

# Plan: Refinamiento del sistema de matching y enrichment

## Problemas detectados en la simulación

1. `start-delay` vacío en minuto 0 → clasificado como VAR (debiera ser pausa genérica)
2. Lesiones sin jugador extraído
3. HT/FT duplicados por diferencia de minutos entre FotMob y ESPN
4. Subs duplicadas por diferencia de minutos y matching incorrecto
5. Matching muy estricto (minuto exacto) para HT/FT/subs

## 1. Clasificación de `start-delay` vacío

**Archivo**: `internal/providers/espn/provider.go` — función `classify()`

**Cambio**: si `start-delay` tiene texto vacío Y minuto es 0, clasificar como `EvPausa` con `pauseType="other"`, no como `EvVideoReview`.

```go
case "start-delay":
    text := strings.ToLower(ke.Text)
    switch {
    case containsStr(text, "injury"):
        return domain.EvInjury
    case containsStr(text, "drink") || containsStr(text, "hydration") || containsStr(text, "agua"):
        return domain.EvPausa
    case text == "" || containsStr(text, "var") || containsStr(text, "video"):
        // Si está en minuto 0 y no hay texto, es delay pre-partido (no VAR)
        minute, _ := parseClock(ke.Clock.DisplayValue)
        if text == "" && minute <= 0 {
            return domain.EvPausa
        }
        return domain.EvVideoReview
    default:
        return domain.EvPausa
    }
```

También actualizar `classifyPauseType` para que devuelva `"other"` en lugar de `"var"` cuando sea un delay genérico.

## 2. Extraer jugador de lesiones

**Archivo**: `internal/providers/espn/provider.go` — función `classifyPauseType` o nueva función `extractPlayerFromDelay`

Agregar extracción de nombre de jugador desde el texto del delay:
```go
"Delay in match because of an injury Luis Romo (Mexico)." 
→ extractPlayerFromText(ke.Text) → "Luis Romo"
```

Se puede reusar `extractPlayerFromText` que ya existe (usa el patrón `Nombre (Equipo)`).

En `mapExtraEvents`, cuando el tipo es `EvInjury`, asignar `ev.Player = extractPlayerFromText(ke.Text)`.

## 3. Match de HT/FT con tolerancia

**Archivo**: `internal/domain/merge.go` — función `IdentityMatch()`

Agregar tolerancia de ±3 minutos para HT y FT:

```go
case EvHT, EvFT:
    diff := ai.Minute - bi.Minute
    if diff < 0 { diff = -diff }
    return diff <= 3 // tolerancia de 3 minutos
```

**Regla de merge**: cuando hay match entre FotMob HT/FT y ESPN HT/FT:
- Tomar el minute + addedTime de ESPN (tiene el tiempo real con added time)
- Tomar detail de ESPN (texto descriptivo como "Second Half ends...")
- El addedTime de ESPN es el que prevalece

## 4. Match de subs por nombres

**Archivo**: `internal/domain/merge.go` — función `IdentityMatch()`

```go
case EvSubstitution:
    // Match primario: subOut debe coincidir
    if ai.SubOut != "" && ai.SubOut == bi.SubOut {
        return true
    }
    // Match secundario: subIn + equipo deben coincidir
    if ai.SubOut == "" && bi.SubOut == "" {
        return ai.Player != "" && ai.Player == bi.Player
    }
    // Si no hay subOut en ninguno, match por subIn
    return false
```

Además, en `Identity()` para subs, almacenar también `subIn` para poder hacer matching secundario:

```go
type EventIdentity struct {
    // ... campos existentes
    SubIn    string // para sustituciones: player entering
}
```

Y permitir tolerancia de ±1 minuto para subs:
```go
diff := ai.Minute - bi.Minute
if diff < 0 { diff = -diff }
if diff > 1 { return false }
```

## 5. IdentityMatch refinado (resumen)

| Tipo | Match por | Tolerancia |
|------|-----------|------------|
| **Goal** | minuto + equipo + jugador | 0 min |
| **Card** | minuto + equipo + jugador + cardType | 0 min |
| **Substitution** | minuto + subOut (o subIn+team) | ±1 min |
| **VAR** | minuto exacto | 0 min |
| **HT** | tipo normalizado | ±3 min |
| **FT** | tipo normalizado | ±3 min |
| **Pausa/Injury** | minuto + tipo | 0 min |
| **Continua** | minuto + tipo | 0 min |
| **AddedTime** | minuto + tipo | 0 min |
| **Shot** | minuto + equipo + jugador | 0 min |

## 6. Normalización de tipos para dedup

**Archivo**: `internal/domain/merge.go` — función `classifyDomain()` y `DedupEvents()`

Agregar normalización de `Half`→`HT`/`FT`:

```go
case EvHalf:
    if strings.HasPrefix(ev.HalfStr, "HT") || ev.HalfStr == "HalfTime" {
        return EvHT
    }
    if strings.HasPrefix(ev.HalfStr, "FT") || ev.HalfStr == "FullTime" {
        return EvFT
    }
    return EvHalf
```

Esto asegura que FotMob `Half+HT` y ESPN `halftime` se normalicen al mismo tipo `HT` y puedan deduplicarse.

## 7. Archivos a modificar

| Archivo | Cambio |
|---------|--------|
| `internal/domain/merge.go` | `IdentityMatch()` con tolerancias; `Identity()` agregar SubIn; `classifyDomain()` normalizar Half→HT/FT |
| `internal/providers/espn/provider.go` | `classify()`: empty start-delay en min 0 → Pausa; `mapExtraEvents`: extraer player de injury; `classifyPauseType`: caso "other" |
| `internal/providers/sofascore/provider.go` | Sin cambios (el matching ya es delegado a `domain.IdentityMatch`) |
| `internal/services/match.go` | Sin cambios (DedupEvents ya usa classifyDomain) |

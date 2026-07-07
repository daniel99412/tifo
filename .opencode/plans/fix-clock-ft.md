# Fix: Reloj no se detiene al llegar FT + se va a ET

## Diagnóstico

### Bug 1: `isMatchFinished` recibe nil events

`tui/model.go:1308` — `isMatchLive(m)` llama `isMatchFinished(m, nil)`, con `nil` events.
El chequeo de `EvFT` dentro de `isMatchFinished` es código muerto desde `isMatchLive`.

Callers afectados:
- `renderTickMsg` (line 629) — el clock tick cada 1s
- `dataTickMsg` (line 681) — auto-refresh check
- `selectMatch` (line 305) — inicialización de detail view
- `formatMatch` (line 1502) — lista de partidos
- `formatPenScore` (line 1556) — lista de partidos

### Bug 2: `computeMatchMinuteFromEvents` mezcla `minute` y `em`

`tui/model.go:1340` — el switch usa `minute` para los primeros 2 cases pero `em` para ET:

```go
case minute <= 45:     // usa minute
case minute <= 90:     // usa minute
case em <= 105:        // usa em ← BUG
case em <= 120:        // usa em ← BUG
```

Cuando `detailMinute` se actualiza desde `lastEventMinute()` con un valor 91-105
(ej: gol en 90+5 → SortTime=90, SortOverload=5 → m=95):

1. `minute <= 45` → false
2. `minute <= 90` → false
3. `case em <= 105` → `em = 95 + phaseSec/60` → true
4. Muestra `"ET X:XX"` aunque el partido esté en tiempo regular

## Fix A: Pasar events a `isMatchLive`

### 1. Cambiar firma (line 1307)

```go
// Antes:
func isMatchLive(m domain.Match) bool {

// Después:
func isMatchLive(m domain.Match, events []domain.MatchEvent) bool {
```

### 2. Actualizar caller en `renderTickMsg` (line 629)

```go
// Antes:
if isMatchLive(*m.selectedMatch) {

// Después:
var liveEvents []domain.MatchEvent
if m.matchDetails != nil {
    liveEvents = m.matchDetails.Events
}
if isMatchLive(*m.selectedMatch, liveEvents) {
```

### 3. Actualizar caller en `dataTickMsg` (line 681)

```go
// Antes:
if m.selectedMatch != nil && !m.loadingDetail && isMatchLive(*m.selectedMatch) {

// Después:
var tickEvents []domain.MatchEvent
if m.matchDetails != nil {
    tickEvents = m.matchDetails.Events
}
if m.selectedMatch != nil && !m.loadingDetail && isMatchLive(*m.selectedMatch, tickEvents) {
```

### 4. Actualizar caller en `selectMatch` (line 305)

```go
// Antes:
if isMatchLive(*selMatch) {

// Después:
if isMatchLive(*selMatch, nil) {
```

### 5. Actualizar caller en `formatMatch` (line 1502)

```go
// Antes:
live := isMatchLive(m)

// Después:
live := isMatchLive(m, nil)
```

### 6. Actualizar caller en `formatPenScore` (line 1556)

```go
// Antes:
if isMatchLive(m) {

// Después:
if isMatchLive(m, nil) {
```

## Fix B: Arreglar `computeMatchMinuteFromEvents`

Usar `minute` consistentemente en todos los cases del switch (line 1349-1378):

```go
func computeMatchMinuteFromEvents(ko, lastUpdate time.Time, minute int, added int) string {
    if minute <= 0 {
        return computeMatchMinute(ko, added)
    }

    phaseSec := int(time.Since(lastUpdate).Seconds())
    em := minute + phaseSec/60
    es := phaseSec % 60

    switch {
    case minute <= 45:
        if em <= 45 {
            if em == 45 && es > 0 {
                return fmt.Sprintf("45+%d", es)
            }
            return fmt.Sprintf("%d:%02d", em, es)
        }
        return fmt.Sprintf("45+%d", em-45)
    case minute <= 90:
        if em <= 90 {
            if em == 90 && es > 0 {
                return fmt.Sprintf("90+%d", es)
            }
            return fmt.Sprintf("%d:%02d", em, es)
        }
        return fmt.Sprintf("90+%d", em-90)
    case minute <= 105:
        if em <= 105 {
            if em == 105 && es > 0 {
                return fmt.Sprintf("105+%d", es)
            }
            return fmt.Sprintf("ET %d:%02d", em-90, es)
        }
        return fmt.Sprintf("105+%d", em-105)
    case minute <= 120:
        if em <= 120 {
            if em == 120 && es > 0 {
                return fmt.Sprintf("120+%d", es)
            }
            return fmt.Sprintf("ET %d:%02d", em-90, es)
        }
        return fmt.Sprintf("120+%d", em-120)
    default:
        return "FT"
    }
}
```

**Cambio clave**: los cases ET usan `minute <= 105` y `minute <= 120` en vez de `em <= 105` / `em <= 120`.

### Verificación

```bash
go build ./...
go vet ./...
```

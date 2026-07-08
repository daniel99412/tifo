# Plan: Corregir pauseType y ordenamiento temporal

## Problemas detectados

1. `pauseType` incorrecto en eventos generados por la simulación Python
2. Ordenamiento temporal entre providers puede intercalar eventos incorrectamente

## Fix 1: `classifyPauseType` debe manejar el caso "other"

**Archivo**: `internal/providers/espn/provider.go`

Actualmente `classifyPauseType` devuelve `"var"` por defecto para cualquier `start-delay` que no sea drink/injury. Pero desde que `classify()` ahora retorna `EvPausa` para empty `start-delay` en minuto 0, el `pauseType` debe reflejarlo.

Ya existe la lógica:
```go
if typ == domain.EvPausa && pauseType == "var" {
    pauseType = "other"
}
```

Esto es correcto en Go. La simulación Python no lo aplicó correctamente. No requiere cambio en Go.

## Fix 2: Separar eventos de juego de eventos de pausa por "línea de tiempo"

El problema de fondo: eventos de juego (shots, subs) y eventos de pausa (injury, VAR, hydration) viven en la misma lista ordenada por minuto. Un shot en minuto 3 y una lesión en minuto 2 se intercalan aunque en tiempo real el shot fue antes.

**Solución**: no mezclar shots y pause events en el mismo timeline. Mantenerlos como listas separadas o asignarles un "subtimeline" (0 = juego, 1 = pausa).

Pero esto es un cambio grande. Alternativa más simple:

**Solución alternativa**: no mezclar shots del shotmap con eventos del timeline. El shotmap es una fuente separada que no tiene correlación temporal exacta con los eventos del partido. Renderizar los shots en una sección aparte (como "Estadísticas" o "Tiros") en lugar de intercalarlos en la lista de eventos.

**Solución mínima**: dejar el sorting como está (por minuto) y aceptar que el shotmap tiene timestamps aproximados. El fix real es no entremezclar shotmap con timeline - solo mostrar shots en una vista de stats.

## Fix 3: Verificar que el código Go real produce pauseType correcto

El Go code ya tiene la lógica correcta:
- `classifyPauseType("injury")` → `"injury"`
- Empty text + minute 2 → `delay_type_at_minute` hereda `"injury"` del primer evento del par
- Empty text + minute 0 → `classify()` retorna `EvPausa`, luego `pauseType` se fuerza a `"other"`

No se requieren cambios en Go. La simulación Python tenía bugs de merge.

## Resumen de cambios necesarios

| Archivo | Cambio | Prioridad |
|---------|--------|-----------|
| Ninguno (Go correcto) | `classifyPauseType` + `pauseType override` ya están | — |
| `tui/` | Separar shotmap del timeline de eventos | Baja |

# Plan: Cachear eventID de SofaScore y guardarlo en ExternalIDs

## Problema

SofaScore no almacena su `eventID` resuelto, causando:
1. 3 HTTP calls por cada auto-refresh (cada 30s) para re-resolver
2. El lookup puede fallar silenciosamente, dejando VAR sin enriquecer
3. No hay visibility del eventID de SofaScore

## Cambios

### 1. Agregar `IDCache` a SofaScore Provider

**Archivo**: `internal/providers/sofascore/provider.go`

```go
type Provider struct {
    client *Client
    lookup *LookupService
    cache  map[int]int // fotmobID → sofascoreEventID
}

func NewProvider() *Provider {
    return &Provider{
        client: NewClient(),
        lookup: NewLookupService(NewClient()),
        cache:  make(map[int]int),
    }
}
```

En `EnrichMatch()`:
```go
func (p *Provider) EnrichMatch(matchID int, ...) *domain.MatchDetails {
    // Intentar caché primero
    eventID, ok := p.cache[matchID]
    if !ok {
        // Resolver y cachear
        eventID, err = p.lookup.Resolve(homeTeam, awayTeam, utcTime)
        if err != nil {
            log.Printf("[sofascore] lookup: %v", err)
            return fotmobDetails
        }
        p.cache[matchID] = eventID
    }
    ...
}
```

### 2. Guardar eventID en ExternalIDs del MatchDetails

En `EnrichMatch()`, después de obtener incidents exitosamente:
```go
out := *fotmobDetails
out.ExternalIDs = out.ExternalIDs.Set("sofascore", fmt.Sprintf("%d", eventID))
```

Esto persiste el eventID de SofaScore en el dominio, permitiendo:
- Debug: ver qué ID usó SofaScore
- Reutilización: el ID queda disponible para otros procesos
- SQLite: el `MatchResolver` ya guarda ExternalIDs en mappings

### 3. Aumentar límite de eventos de 20 a 50

**Archivo**: `internal/providers/sofascore/lookup.go`

```go
events, err := s.client.GetTeamLastEvents(homeID, 50)
```

### 4. Buscar por awayTeam si homeTeam falla

En `Resolve()`:
```go
func (s *LookupService) Resolve(homeTeam, awayTeam string, utcTime time.Time) (int, error) {
    // Intentar con homeTeam primero
    eventID, err := s.resolveForTeam(homeTeam, awayTeam, utcTime)
    if err == nil {
        return eventID, nil
    }
    // Fallback: buscar por awayTeam
    return s.resolveForTeam(awayTeam, homeTeam, utcTime)
}

func (s *LookupService) resolveForTeam(local, visitante string, utcTime time.Time) (int, error) {
    teamID, err := s.findTeamID(local)
    ...
    // Buscar visitante como oponente
    if strings.EqualFold(opponent, visitante) || strings.Contains(...) {
        return ev.ID, nil
    }
    ...
}
```

## Resumen de archivos a modificar

| Archivo | Cambio |
|---------|--------|
| `internal/providers/sofascore/provider.go` | Agregar `cache map[int]int`, cachear eventID, guardar en ExternalIDs |
| `internal/providers/sofascore/lookup.go` | Aumentar límite a 50, agregar fallback por awayTeam, refactor `Resolve` |

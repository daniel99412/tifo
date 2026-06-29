# Agent Instructions

Antes de leer el codigo del repo de forma amplia, revisa primero `graphify-out/`.

Usa esos archivos como mapa inicial del proyecto:

- `graphify-out/GRAPH_REPORT.md` para entender comunidades, modulos principales y puntos de entrada.
- `graphify-out/graph.json` solo para consultas puntuales sobre simbolos, funciones o dependencias.
- `graphify-out/manifest.json` para validar el contenido generado.

No abras todos los archivos del codigo fuente de golpe ni hagas busquedas enormes si `graphify-out` ya puede orientar la investigacion. Primero identifica los archivos y funciones relevantes desde el grafo, y despues lee solo esos fragmentos.

Para cambios en alineaciones, revisa primero los nodos relacionados con:

- `components_matchdetail_renderlineup`
- `components_playerlineup`
- `components_lineupdata`
- `fotmob_provider_maplineups`
- `domain_lineups`

Despues de ubicar el flujo con `graphify-out`, limita la lectura a los archivos concretos que correspondan.

Implement a new "Vertical Lineups" view for the TIFO terminal application.

Tech stack:

- Go
- Bubble Tea
- Lip Gloss
- No images, avatars, emojis or Unicode emoji.
- Terminal-first design.
- Must work correctly in terminals between 100-160 columns.

Goal:
Render both teams' starting XI in a vertical football formation instead of a horizontal pitch.

Requirements:

- The screen is divided into two vertical sections.
- Home team appears first.
- Away team appears below it.
- Each team has:
  - Team name
  - Formation (example: 4-3-3)
  - Coach (optional)
  - Starting XI rendered by tactical lines.

Example:

                 [9] Haaland

[11] Doku [26] Savinho

      [20] Bernardo   [17] De Bruyne
              [16] Rodri

[24] Gvardiol [3] Dias [25] Akanji [82] Lewis

               [31] Ederson

──────────────────────────────────────────────

          [9] Mbappé     [7] Vinicius

[5] Bellingham [14] Tchouameni
[8] Valverde [11] Rodrygo

[23] Mendy [22] Rüdiger [3] Militão [2] Carvajal

             [1] Courtois

Rendering rules:

- The goalkeeper is always alone.
- Each tactical line is centered.
- Horizontal spacing is calculated dynamically.
- Player names should never overlap.
- Long names should be truncated with an ellipsis.
- Jersey number is always shown.
- Every row should have the same height.
- No ASCII football pitch.
- No boxes around players.
- Keep the layout minimal and clean.

Data model:

Each player contains:

- Name
- ShirtNumber
- Position
- IsCaptain
- IsKeeper
- TacticalLine
- TacticalOrder

The formation is already parsed into tactical lines, so DO NOT calculate formations.
Simply group players by TacticalLine.

Implementation:

Create a reusable renderer:

RenderVerticalLineup(
width int,
lineup []Player,
formation string,
teamName string,
) string

The renderer should:

1. Group players by TacticalLine.
2. Sort each line by TacticalOrder.
3. Calculate spacing dynamically.
4. Center every tactical line.
5. Return a Lip Gloss rendered string.

Use Bubble Tea best practices:

- Pure rendering.
- No global state.
- No duplicated layout logic.
- Width-aware rendering.
- Gracefully handle terminal resize.

Avoid magic numbers whenever possible.

The output should look like a professional football terminal application similar to FotMob or Tifo, prioritizing readability over decorative ASCII art.

# Implementation Plan - Layered Rendering and Track Rules

Refactor the rendering pipeline to strictly follow the 3-layer conceptual structure (Under, Normal, Over). This creates the illusion of depth, allowing entities to move "under" or "over" stacks of tiles.

## Proposed Changes

### Domain Logic

#### [system.go](file:///C:/repositories/hunter-gatherers-concentration/internal/domain/system.go)

- Refactor `doMove` to clarify the `CreatureMoved` event payload:
    - `hidden` (bool): Set to `true` ONLY if the creature is truly cloaked (e.g., Shadowstalker). This flag tells the UI to skip the animation entirely.
    - `mode` (string): Set to `under`, `normal`, or `over`. For Burrowers (`ModeUnder`), `hidden` will be `false` to allow animation, but the `mode` will ensure it renders on the correct layer.

### Rendering Pipeline

#### [board_renderer.go](file:///C:/repositories/hunter-gatherers-concentration/internal/ui/renderer/board_renderer.go)

- Update `SubscribeToEvents`:
    - Only skip animations if `hidden` is `true`.
    - Map the `mode` payload to `LayerUnder`, `LayerNormal`, or `LayerOver` in `AnimManager.StartTileMove`.
- Refactor `Render` to strictly enforce layer ordering and coordinate origins:
    - **Surface: Playmat** (`ui.PlaymatX/Y`, Size 700x700)
        - `renderPlaymat`
    - **Surface: Board** (`gridOffsetX/Y`, Size 525x525)
        - `renderEmptyGrid` (Floor)
        - **Layer: Under**: `renderTracksUnder`, `renderMovementsUnder`.
        - **Layer: Normal**: `renderGrid`, `renderTracksBetween`, `renderMovementsNormal`.
        - **Layer: Over**: `renderTracksOver`, `renderMovementsOver`.
    - **Surface: Effects** (`ui.PlaymatX/Y`, Size 700x700)
        - `renderEffectsUnder`, `renderEffectsBetween`, `renderEffectsOver` (including Scanner). These will use the Playmat as their canvas to allow larger visual impacts.

#### [track_renderer.go](file:///C:/repositories/hunter-gatherers-concentration/internal/ui/renderer/track_renderer.go)

- Update `RenderUnder`: Draw `mud` and `broken_grass`.
- Update `RenderOver`: Draw `claws`.
- Update `Draw`:
    - `mud`: Render at the midpoint between `FromPos` and `ToPos`, rotated.
    - `broken_grass`: Render at `FromPos`.
    - `claws`: Render at `ToPos`.

## Verification Plan

### Automated Tests
- `go test ./internal/domain/...`

### Manual Verification
- **Visual Inspection**:
    - **Burrower**: Verify it animates *under* tiles.
    - **Tracks**:
        - `mud`: Under layer, between tiles.
        - `broken_grass`: Under layer, at start tile.
        - `claws`: Over layer, at end tile.
    - **Coordinates**: Verify that tracks/tiles stay within the 525x525 board area, while effects like the scanner cover the 700x700 playmat.

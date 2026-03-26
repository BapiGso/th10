# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
go build ./...          # compile all packages
go run ./cmd/main.go    # launch the game (640x480 window)
go vet ./...            # static analysis
```

No test suite yet. Verify changes by building (`go build ./...`).

## Project Overview

Go + Ebiten v2 recreation of 東方風神録 (Touhou 10: Mountain of Faith). Module name: `th10`. Entry point: `cmd/main.go`.

## Architecture

### Game Loop & Scenes

`game.Game` implements `ebiten.Game` (60 TPS). All screens implement `game.Scene` (Update + Draw). Scene switching via `Game.GoTo()` (deferred to next frame).

Flow: Loading → Title → Select(char/shot/difficulty) → Stage1 → Stage2 → ... → Stage6 → Result

### Stage System

`scene/stage/stage.go` is the generic stage runner. Each stage implements `stage.Script`:
- `Init(ctx)` — setup BGM
- `Update(ctx)` — spawn enemies, fire danmaku on frame timers
- `BgDraw(field)` — draw scrolling background
- `Finished()` — signal stage completion

`stage.Context` provides access to Player, BulletPool, ItemPool, EffectPool, Enemies, Emitter, Audio. Scripts call `ctx.Emitter.Ring/Aimed/Spiral/Random()` for bullet patterns, `ctx.AddEnemy()` to spawn, `ctx.StartDialog()` for conversations.

Stage scripts: `scene/stage/stage1/` through `stage6/` + `extra/`. Chained via `NextScene` callback set in `scene/select/select.go`.

### Entity System (Pre-allocated Pools)

All entity pools use `Active` bool flags to avoid GC pressure:
- `entity/bullet/` — Enemy bullet pool (2048 cap), 5 types (Rice/Small/Middle/Large/Laser)
- `entity/player/` — Player with 4 states (Normal/Dead/Respawn/Bomb), hitbox 2px, graze 16px
- `entity/playerbullet/` — Player bullet pool (256 cap)
- `entity/item/` — Item pool (256 cap), 6 types with gravity + magnetic attraction
- `entity/enemy/` — Enemy with path-following, BossData with SpellCard phase system
- `entity/effect/` — Effect pool (512 cap)

### Collision (`collision/`)

Three methods: CircleHit (standard), RectHit (loose/items), OrthoCircleHit (rect coarse → circle fine, matching original TH).

### Danmaku (`danmaku/emitter.go`)

Pattern primitives: Ring, Aimed (N-way spread at target), Spiral (multi-arm rotating), Random. Fires directly into shared bullet pool.

### Sprite System (`sprite/`)

`sprite.Global` singleton loaded once in Loading scene via `sprite.Load()`. Contains PlayerSprite, BulletSprite, EnemySprite, and Bosses map. Sheet loading uses `sprite.LoadSheet()` → `SubImage` extraction.

### Assets (`assets/`)

Embedded via `//go:embed anm font` + `wav/se_* wav/bgm_ogg/*`. All assets compile into the binary.
- `anm/` — PNG sprites (player, bullet, enemy, background, face, title, etc.)
- `font/` — DFYuGaSo TTC fonts
- `wav/se_*.wav` — Sound effects
- `wav/bgm_ogg/` — 18 OGG Vorbis BGM tracks with loop points

Asset loading pattern used everywhere:
```go
data, _ := assets.Assets.ReadFile("anm/path/file.png")
img, _, _ := image.Decode(bytes.NewReader(data))
ebitenImage := ebiten.NewImageFromImage(img)
```

### Audio (`audio/audio.go`)

BGM: OGG Vorbis with intro+loop byte offsets for seamless looping. SE: WAV, new player per sound for overlapping. Access via `Game.Audio()` or `Context.Audio`.

### Dialog (`scene/dialog/`)

`dialog.Overlay` draws text box + character face portraits. Face images lazy-loaded from `assets/anm/face/` via `sync.Once` cache. Triggered by `ctx.StartDialog(lines)` in stage scripts.

## Key Constants (`game/game.go`)

- Screen: 640×480, Field: 384×448 (left=32, top=16, right=416, bottom=464)
- SampleRate: 44100 Hz

## Game State (`game/state.go`)

Score, Life, Bomb, Power (0-500), Graze, Faith, Point. 2 characters × 3 shot types. 5 difficulties. Faith decays automatically.

## Update Order (per frame in Stage)

1. Input → 2. Pause check → 3. Dialog check → 4. Player → 5. Script (spawn/danmaku) → 6. Enemies → 7. Bullets → 8. Collisions → 9. Items → 10. Effects → 11. Draw (BgDraw → entities → HUD → dialog overlay)

## Conventions

- Strict Update/Draw separation: Draw reads state only, never modifies it
- Boss phases use `[]enemy.SpellCard` with Update closures capturing `ctx`
- Background textures tile vertically using `math.Mod(bgY, texHeight)` scroll offset
- `SpriteID` field on Enemy maps to `sprite.Global.Bosses[stageID]` for boss rendering
- Fallback pattern: all sprite rendering falls back to `vector.FillRect/FillCircle` if image is nil

# TH10 ECL — Ground Truth

Authoritative reference for the Touhou 10 ECL bytecode and our VM, reverse-engineered
from the official `th10.exe` (Ghidra) and cross-checked against thtk source. This is the
"don't guess, restore" record: every claim below is backed by a binary function or thtk's
loader, not inferred from observed behavior.

Sources:
- `th10.exe` decompiles in `.tmp/ecl_dump/` (regenerate via the pipeline in memory `ecl-re-pipeline`).
- thtk `thecl10.c` (container + param model), `expr.c` (system-op table), `thecl.h` (opcode constants).
- Our implementation: `scene/stage/ecl/{loader,opcodes,disasm,vm,game_ops,bullet_ops}.go`.

## 1. Container format (on-disk, little-endian)

From `thecl10.c::th10_open`. Magic `SCPT`.

```
SCPT header (36B):  magic[4] | u16 =1 | u16 include_len | u32 include_off(=36)
                    | u32 0 | u32 sub_count | u32 zero[4]
@include_off  ANIM list: magic[4] | u32 count | count× NUL-strings ; pad to 4
              ECLI list: magic[4] | u32 count | count× NUL-strings ; pad to 4
              u32 sub_offsets[sub_count]      (absolute file offsets)
              sub name table: sub_count× NUL-strings
@sub_offsets[i]  ECLH header (16B): magic[4] | u32 data_off(=16) | u32 zero[2]
                 instruction stream until sub_offsets[i+1] / EOF
```

Instruction (16B header + params):

```
u32 time | u16 id | u16 size | u16 param_mask | u8 rank_mask | u8 param_count | u32 zero
params... (size-16 bytes) ; next instr = this + size
```

`rank_mask` bits `1111LHNE`: bit0 Easy, 1 Normal, 2 Hard, 3 Lunatic (top 4 always set).

## 2. Parameters & variables (`FUN_0044fdb0`, thtk dump loop)

Each parameter consumes one `param_mask` bit, LSB first (`H` type consumes an extra bit
first). Signature types per opcode come from `th10_fmts` (see `opcodes.go`):
`S`=int32, `f`=float32, `m`=string(len-prefixed, pad 4), `x`=xor-string, `o`=jump offset,
`t`=jump time, `D`=typed call-arg(8B: from,to,zero,val), `*`=repeat next.

- **mask bit 0** → the 4 bytes are an **immediate** literal.
- **mask bit 1** → the 4 bytes are a **variable id**:
  - `id >= 0` → local stack-frame variable at **byte offset `id`** (slots are 4-byte).
  - `id == -1` → **pop** the typed data stack.
  - `id < -1` → **special/global** variable (vtable in the binary).

Crucial encoding quirk: a **float** parameter encodes its var id as a **float bit pattern**
(local `%D` at offset 12 is stored as `12.0f`; the aim special as `-9998.0f`). Int params
store the id directly. See `Param.VarID()`.

### Special variables observed in Stage 1 (behavioral mapping — refine via diff)
| id | meaning |
|----|---------|
| -9959 | difficulty / rank (0=E … 4=X) |
| -9989 | angle from this enemy to the player (`atan2(py-ey, px-ex)`) |
| -9999, -9987 | random float [0,1) |
| -9998 | random / spiral seed angle |
| -9986 | spell-practice / replay flag (0 in normal play) |
| -10000 | global frame counter |

## 3. Execution model — the virtual clock (confirmed via Boss1/Boss1At1 Time fields)

Each task has a `clock`. Per frame:
1. If an explicit `wait` is pending, decrement it and yield (clock frozen).
2. Run instructions while `instr.Time <= clock`.
3. When the next instruction's `Time > clock`, yield; `clock++` (one frame elapsed).

- `+N:` time labels in thtk text = instructions carrying an absolute `Time`; the gap is the wait.
- `goto/if/unless` (12/13/14) format `ot`: jump to `instrOffset + o`, **set clock = t**.
  Loops reset their clock with `if @offset t0` (e.g. Boss1's 200-frame loop, Boss1At1's
  1-frame-per-iteration spiral).
- `wait`(83) adds **N extra frames on top of** the clock schedule (clock frozen meanwhile).

## 4. System opcodes 0-93 — the stack machine (`expr.c`, confirmed by the ANM VM `FUN_0043ee30`)

| id | op | id | op |
|----|----|----|----|
|1|delete (RET_BIG: kill owner + end)|10|return|
|11|call (sync, `m*D`)|12|goto `ot`|
|13|unless (pop; !v → jump)|14|if (pop; v → jump)|
|15|callAsync (`m*D`)|16|callAsyncId (`mS*D`)|
|17|killAsync(id)|21|killAllAsync|
|40|stackAlloc (frame bytes → `var`)|42/44|push int/float|
|43/45|pop → int/float var|50-58|+ - * / % (I and F variants)|
|59-70|== != < <= > >= (I/F)|71-77|! && \|\| ^ \| &|
|78|dec (push old, var--)|79/80|sin/cos|
|81|polar (out_x,out_y = cos/sin(angle)·r)|82|validRad (normalize angle var)|
|83|wait(N)|84|neg|88|sqrt|

Expression ops pop operands off the typed data stack and push the result; `LOAD` pushes a
resolved value; `ASSIGN`/`DEC` write a frame variable. `eVal` carries int/float bits and
coerces on read, so mixed int/float arithmetic (`push int 32` then `addf`) works.

## 5. Game opcodes 256-436 (`FUN_0040e770`)

### Enemy create & visuals
- 256/257/260/261/265-268 enmCreate `mffSSS` = (subName, x, y, hp, score, dropPower). The
  named sub runs as the new enemy's routine (owner = new enemy). x,y are ECL coords
  (center-relative X, top-relative Y) → world via `eclToWorld`.
- 258 anmSelect, 259/262 anmSetSprite/Main, 263/264/269 anmPlay → enemy sprite/fairy kind.

### Movement — two layers, final pos = layer A (`enemy+0x58`) + layer B (`+0x84`)
Each opcode has an A form and a "+2" B form. Most enemies use only layer A.
- 280/282 setVel `ff` (angle, speed) instant.
- 281/283 movePosTime `SSff` (time, mode, x, y) — position ease over `time`, easing by `mode`.
- 284/286 setDir/Vel `ff`.
- 285/287 moveVelTime `SSff` (time, mode, angle, speed) — velocity ease; **mode 7 =
  continuous acceleration** (no target); angle uses 2π shortest-path; args ≤ sentinel
  (`_DAT_00470d58`, ≈ large negative) mean "keep previous".
- 288/290 moveCircle, 292/293 moveRand `SSf` (time, mode, speed) — **random heading**
  (not a target point) + steer toward field interior when at a `moveLimit` edge.
- 296/298 moveAdd, 324 moveLimit `ffff`, 325 clear.

### Enemy state / drops / flow
320 setHurtbox, 321 setHitbox, 322/323 flagSet/Clear, 326 dropClear, 327 dropExtra,
329 dropItems, 330 dropMain, 331 setHP, 332 setBoss, 333 timerReset, 334 delayedCall,
335 setInvuln, 336 playSound, 338/339 dialog, 340 deathWait, 343 spellEnd,
355/356 rankPick (5-way by difficulty). 337/341/342/344/347/357-360 are HUD/bookkeeping
(approximated/no-op).

### Bullets — the "ET" emitter templates (each enemy has 16, 0x210B each)
- 400 etNew, 402 sprite/color, 403 offset, 404 angle/angleInc, 405 speed/speedInc,
  406 count(ways `+0x4b8`)/stacks(`+0x4ba`), 407 aim, 408 sound, 409 etEx (22 timed
  modifiers ×0x18B), 411 copy, 410 clearAll, 401 **fire**.
- 401 emits `stacks × ways` bullets: `FUN_004073e0` loops stacks (outer) × ways (inner)
  and calls per-bullet `FUN_004067d0(way, stack, aimAngle)`, `aimAngle = atan2(py-y, px-x)`.
  **Now ported faithfully** (`emit`/`perBullet` in `bullet_ops.go`): speed interpolates
  `speed → speed2` across stacks; angle by template mode (`tp.aim` = `unaff_EBX[0xfc]`):
  0/1 centered fan (0 adds aim, odd index mirrors via `-1`), 2/3 even ring (`way·2π/ways`,
  2 adds aim), 4/5 offset ring (`+π/ways`), 6/7/8 random angle/speed/both. Constants
  (`0.5, 2π, π, -1`) from `consts.txt`. RNG (modes 6-8) stays behavioral (Pareto).
- 412 laserOnA `SSffffSf`, 413 laserStOn `SSSfffSSSSfS` (moving / stationary lasers).
- 420 etCancel / 421 etClear (clear active bullets); 425-427 rank-interpolated count.

## 6. Deliberate approximations (Pareto, per project decision)
- **RNG**: a deterministic xorshift, not ZUN's 16-bit `FUN_0044bb90` — behavior-faithful,
  not frame-perfect.
- **Movement easing modes**: a small curve set (linear / accel / decel / smooth); exact ZUN
  easing function pointers (`&DAT_00476f78…`) not ported.
- **Random ET modes (6-8)** and **lasers (412/413)** and **`etEx` (409) timing** are still
  approximate; the deterministic per-bullet emission (modes 0-5) is ported from `FUN_004067d0`.
- **UI/HUD opcodes** (spell cards, chapters, stars, screen shake) are no-ops.

These are the points to tighten with the differential harness (dump live th10.exe per-frame
enemy/bullet traces for Stage 1 and diff against the VM).

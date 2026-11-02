# M4 模式调查：嵌入 .anm 脚本能用到的渲染模式（2026-11-01）

依据：`anm/vm.go`（`case 65`、`case 67`、`case 74`、`case 84`、`case 88`、`Start`、`Reset`、`readI`、`intPtr`），`anm/anmdraw/anmdraw.go`（`Draw`、`DrawLayer`），`anm/anmdraw/mode46.go`（`corners` 的锚点 3 panic），`docs/research/m4_s3.md`（§1 方法）。调查程序是一次性的 `.tmp/scratch/modesurvey/main.go`（不在仓库包内，`.tmp` 已被 `.gitignore` 忽略），输出保存在 `.tmp/scratch/modesurvey/out.txt`。

---

## 0. 结论

调查覆盖嵌入的全部 17 个 .anm 文件，共 768 个脚本，逐脚本扫描，子脚本单独计。

**一行结论**：嵌入的 .anm 只用到模式 0、1、2、4、6、8、9。模式 5、7 与 10–15 一个脚本都没有。模式 9 只出现在 `bullet.anm`（18 个脚本，全部是 op 88 子脚本）。在会 panic 的模式 5、7、9 中，只有模式 9 实际出现。（2026-11-01 更新：模式 9 已移植，`anm/ring.go`、`anm/anmdraw/mode9.go`；5、7 仍 panic。）

下表按 (文件, 模式) 列出模式 4、5、6、7、9、10–15 的使用情况。“脚本数”是静态扫描中可设置该模式的脚本个数（默认模式 0 也计入）。

| 文件 | 模式 | 脚本数 | 脚本 id | 备注 |
|---|---|---|---|---|
| bullet.anm | 4 | 9 | 410, 411, 412, 413, 414, 428, 429, 432, 433 | 全部是根脚本（无父脚本）；无 op 65；全部含 op 74（参数 1，置 flagScroll），见 §6 |
| bullet.anm | 9 | 18 | 394–397, 401–404, 415, 416, 423–426, 435–438 | 全部是 op 88 子脚本，父为 400、407、417、427、441；op 84 参数是变量 $10000，见 §4 |
| stage01.anm | 6 | 2 | 3, 5 | STD 的 quad 脚本；无 op 65，锚点为 0 |
| stage05.anm | 6 | 2 | 3, 5 | STD 的 quad 脚本；无 op 65，锚点为 0 |
| 模式 5 | — | 0 | 无 | 任何文件都没有 |
| 模式 7 | — | 0 | 无 | 任何文件都没有 |
| 模式 10–15 | — | 0 | 无 | 任何文件都没有 |

**每个模式的总数**（脚本数，文件数，来自 §2 的矩阵）：

- 模式 4：9 个脚本，1 个文件（bullet.anm）。
- 模式 5：0 个脚本，0 个文件。
- 模式 6：4 个脚本，2 个文件（stage01、stage05）。
- 模式 7：0 个脚本，0 个文件。
- 模式 9：18 个脚本，1 个文件（bullet.anm）。
- 模式 10–15：均为 0 个脚本，0 个文件。

其它要点：

- **加载方（模式 5、7、9）**：只有模式 9 出现，只在 `bullet.anm`。它由 `scene/stage/bullets.go`（`bulletFile()`，子弹管理器、炸弹、激光、道具的 File）和 `scene/stage/bomb.go`（炸弹的根脚本 400、407、441）加载，并由 `scene/stage/ecl/game_ops.go` 的 ECL 槽 0 路径使用。详见 §5。
- **锚点 3（模式 4、6）**：所有嵌入脚本中没有 op 65 设置锚点 3（36 条 op 65 的锚点值都是 0、1、2）。模式 4、6 的 13 个脚本都没有 op 65，锚点保持 0。
- **模式 9**：op 84 的参数都是变量 $10000（`VAR-MASK`）。每个脚本在 op 84 之前的 t=0 时刻用字面量写入 $10000（值 32 或 9）和 $10001（值 3 或 8），见 §4。
- **op 74（flagScroll）**：参数非零的脚本有 9 个，全部是 bullet.anm 的模式 4 根脚本，参数都是 1（奇数，置位 0x2000）。
- **变量模式参数**：op 67 的参数都没有掩码，全部是字面量。没有变量模式参数。

---

## 1. 方法

1. **文件清单**：`fs.Glob(assets.Assets, "anm/*.anm")` 列出 17 个文件（bullet、enemy、pl00、stage01–07、stgenm01–07），用 `gamedata.ANM` 解析。不读 th10.dat。
2. **模式**（同 m4_s3.md §1 第 3、4 条）：op 67 设置 `Flags` 的模式位，模式 = `Arg(0) & 0xf`（`case 67`）；op 84 把模式置为 9（`case 84`）。
3. **默认模式 0**（同 m4_s3.md §1 第 5 条）：`Reset` 后 `Flags = 7`，模式为 0。本次调查的判定规则与 m4_s3 相同。另加一个更保守的规则（把 op 63、69、75、81 也当作控制指令）计算 `default0Ext`。两者在所有脚本上结果一致（§7 限制 3）。
4. **锚点**：op 65 把 `Flags` 的位 18–19 设为 `Arg(0) & 3`（X 轴），位 20–21 设为 `(Arg(0) >> 16) & 3`（Y 轴）。`anmdraw` 读取 `Flags>>18&3` 与 `Flags>>20&3`，与之一致。
5. **op 84 参数**：`RingN = readI(in, 0)`，即掩码感知读取。记录原始 `Arg(0)` 与掩码位。
6. **$10000 / $10001 的写入**：op 5、6、8、10、12、14、16、18、20、22、24、26、40 的目的参数是参数 0，经 `intPtr(in, 0)` 写入。只有掩码位 0 置位且参数值为 10000 或 10001 时，才是对变量的写入（`intPtr` 只在掩码置位时映射到变量）。`$10000` 对应 `vm.I[0]`，`$10001` 对应 `vm.I[1]`（`readI` 中 `v-10000` 的索引）。`docs/research/m4_r6.md` 第 20、306 行把模式 9 的顶点数记为 `2·VM+0x30c`，任务说明据此把 `$10000` 对应 `VM+0x30c`、`$10001` 对应 `VM+0x310`。这两个偏移没有在二进制中核对（`m4_r6.md` 第 309 行称 `VM+0x30c` 的写入者未找到），Go 源码的 `I` 也没有偏移注释。
7. **op 88 子脚本**：字面 id（掩码位 0 清，且值在 0–9999 之间）递归跟进。每个子脚本作为独立 VM 单独计数，`parents` 列出所有父脚本（同一父脚本多次 spawn 同一子脚本时会重复列出）。
8. **op 74**：参数非零的脚本。`flagScroll` 由 `Flags&^0x2000 | Arg<<13 & 0x2000` 设置，因此只有参数的最低位为 1 时置位；偶数的非零参数会清位。本次调查中没有偶数的非零参数。
9. **装载方**：在 Go 代码中搜索文件名与 STD 的 `ANM` 字段；ECL 的 ANIM 列表经 `ecl.LoadProgram` 读取（`scene/stage/ecl/loader.go` 的 `AnimNames`）；STD 的 ANM 经 `std.Load` 的 `f.ANM` 读取；ECL 中 ops 259、262、263、264 的字面脚本参数也检查了一遍（§5）。
10. **复现**：从仓库根目录运行 `go run ./.tmp/scratch/modesurvey`，输出即 `out.txt`。

---

## 2. 各文件的模式矩阵

每格是可设置该模式的脚本数（默认模式 0 也计入），`-` 表示 0。

| 文件 | 0 | 1 | 2 | 4 | 6 | 8 | 9 | 5、7、10–15 |
|---|---|---|---|---|---|---|---|---|
| bullet.anm | 117 | 362 | 16 | 9 | - | - | 18 | 0 |
| enemy.anm | 45 | 18 | - | - | - | - | - | 0 |
| pl00.anm | 14 | 18 | - | - | - | - | - | 0 |
| stage01.anm | - | - | - | - | 2 | 5 | - | 0 |
| stage02.anm | - | - | - | - | - | 3 | - | 0 |
| stage03.anm | - | - | - | - | - | 9 | - | 0 |
| stage04.anm | 1 | - | - | - | - | 4 | - | 0 |
| stage05.anm | - | - | - | - | 2 | 5 | - | 0 |
| stage06.anm | 2 | - | - | - | - | 12 | - | 0 |
| stage07.anm | - | - | - | - | - | 10 | - | 0 |
| stgenm01.anm | 16 | 2 | - | - | - | - | - | 0 |
| stgenm02.anm | 17 | 2 | - | - | - | - | - | 0 |
| stgenm03.anm | 19 | 4 | - | - | - | - | - | 0 |
| stgenm04.anm | 16 | 2 | - | - | - | 6 | - | 0 |
| stgenm05.anm | 15 | 2 | - | - | - | - | - | 0 |
| stgenm06.anm | 31 | 7 | - | - | - | - | - | 0 |
| stgenm07.anm | 39 | 4 | - | - | - | - | - | 0 |
| **合计（脚本数）** | 332 | 421 | 16 | 9 | 4 | 54 | 18 | 0 |
| **合计（文件数）** | 12 | 10 | 1 | 1 | 2 | 8 | 1 | 0 |

模式 3 没有出现在任何文件中（矩阵中省略该列，合计为 0）。

---

## 3. 模式 4 与 6 的锚点检查

- **模式 4**：bullet.anm 的 9 个脚本（410、411、412、413、414、428、429、432、433）都没有 op 65，锚点保持 `Reset` 后的 0。
- **模式 6**：stage01.anm 与 stage05.anm 的脚本 3、5 都没有 op 65，锚点同样为 0。
- **全部文件**：36 条 op 65 的锚点值为 `0x20000`（ay=2，25 条）、`0x2`（ax=2，2 条）、`0x10001`（ax=1、ay=1，9 条）。没有任何 op 65 的锚点为 3。`docs/research/m4_render.md` 第 329 行记录的 “46 个 .anm 中未出现锚点 3” 与此一致（那个 46 文件的结论在此没有复核）。

因此，目前嵌入的脚本不会触发 `corners` 中的锚点 3 panic。

---

## 4. 模式 9：op 84 的参数与 $10000 / $10001 的写入

18 个模式 9 脚本都在 bullet.anm 中，都是 op 88 子脚本。下表按父脚本分组。每个脚本的写入都在 t=0，且先于 op 84。写入的源都是字面量（参数 1 的掩码位为 0）。

| 脚本 | 父脚本（op 88 字面量） | 写 $10000（op 6，t=0） | 写 $10001（op 6，t=0） | op 84（参数 $10000） | 写入的值 → RingN |
|---|---|---|---|---|---|
| 394、395、396、397 | 400（396、397 各被 spawn 两次） | @1，字面量 32 | @2，字面量 3 | @3 | 32 |
| 401、402、403、404 | 407（403、404 各两次） | @1，字面量 32 | @2，字面量 3 | @3 | 32 |
| 415、416 | 417 | @1，字面量 32 | @2，字面量 3 | @4 | 32（另有 @3 写 $10002 = 600，不是 $10000/$10001） |
| 423、424、425、426 | 427 | @1，字面量 9 | @2，字面量 8 | @3 | 9 |
| 435、436、437、438 | 441（437、438 各两次） | @1，字面量 32 | @2，字面量 3 | @3 | 32 |

要点：

- 每个模式 9 脚本的 op 84 参数都是 `$10000`，掩码位 0 置位（变量）。它的值由同一脚本中 op 6 在 t=0 写入的字面量决定，所以静态上 RingN 只有 32 和 9 两种。
- 每个模式 9 脚本对 $10000 和 $10001 只有上表中的那一次写入。
- 子脚本是独立 VM（`spawnChild` 调用 `New()`），变量 `I` 从零开始，它的 $10000 不与父脚本共享。上面的写入都是子脚本自己的写入。
- 父脚本 400、407、417、427、441 本身是模式 0（无 op 67、84，`default0` 为真），且不是任何其它脚本的子脚本，因此它们是从文件外启动的根脚本。

---

## 5. 加载方与绘制路径

**模式 5、7、9 的文件**：只有 bullet.anm 用到模式 9（模式 5、7 无人使用）。它由以下代码加载：

- `scene/stage/bullets.go:30`：`bulletFile()` 调用 `gamedata.ANM("bullet.anm")`，作为子弹管理器的 File，并传给 `bullet.Env`、`item.Env`、`laser.Env`。
- `scene/stage/bomb.go:72–79`：灵击的根脚本 `id := 0x190`（灵梦，即 400）、`0x197`（魔理沙，即 407）、`0x1b9`（重型符卡，即 441），经 `c.World.Spawn(bulletFile(), id, …)` 生成。模式 9 的子脚本经 `env.Spawn`（即 `World.Add`）进入世界。
- `scene/stage/ecl/game_ops.go:752–776`：ECL 的文件索引 0 是 bullet.anm，索引 i（i ≥ 1）是 `AnimNames[i-1]`。`scriptSize` 按此索引加载文件。`deathEffect`（`game_ops.go:874`）选择 0x164、0x167、0x16a、0x16d（即 356、359、362、365，模式 0 的根脚本，子脚本为模式 1）在 `game_ops.go:912` 生成。
- `entity/bullet/bullet.go:154`、`entity/laser/laser.go:44`、`entity/item/item.go:124`：`File` 字段都是 bullet.anm。
- 脚本 417（0x1a1）与 427（0x1ab）：`docs/research/systems_gaps.md:63` 记为符卡宣言时创建的名牌与 cut-in 对象（原作 `FUN_00409280`）。Go 代码中没有字面引用，ECL 中 ops 259、262、263、264 的字面脚本参数也没有这两个 id。它们在当前代码中不会被生成，属于尚未移植的部分。

**绘制路径**（决定模式 9 是否会在现行代码中命中 `Draw`）：

- `anmdraw.Draw` 的现行调用者只有 `std/stddraw/stddraw.go:123`（`Node7` 中的 quad）与 `:325`（`pass` 中的 slot）。`pass` 由 `Node7`（pass 0–7）与 `Node0A`（pass 8–11）调用。`Node7` 由 `scene/stage/chain_legacy.go:49` 在 `s.bg` 为 STD 时调用，`Node0A` 由 `scene/stage/chain.go:71` 调用。这些调用处的对象都是 STD 的（stageNN.anm）。
- `DrawLayer` 在现行代码中没有调用者，只在 `scene/stage/std_smoke_test.go:47` 中被测试调用。`Node0A` 的 `layer` 回调在 `chain.go:71` 中走的是旧的图层绘制（`layers(l)`），不是 `DrawLayer`。
- 因此 bullet.anm 的对象（炸弹、子弹、特效、模式 9 的子脚本）现在都不经过 `anmdraw.Draw`。模式 9 目前不会 panic，但一旦 World 图层改为经由 `DrawLayer` 绘制（`std/stddraw/stddraw.go:152` 注释中提到的计划），炸弹与模式 9 子脚本就会命中 `Draw` 中模式 9 的 panic 分支。（2026-11-01 更新：模式 9 已移植，不再 panic。）
- 模式 9 的对象只有在 `Draw` 的前置条件满足（`Flags&3 == 3` 且 `Alpha != 0`）时才会 panic。这是运行时条件，本次静态调查没有判断。

**其它文件的加载方**（只作背景，这些文件没有模式 5、7、9）：

- `enemy.anm`：ECL 的 `AnimNames` 第 1 项（`default.ecl` 与所有 `stageNN.ecl`）。绘制经 `sprite/loaders.go:39` 的 `EnemyAnm.ScriptSprite`（`scene/stage/stage.go:485–489`）；`entity/enemy` 只保存 `MainScript` id。
- `stgenmNN.anm`：各 `stageNN.ecl` 的 `AnimNames` 第 2 项（`stage01.ecl` 为 `stgenm01.anm`，依此类推）。`scene/stage/stage.go:505–509` 与 `sprite/loaders.go:40` 只使用 `stgenm01.anm`。
- `pl00.anm`：`sprite/loaders.go:36`（`ReimuAnm`），`scene/stage/stage.go:443–464` 的 `ScriptSprite`；`entity/playerbullet/playerbullet.go:22` 的 `ScriptID` 注释。
- `stageNN.anm`：`stageNN.std` 的 `f.ANM` 字段，由 `scene/stage/stage.go:228–241` 的 `loadSTD` 读取并 `Enable` 给 STD。

---

## 6. op 74（flagScroll）

有 9 个脚本执行了参数非零的 op 74，都是 bullet.anm 的模式 4 根脚本：

| 脚本 | op 74 位置 | 参数 | 是否置位 0x2000 |
|---|---|---|---|
| 410、411、412、413、414、428、429、432、433 | @4，t=0 | 1 | 是（最低位为 1） |

- 这些脚本都是根脚本（无父脚本），与模式 4 的脚本集合相同。
- 全部 op 74 都是字面量（掩码位 0 清）。没有参数为偶数非零的 op 74，因此没有清位的情况。
- `flagScroll`（0x2000）的含义是 `Offset` 跟随舞台滚动（`anm/vm.go:29`）。
- Go 代码中没有这些脚本 id 的字面引用（`entity/`、`scene/stage/`、`std/`、`anm/` 中均未找到）。激光的光束脚本是 `bullet.ScriptID(type, colour)`，即 `scriptBase[type] + colour`，最大约为 381（colour 取 0–7 时）；激光的发射器是 `colour + 0x103`（259–266）。两者都达不到 410 以上。这些脚本在原作中的用途未确认。

---

## 7. 限制

1. **只覆盖嵌入的 17 个文件。** `anm/anmdraw/anmdraw.go:109` 的注释称 TH10 有 46 个 .anm。`docs/research/th10c_dat.md` 列出的 `front.anm`、`title.anm`、`staff.anm`、`stXXlogo.anm`、`e07–e09.anm` 等文件都没有嵌入（`assets/anm/front/` 是 PNG 目录，不是 .anm）。未嵌入文件的模式、锚点与子脚本都不在本次结论内。要覆盖它们，需要先用 `tools/th10extract` 嵌入（`gamedata` 的文档注释描述了这一流程）。
2. **静态扫描不执行，也不建模跳转。** 模式集合是全部 op 67、84 参数的并集，包含死代码与不可达的分支，因此是上界。op 1、2、4、5、28–39 的控制流，以及 op 63、64、69、81 的标签与中断，都没有建模。本次的结论偏保守，不会漏掉模式，但可能多算。
3. **默认模式 0 的规则。** m4_s3 的规则不把 op 63、69、75、81 当作控制指令。本次另算了加入它们的 `default0Ext`，在所有脚本上与 `default0` 相同（`out.txt` 中 “default0 differences” 为空）。因此这一点不影响本次结论。
4. **变量参数。** op 67 的参数都是字面量（掩码位 0 全为 0）。op 65 与 op 74 的掩码全为 0。op 84 的参数是变量 $10000（18 个脚本，`VAR-MASK`）。Go 的 `case 67` 直接读 `Arg(0)`（不走 `readI`），但本数据中没有变量参数，所以不影响结论。
5. **op 88 子脚本。** 所有 op 88 都是字面量（没有非字面量），也没有指向不存在脚本的 id。
6. **脚本 id 重复。** 每个文件的脚本表都检查过（`rawIDs`，按 anm.Load 的遍历方式），没有重复 id。`anm.Load` 对重复 id 是覆盖，因此这一点很重要，本次确认没有。
7. **外部写入。** 本次只统计脚本内的写入。没有检查 Go 代码中对 VM 变量 `I`、`Flags` 的外部写入。`anm` 包外对 `Flags` 的写入（如 `entity/bullet/update.go` 的 `b.VM.Flags |= 4`）已经看过，不涉及模式位、锚点位或 flagScroll。
8. **运行时条件。** 没有判断 `Draw` 的前置条件（`Flags&3 == 3`、`Alpha != 0`），也没有判断脚本是否真的被生成与可见。§5 的结论是“现行代码的调用图”，不是运行时的覆盖率。

---

## 8. 复现

```
go run ./.tmp/scratch/modesurvey
```

该程序是一次性的，不属于仓库代码。`.tmp` 以点开头，`./...` 的包模式不包含它，所以它不参与 `go build ./...`、`go vet ./...` 与 `go test ./...`。输出保存在 `.tmp/scratch/modesurvey/out.txt`。

# t1 自机系统交付记录

更新时间：2026-11-07。工作树：`D:/GoProjects/th10/.tmp/codex/t1/tree`。

## 1. 摘要

- 阶段 1（D1e-1）：实现已落地，已跑完完整验收，但结果未绿，尚未标记完成。自机本体改用角色级共享的 `pl00.anm`／`pl01.anm` 与嵌入 VM；补齐左右切换、低速判定点、蓝色无敌闪烁、被弹特效和信仰保持矩形。六章 trace 均与基线逐字节相同，motion 一致帧数没有下降。
- 阶段 1 的旧自机精灵路径、ganim8 依赖和 `entity/effect` 池已删除；副机与自机子弹仍由旧节点绘制，按任务要求留待阶段 2。
- 阶段 2、3、4：未开始。严格保持阶段顺序，当前规格测试有待架构师修正的问题，不能宣称阶段 1 通过或整个任务完成。
- 没有修改已有规格测试、golden、主树源码或 `docs/fidelity.md`，没有运行原作或 probe。

## 2. 改动文件清单

新增：

- `entity/player/anm.go`：角色级 ANM 缓存、`AttachAnm` 和直接绘制／矩形路径。
- `entity/player/anm_test.go`：新增自机 VM、判定点、闪烁、被弹和矩形的单元测试。
- `docs/handoff/codex_t1.md`：阶段性交付和验收记录。

修改：

- `anm/anmdraw/rect.go`：按文档实现无纹理 alpha 混合三角带及缓存失效。
- `entity/player/player.go`：新增嵌入 VM 和句柄，移植切换／跟随／无敌／被弹顺序，删除手写动画状态。
- `scene/stage/bullets.go`：接入自机 ANM，复用已有的 `bullet.anm`，删除旧被弹池生成。
- `scene/stage/chain_legacy.go`：旧子弹／副机之后直接画自机 VM，明确记录矩形配置门控恒假。
- `scene/stage/stage.go`：删除旧特效池、更新和绘制，以及旧自机绘制函数。
- `sprite/loaders.go`：删除自机精灵加载器、`Global.Player` 和 ganim8 动画类型。
- `go.mod`：删除不再使用的 ganim8 依赖，未增加依赖。
- `go.sum`：删除 ganim8 的两项校验。

删除：

- `entity/effect/effect.go`：删除整个旧几何特效包。

## 3. 移植的函数地址

| 原作地址 | Go 对应 | 说明 |
|---|---|---|
| `FUN_0043bda0` | `anmdraw.Renderer.Rect` | Flush → 四角 → 无纹理 → alpha 混合 → 三角带 → Invalidate。 |
| `FUN_004247f0`／`FUN_00404f30`，`0x4247f8–0x42481c`、`0x4248ab–0x4248b1` | `player.New`、`playerFiles`、`Player.AttachAnm` | 角色选择文件、以脚本 0 启动不入 World 的嵌入 VM。 |
| `FUN_004250b0`，`0x4251e0–0x425245`、`0x4252d0–0x4252f8` | `Player.move` | 判定点脚本 `0x160` 的生成／中断。 |
| `FUN_004250b0`，`0x425385–0x4253f8` | `Player.move` | 使用原始 vx 与上一帧 vx 的符号启动脚本 1／3／2／4。 |
| `FUN_004250b0`，`0x4254c8–0x42551b` | `Player.move` | `syncPos` 后、副机更新前，写判定点原始屏幕 Offset。 |
| `FUN_00425730`，`0x425e46–0x425efb`、`0x42618e`、`0x4261f2` | `Player.tail` | 无敌递减 → 旧状态时钟决定蓝色闪烁 → VM.Run → 状态／初始化时钟递增。 |
| `FUN_00426cf0`，`0x426d09–0x426e05`、`0x426ecb–0x426ed0` | `Player.Hit` | state 4 → 尾插 `0x162` 一个与 `0x163` 三十二个 → 状态时钟复位 → 无敌 6 → 自机脚本 0。 |
| `FUN_00426360`，`0x426366–0x4263a5`、`0x426441–0x4264e8` | `Player.Draw` | state 2 跳过，自机位置与两个矩形；逐步 float32 运算。 |
| `FUN_00427c50` → `FUN_00452ff0` | `Player.Draw` 内的 `math.Floor` | 按任务确认的 floor 语义使用。 |

矩形公式已逐条核对 `r8/func_426360.asm`：`PUSH ECX` 后的栈基准、先乘宽度再加 x1、两个矩形的颜色与整体平移均与任务设计一致。宽度常量使用位模式 `0x3e95a95b`，没有替换为简化的十进制常量。

## 4. 测试

新增测试：`TestPlayerAnmFiles`、`TestMoveAnmTurns`、`TestMoveAnmSameSign`、`TestFocusHitbox`、`TestInvincAnmBlink`、`TestHitAnm`、`TestHitAnmBeforeClockReset`、`TestPlayerEffectsUseAltRNG`、`TestPlayerDraw`、`TestFaithHoldRectFloat32`、`TestFaithHoldRectGates`、`TestPlayerNilAnm`。

- `TestMoveAnmTurns` 覆盖四种切换、左右反向和冲突方向；极低游戏速度下实际步长截断为零，仍按原始 vx 切换。
- `TestFocusHitbox` 覆盖初始化锁定、生成、避免重复生成、跟随、中断而非删除、失效句柄回收和下一帧重新生成。
- `TestInvincAnmBlink` 覆盖无敌 1 递减到 0 的闪烁帧、未推进状态时钟、非三倍数、非无敌状态和颜色保留。
- `TestHitAnm` 检查三十三个主特效的数量／顺序／Offset；子 VM 生成回调用来观察特效必须先于时钟复位。
- 矩形测试比较 float32 位模式，包括 `y = 7.999999f` 在 `y+17` 处先舍入到 25 的边界。

完整验收结果：

| 检查 | 结果 |
|---|---|
| `go build ./...` | 退出码 0，通过。 |
| `go vet ./...` | 退出码 0，通过。 |
| `go test -count=1 ./...` | 退出码 1；只有不可修改的 `anm/anmdraw.TestRect` 失败，其余包通过，golden 没有变化。 |
| WASM，排除 tools | 46 个包构建，退出码 0，通过。先在宿主目标下枚举包，再用 `GOOS=js GOARCH=wasm` 构建。 |
| `gofmt -l .` | 未通过，列出 37 个继承文件；本次修改的 Go 文件及新增测试均已 gofmt。 |
| `golangci-lint run` | 退出码 1，共 42 项：errcheck 30、ineffassign 1、staticcheck 5、unused 6。基线也为相同分类／数量，t1 新增代码没有 lint 报告。 |

日志在 `D:/GoProjects/th10/.tmp/codex/t1/logs/stage1/`：`build.txt`、`vet.txt`、`test.txt`、`wasm.txt`、`wasm_packages.txt`、`lint.txt`、`lint_base.txt`、`gofmt.txt`、`gofmt_base.txt`。

环境说明：第一次 lint 因默认缓存指向沙盒外的 `%LOCALAPPDATA%/golangci-lint` 而报缓存写入拒绝；随后把 `GOLANGCI_LINT_CACHE` 设到 `.tmp/codex/t1/lintcache`，复跑不再有权限警告，仍留下上述 42 项基线问题。没有请求提权或绕过沙盒。规则所给的 WASM 枚举命令在 tools/th10probe 处会报告 Windows 构建约束，因此最终检查先在宿主环境枚举并排除 tools，再做 WASM 构建；非 tools 包列表完整保留在日志中。

规格测试失败：

```text
--- FAIL: TestRect
    rect_test.go:56: vertex 0
        got  {X:100 Y:100 Z:0 RHW:1 Diffuse:4294967295 Specular:4278190080 U:0.25 V:0.5}
        want {X:84 Y:92 Z:0 RHW:1 Diffuse:4294967295 Specular:4278190080 U:0.25 V:0.5}
```

原因：`rect_test.go` 以 `obj(f, 0, 2, 1, 1)` 创建对象，`obj` 把最后两个参数写入 x／y 锚点；锚点 1 是左上锚点，所以位置 `(100,100)`、尺寸 `32×16` 的四边形为 `(100,100)–(132,116)`。最后一条断言却期待中心锚点 0 的 `(84,92)–(116,108)`。原作依据是 `FUN_00443290` 的 `0x443307–0x443333` 与 `0x4433c9–0x4433fc`；中心锚点分支分别在 `0x443335` 和 `0x4433fe`。本次 `Rect` 不改变 VM 或 ANM 四角暂存，前三组设备调用与矩形顶点断言已通过。

建议由架构师把测试对象改为 `obj(f, 0, 2, 0, 0)`，或者把末尾断言改成左上锚点对应的坐标。t1 没有修改该测试，也没有为它改变锚点语义。

`gofmt -l .` 的 37 个文件全部也在基线的 39 个文件清单里；差额是本任务删除的 `entity/effect/effect.go` 和已格式化的 `scene/stage/bullets.go`。逐文件核对发现 36 个文件只是 CRLF → LF，另一个 `scene/dialog/scripts.go` 还有原有排版差异。没有对这些无关文件做批量修改。完整清单见 `gofmt.txt`。

规格测试 SHA-256 与基线相同：`F3BCEC058CE8CDE4188C04DD47C7338B960F7E25C5BB65F709FE3F3865FF1A70`。ganim8 的 require 与两个 sum 项为手工删除，未运行 `go mod tidy`，没有引入其他依赖变更。

## 5. Trace 结果

六章均以 `th10_05.rpy`、`-frames 0` 顺序运行；cmp 使用 `D:/GoProjects/th10/.tmp/trace/d1c2/base_cN.txt`。结果：

| 章 N | 基线 cmp | 原作参考 | motion 一致帧数 | 与规则基线相比 |
|---|---|---|---|---|
| 0 | 退出码 0，逐字节相同 | `orig_d.txt`，见下方资料问题 | 15853 | 不下降。 |
| 1 | 退出码 0，逐字节相同 | `orig_05_s2.txt` | 17733 | 不下降。 |
| 2 | 退出码 0，逐字节相同 | `orig_05_s3.txt` | 29763 | 不下降。 |
| 3 | 退出码 0，逐字节相同 | `orig_05_s4.txt` | 29469 | 不下降。 |
| 4 | 退出码 0，逐字节相同 | `orig_05_s5.txt` | 39343 | 不下降。 |
| 5 | 退出码 0，逐字节相同 | `orig_05_s6.txt` | 40099 | 不下降。 |

**c0 原作资料问题：** 按规则指定的 `orig.txt` 比较，首次分歧为第 4915 帧的 MSG：原作记录仍是 clock 34／op 16，基线与本任务都是 clock 35／op 12。直接对 `base_c0.txt` 运行同一个命令也得到完全相同的分歧，故不是移植回归。改用已有的 `orig_d.txt` 分别比较基线和本任务，两者都 MATCH 15853 帧，符合规则列出的一面基线数字。这与 `docs/fidelity.md` 更新记录中的 op 16 采样半帧问题一致。没有修改任何原作 trace；建议架构师修正规则第 6 节 c0 的参考路径为 `orig_d.txt`。

没有与基线的分歧帧；自机本体和新增判定点／被弹 VM 没有影响游戏 RNG 或 trace 字段。新增的原版特效单元测试也确认只消耗备用 RNG。

本任务生成的 `c0.txt` 至 `c5.txt` 均已在比较后删除，仅保留小日志：`sim_cN.txt`、`motion_cN.txt`、`trace_summary.txt`，以及 c0 原路径问题的 `motion_c0_provided.txt`、`motion_base_c0.txt`、`motion_base_c0_orig_d.txt`。三、四、六面原作参考结束后的 1699／1301／5288 个复刻帧不在原作 trace 覆盖范围内，没有宣称这些尾部已做原作差分。

## 6. 待决与推测

- 阶段 1 的 `TestRect` 规格存在锚点／期望不一致，需架构师修正或明确授权后才能通过；目前不可宣称阶段完成。
- 基线的 42 项 lint 和 37 个继承格式问题尚未处理；t1 不扩大任务范围去修改音频、回放、对话、probe 等其他子系统。需要架构师统一处理验收基线。
- 共用规则给出的 `orig.txt` 在 c0 第 4915 帧含半帧 MSG 采样；使用现成 `orig_d.txt` 可以复现 15853 帧全匹配，原路径错误也已用未改基线复验。
- 未改动相关场景以外的代码；全仓检索确认选人等场景没有使用 `Global.Player` 或 `LoadPlayer`，不需要保留旧加载器。
- env 或 world 为 nil 时，自机 VM／判定点／特效／直接绘制都是空操作；逻辑计时和移动仍然运行。相关兼容路径有单元测试。
- 当前没有新增 dump 请求。阶段 3 的回调取证和阶段 4 的灵击演出核对尚未开始，不能将其写为已移植。

## 7. 建议的 `docs/fidelity.md` 改动

当前验收未绿，不建议将总账标记完成。待规格修正并验收通过后，可采用以下新增状态行：

```text
| 自机本体／判定点／被弹特效／信仰保持矩形 | 移植（待用户目视确认） | `FUN_004247f0`／`FUN_004250b0`／`FUN_00425730`／`FUN_00426cf0`／`FUN_00426360`、`docs/research/m4_r10.md`、`docs/research/m4_r11.md`；自机嵌入 `pl0N.anm` VM，脚本 0–4 直接启动；判定点与被弹特效为备用 RNG 的 World VM；矩形使用 `anmdraw.Rect`，关卡接线因隐藏配置位 0x80 恒假而不绘制 | `entity/player/anm_test.go`、`anm/anmdraw/rect_test.go`、th10_05 六章逐字节差分、待 L3 |
```

候选更新记录（尚未满足验收，暂不写入总账）：

```text
- 2026-11-07 — D1e-1：自机本体改由角色级共享 `pl00.anm`／`pl01.anm` 的嵌入 VM 驱动（`entity/player/anm.go`、`player.go`）；按原始 vx 符号启动脚本 1／3／2／4，低速判定点 `bullet.anm` 0x160 以标签 1 淡出，无敌闪烁改用蓝色 Color2 与 0x8000，被弹尾插 0x162 一个和 0x163 三十二个。`anmdraw.Rect` 移植无纹理 alpha 混合三角带；信仰保持矩形按逐步 float32 与位模式常量计算，接线因 `th10.cfg` +0x30 的隐藏 0x80 位关闭而不绘制。删除旧自机精灵加载器、ganim8 和 `entity/effect` 包。副机／自机子弹暂留旧绘制节点，待 D1e-2。新增 12 个单元测试；th10_05 六章均与改动前逐字节相同，motion 一致帧数 15853／17733／29763／29469／39343／40099，不下降（一面使用已纠正采样的 orig_d.txt）。构建、vet、WASM 通过；不可修改的 Rect 规格锚点期望问题及基线 lint／格式问题解决前，不标记完成。
```

## 8. 需要用户目视确认的项目

运行复刻：`go run ./cmd -debug-stage=1 -debug-power=400`，分别选择两个角色，观察：

- 静止脚本的帧序与节拍；左右起动、直接反向和松键回中。
- 长按 Shift 后判定点的淡入；释放 Shift 后淡出，不是突然删除。
- 开局／复活／灵击无敌期的蓝色闪烁。
- 被弹时的中心特效和三十二个散射火花，以及决死窗口内自机回到脚本 0。
- 信仰保持条在普通配置下不出现。`Player.Draw` 已支持该矩形，但本次没有提供隐藏配置开关。

这里没有启动复刻窗口或原作，不能宣称已经完成 L3 视觉对比。

## 9. 与其他任务的接口

- t3：`FUN_004054b0` 火力槽仍属于 GUI；`Player.bomb` 的扣火力／`RefreshOptions` 后是自机侧调用点，t1 未实现它。
- t3：`FUN_0042b9c0` 得分弹出字及特效 0x189 的调用关系要到阶段 4 核对；当前未修改该接口。
- 共享已存在的 `World.Interrupt(*anm.VM, int16)`、`SpawnTail`、`SetOffset`、`Contains`；没有修改 `anm/world.go` 或架构师句柄规格测试。
- 新增对外接口：`Player.AttachAnm(w *anm.World, env *anm.Env, bullet *anm.File)`；由 `stage.NewContext` 调用，bullet 参数复用 `c.BulletEnv.File`。
- 新增对外接口：`Player.Draw(draw func(*anm.VM), rect func([4]float32, gfx.Color), hold int32, bar bool)`；供节点 0x16 调用，bar 是外部门控组合，目前传 false。
- 新实现接口：`Renderer.Rect(v [4]float32, c gfx.Color)`；原作调用者 `FUN_00426360`，会先 Flush、最后 Invalidate，调用方不需要另做缓存操作。
- 本阶段不改 `chain_test.go` 的 legacy 列表；阶段 2 才按授权将 0x16 移出，0x2b／0x2e 留给 t3／t5。

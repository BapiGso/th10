# t2：符卡逻辑（M6）交接

## 1．摘要

任务工作树：`D:\GoProjects\th10\.tmp\codex\t2\tree`。已完整阅读 rules、真值台账、harness 与指定研究资料。仅修改 t2 工作树；GUI、347／360 与持久化格式不在范围内。

阶段进度：

- 取证：现成 dump 已核对；缺失 dump 已申请，Ghidra 权限阻塞未解除。
- 宣言／衰减／结算：公式、计时、奖励衰减、结算与内存历史已移植；宣言图形仅完成有证据的部分。
- 演出：`bullet.anm` 的 0x1a1／0x1ab、1–6 面 cut-in、环的跟随、宣言牌淡出与结束接口已实现；前三个 World VM、cdbg 初始化／绘制和 Extra cut-in 判断待补 dump。
- 震屏：已移植 `FUN_0043c5ca` 已知采样主体并新增逐值测试，但没有接入运行时；337 的旧方向图案尚未替换，需构造与完整 calc dump。
- 验收：31 个新增顶层测试通过；完整 Go／WASM 检查和六章 trace 已执行，结果见第 4／5 节。六章与基线逐字节相同；完整检查仍有基线问题和旧奖励断言失败。本任务尚未完成，不写 DONE。

当前工作树不是完整 M6：符卡 60 帧后已隐藏 STD，但 cdbg 尚未初始化，画面会缺少符卡背景。不得把“逻辑回归通过”当作“演出已完成”。需要架构师补 dump 后继续。

## 2．改动文件清单

新增：

- `game/spell_history.go`：110 张卡、6 机体与汇总的内存记录表及原作模式门控。
- `game/spell_history_test.go`：记录布局、机体索引、计数饱和与原始名字的测试。
- `scene/stage/spell.go`：奖励公式、宣言、衰减和结束结算。
- `scene/stage/spell_test.go`：奖励、计时、衰减与收取／失败各路径测试。
- `scene/stage/ecl/spell_logic_test.go`：操作数顺序、难度偏移、中断结果和 343 的集成测试。
- `scene/stage/ecl/spell_visual.go`：World 环／cut-in、淡出门控与环跟随。
- `scene/stage/ecl/spell_visual_test.go`：World 脚本、层、偏移、时限参数和演出时序测试。
- `scene/stage/ecl/shake.go`：未接入运行时的原作震屏采样主体。
- `scene/stage/ecl/shake_test.go`：采样序列、整数浮点运算顺序与 RNG 消耗测试。
- `anm/spell_child_test.go`：根对象子链脚本查找及删除后的行为测试。
- `std/stddraw/spell_gate_test.go`：隐藏 STD 时仍更新相机和绘制 World 层的测试。
- `docs/handoff/codex_t2.md`：阶段进度与跨任务交接。

修改：

- `game/state.go`：增加原作回放模式值和内存历史指针，供 t5 接入。
- `scene/stage/bomb.go`：扩充符卡字段与 World／内联 VM 状态，保留灵击失败判断。
- `scene/stage/bullets.go`：被弹时按 `FUN_00426cf0` 清除符卡收取资格。
- `scene/stage/ecl/game_ops.go`：替换脚本奖励，接入宣言／结算并留 GUI 调用注释。
- `scene/stage/ecl/interrupt.go`：LateUpdate 调用完整的已知符卡 calc 部分。
- `scene/stage/ecl/vm.go`：标明旧震屏路径仍需替换，避免误报完成。
- `anm/vm.go`：提供原作子链按脚本号查找。
- `scene/stage/stage.go`：符卡 calc 移到 World calc 之前，计时推进本身不改。
- `std/vm.go`：表达原作背景可见位，不冻结相机脚本。
- `std/stddraw/stddraw.go`：按可见位跳过 STD 槽与 pass，不跳过 World 层 17／18。
- `sim/sim.go`：回放模拟设置 mode 1，禁止历史写入。

删除：暂无。

## 3．移植的函数地址

- `FUN_00409280` → `stage.SpellCard.Declare`、`stage.SpellBonus`、`ecl.VM.declareSpell`、`ecl.VM.spawnSpellEffects`；宣言图形仍有未决部分。
- `FUN_00408d60` → `ecl.VM.updateSpell`、`stage.SpellCard.DecayBonus`、`stage.SpellCard.Tick`；内联 cdbg 的时序接口已留出。
- `FUN_00409c00` → `stage.SpellCard.Finish`、`ecl.VM.endSpell`；GUI 调用只留 t3 注释。
- `FUN_00409e50` → `anm.World.Interrupt`，以 label 1 作用于根和子链。
- `FUN_00409280` 的子链脚本查找 → `anm.VM.Child`，用于写 0x19f／0x1a0 的 `I[2]`。
- `FUN_0043c5ca` 已知主体 → `ecl.sampleShake`；`FUN_0043c8b0`／`FUN_0043c970`／完整回调尚未移植。

## 4．测试

新增 31 个顶层测试：奖励各关卡／原作奇数信仰／项目非十倍信仰／初始奖励上限、宣言操作数弹栈顺序、逐帧整数衰减（含慢速与失败后负值）、计时位模式、收取／失败／耐久／重复结束、内存历史布局与独立饱和计数、hit／bomb／timeout 路径、World 环脚本／层／位置／时限参数、cut-in 脚本与图层、Boss 环平滑序列、淡出迟滞、STD 隐藏门控、子链查找，以及震屏采样序列／FILD、FIDIV、FIADD 的整数精度／零幅度 RNG 消耗。

阶段命令 `go test -count=1 ./anm ./game ./scene/stage ./scene/stage/ecl ./std/stddraw -run '^TestSpell|^TestShake'` 已全部通过。既有测试断言、容差与 golden 保持原样。

已发现旧规格失败：`TestVMSpellBgDeclaresHUD`（`scene/stage/ecl/vm_test.go:190`）期待脚本参数 1000000，原作公式实际为 500000。该夹具 `Stage=0`、`Faith=50000`，故 `(0*3+10)*5000*10=500000`。输出：`BossHUD = name "Test Spell" time 70 bonus 500000`。依据 `FUN_00409280` 的 `+0x3790/+0x3794` 写入；建议架构师把夹具设置为真实关卡后，按公式更新奖励断言，不改时限和名字断言。

完整检查最终一轮（2026-11-07，17:43–17:45 CET）：

| 命令 | 结果 |
|---|---|
| `go build ./...` | 通过，exit 0 |
| `go vet ./...` | 通过，exit 0 |
| `go test -count=1 ./...` | exit 1；两个失败包，见下一段；其余包通过 |
| `golangci-lint run` | exit 1；报告 42 个已有代码处的问题：errcheck 30、ineffassign 1、staticcheck 5、unused 6 |
| `gofmt -l .` | 38 个已有未格式化文件；全部与基线逐字节相同；t2 修改／新增的 Go 文件均已 gofmt |
| 规定的 `GOOS=js` 下 `go list ./...` 过滤 tools | `tools/th10probe` 依赖 windows，list exit 1；收集到的 47 个非 tools 包 build exit 0 |
| 本机 `go list ./...` 过滤 tools，再设 `GOOS=js GOARCH=wasm` 构建 | list exit 0、build exit 0，共 47 个包 |

另一个失败是 `anm/anmdraw.TestRect`：`panic: anmdraw: Rect not implemented`，位置 `anm/anmdraw/rect.go:32`。这是 t1 范围的既有桩和架构师规格测试，不修改实现或断言。完整日志是主树 `.tmp/codex/t2/logs/test.log`；`TestVMSpellBgDeclaresHUD` 的失败与建议见本节前文。Golden 没有重生成。

基线 `gofmt -l` 实际有 39 个文件；t2 因修改 `scene/stage/bullets.go` 而格式化了它，其余 38 个文件保持原样，SHA-256 已逐个核对。没有跨任务格式化。清单与核对结论在 `logs/gofmt.log`、`logs/base_gofmt.log`、`logs/format_baseline.txt`。Lint 的已报位置没有新增功能代码；完整输出在 `logs/lint.log`，不修改其他任务／通用代码来掩盖失败。

可重跑脚本是主树 `.tmp/codex/t2/verify.ps1`；它顺序执行检查并把 Go、临时目录和 lint 缓存设在规则允许的目录。初次检查发现的本任务空 GUI 分支已改成提前返回；最终一轮没有该新增 lint 问题。

## 5．Trace 结果

2026-11-07，六章顺序生成；`cmp -s` 与主树 `.tmp/trace/d1c2/base_cN.txt` 比较，全部 exit 0，连得分、信仰、道具数也逐字节相同。

| 章 | 关卡 | 基线 cmp | 原作 motion |
|---|---|---|---|
| c0 | 1 | 相同 | 规定的 `orig.txt` 在 4915 帧因 MSG 半帧样本分歧；基线对同一文件也在同帧分歧；补验 `orig_d.txt` 为 15853 帧全匹配 |
| c1 | 2 | 相同 | `orig_05_s2.txt`：17733 帧全匹配 |
| c2 | 3 | 相同 | `orig_05_s3.txt`：29763 帧全匹配 |
| c3 | 4 | 相同 | `orig_05_s4.txt`：29469 帧全匹配 |
| c4 | 5 | 相同 | `orig_05_s5.txt`：39343 帧全匹配 |
| c5 | 6 | 相同 | `orig_05_s6.txt`：40099 帧全匹配 |

一面补验没有改原作 trace、过滤帧、改容差或跳过 MSG。重新生成了 c0，再次 cmp 为 0，再对现存 `orig_d.txt` 得到 `MATCH: 15853 frames compared`。`orig.txt` 的首个问题是 `{clock 34, at 34, op 16}` 对 `{clock 35, at 38, op 12}`；真值台账 2026-10-31 已记载 probe 的 GDI 半帧采样问题。证据在 `logs/c0_baseline_original_motion.log` 和 `logs/c0_original_d_motion.log`。

一致帧数没有下降。用户说明 th10_05 不射击、符卡超时而没有收取，因此结算没有给这个回放新增分数；本次实际没有得分与基线的分歧，不能编造预期中的得分变化。收取奖励入账由新增单测覆盖。旧原作 trace 的 score／faith／state 地址问题仍存在，不将其用作完整数值真值。

没有新增运行时震屏 RNG 消耗，因为采样器尚未接入；有证据的宣言 World VM 使用 `FUN_00449870` 的备用 RNG 标志，各关 cut-in／环单测也校核了主 RNG 消耗为 0。将来接入 `FUN_0043c5ca` 时，每次采样须消耗 4 个主 RNG step（两个 `Dup()%3`），包括幅度为零的情况，届时必须重跑六章。

生成的六个大 trace 和补验 trace 均已在比较后删除，`trace/` 为空；只保留日志。顺序运行脚本是 `.tmp/codex/t2/trace.ps1`，结论在 `logs/trace_results.txt`。

## 6．待决与推测

Ghidra 首次启动被沙盒拒绝写入用户配置目录；没有重试或绕过。完整命令与用途见主树 `.tmp/codex/t2/dump_requests.md`。

待确认：宣言前三个 World VM 的文件参数与 cdbg 脚本号、宣言音效的寄存器参数；Extra 分支读取的 `DAT_00474c84`；震屏构造／计时／回调；练习模式到 `[DAT_00477838+0x10]` 的映射。未用符卡 id 代替 MSG 状态，也未用 `SingleStage`／`DebugMode` 替代原作历史门控。模式控制器归 t5，当前普通练习入口还未设置新字段，不能声称练习模式接线已完成。

已查明 cdbg 是 `DAT_004776f4+0x10/+0x3bc` 的两个内联 VM，不在效果世界；由 `FUN_00404f30` 启动、`FUN_00408d60` 步进、`FUN_00409230` 在 draw 0x0e 绘制。当前只留出状态和步进时序，尚未按未知脚本号启动，也没有接 draw 0x0e。切勿把它们错误改成 World VM／calc 0x1a。

前三个自由 VM（脚本 1／0x48／2，句柄 `+0x768/+0x76c/+0x770`）确实由宣言直接挂进 World，属于 t2；其文件寄存器未见于现成 C，所以暂未生成，不能仅凭脚本号猜文件。外部原始 `front.anm` 已作静态检查并保存在任务草稿目录，未作为无用新资源嵌入工作树。

代码没有用猜测的方向／计时器代替震屏取证。旧 `ShakeOffset` 留有明确 TODO；`sampleShake` 的测试只证明已知采样主体，不证明构造或寿命。`43c870` 的回调地址来自现成 `faith/_writes.txt` 的 `0x43ca7b` 写入，已加入补 dump 请求。

奖励上限细节：原作当前奖励 `+0x3790` 接收未钳制的 int32 乘积，仅衰减基数 `+0x3794` 钳到 99999999；移植保留这一差别。正常最大关卡与信仰只能得到 30999690，不会触发上限分支。奖励公式和衰减均为整数；环跟随的 0.05 与淡出阈值按 PE 读出的 float32 位模式实现。

现成 `FUN_00409c00` ASM 没有清弹、清激光或生成道具调用；343 原有“不消弹”的逻辑须保留，不能仅按任务描述添加清弹。

## 7．建议的 `docs/fidelity.md` 改动

以下是阶段性交接用的新状态行，不能改写成“完整 M6 已完成”。不直接修改真值台账。

```markdown
| 符卡状态（灵击读取部分） | 位精确（已移植的奖励、衰减、计时与收取逻辑）；宣言演出、练习模式接线待补取证 | DAT_004776f4（stage.SpellCard）：FUN_00409280 按 int32 顺序计算 (stage*3+10)*(Faith/10)*10，+0x3794 上限 99999999；FUN_00408d60 在道具之后、World 之前，pre-tick 计时 >=300 且非耐久卡时按整数公式衰减并清个位，计时原样推进；FUN_00409c00 只凭收取 bit 1 用 AddPoints 入账、更新内存历史并结束 World 环／宣言牌，不清敌弹／激光、不生成道具；FUN_00405500／00426cf0 计时 >59 时失去收取，FUN_004127a0 非耐久超时同样处理；内存历史按 mode !=1 门控，持久化归 t5 | 新增 Spell／Shake 测试；六章基线 cmp 相同；演出及模式接线仍待 dump |
| 符卡演出（M6，t2） | 部分移植；未完成 | FUN_00409280 的 bullet.anm 0x1a1／0x1ab 与 1–6 面 stgenm cut-in 已生成到 World；两个 cdbg 为 helper 的内联 VM、calc 0x16／draw 0x0e，不是 World；前三个 World VM、cdbg 初始化和 Extra 分支待宣言 ASM；GUI 名字／奖励／历史数字归 t3 | World 脚本／层／偏移／时限参数、环跟随和淡出门控测试；待用户目视 |
| ECL 337 震屏 | 未完成（原作采样主体已移植，旧运行时方向图案仍在） | FUN_0043c8b0／0043c970／完整 calc 待 dump；已知 FUN_0043c5ca 每轴 Dup()%3 选择 0／正／负幅度，一次 4 个主 RNG step；写 Field.Shift，再由 ANM emit 做绘制平移，viewport 不移动 | sampleShake 序列／x87 整数运算顺序／零幅度 RNG 单测；生命周期与运行时接线待取证 |
```

更新记录建议：

```markdown
- 2026-11-07 — t2（M6）阶段性交接：移植符卡奖励公式、整数衰减、计时／收取结算、被弹失败和内存历史；ECL arg 2 仍照原顺序求值但不作为奖励。生成 bullet.anm 的 0x1a1／0x1ab 与 1–6 面 cut-in，按 FUN_00408d60 保持环跟随和淡出时序，符卡 calc 校正到 World 之前。343 不消弹、不产道具，耐久卡不衰减并保留超时收取；GUI 和持久化格式不实现。31 个新增测试通过；build、vet、47 个非 tools 包 WASM 构建通过；全量 test 仍有 Rect 桩及旧奖励断言失败，lint 报 42 个已有代码问题，gofmt 余下 38 个文件与基线相同。六章 cmp 逐字节相同，motion 15853／17733／29763／29469／39343／40099 不下降，一面的 orig.txt 有旧 GDI 半帧样本，用未修改的 orig_d.txt 补验。Ghidra 被沙盒拒绝，前三个 World VM、cdbg、Extra 分支及震屏构造／回调未完成，补 dump 请求见 .tmp/codex/t2/dump_requests.md；不写 DONE。Golden 未重生成。
```

## 8．需要用户目视确认的项目

进入工作树，按 rules 第 2 节设置 Go／临时目录。已有命令参数为：

```powershell
go run ./cmd -debug-stage 1 -debug-diff easy -debug-power 500
go run ./cmd -debug-stage 6 -debug-diff easy -debug-power 500
go run ./cmd -debug-stage extra -debug-diff extra -debug-power 500
```

- Cut-in：先看 1 面中 Boss 静叶（id <2）和穰子（id >=2）立绘区别，再把 `-debug-stage` 换成 2–6，确认滑入／上移／淡出、图层和裁切；调试入口当前固定灵梦 A。
- 环与宣言闪光：符卡发动时看 0x1a1 的两层环是否在 Boss 位置，移动时的滞后是否正确，结束时是否整组删除；0x1ab 的自由闪光不应被再次加 (224,16)。
- 宣言牌：当前前三个 World VM 尚未生成，不能完成这项目视确认。补完后观察自机 y <96 且符卡计时达到 120 时 label 3 淡出、y >128 时 label 2 恢复，结束 label 1；数字与名字由 t3 完成后一起确认。
- 符卡背景：当前缺 cdbg 初始化／绘制，不能确认最终效果。补完后看前 60 帧 STD 与 cdbg 的交接、结束时 STD 恢复、cdbg 的透明度／纹理移动；不得把静态空背景当作完成。
- 震屏：当前仍是旧图案，不要求用户认可它。补 dump 并替换后，在 BossDead 337(30,12,0) 处确认离散 0／正／负偏移和衰减、场地裁切不移动、14 层以后 GUI 不受震屏影响。

## 9．与其他任务的接口

### t3：GUI 调用与字段

代码在调用点留 `// t3: FUN_xxxxxxxx (GUI) — ...`，没有虚构 GUI 实现或新空接口。t3 接入时须按原作顺序补齐以下部分。

| 原作地址 | 参数／字段 | 时机与现有交接点 |
|---|---|---|
| `FUN_0043e710` | GUI `+0x9ec8` 的 ANM 文件；helper `+0x778+i*0x3ac` 的 8 个奖励 VM，`+0x24d8+j*0x3ac` 的 5 个历史 VM；脚本／位置寄存器待宣言 ASM | `ecl.VM.declareSpell` 注释处；每次宣言 |
| `FUN_00447ae0` | C 显示 `(iVar5, 0xffffff, param_3)`，`param_3` 是原始 Shift-JIS 名字；寄存器与 `iVar5` 含义待 dump，不编造签名 | 宣言的 `+0x76c` 名牌句柄查找之后；白色名字纹理生成 |
| `FUN_00449470`／直接 `VM+0x304` 写入 | label 3／2；三个 World 宣言牌由 t2 处理，8+5 个内联数字 VM 由 t3 处理 | `ecl.VM.updateSpell` 的淡出分支；tick 后计时 >=120、自机 y <96 或 >128 |
| `FUN_0043e5a0` | 奖励数字 sprite `0x1e+digit`，8 位按 10000000 开始分解；失败时仅 helper `+0xed0` 改 sprite 0x29 | `ecl.VM.updateSpell`；计时推进之后 |
| `FUN_0043ee30` | 收取中步进 8 个奖励 VM；失败步进 `+0xed0`；随后步进 5 个历史 VM | `ecl.VM.updateSpell`；主卡仍 Active 时每帧 |
| `FUN_0043e5a0` | 机体对应的 `Captures`／`Attempts` 分别最多显示 99；十位／个位均为 sprite `0x1e+digit`；分隔符 VM 不改 sprite | 同上；读 `GameState.SpellCounts(card.ID)`，不要修改原始计数 |
| `FUN_00405500`、`FUN_00426cf0`、`FUN_004127a0` 的 GUI 部分 | 清 7 个奖励 VM flag bit 1：helper `+0xad4/+0xe80/+0x15d8/+0x1984/+0x1d30/+0x20dc/+0x2488` | 灵击／被弹（timer >59），以及非耐久卡超时（timer >59）；数值／Capture 已由 t2 写入 |
| `FUN_004172e0` | EAX=0，栈参数 GUI 对象、当前 `Bonus` | `ecl.VM.endSpell` 收取成功时；显示 Get Spell Card Bonus |
| `FUN_004492a0`、`FUN_00448d00` | 删除并清 GUI `+0x9e14` 旧句柄，再生成 `front.anm` script 0x48、初始 layer 15，写回新句柄 | `ecl.VM.endSpell` 失败路径；这个 GUI 结果 VM 不由 t2 生成 |
| `DAT_00474c84` | 最后 MSG id+1；Extra 宣言 `<0x18` 选 cut-in 0x28，否则 0x1e；两个 cdbg 的选择也有分支 | `ecl.VM.spawnSpellEffects` 的 case 7 注释；需 t3 提供正确原作 MSG 状态，待模式 dump 复核 |

`SpellCard` 的 `Name` 保留 Shift-JIS，`vm.spell.name` 是现有 HUD 使用的 UTF-8；t3 不要把 UTF-8 字节写回历史或拿它代替原作纹理参数。`Placards [3]*anm.VM` 是 t2 应生成的三个 World 句柄，当前待宣言 ASM；`Faded` 对应 bit 2，宣言不清它。`Bonus`、`InitialBonus`、`Duration`、`Timer`、`Capture`／`Active`／`Survival` 可从 `Context.Spell` 读取。

### t5：内存历史布局与模式

`GameState.SpellHistory *game.SpellHistory` 可由 t5 绑定到加载后的公共历史对象，跨局复用；t2 不创建／读取／写入任何存档。它包含 `Shots [6][110]SpellRecord` 和 `Total [110]SpellRecord`，机体索引为 `character*3+shot`，难度由原作 card id 分组，不另加难度维度。

| 原作记录相对偏移 | Go 字段 | t2 写入 |
|---|---|---|
| `+0x00`，128 字节 | `Name [0x80]byte` | 宣言复制原始 Shift-JIS 到 NUL，旧名字 NUL 后的尾部保留 |
| `+0x80`，int32 | `Captures` | 成功结束 +1，独立饱和于 99999 |
| `+0x84`，int32 | `Attempts` | 宣言 +1，独立饱和于 99999 |
| `+0x88/+0x8c`，两个 dword | `Reserved [2]uint32` | 不写入，不清零已有内容 |

单条记录大小 `0x90`。单机体记录在 `DAT_0047783c + 0x5a4 + (character*3+shot)*0x437c + id*0x90`，汇总记录在 `DAT_0047783c + 0x19a8c + id*0x90`。Go 表省略数组之间的其他存档字段，不能直接把整个 Go 表内存当作完整原作存档格式。

`GameState.ReplayMode int32` 表达 `[DAT_00477838+0x10]`；两处历史更新独立检查 `mode !=1`，不是只允许 mode 0。sim 已设为 1。练习／正常游戏／录制／播放的赋值由模式控制器负责，练习映射待 dump，不改动 t5 的控制器。

可调用接口：`RecordSpellDeclaration(id int32, rawName string)`、`RecordSpellCapture(id int32)`、`SpellCounts(id int32) (captures, attempts int32)`，均为 `*game.GameState` 方法；前两者对应宣言／成功结束，后者供 GUI 读取原作机体计数。

### t2 实现／使用的其他接口

- `stage.SpellBonus(stage int, faith int64) (bonus, initial int32)`：奖励公式与两字段区别。
- `(*stage.SpellCard).Declare(state *game.GameState, id int32, rawName string, duration int32)`：ECL 342／348／357–359 调用；arg 0、arg 2、arg 1 的求值顺序已保留，357／358／359 难度偏移核对通过。
- `(*stage.SpellCard).Finish(ctx *stage.Context) bool`：ECL 343 使用；恢复 STD、向三个宣言牌发 label 1、清 Active、删除环／子链、按 bit 1 入账／记收取；返回是否收取，不清弹／激光／道具。
- `(*stage.SpellCard).Bombed()`：原作 `FUN_00405500` 的失败数值部分；t1 灵击和被弹演出应继续沿用。自机被弹的 `Context.hitPlayer` 已调用它，bomb 路径已有调用。
- `(*anm.VM).Child(id int) *anm.VM`：只在本根的子链中找原作脚本号，删除后返回 nil；宣言将 0x19f／0x1a0 的 `I[2]` 写为 Duration。
- `std.VM.BackgroundVisible bool`：原作 STD `+0x2a18` bit 0；卡计时到 60 清、卡结束置；STD 相机脚本继续运行，draw 7 跳过槽与 pass 0–7，draw 0x0a 只跳过 pass 8–11，不跳过 World 层 17／18。
- 现有 `anm.World.SpawnTail`／`Place`／`Interrupt`／`Delete` 直接复用；`FUN_0043dc90` 的成功收取 SE 0x2d 通过 `Context.PlaySEID` 调用，宣言 SE 参数待 ASM。原作 `FUN_0041bb00` 没有出现在 343 调用链中，不增加额外调用。

震屏落点已查明：`DAT_00491e64/68` 是 `Field.Shift`，不是 viewport 的 x／y，也不改变逻辑坐标；`Supervisor.Flat/Perspective` 把它复制到 `ANM.Shift`，`FUN_00442670` 的 `emit` 在取整前平移 quad 顶点。draw 0x24 的 `Unshake` 清 Field／ANM shift，0x32 再清 Field。现有各关 `ShakeOffset()` → `Stage.Draw` → `Field.Shift` 路径保留，构造与 calc 补完后替换偏移来源；不另加 viewport 抖动或移动自机／敌弹坐标。

# t9：文档整合（fidelity 总账与研究文档）

更新时间：2026-11-08。工作树：`D:\GoProjects\th10\.tmp\codex\t9\tree`。

## 1．摘要

- 阶段 1：已完整阅读共用规则、总账、harness 与第一轮 t1–t5 交付文档；已核对合并后的代码、规格断言和现成补充 dump 清单。
- 阶段 2：已整合 25 条总账状态行，保留 2026-11-07 的八条任务记录，新增一条 2026-11-08“第一轮合并”记录；已更新两份真值文档、16 份研究文档与 harness，建立四类待决清单。
- 阶段 3：已完成文档结构／引用／等级／日期／摘录／空白检查和 668 个非 docs 文件 SHA-256 核对；最终交付同步两份，任务目录写入 DONE。
- 只修改工作树内的 `docs/`。不修改代码、测试、资源、依赖、golden 或原作数据，不运行原作、probe 或 Ghidra。

主要消歧：

- 以代码确认符卡奖励／衰减／收取与内存历史已实现；GUI fallback 还没有读取真实计数，宣言／结算／失败的精确调用留 t7，不能把逻辑历史与显示接线混为一谈。
- `ReplayMode` 保留 t5 的 bool，历史写入用 `!ReplayMode`；备用 RNG 当前仍为每 GameState 种子 0，只把应用级启动一次播种写为已定策略、t8 待实现。
- t5 的 mode 13 重开已按 Continued 直接设 rank −512，不能仅因 `ResetRank(true)` 无调用就保留“等 t5 接线”；完整原模式字段仍由 t8 核对。
- 0x23／0x25 已画，0x2b／0x2e 非 legacy，0x2e 是 `DAT_00477830` 菜单，唯一 legacy 节点仍为 0x16；本体与 Rect 已接，但副机／子弹／外部门控未齐，cdbg 0x0e 仍未接。
- 保留 t4 已落地的 R8／R10／RRE 矩形、回调、门控与擦弹勘误；本次更新过时状态／代码入口，未重复导出或声称新增移植。

## 2．改动文件清单

新增：

- `docs/handoff/codex_t9.md`：记录文档整合进度与交付边界。
- `docs/open_items.md`：按四类汇总 32 项取证／验证／适配事项，标明 t6／t7／t8 与已定论项目。

修改：

- `docs/fidelity.md`：合并五任务建议、消除重叠与过时状态、追加各任务历史和七项合并决定。
- `docs/player_truth.md`：更新本体／灵击／道具／信仰／命中得分与回放状态边界，保留未完成射击项。
- `docs/ecl_truth.md`：改为简体中文，纠正旧时钟／特殊变量／指令／近似清单与符卡／GUI 状态。
- `docs/harness.md`：纠正回放保存函数、变长 USER、重复保存计数及 x87 24 位精度说明。
- `docs/research/ecl_gaps.md`：把已实现的取整／缓动／弹幕／RNG／调度等旧缺口替换为当前状态。
- `docs/research/faith.md`：更新 t2／t3 calc 顺序和 t5 fresh-run 重开／完整模式边界。
- `docs/research/m1_interrupts.md`：区分已实现更新交错／创建／异步／伤害／掉落与仍缺本体碰撞。
- `docs/research/m2_bullets.md`：更新 t4 声像／闲置 +6／备用 RNG，以及绘制、种类 8 与 calc 顺序。
- `docs/research/m2_motion.md`：纠正旧一面可比范围、统一伤害／掉落／rank／激光缺口和 STD op 9／14 状态。
- `docs/research/m4_modes.md`：保留原 17 文件统计但注明当前 28 文件范围，更新模式 9 调用图与新增 VM 接入。
- `docs/research/m4_r6.md`：更新 GUI 数组角色、ascii 箭头／front 数字归属及非 legacy 状态，修正同地址 dump 的目录引用。
- `docs/research/m4_r8.md`：更新敌机空桩、自机部分迁移、实体直接 ANM、0x23／0x25／0x2b／0x2e 与已解取证项。
- `docs/research/m4_r9.md`：更新 D1c-2 槽／方向／闪烁／文件／删除与 t1 被弹，纠正 draw 节点和转义表格位运算符。
- `docs/research/m4_r10.md`：更新 t1 本体／蓝闪／判定点／被弹／矩形，保留逐步 float32 和 t6 剩余边界。
- `docs/research/m4_r11.md`：更新 t5 cfg／Menu.State、t1 Rect 与四门控未接，保留原门控定义。
- `docs/research/m4_render.md`：更新 D1 节点／阶段状态，明确原设计分工与尚未实现的截图 CLI。
- `docs/research/m4_render_re.md`：补齐节点／GUI／cfg 状态、表格转义与同地址 render3／render4 证据路径。
- `docs/research/presentation_gaps.md`：纠正默认 STD／gfx／ANM、实体／GUI 来源与旧背景回退，保留 cdbg／L3 等缺口。
- `docs/research/replays.md`：纠正 `FUN_0042a450`、Score／Faith／Hold 字段、重复保存计数与实时不透明状态。
- `docs/research/systems_gaps.md`：更新 MSG／HUD／符卡／灵击／计分与已删除特效池，原旧实现明确标为历史。

删除：无。

`m4_r7.md`、`m4_s2b.md`、`m4_s2c.md`、`m4_s3.md`、`th10c_dat.md` 的既有取证／勘误／数据事实仍有效，未改正文；不扩展原静态统计或宣称所有研究未决已关闭。
主树仅同步本交付的 `docs/handoff/codex_t9.md` 副本；文档核对日志与初始哈希清单在 `.tmp/codex/t9/`，不属于合并文件。

## 3．移植的函数地址

无。

## 4．测试

未新增或修改测试。按 t9 任务说明，未改代码，rules 第 6 节的构建、测试、WASM、gofmt、smoke 与 trace 不必运行；本任务不复跑，也不把第一轮合并验收冒充本次运行结果。`golangci-lint` 按 rules 第 6．5 节不是验收项，未运行。

文档核对与最终副本检查已通过：

- 初始 698 文件清单与交付前逐文件 SHA-256 比较：668 个非 docs 文件全部保持，源码／测试／资源／依赖无改动；工作树为 20 个文档修改、2 个文档新增，无删除。
- 静态检查覆盖 22 份改动文档、126 个表格块、25 条总账新状态、四类待决与九节交付；新增结构／等级／重复／日期／摘录问题为 0。旧取证表的 11 处未转义位运算符已补 Markdown 转义，二进制事实不变。
- 仓库路径、数据命名模板、Go 限定符号分别核对；已删除的 HUD／effect 路径只作明确历史说明，旧 render 下 26 处引用改到同地址现成 render3／render4，不新增 dump 请求。
- `git diff --no-index --check` 按 Windows CRLF 使用单次 `cr-at-eol`，输出为空；退出码 1 表示两目录有文档差异，不是空白错误，没有写 Git 配置。初次不识别 CRLF 的行尾告警已经区分为 CR，不把它当代码失败或实际尾随空格。
- 核对日志：`.tmp/codex/t9/docs_checks.txt`、`whitespace_check.txt`、`docs_changes.json`、`inventory_before.json`；最终完整状态／任务更新摘录从总账生成，不手写另一套冲突状态。

## 5．Trace 结果

未改代码，不适用。

## 6．待决与推测

第一轮待决已集中写入 `docs/open_items.md`：等 dump、等用户目视、等用户录回放或采 trace、推测待证四类；已供 dump 的请求明确写为“证据已到、实现待接”，不再重复申请。
清单当前在 `D:\GoProjects\th10\.tmp\codex\t9\tree\docs\open_items.md`；主树只收到本交付副本，尚未合并其他文档。

- t6：副机／自机子弹、其余五机体、灵击余项、现场 capture／restore 与信仰矩形四门控。
- t7：前三个宣言 VM、cdbg／0x0e、Extra／344、原震屏、真实历史与宣言／结果 GUI 接口。
- t8：score.dat／跨局历史、原模式／初始化字段、高分／extend／通关与应用级备用 RNG。
- 未分派或待用户：原暂停 flag、host slow-rate、直接道具／激光／有效数值 trace、字体／捕获像素、音效设备试听、lint 合并后余额。

t9 没有新增函数、常量推测或 dump 请求；原研究中仍有效的其他未决按原文保留，不提前把第二轮工作写成通过。

## 7．建议的 `docs/fidelity.md` 改动

t9 任务说明明确授权直接整合总账，优先于通用交付模板的“不直接修改”要求。以下摘录直接从当前总账同步，完整状态行／日期记录与 `docs/fidelity.md` 一致。
总账当前在 `D:\GoProjects\th10\.tmp\codex\t9\tree\docs\fidelity.md`，不是尚待架构师合并的主树旧总账。

已应用的完整状态行：

```markdown
| ECL 敌机/移动 256-368 | 有界漂移（运动核心与 349–354 已移植；337、344 的运行时接线未做） | `FUN_0040e770`；t4 按 `0x4118bb–0x411b6b` 逐条实现 349–354：349／352 在 rank < 512 时读 p0，350／353 按 −400／−200／200／600 选 p0–p4，保留参数索引缺陷；351／354 按原求值顺序做 float32／int32 插值，354 格式为 `SSS`；七关均未使用这六条。346（401 禁射半径）已移植；347／360 由 t3 写 GUI 比例／ARGB 与星数；337、344 见下行及待决清单 | `TestRankOpcodeFormats`、`TestRankOpcodesUnusedInStages`、`TestRankSelect349And352`、`TestRankSelect350And353`、`TestRankLerp351`、`TestRankLerp354`、`TestRankOpcodeSourceEvaluation`、`TestVMAllStagesStableAndFire` |
| ZUN RNG | 有界漂移（生成器与回放取种已移植；备用 RNG 生命周期待 t8） | `FUN_0044bb90` 常量 PE 直读；主种子取章节头 +0x02（`GameState.RNGSeed`，无回放时仍为 `0x1234`），ECL／子弹 ANM 共用 `ctx.RNG`。t4 查明备用 `DAT_004918a8` 由 `FUN_004201b0` 在 `0x4201e1` 写入启动毫秒计时低 16 位，跨关／回放不重置、不录制；当前 `GameState.AlternateRNG()` 每个 GameState 从 0 开始，局内跨关共享 state／count，不抽主 RNG 派生。架构师已决定改为应用级启动时取墙钟毫秒低 16 位、一次播种、跨局不重置，sim／测试固定 0；第二轮 t8 处理中，尚未生效 | `TestAlternateRNGDeterministicSeed`、`TestAlternateRNGPersistsAcrossContexts`、`TestAlternateRNGDoesNotChangeReplayTrace`（两备用种子的六章完整 trace 与主 RNG state／count 相同） |
| 动态 rank | 位精确（正常／回放规则已取证；模式入口待 t8） | `DAT_00474c98`，`GameState.Rank` 跨关保留、章节头 +0x20 直接恢复。t4 核对全部写入点与五个 `AddRank` 物理调用点；`FUN_00425730` 的 `0x426256–0x4262b4`：Tick 后状态时钟 +0x478 对 60 取余为 0、GUI+0x9eb8 = 0、敌机管理器 +0x60 ≠ 0 时 +1，不要求 Cur ≠ Prev，慢帧停在倍数时重复增长，钳到 ±0x400。道具、死亡 −0x400、灵击 −0x80 已接；新局 `ResetRank(false)` 为 0，`FUN_00417870` 的 `0x4179e2` 对 `DAT_00474ca0 & 8` 分支为 −512，`ResetRank(true)` 无运行时调用，但 t5 的 `RestartStage` 在 mode 13 按 Continued（0x8）直接设 −512；普通练习首次初始化与完整模式字段仍待 t8 | `TestResetRank`、`TestAddRankClamp`、`TestDynamicRankClockAndGates`、`TestDynamicRankHeldClockAndAllStates`、golden、六章基线逐字节比较 |
| ECL 337 震屏 | 未做（原作运行时未接；采样主体已移植） | t2 的 `ecl.sampleShake` 对应 `FUN_0043c5ca` 已知主体，每轴 `Dup()%3` 选 0／正／负幅度，一次消耗四个主 RNG step，零幅度也消耗；当前 `ShakeOffset` 仍用旧方向图案。构造 `FUN_0043c8b0`／`FUN_0043c970`、回调 `0x43c870` 与 Field.Shift 接线由第二轮 t7 处理，现成 dump 已补齐；不移动逻辑坐标或 viewport | `scene/stage/ecl/shake_test.go`；生命周期、运行时 RNG 与视觉仍待验收 |
| 音效表 | 位精确（表与队列）；PCM 增益等效，听音待确认 | t4：`0x4749c8` 的 47 项与 wav 表 `0x474b40`；`FUN_0043dd10` 按 pan = trunc(f32(x·0x40a6aaab)) 排队，同 id 每帧最多 128 个 pan 请求、整数平均（`0x43e28d`），每帧最多 12 个不同 id，每 id 重启独立 voice。振幅 10^(音量/2000)，正 pan 衰减左声道、负 pan 衰减右声道；跨平台 `audio/pan.go` 包装 PCM，`Game.Update` 在场景 calc 后 flush（`FUN_00439390`，`0x43943b`）。+6 只复制到 manager+0x408，本版无运行时读者，保留闲置属性，不猜成冷却／优先级；自机 SHT 发射音效尚待 t6 | `TestSEPan`、`TestSEPanGains`、`TestPanStreamSamples`、`TestPanStreamSeek`、`TestSEQueuePanAndAux`、`TestSEQueueLimits`、`TestPositionalSoundWiring`、`TestPlaySEPanning`（需用户启用） |
| 敌机逐个更新顺序 | 位精确（移动、ECL 与伤害；六章运动范围无回归） | `FUN_0040d750`／`FUN_0040dc80`／`FUN_0040cfb0`：按链表逐个“移动→ECL→清 0x2000→伤害（flags&0x11 为 0 时）→无条件中断检查→无敌与阶段计时”，0x400 标志与“先取 next”时序照搬，尾部新建对象少更新一帧；新敌人先首次更新再入链表，main 跑在隐藏敌人上。D1c-2 已在 `ecl/visual.go` 接入方向切换、蓝色受击闪烁与受击音效，t4 接 x 声像；未移植的是本体碰撞，不再把受击 ANM 列为缺口 | `interrupt_test.go`、`scene/stage/ecl/visual_test.go`、golden、六章基线逐字节比较 |
| 道具管理器 | 位精确（移植；一至五面运动范围无回归，道具数待 probe 采样） | `DAT_00477818`（`entity/item`）：150 个掉落槽 + 2048 个消弹槽，步长 0x3f0。生成 `FUN_0041bb00`：消弹道具（种类 8）取游标槽（占用则不生成，游标照走），延迟按上次更新以来的生成数取槽号 &3 / &7+4 / &15+8 / &31+16，进入状态 5；其余种类取首个空槽，x 钳到 [−192, 192]，火力 > 99 时 1/4→9、10/11→5；10/11 的外观同 1/4；`bullet.anm` 脚本为种类 + 0x176。更新 `FUN_0041afd0`（按槽序）：状态 5 倒数结束转 2 并启动脚本；状态 1 在自机 y < 128（且非状态 2/4）或灵击中转 3 追踪（速度灵梦 5、魔理沙 7，`0x476fa0`），否则移动、重力 0.03、开始下落后横向速度清零、竖直速度上限 2、y > 472 消失；状态 2 上升，停止上升后转 3，y > 472 消失并 rank −4；状态 3/4 追踪，速度每帧 +0.2 直到 12，决死窗口中转回 1、速度清零。自机不在状态 2 时：在拾取盒内拾取（`FUN_00425730` 尾部，±10/±12），状态 1/2 进入吸附盒（低速 ±50/±59，否则 ±20/±24）转 4，速度为追踪速度 ×0.33333334。收集（`FUN_0041be80(n)` 给信仰加 n/10，`FUN_00409d90(n)` 给得分加 n/10）：1/10 火力 +1，升级时 rank +12、刷新副机 `FUN_00426f70`、音效 0x1d；4/11 火力 +20，升级时 rank +24；2 得点，自机 y < 144 时得 信仰×10、rank +8，否则 信仰×10/2 − ftol((y−144)·(1/304)·(信仰·0.5 − 5000))、个位清零、rank +1；3 按难度 n = 5000/5000/8000/10000；5 得 信仰×10、rank +8；7 残机 +1（`FUN_004188a0`，音效 0x2c，上限 9）、rank +0x100；8 n = 10、得分 +10；9 n = 100；拾取音效 0x14。达到满火力的那次更新末尾 `FUN_0041ba50` 把场上的 1/4 号道具换成以 −π/2、2.2 抛起的 9 号道具。绘制 `FUN_0041b8e0`：场内用精灵 外观 + 0x157（原作与 +0x158 比较，永不相等，所以每帧重设 alpha 0xff）；y < 8 时画在 y 24 的箭头（外观 + 0x160），alpha = (char)ftol((y−8)·0.03125·255)，为负数时取低字节。t3 已移植独立 `DAT_00477840` 得分弹出字（`FUN_0042b9c0`，calc 0x0f／draw 0x23）与 GUI Power 数字（`FUN_004054b0`）；第一轮合并后的完整 smoke 已通过，像素效果仍待用户。未移植：特效 0x189，第二轮 t6 处理中 | `entity/item/item_test.go`、`scene/stage/power_test.go`、`scene/stage/score_popup_test.go`、`entity/item/popup_test.go` |
| 死亡与复活 | 位精确（逻辑已移植；六章运动范围无回归，状态字段待新 trace） | `FUN_004269d0`（提交死亡：残机 −1；存储信仰超过 5000 的部分减去三分之一、不低于 5000；rank −0x400 后钳到 ±0x400；state 2；无敌 180；副机数清零）；`FUN_00425730` state 2 timer ≥ 30 且残机 ≥ 0 时进入 state 0，游戏速度 `DAT_00476f78` 复位为 1，留下伤害圈（`FUN_00427b50`），移到 (0, 480)，无敌 280；state 0（60 帧）Y 480→400（`IY = 48000 − t·8000/60`，t 为 state 计时），t < 30 时以死亡点为圆心清弹（半径 64 + t·17.066668，`FUN_00408100` 在 r 处做免疫检查、r/4 处不检查，`FUN_0041c800` 清激光）；t ≥ 30 及 state 1 前 30 帧清全部非免疫弹与激光且不出道具；t > 59 → state 1 | `player_test.go`（`TestDeathAndRespawn`、`TestHittable`、`TestAreaDamage`）；六章基线逐字节比较，原状态列仍待有效新采样 |
| 灵击生命周期 | 位精确（逻辑，六面运动差分）；剩余演出待 t6／t7 | `DAT_004776ec`（`scene/stage/bomb.go`）。发动 `FUN_00405860`：state 1，计时归零，半径 32，位置取自机，`bullet.anm` 脚本灵梦 0x190、魔理沙 0x197、重型符卡中 0x1b9（符卡进行中且 id 0x5d–0x60，或 id 0x6d，符卡结束后也算）；SE 0x26；`FUN_00405500`；rank −0x80；符卡计时 ≥ 60 时为强灵击（+0x44）。每帧 `FUN_00405750`（calc 0x11，紧接自机）：ANM 对象不在世界里（`FUN_004491c0`）则结束；否则上升（灵梦 1、魔理沙 1.3、重型符卡进行中 0.5），`FUN_004492f0` 把对象移到 位置 + (224, 16)，半径取对象的 y 缩放，然后 `FUN_00405ac0`：先 `FUN_00408100`（出道具 = 非符卡中，免疫检查 = 非重型符卡），再 `FUN_0041c800`；计时随游戏速度推进。伤害 `FUN_004059f0`：圈内符卡中强灵击 38、符卡中 3、有 Boss 38、否则 5；圈外仅当非符卡、有 Boss、非强灵击时 20（魔理沙 16），或符卡中强灵击 26（魔理沙 23），其余 0。自机侧（`FUN_00425730`）：state 1 灵击无敌 270、Power −100、信仰 `FUN_00412e70(3000)`（−300、下限 5000，本项目 ×10）；决死灵击不扣信仰；灵击进行中不能再次发动。ECL 366：0x100000 标志 + 灵击时的方向脚本（+0x1448），灵击进行中置 0x200001（不受伤）并换脚本，结束后恢复（`FUN_0040dc80`）。D1c-2 已实际重启 366 槽 VM，不再只记脚本号；t3 已实现 GUI Power 数字（`FUN_004054b0`）与独立弹出字（`FUN_0042b9c0`），具体灵击调用及特效 0x189 待第二轮 t6，符卡失败 GUI 接线待 t7 | `bomb_test.go`（`TestBombLifetime`、`TestBombDamage`、`TestBombDuringCard`）、`player_test.go`（`TestBombConsumesPower`） |
| 符卡状态（奖励、计时、收取） | 位精确（已移植的数值与内存历史）；模式和 GUI 接线待 t7／t8 | t2：`DAT_004776f4`（`stage.SpellCard`）；`FUN_00409280` 按 int32 顺序计算 `(stage*3+10)*(GameState.Faith/10)*10`，当前奖励 +0x3790 不钳制，仅衰减基数 +0x3794 钳到 99999999；ECL arg 2 照原顺序求值但不作为奖励。`FUN_00408d60` 在道具之后、World 之前，pre-tick 计时 ≥ 300 且非耐久时整数衰减并清个位；`FUN_00409c00` 只按收取 bit 1 入账／更新历史，恢复 STD、结束宣言牌与环，不清敌弹／激光、不产道具。灵击／被弹计时 > 59 及非耐久超时失去收取。历史为六机体各 110 条加汇总，原始 Shift-JIS 名字与独立饱和计数；按 `!ReplayMode` 写入，bool 表示 `DAT_00477838+0x10 == 1`，与原作 mode != 1 相同。历史持久化及原模式入口待 t8；真实计数与结算提示的 GUI 调用待 t7 | `game/spell_history_test.go`、`scene/stage/spell_test.go`、`scene/stage/ecl/spell_logic_test.go`、`TestVMSpellBgDeclaresHUD`（二面、原作信仰 5000，奖励 800000）、六章基线逐字节比较 |
| 符卡演出（M6） | 近似（环、闪光与 1–6 面 cut-in 已移植；其余未接） | t2 的 `FUN_00409280` 已生成 `bullet.anm` 0x1a1／0x1ab 与各面 stgenm cut-in，按 `FUN_00408d60` 跟随 Boss 与淡出。前三个宣言 World VM 仍未生成；cdbg 为 `DAT_004776f4+0x10/+0x3bc` 两个内联 VM，calc 0x16／draw 0x0e，不是 World／calc 0x1a；当前只留步进接口，初始化与 0x0e 绘制未接，计时 ≥ 60 隐藏 STD 后背景会缺失。Extra 的 `DAT_00474c84` 分支及 ECL 344 未接；第二轮 t7 处理中，所需现成 dump 已供应 | `scene/stage/ecl/spell_visual_test.go`、`anm/spell_child_test.go`、`std/stddraw/spell_gate_test.go`；待完整演出验收与用户目视 |
| 自机完整状态机顺序 | 位精确（已移植的顺序）；state 3 与动态状态差分未做 | `FUN_00425730` 尾部：伤害圈→无敌计时／蓝色闪烁→嵌入 VM.Run→状态时钟 Tick（`0x42618e–0x4261ec`）→初始化时钟 Tick（`0x4261f2–0x426250`）→状态 1 射击→自机子弹更新；t4 已复核时钟位置，t1 把 VM.Run 放在 Tick 前。初始化计时 < 4 禁低速；`DAT_00477704+0x60` 的集中键门控仍未接，state 3 未移植 | `entity/player/player_test.go`、`entity/player/anm_test.go`；待有效 state／invinc 原作 trace |
| 自机本体／判定点／被弹特效／信仰保持矩形 | 有界漂移（VM 与几何已移植，待 L3）；矩形外部门控待 t6 | t1：`FUN_004247f0`／`FUN_004250b0`／`FUN_00425730`／`FUN_00426cf0`／`FUN_00426360`，依据 R10／R11；角色级共享 `pl00.anm`／`pl01.anm`，嵌入 VM 用脚本 0–4，按原始 vx 符号切换；判定点 0x160 与被弹 1×0x162＋32×0x163 为备用 RNG 的 World VM，无敌用蓝色 Color2 与 0x8000。`Renderer.Rect`＝`FUN_0043bda0`，矩形按逐步 float32 与 `0x3e95a95b` 计算；当前节点传 bar = false，虽然 t5 已提供 `ConfigFlags`／`Menu.State`，四个外部门控尚未接。旧精灵加载器、ganim8 与 `entity/effect` 已删除；0x16 仍为 legacy，仅因副机／自机子弹旧绘制尚存 | `entity/player/anm_test.go`、`anm/anmdraw/rect_test.go`、六章基线逐字节比较；用户目视待确认 |
| .rpy 回放解析 | 位精确（解码与编码）；方向按 `FUN_004250b0` | t5：`FUN_00428f60` 是管理器／加载入口，`FUN_00429b60` 才是写出；`FUN_004359b0`／`FUN_00436000`／`FUN_00436210` 等树形 LZSS 与 `FUN_0044b220` 逆加密已移植，解码沿用 `FUN_0044b0d0`／`FUN_00435dc0`。保留变长 USER、slow-rate 与保留字；+0x04 的重复保存累加按原作保留，不把它当实际输入记录数；章节仍按 +0x08 正文长度定位 | `replay/encode_test.go`（9 用户槽位＋4 demo 全文件 Decode→Encode 逐字节相同，Decode 后 DeepEqual）、`replay/replay_test.go`、`replay/validate_test.go` |
| 无头运行器 | 位精确（确定性）；支持同步录制观察者 | t5：`sim/`、`stage.NewHeadless` 与游戏内 `stage.Playback` 共用 `replay.ConfigFor`／`Stream`／`RestoreReplay` 及章头恢复；回放置 `ReplayMode = true`，不启动新录制器、不写进度或符卡历史 | `TestHeadlessDeterministic`、`TestRecordTh10_05AllChapters`、`TestInteractivePlaybackMatchesSim` |
| 差分闭环 | 有界漂移（六章原作采样覆盖范围内 motion 全匹配） | th10_05 第一轮合并后 c0–c5 与 `d1c2/base_cN.txt` 逐字节相同，motion 一致帧数为 15853／17733／29763／29469／39343／40099。c0 使用 `orig_d.txt`；弃用 `orig.txt` 的第 4915 帧 GDI 半帧 MSG 样本，不删 M 行、不放宽容差。c1–c5 用 `orig_05_s2.txt`–`orig_05_s6.txt`；三、四、六面参考结束后的 1699／1301／5288 个复刻帧没有原作对照。一面双方第 15972 帧由 MSG op 20 结束，16031 条记录含通关尾部，不是逻辑差 60 帧。五面掉落 RNG 与六面灵击的旧分歧已经打通。旧 trace 的 score／faith／state／invinc 列无效；当前 probe 读 score `0x474c44`、faith `0x474c4c`（均 ×10）、自机 +0x458／+0x4310，graze 恒 0，需用户重新采集后才能验收 `-scope all`；激光尚无直接 trace 行 | 架构师第一轮合并验收；`th10probe`→`th10sim`→`th10diff`；t9 仅整合报告，未复跑 |
| 资源嵌入 | 位精确（原样资源；汉化只用于演出） | `gamedata/` 只读嵌入，`tools/th10extract` 按 `datfs.EmbedPath` 布局原样提取。t3 新增 front／ascii／text／七面 logo 及繁体 ANT／MST／ECT；逻辑仍使用 th10.dat 的原版 ECL／MSG，汉化只覆盖贴图和演出文字，不运行 ECT 指令 | `gamedata/embed_test.go`（原文件逐字节、贴图逐像素，无数据时跳过）、`gamedata/presentation_test.go`（对照 th10c.dat） |
| ANM 完整 VM | 有界漂移（VM 逻辑已移植；模式 10 粒子未做） | `FUN_0043ee30`、重置 `FUN_00401de0`、效果列表 `FUN_00448860`／`FUN_00448db0`（calc 0x1a、尾部规则）及主／备用 RNG 选择。D1c-2 的敌机槽、t1 自机本体／判定点／被弹特效、t2 已接符卡环／cut-in、t3 GUI／MSG、t5 关卡内菜单均使用 VM；自机本体与 GUI 数字／cdbg 内联对象按所属管理器步进，不应全塞入 World。副机／自机子弹 VM 待 t6，cdbg 初始化待 t7；模式 10 仍 panic，不能称全部路径完成 | `anm/vm_test.go`、`anm/vm_internal_test.go`、`anm/spell_child_test.go`、`anm/world_find_test.go` |
| ANM 绘制（anmdraw） | 有界漂移（模式 0–4、6、8、9 与 Rect 已移植，像素待 L3） | 渲染模式 0–3（A1a）：`anm/anmdraw/anmdraw.go`，`Draw` = `FUN_004451c0`；模式 0 `quad0` = `FUN_00443080`；模式 2 与不旋转的 1、3 `quad2` = `FUN_00443290`；旋转的 1、3 同属 `FUN_004451c0`；`emit` = `FUN_00442670`；`DrawLayer` = `FUN_00448980`。模式 8（`anm/anmdraw/mode8.go`）= `FUN_00444760`，顶点缓冲 `FUN_004462f0`，3D 状态 `FUN_004423e0`。模式 4、6（`anm/anmdraw/mode46.go`）：`FUN_00443f80`（模式 4）、`FUN_00443fb0`（模式 6），四角由 `corners`（`FUN_00443b60`）生成；2D 状态 `FUN_004425a0`。模式 9（`anm/anmdraw/mode9.go`）= `draw9`（`FUN_00444ce0`），2D 三角带；环顶点由 `anm/ring.go` 的 `fillRing` 每帧填入（`FUN_0043ee30` 帧尾 0x44115d–0x44133e）。t1 的 `Rect`＝`FUN_0043bda0`，入口 flush、无纹理 alpha 三角带、末尾缓存失效。t3 的 0x23 弹出字、0x25 符卡数字、0x2b GUI 与 t5 的 0x2e 菜单均非 legacy；链表仅剩 0x16，因为副机／自机子弹旧画法尚存；0x0e 仍未接。模式 5／7 未移植，遇到仍 panic；旧模式调查的文件范围见 `docs/research/m4_modes.md` | `anm/anmdraw/anmdraw_test.go`、`mode8_test.go`、`mode46_test.go`、`mode9_test.go`；`anm/ring_test.go`；`gfx/d3dx.go`（D3DX 矩阵，仅用于绘制）；现有 anmdraw／ring／GUI／MSG／弹出字／菜单测试、`TestRect`、`TestChainDraws`；架构师合并后完整 `TH10_SMOKE=full` 通过，尚非 L3 像素验收 |
| 对话 .msg | 近似（时序已移植，ANM 演出已接；字体与场景控制待验证） | `FUN_00415e90` 的时钟／wait／signal／Ctrl／SE／BGM 时序保持；对话期间世界运行，338／339／345 的消弹／阻塞／扫敌及射击、灵击、敌弹判定、rank 门控不变。t3 由原 World VM 绘制两半身、表情、亮暗、入场／退出、front 0x5a 文本框、Boss 名牌、logo 脚本 2 曲名；op 22／23 发中断 7，op 17 实际暗化文字 VM +0x4c／+0x50，不是立绘。text.anm 时间线与行距／颜色保留，MST 只覆盖繁体演出台词。嵌入 DFYuGaSo W3／W7 的 CPU 字形、度量、抗锯齿、阴影是 GDI／D3DX 近似；MSG op 20 通关加分与最终场景请求待 t8，`DAT_00474c84`／344 接线待 t7 | 既有 MSG／dialog／ECL 时序测试、`scene/dialog/presentation_test.go`、`render/vmtext_test.go`、六章基线逐字节比较；合并后完整 smoke 通过，用户目视待确认 |
| GUI 侧栏／标题／得分弹出字 | 近似（VM、数字与节点已移植；字体和控制器适配待验证） | t3：`FUN_00413810`／`FUN_00413980`／`FUN_00413a20` 构造，`FUN_00413bc0` 启动框／标签／标题／难度；`FUN_00415ae0` calc，`FUN_00415af0`→`FUN_00415800` 在 0x2b 画内嵌数组，框／标签等自由 VM 在 World 层 13–16，不把所有 HUD 塞进 0x2b。Power＝`FUN_004054b0`，残机＝`FUN_00413790`，信仰数字／保持条、滚动得分已接；`DAT_00477840` 独立弹出字为 723 槽／720 生成槽、calc 0x0f／draw 0x23。旧 ebitenui HUD 已删除。练习提示仍用 SingleStage、回放提示用 HasSeed 适配；高分／extend 与最终通关副作用待 t8，字形与出生帧非有限顶点待用户确认 | `scene/stage/gui_test.go`、`scene/stage/score_popup_test.go`、`entity/item/popup_test.go`、`render/vmtext_test.go`；合并后完整 smoke 通过 |
| Boss HUD/符卡名 | 近似（GUI 几何与数字已移植；字体和符卡接口未完全接线） | t3：`FUN_00414900` 血条按 float32 0x3ccccccd 渐增、扣血即时下降；347 写四个比例／ARGB 并按槽序画色段，360 写星数；`FUN_004127a0` 写倒计时，front 0x50／0x51 发警告中断，ascii 箭头按阈值恢复。符卡 text 0x48 名字与 8＋5 个 front 数字由 0x25（`FUN_00409270`→`FUN_004091c0`）绘制；t2 已计算奖励、结算及内存历史，GUI 不另算／发分。真实历史计数、宣言所有权、Get Spell Card Bonus 和失败提示的调用仍待 t7，不能因内存历史已实现就称显示接口完成；344 的 Boss 名字选择仍用敌人类型适配。字体不称逐像素相同 | `scene/stage/gui_boss_test.go`、`scene/stage/ecl/gui_test.go`、`TestVMSpellBgDeclaresHUD`、`render/vmtext_test.go`；完整 smoke 通过，用户目视与接线验收待完成 |
| 回放录制/播放 | 近似（输入与已解码章头已移植；现场不透明状态、慢放统计未齐） | t5：`FUN_0042a8a0`／`FUN_004297d0`／`FUN_0042a450`，录制与播放共用 ConfigFor／Stream／RestoreReplay，存储中的原 .rpy 可多章游戏内播放；练习也录制，回放不录制、不写进度或符卡历史，Debug 是项目侧例外。现场 33 项位置历史与四组副机内部状态只有 capture／restore hook，待第二轮 t6；总 slow-rate 当前为 0，逐 30 帧样本来自 host logic FPS，首窗口 60 Hz 是适配；源回放的原字段保持 | th10_05 六章三字／样本／452 字节章头强测试、`TestInteractivePlaybackMatchesSim`（1500 帧）、`TestPlaybackNeverMarksProgress`、六章基线逐字节比较；待用户互通回放 |
| 菜单/结局/Staff | 近似（关卡内状态 6–13 已移植；其余场景与暂停仍为旧适配） | t5：`DAT_00477830`，`FUN_00422360`／`FUN_004223f0`／`FUN_00422660`／`FUN_004231d0`／`FUN_00423370`；排行、八字名字、回放槽／保存与 mode 13 重开经 front／ascii VM，0x2e 非暂停、非 legacy。截图由 `FUN_00424480` 的请求适配为 0x29 前捕获场地；独立菜单 World 共享备用 RNG。旧暂停提示在链外，原 flag 未定位；主菜单／结局／Staff 视觉与完整关间 fade 未移植，普通六面暂经 现有 ending hook，最终通关加分／scene request 待 t8 | `TestMenu*`、`TestRankingInsertTieAndShift`、`TestRestartStageOriginalFreshState`；截图、小字库、选择动画待用户目视 |
| 存档/配置 | 近似（原 .rpy 存储与 52 字节 cfg 已互通；score.dat 未做） | t5：`game/persistence*` 集中用户 I/O；桌面 `replay/th10_NN.rpy` 安全覆盖，WASM 为页面会话内存。`FUN_00418c40`／`FUN_00420870` 对应 cfg 格式／默认值与保留字，`ConfigFlags`／`ReplayConfig` 已提供，0x200 自动低速输入已接。`FUN_00421fa0` 的排行布局与同分插前已实现，但仍存既有 JSON；t2 内存符卡历史尚未绑定原 score.dat 或跨局公共表。score.dat 容器、校验／压缩、历史持久化及模式字段由第二轮 t8 处理，原样本仍缺 | 存储／覆盖／损坏／越界测试、`TestNativeConfigOriginalFileRoundtrip`（52 字节）、`TestRankingInsertTieAndShift`；WASM 合并验收通过 |
```

已应用的任务更新记录与第一轮合并条目：

```markdown
- 2026-11-07 — t1（D1e-1）：自机本体改用角色级共享 `pl00.anm`／`pl01.anm` 嵌入 VM；按原始 vx 符号启动 1／3／2／4，低速判定点 0x160 发标签 1 淡出，无敌用蓝色 Color2 与 0x8000，被弹尾插 1×0x162＋32×0x163。`Renderer.Rect` 移植 `FUN_0043bda0` 的无纹理 alpha 三角带与缓存失效；信仰矩形按 float32 分组计算，外部门控仍传 false。删除旧自机加载器、ganim8 与 `entity/effect`；副机／自机子弹保留旧节点。新增 12 个测试，六章 trace 与基线逐字节相同；交付当时 `TestRect` 锚点期望与旧格式／lint 未绿，后续合并处理见 2026-11-08 条目，不把当时失败写成通过。
- 2026-11-07 — t4：ECL 349–354 按 `FUN_0040e770` 的 `0x4118bb–0x411b6b` 移植，保留选择 p0／少读参数的原作缺陷；354 的 `SSS` 格式及七关使用次数 0 经反汇编／IR 双重核对，新增边界、求值顺序、特殊输出与整数溢出测试。动态 rank 全部直接写与五个 `AddRank` 物理调用点核对，使用 Tick 后状态时钟、慢帧重复增长，补齐上下限钳位和新局／特定模式 reset hook；六章 trace 不变，golden 未重生成。
- 2026-11-07 — t4：查明备用 RNG `DAT_004918a8` 的唯一播种点为应用启动 `FUN_004201b0` 的 `0x4201e1`，来源 `timeGetTime()` 低 16 位，关卡／回放不恢复。第一轮代码仍以每个 GameState 种子 0 代替，但修正了每次 NewContext 清零，局内跨关共享 `AlternateRNG()`。种子 0／0x1234 在 STD／ANM 初始化前分别运行六章，完整 trace 与主 RNG state／count 相同，只改变该验收范围的演出；最终应用级策略 见 2026-11-08 决定，不能把每局重置当最终还原。
- 2026-11-07 — t4：移植按 x 声像、同 id 128 请求平均、每帧 12 id 与独立 voice 重启，`FUN_0043dd10` 的 float32／整数运算与 DirectSound 单侧 dB 衰减由跨平台 PCM 流表达；支持奇数／短读／Seek，主音量不再覆盖表增益。+6 经全程序／PE 审计确认只写不读，保留闲置属性。音效 flush 位于 calc 之后（`0x43943b`），敌机／弹幕／激光／道具／灵击接线完成；固定样本与接线测试通过，六章 trace 不变，听音待用户。
- 2026-11-07 — t4：完成 R8／R10／RRE／渲染设计／系统审计／player_truth 的指定勘误，注明 R10／R11 与 dump 地址；`FUN_0043bda0` 的入口 flush、SELECTARG1／DIFFUSE、INVSRCALPHA 和末尾缓存失效已澄清，StateTimer Tick 位置确认现有 tail 顺序相符。21 份研究文档的 220 条缺口说法已审计，但只把过时清单交付，正文留待整合；交付当时 Rect 桩、38 个格式项、40 个 lint 项及 c0 旧参考仍未绿。t9 本轮修正文档现状，不重复已落地的指定勘误。
- 2026-11-07 — t5：移植 `FUN_00429b60` 写出、`FUN_004359b0` 树形 LZSS 与 `FUN_0044b220` 逆加密，9 用户槽位＋4 demo 全文件 Decode→Encode 字节相同；变长 USER、逐 30 帧样本与重复保存 +0x04 累加保留。录制按 `FUN_0042a8a0`／`FUN_0042a450`，th10_05 六章输入／样本／452 字节章头强测试通过，不透明自机历史／副机仅留 hook。`DAT_00477830` 的关卡内状态 6–13、名字／排行／保存与 mode 13 重开经 front／ascii VM，0x2e 正名并移出 legacy，暂停提示移到链外。桌面存原 .rpy、WASM 会话内存，多章播放返回且不写进度；52 字节 cfg／默认值／保留字／配置快照互通，score.dat 未做。六章 trace 不变、settled motion 不下降；交付当时 Rect 与旧 lint／格式仍未绿，后续见合并条目。
- 2026-11-07 — t3：GUI／MSG 演出恢复原节点与 World 层；侧栏、残机、Power、信仰／保持条、Boss 名／星数／ARGB 血条／倒计时／箭头、标题／曲名经 front／ascii 与 anmdraw，删除旧 ebitenui HUD／PNG 叠层。独立 `DAT_00477840` 弹出字为 calc 0x0f／draw 0x23，十三个符卡数字在 draw 0x25；MSG 两半身、表情、亮暗、退出、名牌、曲名与 op 22／23 已接原 VM，op 17 暗化文字。347／360 写原字段，MST／ECT 只提供繁体演出文字；DFYuGaSo CPU 光栅化明确近似。四阶段六章 trace 不变；交付当时 Rect／旧绘制列表使完整 smoke 未通过，合并后已解除。奖励／结算／历史由 t2 实现，GUI 精确调用接线仍待第二轮。
- 2026-11-07 — t2（M6）：移植符卡奖励公式、整数衰减、计时／收取结算、被弹／灵击／超时失败及内存历史；arg 2 按原顺序求值但不作奖励，343 不消弹、不产道具，耐久卡不衰减且超时保留收取。生成 bullet 0x1a1／0x1ab 与 1–6 面 cut-in，按 `FUN_00408d60` 跟随／淡出，符卡 calc 校正到 World 前；31 个新增测试通过，六章 trace 不变，golden 未改。交付当时 Rect 桩／旧奖励断言／lint／格式未绿；前三个 World VM、cdbg、Extra 分支、GUI 接线及震屏构造／回调未完成。原补 dump 请求已由现成目录供应，后续实现归第二轮 t7，模式／持久化归 t8。
- 2026-11-08 — **第一轮合并**：架构师决定与合并结果作为本轮总账依据，t9 仅核对代码、整合文档，未复跑验收。
  1．`TestRect` 的 `obj(f, 0, 2, 1, 1)` 是左上锚点 1，不是居中；按 `FUN_00443290` 的 `0x443307–0x443333` 与 `0x4433c9–0x4433fc`，期望改为 `rect(100, 100, 132, 116, …)`。
  2．`TestVMSpellBgDeclaresHUD` 不取脚本奖励，按 `FUN_00409280` 的 `(stage*3+10)*faith*10` 计算；二面、原作存储信仰 5000（项目 50000），期望 800000。
  3．`TestChainDraws` 加入 0x23（`FUN_0042b4d0` 注册的得分弹出字）与 0x25（`FUN_00409270`→`FUN_004091c0` 的奖励／历史数字）；legacy 列表仅剩 0x16，0x2b／0x2e 已移出，0x2e 是菜单而非暂停。
  4．c0 原作参考统一为 `orig_d.txt`；`orig.txt` 第 4915 帧是 GDI 半帧 MSG 采样，弃用，不删对话行或改容差。
  5．保留 t5 的 `GameState.ReplayMode bool`，表示 `DAT_00477838+0x10 == 1`；t2 两处“mode != 1 才写历史”改为 `!ReplayMode`，语义不变。内存历史已实现，GUI 读取真实计数仍待 t7，不能混同。
  6．备用 RNG 定为应用级：启动时取墙钟毫秒低 16 位，一次播种，跨局不重置，依据 `FUN_004201b0` 的 `0x4201e1`；sim／测试固定 0。第二轮 t8 处理中，当前代码仍是每个 GameState 种子 0，不提前记为已实现。
  7．合并顺序为 t1→t4→t5→t3→t2；架构师报告 build、vet、test、WASM、gofmt 与完整 smoke 全部通过，六章 trace 与基线逐字节相同，motion 一致帧数为 15853／17733／29763／29469／39343／40099。合并报告未提供新的 lint 结论，rules 第 6．5 节不把它列为验收项；不宣称 lint 已清零或 L3 目视已完成。
```

## 8．需要用户目视确认的项目

本任务不产生新的视觉效果。第一轮目视／听音项目和具体入口已合并到 `docs/open_items.md` 的 V1–V7：自机、本体矩形、符卡／震屏、GUI／MSG／字体／弹出字、立体声、排行／保存／捕获／重开。
当前 cmd 调试入口默认灵梦 A；两个角色通过正常选人确认，未把仅存在于测试中的 debug-char／debug-shot 当作 cmd 参数。
未启动复刻或原作窗口，不宣称 L3 已通过；t7 未接 cdbg／原震屏前，不要求用户认可旧缺失背景／旧图案。

## 9．与其他任务的接口

- 本任务没有新增／修改代码调用点、hook 或实现函数；以下只是从合并代码记录尚待跨任务接入的现成接口。
- t6：`Player.Draw(draw func(*anm.VM), rect func([4]float32, gfx.Color), hold int32, bar bool)` 对应 `FUN_00426360`，nodePlayer 当前传 false；`Context.CaptureReplayPlayer`／`RestoreReplayPlayer` 的 `func(header []byte)` 对应 `FUN_0042a450` 章头，仅保留原字节，实时历史／副机待接。
- t7：`GUI.BeginSpellPresentation(name string)`、`UpdateSpellPresentation(bonus int64, capture bool, captures, attempts int32, frames int32)`、`EndSpellPresentation()`、`ShowCaptureBonus(amount int64)`、`SetBossSection(n int32)` 已由 t3 提供；原调用者为 `FUN_00409280`／`FUN_00408d60`／`FUN_00409c00`／`FUN_00414900`，当前 t2 调用点仍留注释，fallback 未供真实计数。
- t8：`GameState.SpellHistory *game.SpellHistory` 与 `SpellCounts(id int32) (captures, attempts int32)` 对应 `DAT_0047783c`；尚未绑定公共持久表。`AlternateRNG() *zun.RNG` 由 NewContext 接入，原启动写入为 `FUN_004201b0` 的 `0x4201e1`，当前生命周期仍是每局。`ResetRank(bossPractice bool)` 有旧 hook 名称，但 mode 13 已直接按 Continued 设 −512，后续用原字段消歧，不强迫改为调用 hook。
- 不读取第二轮其他任务的工作树，不提前把其实现写成已合并；以本工作树的第一轮合并代码为当前状态。

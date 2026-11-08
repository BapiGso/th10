# Codex t4 交付记录

## 1. 摘要

任务范围：ECL 349–354、动态 rank、备用 RNG 初始化、音效声像与表字段 +6，以及指定研究文档勘误。未修改 `docs/fidelity.md`、`entity/player/` 或其他任务的工作树。

| 阶段 | 状态 |
|---|---|
| 1．ECL 349–354 | 取证、实现、新增测试与六章基线比较完成；全库基线阻塞见 §4 |
| 2．动态 rank | 取证、实现、测试与六章基线比较完成；全库基线阻塞见 §4 |
| 3．备用 RNG | 取证、实现、种子隔离测试与六章基线比较完成；全库基线阻塞见 §4 |
| 4．音效声像与 +6 | 取证、实现、固定 PCM／接线测试及六章基线比较完成；听音待用户，全库基线阻塞见 §4 |
| 5．文档勘误与过时说法审计 | 指定勘误完成；21 份全文审计与过时说法清单完成 |
| 最终 Go／WASM／六章 trace 验收 | 2026-11-07 已执行并复核；已有全库／参考阻塞，不宣称全绿，不写 DONE |

### ECL 使用统计与语义

按 `m2_bullets.md` §3 的方法检查 `stage01–07.ecl` 的 thtk 反汇编，六条指令在每一关均为 0 次；新增测试还直接统计嵌入的七份二进制 IR，以免空反汇编文件造成误判。四至六面的现成 `.ecl.txt` 为空，因此使用非空的 `.ecl.thtk12.txt`。

| 指令 | 格式 | 原作行为与依据 |
|---|---|---|
| 349 | `ffff` | rank ≥ 512 读取 p2，否则读取 p0；求值后取得 p0 输出地址。`0x4118bb–0x41196e`，低、中档均为 p0 是原作缺陷。 |
| 350 | `ffffff` | 以 −400、−200、200、600 为边界选择 p0、p1、p2、p3、p4，p5 不读取。`0x4118e2–0x41196e`。 |
| 351 | `fff` | 依次求值 p1、p2，再取得 p0 输出地址；`(hi−lo)·(rank+1024)·2⁻¹¹+lo`，逐步 float32。`0x41198b–0x4119e3`；常量 `0x470d48 = 0x3a000000`、`0x470d4c = 0x44800000`，PE 直读。 |
| 352 | `SSSS` | 先取得 p0 输出地址，rank ≥ 512 读取 p2，否则读取 p0；`0x4119fc–0x411acf`。`0x411a0e` 的 −512 比较没有分支，不能据此“修复”中档。 |
| 353 | `SSSSSS` | 先取得 p0 输出地址，分档与 350 相同；`0x411a3e–0x411b0a`。 |
| 354 | `SSS` | p1、p2 为有符号 32 位整数，p0 为整数输出地址；32 位乘积加符号修正后右移 11，向零取整。格式由 `0x411b23–0x411b6b` 的两次 `FUN_00412a00` 与一次 `FUN_00412a10` 核对，现有格式正确。 |

依据：`.tmp/ecl_dump/top/func_0040e770.c` 及 `.tmp/ecl_dump/r9/func_40e770.asm`，与 `bmgr/`、`emgr/` 的对应 case 一致。

### 动态 rank 全部变化点对照

先尝试了共用 Ghidra 命令 `--callers 0x405b60` 与 `--ref-funcs 0x474c98`，但 shell 被沙盒拒绝。替代取证为完整原作 `.text` 的只读 native objdump，加 PE 全节的 32 位地址／相对 CALL 扫描；未启动原作、未修改 Ghidra 工程。证据见 `.tmp/ecl_dump/t4_native/exe.asm`、`.tmp/codex/t4/logs/rank_rng_refs.txt`，脚本为 `.tmp/scripts/t4_pe_audit.py` 与 `t4_native_refs.py`。`AddRank` 恰有下表五个物理调用点；同一物理点可有多个道具分支。

所有增减都在运算后钳到 [−0x400，0x400]；初始化与回放恢复为直接赋值，不钳位。

| 位置 | 条件 | 变化 | Go 对照／处理 |
|---|---|---|---|
| PE 数据 `0x474c98` | 程序装载 | 初值 0 | `NewState` 初始 rank 0 |
| `FUN_00417870`，`0x4179e2` | `DAT_00491fc4 != 0` 的新局分支；`DAT_00474ca0 & 8` 为 0／非 0 | 赋 0／−512 | 新增 `GameState.ResetRank(bossPractice bool)`；`NewState` 调用 false。true 为 boss 练习入口 hook，不把 `SingleStage`（普通关卡练习／sim）误当该位。跨关不调用。 |
| `FUN_00425730`，`0x426290`（钳位 `0x426297`／`0x4262aa`） | GUI+0x9eb8 = 0；敌机管理器存在且 +0x60 ≠ 0；已经 Tick 的自机 +0x478 对 60 取余为 0 | +1 | 原有时钟与门控正确；改为复用 `AddRank(1)`，补齐下限钳位。没有自机状态限制，也没有 Cur ≠ Prev 条件；停在倍数的慢帧每帧都加。 |
| `FUN_0041afd0` → `FUN_00405b60`，`0x41b1ce` | 道具状态 2，y > 472，在上升／重力路径越出底部 | −4 | `item.Manager.Calc` 的 `StateRise`，已有且相符；普通状态 1 的出屏不扣 rank。 |
| 同函数，`0x41b467` | 拾取种类 1／10，`FUN_00418930(1)` 报告火力级别改变 | +12 | `item.Manager.collect`，已有且相符；未升级不加。 |
| 同函数，`0x41b6ae` | 拾取种类 4／11，`FUN_00418930(20)` 报告火力级别改变 | +24 | `item.Manager.collect`，已有且相符。 |
| 同函数，共用 `0x41b55e` | 拾取种类 2 且自机 y < 144 | +8 | `item.Manager.collect`，已有且相符。 |
| 同函数，共用 `0x41b55e` | 拾取种类 2 且自机 y ≥ 144 | +1 | `item.Manager.collect`，已有且相符。 |
| 同函数，共用 `0x41b55e` | 拾取种类 5 | +8 | `item.Manager.collect`，已有且相符。 |
| 同函数，`0x41b713` | 拾取种类 7；无论残机是否已到上限 9 | +0x100 | `item.Manager.collect`，已有且相符。 |
| `FUN_004269d0`，`0x426cc1`（钳位 `0x426cc9`／`0x426cde`） | 提交死亡 | −0x400 | `Player.die` → `AddRank(-0x400)`，已有且相符；未修改 t1 文件。 |
| `FUN_00405860`，`0x405996`（钳位 `0x40599e`／`0x4059b1`） | 发动灵击 | −0x80 | `Bomb.Start` → `AddRank(-0x80)`，已有且相符。 |
| `FUN_00428f60`，`0x42931f` | 回放初次加载，模式 1 | 赋章节头 +0x20 | `sim` 的配置构造／状态恢复已有；t5 回放入口应复用直接赋值。 |
| `FUN_0042a6a0`，`0x42a7d6` | 回放章节开始，模式 1 | 赋章节头 +0x20 | 同上；不得再叠加新局 reset。 |

排除项也核对了：`FUN_004187a0` 的 `0x4187b6` 与 `FUN_004187d0` 的 `0x4187e6` 是 ANM World（`DAT_00491c10`）+0x58 清零，不是 rank；`0x4189eb` 的四字段清零 helper 与 `0x418b40` 的 +0x58 setter 均无代码／数据引用，不是可达变化点。Boss 击破、擦弹、得分自然 extend 不另加 rank；自然 extend 的 `FUN_004188a0` 只加残机（`0x4188a0–0x41892c`），rank +0x100 仅在种类 7 的调用点。

敌机 +0x60 是所有链表节点数，包括隐藏 main，不是“屏上可见敌人数”：创建的 `0x40d198` 无条件加 1，删除的 `0x40d24d` 减 1。当前 `len(vm.actors)` 在 prune 后与这个计数对应。

### 备用 RNG 的初始化、写者与逻辑隔离

备用状态是 `DAT_004918a8` 的 {u16 state，padding，u32 count}，+4 为 `0x4918ac`。它处在 PE `.data` 的未初始化部分，装载时 state／count 均为 0；没有静态构造函数单独播种。完整 PE 地址扫描只有 7 处 `0x4918a8` 引用（`logs/rank_rng_refs.txt`），其中唯一不属于 ANM 抽数的引用是启动播种点。

| 原作位置 | 条件／来源 | 写入与结论 |
|---|---|---|
| `FUN_004201b0`，`0x4201d0` | 应用程序启动；调用 IAT `0x466270`，PE 导入表明确为 WINMM `timeGetTime` | EAX 先写 `DAT_00491ff8`（`0x4201d6`），AX 依次写主 RNG（`0x4201db`）和备用 RNG（`0x4201e1`）；备用种子是毫秒系统计时的低 16 位，不是常量、回放头或一次主 RNG 抽数。 |
| `FUN_0043eac0`，`0x43eb44`／`0x43eb6a`／`0x43eb90` | ANM 特殊浮点变量 10011／10012／10010，VM 有 0x40000000 位 | 通过 `FUN_0044bb20`／`FUN_0044bb90` 抽备用 RNG，推进 state 和 count；无该位抽主 RNG。 |
| `FUN_0043ee30`，`0x440696`／`0x440700`／`0x44070a` | ANM op 40／41 的备用分支 | 同上；state 的逐步写入与 count += 2 位于共用生成器 `[ECX]`／`[ECX+4]`。 |
| `FUN_00417870` 关卡开始 | 所有分支 | 无备用 state／count 写入，不重播种。 |
| `FUN_00428f60`，`0x4292d2`／`0x4292d8` | 回放首次加载 | 只恢复主 RNG 的章节头 +2，并把主 count 清零；备用状态不写。 |
| `FUN_0042a6a0`，`0x42a755`／`0x42a75b` | 回放章节开始 | 同上；备用状态不在章节头中，不进入回放记录。 |

证据：`.tmp/ecl_dump/t4_native/exe.asm`、`pe_headers.txt`（导入表）、`logs/alt_rng_all_refs.txt`，以及既有 `rngcallers/func_428f60.asm`、`rngcallers/func_42a6a0.asm`、`rngcallers/func_43eac0.asm`、`r6/func_43ee30.asm`。生成器每次抽数推进两个 16 位步；新逻辑没有消耗主 RNG 来派生备用种子。

**确定性替代：每个新 `GameState` 使用种子 0，之后一局内跨关共享状态与 count；回放头不恢复它。** 原作是应用程序生命周期内的时间种子，不保证演出可重现；项目选每局固定种子而非系统时间，以保证测试／WASM／单章 sim 的演出可重现，并保留既有种子 0 的表现。这个“每局初始化”是明确的确定性替代政策，不伪称与原作启动时机位精确。跨关原有错误（每次 `NewContext` 重新清零）已修正。

备用 RNG 的可达抽数者都是 ANM；不能只凭这一点宣称没有逻辑影响，因为敌机创建时读取精灵尺寸、子弹读取 VM 的生成／结束状态。为此新增完整六章隔离测试：在任何 STD／ANM 初始化之前分别设备用种子 0 与 `0x1234`，逐帧比较**完整编码 trace**及主 RNG 的 state／count，不把画面当逻辑证据。结果如下；ANM 确实大量使用备用 RNG，两边抽数次数相同，逻辑和主 RNG 全章逐字节一致。

| 章节 | 逐字节一致帧数 | 备用抽数次数（两侧相同） | 主抽数次数（两侧相同） |
|---|---|---|---|
| c0 | 15972 | 29607 | 3719 |
| c1 | 19831 | 136427 | 4122 |
| c2 | 31699 | 71109 | 6612 |
| c3 | 31301 | 137535 | 17117 |
| c4 | 40667 | 97966 | 8228 |
| c5 | 50288 | 85024 | 22049 |

结论：原资源的本验收六章中备用种子只改变演出，不改变逻辑／主 RNG／回放；没有发现逻辑依赖。对任意新增／自定义 ANM 脚本不作同样保证，通用 VM 的随机变量仍可进入逻辑读者，故保留这项回归测试。

### 音效声像、队列与 +6 字段

- `FUN_0043dd10` 的 `0x43dd15` 乘 `0x470b20`，PE 位模式 `0x40a6aaab`（5.208333492279053），`0x43dd24` 经 `FUN_00463b2c` 向零取整：`pan = trunc(float32(x·5.2083335))`。x 是游戏坐标，场地边缘 ±192 对应 ±1000，即对另一声道衰减 10 dB，不是硬切左右。
- 排队（`0x43dd31–0x43dd99`）：每帧最多 12 个不同 id；同 id 最多接受 128 个 pan 请求。`FUN_0043dc90` 是直接整数 pan 入口，已有调用者均给 0；两个入口共用同一队列，不能只保留首个 x。
- `FUN_0043ddf0`（`0x43e279–0x43e345`）以有符号 32 位求和、除以请求数，向零取平均；对 id 对应的独立 DirectSound buffer 依次 Stop（vtable +0x48）、SetCurrentPosition(0)（+0x34）、SetPan(平均值)（+0x40，`0x43e2c7`）、SetVolume（+0x3c）、Play（+0x30）。同 wav 不同 id 也有各自的 buffer（`0x43dbe0–0x43dc1a`）。Go 改为按 id 重启一个 voice，而非每帧累加重叠副本。
- `FUN_00439390` 在 calc chain `0x43942f` 之后、`0x43943b` 调音效处理；故 `game.Game.Update` 的音频 Update 移到场景 Update 后，既无额外一帧延迟，也能平均本帧全部请求。
- PCM 包装流 `panStream`：表音量振幅 `10^(vol/2000)` 乘声道系数；pan > 0 时 left = `10^(−pan/2000)`、right = 1，pan < 0 时反之。以 16 位有符号样本乘合并增益、向零量化一次；Player volume 只保留主 SE 音量，因此调整主音量不再丢失表音量。流按完整 stereo frame 缓冲，支持奇数／短读与 Seek，既不修改共享 PCM，也不依赖 OS；非 tools WASM 构建通过。

**+6 的实际语义是本版闲置的附加属性，不是可证明的冷却、优先级或声像字段。** `FUN_0043dc90`／`FUN_0043dd10` 在新 id 入队时符号扩展 short，存到管理器 `+0x408+id·4`（`0x43dcf0`／`0x43dd8b`）。属性数组在 `FUN_004659a0`（全局地址 `0x492998`）及 `FUN_0043e580` 初始化为 −1；逐函数追踪所有管理器引用、全程序 +0x408 引用以及这 128 个全局 dword 的 PE 地址扫描，只有初始化与两处赋值，没有运行时读者。SE 队列 flush 不读取 +6。Go 将它保留为 `seEntry.aux` 与 `Manager.seAux`，按原作复制，不赋予臆造行为。原作者预留此属性的设计意图不能由二进制证明，但本版“只写不读、无可观察效果”已闭合。

证据：既有 `item/func_43dd10.c`、`laser3/func_43dd10.c`；`.tmp/ecl_dump/t4_native/sound_queue_flush.asm`、`sound_flush.asm`、`pe_headers.txt`；`logs/sound_callers.txt`、`sound_aux_refs.txt`；`render/func_439390.c` 与 native `0x43942f–0x43943b`。

已接原作有位置的调用：ECL 336（`0x410d59`）、敌机击破（`0x40e60e`）、受击（`0x40e4f4`／`0x40e509`）、灵击（`0x405942`）、弹幕发射／擦弹／指定 etEx 音效、激光擦弹与指定 etEx、道具升级／拾取（`0x41b446`／`0x41b68c`／`0x41b72d`，取道具 x，不是自机 x）。自然 extend、普通变换音效等 `FUN_0043dc90` 路径仍居中。自机 SHT 发射音效 `0x42814d` 留给 t1，见 §9。

### 指定文档勘误

已就地更正 R8 的回调归属、层 8、同尺寸矩形平移与门控 4，并标注 R10／R11 更正；R8 未决 #4 与 R10 未决 #14 的门控已闭合。R10 未决 #6 的 StateTimer Tick 在 `0x42618e–0x4261ec`，Go `tail()` 的闪烁／动画／Tick 顺序正确，未修改 `entity/player/`。`FUN_0043bda0` 在所有研究文档中的描述已核对：纠正 RRE 的函数名笔误与“函数不 flush”说法，补齐不绑定纹理而用 SELECTARG1／DIFFUSE 忽略绑定纹理、DESTBLEND = INVSRCALPHA、绘制后缓存失效。节点 0x2e 纠正为排行／回放文字；擦弹火花绘制路径纠正为 `chain.go` → `layers` → `sv.ANM.DrawLayer`；`player_truth.md` 的“命中只置标志 4”改为擦弹。每处附节号／dump 地址。

### 全面审计：过时的“未移植／未实现”说法

完整读取 `docs/research/*.md` 的 21 份文档，逐条检查 220 处实现缺口／缺失说法及其相邻段落；原始索引在 `logs/docs_gap_audit.txt`（`t4_docs_audit.py`）。表中只列确实已经过时的实现状态；历史研究快照可保留日期并追加新状态链接，是否改正文由架构师决定。**除任务明确指定的勘误，不修改这些说法。** 对仍未做的自机 ANM、其余射击、符卡奖励／VM、GUI、冻结 hook、模式 5／7／10、激光 trace、矩形 Rect 等不误报已完成。

| 文档位置 | 过时说法 | 当前代码与总账依据 |
|---|---|---|
| `ecl_gaps.md` §1–§2 | 无位置取整、`FUN_00404610`／完整缓动未移植，模式 8／17 缺失 | `ecl/motion.go`、`zun/interp.go`／`zun/zun.go`；fidelity“运动核心”“缓动与插值器” |
| `ecl_gaps.md` §3 | 283／289／291／296／297 格式错误，295／297 未实现，绕圆／两层／flip／哨兵不符 | `ecl/opcodes.go`、`game_ops.go`、`motion.go`；fidelity“运动核心” |
| `ecl_gaps.md` §3–§4 | layer B 没有 3D 全局偏移，全程 float64、角度常量未取证 | `motion.go` 的 3D 分支、float32 与 `zun.NormAngle`；fidelity“运动核心”“特殊变量” |
| `ecl_gaps.md` §5 | 回放种子写死、ANM 共享 RNG 未建模、特殊变量只处理六项 | `Context.RNG`、`VM.Update`、`ecl/special.go`；fidelity“ZUN RNG”“特殊变量” |
| `ecl_gaps.md` §5 | 随机 ET 模式／randSpeed 公式仍错 | `entity/bullet/emit.go` 的 6／7／8 模式；fidelity“ECL 弹幕” |
| `ecl_gaps.md` §6 | 子弹没有出生状态，按年龄永久动作、固定 64px 出屏 | `entity/bullet/update.go`、`ex.go`、`emit.go`；fidelity“ECL 弹幕”“出屏删除” |
| `ecl_gaps.md` §6 etEx 表 | 加速、反弹、宽限、重设精灵、等待、删除、音效、分裂、跳转等“缺” | `entity/bullet/ex.go`／`update.go`；fidelity“ECL 弹幕” |
| `ecl_gaps.md` §6 | 400 默认值／401 原点与禁射半径缺失，28 弹型压成 4，池为 2048 | `ecl/bullet_ops.go`、`entity/bullet/tables.go`／`bullet.go`；默认值测试、2000 游标槽 |
| `ecl_gaps.md` §6 | 激光只是占位，431／433 未拷贝变换表 | `ecl/bullet_ops.go`、`entity/laser`；fidelity“ECL 弹幕”，M2 激光管理器 |
| `ecl_gaps.md` §6 | 擦弹有 Age > 4 近似 | `bullet.update`／`stage.bulletHit`；fidelity“自机被弹／擦弹” |
| `ecl_gaps.md` §7 | 346、414–419、422–424、429／432／434、430 缺失 | `ecl/game_ops.go`／`bullet_ops.go`；fidelity“ECL 弹幕”及相关测试 |
| `ecl_gaps.md` §7 | 349–354 缺失 | 本任务 `rank_ops.go` 与逐条测试；七关出现次数均 0 |
| `ecl_gaps.md` §7 | 420／421 只转普通得点，367 只记标志，272／273 空操作 | `item.KindCancel`、`gameSpeed` 接线、`visual.effectOp`；fidelity“道具管理器”“运动核心”，D1c-2 |
| `ecl_gaps.md` §8–§10 | 全敌统一积分／ECL、对话冻结、创建晚一帧、通用出屏删除、差分尚未打通；优先级清单仍列这些缺口 | `VM.updateActor`、`createOp`、`Stage.update`；fidelity“逐个更新顺序”“差分闭环” |
| `m1_interrupts.md`“推迟到 M2”#1–#3 | 更新交错／创建双重更新未核对，异步按旧先执行 | `VM.Update`／`createOp`／`runActor`；fidelity“逐个更新顺序”，`m2_motion.md` §6／§11 |
| `m1_interrupts.md` 同节 #4 | /5 与 HP 变化音效未移植 | `interrupt.hurt`、`Context.EnemyDamage`、`visual.updateVisual`；fidelity“敌机受伤与击破”，D1c-2 |
| `m1_interrupts.md` 同节 #6 | 得分与掉落未移植 | `VM.kill`／`dropItems`／`scatterDrops`；fidelity“敌机受伤与击破”“敌机掉落”；本项的体碰撞计时仍缺，需拆句 |
| `m2_motion.md` §10–§11 | 道具尚未移植，掉落沿用旧实现放到 M7 | `entity/item`、`ecl.scatterDrops`；fidelity“道具管理器”“敌机掉落” |
| `m2_motion.md` §11 | 方向动画与灵击 ANM 换脚本未实现 | `visual.updateVisual`／`dirScript`／366 restart；fidelity D1c-2 |
| `m2_motion.md` §11 | 动态 rank 每 60 帧为近似 | 本任务完整写入点表与 `rank_test.go`；原作 `0x426256–0x4262b4` |
| `m2_motion.md` §11 | 激光管理器未移植 | `entity/laser`；fidelity“ECL 弹幕”与 `m2_bullets.md` §10 |
| `m2_bullets.md` §6／§8 | 音效 +6 未查明、按 x 声像未实现 | 本任务声像／PCM／闲置属性取证，`audio/pan.go`／`se.go` |
| `m2_bullets.md` §7／§8 | 备用 RNG 种子未知、消弹种类 8 暂代普通点数 | 本任务时间源／共享替代种子；`entity/item.KindCancel` 已有，fidelity“道具管理器” |
| `m4_modes.md` §5 | DrawLayer 只有测试调用，World／子弹／灵击都不经 anmdraw，模式 9 会 panic | `chain.layers`、实体三节点、`anmdraw.mode9.go`；fidelity D1b／D1c；正文与自身 2026-11-01 更新矛盾 |
| `m4_modes.md` §7 #1 | 现行嵌入资源只有 17 个；无法核对未嵌入的 pl01 | 工作树实际有 18 个 ANM，含 `assets/anm/pl01.anm`；front／title／staff 仍未嵌入，不能一并报完成 |
| `m4_r8.md` R8-H，0x14 行 | Go 还在 nodeEnemies／drawEnemies 画红矩形，没有敌机 World VM | `chain.go` 的 0x14 空节点、`eclActor.vms`／`visual.go`；fidelity D1c-2 |
| `m4_r8.md` R8-H，0x19／0x1b／0x1d 行 | 入口仍在 chain_legacy.go 的 drawItems／drawLasers／drawBullets | `chain.go` 的三个直接 ANM 节点；fidelity D1c；冻结 hook 未接仍正确 |
| `m4_r8.md` R8-H／未决 #2–#3 | 子机层表是推测、层号／回调仍未确定 | R10-B.1／B.2 已闭合；本任务只改指定 B.3，历史表和其余未决由架构师决定如何消歧 |
| `m4_r8.md` 未决 #1 | 敌机 VM 创建与层未知 | R9-A／B 与 `visual.slotOp`／`effectOp`；fidelity D1c-2 |
| `m4_r9.md` 一行结论／速览／Go 对照 | 敌机仍用 ebiten，槽句柄、方向、受击计数没有 Go 字段 | `eclActor.vms`／`dir`／`flash`；fidelity D1c-2 |
| `m4_r9.md` B／F／Go 对照 | 259／262／269 不生成 VM，263／264 忽略文件，272／273 不处理，World 没有头插／相关句柄 API | `visual.slotOp`／`effectOp`、`World.SpawnFront`／`SpawnTail`／句柄辅助；fidelity D1c-2 |
| `m4_r9.md` Go 对照 | MainScript／AnmSlot／Hidden 仍为 Enemy 字段，flags 仅 0x20000／0x8000 有效 | 这些旧字段已删；`visual.go` 实现隐藏、方向、3D 定位、bomb swap 等；fidelity D1c-2 |
| `m4_r9.md` Go 对照，受击变色 | updateVisual 没有受击闪烁 | `visual.updateVisual` 的 hitColor2／hitFlash／hitSE；fidelity D1c-2 |
| `m4_r10.md` 未决 #1 | pl01.anm 不在仓库，Marisa 脚本无法核对 | `assets/anm/pl01.anm` 已嵌入；Marisa 射击尚未移植仍正确 |
| `m4_render.md` §2.2 D1c | 敌人尚待 R9 取证后移植 | D1c-2 已移植；自机 0x16 待 t1 仍正确 |
| `presentation_gaps.md` 结论／§2／§3／§7 | 没有 STD 解析／VM／相机、3D、透视、shader、DrawTriangles，七面平铺 PNG | `std`／`stddraw`／`supervisor`／`gfx`；fidelity“STD”与 S3，全七面默认 STD |
| `presentation_gaps.md` §1／§6 | 没有过滤、混合、层表，顺序只硬编码在旧 Stage.Draw | `gfx`／`anmdraw`／`chain.go`；fidelity D1b／D1c 与 ANM 绘制 |
| `presentation_gaps.md` §2 | ANM 只切精灵，无逐帧 VM；位置／缩放／插值／数学等全部缺 | `anm/vm.go`／`world.go`，`zun.Interp`；fidelity“ZUN RNG”“缓动”“子弹／敌人贴图” |
| `presentation_gaps.md` §2 | loader 故意忽略纹理；SubImage 裁 UV，Sheet.Sub("") 给全弹种／Boss同图 | THTX loader、`gamedata`／`anmdraw.NewSheet`；旧敌机 sprite loaders 已删，实体改用 VM 与 wrap shader |
| `presentation_gaps.md` §2 | bullet.anm 从未加载，stgenm02–07／stage0N.anm 不在 assets | `bulletFile`、ECL `fileAt`、STD Enable；这些资源已嵌入；front／ascii ANM 仍缺，需拆句 |
| `presentation_gaps.md` §5 | 道具仍彩色矩形、激光仍矢量线、敌弹四种形状、Boss 全用 stgenm01 | `item.Draw`、`laser.Common`、`bullet.VM`、`visual.fileAt`；fidelity 对应实体与 D1c／D1c-2 |
| `presentation_gaps.md` §5 | ECL 272／273 空操作 | `visual.effectOp`；fidelity D1c-2 |
| `systems_gaps.md` 摘要／§3 | stage1 完全没接 MSG／HUD／Boss BGM；.msg 只取文本不执行 | `stage1.eclScript.Init`／`BossHUD`／`Context.LoadMSG`、`dialog.msgvm`；fidelity“对话 .msg”“Boss HUD” |
| `systems_gaps.md` §2（0x378c bit3） | 受击蓝闪与 sound 未移植 | `visual.updateVisual`／`hitSE`；fidelity D1c-2 |
| `systems_gaps.md` §5 | 366 的 ANM 切换只记脚本号 | `visual.updateVisual` 的 `World.Restart`；fidelity D1c-2；GUI／自机灵击演出未做仍正确 |
| `systems_gaps.md` §5 保留旧列表 | 所有机体仍共用灵击范围／伤害／结束条件 | `bomb.go` 的角色／重型符卡分支、ANM 生命周期；fidelity“灵击生命周期”，该段前面已标旧占位作废 |
| `systems_gaps.md` §9 建议顺序 | 334／341／340、stage1 MSG/HUD、MSG 解释器、信仰保持／衰减仍列待修 | `interrupt.go`、各关包装、`dialog`、`GameState.TickFaith`；fidelity 对应行；其余 SHT／GUI／符卡仍需保留 |

补充核对：`m4_r6.md` 的模式 10 粒子未移植、`m4_s2c.md` 的特殊 STD 路径静态未找到、`m4_s3.md` 的“当时 mode6 未做”已明确标注历史并紧跟完成更新、`replays.md` 与 `th10c_dat.md` 的数据说明均不应当作过时实现状态改写。`m4_s2b.md` 已在补正节标明 Hermite 分组修复，不重复列为新缺口。

## 2. 改动文件清单

新增：

- `audio/pan.go`：DirectSound 声像公式与跨平台 stereo PCM 增益／Seek 包装流。
- `audio/pan_test.go`：声像、固定样本、短读／奇数读／Seek、队列上限与闲置 +6 属性测试。
- `audio/pan_listen_test.go`：默认跳过的立体声听音检查，需用户显式启用。
- `scene/stage/ecl/rank_ops.go`：逐条实现 349–354，保留原作参数索引缺陷。
- `scene/stage/ecl/rank_ops_test.go`：格式、七关使用统计、六条指令边界、整数溢出与栈求值顺序测试。
- `scene/stage/ecl/rank_test.go`：自机状态时钟、所有状态、重复慢帧、main 计数、MSG 与上下限测试。
- `game/rank_test.go`：新局／boss 练习 rank 重置与加减钳位测试。
- `game/alternate_rng_test.go`：备用固定种子与不同局状态独立测试。
- `scene/stage/alternate_rng_test.go`：备用状态跨 Context／关卡保留，回放主种子不污染备用状态。
- `sim/alternate_rng_test.go`：两个不同备用种子的六章完整逻辑 trace 与主 RNG 隔离测试，逐章顺序运行，不保存大 trace。
- `scene/stage/sound_test.go`：弹幕、激光、道具和灵击的有位置／居中音效接线测试。
- `scene/stage/ecl/sound_test.go`：ECL 336 与敌机击破从游戏坐标 x 发音的测试。
- `docs/handoff/codex_t4.md`：本交付记录。

修改：

- `audio/audio.go`：声音按 id 重启 voice、保留表音量、帧尾 flush，改用停止读取 API。
- `audio/se.go`：每 id 聚合至多 128 次声像请求，记录闲置 +6 属性。
- `entity/item/item.go`：道具升级与拾取传入道具 x，自然 extend 仍居中。
- `game/game.go`：音效 flush 移到 calc／场景更新之后，与 `0x43943b` 对应。
- `scene/stage/ecl/opcodes.go`：增加六条指令的命名常量与二进制依据。
- `scene/stage/ecl/game_ops.go`：接入六条指令的执行分支。
- `scene/stage/ecl/vm.go`：更正 rank 注释，周期增长复用完整上下限钳位。
- `game/state.go`：显式新局 rank 初始化及 boss 练习入口的 reset hook。
- `scene/stage/bullets.go`：ANM 的备用 RNG 改为同一局共享状态。
- `scene/stage/ecl/visual.go`：受击音效传入敌机的游戏坐标 x。
- `scene/stage/bomb.go`：灵击发音传入原作取到的发动位置 x。
- `scene/stage/sound.go`：新增可兼容现有无头 recorder 的 `PlaySEIDAt` hook。
- `zun/rng.go`：补上时间种子的地址依据，改正“只有 op 88 子对象用备用 RNG”的过窄注释。
- `docs/research/m4_r8.md`：R10／R11 的指定就地勘误及矩形函数阶段状态补正。
- `docs/research/m4_r10.md`：StateTimer Tick 定位与门控 1／4 标为已解决。
- `docs/research/m4_render.md`：节点 0x2e 纠正为排行／回放菜单文字。
- `docs/research/m4_render_re.md`：矩形函数名笔误、纹理／混合／flush 与缓存说明补正。
- `docs/research/systems_gaps.md`：第 8 节区分旧池与 World，纠正擦弹火花的 D1b 绘制路径。
- `docs/player_truth.md`：标志 4／火花／音效这句话描述擦弹，不是命中。

删除：无。共新增 13 个、修改 19 个工作树文件；`logs/inventory.txt` 以主树 `.tmp/codex/base` 为只读基线核对。所有已有测试、golden、`go.mod`／`go.sum` 与 `entity/player/` 均未改。

不合并的取证／验收辅助：主树 `.tmp/scripts/t4_pe_audit.py`、`t4_native_refs.py`、`t4_filter_msg.py`、`t4_docs_audit.py`、`t4_inventory.py`；任务目录 `env.ps1`、`verify.ps1`、`trace.ps1`、`dump_requests.md`；`t4_native/` 输出与 `logs/` 日志。它们没有写入原作数据目录或其他任务。

## 3. 移植的函数地址

| 原作 | Go |
|---|---|
| `FUN_0040e770` case 349–354 | `(*ecl.VM).execRankOp` |
| `FUN_0040e770` 的五档选择 | `ecl.rankSelectTier5` |
| `FUN_00450070`／`FUN_00412a10` 输出引用语义 | `(*ecl.VM).storeRankResult`，复用既有 `storeVar` |
| `FUN_00417870` 的 `0x4179e2` | `(*game.GameState).ResetRank` |
| `FUN_00425730` 的 `0x426256–0x4262b4` | `(*ecl.VM).Update` 的 rank 步骤；复用 `GameState.AddRank`（`FUN_00405b60`） |
| `FUN_004201b0` 的 `0x4201e1` | `GameState` 的备用 RNG 初始化（时间源替代为每局种子 0）／`AlternateRNG` 共享 |
| `FUN_0043dd10` | `audio.SEPan`、`Manager.PlaySEIDAt`／`queueSE` |
| `FUN_0043dc90`（pan = 0 的已有调用者） | `Manager.PlaySEID`／`queueSE` |
| `FUN_0043ddf0` 的 `0x43e22b–0x43e345` | `Manager.flushSE`，平均 pan／独立 id voice；DirectSound 声道增益由 `panStream` 等效 |
| `FUN_004659a0`／`FUN_0043e580` 的闲置属性初始化 | `audio.NewManager` 的 `seAux = -1` |
| `FUN_00439390` 的 `0x43943b` | `Game.Update` 的帧尾音效 flush |

## 4. 测试

新增 ECL 测试：`TestRankOpcodeFormats`、`TestRankOpcodesUnusedInStages`、`TestRankSelect349And352`、`TestRankSelect350And353`、`TestRankLerp351`、`TestRankLerp354`、`TestRankOpcodeSourceEvaluation`、`TestRankOpcodeSpecialOutput`。

阶段 1：`go build ./...`、`go vet ./...` 退出 0；`go test -count=1 ./...` 中 ECL 包及全部新增测试通过，但 `anm/anmdraw.TestRect` 因原有 `Rect not implemented` panic 失败（t1 范围，文件未改）；其余包通过。`gofmt -l .` 列出 39 个原有未格式化文件，未做越界格式整理。`golangci-lint run` 报 42 个原有问题（30 errcheck、1 ineffassign、5 staticcheck、6 unused）。WASM：过滤 tools 后的 `go build` 退出 0；`go list ./...` 在 js／wasm 下因 tools/th10probe 导入 windows 报错，退出 1，非 tools 的包列表与构建成功。完整日志在 `.tmp/codex/t4/logs/ecl_*`。

已有断言、容差、golden 均未修改；尚不能称全库验收通过。

阶段 2 新增 `TestResetRank`、`TestAddRankClamp`、`TestDynamicRankClockAndGates`、`TestDynamicRankHeldClockAndAllStates` 全部通过；周期规则不再标“每 60 帧近似”。完整检查与阶段 1 相同：build／vet／非 tools WASM build 退出 0，TestRect／39 个原有格式项／42 个原有 lint 项仍阻塞。

阶段 3 新增 `TestAlternateRNGDeterministicSeed`、`TestAlternateRNGPersistsAcrossContexts`、`TestAlternateRNGDoesNotChangeReplayTrace` 全部通过；完整六章种子隔离测试耗时 41.17 秒（完整测试包 43.381 秒）。build／vet／非 tools WASM build 退出 0；全库仅 TestRect 失败；gofmt 原有格式项降为 38（本任务触及的 `bullets.go` 经 gofmt）；lint 原有 42 项仍失败。

阶段 4 新增 `TestSEPan`、`TestSEPanGains`、`TestPanStreamSamples`、`TestPanStreamShortSourceReads`、`TestPanStreamSeek`、`TestPanStreamIncompleteFrame`、`TestSEQueuePanAndAux`、`TestSEQueueLimits`、`TestPositionalSoundWiring`、`TestItemPositionalSounds`、`TestEnemySoundPanning` 全部通过。`go test -count=1 ./audio ./entity/item ./scene/stage ./scene/stage/ecl` 退出 0（2.472／6.933／19.796／16.317 秒）；完整检查 build／vet／非 tools WASM build 退出 0，全库仍仅 TestRect 失败，38 个未格式化项不变。lint 剩 40 个原有问题（30 errcheck、1 ineffassign、3 staticcheck、6 unused），因本任务音效生命周期改用 `PauseAndStopReading`，原有两个弃用 Close 告警消失；没有新增 lint 问题。`TestPlaySEPanning` 是新加入的 opt-in 检查，默认跳过，未在本次启用设备试听。

**最终验收（阶段 5 结束，2026-11-07）**：所有命令在 t4 工作树执行，完整日志 `logs/final_*`。没有因为原有失败提前跳过后续检查。

| 检查 | 实际结果 |
|---|---|
| `go build ./...` | 退出 0 |
| `go vet ./...` | 退出 0 |
| `go test -count=1 ./...` | 退出 1，仅 `anm/anmdraw.TestRect` 因未实现占位 panic；其余包通过。audio 5.434 秒、item 1.984 秒、game 1.926 秒、stage 15.966 秒、ECL 13.468 秒、sim 40.111 秒。 |
| `gofmt -l .` | 38 个原有未格式化文件。逐一 cmp 对基线均相同（38／38），所有本任务修改的 Go 文件已格式化；未越界整理其他文件。 |
| `golangci-lint run` | 退出 1，40 个原有问题：errcheck 30、ineffassign 1、staticcheck 3、unused 6。 |
| js／wasm 构建（排除 tools） | `go build` 退出 0；规则命令的 `go list ./...` 仍因 tools/th10probe 的 windows 导入报错退出 1，过滤后的非 tools 列表与构建完整。 |
| 已有断言／golden／依赖／player 范围 | `t4_inventory.py` 对基线确认全部未改；`Rect` 占位文件 cmp 退出 0。 |

全库格式与 TestRect／lint 不绿，指定 c0 原作参考也仍含旧 M 行失败，所以按“没绿不说做完”的规则**不写 DONE**。交付代码、证据、文档已保存两份，供架构师审查合并；待架构师协调基线／t1 与参考后再做全绿验收。

## 5. Trace 结果

阶段 1 六章均与 `d1c2/base_cN.txt` 经 `cmp -s` 逐字节一致（退出 0），故 ECL 补齐没有改变任何 trace。

| 章节 | 与基线 cmp | 原作 `-scope motion` |
|---|---|---|
| c0 | 相同 | 指定命令在 4915 帧报告 MSG clock／op 分歧；原作对基线也在同帧同字段分歧。仅排除 M 行的补充比较中 15854 帧全一致。 |
| c1 | 相同 | 17733 帧一致 |
| c2 | 相同 | 29763 帧一致 |
| c3 | 相同 | 29469 帧一致 |
| c4 | 相同 | 39343 帧一致 |
| c5 | 相同 | 40099 帧一致 |

原规则给出的 c0 motion 基线数为 15853；本任务不会把报错命令伪报为该结果。六个生成文件比较后均删除，仅保留日志与结论。

阶段 2 六章再次 cmp 全相同，c1–c5 的指定 motion 比较帧数不变。c0 的补充比较由 `.tmp/scripts/t4_filter_msg.py` 只删除原作与本任务的 M 行，保留 F／E／B 行及所有帧索引，交给同一 `th10diff -scope motion`；没有修改原作、基线或 golden，临时副本比较后删除。此补充只证明运动无回归，**不掩盖指定命令的 MSG 失败**。该 MSG 问题属 t3 范围，见 `logs/rank_c0_reference_vs_base.txt` 与 `rank_c0_motion_only.txt`。

阶段 3、阶段 4 也分别顺序生成并比较六章：12 次 cmp 全相同，指定 motion 的 c1–c5 一致数均不变，c0 的 M 行问题与纯运动补充结果也不变。声像接线、PCM 包装、音效队列没有改变 trace；所有生成的大文件及 M 行过滤副本均在比较后删除。

**c0 分歧的原作依据**：`docs/fidelity.md` 的 2026-10-31“差分工具”更新记录明确记载，4915 帧 MSG op 16 用 GDI 时旧 probe 会采到半帧内部状态，后来以 `sampleSettled` 双采样丢弃该样本修正。因此规则指定的 `orig.txt` 中这项 M 差异不是本任务引入的逻辑分歧；给基线运行同一命令也在 4915 帧同字段失败。请架构师核对参考是否应使用修正后的采样，或保留这里的补充纯运动结论；本任务没有运行／重新采样原作。

**最终六章复核**：`logs/final_trace.txt` 中 c0–c5 的 sim 与 cmp 退出码全部 0；c1–c5 的原作 motion 命令退出 0，分别 17733／29763／29469／39343／40099 帧；c0 指定命令在 4915 帧退出 1，基线也相同，M 行过滤的补充比较退出 0、15854 帧。没有任何运动帧数下降或新分歧。任务 `trace/` 目录最终为空（生成文件与过滤副本已删除）。

## 6. 待决与推测

1. Ghidra 取证命令在启动 shell 时被沙盒策略拒绝。已按规则停止尝试，完整命令记录在主树 `.tmp/codex/t4/dump_requests.md`；所有必需逻辑点已以既有 Ghidra dump ＋只读 PE／native objdump 替代闭合。架构师可选运行该命令独立复核，不需要新的 dump 才能审查当前实现。
2. 备用 RNG 的时间源明确不可重现。本任务每局种子 0 是**明确的确定性替代**，不是未经标注的原作初始化近似；应用程序菜单到新局边界的 policy 见 §9。
3. +6 在本版没有运行时读者。属性的原始设计命名／意图未知，不猜成优先级或冷却；“只写不读、无行为”已经取证并测试。
4. boss 练习入口需要 t5 调用 `ResetRank(true)`；普通新局／回放／普通关卡练习已经不误用该位。没有越界实现菜单控制器。
5. 当前工作树的全库 TestRect、格式／lint 旧问题与 c0 旧参考 M 行阻塞是验收失败，不能称全绿；不通过修改 t1、已有断言、golden 或其他文件来掩盖。

## 7. 建议的 `docs/fidelity.md` 改动

建议替换以下完整状态行，合并时还要保留其他任务对同一行的补充；未直接修改总账。测试栏列真实已通过的专门测试，不把全库阻塞伪报为通过。

```markdown
| ECL 敌机/移动 256-368 | 有界漂移（运动部分已移植，见“ECL 运动核心”；349–354 已逐条移植） | `FUN_0040e770`；349/352 在 rank < 512 时读 p0，350/353 按 −400/−200/200/600 选择 p0–p4（保留原作参数索引缺陷）；351/354 按原作求值顺序与 float32／int32 向零取整插值；354 格式 `SSS` 经汇编核对；七关均未使用这六条。346（401 禁射半径）已移植 | `TestRankOpcodeFormats`、`TestRankSelect349And352`、`TestRankSelect350And353`、`TestRankLerp351`、`TestRankLerp354`、`TestRankOpcodeSourceEvaluation`、`TestVMAllStagesStableAndFire` |
| ZUN RNG | 有界漂移（主 RNG 回放取种；备用 RNG 为确定性演出替代） | `FUN_0044bb90` 常量 PE 直读；主 RNG 种子来自章节头 +0x02，与 ECL／子弹 ANM 共用；备用 `DAT_004918a8` 原由 `FUN_004201b0` 在 `0x4201e1` 写入启动 `timeGetTime()` 低 16 位，关卡／回放不恢复、不录制。Go 每局替代为种子 0，跨关共享 state/count，不抽主 RNG 派生；两个不同备用种子的 th10_05 六章完整逻辑 trace 与主 RNG state/count 相同 | `TestAlternateRNGDeterministicSeed`、`TestAlternateRNGPersistsAcrossContexts`、`TestAlternateRNGDoesNotChangeReplayTrace` |
| 动态 rank | 位精确（正常／回放规则已取证；boss 练习入口 hook 待 t5） | `DAT_00474c98`，`GameState.Rank` 跨关保留、章节头 +0x20 恢复；`FUN_00425730` `0x426256–0x4262b4`：Tick 后 +0x478 对 60 取余为 0、GUI+0x9eb8 = 0、敌机管理器 +0x60 ≠ 0 时 +1，不要求 Cur ≠ Prev，钳 ±0x400；道具、死亡 −0x400、灵击 −0x80 与全部调用点相符。新局 0，`DAT_00474ca0 & 8` 的 boss 练习为 −512（`ResetRank(true)`）；完整变化点表见 `docs/handoff/codex_t4.md` | `TestResetRank`、`TestAddRankClamp`、`TestDynamicRankClockAndGates`、`TestDynamicRankHeldClockAndAllStates`、golden、六章基线 cmp |
| 音效表 | 位精确（表／队列）；PCM 声道增益等效，听音待确认 | `0x4749c8` 47 项与 `0x474b40` wav 表；`FUN_0043dd10`：pan = trunc(f32(x·0x40a6aaab))，同 id 一帧至多 128 个 pan，整数平均（`0x43e28d`），每帧最多 12 个不同 id、每 id 一个重启 voice；振幅 10^(音量/2000)，DirectSound 正 pan 衰减左声道、负 pan 衰减右声道，portable PCM 包装流实现，Game.Update 帧尾 flush（`0x43943b`）。+6 只复制到 manager+0x408，本版无运行时读者，按原作保留闲置属性而不猜冷却／优先级 | `TestSEPan`、`TestSEPanGains`、`TestPanStreamSamples`、`TestPanStreamSeek`、`TestSEQueuePanAndAux`、`TestSEQueueLimits`、`TestPositionalSoundWiring`、`TestPlaySEPanning`（需用户启用） |
```

建议追加更新记录（实际日期 2026-11-07）：

- 2026-11-07 — t4：ECL 349–354 按 `FUN_0040e770` 的 `0x4118bb–0x411b6b` 移植，保留原作选择 p0／少一位参数的缺陷；354 的 `SSS` 格式核对，七关反汇编／IR 均出现 0 次。新增逐条边界、栈求值、特殊输出、整数溢出与格式测试。动态 rank 所有直接写与五个 AddRank 物理调用点核对；周期使用 Tick 后的状态计时，慢帧停在倍数时重复增长；修正下限钳位，新增新局／boss 练习 reset hook。六章 trace 与基线逐字节相同，golden 未改。
- 2026-11-07 — t4：备用 RNG `DAT_004918a8` 的唯一播种点为应用启动 `0x4201e1`，来源 `timeGetTime()` 低 16 位，关卡／回放不恢复。选择每局种子 0 的明确确定性替代，修正跨关重新清零，提供 `GameState.AlternateRNG()`。分别从种子 0／0x1234 在 STD／ANM 初始化前开始跑 th10_05 六章，完整 trace 与主 RNG state/count 全部相同，备用只改变该验收范围的演出。
- 2026-11-07 — t4：移植音效按 x 声像、同 id 128 请求平均、每帧 12 id、独立 id voice 重启；音量与 DirectSound 单侧 dB 衰减经 portable PCM 包装流实现，支持奇数／短读／Seek 与 WASM，主音量调整不再丢失表增益。+6 经全程序／PE 审计确认仅写未读，保留闲置属性，无臆造冷却。音效 flush 在 calc 之后，敌机／弹幕／激光／道具／灵击接线完整；固定样本与接线测试通过，六章 trace 不变，听音留用户。
- 2026-11-07 — t4：指定研究文档勘误就地更新，附 R10／R11 节号与 dump 地址；`FUN_0043bda0` 纠正为 SELECTARG1／DIFFUSE 忽略绑定纹理、DESTBLEND = INVSRCALPHA、入口 flush 与绘制后缓存失效；StateTimer Tick 定位后确认 Go tail 顺序相符，未改 player。21 份研究文档的 220 条缺口说法审计，过时清单只写入交付文档，正文是否改由架构师决定。全库 TestRect、38 个原有格式项、40 个原有 lint 项与 c0 旧参考 M 行问题仍按真实失败报告。

## 8. 需要用户目视确认的项目

未启用音频设备试听、未运行原作或打开游戏窗口。建议戴立体声耳机，在本工作树运行：

```powershell
. 'D:\GoProjects\th10\.tmp\codex\t4\env.ps1'
$env:TH10_TEST_AUDIO='1'
go test -count=1 ./audio -run '^TestPlaySEPanning$' -v
Remove-Item Env:TH10_TEST_AUDIO
```

顺序为 x = −192／0／+192／0：−192 时右声道比左低 10 dB，+192 时左声道比右低 10 dB，中心等量；应是另一侧仍可听到，而不是硬切。检查缓存 PCM 未污染、同 id 重启没有异常裂音。正常游戏可用真实存在的参数 `go run ./cmd -debug-stage 1 -debug-diff easy -debug-power 500`，听敌机发弹／击破的 x 声像与擦弹。没有自机 SHT 音效还需 t1 接 hook，不能据此误判本次声像流。

研究勘误与 rank／ECL 不需要额外目视确认；WASM 的真实浏览器音频输出仍需用户验证，编译已检查。

## 9. 与其他任务的接口

未修改 t1 的 `entity/player/`；状态计时器顺序的取证结论已写入 R10 与本交付文档，接线建议如下。

StateTimer：`FUN_00425730` 中 Tick 位于 ANM 步进与闪烁之后、rank／射击之前（`0x42618e–0x4261ec`）。当前 Go `tail()` 已相符；t1 增加本体 VM 时应把 Run 放在既有动画位置、Tick 之前，而不是改变计时顺序。

提供给 t5：`func (s *game.GameState) ResetRank(bossPractice bool)`。原作调用点为新局 `FUN_00417870` 的 `0x4179e2`。正常新局已经由 `NewState` 调用 false；若实现 `DAT_00474ca0 & 8` 的 boss 练习入口，应调用 true（−512），普通关卡练习不调用 true。关卡切换、回放头恢复之后不可调用 reset。

提供给 t1／t2／t3／t5：`func (s *game.GameState) AlternateRNG() *zun.RNG`，`Context.Anm.Alt` 已接入它。所有需要备用随机数的 ANM 对象应共享这个指针，不从回放章节种子派生、不在关卡切换清零。主菜单暂无 GameState 时如何保持“每局替代”的边界由架构师决定；原作是应用程序全局时间种子，而本任务明确选择每局重置 0。

提供给所有音效调用者：`func (c *stage.Context) PlaySEIDAt(id int, x float32)` 与 `func (m *audio.Manager) PlaySEIDAt(id int, x float32)`，x 是游戏坐标（0 位于场地顶边中点）。未提供此可选方法的现有无头 `Sound` recorder 回退到 `PlaySEID`，不破坏已有规格测试。t1 的 `FUN_00428160` 在 `0x42814d` 从 SHT sound 字段发音、x 取实际发射原点，应调用 At；`FUN_00426cf0` 的被弹 SE 4（`0x426e6f`）是居中 `PlaySEID`。t2／t3／t5 菜单／宣言的原作 `FUN_0043dc90` 调用应保持居中，不无依据地用自机 x。

本任务调用的外部范围原作函数保留原接线：道具等级变化仍调用 `FUN_00426f70`（`ItemEnv.Options = Player.RefreshOptions`）；符卡 `FUN_00405500`／奖励结算未实现或改写，灵击仍通过既有 `SpellCard.Bombed` 接口（t2）；节点 0x2e 只改研究描述，没有实现控制器或菜单（t5）。`Game.Update` 的音频 flush 移到当前场景 Update 后，请 t5 接管回放／菜单时保持每个实际逻辑帧只 flush 一次。

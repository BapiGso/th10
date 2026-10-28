# 真值台账（Fidelity Ledger）

每个子系统的还原状态、依据与验收方式。**每次改动必须更新此表**；golden 快照只在有意的
语义变更后重生成，且必须在提交信息中说明原因——禁止用更新 golden 掩盖分歧。

状态枚举：

- **位精确** — 逐位/逐值与原作一致，有二进制依据
- **有界漂移** — 浮点路径，短窗口 epsilon 内一致且不发散
- **近似** — 行为形状对，数值未取证
- **未做** — 缺失或占位

依据栏：函数地址（th10.exe 反编译）、数据表地址、trace/回放编号、文档路径。

## 关卡逻辑

| 子系统 | 状态 | 依据 | 验收 |
|---|---|---|---|
| ECL 容器/指令解码 | 位精确 | `thecl10.c`、`FUN_0044fdb0/0044fe40` | `scene/stage/ecl/loader_test.go` |
| ECL 虚拟时钟 | 位精确 | `FUN_0044e1a0`：wait(n) 令时钟倒退 n；返回 -1 的指令下一帧重试且时钟不前进；op 1 与栈空时的 op 10 结束任务，主任务随之删除敌机，异步任务只释放自身 | `TestVMClockTimingFirstWave`、`TestVMWaitDelaysOneFrame`、`TestVMBlockedOpRetries`、`TestVMReturnEndsTask` |
| ECL 栈机 opcode 0-93 | 位精确 | `FUN_0043ee30` | `vm_test.go` |
| ECL 敌机/移动 256-368 | 有界漂移（运动部分已移植，见"ECL 运动核心"） | `FUN_0040e770`；未覆盖：349–354；346（401 的禁射半径）已移植 | `TestVMAllStagesStableAndFire` |
| ECL 弹幕 400-436 | 位精确（移植，待差分）；激光近似 | `FUN_0040e770` case 400–0x1b4 逐条核对，参数求值顺序照搬（`docs/research/m2_bullets.md` §2）；管理器 `FUN_004073e0`/`FUN_004067d0`/`FUN_00406d90`/`FUN_00406240`/`FUN_004065c0`/`FUN_00408030`/`FUN_00408100`/`FUN_00408210` 移植为 `entity/bullet`（float32、游戏坐标、出生阶段、etEx 变换链、按尺寸出屏）；343 不消弹；激光管理器 `FUN_0041c510` 未移植，`entity/laser` 为第一版模型 | `emit_test.go`、`entity/bullet` 测试、golden |
| ZUN RNG | **有界漂移** | `FUN_0044bb90` 常量 PE 直读；种子取自回放章节头 +0x02（`GameState.RNGSeed`，无回放时仍为 `0x1234`）；ECL 与 ANM 引擎（`FUN_0043ee30`，子弹脚本）共用 `ctx.RNG`；备用 RNG `DAT_004918a8` 的种子未知，暂为 0 | `vm.go` rotl16 实现 |
| 缓动与插值器 | 位精确（移植，待差分） | `FUN_0044c350` 全部 mode、`FUN_00412ac0`/`FUN_00404610`（mode 7 累加、17 加加速度、8 Hermite）；x87 24 位精度按 float32 逐步取整 | `motion_test.go` |
| 出屏删除 | 位精确（移植，待差分） | `FUN_0040dc80`：按 0 号槽精灵尺寸（ANM op 3/50 推出）判定，进过屏幕（0x100）且未置 0x4 才删除；`cleanEnemies` 不再做 64px 通用删除；`stgenm02–07.anm` 已嵌入，各面 Boss 尺寸齐全 | `TestMotionOffscreen`、golden |
| 动态 rank | 近似 | `DAT_00474c98`：存于 `GameState.Rank`，跨关保留，回放章节头 +0x20 恢复；每 60 帧 +1 的规则未按 `FUN_00425730` 取证 | golden |
| 弹种尺寸表 | 位精确 | `DAT_004741e0` PE 直读 | `_data_tables.txt` |
| ECL 全关卡稳定性 | 有界漂移 | stage01-06.ecl | `TestGoldenStages`（3600 帧 FNV；golden 读取已兼容 CRLF 检出） |
| ECL Boss 阶段中断 334/341 | 位精确（逻辑） | `FUN_00412720`、`FUN_004127a0`、`FUN_0040dc80`（`scene/stage/ecl/interrupt.go`）：8 个槽 {HP、时间、子程序、超时子程序}；HP 触发在受伤后检查并当场执行新子程序，时间触发在帧末检查、下一帧生效（HP 置为槽值，置 0x10000，信仰 −3000、下限 5000，本项目 ×10）；中断释放该敌机的异步任务，重置 IP/时钟，保留调用栈；阶段计时器 333 重置；335 无敌计时；332 Boss 槽，340 只等 0 号槽；创建指令当场执行新敌机的 time-0 指令 | `interrupt_test.go`（HP/超时触发、BossDead 0 HP 存活、道中 Boss 随阶段死亡、1–3 面不射击全程超时通关） |
| ECL 运动核心 | 位精确（移植，待差分） | `FUN_0040dc80`/`FUN_0040e770`：两层位置（A/B）并行、每层与总和 0.01 取整（`FUN_0044c2a0`）、绕圆、移动边界、flip 只镜像 284–287 的角度、"未设置"哨兵 −999999；常量按 .rdata 位模式 | `motion_test.go`、`docs/research/m2_motion.md` §4–§7 |
| ECL 特殊变量 | 位精确（移植，待差分） | `FUN_0044fe40`/`FUN_0044fdb0` 及敌机 getter：完整表见 `m2_motion.md` §8；I0–I3/F0–F3 共用 8 个 dword；参数求值顺序照搬（影响 RNG 消耗） | `scene/stage/ecl/special.go` |
| 敌弹槽位分配 | 位精确 | `FUN_004067d0`：2000 槽，从游标环形搜索空槽，成功后游标指向下一槽 | `entity/bullet`（`Pool.spawn`） |
| 自机被弹/擦弹 | 位精确（移植，待差分） | `FUN_004266b0`：子弹盒 ±W/2 对自机盒（+0x404…+0x414，边长灵梦 2.0、魔理沙 3.5，`0x476fb8`）；擦弹盒 ±24 无条件判定；状态 2/3/4 不命中，无敌时只消弹（`m2_bullets.md` §4–§5）；GUI+0x9eb8 按 0 处理 | `stage.Context.bulletHit`、`entity/bullet` 测试 |
| 音效表 | 位精确（音量、去重、每帧上限） | `0x4749c8`（47 项）与 wav 名表 `0x474b40`：振幅 10^(音量/2000)，每帧同编号一次、最多 12 个；+6 字段与按 x 声像未实现 | `audio/se.go` |
| ECL 子程序命名空间 | 位精确 | `FUN_00450470`：stage ECL 与其 ECLI 列表（`default.ecl`：`Ecl_EtBreak` 等）共用一张子程序表；找不到的子程序令敌机在下一次执行时被删除 | `TestStageVMIncludesDefaultECL` |
| 敌机逐个更新顺序 | 位精确（移动+ECL）；伤害仍统一结算 | `FUN_0040d750`/`FUN_0040dc80`/`FUN_0040cfb0`：按链表逐个"移动→ECL"，0x400 标志与"先取 next"时序照搬（尾部敌人新建的敌人少更新一帧）；新敌人先首次更新再入链表；main 跑在隐藏敌人上；伤害、中断、计时器仍在 `LateUpdate` 统一执行（`docs/research/m2_motion.md` §6、§11） | `interrupt_test.go`、golden |

## 自机

| 子系统 | 状态 | 依据 | 验收 |
|---|---|---|---|
| SHT 移动参数 | 位精确 | `FUN_00426520`、`FUN_004250b0` | `sht/movement_test.go` |
| 移动边界 | 位精确 | `FUN_004250b0` | `player_test.go` |
| 灵梦 A 射击组/发弹 | 有界漂移 | `FUN_004281d0/00428160`、`pl00a.sht` | `reimu_a_test.go` |
| 灵梦 A 诱导 | 有界漂移 | `0x428ad0/00428b10/00428ce0` | `reimu_a_test.go` |
| 副机布局/插值 | 有界漂移 | `0x476f7c`、`FUN_00426f70` | `reimu_a_test.go` |
| 其余 5 机体射击 | 近似 | 旧实现，未接 SHT 组表 | 待 M1 |
| Power 资源规则 | 位精确 | `FUN_00425730`、`FUN_00418930` | `player_test.go`、`power_test.go` |
| 决死窗口 | 位精确 | `FUN_00425730` state 4 | `player_test.go` |
| 死亡 Power 损失/掉落 | 有界漂移 | `FUN_00425730` state 2 | `player_test.go`（延时/轨迹近似） |
| 灵击生命周期 | **近似** | `FUN_00405750/00405860/004055c0`（已 dump 未实装） | 待 M1（现为 300 帧占位） |
| 自机完整状态机顺序 | 近似 | `FUN_00425730` caller 链 | 待 M1 |

## 验证基础设施（M0，2026-10-04 建成）

| 组件 | 状态 | 依据 | 验收 |
|---|---|---|---|
| .rpy 回放解析 | 位精确（解码）；方向位已按 `FUN_004250b0` 更正 | `FUN_00428f60/0042a200/004297d0/0044b0d0/00435dc0/004250b0` | `replay_test.go`（4 demo + trailer 交叉验证、章节步长、方向判定）；另有 5 个网上回放，见 `docs/research/replays.md` |
| trace 格式 | v2 | `trace/`：坐标统一为原作游戏坐标；敌人按管理器链表顺序（含 main），子弹按槽位顺序 | `th10diff` 测试 |
| 无头运行器 | 位精确（确定性） | `sim/`、`stage.NewHeadless` | `TestHeadlessDeterministic`（逐字节一致） |
| 原作只读采集 | 待动态验证 | `tools/th10probe/`：敌人改为遍历 `DAT_00477704`+0x58 链表（旧版误用子弹管理器地址），新增 2000 槽子弹采样；按帧计时器编号，等待 3 ms 后采样并丢弃跨帧样本 | 需运行原作（尚未跑） |
| 差分器 | — | `tools/th10diff/`：按帧号对齐（跳过漏采帧），`-shift`、`-scope motion` | `main_test.go` |
| 差分闭环 | **未打通** | 复刻侧：`th10sim` 已能跑 th10_05 一面（16031 帧）；原作侧 trace 尚未采集；约第 4777 帧起复刻进入对话冻结（M5），目前只比较道中 | 原作侧 trace 尚未采集 |
| th10.dat（THA1）读取 | 位精确 | `datfs/`：头部/文件表解密 + LZSS，移植自 thtk `thdat95.c`；全部条目与独立解包逐字节一致 | `datfs/datfs_test.go`（`TestRealArchive`） |
| thbgm.dat BGM 循环点 | 采样对齐（有损编码） | thbgm.dat 的 44.1 kHz PCM 逐采样编码为 Ogg Vorbis（嵌入 `assets/wav/bgm/`）；循环点取自 `thbgm.fmt`（52 字节记录：Intro/Size）；旧手填 `TH10BGM` 表的 idx 10、16 循环点有误 | `audio/audio_test.go`（`TestBGMLoopSeam`、`TestBGMMatchesThbgm`） |
| THTX 贴图解码 | 位精确 | `anm/texture.go`：格式 1/3/5（BGRA8888/RGB565/ARGB4444），与旧 PNG 对比 8888/4444 逐位一致；565 按 D3D 位复制展开（thanm 的展开方式与此相差 ≤7）；嵌入的 565 贴图（50 个 PNG）已按此方式重绘 | `anm/loader_test.go`（`TestEnemyTexture`） |
| 资源嵌入 | 位精确 | `gamedata/` 只读嵌入资源；`tools/th10extract` 从 th10.dat 原样提取（`datfs.EmbedPath` 布局），按里程碑逐步加入 | `gamedata/embed_test.go`（`TestEmbeddedMatchesDat`：76 个原始文件逐字节、219 张贴图逐像素；无原作数据时跳过） |

## 表现层

| 子系统 | 状态 | 依据 | 验收 |
|---|---|---|---|
| ANM 容器/精灵表 | 位精确 | `anm/loader.go`；贴图可由 THTX 直接解码（`anm/texture.go`），嵌入的解码 PNG 与之逐像素一致 | `loader_test.go` |
| ANM 时间线子集 | 有界漂移 | `anm/resolve.go`（ins_0/1/3/4/5） | `loader_test.go` |
| ANM 完整 VM | 移植（逻辑；绘制待 M3） | `FUN_0043ee30` 全部 case、重置 `FUN_00401de0`、效果列表 `FUN_00448860`/`FUN_00448db0`（优先级 0x1a、尾部规则）、游戏/备用 RNG 选择（`docs/research/m2_bullets.md` §9）；目前驱动子弹与效果，敌人、自机、背景仍走旧路径 | `anm/vm_test.go`、`anm/vm_internal_test.go` |
| STD 背景 | 近似 | `stage01.std.txt` | 待 M2 |
| 子弹/敌人贴图 | 子弹：移植（待截图对比）；敌人：近似 | 子弹由 `bullet.anm` 的 ANM VM 驱动，6 个图层（`0x474250`）、加法混合，`sprite.DrawVM`；敌人仍为旧的 `sprite/anm.go` 回放 | 待 L3 |
| 对话 .msg | **近似** | `scene/dialog/msg.go`；仅从 .msg 中提取文本（op 7/8/12/13/16），MSG 解释器 `FUN_00415e90`（43 个 case）未移植，时序、op 10 超时、op 11 续播、op 18/19/20 均被忽略 | `msg_test.go` |
| Boss HUD/符卡名 | **近似** | 数据位精确：血条取 0 号 Boss 槽，计时取 `FUN_004127a0` 写入的 `min(99, (time−cur+59)/60)`，符卡名/奖励由 ECL 357/359 宣言（Shift-JIS，读自 ECL），1–6 面与 Extra 均已接 HUD、对话桥与 Boss BGM；演出（ANM 名牌、cut-in、符卡背景、`ascii.anm` 数字）仍为近似 | `TestVMSpellBgDeclaresHUD`、`TestVMInterruptOnHP` |

## 外围

| 子系统 | 状态 | 依据 | 验收 |
|---|---|---|---|
| Extra 关 | 近似 | `scene/stage/extra/extra.go` 经 ECL VM 运行 `stage07.ecl`；演出（表现层）为近似 | 待 M3 |
| 回放录制/播放 | 未做 | — | 待 M3 |
| 菜单/结局/Staff | 近似 | — | 待 M3 |
| 存档/配置 | 近似 | `game/persistence.go` | — |

## 更新记录

- 2026-10-04 — 建表；M0 差分脊柱建成（replay/trace/sim/th10diff/th10probe）。
- 2026-10-26 — 审计修正：RNG/出屏/.msg/Boss HUD 降级，Extra 依据更正；详见 `docs/research/`。
- 2026-10-26 — M0 资源管线：新增 `datfs/`、`gamedata/`、THTX 解码；ECL/ANM/MSG/SHT/SE/BGM 的资源入口统一到 `gamedata/`；音频改 44.1 kHz，移除 Opus 依赖；BGM 循环点改用 `thbgm.fmt`（修正旧表 2 首错误）。运行时读取原作数据的做法已于当日撤回，见下条。
- 2026-10-26 — 资源改回嵌入：不再运行时读取 `th10.dat`，原作资源经 `tools/th10extract` 原样提取后 `go:embed` 进二进制（游戏不再需要 `-th10-dir`/`TH10_LANG`）；SE 改为原版 WAV，BGM 改为 thbgm.dat PCM 逐采样对齐编码的 Ogg Vorbis；50 张 RGB565 贴图按 D3D 展开重绘（差异 ≤7）；`gamedata`/`audio` 校验测试对照本地原作数据，无数据时跳过。
- 2026-10-27 — M1 Boss 阶段：334/341 改为 8 槽中断（替换主子程序，不再追加任务），332 Boss 槽、333 阶段计时、335 无敌、340 只等 0 号槽；ECL 时钟改为原作模型（wait 倒退时钟、-1 阻塞重试、op 1/10 删除敌机）；创建指令当场执行新敌机 time-0；嵌入 `ecl/default.ecl` 并合并进子程序表；未设 ANM 的敌机不绘制（`Enemy.Hidden`）；各关包装器转发 `LateUpdate`。golden 因上述有意语义变更重生成（敌机数在 ±1 内，弹数在若干窗口变化，如 4 面 1800 帧 1360→1534、6 面 3000 帧 245→495）。依据与遗留差异见 `docs/research/m1_interrupts.md`。
- 2026-10-27 — 回放解析修正：方向键原为猜测的"2-of-4 编码"，按 `FUN_004250b0`/菜单代码改为 0x10 上、0x20 下、0x40 左、0x80 右，冲突组合按原作判定顺序解析（旧解码会让 sim 中的自机走错方向）；章节遍历改按全局头 +0x4c 个数与章节头 +0x08 正文长度（FUN_0042a200），修复 +0x04 帧数偏大的回放只能解析出第 1 面的问题。下载了 5 个 v1.00a 回放（含灵梦 C Easy 全程不射击的 1–6 面），见 `docs/research/replays.md`。
- 2026-10-27 — M2 运动核心：敌人更新改为按管理器链表逐个"移动→ECL"（`FUN_0040d750`/`FUN_0040dc80`，含 0x400 时序与"先更新后入链表"），关卡 main 跑在隐藏敌人上；任务时钟改为按游戏速度推进的 float；运动改为两层并行 + 0.01 取整 + 完整缓动/插值器；特殊变量表补全；出屏删除交给 ECL（按精灵尺寸），删除 `cleanEnemies` 的 64px 通用删除；RNG 种子与 rank 取自回放章节头；敌弹池改为 2000 槽游标分配。嵌入 `stgenm02–07.anm`。差分闭环：trace v2（游戏坐标、链表顺序）、th10probe 地址更正并采样子弹、th10diff 帧号对齐与 `-scope motion`、th10sim 按章节取关卡号。golden 因上述有意语义变更重生成（例如 5 面 600 帧弹数 133→321：`Cir00_at` 普通难度每敌 3 发、30–89 帧一轮，旧版只在统一阶段跑脚本、子弹数偏低）。th10_05 一面在约第 4777 帧进入对话，复刻会冻结世界（M5 修正），差分目前只能覆盖道中。
- 2026-10-28 — M2 子弹：ECL 400–436 按 `FUN_0040e770` 逐条重写（`bullet_ops.go`，参数求值顺序照搬，16 个 `bullet.Template` 模板与发射偏移，346 禁射半径，422–427/435/436 的 rank、难度分档），接上 float32 子弹管理器 `entity/bullet`；343 不再消弹（各关脚本在 343 前自行 `etClear`/`ins_410`）；367 同步到 ANM 游戏速度；ECL 音效改走原作音效表 `0x4749c8`；自机被弹/擦弹按 `FUN_004266b0`；子弹由 `bullet.anm` 驱动绘制；激光移到 `entity/laser`（仍为近似，加句柄供 413/414–419）。各关反汇编统计：414–419、422–424、426/427、429/430/432/434 从未使用，409 下标只用 0–7（`docs/research/m2_bullets.md` §3）。golden 因有意语义变更重生成：子弹改为游戏坐标 float32 并带出生阶段与按尺寸出屏；ECL 改用上下文 RNG（与子弹 ANM 共用，种子 0x1234）；子弹会真正命中自机。一面各检查点敌人数不变，600 帧哈希不变（尚无子弹）。

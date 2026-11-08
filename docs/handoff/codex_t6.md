# t6：自机系统续作交付记录

更新时间：2026-11-08 04:51 CET。工作树：`D:/GoProjects/th10/.tmp/codex/t6/tree`。

## 1. 摘要

| 阶段 | 状态 | 内容 |
|---|---|---|
| A．副机与自机子弹 World VM | 完成，rules 第 6 节验收全通过 | 六种布局的副机、满火力附加 VM、魔理沙 A 历史回调、魔理沙 C 锁定回调；灵梦 A 飞行与命中 VM；取消 0x16 的 legacy。 |
| B．矩形门控与回放自机状态 | 完成，rules 第 6 节验收全通过 | 接入 cfg／Boss／GUI 对话／菜单门控；33 项历史及四组副机对直接 capture／restore；录制与播放共用。 |
| C．其余五种机体射击 | 实现完成，验收未全绿 | 六份 SHT 共用原 burst／回调／kind 0、2、3 运行时；14 个新增测试与 28 章 CLI 均通过；两个不可修改的旧近似规格失败。 |
| D．灵击演出余项 | 实现完成，验收未全绿 | 接原灵击／死亡火力数字更新，补两个道具路径的 0x189；核对其余原视觉路径已由 ANM 执行；仅 C 的两个旧规格继续失败。 |

阶段 A、B、C、D 及最终交付复验的五轮六章 trace 均与基线逐字节相同，没有运动一致帧下降。各阶段及交付前均已实际跑完整 smoke、构建／vet／测试／WASM／格式检查；C／D／最终的全仓测试仍因两个旧近似规格未绿，所以**不宣称任务完成，不写 `DONE`**。没有运行原作、probe、Ghidra 或数据目录内的程序；未改主树源码、原有 golden、`docs/fidelity.md`。

## 2. 改动文件清单

新增：

- `entity/player/options.go`：四个实时副机槽、33 项定点历史、六种布局及 World 句柄。
- `entity/player/options_test.go`：副机生成顺序、层参数、中断、历史与锁定回调测试。
- `entity/playerbullet/anm.go`：射击 VM 生成、命中火花、精灵尺寸出屏与失效句柄回收。
- `entity/playerbullet/anm_test.go`：飞行／命中生命周期、旋转、层号、备用 RNG 测试。
- `entity/player/replay_test.go`：完整章头布局、六章 roundtrip、首帧不吸附、魔理沙 C 特殊恢复测试。
- `scene/stage/player_replay_test.go`：实际矩形门控绘制、Stage hook、现场下一章 capture 与原文件写出测试。
- `entity/player/shots_test.go`：六机体／十组、慢放 burst、激光单源门控与对话限制测试。
- `entity/playerbullet/callbacks.go`：初始化回调 1、更新回调 1／2 的逐函数实现。
- `entity/playerbullet/sht.go`：共用射击运动、网格、计时、激光结束和命中几何。
- `entity/playerbullet/callbacks_test.go`：各回调、负残差网格、激光、爆破区域与音效测试。
- `sht/runtime_test.go`：六份原 SHT 所有回调表索引及未知语义拒绝测试。
- `entity/player/power_visual_test.go`：扣火力／刷新副机后、扣信仰前的调用顺序，以及死亡掉落前更新火力图。
- `entity/item/effects_test.go`：0x189 的旧槽 kind 分支、满火力转换、位置／层／顺序／寿命和 nil 环境。
- `scene/stage/bomb_visual_test.go`：真实 GUI 火力图的前后更新，以及原 ANM 初始／定时子 VM 的接线验证。
- `docs/handoff/codex_t6.md`：本交付文档。

修改：

- `entity/player/player.go`：六份 SHT 布局共用副机实时结构，移动更新历史，死亡中断两组副机句柄。
- `entity/player/anm.go`：连接自机子弹的 ANM 环境并初始化副机。
- `entity/player/reimu_a.go`：副机状态移到 Player，射击执行改为六机体共用的原 Timer burst。
- `entity/player/replay.go`：实时历史／副机 capture 与 restore，恢复后执行原初始化时钟／判定点路径。
- `entity/playerbullet/playerbullet.go`：区分活动／命中演出／空闲槽，命中后等待 World VM 结束。
- `scene/stage/chain.go`：0x16 改为直接自机 VM 节点，不再依赖 Ebiten back buffer。
- `scene/stage/chain_legacy.go`：删除旧自机节点及不再使用的 legacy 裁切助手。
- `scene/stage/stage.go`：删除旧自机子弹／副机绘制函数与对应导入。
- `scene/stage/chain_test.go`：仅授权的一处，将 legacy 列表中的 0x16 移出，列表为空。
- `scene/stage/bullets.go`：安装自机 capture／restore hook，恢复后调用既有 `GUI.SetLives`。
- `sprite/loaders.go`：删除不再有调用者的 `Global.ReimuAnm`，不再重复加载旧 ANM sheet。
- `sht/shot.go`：新增六机体已知回调／kind 验证，明确后两张零占位表，不添加未知回调近似。
- `entity/item/item.go`：原两处 0x189 使用同一 World／bullet.anm，不改道具逻辑和槽分配。
- `scene/stage/bomb.go`：只在原位置接 `GUI.SetPower`，`c.Spell.Bombed()` 和其后符卡相关语句位置／顺序不变。

删除文件：无。

共 14 个修改文件、15 个新增文件（包含交付文档），无删除；新增 Go 测试共 41 个。`go.mod`、`go.sum`、`anm/world.go`、`docs/fidelity.md` 与 base 的 SHA-256 相同。工作树没有 `.git`，仅用只读 `git diff --no-index` 核对授权测试改动与共享 `bomb.go` 的原语句顺序。

## 3. 移植的函数地址

| 原作函数／指令 | Go 对应 | 内容 |
|---|---|---|
| `FUN_00426f70` | `Player.RefreshOptions` | 先处理满火力附加 VM，再检查副机数；主 VM 尾插，附加 VM 头插，生成层参数均为 15。 |
| `0x427206–0x42723f`、`0x427873–0x4278f0` | `Player.RefreshOptions` | 保留槽旧主 VM 打 FlagDelete；减少的槽中断标签 1。 |
| `FUN_004250b0`，`0x425521–0x4255b3` | `Player.updatePositionHistory` | 非低速且原始方向步长非零时移动历史，尾部重复末副机的终端样本。 |
| `FUN_004250b0`，`0x4255b9–0x4256d8` | `Player.updateOptions` | 回调、定点插值、依次 Place 两个句柄。 |
| `FUN_00427950` → `FUN_00427960` | `Player.trailOption` | 魔理沙 A 的位置历史回调；七项权重为 1/8 至 7/8。 |
| `FUN_00427ad0` → `FUN_00427ae0` | `Player.lockOption` | 魔理沙 C 的标签 3／6 与绝对位置锁定；不是绘制回调。 |
| `FUN_004250b0`，`0x4251a0–0x4251d0` | `Player.move` 的 `optionEase` | 魔理沙 C 解除低速后 30 tick 才恢复插值。 |
| `FUN_004269d0`，`0x426b0b–0x426c03` | `Player.clearOptions` | 死亡时清逻辑数量，两组 VM 中断 1，不立即删除。 |
| `FUN_00427e90`，`0x42801a–0x4280ca` | `Pool.spawnShot` | 尾插原 pl0N 飞行脚本，以 15 传层参数、保留 AltRNG。 |
| `FUN_00428280`，`0x4283ca–0x428577` | `Pool.prep` | 查句柄、按 VM 精灵缩放尺寸出屏、写位置与自动旋转。 |
| `FUN_00428d70` | `Pool.prep` 的出屏测试 | age ≥ 10，边界 −192／192／0／448；R6 的运行时 +0x34／+0x30 分别为 W／H，已在 C 校正适配。 |
| `FUN_00428630`，`0x4287d1–0x428832` | `Pool.hit` | 保留旋转，尾插命中脚本，速度乘 0.125，z＝0.1，状态 2。 |
| `FUN_00426360` | `Stage.nodePlayer`、`Stage.playerFaithBarGate` | 节点 0x16 只画嵌入自机 VM 和矩形；读取隐藏 cfg 位、Boss 槽、GUI 对话和菜单状态。 |
| `FUN_0042a450`，`0x42a512–0x42a56a` | `Player.CaptureReplayPlayer` | +0x2c 的 33 项历史；+0x134／154／174／194 的四组副机对。 |
| `FUN_0042a450`，`0x42a5d2–0x42a687` | `Player.RestoreReplayPlayer` | 先历史与既有 focus，再 RefreshOptions，再四组对，清吸附标志；魔理沙 C 的 slow home 特例。 |
| `FUN_00424d90` | `Player.RestoreReplayPlayer` 尾部、Context restore hook | 复位状态／射击／初始化时钟，删判定点，保持无敌时钟，并调 `GUI.SetLives`。 |
| `FUN_00426520`，`0x4265b0–0x4265e1` | `sht.File.ValidateTH10` | 初始化表 0／1，更新表 0／1／2，后两张表只接受原 SHT 的 0 索引。 |
| `FUN_004281d0`／`FUN_00428160` | `Player.updateShot`／`Player.fireSHT` | 六份组表共用原 burst Timer，按 Cur≠Prev 发弹，慢放不重复同一相位。 |
| `FUN_00427e90` | `Pool.FireSHT` | 128 槽、来源、初速度补偿、初始化回调与原 SE，激光按源门控。 |
| `FUN_00428ad0` | `Bullet.initHoming` | 回调 1 仅复制目标；魔理沙 A 火箭只有该初始化回调，不因此变成诱导弹。 |
| `FUN_00428b10`／`FUN_00428ce0` | `Bullet.tickHoming` | float32 短角差分组、转向、速度阈值、120 tick 后无上限加速、命中态跳过。 |
| `FUN_00428c20` | `Pool.laserMuzzle` | 更新回调 2，取当前副机定点坐标，减旧速度后由共用运动再加速度。 |
| `FUN_00428280` | `Pool.updateSHT`／`Pool.endLaser` | kind 3 的时钟／副机数／MSG／管理器门控，两 VM 中断 1、源立即释放；命中高亮与下一次无命中变暗。 |
| `FUN_0044c2a0`／`FUN_0044c5d0` | `shotGrid`／`zun.Polar` | 游戏坐标 float32 运动与 floor(v×100)×0.01；速度不乘游戏速度，只有 Timer 按 gs 走。 |
| `FUN_00428630`／`FUN_00427c70`／`FUN_00427b50` | `Pool.HitEnemy`／`shotOverlap`／既有 `Player.addArea` | 端点包含，普通弹和激光不同的顶边门控；激光每四 tick 伤害且不销毁；kind 2 生成 32／1.4／13／damage÷3 的区域。 |
| `FUN_00405860`，`0x405947–0x405975` | `Bomb.Start` → 既有 `GUI.SetPower` | 原 SE 后、符卡失收前更新旧 Power 数字。 |
| `FUN_00425730`，`0x425a10`／`0x425af7`／`0x425c7f` | `Player.refreshPowerGauge` → `PowerChanged` | 普通／决死灵击扣 Power 和刷新副机后；死亡第 3 tick 扣 Power 后、掉落前。 |
| `FUN_0041bb00`，`0x41bc23–0x41bc42` | `Env.faithEffect` 的 Spawn 调用点 | 判断槽的旧 kind 3，在覆盖新 kind 前以新槽位置生成 0x189。 |
| `FUN_0041ba50`，`0x41bad2–0x41badd` | `Manager.fullPower` → `Env.faithEffect` | 每个 1／4 号道具转换后生成 0x189，10／11 不转换。 |
| `FUN_00448db0` | 既有 `World.Spawn` | 0x189 位置来自 EDI 指向的道具位置；栈上的 out 参数不是位置向量，生成首帧执行原脚本。 |

取证使用既有 `r8/func_426f70.asm`、`r10/func_4250b0`、`r10/func_427960.asm`、`r10/func_428280.asm`、`r10/func_428630` 与原 `pl00.anm`／`pl01.anm`。研究文档 R10 的摘要把标签 3／6 归给灵梦，实际 `0x427254–0x42737c` 的分支为字符 1、射击 2，即魔理沙 C；实现按汇编。`0x427a77` 加的是留在 FPU 栈上的权重，不是 y 坐标，已闭合原未决点。

## 4. 测试

新增 11 个测试：`TestOptionWorldScriptsAndPlacement`、`TestOptionFullPowerFrontOrderAndLayerArgument`、`TestOptionRefreshSameCountAndDrop`、`TestOptionDeathInterruptsBothHandles`、`TestPositionHistoryMovementGates`、`TestMarisaATrailInterpolationWeights`、`TestMarisaCLockInterruptsAndReleaseDelay`、`TestShotWorldTailAndRotation`、`TestShotHitSparkKeepsSlotUntilWorldEnd`、`TestShotSpriteCullAndStaleHandle`、`TestShotSpawnLayerArgumentAndAlternateRNG`。

阶段 B 新增 8 个测试：`TestReplayPlayerPairLayoutAndClockReset`、`TestReplayPlayerOriginalChaptersRoundtrip`、`TestReplayOptionsDoNotSnapOnFirstMove`、`TestReplayMarisaCLockHomeNormalisation`、`TestReplayPlayerShortHeader`、`TestPlayerFaithRectangleStageGates`、`TestStageReplayPlayerHookRoundtrip`、`TestRecordingNextChapterCapturesLivePlayerFields`。原 `sim.TestRecordTh10_05AllChapters` 实际重录六章，输入三字、slow-rate、重复保存后六份 452 字节章节头全部一致；本次历史／副机已来自真实结构，不再靠保留 Raw 通过。新现场录制测试经 `replay.Encode`／`Decode` 检查下一章写出的原字段，旧章头被冻结，不被后续移动覆盖。

矩形测试包含隐藏位打开时两个真实三角带、默认 cfg、无 Boss 管理器／GUI／菜单、Boss 占槽、对话、结算菜单、保持计时零和状态 2；host 暂停不等于原菜单状态，Frame＝0 也不附加信仰衰减的 90 帧门槛。

阶段 C 新增 14 个测试，逐个回调均覆盖；六机体、0.00 至 5.00、两种 focus 共 144 个组合的首相位执行原 SHT。激光覆盖双句柄、每源一束、四 tick 伤害、亮／暗、中断退出及四个结束原因；爆破覆盖区域的参数与调用顺序；普通弹覆盖原包含端点的矩形和 y＝0 门控；验证微小负残差必须落到 −0.01。

C 测试名：`TestAllTH10ShotCallbackTables`、`TestRejectUnknownTH10ShotSemantics`、`TestAllSixSHTGroupsFireAtOriginalPhase`、`TestSHTClockSlowFramesDoNotRepeatShots`、`TestMarisaBLasersStayOnePerSourceAndEndWithBurst`、`TestSHTNoAttackResetsBurstButMovesExistingShots`、`TestInitCallbackOneDoesNotImplyHoming`、`TestHomingCallbackHitStateAndShortestWrap`、`TestHomingBirthGridNegativeZeroSide`、`TestLaserMuzzleCallbackUsesCurrentSource`、`TestLaserSourceGatesAndEndReasons`、`TestLaserDamageFourTickGateAndBrightness`、`TestKindTwoHitAddsOriginalDamageArea`、`TestSHTSoundAndInclusiveTopGate`。

阶段 D 新增 8 个测试：`TestBombPowerGaugeHookAfterRefreshBeforeFaithLoss`、`TestDeathPowerGaugeHookBeforeDrops`、`TestFaithEffectUsesPreviousSlotKindAndClampedPosition`、`TestFullPowerEffectsKeepItemConversionAndSlotOrder`、`TestFaithEffectEndsThroughANMWithoutRNG`、`TestFaithEffectNilAnimationAndCancelPath`、`TestBombPowerGaugeBeforeAndAfterCallerCost`、`TestBombVisualsStayInOriginalANMScripts`。灵击覆盖 SE／旧数字／失收／扣火力／副机／新数字／扣信仰顺序；ANM 验证灵梦／魔理沙／重型脚本原子 VM 与第 10 帧子 VM，禁止另造手写特效；0x189 仅原层 11、时间 20 结束、不消耗 RNG。

验收日志：`D:/GoProjects/th10/.tmp/codex/t6/logs/phase_a/`、`phase_b/`、`phase_c/`、`phase_d/`。

| 检查 | A | B | C | D |
|---|---|---|---|---|
| `go build ./...` | exit 0。 | exit 0。 | exit 0。 | exit 0。 |
| `go vet ./...` | exit 0。 | exit 0。 | exit 0。 | exit 0。 |
| `go test -count=1 ./...` | 全通过。 | 全通过。 | exit 1，两个旧规格。 | 同样仅两个旧规格。 |
| `gofmt -l .` | 空。 | 空。 | 空。 | 空。 |
| WASM 构建，排除 tools | exit 0。 | exit 0。 | exit 0。 | exit 0。 |
| `TH10_SMOKE=full`，`TestReplayDrawSmoke` | 24 章，138.395 秒。 | 24 章，117.279 秒。 | 24 章，130.691 秒。 | 24 章，117.284 秒；全程通过。 |
| `golangci-lint run --timeout 10m` | 33 项。 | 33 项。 | 33 项。 | 33 项，无本任务文件报告。 |

lint 分类为 errcheck 26、staticcheck 2、unused 5。rules 明确不以 lint 作为 DONE 门控，仍按全局 AGENTS 要求实际执行并报告。WASM 先在 host 枚举包，避免 Windows-only probe 的枚举错误。原有测试、golden 未改；只有授权的 legacy 列表一处变化。

最终交付复验日志：`.tmp/codex/t6/logs/phase_final/`，2026-11-08 04:46～04:50 CET。build／vet／WASM＝0，gofmt 空；全仓 test＝1，仍仅两个下述旧规格失败；完整 smoke 24 章通过，123.619 秒；六章 cmp／motion 全部通过。lint 仍为 33 项，逐个核对其报告所涉及的 18 个文件，均与 base 逐字节相同，无新增 lint 问题。大 trace 目录已空，不保留 cN 或 sanity 文件。

开发中新加的魔理沙 C 测试最初把锁定目标误当作当帧插值后的坐标；原回调在插值前保存上一帧位置。新测试增加一帧静止以观察稳定锁定，已通过，未改任何既有断言。

C 的失败规格保持原样，文件 SHA-256 与 base 一致：`F89CF228237C7279AE003470D0E0D47A8630218ACA55B0DB2BC2247ED5BCF494`。

- `TestOtherShotsRemainLegacy`，`entity/player/reimu_a_test.go:157`：`legacy first-frame cadence changed`。原 `0x428201` 以 Timer.Set(0) 起枪，`0x428223` 当帧执行组表；其余五机体应和灵梦 A 一样在 SHT 相位 0 发射，不再保持旧第三帧才发弹的近似。建议授权改为 SHT 首相位断言，新 `TestAllSixSHTGroupsFireAtOriginalPhase` 已锁定全部组合。
- `TestReimuAPowerLevelsAndBirths`，`entity/player/reimu_a_test.go:69`：观察自机横坐标 224 的诱导弹首帧为 `223.99000000022352`，旧期望为 224。原 `FUN_00427e90` 的初速度补偿后，更新回调把速度 12 加为 float32 12.1，cos(−float32 π/2) 产生微小负横向余量；`FUN_00428280` 在 `0x4283c5` 调 `FUN_0044c2a0`，floor 后为游戏坐标 −0.01（`0xbc23d70a`）。建议授权按同一 float32 分组和网格算精确期望，不放宽容差。新 `TestHomingBirthGridNegativeZeroSide` 已锁定此原作边界。

阶段 C 合理性检查：实际运行 4 个 demo 与 th10_01～09 的全部 28 章 `go run ./tools/th10sim -frames 0`，每章 exit 0，无 panic。独立 sim 观察者再次顺序运行，28 章末帧／得分与 CLI 一致。大文件已逐章删除，保留 `replay_cli_results.csv`、`replay_stats.csv`。下表“HP 观察”统计见过的敌机转为 inactive 且 HP＜1、非 0x40；包括 HP 归零的超时删除，可能漏掉同帧新生同帧死亡，不把它冒充精确原作击破数。

| 回放 | 章 | 帧数 | 末得分 | HP 观察 | 残机／火力 | 自机弹峰值 |
|---|---:|---:|---:|---:|---|---:|
| demo0／th10_01，灵梦 B | 0 | 7824 | 13185750 | 174 | 0／0 | 64 |
| demo1／th10_02，灵梦 C | 0 | 6360 | 1490960 | 145 | 0／105 | 50 |
| demo2／th10_03，魔理沙 B | 0 | 7032 | 1680760 | 134 | −1／30 | 18 |
| demo3／th10_04，魔理沙 A | 0 | 6981 | 3139510 | 122 | −1／0 | 46 |
| th10_05 | 0 | 15972 | 1090 | 1 | 2／0 | 0 |
| th10_05 | 1 | 19831 | 2760 | 1 | 2／0 | 0 |
| th10_05 | 2 | 31699 | 10990 | 1 | 2／0 | 0 |
| th10_05 | 3 | 31301 | 18810 | 1 | 2／0 | 0 |
| th10_05 | 4 | 40667 | 447650 | 14 | 1／340 | 0 |
| th10_05 | 5 | 50288 | 1655940 | 18 | 0／120 | 0 |
| th10_06 | 0 | 16132 | 1250 | 1 | 2／0 | 0 |
| th10_06 | 1 | 19941 | 3290 | 1 | 2／0 | 0 |
| th10_06 | 2 | 32665 | 12760 | 1 | 2／0 | 0 |
| th10_06 | 3 | 32093 | 616940 | 87 | −1／0 | 0 |
| th10_06 | 4 | 41023 | 93200 | 31 | 1／0 | 0 |
| th10_06 | 5 | 50304 | 234740 | 29 | −1／0 | 0 |
| th10_07 | 0 | 60000 | 70900 | 15 | −1／0 | 0 |
| th10_08 | 0 | 8969 | 7705200 | 159 | −1／0 | 52 |
| th10_08 | 1 | 12330 | 24857120 | 155 | −1／60 | 56 |
| th10_08 | 2 | 14778 | 49383160 | 295 | −1／0 | 60 |
| th10_08 | 3 | 15227 | 63056550 | 171 | −1／20 | 56 |
| th10_08 | 4 | 18647 | 108392250 | 355 | 2／40 | 54 |
| th10_08 | 5 | 23325 | 143156790 | 191 | 0／250 | 56 |
| th10_09 | 0 | 46314 | 86028180 | 324 | −1／0 | 61 |

文件中没有魔理沙 C 的射击回放。其数据／组表／副机锁定与射击单测已覆盖，动态原作验证另需用户采集。

## 5. Trace 结果

阶段 A、B、C、D 与最终复验各六章顺序生成、cmp、与原作做 motion 差分，生成的大 trace 已逐章删除。

| 章 N | A、B、C、D、最终与基线 cmp | 原作 motion 一致帧数，五轮均相同 |
|---|---|---:|
| 0 | exit 0，逐字节相同 | 15853，使用 `orig_d.txt`。 |
| 1 | exit 0，逐字节相同 | 17733。 |
| 2 | exit 0，逐字节相同 | 29763。 |
| 3 | exit 0，逐字节相同 | 29469。 |
| 4 | exit 0，逐字节相同 | 39343。 |
| 5 | exit 0，逐字节相同 | 40099。 |

字节数仍为 94186573／144171327／271969413／319443819／430006225／510140938。没有与基线的分歧帧，没有一致帧下降；原作未采样和三／四／六面参考结束后的帧不冒充已验证。

## 6. 待决与推测

- A／B 验收通过；C／D 的实现、新测试／完整 smoke／六章 trace 已落地。用户已授权修改 `TestOtherShotsRemainLegacy` 与 `TestReimuAPowerLevelsAndBirths` 两个旧规格，架构师已在 t6 工作树重跑全套验收并通过（日志在 `.tmp/codex/t6/logs/lead_verify/`，六章 trace 以其中的 `trace_summary.txt` 为准；`summary.txt` 里首次的 trace 因脚本路径问题失败，已由 `traces_fixed.sh` 重跑）。不将规格失败当作基线问题。
- `TestOtherShotsRemainLegacy` 要求其余五机体前两帧没有新弹、第三帧才发射，与阶段 C 原 SHT 的第 0 tick 射击冲突；`TestReimuAPowerLevelsAndBirths` 则沿用旧零横移近似。两个旧规格已按用户授权按原作修正；仍不能用兼容分支绕过或放宽容差。
- 六份 SHT 共 272 条记录：pl00a 仅 kind 0、初始化／更新 0／1；pl00b、pl00c、pl01c 无非零回调；pl01a 的 kind 2 只有初始化回调 1；pl01b 的 kind 3 用更新回调 2。没有额外 Master Spark 回调或其他 kind 的原数据，不另造近似行为；魔理沙灵击由原 0x197 ANM 驱动。
- `0x491bf4`／`0x491bf8` 位于 PE `.data` 的未初始化虚拟尾部（.bss 意义），并非文件中有两个未知函数指针。装载初值为零，完整 exe.asm 各只有一处 indexed 读取、无写入；六份原 SHT 对应索引全为 0，所以它们是单零项占位表。取证在 `.tmp/codex/t6/evidence/tables.txt`，脚本 `.tmp/scripts/t6_tables.py`。没有假造未找到的函数。
- 共同射击运动已补齐原游戏坐标与 0.01 网格、正确 W／H 出屏、直接写主／附加 VM Offset（不把副机的 child-chain Place 语义误用于子弹）。选敌仍使用第一轮的稳定首个合格目标桥接，需动态原作 trace 检查是否每个场景都锁中同一敌人，不能仅靠回调单测称完整射击动态位精确。
- D 逐行核对了 `r2_spell/func_405750`／`405860`／`405ac0`／`4059f0` 与 `bomb/func_4055c0`／`405850`：0x22 的回调是 `MOV EAX,1; RET`；更新本体只有移动／SetOffset／取半径／清弹／计时，没有额外遗漏的手写 VM 分支。初始与定时子 VM 全由原 0x190／197／1b9 的 op 88 生成，保持 M7 生命周期和伤害，不重复生成。
- 0x189 原脚本使用 sprite 0x14c、层 11、0.5→1.5 缩放及 10～20 帧淡出，时间 20 结束，无 RNG 操作。两处触发条件和顺序按原作；`FUN_0041bb00` 的条件确实是槽的旧 kind 3，不改成“新生大信仰就发特效”的近似。
- 暂无新 dump 请求；已有完整 exe.asm 可补核预先 dump 中的空 ASM 文件。
- 备用 RNG 种子沿用第一轮接口，启动时应用级种子由 t8 负责；本任务不改其种子政策。

## 7. 建议的 `docs/fidelity.md` 改动

只提供建议，不直接改总账。按阶段列出完整候选状态行；A／B 已验收，C／D 与整体任务状态必须保留旧规格待决：

```text
| 副机布局／插值／World VM | 移植（六布局，演出待目视） | `FUN_00426f70`、`FUN_004250b0`、`FUN_00427960`、`FUN_00427ae0`；四个定点槽与 33 项历史，主脚本 0x11+shot 尾插、满火力 0x14+shot 头插，均以层 15 生成、脚本 op 68 改为层 8；魔理沙 A 的七项历史按 1/8 至 7/8 插值，魔理沙 C 用标签 3／6 锁定与 30 tick 释放延迟；死亡／降火力中断 1，重建旧句柄 FlagDelete | `entity/player/options_test.go`；th10_05 六章逐字节 cmp、全程 smoke；待 L3 |
| 灵梦 A 射击飞行／命中 ANM | World VM 演出移植；射击浮点逻辑仍有既有有界漂移 | `FUN_00427e90`、`FUN_00428280`、`FUN_00428630`；pl00 飞行与 SHT 命中脚本在 World 层 8、节点 0x15；命中保留旋转、速度乘 0.125、z＝0.1、记录进入状态 2，World VM 结束才回收；0x16 不再绘制子弹／副机 | `entity/playerbullet/anm_test.go`、既有射击测试；六章 cmp 与全程 smoke；待 L3 |
```

候选更新记录：

阶段 B 完整状态行建议：

```text
| 自机信仰保持矩形门控 | 移植（默认配置不绘制） | `FUN_00426360` 0x4263d7–0x42642e；读取 `GameState.ConfigFlags` 的 0x80 位（cfg +0x30 → stage +0x54）、敌人管理器与 Boss 槽 0、GUI 的 MSG 指针、`Context.Menu.State == 0`；保持计时非零、状态非 2 才画两条原矩形；不把 host 暂停或信仰衰减的 90 帧门槛混入绘制门控 | `TestPlayerFaithRectangleStageGates`、既有矩形公式测试；完整 smoke |
| 回放自机状态 capture／restore | 移植（33 项历史、四组副机对） | `FUN_0042a450` 0x42a512–0x42a687；+0x2c 为 33 个定点 XY 历史对，+0x134／154／174／194 为四副机的目标／当前位置／高速偏移／低速偏移或锁定 home；恢复先 RefreshOptions，再原字复制并清首帧吸附标志，魔理沙 C 特例按目标恢复 home；sim／游戏内播放共用 `Stage.RestoreReplayChapter`，现场录制读实时字段，不再靠 Raw 透传自机块 | th10_05 六章“恢复后保存”452 字节一致，既有六章重录强测试；章节边界现场 capture→Encode→Decode；六章 trace 逐字节相同 |
```

阶段 C 状态行建议（需保留旧规格待决，不先标“验收全绿”）：

```text
| 其余 5 机体射击 | 移植（数据／回调运行时；动态射击差分与旧规格校正待决） | `FUN_00426520`、`FUN_004281d0`、`FUN_00428160`、`FUN_00427e90`、`FUN_00428280`、`FUN_00428630`、`FUN_00428ad0`／00428b10／00428c20；六份 SHT 十组共 272 条记录，128 槽原 burst／float32 网格与 World VM；灵梦 B／C、魔理沙 C 是原 kind 0，魔理沙 A 火箭 kind 2 在命中时生成原区域，魔理沙 B kind 3 激光双 VM、单源门控、每四 tick 伤害、亮暗中断与退出。删除 shootReimuB／C 和 shootMarisaA／B／C 的硬编码。后两表为未初始化虚拟数据的单零项，不猜未知回调。目标选择保留既有稳定首目标桥接 | 新增 14 测试；4 demo＋9 用户回放 28 章无 panic；th10_05 六章 cmp 相同、motion 不下降、完整 smoke 通过；两项旧近似规格保持不改，待授权 |
| 灵梦 A 诱导／共用射击运动 | 移植（float32 原分组；动态目标锁存待差分） | `FUN_00428b10`／00428ce0／0044c2a0；命中态跳过诱导，120 tick 后无上限加速，最短角差按原重新分组；游戏坐标按 floor(v×100)×0.01，下限零附近负残差实际变成 −0.01，而非旧近似的零 | 既有回调测试与新边界测试；旧 TestReimuAPowerLevelsAndBirths 的无横移期望待授权修正，不能放宽容差 |
```

阶段 D 状态行建议：

```text
| 灵击生命周期／演出 | 原逻辑位精确；World ANM 演出接线移植，目视与全仓规格校正待决 | `FUN_00405860`／00405750／00405ac0／004059f0；M7 的 0x190／0x197／0x1b9 原本体、op 88 初始／定时子 VM、上升／半径／清弹／伤害和 VM 结束门控保持；FUN_004055c0 注册的 draw 0x22 是原空桩，不补造额外视觉。Power 数字在起击、普通／决死扣火力后、死亡掉落前按原调用点接既有 FUN_004054b0；0x189 的两个道具路径通过同一 World／bullet.anm，按旧槽 kind 3／满火力转换顺序在层 11 执行原 20 帧缩放淡出脚本。奖励数字七 VM flag 清理属于 t7，不在此移植；无凭空的灵击得分弹出字调用 | 既有 bomb_test.go、8 个新演出／特效／调用顺序测试、24 章完整 smoke、六章 trace cmp 相同；两个射击旧规格未绿，不先标整个任务完成 |
```

```text
- 2026-11-08 — t6 阶段 A：副机与灵梦 A 射击／命中改为原 World VM，层 15 生成后由 pl0N op 68 改到层 8，0x16 取消 legacy，仅直接画自机 VM 与矩形。四副机槽／33 项历史、魔理沙 A 的递进历史权重、魔理沙 C 的绝对锁定与标签 3／6、30 tick 释放延迟按原汇编移植；重建打 FlagDelete，死亡／降火力中断 1。命中射击记录保留至火花 VM 结束，不立即复用。删除旧 drawPlayerBullets／drawPlayerOptions 与重复 ReimuAnm sheet。新增 11 测试，全仓 build／vet／test／gofmt／WASM／完整 smoke 全通过，th10_05 六章与基线逐字节相同，motion 为 15853／17733／29763／29469／39343／40099。其余五机体射击、矩形门控／回放接线、灵击演出仍待后续阶段。
- 2026-11-08 — t6 阶段 B：原隐藏 cfg 位 0x80、Boss／GUI 对话／菜单状态门控接信仰保持矩形，默认不绘制。自机 33 项历史和四组副机定点对通过 t5 hook 接实时 capture／restore，恢复顺序为历史／focus→RefreshOptions→四组对→清吸附标志→原状态／初始化时钟重置；无敌时钟不重置，GUI 残机仅调用已有 SetLives。新增 8 测试，th10_05 六章章头恢复后逐字节一致、原六章重录强测试全通过、下一章现场原字段经 Encode／Decode 保留；完整验收与全程 smoke 通过，六章 trace 再次与基线逐字节相同。
- 2026-11-08 — t6 阶段 C：删除其余五机体硬编码射击，六份原 SHT 共用 128 槽 burst Timer／组表／World VM。初始化回调 1、更新回调 1／2 逐函数移植，确认 +0x2c／+0x30 的索引表为未初始化虚拟尾部的单零项；kind 2 命中区域、kind 3 双 VM／每源一束／四 tick 伤害／亮暗与退出接入。共同运动改原 float32 游戏坐标、0.01 floor 网格、正确 W／H 出屏，补原 SE。新增 14 测试，28 章 CLI 无 panic，完整 smoke 通过，六章 trace 逐字节不变。原 TestOtherShotsRemainLegacy 与 TestReimuAPowerLevelsAndBirths 的旧近似期望冲突，保持测试不改、未写 DONE；不得据此称全部验收完成。射击动态 trace 仍需用户采集。
- 2026-11-08 — t6 阶段 D：核对原灵击余项，0x22 为原空绘制桩；本体和所有初始／定时子 VM 已由原 ANM 驱动，保持 M7 逻辑不额外手写特效。FUN_004054b0 的起击／普通和决死扣火力后／死亡掉落前调用接既有 GUI.SetPower；0x189 按 FUN_0041bb00 的旧槽 kind 3 与 FUN_0041ba50 满火力转换两个原调用点加入 World 层 11、原 20 帧缩放淡出，不耗 RNG。未碰奖励七数字 VM 或重排 Spell.Bombed。新增 8 测试，完整 smoke 和六章 cmp 通过；全仓仍只有两个射击旧规格失败，保留原断言，等待授权，未写 DONE。
```

## 8. 需要用户目视确认的项目

运行复刻：`go run ./cmd -debug-stage=1 -debug-power=500`，通过已有选人／Debug 参数选择六种机体。

- 副机入场缩放与旋转，满火力叠加图；降火力和死亡时按原脚本退出，不突然消失。
- 魔理沙 A 的高速轨迹拖尾、低速相对位置；魔理沙 C 的原地锁定与解除后的延迟。
- 灵梦 A 的飞行弹旋转、命中火花与层序；火花应在自机本体之前，不能复用成新弹。
- 六机体的射击图、命中图、爆破区域与魔理沙 B 激光亮／暗、停止射击退出需目视；已删除旧直线弹占位绘制。
- 普通 cfg 不出现自机旁的信仰条；启用 cfg +0x30 的 0x80 隐藏位后，无 Boss／无对话／未开结算菜单且保持计时非零时出现两个矩形。没有提供新的游戏内配置入口。
- 用户回放菜单播放 th10_05，可逐章进入；副机恢复后第一帧应从章头当前位置继续插值，而不是吸附到目标。
- th10_05 第 5 章（六面）的第 1649 帧发动灵击：确认原环、旋转光晕与持续子 VM，不重复生成第二套近似特效；Power 数字正确下降。
- 满火力时尚在场上的小／大 P 转为信仰道具，位置出现原绿光 0x189；其缩放／淡出为 20 帧。复用曾放大信仰的空槽也会触发该特效，这是原条件，不是只对新大信仰生成。

本任务没有启动窗口或做像素差分，不报告 L3 已完成。

射击原作 trace 请求（用户自行用 probe 采集，本任务不启动原作）：

| 机体／目标 | 回放与章 |
|---|---|
| 灵梦 B 高／低速与火力变化 | `.tmp/th10_dat_extract/demo0.rpy` 第 0 章，或同内容的 th10_01。 |
| 灵梦 C 广域／低速组 | demo1 第 0 章，或 th10_02。 |
| 魔理沙 B 激光／每四 tick 与停止射击 | demo2 第 0 章，或 th10_03。 |
| 魔理沙 A 拖尾副机／火箭区域 | demo3 第 0 章，或 th10_04。 |
| 灵梦 A 诱导动态与命中图 | th10_08 第 0 章；若再做完整校验采第 1～5 章，Extra 用 th10_09 第 0 章。 |
| 魔理沙 C 锁定副机／普通 kind 0 | 现有 13 文件没有该机体的射击数据，需另录一面，包含高／低速切换、释放 30 tick、1.00～4.00 火力；再采对应第 0 章。 |

## 9. 与其他任务的接口

- 使用既有 `World.SpawnTail`／`SpawnFront`／`Delete`／`Interrupt`／`Place`／`Contains`，不改 `anm/world.go` 及其规格测试。
- `Player.AttachAnm(w *anm.World, env *anm.Env, bullet *anm.File)` 仍由 `NewContext` 调用，内部新增 `Pool.AttachAnm(w, env, p.Anm)`；敌弹 `bullet.anm` 仍复用已有文件。
- `Player.RefreshOptions()` 保持道具管理器现有 `ItemEnv.Options` 接口，对应 `FUN_00418930` 与灵击／死亡路径的 `FUN_00426f70` 调用。
- t5 的 `Context.CaptureReplayPlayer`／`RestoreReplayPlayer` 已在 `NewContext` 接到 `Player.CaptureReplayPlayer(header []byte)`／`RestoreReplayPlayer(header []byte)`；位置／focus 仍由既有 Stage 类型字段处理，其余自机字段写实时结构。sim 与 `stage.NewPlayback` 均走 `Stage.RestoreReplayChapter`，不另造播放路径。
- 恢复后的 GUI 残机标志通过既有 `GUI.SetLives(n int)`，对应 `FUN_00424d90` → `FUN_00413790`；仅改 FlagDraw，不发残机、不动得分或无敌时钟。
- t7 的符卡奖励七 VM 隐藏不在本任务；本阶段没有改 `c.Spell.Bombed()` 的调用点或顺序。
- t3：`GUI.SetPower(whole, percent int)`＝FUN_004054b0，已接 `Bomb.Start` 的 0x405975 与自机的 0x425a10／0x425af7／0x425c7f；提供 `Player.PowerChanged func()`，由 Context 包装既有 `powerDigits`／`GUI.SetPower`。既有 `GUI.Calc` 的显示缓存仍保留，没有把它改成新的资源计算。
- t3：`Context.AddScorePopup(pos [3]float32, value int32, color uint32)`＝FUN_0042b9c0，原道具调用点已由 t3 接好。本次核对灵击 Start／Update／clear／Damage／注册函数均无该调用，不为灵击伪造弹出字或加分。`ItemEnv.Popup` 仍保持已有接线。
- `item.Env.Effects *anm.World` 是两个 0x189 调用点的表现层依赖，`NewContext` 接同一 `c.World`；`Env.File` 仍复用已有 `c.BulletEnv.File`，没有额外载入资源或新建效果池。
- 常规关间 fade／控制器对象复用仍是第一轮 t5／t8 的边界；本任务的 capture 在 `Stage.StartRecording` 读当前实际玩家字段，播放经共同 restore 路径，不自行重排关间控制器或 RNG 初始化。
- SHT 子弹的 `Pool.SourcePosition func(int) [3]float32`、`GameSpeed func() float32`、`OptionCount func() int`、`ShotClock func() int32`、`NoAttack func() bool` 由 Player 接实时字段；`ManagerPresent func() bool` 由 Context 接 ECL 管理器，供 `FUN_00428280`／`FUN_00428c20` 读取。
- `Pool.HitArea func(x, y, radius, growth float32, frames, damage int32)` 接既有 `Player.addArea`（FUN_00427b50），`Pool.PlaySE func(id int, x float32)` 接 `Context.PlaySEIDAt`（FUN_0043dd10），不另做区域伤害或声音系统。

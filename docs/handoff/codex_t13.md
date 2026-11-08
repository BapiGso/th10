# t13：差分修正 B 交付

## 1．摘要

阶段一已完成：三条灵梦回放的有效 motion 样本已推进至当前原作数据末尾，本体碰撞修正消除了两个真实分歧。
已完整阅读共用规则、真值台账、差分工具说明和相关研究／t1–t9 接口。
阶段二、最终复验已完成：按原作实现全部 0–8 kind，接通通关屏幕效果；补 MSG 慢速计时与连续跨面的主 RNG 引用。
只使用 motion 比较，不修改比较容差、既有测试断言或 golden，不改关外场景。
未修改灵梦 SHT／自机子弹射击路径、自机移动步长或死亡／复活状态机；两个真实问题来自同一本体碰撞遗漏。
规定 build／vet／test／WASM／gofmt／完整 smoke 均通过，六条新回放无倒退；额外 lint 仍有 33 个既有报告，不称 lint 全绿。

## 2．改动文件清单

新增：

- `tools/th10diff/from_frame_test.go`：显式续比区间的边界、shift、容差与缺失尾部测试。
- `trace/short_line_test.go`：截断 F／E／B／M 行返回错误，不 panic、不误报 MATCH。
- `scene/stage/ecl/contact_test.go`：本体碰撞门控、独立 361 计时器、移动后的判定盒与下一帧冻结。
- `scene/stage/ecl/screen_effect.go`：全部 kind 的 calc／draw、alpha／包络／原 RNG 路径。
- `scene/stage/ecl/screen_effect_test.go`：屏幕效果的构造、时钟、生命周期、几何、慢帧、RNG、最终 MSG 与跨面测试。
- `scene/stage/screen_effect_chain_test.go`：动态 draw 优先级及实际 scene 0x0b 的 RNG／效果交接。
- `scene/dialog/msg_speed_test.go`：慢速 MSG timeline／wait／Ctrl 分数计时测试。
- `game/stage_rng.go`：主 RNG 在连续关卡间共享，回放仍按章头恢复。
- `docs/handoff/codex_t13.md`：本交付文档。

修改：

- `tools/th10diff/main.go`：新增 `-from-frame` 和明确的 INCOMPLETE 计数，容差不变；预置的 `-bullets` 分诊代码保留。
- `trace/trace.go`：检查短行与空白行，并为 B 行错误附上帧号。
- `scene/stage/ecl/opcodes.go`：命名原 ECL 361 接触禁用指令。
- `scene/stage/ecl/game_ops.go`：增加独立接触计时器与 361 的原参数读写，不改伤害计算。
- `scene/stage/ecl/vm.go`：原位置接本体碰撞／计时器，绑定屏幕效果 hook。
- `scene/stage/ecl/shake.go`：泛化原 kind 1 存储，保留其采样主体，提供完整构造／清理与返回对象。
- `scene/stage/bullets.go`：复用原命中门控接本体矩形；连续局读共享主 RNG。
- `scene/stage/stage.go`：MSG 接游戏速度，增加屏幕效果的 draw／清理／原控制字段。
- `scene/stage/chain.go`：动态效果按原优先级插在静态节点前，不改原节点列表。
- `scene/stage/controller.go`：注明 kind 5 的淡黑语义；关卡释放清效果，scene 0x0b 传主 RNG。
- `scene/dialog/msgvm.go`：clock／wait 改原 ZunTimer，按游戏速度步进，Ctrl 重置整数与分数计时。
- `game/state.go`：保存连续关卡的主 RNG 引用，不改变资源规则。

删除：无。既有测试文件、golden、`go.mod`／`go.sum` 未修改。

工作树外仅有本任务辅助文件：`.tmp/codex/t13/inspect.go`、`resume_msg.ps1`、`verify.ps1`、logs／evidence；`.tmp/scripts/t13_trace.py` 为只读 trace 观察器。没有新 Ghidra dump，没有保留大 trace。

## 3．移植的函数地址

| 原作 | Go 函数／分支 |
|---|---|
| `FUN_0040dc80`，`0x40e2bd–0x40e2df`／`0x40e558–0x40e56d` | `VM.updateActor` 的本体接触门控与接触计时器递减。 |
| `FUN_0040e770` case 361，`0x41004c–0x41005d` | `VM.execGameOp(opContactOff)`，`eclActor.contactOff`。 |
| `FUN_004266b0`，`0x4266b3–0x426761` | `Context.EnemyContact`；复用 `bulletHit → hitPlayer → Player.Hit`。 |
| `FUN_00415b00`／`FUN_00415e90` | `NewMSG`／`MSG.Update`；原 clock／wait 绑定 `DAT_00476f78`。 |
| `FUN_00405410`／`FUN_00404ed0`／`FUN_0044bf40` | 复用 `zun.Timer.Set`／`Tick`／`Add`，没有改生成器或 Timer 主体。 |
| `FUN_0043c8b0`／`FUN_0043c970`，表 `0x43cad8` | `ScreenEffect`、`VM.NewScreenEffect`／`RequestShake`；全部 0–8 kind。 |
| `FUN_0043bd40` | `ScreenEffect.calc` 的 0／3：按 tick 前时钟淡入。 |
| `FUN_0043c230` | `ScreenEffect.calc` 的 2／5：淡出与两次满 alpha 尾部。 |
| `FUN_0043c460` | `ScreenEffect.calc` 的 4：重复闪烁，return 0 只摘 calc 节点。 |
| `FUN_0043c310` | `ScreenEffect.calc` 的 6／7：半透明保持与外部 phase 控制的八 tick 退出。 |
| `FUN_0043c550 → FUN_0043c5ca` | `ScreenEffect.calc` kind 1／既有 `sampleShake`，先 tick、再过期、再两个 Dup。 |
| `FUN_0043c710 → FUN_0043cb80` | kind 8／`sampleEnvelopeShake`，rise／hold／fall 与两个 Raw()%3，门控保留。 |
| `FUN_0043c1a0`／`0043c2c0`／`0043c3c0`／`0043c410`／`0043c500` | `VM.DrawScreenEffects` 的全屏／场地矩形与原颜色；复用 `Renderer.Rect`。 |
| `FUN_0043c870`／`0043cb00`／`00449c40`／`00449f60` | return 7／return 0 生命周期与 `VM.ClearScreenEffects`。 |
| `FUN_00449b70` | `Stage.runChain` 的动态优先级区间，同优先级新注册者在前。 |
| `FUN_00417c80`／`DAT_004918b0` | `GameState.CarryStageRNG`、`NewContext`、`Stage.deliverSceneRequest` 的连续局交接；回放 `FUN_0042a6a0` 仍覆盖章节种子。 |

构造／calc／draw 的新分支与常量均在代码中注明地址。alpha 的 255／128 为 `0x437f0000`／`0x43000000`，无符号转换的 2^32 为 `0x4f800000`；场地／全屏矩形为原立即数 float32 位模式。

## 4．测试

阶段一：`gofmt -l .` 输出为空；build／vet／`go test -count=1 ./...` 全部通过；完整 smoke 通过（114.911 秒）；WASM 构建通过。
首次 WASM 验证脚本在切换 GOOS 后列包，被 Windows 专用 tools 阻断，未实际进入 build；已修正为本机先列非 tools 包，js／wasm build 补验 exit 0（4.242 秒）。这是验证脚本问题，不是代码基线失败。
新增区间／短行解析测试及敌机本体碰撞三项测试通过；既有测试断言和 golden 未修改。
日志：`.tmp/codex/t13/logs/phase1/`，WASM 补验见 `wasm_corrected_result.txt`。

最终复验（同一份最终代码），日志 `.tmp/codex/t13/logs/final/`：

| 检查 | 结果 |
|---|---|
| `gofmt -l .` | 输出为空，exit 0。 |
| `go build ./...` | exit 0。 |
| `go vet ./...` | exit 0。 |
| `go test -count=1 ./...` | 全包通过，exit 0，93.319 秒。 |
| 非 tools js／wasm 构建 | exit 0，3.678 秒；列包先在本机完成，避免 Windows 专用 tools 阻断。 |
| `TH10_SMOKE=full go test -count=1 -run TestReplayDrawSmoke ./scene/stage/` | 通过，测试 117.615 秒，命令 119.043 秒。 |
| `golangci-lint run` | exit 1，33 项：errcheck 26／staticcheck 2／unused 5。额外检查，不属于 rules 第 6 节门槛。 |

另在只读 base 中实际复跑 lint，同为 33 项、相同分类，日志 `logs/baseline_lint.txt`。新增代码没有新的报告项；旧语句移动后的行号不同，预置 tools 测试扩展与 base 的定位有差别，不声称所有 lint 行逐项字节相同。不为清 lint 越界重构，亦不称 lint 全绿。

新增 23 个顶层测试，分组如下：

- `TestDiffFromFrame*` 三项；`TestReaderTruncatedRowsReturnErrors`／`TestReaderWhitespaceOnlyLine` 两项。
- `TestEnemyContact*` 三项：0x1 不阻止接触、0x2／0x10 阻止接触、361 到期边界、335 独立、移动后贴边命中与随后冻结。
- `TestScreenEffect*` 八项：全部 kind 构造、旧／新时钟位置、原 alpha 取整顺序、两满 alpha 尾部、kind 4 的 calc-only 摘除、6／7 的分数边界、kind 8 的 Raw／Dup 差别、零幅度／慢帧／门控、全屏／场地几何与纯绘制。
- `TestFinalClearConnectsKind5AtOriginalBirthFrame`／`TestShakeContinuousCarryAndReplayChapterReset` 两项：真实 MSG post 条目在 clock 540 起通关效果，创建帧不提前 calc；116 个主 step 传到下一关，回放仍重新播种。
- `TestScreenEffectDrawRangesInterleaveStaticChain`／`TestLiveSceneTransferCarriesRNGAndReleasesEffects` 两项：0x31 在 Finish 前，实际 scene 0x0b 先交 RNG／释放效果再构造下一关。
- `TestMSGQuarterSpeedTimeline`／`TestMSGScaledWaitUsesTruncatedTimer`／`TestMSGSkipResetsFractionalClock` 三项。

开发中新测试一度误把六面 MSG post 的 op 20 当 time 0，另一次 fixture 未设置 Stage＝1；已按原数据 time 540 和有效 STD 场景修正，只改本任务新增测试。既有断言保持原样，最终无测试失败。

## 5．Trace 结果

### 三条修正记录

`th10_08`，4915，基线复现：

```text
DIVERGE at frame 4915: dialog {entry 0 clock 34 wait 0 signal 0 at 34 op 16} != {entry 0 clock 34 wait 599 signal 0 at 34 op 10}
```

采样问题，非 op 10 提前通过：原 MSG 的同时间戳 op 16 在 op 10 之前。
`orig.txt`／`orig_th10_08.txt` 在该帧均停在绘字的 op 16；`orig_d.txt` 没有采到 4915。
`orig_th10_08.txt` 的 4916 为 clock 34／wait 598／op 10，和复刻完全相同；若原作真的越过 op 10，不可能下一帧退回它。
输入记录 4910–4917 为 keys 0x1／press 0／release 0，没有 Z 按下沿或 Ctrl。
`FUN_00415e90` 的 op 10 检查当帧 `DAT_00474e62 & 0x1001`，Ctrl 为 `DAT_00474e5c & 0x100`；回放输入 `FUN_0042a3d0 → FUN_004297d0` 在 calc 0x0b，MSG 在 GUI calc 0x18，输入不应错移一帧。
probe 的 settle 后双读相等不能证明 calc 完成：同步 GDI 绘字阻塞超过两次读取间隔时，两次都能看到 op 16。
原作 PC 只在指令结束后由 `0x416f09` 附近推进，`FUN_00447a50` 尚未返回时没有执行后面的 op 10。
相同采样形态另在 4993／5036／5151／5211／5309／8773 出现，下一有效样本均追上复刻。
显式续比最终 `-from-frame 8774` 为 137 帧 MATCH；这些半帧不算一致，不删除原作数据或放宽容差。

`th10_02`，6039，基线复现：

```text
DIVERGE at frame 6039: player pos (0.000,480.000) != (0.000,400.000)
```

根因：`FUN_0040dc80` 的 `0x40e2bd–0x40e2df` 本体碰撞没有接线。原作 6000 已被碰到，复刻直到 6040 才被敌弹击中。
修正后 6000 state 4／invinc 6，6009 state 2／life 0，6012 掉火力，6039 state 0／y 480／invinc 279 全部对齐；没有改变死亡／复活计时。
6359 以前 6175 个有效采样一致；原作末帧 6360 最后一行是 `B 483.200 139.284 ang 2.64680 spd`，不能验证不完整帧。
解析器改为明确 read error，不再越界 panic，绝不把截断数据当 MATCH。

`th10_01`，4637，基线原文：

```text
DIVERGE at frame 4637: player pos (-71.680,105.820) != (-76.180,105.820)
```

同一本体碰撞遗漏：4636 原作已经 state 4／invinc 6；复刻尚为 state 1，4637 多走了一帧高速左移 4.5。接线后冻结点正确，不改步长或移动条件。
后续首分歧：

```text
DIVERGE at frame 6203: enemy count 3 != 4
```

此为符卡宣言绘字半帧：原作 player 已更新、敌弹仍与 6202 完全相同，Ecl_EtBreak 尚未入链；6204／6205 漏采，6206 起四敌机与敌弹恢复对齐。
`FUN_00409280` 的 `0x409681` 名字绘制调用同步阻塞；续比 `-from-frame 6204` 为 1615 帧 MATCH，至 7824。保留该异常帧，不调整 ECL 调度或容差。

### 六条新回放

| 回放 | 原作 | 修正前首分歧 | 最终完整比较首分歧／状态 |
|---|---|---:|---|
| th10_08 第 0 章 | orig_th10_08.txt | 4915 | 4915，采样异常；分段覆盖至 8910，无真实 motion 分歧。 |
| th10_04 | orig_th10_04.txt | 2553 | 2553，`enemy count 7 != 4`，未倒退，留 t12 分诊。 |
| th10_03 | orig_th10_03.txt | 252 | 252，`enemy[2] hp 11 != 20`，未倒退，留 t12。 |
| th10_10 | orig_th10_03_as_C.txt | 5011 | 无有效样本分歧；5340 个样本至 5351，一致；5352 截断 read error。 |
| th10_02 | orig_th10_02.txt | 6039 | 无有效样本分歧；6175 个样本至 6359，一致；6360 截断 read error。 |
| th10_01 | orig_th10_01.txt | 4637 | 6203，采样异常；分段覆盖至 7824，无真实 motion 分歧。 |

六条全部使用最终代码顺序重跑；不是只复跑三条灵梦。r08 又从 4916 起逐个续比全部七个异常点（日志 `logs/final_r08_all_segments.txt`），不是只验证最后 137 帧。帧 0 不参与比较；r08 分段合计 8790 个有效样本，r01 分段合计 7744 个有效样本。完整运行仍报告采样异常，绝不把它们算 MATCH。

### th10_05 六章

阶段一六章 cmp 均 exit 0。最终结果如下，所有 motion 数值与规定基线相同：

| 章 | 最终 cmp | 与 base 的第一个分歧帧及原因 | 原作 motion 一致样本数 |
|---|---|---|---:|
| c0 | exit 0，逐字节相同 | 无；接触修正与正常速度 MSG 不改变该回放。 | 15853 |
| c1 | exit 0，逐字节相同 | 无，同上。 | 17733 |
| c2 | exit 0，逐字节相同 | 无；结束后的未采样区间不冒充原作验证。 | 29763 |
| c3 | exit 0，逐字节相同 | 无。 | 29469 |
| c4 | exit 0，逐字节相同 | 无。 | 39343 |
| c5 | exit 1，不同 | 49748，MSG post 绑定 0.25 速度，clock 1→0；不是震屏少取／多取 RNG。 | 40099 |

```text
DIVERGE at frame 49748: dialog {entry 1 clock 1 wait 0 signal 0 at 300 op 21} != {entry 1 clock 0 wait 0 signal 0 at 300 op 21}
```

此处 A 是改动前 base，B 是最终复刻，不是原作。`FUN_00415b00` 把 +0x24／+0x38 两个 timer 指针写为 `DAT_00476f78`；`FUN_00415e90` 的 op 10 调 `FUN_0044bf40(-1)`，尾部调用 `FUN_00404ed0`，原 GUI 调用者没有把它改成常速。旧实现直接 clock++／wait--，丢失分数计时。
已改为原 Timer，并补 Ctrl 同时重置 Cur／F；三条新的灵梦回放与 c0–c4 的相关 MSG 都是正常速度，无倒退。c5 逻辑末帧由 50288 变为 50408；原作参考到 45000，后续 5408 帧不算原作验收，完整六面战后有效 trace 仍待用户。

### 震屏 RNG 独立复核

修改前、最终各顺序观察六章。每章仅执行一次 `BossDead` 的 `337(30,12,0)`，结果两轮相同：

| 章 | 执行帧 | 首次采样 | 最后采样 | 实帧采样数 | 主 RNG step |
|---|---:|---:|---:|---:|---:|
| c0 | 15878 | 15879 | 15907 | 29 | 116 |
| c1 | 19733 | 19734 | 19762 | 29 | 116 |
| c2 | 31444 | 31445 | 31473 | 29 | 116 |
| c3 | 31160 | 31161 | 31189 | 29 | 116 |
| c4 | 40490 | 40491 | 40519 | 29 | 116 |
| c5 | 49747 | 49748 | 49866 | 119 | 476 |
| 合计 | 六次 | — | — | 264 | 1056 |

证据 `evidence/shake_before_cN.txt`／`shake_final_cN.txt` 记录每次 calc 的真实主 RNG Count 增量。前五章速度 1，六面 0.25；Cur 未变的实帧仍采四个 step，过期帧不采。t7 的时机、条件与次数正确，没有为了维持 cmp 而省略采样。
原逻辑 trace 不含 RNG State／Count 或绘制偏移，这些取数发生于最后一批运动对象生成之后，后续没有由这些新随机数改变的 trace 行，因此五章可保持逐字节相同。回放每章恢复独立种子；连续局则必须保留主 RNG 引用，已有正向测试覆盖。

`FUN_0043c970` 表的 kind 5 指向 `0x43c9ee`，选 `FUN_0043c230`／`FUN_0043c1a0`；两者没有 RNG 调用。通关的 5／120／0／0／0 是全屏淡黑，不能按“震屏”名称伪造四步取数。kind 1 是两个 Dup()%3；kind 8 是两个 Raw()%3，同样四步，但符号序列不同。新增测试分别锁定它们。

每条大 trace 比较后立即删除，最终 `trace/` 为空。仅保留日志、小窗口与观察结论。

## 6．待决与推测

没有新增未证实的运行时近似；没有运行 Ghidra、原作或 probe，也没有新增函数 dump 请求。
两份原作 trace 末行截断需要重采：`orig_th10_02.txt` 的 6360、`orig_th10_03_as_C.txt` 的 5352；不请求无意义的新 dump。
完整的用户／架构师补采命令与用途写在 `.tmp/codex/t13/dump_requests.md`；其中函数 dump 请求为“无”，另列六面战后慢速 MSG 的可选补采。
独立复核六章 ECL 337 每章执行一次，前五章各 29 次采样／116 主 step，六面速度 0.25、119 次／476 step，共 1056；与 t7 一致，不需要更改 kind 1。
kind 5 已按原淡黑接线。kind 6／7 的 +0x2c 外部 phase 写者与关外菜单／加载／结局生命周期不在范围内；提供原对象、SetPhase／SetTimer 与任意 draw priority，不推测那些场景调用。`Context.StageFlags` 是原 stage+0x58，不能混成 RunFlags；其他控制器可接原位，当前已复用菜单的已知 0x10 冻结语义。
全屏淡黑、离散震屏及字体像素仍需目视。完整 smoke 是稳定性检查，不等于 L3 像素还原。

## 7．建议的 `docs/fidelity.md` 改动

只建议以下完整状态行，不直接改总账：

```text
| 自机本体接触／死亡触发 | 移植（新回放有效 motion 验证） | FUN_0040dc80 0x40e2bd–0x40e2df：在伤害与两次中断检查后按 flags&0x12／独立接触计时器调用 FUN_004266b0；ECL 361 设置 EnemyData+0x1430，尾部递减。共用原自机被弹／决死／死亡／复活，0x1 不抑制接触，贴边按 float32 矩形判定 | 三项接触测试；th10_01 4637／th10_02 6039 真分歧消除，th10_10 5011 连带修正；有效采样达末尾，采样／截断例外见 codex_t13.md |
| 对话 .msg 时序 | 移植（整数／分数计时；演出字体仍近似） | FUN_00415b00 为 clock／wait 绑定 DAT_00476f78，FUN_00415e90 的 Ctrl 同时设置 Cur／F，op 10 按 FUN_0044bf40(-1) 等待，尾部 FUN_00404ed0 步进；输入在 calc 0x0b，MSG 在 0x18。新的一面七个 op 16 中间采样不算逻辑分歧，不放宽容差 | 既有 MSG 测试＋三项慢速测试；th10_08 分段 8790 有效样本至 8910；六章 motion 不下降，c5 相对 base 的 49748 改动符合 0.25 速度 |
| 震屏／屏幕效果 | 移植（全部 0–8 kind；关外调用／像素待确认） | FUN_0043c8b0／0043c970 0x43cad8 的构造与返回对象，calc 0x0e、原 draw priority；0／3 淡入、2／5 淡出、4 闪烁 calc-only 摘除、6／7 半透明保持／phase 退出、1／8 离散震屏。1 按 FUN_0043c5ca 取两个 Dup，8 按 FUN_0043cb80 取两个 Raw；每样本四主 step，零幅度／慢帧保留，过期帧不取。MSG 0x416d85 的 kind 5／120／0／0／0 是 draw 0x31 全屏淡黑，无 RNG；不把效果移进 ANM 世界或改逻辑坐标 | 新增十项效果／RNG／最终 MSG 测试与两项链／交接测试，完整 smoke 通过；六章每章一次 ECL 337，共 1056 主 step；连续局保留主 RNG，回放章头重设种子 |
```

更新记录建议：

```text
- 2026-11-08 — t13：三条灵梦回放差分取证，确认一面七个 MSG op 16 与五面 6203 符卡宣言为 GDI 中间采样；新增显式 -from-frame，不改容差，报告截断 trace 的 INCOMPLETE。补 FUN_0040dc80 本体碰撞及 ECL 361 独立计时，消除灵梦 B 4637 的多走一帧、灵梦 C 6039 的漏死亡／复活，连带消除魔理沙 C 5011 弹数差。按 FUN_00415b00／00415e90 将 MSG clock／wait 改游戏速度 ZunTimer，六面 post 的 0.25 计时不再每实帧加一。按 FUN_0043c970 移植全部 0–8 kind，保留 1／8 的四步 RNG 与 Raw／Dup 区别；通关 kind 5 实为无 RNG 的 draw 0x31 全屏淡黑，已接钩子。连续 scene 0x0b 携带主 RNG／Count，释放旧效果，回放仍恢复章头种子。新增 23 测试，完整 Go／WASM／gofmt／smoke 通过，六条新回放无倒退；th10_05 c0–c4 cmp 相同，c5 首差 49748 来自 MSG 分数时钟，原作 motion 15853／17733／29763／29469／39343／40099 不下降。两条原作末行截断及像素目视待补；lint 仍报既有 33 项，明确不是全绿。
```

## 8．需要用户目视确认的项目

在本工作树运行复刻 `go run ./cmd -debug-stage=6 -debug-power=500`，正常非回放通过六面；不要用回放去观察 kind 5，因为原回放分支就不调用它。

- 最终 MSG op 20 后，draw 0x31 的黑色全屏矩形在原 GUI 120 个后续 calc 等待中淡入黑；不仅覆盖场地，侧栏也属于全屏范围。速度为 0.25 时 alpha 同样按游戏速度变，不强制一实帧一 tick。
- 一至五面 BossDead 的 `337(30,12,0)`：创建帧无采样，下一帧从 11.6 开始的离散 0／正／负偏移，29 样本。六面 0.25 速度为 119 样本；相机／逻辑实体坐标不动，draw 0x24 后 GUI 不震。
- 被 Boss 或妖精本体碰到后立刻停止下一帧移动，经过原决死／死亡／复活从 y＝480 入场；不是无端加减移动一帧。

本任务没有启动窗口、原作或 probe，没有宣称目视／像素 diff 已完成。

## 9．与其他任务的接口

不改 t10／t11 关外场景。若需要修改 t12 共用自机子弹／敌机受伤／ECL 函数，将在此逐项记录。
`Context.RequestShake func(kind, frames, x, y, z int32)` 是 t8 留出的通关调用点 `0x416d85`，本任务接线。
已改共用 `VM.updateActor`：在伤害与两次中断检查之后、ANM 与计时器之前，按 flags 0x12／独立接触禁用计时器调用 `Context.EnemyContact`。这是敌机本体碰撞，不是 t12 的自机子弹伤害修正。
已实现 ECL 361 的独立计时器（`EnemyData+0x1430`），不能与 335 的受伤无敌计时器混用；只在敌机尾部递减，不改 `VM.hurt` 或 `EnemyDamage`。
共同修正也消除了 th10_10 的 5011 敌弹数量分歧；魔理沙 A／B 的既有首分歧仍为 2553／252。

与 t12 合并时按函数／字段去重，不整文件覆盖：

- `VM.updateActor(*eclActor) bool`：仅新增本体接触调用及独立计时器递减；`VM.hurt`、`Context.EnemyDamage`、自机子弹公共发射／命中、`GainPower` 与所有 SHT 回调未改。
- `VM.execGameOp`：仅本任务的 361 分支与接触字段；337 的参数 2→1→0 读序保持 t7 原样，其他 ECL 指令未改。
- `VM.Update(*stage.Context)`：增加 RequestShake／DrawScreenEffects／ClearScreenEffects 绑定；现有伤害／选敌／计时顺序不改。
- `GameState`：仅新 `stageRNG` 引用及独立 `game/stage_rng.go` 两方法，不改 t12 的火力／资源规则。
- `Context.EnemyContact(e *enemy.Enemy)`：只复用 `FUN_004266b0 → FUN_00426cf0` 的现成命中副作用，原调用点 `0x40e2df`；不是额外伤害／击破算法。

对外接口／其他任务可复用：

| 签名 | 原作与注意事项 |
|---|---|
| `VM.RequestShake(kind, frames, x, y, z int32)` | 对接 t8 原 hook；仅此调用点默认 draw priority 0x31，kind 5 无 RNG。 |
| `VM.NewScreenEffect(kind, frames, x, y, z, priority int32) *ScreenEffect` | 原 `FUN_0043c8b0` 返回对象；priority 是 `FUN_0043c970` 的隐藏 EBX，不假设所有调用都在 0x31。 |
| `ScreenEffect.SetPhase(int32)`／`SetTimer(int32)` | +0x2c 原字写入与 `FUN_00405410`；phase 不自动重置 timer，6／7 调用者自行按原作写。 |
| `VM.DrawScreenEffects(*supervisor.Supervisor, first, last int32)` | 原 draw 回调的闭区间；Stage 动态插链，同优先级新注册者在前，Draw 不采 RNG／不 tick。 |
| `VM.ClearScreenEffects()` | `FUN_0043cb00`，释放包括 kind 4 残留 draw 在内的全部节点。 |
| `Context.StageFlags uint32`／`StageExiting bool` | stage+0x58／`DAT_004918a4`；不可使用 `RunFlags` 或猜成 SHT 门控，关外写者不在本任务实现。 |
| `GameState.CarryStageRNG(*zun.RNG)`／`StageRNG() *zun.RNG` | scene 0x0b 保留主状态和 Count；NewContext 仅非回放使用，回放仍按章头覆盖；不从备用 RNG 派生。 |
| `MSG.GameSpeed func() float32` | `FUN_00415b00` 的原 timer 指针，由 `Context.OpenMSG` 绑定；单独 MSG 默认速度 1。 |

复用且不改外部实现：`zun.Timer`／`zun.RNG`，`anm.World`，`Renderer.Rect`＝FUN_0043bda0，`SpellCard.Bombed` 与原 SE；没有把画面效果塞进 World，也没有复制渲染后端。

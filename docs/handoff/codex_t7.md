# t7：符卡演出续交接

工作树：`D:\GoProjects\th10\.tmp\codex\t7\tree`。更新日期：2026-11-08。

## 1．摘要

- 阶段 A：已完成并通过 rules 第 6 节完整验收。宣言初始化奖励／历史十三个 VM，原始 Shift-JIS 名字仅在字体边界解码；符卡 calc 显式驱动数字、历史与淡出，GUI calc 不再重复步进。
- 修正失败清位：保留奖励 VM 2（`+0xed0`），清 VM 0、1、3～7 的 flag bit 1。清位发生在灵击／被弹／非耐久超时的原作调用点，不在每帧显示中补写。
- 收取按“加分 → GUI 横幅／奖励数字 → 记历史 → 音效”顺序；失败替换 GUI `+0x9e14` 为 `front.anm` script `0x48`，层 15。十三个数字仍在 draw `0x25`，不挪到 GUI `0x2b`。
- 阶段 B：已完成，格式修正后的完整复验通过。三个宣言牌分别用 ascii 1、text 顺序脚本 `0x48`、ascii 2，初始层 15，由原 ANM 改到 14；cdbg 是两个内联 VM，在 draw `0x0e` 绘制；Extra 分支、MSG 打开／结束与 344 共用原始 `DAT_00474c84`。
- 阶段 C：已完成，格式修正后的完整复验通过。ECL 337 保留参数 2 → 1 → 0 求值顺序，构造 duration／start／end 的独立震屏对象，不过滤零幅度或负幅度；calc `0x0e` 先 tick、再检查寿命、再按原采样消耗主 RNG；已移除三角函数方向图案。
- A → B → C 均执行完整 Go／WASM、六章 trace 和完整 smoke；B／C 的授权新增测试行补 gofmt 后又顺序完整复验。共新增 23 个顶层测试；五轮共 30 次 cmp 均与基线逐字节相同，原作 motion 一致帧数没有下降。
- 不修改模式／回放历史门控、存档、自机逻辑或 golden；既有测试唯一变更是明确授权的 drawing `0x0e` 一行。额外 lint 仍有 33 个既有问题，按 rules 第 6 节不属于验收门槛，未声称 lint 全绿。

## 2．改动文件清单

新增：

- `scene/stage/ecl/spell_gui_test.go`：宣言重置、原始名字、真实历史、单次步进、失败／超时与结果脚本的集成测试。
- `scene/stage/spell_gui_test.go`：灵击／被弹的 59／60 帧清位边界及其它 flag 位不变测试。
- `scene/stage/spell_visual_test.go`：三牌文件／位置／World 顺序、cdbg 节点及 helper Active 绘制门控测试。
- `scene/stage/ecl/spell_background_test.go`：各关 cdbg／Extra 两分支、内联所有权、STD 交接及 344 原字段测试。
- `scene/stage/boss_section_test.go`：MSG id＋1／结束清零，以及 Boss 名字不再读 enemy.Type 的测试。
- `scene/stage/ecl/shake_runtime_test.go`：337 求值顺序、寿命／幅度／慢速／同优先级顺序、出生帧／calc 时机和 Field.Shift 落点测试。
- `docs/handoff/codex_t7.md`：分阶段九节交接，另复制到主树同名文档。

修改：

- `scene/stage/bomb.go`：给符卡绑定 helper HUD，`Bombed` 在既有数值写入之后清七个奖励 VM 的 bit 1。
- `scene/stage/spell.go`：按原作顺序调用收取／失败 GUI，生成三个 World 宣言牌，停止数字节点并保留退出动画。
- `scene/stage/gui_spell.go`：修正失败 VM 范围，分离淡出中断与数字步进，接真实历史，避免 GUI／ECL 双驱动。
- `scene/stage/gui_boss.go`：名字纹理接口接收原始 Shift-JIS，在便携字体边界转为 Unicode。
- `scene/stage/gui.go`：移除过渡的 GUI 私有 section／hasSection，名字选择读原共享字段。
- `scene/stage/ecl/game_ops.go`：每次宣言调用 GUI 初始化，接 ECL 344 原字段，删除占位注释。
- `scene/stage/ecl/interrupt.go`：非耐久超时复用原作失败数值／GUI 清位路径。
- `scene/stage/ecl/spell_visual.go`：原淡出调用点发送十三个中断，显式驱动真实奖励／历史；按原 ASM 初始化 cdbg、宣言音效与 Extra cut-in。
- `scene/stage/ecl/vm.go`：Boss HUD 的中 Boss 判别改读 `DAT_00474c84`；以独立对象列表／采样偏移替换旧震屏字段，绑定 calc hook。
- `scene/stage/ecl/shake.go`：kind 1 构造、计时／删除、同优先级顺序、原四步 RNG 采样与纯读取偏移接口。
- `scene/stage/stage.go`：提供原 BossSection 字段及 MSG 赋值；在 STD 后、自机前调用震屏 calc，Draw 只复制已采样偏移。
- `scene/stage/chain.go`：接非 legacy draw `0x0e`，只画 Active 符卡的两个 cdbg。
- `scene/stage/chain_test.go`：仅按授权在 `TestChainDraws.drawing` 加 `0x0e`，其余测试不变。

删除：暂无。

工作树外辅助文件：`.tmp/codex/t7/verify.ps1` 顺序执行完整验收、六章 trace、比较与删除；`.tmp/scripts/t7_dump_anm.py` 只把指定嵌入 ANM 当数据读出，证据存入 `.tmp/codex/t7/evidence/anm_selected.txt`；`.tmp/codex/t7/observe_shake.go` 是开发用的回放观测器，顺序统计震屏创建帧与主 RNG 消耗，非发布代码。

## 3．移植的函数地址

| 原作 | Go 接线／实现 |
|---|---|
| `FUN_00409280`，`0x4093d0`、`0x409400` → `FUN_0043e710` | `VM.declareSpell` → `GUI.InitializeSpellPresentation` → `SpellHUD.start`，八个奖励脚本 `0x3a～0x41`、五个历史脚本 `0x42～0x46` |
| `FUN_00409280`，`0x409422～0x409681` → `FUN_00449870`／`FUN_004489d0` | `SpellCard.SpawnPlacards`，三个 World 宣言牌，ascii 1／text `0x48`／ascii 2；原备用 RNG |
| `FUN_00409280`，`0x409681` → `FUN_00447ae0` | `GUI.SetSpellName(vm, rawName)`，白色、右对齐，原 SJIS 参数保留 |
| `FUN_00408d60`，`0x408e5b～0x408f76` → `FUN_00449470` | `VM.updateSpell` 对三个 World 牌发 label 3／2，再调用 `GUI.InterruptSpellPresentation` 写十三个内联 VM 的 `+0x304` |
| `FUN_00408d60`，`0x408f7c～0x40910e` → `FUN_0043e5a0`／`FUN_0043ee30` | `GUI.UpdateSpellPresentation` → `SpellHUD.update`，收取中八个奖励，失败只 VM 2，随后五个历史 |
| `FUN_00405500`／`FUN_00426cf0`／`FUN_004127a0` | `SpellCard.Bombed` → `SpellHUD.loseCapture`，timer >59 时清七个位；超时额外检查非耐久 |
| `FUN_00409c00`，`0x409cbb` → `FUN_004172e0` | `SpellCard.Finish` → `GUI.ShowCaptureBonus`，只显示奖励，不重复发分 |
| `FUN_00409c00`，`0x409d49～0x409d76` → `FUN_004492a0`／`FUN_00448d00` | `SpellCard.Finish` → `GUI.ShowNotice(1, 0)`，替换 `+0x9e14` 结果句柄 |
| `FUN_00409270` → `FUN_004091c0` | `Stage.nodeSpellHUD`，draw `0x25`，以 helper Active 为门控 |
| `FUN_00409280`，`0x409964～0x409b95` → `FUN_00404f30` | `VM.spawnSpellEffects`，两内联 cdbg；Extra `<0x18` 只启动 VM 0 的 `0x26`，否则 `0x1b／0x1c` |
| `FUN_00409230` | `Stage.nodeSpellBackground`，draw `0x0e`，VM 0 后 VM 1，不入 World |
| `FUN_0040e770`，`0x41188f～0x41189f` → `FUN_00412ed0` | ECL 344 → `Context.SetBossSection(arg0)`；EAX＝`0x474c40`，`+0x44` 即 `DAT_00474c84` |
| `FUN_00415db0`，`0x415e01～0x415e0c`／`FUN_00415e90`，`0x416f3c～0x416f50` | `Context.OpenMSG` 写 id＋1，`Context.UpdateMSG` 在结束时清零 |
| `FUN_00414900`，`0x414e14／24／38／4c／71` | `GUI.calcBoss` 以原 BossSection 的 `<0x18` 选择名字 |
| `FUN_0043c8b0`／`FUN_0043c970`（kind 1） | `newScreenShake`／`VM.queueShake`，timer 初值 0，calc 回调 `0x43c550` 注册在 `0x0e`，不注册 draw |
| `FUN_0043c550` → `FUN_0043c5ca` | `VM.CalcShake`／`sampleShake`，tick → 寿命检查 → `float32((end-start)*F/duration+start)`，每次采样两个 `Dup()%3` |
| `FUN_0043c870`／`FUN_0043cb00` | `VM.CalcShake` 删除到期对象；阶段退出时由本关 VM 生命周期释放，不把对象留给下一关 |

## 4．测试

阶段 A 新增六个顶层测试：

- `TestSpellGUIInitializesEveryDeclaration`：十三个脚本每次重新初始化，SJIS／历史不被显示转换覆盖，机体专属计数及 99 显示上限，ECL／GUI 不重复步进。
- `TestSpellGUIFailureStepsOnlyThirdBonusVM`：失败仍步进 `+0xed0` 与五个历史，其余七个奖励冻结且隐藏。
- `TestSpellGUITimeoutCaptureGate`：非耐久与耐久超时、59／60 帧清位边界。
- `TestSpellGUIResultWorldScripts`：收取 `0x47`＋八个奖励数字，失败 `0x48`，重复 343 不重建结果。
- `TestSpellGUIBombAndHitFailureWrites`：灵击／被弹两条运行时路径的边界。
- `TestSpellGUIFailureDoesNotChangeOtherFlagBits`：仅 bit 1 变化，VM 2 不变。

开发检查：`go test -count=1 ./scene/stage ./scene/stage/ecl -run 'TestSpell|TestGUISpell|TestVMSpellBgDeclaresHUD'` 通过。新增名字测试最初用了不在 Shift-JIS 内的汉字“卡”，编码器拒绝；改用受支持的“符名”，未动已有测试。

阶段 A 完整验收（2026-11-08，03:33～03:38 CET），日志目录：`.tmp/codex/t7/logs/phaseA/`：

| 检查 | 结果 |
|---|---|
| `go build ./...` | 通过，exit 0 |
| `go vet ./...` | 通过，exit 0 |
| `go test -count=1 ./...` | 全部通过，exit 0 |
| `gofmt -l .` | 输出为空 |
| 非 tools 的 WASM 构建 | 通过，46 个包，exit 0；js 下 list 对只支持 Windows 的 th10probe 发诊断，实际非 tools 包均已构建 |
| `TH10_SMOKE=full go test -count=1 -run TestReplayDrawSmoke ./scene/stage/` | 全程通过，136.538 秒 |
| `golangci-lint run`（规则第 6 节明确非验收项，仍按全局指示运行） | exit 1，33 个既有问题：errcheck 26、staticcheck 2、unused 5；没有本任务新代码处的问题，不越界修其它代码 |

补验：本机 `go list ./...` 过滤 tools 后 exit 0，同样 46 个非 tools 包再设 js／wasm 构建 exit 0，见 `wasm_native_list.log`。

阶段 B 新增九个顶层测试：`TestSpellPlacardsWorldFilesAndOrigin`、`TestSpellBackgroundDrawNodeOrderAndActive`、`TestSpellHUDDrawUsesHelperActiveBit`、`TestSpellBackgroundScriptsAndOwnership`、`TestSpellExtraLeavesSecondBackgroundUntouched`、`TestSpellBackgroundSTDTransition`、`TestECL344WritesBossSection`、`TestMSGBossSectionLifecycle`、`TestGUIBossNamesUseOriginalSection`。相关包检查通过。

阶段 B 首轮完整检查（2026-11-08，03:47～03:52 CET），日志 `.tmp/codex/t7/logs/phaseB/`：build／vet／`go test -count=1 ./...` 全部 exit 0，46 个非 tools 包 WASM 构建 exit 0，完整 smoke 通过（116.763 秒）。本机 list 再切 js／wasm 的补验同样 list／build 均 exit 0；指定 js list 的 th10probe 诊断同 A。额外 lint 仍是相同的 33 项既有问题，没有新增处。首轮 gofmt 退出 0 但实际列出新增的 chain 测试行，最终核对时发现并补格式，见本节复验说明。

新增 cdbg 测试初次错误假设 stop 后 ANM 主时钟仍递增。按原 ANM op 63，静态 cdbg 的 T 停在 0，但 alpha／UV 照常更新；颜色循环的叠层则先递增 T。只修新增测试，实际移植与既有规格不动。`chain_test.go` 与基线 diff 已核对，仅授权的一行。

阶段 C 新增八个顶层测试：`TestShake337OperandOrderAndConstructor`、`TestShakeRuntimeSequenceAndLifetime`、`TestShakeRuntimeZeroNegativeAndConstantAmplitudes`、`TestShakeSlowFrameSampling`、`TestShakeEqualPriorityOrder`、`TestShakeOffsetReadIsPure`、`TestShakeStageCalcPriorityAndBirthFrame`、`TestShakeOffsetFeedsFieldOnly`。全部相关检查已通过。

阶段 C 首轮完整检查（2026-11-08，04:01～04:05 CET），日志 `.tmp/codex/t7/logs/phaseC/`：

| 检查 | 结果 |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test -count=1 ./...` | 所有包通过，exit 0 |
| `gofmt -l .` | exit 0，但实际列出新增 chain 测试行；已补格式并安排完整复验 |
| WASM | 46 个非 tools 包 build exit 0；本机 list 的补验 list／build 均 exit 0；js list 的 th10probe 诊断与 A／B 相同 |
| 完整 smoke | `TH10_SMOKE=full` 通过，122.479 秒 |
| 额外 lint | 仍是同一组 33 项：errcheck 26、staticcheck 2、unused 5；无本任务新增问题，不越界修无关代码 |

移植没有让已有测试或 golden 失败，没有重生成 golden。文件哈希审计见 `logs/changed_files.txt`：13 个修改的 Go 文件、6 个新增测试文件、1 个新增交接文档；没有删除。所有既有测试除授权的 chain 一行外均与基线相同。

最终逐项核对发现验收脚本最初只记录 `gofmt` 退出码，漏了“输出必须为空”检查；B／C 的旧 `gofmt.log` 均列出 `chain_test.go`。只对齐授权新增行的注释，diff 仍仅增加这一行，未动其它断言。脚本已改为明确记录 `gofmt-empty: True`，任何必需检查失败都会报错；原首轮日志保留。复验使用当前 A＋B＋C 的完整实现，没有回退或切换测试／golden。

最终完整复验（2026-11-08）：

| 检查 | B 复验，04:16～04:21 CET | C／交付前复验，04:21～04:25 CET |
|---|---|---|
| build／vet／`go test -count=1 ./...` | 全部 exit 0 | 全部 exit 0 |
| gofmt | `gofmt-empty: True`，日志 0 字节 | `gofmt-empty: True`，日志 0 字节 |
| 非 tools WASM | 46 个包，exit 0 | 46 个包，exit 0；本机 list／build 补验均 exit 0 |
| 完整 smoke | 通过，128.537 秒 | 通过，120.111 秒 |
| 六章 cmp／motion | 全相同／原作一致帧数不下降 | 全相同／原作一致帧数不下降 |
| 额外 lint | 同一组 33 项既有问题 | 同一组 33 项既有问题 |

完整日志在 `.tmp/codex/t7/logs/phaseB_final/`／`phaseC_final/`。最终 `gofmt -l .` 另行核对仍为空；没有把 gofmt 的 exit 0 再误记成输出为空。

## 5．Trace 结果

阶段 A 六章顺序生成并比较，六次 `cmp -s` 均 exit 0，与 `.tmp/trace/d1c2/base_cN.txt` 逐字节相同。原作 c0 用 `orig_d.txt`，其余用 `orig_05_s2.txt～orig_05_s6.txt`。

| 章 | 基线 cmp | 原作 motion 一致帧数 |
|---|---|---|
| c0 | 相同 | 15853 |
| c1 | 相同 | 17733 |
| c2 | 相同 | 29763 |
| c3 | 相同 | 29469 |
| c4 | 相同 | 39343 |
| c5 | 相同 | 40099 |

一致帧数没有下降，没有新增主 RNG 消耗；生成的六个大 trace 已比较后逐个删除，只留日志。未使用已弃用的一面 `orig.txt`，未过滤帧或改容差。

阶段 B 同样六次 cmp 均 exit 0，原作 motion 仍为 15853／17733／29763／29469／39343／40099；新增三牌／cdbg／MSG 字段没有改变逻辑 trace。六个大 trace 已逐个删除。

阶段 C 最终结果：

| 章 | 与基线 cmp | 原作 motion | 震屏创建帧 | 首个新增 RNG 帧 | 最后采样帧 | 实帧采样数／新增主 RNG step |
|---|---|---|---|---|---|---|
| c0 | 相同 | 15853 | 15878 | 15879 | 15907 | 29／116 |
| c1 | 相同 | 17733 | 19733 | 19734 | 19762 | 29／116 |
| c2 | 相同 | 29763 | 31444 | 31445 | 31473 | 29／116 |
| c3 | 相同 | 29469 | 31160 | 31161 | 31189 | 29／116 |
| c4 | 相同 | 39343 | 40490 | 40491 | 40519 | 29／116 |
| c5 | 相同 | 40099 | 49747 | 49748 | 49866 | 119／476 |

**首个分歧的说明：逻辑 trace v2 没有分歧帧，六章 cmp 仍为 0；主 RNG 状态／计数的新消耗最早在 c0 第 15879 帧出现。** 原因是第 15878 帧 ECL 337 创建 kind 1 节点，下一帧的 calc `0x0e` 首次执行两个 `Dup()%3`，比旧三角函数占位路径多 4 个主 step。每章只有一次 BossDead 的 `337(30,12,0)`，且没有再改变 trace 中的运动／计分字段；trace v2 本身不采 RNG 计数或绘制偏移，不能把 cmp 相同解释成“没有消耗 RNG”。六章共新增 1056 个主 step，没有为了 cmp 省略任何采样。

c0～c4 的游戏速度为 1，duration 30 在实帧 30 到期，前 29 次采样。六面这段速度为 0.25，因此在实帧 120 到期，前 119 次采样，包括 timer.Cur 未推进的实帧；没有按 Cur／Prev 相等过滤 RNG。逐章观测记录在 `phaseC/shake_cN.log`，六面速度与首帧 RNG 状态补验在 `phaseC/shake_c5_speed.log`。这些观测调用既有 sim，不运行原作；原作 motion 覆盖范围仍全部一致，大 trace 均已比较后删除。

## 6．待决与推测

- A／B／C 的任务范围及最终完整验收均通过；没有等架构师 dump 的阻塞项，没有新增“推测”实现。
- 震屏原 `FUN_0043c550` 的阶段退出门控 `DAT_004918a4 !=0` 返回 7；该全局由关卡启动／析构写入，本任务不扩控制器字段。本实现的对象归本关 ECL VM，场景退出即释放，不跨关继承。
- 字体沿用 t3 已明确的 GDI／D3DX 便携近似，不声称字形、度量、抗锯齿或阴影逐像素一致。
- `DAT_00474c8c` 的独立“section 计时”属控制器／回放范围，本任务按说明只接 `DAT_00474c84` 本身。原 `FUN_00412ed0`／MSG 在 section 变化时还清该时钟；t8 可在共享 `Context.SetBossSection` 里补条件清零，不应另造一份 Boss 选择字段。
- `FUN_00404f30` 写内联 VM 的两个字节 `+0x3a0／+0x3a1＝0x10`，沿用既有 STD 适配：这些无绘制影响的元数据目前未在 ANM VM 建模；位置、Reset／Start、原脚本、层、alpha／UV／颜色时序均已移植。
- 本阶段没有新猜测、没有尝试 Ghidra、没有 dump 请求。已读取预先 dump 与现有 ASM。

## 7．建议的 `docs/fidelity.md` 改动

以下为最终完整建议行；没有直接修改 fidelity：

```markdown
| 符卡状态（灵击读取部分） | 位精确（逻辑）；演出已接，字体光栅化近似 | DAT_004776f4（stage.SpellCard）：FUN_00409280 按 int32 顺序计算 (stage*3+10)*(Faith/10)*10，+0x3794 上限 99999999；FUN_00408d60 在 calc 0x16 的 pre-tick timer >=300、非耐久时按整数公式衰减并清个位；FUN_00409c00 只凭收取 bit 1 加分，再显示 GUI／记内存历史／播放 SE，343 不清弹／激光／道具；FUN_00405500／00426cf0／004127a0 在 timer >59 失去收取并清七个奖励 VM 的 bit 1，耐久超时保留收取；历史回放门控及持久化归原状态／存档层，本任务不改 | 原 Spell 测试与新增 GUI 接线测试；三阶段六章 cmp 相同，motion 一致帧数不下降 |
| Boss HUD/符卡名 | GUI 与符卡演出已接线；字体近似，待用户目视 | FUN_00414900 的血条每帧 float32 +0.025、扣血立即下降；347 保留四个有序比例／ARGB，360 保留星数，倒计时／箭头沿 t3 原 VM；Boss 名字读 DAT_00474c84，344 按原参数写该字段，MSG 打开写 id＋1、结束清零。FUN_00409280 每次初始化八奖励／五历史 VM；三牌为 ascii 1／text 顺序脚本 0x48／ascii 2，初始层 15，原脚本改到 14 并设置位置；FUN_00447ae0 参数取 SpellCard.Name 原始 SJIS。FUN_00408d60 在 calc 0x16 读取真实 SpellCounts，按奖励／失败 VM 2／历史顺序步进，淡出 label 3／2 同步三牌和十三个数字；失败保留 +0xed0，其它七个清 bit 1；收取显示 Get Spell Card Bonus／八数字，失败 front 0x48，数字仍在 0x25，GUI 0x2b 不重复驱动 | 新增 SpellGUI／BossSection 测试；A／B／C build／vet／test／WASM／完整 smoke 通过，gofmt 清零；六章 cmp 相同；字体不称逐像素相同 |
| 符卡演出（M6） | 已移植；字体光栅化近似，待用户目视 | FUN_00409280 的三个宣言牌及 bullet 0x1a1／0x1ab／stgenm cut-in 在 World，备用 RNG；cdbg 是 helper +0x10／+0x3bc 的两个内联 VM，由 FUN_00404f30 初始化，FUN_00408d60 在 timer 前步进，FUN_00409230 在 draw 0x0e 按 VM 0／1 绘制；STD 在 pre-tick timer >=60 隐藏、343 恢复；Extra DAT_00474c84 <0x18 时只启 cdbg 0x26、cut-in 0x28，否则 cdbg 0x1b／0x1c、cut-in 0x1e，不用符卡 id 或 enemy.Type 替代 MSG／344 字段；三牌结束发 label 1，保留退出动画 | World 文件／脚本／层／原位置、内联所有权／步进、STD 交接、Extra 两分支测试；完整 smoke 通过 |
| ECL 337 震屏 | 位精确（构造、计时、采样及 RNG 接线）；待用户目视 | FUN_0043c8b0／0043c970 的 kind 1 注册 calc 0x0e（STD 后、自机前），ECL 参数按 2→1→0 求值，创建当帧不采样；FUN_0043c550 先 tick、达到 duration 取 return 7 删除，否则 FUN_0043c5ca 按 float32 (end-start)*F/duration+start 采样，每轴 Dup()%3 选 0／正／负，一次 4 个主 RNG step，包括零幅度与 timer.Cur 未推进的实帧；同优先级新节点先跑，最后节点覆盖共享偏移。ShakeOffset 只读采样值，经 Stage.Draw→Field.Shift→Supervisor→ANM.Shift→emit 平移，不改逻辑坐标／viewport；draw 0x24 清偏移保护 GUI，0x32 再清 Field | 新增 ShakeRuntime 八项、原 sampleShake 精度／序列测试；六章逻辑 cmp 相同，新增主 RNG step 共 1056；motion 15853／17733／29763／29469／39343／40099 不下降，完整 smoke 通过 |
```

更新记录建议：

```markdown
- 2026-11-08 — t7 阶段 A：将 t2 符卡与 t3 GUI 显式接线，宣言重置十三个数字 VM，原始 SJIS 名字在字体边界解码；真实机体历史读取并只对显示钳到 99。奖励／历史／失败步进仅由 calc 0x16 驱动，淡出沿原严格阈值和 label 3／2 同步；修正失败清位为保留 +0xed0，timer >59 在灵击／被弹／非耐久超时调用点清其它七个 VM 的 bit 1。收取按加分、GUI、历史、音效顺序，失败替换 front 0x48，draw 0x25／0x2b 分工不变。新增六项集成／边界测试通过；完整 build／vet／test／WASM／smoke 通过，gofmt 清零；六章 trace 与基线逐字节相同，原作 motion 15853／17733／29763／29469／39343／40099 不下降。额外 lint 仍报 33 项既有问题。B／C 未完成，不写 DONE；既有测试断言和 golden 未改。
- 2026-11-08 — t7 阶段 B：按宣言 ASM 补齐三个 World 牌（ascii 1／text 顺序脚本 0x48／ascii 2，备用 RNG、初始层 15，原脚本位置不二次平移）、宣言 SE 14、两个内联 cdbg 与 draw 0x0e。STD 前 60 帧与 cdbg 渐入交接，343 恢复 STD，三牌通过 label 1 退出。Extra 的 <0x18 分支只初始化 cdbg 0x26，另分支用 0x1b／0x1c；cut-in 为 0x28／0x1e。344 经 FUN_00412ed0 写 DAT_00474c84，MSG 打开写 id＋1、结束清零，Boss 名字和 Extra 都读此字段，不用 enemy.Type 或符卡 id 代替。新增九项测试通过；完整 build／vet／test／WASM／smoke 通过，gofmt 清零；六章 cmp 相同，原作 motion 不下降。仅按授权在 TestChainDraws 加 0x0e，其它既有测试／golden 未动。C 尚未完成，不写 DONE；额外 lint 仍报相同 33 项既有问题。
- 2026-11-08 — t7 阶段 C／最终交付：替换 ECL 337 的三角函数占位，按 FUN_0043c8b0／0043c970 构造 kind 1，接 calc 0x0e；先 tick、再寿命检查、再按 FUN_0043c5ca 消耗两个 Dup()%3。零／负／常量幅度与 Cur 未推进帧都保留采样，多个同优先级对象按后注册在前的原顺序运行，Draw 只读 Field.Shift，不移动逻辑坐标或 viewport。新增八项测试通过，完整 build／vet／test／WASM／smoke 通过，gofmt 清零；三阶段共 23 项新增测试。六章最终 cmp 仍相同，原作 motion 15853／17733／29763／29469／39343／40099 全部匹配。每章仅 BossDead 一次震屏：前五章各 29 次采样／116 主 step，六面速度 0.25、119 次／476 step，共 1056；首个新增 RNG 消耗在 c0 15879 帧，逻辑 trace v2 没有分歧帧，不能据此声称 RNG 未变化。额外 lint 仍报 33 项既有问题（规程明确非验收项），没有越界修正。字体仍为便携光栅化近似，目视项目列于交接第 8 节；模式／存档／自机边界不改，golden 未重生成。
- 2026-11-08 — t7 最终格式复验：核对发现 B／C 首轮 gofmt 日志列出授权新增的 chain 行，原脚本误只看退出码；只对齐新增行的注释，既有测试 diff 仍仅一行。验收脚本增加输出为空与必需失败检查，保留旧日志，顺序重跑 B／C 全量 build／vet／test／WASM／完整 smoke／六章 trace，全部必需验收通过，gofmt-empty 明确为 True；五轮 30 次 cmp 均相同，motion 不下降。额外 lint 仍报同一组 33 项既有问题；不删日志、不改断言或 golden。
```

## 8．需要用户目视确认的项目

在工作树按 rules 设置 Go／临时目录后，可用 `go run ./cmd -debug-stage 1 -debug-diff easy -debug-power 500`，或将 stage 换为 6／extra。需要确认：

- 宣言名字与八位奖励，真实历史的两个两位数及分隔符。
- 自机上移到 y <96 且符卡达到 120 帧后的淡出，y >128 恢复；等号不应改变状态。
- timer >59 后灵击／被弹只剩失败标记，历史仍更新；成功结算显示 Get Spell Card Bonus 与奖励，失败显示原失败横幅。
- 三个宣言牌：ascii／text 按原脚本从下方放大宣言后上移到顶部，结束只发 label 1，让退出动画滑出，不立即删名牌。
- cdbg：前 60 帧 alpha 渐入与 STD 同时可见，下一次 pre-tick＝60 时 STD 隐藏；结束 STD 恢复。1 面背景 VM 0 是 b，VM 1 是 a；Extra 中 Boss 只有 `cdbg06a`，Boss 使用 `cdbg07a／b`。
- Extra 可运行 `go run ./cmd -debug-stage extra -debug-diff extra -debug-power 500`，看中 Boss 神奈子与 Boss 诹访子的名字／立绘／背景区别。
- 震屏：正常速度的 `337(30,12,0)` 创建当帧不采样，从下一帧以 11.6 开始衰减，共 29 次离散的 0／正／负偏移；六面尾部速度 0.25 则实帧采样 119 次。看场地裁切不移动、自机／敌弹逻辑坐标不抖、draw `0x24` 之后的 GUI／宣言牌不受震动。绘制偏移落点另有假设备单测，不把完整 smoke 当作像素级验收。

## 9．与其他任务的接口

- t6：继续调用既有 `(*SpellCard).Bombed()`，签名不变；本任务只在原奖励／Capture 写入后附加 helper HUD 清位。未改自机或 `Context.hitPlayer`。
- t8：继续提供 `GameState.SpellCounts(id int32) (captures, attempts int32)`；本任务只读取。历史写入仍由 t2 的声明／收取方法承担，现有 `GameState.ReplayMode` 门控未改，未接存档。
- t3 已有接口：`GUI.UpdateSpellPresentation(bonus int64, capture bool, captures, attempts int32, frames int32)`、`GUI.EndSpellPresentation()`、`GUI.ShowCaptureBonus(amount int64)`、`GUI.ShowNotice(kind int, amount int64)`、`GUI.SetSpellName(vm *anm.VM, rawName string)` 已真实接入。`BeginSpellPresentation` 保留独立 GUI 使用的兼容入口，真正 ECL 宣言不用它重复生成名字。
- 新接口：`GUI.InitializeSpellPresentation()` 只初始化十三个内联 VM；`SpellCard.SpawnPlacards(ctx *Context)` 随后生成三个 World 牌并调用原 SJIS 名字接口。
- 新字段／接口：`Context.BossSection int32`、`Context.SetBossSection(n int32)` 表达 `DAT_00474c84`；344 写原参数，成功打开 MSG 写 id＋1，MSG 结束写 0。`GUI.SetBossSection(n)` 写同一字段，GUI／Extra 都只读它。独立 section 时钟交 t8。
- 新接口：`GUI.InterruptSpellPresentation(label int16)`，供 `FUN_00408d60` 在三个 World 中断之后写十三个内联 VM 的标签。
- `SpellCard.HUD *SpellHUD` 指向十三个 helper VM；GUI 只供应 front 文件／字体，draw 仍在 `0x25`，不是 GUI `0x2b` 的数组。
- 共享 ANM `Start`／`SetSprite`／`Run`／`World.Interrupt`／`World.SpawnTail` 分别复用原 `FUN_0043e710`／`0043e5a0`／`0043ee30`／`00449470`／`00448d00`，未另写第二套后端。
- C 新接口：`Context.CalcShake func()` 指向 `(*ecl.VM).CalcShake()`，由 ECL Update 绑定；Stage 在 calc `0x0e` 调用。各关既有 `ShakeOffset() (float64, float64)` 签名不变，但改为读取已经采样的两个 float32；Draw 不消耗 RNG，也不更新计时。
- 震屏仍走现有 `ShakeOffset → Stage.Draw → Field.Shift → Supervisor → ANM.Shift → emit`。draw `0x24`／`0x32` 的清位继续复用共享 Supervisor；未移动 viewport 或实体坐标，未改自机逻辑。

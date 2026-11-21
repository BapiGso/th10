# t16 交付：收尾统一

日期：2026-11-08。工作树：`D:\GoProjects\th10\.tmp\codex\t16\tree`。A–E 必需验收全部通过；全仓 lint 仍有范围外告警，不称全仓 lint 全绿。

## 1．摘要

阶段 A 先成文普查；B 删除死路径；C 将七关包收缩成声明级纯配置，仍执行的旧背景和安全 cap 原样移到共用包；D 以 t10 为主合并根操作、淡出、小字库和方向重复，并将 t11 ANM-91 补发放到同一共用位置；E 加入唯一 lint 配置、清零本范围告警。A–E 各阶段完整 Go／WASM／格式／smoke／六章 trace 验收均通过。不移植关内逻辑、不调整演出或状态机时序、不删除仍可执行的 `chain_legacy.go`／旧背景。

| 阶段 | 当前状态 |
|---|---|
| A．普查 | 完成；全量 Go／WASM／格式／完整 smoke／六章 trace 通过。 |
| B．删除不可达代码 | 完成；全量验收通过，原测试保护检查通过。 |
| C．关卡包纯配置化 | 完成；全量验收通过，15 个背景函数体不变，七包纯配置测试通过。 |
| D．共用演出统一 | 完成；8 组演出摘要相同，原状态／时序测试及全量验收通过。 |
| E．lint 统一 | 完成；配置验证、本范围清零、最后完整验收通过，全仓 46 项范围外告警列明。 |

## 2．改动文件清单

新增：`docs/research/cleanup.md`，可达性、关卡手写行为、重复实现与 lint 取证；`docs/handoff/codex_t16.md`，分阶段交付记录。

阶段 C 新增：`scene/stage/script/script.go`，七关共用 ECL 包装器；`scene/stage/script/legacy_background.go`，原样保留仍可执行的七关旧背景；`scene/stage/script/config_test.go`，纯配置／旧函数签名／首次 calc 前 hook 与资源配置测试；`scene/stage/script/script_test.go`，安全结束／float64 滚动／nil VM 可选接口测试。

阶段 C 修改：`scene/stage/stage1/ecl_script.go`、`scene/stage/stage2/stage2.go`、`scene/stage/stage3/stage3.go`、`scene/stage/stage4/stage4.go`、`scene/stage/stage5/stage5.go`、`scene/stage/stage6/stage6.go`、`scene/stage/extra/extra.go`，每个包只留配置、构造方法值与兼容类型别名，没有函数体或状态字段。

阶段 D 新增：`scene/ui/presentation.go`，共用 TitleAnimations、ANM-91 LoadingEmitter 和小字库提交；`scene/ui/presentation_test.go`，根句柄所有权和补发 opt-in／出生 pass 测试；`input/menu_unified_test.go`，与旧方向查询等价和相反方向只读快照测试。

阶段 D 修改：`scene/ui/title_controller.go`、`scene/ui/title_pages.go` 调用共用根操作／World step／小字库；`scene/ui/fade.go` 保留唯一 Calc 并拆出无 Use／Fog 变更的 DrawRect；`scene/menus/native.go` 调用共用根操作、补发和 Fade；`scene/menus/musicroom.go`、`scene/menus/playerdata.go`、`scene/menus/replay.go`、`scene/stage/menu.go` 方向只读 MenuInput.Move；`scene/ending/fade.go` 只保留兼容观察和生命周期适配，不再有第二份 alpha／timer 算法。

阶段 E 新增：`.golangci.yml`，标准 linter＋unused、包含测试、无统计截断；`scene/dialog/msg_wait_gate_test.go`，八种 skip／advance 门控和声音分支测试。修改：`scene/dialog/msgvm.go` 顺序展开原短路门控并清掉已删 overlay 的注释；`scene/stage/ecl/vm.go` 保留 execStackMath 调用而去掉空分支；`scene/stage/stage.go` 删除恒为 nil 的内部 error 返回及不可能触发的生产错误分支；`scene/ending/fade.go` 去掉本任务新增代码中的冗余嵌入字段选择器。

修改：`input/input.go` 删除旧 Snapshot；`scene/dialog/msg.go`、`scene/dialog/msg_test.go` 删除旧表情后缀助手及其专用测试；`scene/menus/replay.go`、`scene/menus/replay_test.go` 删除旧标签助手及其专用测试；`scene/select/select.go` 删除独立 Select 壳；`scene/ui/ui.go` 删除无调用者的 UI 壳和构造助手；`scene/ui/title_controller.go` 删除 RestartChild；`scene/stage/ecl/game_ops.go`、`scene/stage/ecl/vm.go`、`scene/stage/ecl/opcodes.go` 删除 unused 助手／旧别名；`entity/enemy/enemy.go`、`collision/collision.go` 删除旧手写关卡助手，保留仍执行的更新、矩形碰撞和原 hurt／flags 测试接口。

删除文件：

| 文件 | 原文件行数 | 原因 |
|---|---:|---|
| `Deprecated/stage1/stage1_hardcoded.go` | 311 | ignored 的旧手写关卡，无入口。 |
| `scene/dialog/dialog.go` | 13 | 只为旧台词保留的 Line 类型。 |
| `scene/dialog/scripts.go` | 273 | 只有 ignored 关卡调用的近似台词表。 |
| `scene/ui/menu.go` | 141 | 无调用者的旧 MenuList。 |
| `scene/menus/option.go` | 42 | 无调用者的独立 Option 壳。 |
| `scene/stage/ecl/disasm.go` | 145 | 无游戏、工具或测试读者的旧文本反汇编助手。 |

阶段 B 共删除 6 个文件、925 行；包含局部助手／导入／专用测试的纯删除总计 1302 行，新增源码 0 行。日志：任务根 `logs/changes-B.tsv`。阶段 C／D 的原样移动和统一不混入这份“死代码删除”统计。

最终 Go 源码差分为 43 个文件：8 个新增、29 个修改、6 个删除；1247 行新增、2515 行删除，净减少 1268 行。该总数包含测试和代码移动，与上述 1302 行“死路径纯删除”不是同一口径。`.golangci.yml` 与两份新增文档另计；`go.mod`／`go.sum` SHA-256 均与基线相同，没有新依赖。

最终全树范围核对（跳过只读 junction、工具缓存和编辑器目录）共 863 个文件，46 个改动均在本任务清单内，范围外改动 0；资源、音频、工具、trace、存储与既有 golden 文件保持原字节。日志：任务根 `logs/scope-audit.txt`。

## 3．移植的函数地址

本任务不移植新行为。复用的原作函数包括：`FUN_0042c670／0042c6d0／0042c770`（标题根操作），`FUN_0043bd40／0043c230／0043c1a0`（淡入淡出），`FUN_00401760`（小字库），`FUN_0044a950／0040ace0`（菜单按位重复）。实际 Go 签名见第 9 节。

阶段 C 仅合并现有调用点：`FUN_00417870 → stagescript.Script.Init`（BindContext 先于首个 calc），`FUN_0040d750／0040dc80 → stagescript.Script.Update` 的 VM 转发，`FUN_00408d60 → stagescript.Script.LateUpdate`，`FUN_0043c550／0043c5ca → stagescript.Script.ShakeOffset`。frame cap／旧背景不是新移植，实际地址／缺口见第 6 节。

阶段 D 的复用对照：`FUN_0042c670／0042c6d0／0042c770 → ui.TitleAnimations.Spawn／Leave／Interrupt`；`FUN_0043ee30` 的 `0x43ef55／0x43f00d → ui.LoadingEmitter.afterRun`；`FUN_00448860 → ui.TitleAnimations.Run`；`FUN_0043bd40／0043c230 → ui.Fade.Calc`；`FUN_0043c1a0／0043c2c0 → ui.Fade.Draw／DrawRect`；`FUN_00401760 → ui.GlyphTextFile`；`FUN_0044a950／0040ace0 → input.MenuInput.Update／Move`。关内／结局 MSG 仍分别是 `FUN_00415e90／0040bd80`，没有合并。

## 4．测试

原断言、期望值、容差与 golden 均未修改。已按授权逐项删除：

| 测试 | 文件 | 理由 |
|---|---|---|
| `TestMSGFaceExpressionMapping` | `scene/dialog/msg_test.go` | 只验证被删的 PNG 表情后缀字符串映射；原 MSG 立绘按 ANM face id，其他 MSG／Presentation 测试保留。 |
| `TestReplayEntryLabels` | `scene/menus/replay_test.go` | 只验证被删的旧 replayEntryLabel 字符串；同文件 `TestReplayMenuLoadsStorage` 和 t11 列表／详情／选章／返回测试原样保留，只清掉不再需要的 strings 导入。 |

对基线所有 `_test.go` 的函数源码做 SHA-256 比较：982 个声明中保留 980 个、授权删除 2 个、修改 0 个。相关 `scene/menus`／`scene/ui`／`scene/select`／`scene/dialog`／`input` 测试通过。

阶段 C 新增并通过：`TestStagePackagesContainOnlyConfiguration`、`TestStageConfigurationFactoriesBindBeforeFirstCalc`、`TestCommonScriptKeepsSafetyCapAndRealFrameScroll`、`TestCommonScriptNilVMOptionalInterfaces`。另将七个 BgDraw 和 stage1 八个相机／雾／云辅助函数体与基线逐文本比较：15 个函数体不变、修改 0 个；`background-guard-C.txt`／`test-guard-C.txt` 保留结果。

阶段 D 新增并通过：`TestUnifiedDirectionRepeatMatchesLegacyQueries`、`TestUnifiedDirectionRepeatPreservesOppositeKeySnapshots`、`TestSharedTitleAnimationsKeepOptionalRootOwnership`、`TestSharedLoadingEmitterStaysOptInAndSkipsBirthPass`。原 t10／t11 标题、选人、Option、MenuInput、EndingFade、三菜单测试通过；旧测试函数保护仍为 980 保留、2 授权删除、0 修改。

阶段 E 新增 `TestMSGWaitKeepsSeparateSkipGatesAndAdvanceSound`，八种组合通过。没有修改仍可达行为的原测试；恒 nil error 返回由运行时代码去掉，不用 `_ =` 或 nolint 修改原测试。MSG 的 QF1001 用两次原顺序条件判断解决，没有 De Morgan／批量 modernize 改写；ECL 保留函数求值副作用，只有空分支删除。本任务没有新增 nolint 或排除规则。

任务根 `scenedigest.go` 用记录 Supervisor 验证变更前后的八组采样：title intro／main／difficulty 各 180 帧；Music／Data／Replay 各 180 帧；Ending 2400 帧；Staff 5200 帧。摘要包含逐帧备用 RNG、设备调用和全部绘制顶点，标题另含状态／计时器／Selector／World 长度。`logs/scenes-before-D.txt`、`logs/scenes-after-D.txt`、`logs/scenes-final.txt` 八行 SHA-256 逐字节相同，两次 cmp 退出 0；不是像素 golden，也没有重生成已有 golden。

阶段 A／B／C／D／E 的必需验收全部通过，日志位于任务根 `logs/phase_A/` … `logs/phase_E/`。最终 E 于 `2026-11-08T13:02:20+01:00` 结束，完整 smoke 包含全部回放章，测试输出 `ok th10/scene/stage 115.833s`。每次都使用 `-count=1`，不以缓存命中冒充重新验收。

| 阶段 | build | vet | test | WASM | gofmt 空输出 | full smoke |
|---|---|---|---|---|---|---|
| A | 0 | 0 | 0 | 0 | 是 | 0 |
| B | 0 | 0 | 0 | 0 | 是 | 0 |
| C | 0 | 0 | 0 | 0 | 是 | 0 |
| D | 0 | 0 | 0 | 0 | 是 | 0 |
| E／最终 | 0 | 0 | 0 | 0 | 是 | 0 |

运行命令为 `go build ./...`、`go vet ./...`、`go test -count=1 ./...`、`gofmt -l .`；WASM 用 `$env:GOOS='js'; $env:GOARCH='wasm'; go build (go list -e ./... | Where-Object { $_ -notmatch '/tools/' }); Remove-Item Env:GOOS,Env:GOARCH`；smoke 用 `$env:TH10_SMOKE='full'; go test -count=1 -v -run TestReplayDrawSmoke ./scene/stage/; Remove-Item Env:TH10_SMOKE`。WASM 只有 tools 被排除，所有运行时包实际构建。

开发期已修正的失败：验收脚本 Tee-Object 参数集不能混用；WASM 枚举中的 Windows-only tools 要用 `go list -e ./...` 列举再排除，实际 runtime 包 build 仍必须成功；删除旧回放标签后残留 time 导入，编译器指出后删除，相关测试重跑通过。它们不作为基线失败或遗留失败。

### lint 版本、命令与数量

版本：`golangci-lint 2.15.0`，构建 Go 1.27.0，提交 `25c5369a`，构建日期 `2026-10-16T11:03:09Z`；本机 `go1.27.3 windows/amd64`。配置已通过 `golangci-lint config verify`。

唯一常规命令：`golangci-lint run`。默认 standard＋显式 unused，tests=true、并发 2、timeout=10m、modules=readonly，max-issues-per-linter=0、max-same-issues=0、uniq-by-line=false。无新增依赖、现代化 linter、告警豁免或范围排除。JSON 捕获的输出参数不改变这套统计。

| linter | 改动前统一口径 | 改动后 | 本范围剩余 |
|---|---:|---:|---:|
| errcheck | 44 | 42 | 0 |
| govet | 0 | 0 | 0 |
| ineffassign | 0 | 0 | 0 |
| staticcheck | 4 | 2 | 0 |
| unused | 4 | 2 | 0 |
| 总计 | 52 | 46 | 0 |

历史默认截断口径在本树实测为 32；阶段 B／C 无截断各 50，D 为 51（新适配有一个 QF1008，E 已修）；非测试 unused 基线为 18。它们不再与唯一常规口径混称。最终命令实测退出 1，原因仅为下表 46 项范围外告警，不能宣称全仓 lint 全绿。原始记录：`logs/lint-before.*`、`logs/lint-after.*`、`logs/lint-remaining.tsv`。

| 剩余包 | errcheck／staticcheck／unused | 总数 | 归属与处理 |
|---|---|---:|---|
| `audio` | 3／2／0 | 5 | t14；两个 deprecated Player.Close 与三次 Close 返回值，只读不改。 |
| `datfs` | 3／0／0 | 3 | 历史 M0 archive／测试，未在本轮分配；与 t14 资源校验有关，非 t16。 |
| `gamedata` | 1／0／0 | 1 | t14；embed_test 的 Close，不改。 |
| `gfx` | 0／0／1 | 1 | 历史 M4 renderer，未分配；pipeline_test 的未用 near，不改仍可达绘制规格。 |
| `sim` | 1／0／0 | 1 | 历史 M0 runner，未分配；调试 fmt.Fprintf，非 t16，留架构师另派。 |
| `tools/th10diff` | 3／0／0 | 3 | t17；两个文件 Close 与测试 Close，不改。 |
| `tools/th10extract` | 2／0／0 | 2 | t14；文件 Close，不改。 |
| `tools/th10pix` | 16／0／0 | 16 | 历史像素工具，本轮未分配；fmt 输出／文件 Close，用户明确禁止改 tools。 |
| `tools/th10probe` | 12／0／0 | 12 | t17；句柄／文件／GDI 返回值，不改。 |
| `tools/th10sim` | 1／0／0 | 1 | t17；输出文件 Close，不改。 |
| `trace` | 0／0／1 | 1 | t17；Reader.err 字段，不删。 |

剩余位置按包见 `logs/lint-remaining.tsv`；数据和测试所属代码没有越界修正。必需 Go／WASM／trace／smoke 验收与额外 lint 的失败口径分开记录。

## 5．Trace 结果

阶段 A–E 六章均逐章生成，30 次 `cmp -s -- base_cN cN` 全部退出 0，没有首差帧。五轮 motion 一致帧数全部不降，原作参考和最终结果如下。

| 章 | 原作参考 | A–E 的 cmp | 最终 th10diff motion |
|---|---|---|---|
| c0 | `orig_d.txt` | 五次退出 0 | MATCH 15853 |
| c1 | `orig_05_s2.txt` | 五次退出 0 | MATCH 17733 |
| c2 | `orig_05_s3.txt` | 五次退出 0 | MATCH 29763 |
| c3 | `orig_05_s4.txt` | 五次退出 0 | MATCH 29469 |
| c4 | `orig_05_s5.txt` | 五次退出 0 | MATCH 39343 |
| c5 | `orig_05_s6.txt` | 五次退出 0 | MATCH 40099 |

每章命令：`go run ./tools/th10sim -replay ".tmp/[th10]/replay/th10_05.rpy" -chapter N -frames 0 -out D:\GoProjects\th10\.tmp\codex\t16\trace\cN.txt`；与 `D:\GoProjects\th10\.tmp\trace\d1c2\base_cN.txt` 用 cmp 比较，然后 `go run ./tools/th10diff -a <上述原作参考的绝对路径> -b <cN绝对路径> -scope motion`。所有生成的 cN 已在比较后删除；任务根 trace 目录最终为空。没有改 probe、schema、差分阈值、采样过滤或 golden。

## 6．待决与推测

- `TH10_LEGACY_BG=1`、STD 缺失和非转换关仍可执行 `chain_legacy.go → stagescript.Script.BgDraw`；七关 float64 相机／滚动／PNG／vector 背景已经原样移到 `scene/stage/script/legacy_background.go`。对应正常原路径 `FUN_00403c80／00402850／00402ca0`，旧经验系数没有逐项原地址，不能宣传为移植。
- 七关 60000 帧 cap 与 MainDone 的安全结束仍执行，集中到 `scene/stage/script/script.go`。原控制器为 `FUN_00418190`／MSG `FUN_00415e90`，60000 本身没有原作地址，后续任务另行移植。
- 七关未绘制图片的加载调用仍实际发生，不按 unused 字段推断删除资源；贴图／字体加载由 t14 负责。
- `Enemy.Update` 仍在每帧 Stage 中执行，旧 path／Boss nil 分支不是本轮移植对象。对应原运动／阶段函数 `FUN_0040dc80`，本轮仅删无调用者的旧公开助手。
- t10 难度页的 ASCII script 8 仍只有通用 ANM-91 no-op；t11 已调用共用 16＋周期补发。不能在本轮给 t10 增加此前不存在的 VM／备用 RNG／像素变化。原指令地址 `FUN_0043ee30` 的 `0x43ef55／0x43f00d`。实现已经统一，但缺失演出和持久单对象生命周期不在本轮改行为。
- 原结局 MSG `FUN_0040bd80` 与关内 MSG `FUN_00415e90` 不同，保持独立；不合并 t10／t11 状态机或时钟。
- 没有新增 Ghidra／动态 dump 请求。未证明的候选、原 native 规格测试接口、其他任务的贴图／字体／存储／工具代码仅列出，不删。

## 7．建议的 `docs/fidelity.md` 改动

没有直接修改总账。建议替换原 STD 行，并新增整理／共用演出／lint 状态行；不升级其他未验证的原作／L3 状态。

```markdown
| STD 背景（绘制） | 移植（待用户目视；旧对比路径仍可执行） | 七面默认 std/stddraw.Node7／Node0A，FUN_00402850／00402ca0；TH10_LEGACY_BG=1 或缺失 STD 仍经 chain_legacy.go，手写背景原样集中到 scene/stage/script/legacy_background.go，不称近似已移植 | TestSTDSmoke、15 个旧背景函数体不变、五轮完整 smoke、六章各五次 cmp 相同；L3 待用户 |
| 关卡包／旧路径整理 | 纯配置化；原行为不变 | stage1–6／extra 无函数体或状态，Config 绑定共用 stagescript.Script；删除 6 个死文件 925 行，局部死路径总计 1302 行；保留 60000 帧人为 cap、旧滚动／相机／PNG 加载，详见 docs/research/cleanup.md | 纯配置／兼容签名／首个 calc 前 BindContext 测试，原 980 个测试函数不变、授权删除两项旧近似测试，六章 trace 不变 |
| 标题／菜单共用演出 | 实现统一；不新增缺失演出或改生命周期 | FUN_0042c670／0042c6d0／0042c770 → ui.TitleAnimations；ANM-91 0x43ef55／0x43f00d → ui.LoadingEmitter；Fade.Calc／DrawRect、GlyphTextFile 与 MenuInput.Move 共用；t10 仍 Run(nil)，t11 opt-in；结局／关内 MSG 保持不同函数 | 原 t10／t11 状态／时序测试不变，八组状态／备用 RNG／设备调用／绘制顶点摘要相同，六章 cmp 与完整 smoke 通过；L3 未升级 |
| lint 口径 | 已统一；t16 范围清零，全仓非全绿 | golangci-lint 2.15.0，仓库 .golangci.yml，default standard＋unused，含测试、无同文／同行截断；唯一命令 golangci-lint run | 统一基线 52 → 46：errcheck 44→42、staticcheck 4→2、unused 4→2，govet／ineffassign 0；46 项范围外按包列在 codex_t16.md |
```

更新记录建议：

```markdown
- 2026-11-08 — t16：先完成入口／工具／测试可达性和场景重复普查，再删除 ignored 手写一面、旧台词／后缀、旧 MenuList／Select／Option 壳和无调用者的助手；6 个死文件 925 行，局部纯删除共 1302 行。七关包收缩成配置／方法值／类型别名，共用 ECL 包装器，仍执行的旧背景与 60000 帧 cap 原样保留；15 个背景函数体和 980 个原测试函数不变，只按授权删除两项旧后缀／标签测试，新增 9 项测试。以 t10 为主共用根操作、淡出、小字库和按位方向重复，t11 ANM-91 补发移入共用 opt-in，t10 原 no-op 与两边生命周期不改；八组演出顶点／设备调用／备用 RNG 摘要前后一致。A–E 五轮 build／vet／test／WASM／gofmt／完整 smoke 全部通过，六章 30 次 cmp 相同，motion 15853／17733／29763／29469／39343／40099 不降。加入 golangci-lint 2.15.0 唯一配置与命令，本范围告警清零，全仓统一 52→46，剩余范围外，不称全仓 lint 全绿。没有新增依赖、修改工具／资源／存储或重生成 golden；L3 与原缺失演出仍待后续。
```

## 8．需要用户目视确认的项目

本任务不新增演出或改时序；原 V8–V14 的 L3／试听项目仍有效。清理后应特别核对 `TH10_LEGACY_BG=1` 背景、标题到三菜单的背景生命周期、回放确认淡出、结局／Staff 淡出与菜单方向长按。完整 smoke 只证明稳定性，不宣称逐像素相同。

- 在工作树由用户运行 `go run ./cmd`，按原方式进入标题、难度和三菜单，核对背景脚本和文字／淡出层序没有新增或丢失。
- 旧背景对比：`$env:TH10_LEGACY_BG='1'; go run ./cmd -debug-stage 1; Remove-Item Env:TH10_LEGACY_BG`，其余关用 2–6／extra；PNG 平铺、stage1 云／雾与震屏裁剪应保持旧路径，不因配置化转成新画面。
- 场景 B 可用既有 `go run ./tools/th10scenes -scene staff`、`-scene ending`、`-scene music`、`-scene data`、`-scene replay`，观察原等待、回放确认与黑幕；长按方向在第 26／34／42…帧重复，两相反方向仍按原代码顺序处理。

## 9．与其他任务的接口

- t14 的 `assets/`／`gamedata/`／`audio/`、ANM 贴图和字体加载只读不改；旧 sprite 加载与字体候选只列清单。
- t15 的 `cmd/`、浏览器存储和对应测试不改；`NewDebugStage` 的 cmd 规格仍测试真实构造，保留入口。
- t17 的 `tools/`／`trace/` 不改。阶段 C 通过绑定配置的方法值保持 `stageN.NewScript()` 与具体类型兼容，不要求工具／sim／旧测试改调用点。
- t14 合并时须保留 Config.Backgrounds 的全部旧 PNG 路径：包括 stage3 第 3／4 张、stage4 第 3 张、stage6 第 3–5 张、Extra 第 3 张；这些初始化调用没有删。`render.LoadImage`、字体／ANM 解码实现没有修改。
- `entity/enemy/enemy.go`／`collision/collision.go` 的旧助手删除涉及第一至三轮共享实体；原 ECL hurt、flags、数据布局、逐敌更新和 `TakeDamage`／`Shootable` 规格接口保留，合并按函数审查，不整文件覆盖。
- `Stage.OnContinue`、`Ending.After`、`ReplayScript`、`stage.NewPlayback`、原回放 codec／RNG／draw priority 不改变。共用演出正式签名列在本节。

共用签名及调用者：

```go
type stagescript.Config struct {
    ECL, MSG string
    BGM, BossBGM int
    FrameCap int
    BackgroundStage int
    BackgroundScroll float64
    Backgrounds [5]string
}
func (stagescript.Config) NewScript() *stagescript.Script
func (stagescript.Config) NewScriptInterface() stage.Script
type ui.TitleAnimations struct {
    World *anm.World
    Env *anm.Env
    File *anm.File
    Roots map[int]*anm.VM
}
func (ui.TitleAnimations) Spawn(int) *anm.VM
func (ui.TitleAnimations) Root(int) *anm.VM
func (ui.TitleAnimations) Leave(int)
func (ui.TitleAnimations) Interrupt(int, int16)
func (ui.TitleAnimations) InterruptVM(*anm.VM, int16)
func (ui.TitleAnimations) Child(int, int) *anm.VM
func (ui.TitleAnimations) InterruptChild(int, int, int16)
func (ui.TitleAnimations) Run(*ui.LoadingEmitter)
type ui.LoadingEmitter struct { Parent *anm.VM; File *anm.File; Burst bool }
func (*ui.Fade) DrawRect(*supervisor.Supervisor)
func ui.GlyphTextFile(*supervisor.Supervisor, *anm.File, string, float32, float32, uint32)
```

TitleController 和 nativeMenu 分别供应原 World／Env／File；不重置 GameSpeed、不把 raw title 文件换成汉化文件。t10 Run(nil) 保留 no-op，t11 传 LoadingEmitter 保留补发；两个场景的生命周期不统一成新持久全局对象。`DrawRect` 不选平面、不改 Fog、不判生命周期，Ending／Replay 保持原提交顺序；t10 原 Fade.Draw 仍检查 Active、Use Screen、关闭 Fog。`screenFade.alpha` 仅让原 `TestEndingFadeTiming` 观察同一 Fade.Alpha，不是计算副本。

# t10：关外场景 A

## 1．摘要

工作树：`D:/GoProjects/th10/.tmp/codex/t10/tree`。阶段 A → B → C → D 及收尾轮均完成，rules 第 6 节规定验收全部通过。没有运行原作、probe 或 Ghidra；没有修改主树源码、关内 ECL／子弹／自机／GUI 逻辑、旧测试断言或 golden，也没有新增依赖。

- A：先单独完成 `docs/research/scenes_a.md`，逐地址普查场景管理、加载、标题页、选择、Option、暂停、输入与 fade。
- B：原场景号／请求／前场景号、延迟工厂与生命周期；300 次 draw 的加载门控；16 位菜单输入、26／8 重复；共享 Selector、ANM 句柄与 fade。
- C：原 ANM 标题／八项主菜单，难度 → 角色 → 装备 → 练习选关，解锁读原 CR／ST；900 帧 demo，顺序 demo1／2／3／0，同一个 t5 回放入口；普通／Extra／练习初始字段使用 t8 GameState。
- D：六项 Option、On／Off／Auto Hint、音量、五键手柄配置；暂停状态 1–5，正常三项／回放两项、确认框、Q／R、恢复、回标题、请求 10 的初始关卡重开；保留 t5 状态 6–13 的回放保存入口。

结局、Staff、Music Room、Player Data、回放选单属于 t11，没有移植其状态机或视觉；现有外部场景仍可使用，原生页 hook 可替换兼容入口。没有把完整 smoke 当作原作逐像素或听音验收。

## 2．改动文件清单

### 新增（32 个）

- `assets/anm/sig.anm`：原始启动签名与贴图。
- `assets/anm/title.anm`：原始标题／选择／Option 脚本和 THTX。
- `assets/anm/title_v.anm`：原始版本标记脚本与贴图。
- `assets/misc/demo0.rpy`：原始官方演示回放。
- `assets/misc/demo1.rpy`：原始官方演示回放。
- `assets/misc/demo2.rpy`：原始官方演示回放。
- `assets/misc/demo3.rpy`：原始官方演示回放。
- `audio/native_menu.go`：原 BGM 平方／SE 立方百分比曲线、Pause／UnPause。
- `audio/native_menu_test.go`：原 dB 曲线与暂停状态测试。
- `docs/research/scenes_a.md`：阶段 A 独立地址普查。
- `docs/handoff/codex_t10.md`：本交付文档。
- `game/retry.go`：请求 10，返回初始关卡，不计请求 13 的续关。
- `game/retry_test.go`：正常／练习重开的资源与继续计数测试。
- `game/scenes.go`：编号、请求、规范化、延迟工厂、可选生命周期。
- `game/scenes_test.go`：场景请求、构造／析构、系统退出与相同请求测试。
- `input/menu.go`：原菜单输入字、按键重复、手柄与数字样本。
- `input/menu_test.go`：16 位重复／边沿、冲突方向、回放隔离、数字小键盘测试。
- `scene/loading/loading_test.go`：300 次 draw 而不是 Update、系统跳过与无头 ANM 测试。
- `scene/stage/pause.go`：原暂停状态 1–5 和共用菜单帧入口。
- `scene/stage/pause_playback_test.go`：暂停不消费回放记录、恢复同帧推进测试。
- `scene/stage/pause_test.go`：暂停门槛／确认／快捷键／计时／冻结／绘制测试。
- `scene/stage/scene_lifecycle_test.go`：释放旧世界与 Ending.After 冻结画面保留测试。
- `scene/stage/scene_mode.go`：原场景号 7、stage 冻结位与生命周期。
- `scene/title/title_test.go`：嵌入 demo 的模式／存档隔离与 t11 hook 测试。
- `scene/ui/fade.go`：原 kind 0／2／3／5 的矩形 fade。
- `scene/ui/native_test.go`：Selector、fade、共享句柄／页回调／无头绘制测试。
- `scene/ui/option.go`：原 Option／手柄配置状态机与数字精灵。
- `scene/ui/option_test.go`：Hint 三态、音量／默认／取消、键位暂存与交换测试。
- `scene/ui/selector.go`：原光标环绕、不可选项、16 槽保存／恢复栈。
- `scene/ui/title_controller.go`：一个共享标题对象、ANM 句柄、注册接口与绘制。
- `scene/ui/title_pages.go`：标题／主菜单／难度／角色／装备／练习的原状态机。
- `scene/ui/title_pages_test.go`：原帧数、菜单输入、模式字段、解锁与各页无头绘制测试。

### 修改（18 个）

- `assets/assets.go`：加入原 demo 的 embed pattern；ANM 已由现有 pattern 嵌入。
- `audio/audio.go`：接原百分比音量／暂停与已存在 BGM fade 的 native 音量路径。
- `audio/se.go`：原 SE 表 dB 与百分比曲线同帧合成，保留原 pan／队列。
- `game/config.go`：BGM 默认 100 与原 cfg 对齐。
- `game/game.go`：接场景切换器、共享当前 GameState 引用、输入键表和原清屏色。
- `game/native_config.go`：原九 word 手柄键表访问，提交只改五个活动字段。
- `game/persistence.go`：读 cfg 时保留三态 Hint，并同步旧 bool 兼容字段。
- `input/input.go`：采样完整菜单键、手柄、练习数字键；回放位编号不变。
- `scene/loading/loading.go`：替换静态 PNG／180 Update 近似为原 sig／ascii ANM 与门控。
- `scene/menus/option.go`：保留公开入口，删除 ebitenui slider／checkbox 近似。
- `scene/select/select.go`：改为共享选择状态机适配器，保留关卡工厂／调试／t5 回调。
- `scene/stage/chain_legacy.go`：删除链外手画暂停黑幕，不动 STD／关内节点。
- `scene/stage/controller.go`：只递送原请求号 11／14／2；t8 回调时机不变。
- `scene/stage/menu.go`：分派状态 1–5，尾部重新读取解除暂停后的速度；6–13 原断言不改。
- `scene/stage/playback.go`：新增 demo 入口／3000 帧返回，物理暂停与记录输入分离。
- `scene/stage/sound.go`：只加原 demo 不替换标题曲的 BGM 门控。
- `scene/stage/stage.go`：暂停接线与请求 10 hook，删除旧 Esc bool toggle。
- `scene/title/title.go`：原标题对象适配器、demo 构造与 t11 页 hook。

### 删除

没有删除文件。删除的是旧加载／标题／选择／Option UI 的近似代码与链外暂停函数。文件级清单实际来自 `changed_files.json` 与只读 `.tmp/codex/base` 的比较。

## 3．移植的函数地址

| 原作 | Go 对应 |
|---|---|
| `FUN_00420470／0041ff80／004218d0` | `game.initScenes／CalcScene／normalizedScene` 与生命周期／延迟工厂 |
| `FUN_0040ac90／0041ff30` | `Game.RequestScene／GoToFactory`，区分系统覆盖与直接请求 |
| `FUN_0041f850／0041fd00／0041f990／0041fac0` | `loading.New／SceneEnter`，同步嵌入资源代替 OS 线程 |
| `FUN_0041fdd0／0041feb0／0042c9f0／0041fb50` | `Loading.DrawPorted／Update／TitleGate.Update／SceneLeave` |
| `FUN_0042c920／0042caa0／0042cb60` | `ui.NewTitleController／title.SceneEnter／Dispose／SceneLeave` |
| `FUN_0042c5c0／0042c620` | `TitleController.SetState／SetSubstate` |
| `FUN_0042c670／0042c6d0／0042c730／0042c750／0042c770` | `Spawn／LeaveScript／Interrupt`，含根子链 |
| `FUN_004497d0／00449670` | `Child／SetChildSprite`；后者不是重启脚本 |
| `FUN_0044bea0／0044be20／0044be70／0040ad20` | `Selector.Move／Push／Pop／Set` |
| `FUN_0042cdf0` | `TitleController.calcPage／initialize`、`title.startDemo` |
| `FUN_0042d300／0042d420` | `calcIntro／calcMain` |
| `FUN_00430320／004306a0／00430a60／00430ff0` | `calcDifficulty／calcCharacter／calcShot／calcPractice` |
| `FUN_0042c840／0042c850` | `ui.ExtraAvailable`，原 ST 单机体解锁字节 |
| `FUN_00431410／00401760` | `drawPractice／GlyphText`，小字库与八位 stored score＋0 |
| `FUN_0040c540` | `TitleController.ShowLoading`（ascii 6），不是捕获 |
| `FUN_0043c8b0／0043c970` kind 0／2／3／5 | `ui.NewFade／TitleController.AddFade` |
| `FUN_0043bd40／0043c230／0043c1a0／0043c2c0` | `Fade.Calc／Draw`，先 alpha 后 Tick |
| `FUN_0042d920／0042e5a0` | `calcOption／refreshHint／refreshVolumes` |
| `FUN_0042f540／0042f8b0／00430250` | `calcKeys／refreshKeys／BindButton`，原重复键交换 |
| `FUN_0044a950／0040ace0` | `MenuInput.Update／Move`，26／8 与 pressed/released |
| `FUN_0044a5f0／0044a1a0` 的菜单部分 | `State.Update／sampleGamepad／menuKeys`，跨平台键／轴适配 |
| `FUN_004223f0／00422ab0／00422c80／00422c30` | `MenuController.Calc／TryPause／OpenPause／calcPause／unpause` |
| `FUN_004218d0` 请求 10，`0x421a03–0x421a34` | `GameState.RetryInitialStage／Stage.OnRetry` |
| `FUN_0044d4e0／0x43e2da–0x43e31c` | `audio.NativeVolumeDB`，BGM 平方／SE 立方、int32 截断与表 dB |
| `FUN_0043e460` Pause／UnPause | `audio.Manager.PauseBGM／ResumeBGM` |

已有 ANM VM／World、Renderer.Rect、回放 codec／Stream、t5 RestartStage 与 t8 模式／进度 API 直接复用，没有另造后端或第二套关内逻辑。

## 4．测试

新增 12 个测试文件、35 个顶层测试。覆盖原 10／30／20／14／40 帧门槛、26／8 输入重复、双向键顺序、确认边沿、原本不能暂停的前 30 帧／demo、确认默认 No、20／30／20／12 暂停时序、恢复时重新读取速度、原 ST／CR 解锁、练习数字键残机、普通／Extra／练习初始 flags／mode／rank／power／life、900 帧与 1／2／3／0 demo、每个本范围页无头绘制、回放不消费记录／不改存档、生命周期与 t11 注册。

五轮完整真实验收：

| 轮次 | build／vet／test／WASM／full smoke | gofmt | full smoke |
|---|---|---|---:|
| A | 全部 exit 0 | 空 | 136.820 秒 |
| B | 全部 exit 0 | 空 | 120.580 秒 |
| C | 全部 exit 0 | 空 | 121.544 秒 |
| D | 全部 exit 0 | 空 | 117.269 秒 |
| delivered | 全部 exit 0 | 空 | 115.388 秒 |

WASM 用 host 下 `go list ./...` 排除 tools 后构建；没有运行浏览器或原作。记录在 `.tmp/codex/t10/checks_{A,B,C,D,delivered}.log` 与同名 `results_*.json`。

新测试首次暴露并修好的问题：默认 fixture 的 Native 为空、demo 未加入 embed pattern、Go defer 提前求值导致解除暂停的尾部错误使用旧速度。只修新 fixture／真正实现，旧断言、容差、golden 都没有修改。

额外 `golangci-lint run`：A 为 33 条；D／delivered 为 32 条（errcheck 26／staticcheck 2／unused 4），不是 exit 0。另在只读 baseline 与本树关闭报告上限实际重跑：51 条 → 50 条；标准化路径与行号后唯一差异是删除旧 Option 的 unused `focusIdx`，没有新增告警。依据 `lint_full_base.log／lint_full_tree.log／lint_comparison.txt`。按 rules 第 6．5 节不把这些继承告警计作验收门槛，不虚报 lint 全绿。

## 5．Trace 结果

A／B／C／D／delivered 均逐章顺序生成、二进制流式 cmp、原作 motion 对比。全部五轮六章与 `d1c2/base_cN.txt` 逐字节相同，没有第一分歧帧。

| 章 | cmp | 字节数 | 原作 motion 一致帧 |
|---|---|---:|---:|
| c0 | 相同 | 94186573 | 15853 |
| c1 | 相同 | 144171327 | 17733 |
| c2 | 相同 | 271969413 | 29763 |
| c3 | 相同 | 319443819 | 29469 |
| c4 | 相同 | 430006225 | 39343 |
| c5 | 相同 | 510140938 | 40099 |

c0 用 `orig_d.txt`；其余用 `orig_05_s2.txt`…`orig_05_s6.txt`；`th10diff -scope motion` 均 exit 0，无覆盖下降。所有生成的 cN 大文件已在比较后安全删除，只保留 `trace_*.log／results_*.json`。未改基线或原作 trace。

## 6．待决与推测

没有等待 dump 才能完成的本范围代码，没有把未确定分支当真值实现。`.tmp/codex/t10/dump_requests.md` 提供可选独立 Ghidra 复核命令；按照 rules 没有尝试 Ghidra。实际依据是 native 全程序汇编、已供应 C／ASM、原 ANM／PE 表。

明确的宿主适配／边界：

- 原加载线程／critical section 改为同步 embedded setup 与 scene-4 TitleGate，保留资源就绪、300 次 draw、最先保留 loading draw／禁用 title calc 的顺序，不伪造 Windows 线程或固定 180 帧动画。
- GoTo 与 t5／t8 回调允许已经构造的对象；回调仍在旧规格要求的请求帧执行，真正切换／SceneEnter／SceneLeave 在下一 calc。新选择／demo／暂停 retry 用真正延迟工厂。没有为了移植监督者破坏 Ending.After 或已有测试。
- 原 Ending.After 会用完成 Stage 的冻结画面打开 t5 菜单，因此 scene 14 保留该 Stage；其余离开清除旧世界。immutable embed 文件／sheet 缓存由 Go 宿主持有，不模拟原堆地址或 COM 生命周期。
- 练习原 draw 六行灰色占位；任务明确要求只显示解锁关卡，所以显示／导航过滤未解锁行，选择值仍是原关卡索引。没有把 Available 与 Cleared 混同，没有默认解锁所有面。
- Hint 三态 cfg 已接线；原关内 Hint 管理器的演出不在范围内，不能据 Option 可编辑就升级那个管理器。
- 外部结局／Staff／Music Room／Player Data／Replay 仍是 t11 旧场景的兼容入口，页 hook 已供应。Extra 可选性只读原 ST＋0x1c…＋0x21，不使用 aggregate bool；该字节由原 ending `0x40b6d5` 等写入，t11 必须补其写者。普通 CR 的次数不能冒充 ST 字节。
- 矩形 fade 使用已移植 Renderer.Rect，菜单原贴图／脚本原样嵌入；截图使用 t5 RGBA→BGRA 适配。字库／抗锯齿／采样／设备、BGM loop／Pause 与手柄实际轴方向仍需要用户目视／试听，不称 L3 完成。
- 音量保留原数学路径：BGM 用平方，SE 用立方与每 id 表 dB；`FUN_0042e5a0` 的额外 `DAT_0049785c` 写值全程序没有读者，没有猜成另一个 master volume。
- malformed host 数据导致全部项禁用时 Selector 避免无限循环，不虚构可选项；正常原作路径不会把所有项都禁用。

## 7．建议的 `docs/fidelity.md` 改动

没有直接修改总账。建议替换／补充以下完整行（不升级 t11 或关内 GUI）：

```text
| 菜单/结局/Staff | 场景 A 与暂停已移植，视觉待用户；结局/Staff/Music Room/Player Data/回放选单待 t11 | t10：FUN_004218d0/0042cdf0/0042d300/0042d420/00430320/004306a0/00430a60/00430ff0/0042d920/0042f540/00422ab0/00422c80；title/title_v/sig 原 ANM，共享标题对象与 Selector，900 帧 demo1/2/3/0，暂停 1–5 接 t5 6–13；保留 Ending.After/OnContinue | 35 项新增测试；五轮 build/vet/test/WASM/full smoke 全绿，gofmt 空；六章 trace 字节相同；用户目视 |
| 场景管理/菜单输入 | 已移植本范围原状态与时序；OS/Scene 生命周期适配明确 | FUN_00420470/0041ff80/004218d0/0040ac90/0044a950/0040ace0；原场景号与请求规范化、延迟工厂、300 次 draw loading gate、每位 26/8 重复、Enter/Z 与 Esc/X/Q/R 独立 | 生命周期/输入/门槛/各页无头 draw；六章基线 cmp 相同 |
| 存档/配置 | 保留 t8 原 score 格式与 t5 cfg；Option 字段/键表已接线 | FUN_0042d920/0042e5a0/0042f540；cfg+0x22 Hint On/Off/Auto、+0x20/+0x21 音量、+4/+6/+8/+0xa/+0x14 手柄五键；Default 不清保留字；Extra读ST、练习读CR Cleared | cfg opaque/三态/五字提交/冲突交换/百分比原曲线测试；旧存储测试保留 |
```

建议按现有格式追加：

```text
- 2026-11-08 — t10 A–D：先成文关外 A 地址普查；场景监督者编号/请求/规范化/延迟工厂/生命周期、原 sig/ascii 加载与300次draw门控、FUN_0044a950的26/8重复移植。标题/主菜单/难度→角色→装备→练习由同一title/title_v ANM对象、原Selector与中断驱动；900帧demo1/2/3/0用t5回放入口，原demo位与2940/3000返回、CR/ST解锁、t8初始模式与资源接线。Option六项、三态Hint、原百分比BGM平方/SE立方曲线、五键暂存/重复交换/提交；暂停1–5、确认默认No、Q/R、请求10与请求13区分，回放两项且不消费记录，保留t5后半/Ending.After。35项新增测试；五轮build/vet/test/WASM/full smoke全部通过，gofmt空，六章trace逐字节不变与motion覆盖15853/17733/29763/29469/39343/40099不降。额外lint默认32条/完整50条为原遗留51条减旧Option unused字段，无新增；L3/听音与t11外部页/ending解锁写者待其范围确认，golden未改。
```

## 8．需要用户目视确认的项目

以下只是用户确认方式，本任务没有启动交互游戏或原作。

PowerShell 在合并树设独立可写数据目录，例如 `$env:TH10_DATA_DIR='D:\GoProjects\th10\.tmp\codex\t10\manual-data'; go run ./cmd`。不要把原作资源 junction 当可写用户目录。原 scoreth10.dat 样本可由用户备份后放入该目录的 score.dat（格式相同）。

1．启动：看原 sig 两块拼接、10 帧颜色进入、少女祈祷中 ANM，300 次 draw 后标题；与原作对照至少显示时间、没有旧静态 PNG 枫叶块与3秒近似。
2．标题／主菜单：初次先 title intro，再按 Z／Enter／X／Esc 等进入菜单；看原 logo／版本／阴影／八项与光标中断；按住上下检查首移与第26、34…帧重复；X／Esc先去Quit再退出，淡入淡出无重复静态文字。
3．待机演示：标题或主菜单不按确认类键900帧进入demo，方向/Shift不重置；反复返回观察1/2/3/0顺序、约2940帧黑色fade与3000帧回标题、按住Z/X/Esc/Ctrl等即时返回、无录制/存档增加。
4．普通选择：Game Start → 难度 → 左右角色 → 上下装备 → 关卡；对照每次6/14/10/40帧阶段、两个角色/六装备图、选择中断、确认后加载标记与32帧黑色fade。
5．Extra／练习：需要真实已解锁ST/CR存档；Extra只选已解锁机体、初始Power4.00；Practice按原难度/机体过滤Cleared，保持原关卡索引、原成绩八位＋0；确认时按1–9看残机覆盖；后续关卡Power4.00、rank0、不误加continued位。ST解锁写者等待t11的ending，不能要求当前旧ending自行解锁。
6．Option：主菜单Option，依次Hint On/Off/Auto、BGM/SE每次5、Key Config、Default、Quit；看百分比数字/前导零隐藏/原阴影；SE项每60帧试音，Default只恢复100/80/On，不清原flags；重启cfg保留。
7．手柄：Key Config七项，按按钮0–30、重复绑定应交换，Default恢复当前live键表；Quit确认提交，Quit上X/Esc退出不提交；上下轴阈值与WASM浏览器按钮编号需实际设备确认，不拿CPU测试当手柄验收。
8．暂停：任一关运行至少30帧后Esc，继续/回标题/重开；对照场地捕获无偏移、front 0x79子动画、确认20帧打开/30帧能操作、No默认、Yes/No退出20帧、最终12帧；Esc恢复，Q回标题、R回初始关卡、不计续关。回放Esc只两项、暂停不推进输入、恢复同帧继续；Game Over/Clear的t5回放保存入口仍可用。
9．音频与浏览器：对照原BGM平方/SE立方音量、Pause不重播/回卷、恢复无突跳、6秒淡出、浏览器声音与画布。旧t5捕获/GDI字形和驱动过滤仍需目视，不因full smoke通过关闭。

## 9．与其他任务的接口

### t11：直接可用的签名（无需读取本工作树）

```go
type game.SceneID int32
type game.SceneFactory func(*game.Game) (game.Scene, error)
func (*game.Game) RegisterScene(id game.SceneID, factory game.SceneFactory)
func (*game.Game) RequestScene(id game.SceneID)
func (*game.Game) GoToFactory(id game.SceneID, factory game.SceneFactory)
func (*game.Game) CalcScene() error
func (*game.Game) CurrentSceneID() game.SceneID
func (*game.Game) RequestedSceneID() game.SceneID
func (*game.Game) PreviousSceneID() game.SceneID
func (*game.Game) CurrentScene() game.Scene
func game.NumberScene(id game.SceneID, s game.Scene) game.Scene
```

原 `GoTo／SetScene` 不变。场景0→1；10/11/12/13→7；15→4且TitleEntry＝3；14为ending。`NumberedScene` 可选 `SceneMode() game.SceneID`；`SceneLifecycle` 可选 `SceneEnter(*game.Game) error／SceneLeave(*game.Game,game.SceneID)`。t11请对ending/staff标号，或返回 `NumberScene` 包装；包装不影响设置原 `Ending.After`。`Game.RunState *game.GameState` 指向t8同一状态，不是新模式。

```go
func (*input.State) Menu() input.MenuInput
func (*input.State) RestoreMenu(raw uint16)
func (*input.State) MenuButtons() uint32
func (*input.State) PracticeLives() int
func (*input.State) SetMenuAux(buttons uint32, lives int)
func (input.MenuInput) Move(mask uint16) bool
```

MenuInput 的 Current/Previous/Repeat/Pressed/Released 均uint16。确认 `Pressed & MenuConfirm(0x1001)`，取消 `Pressed & MenuCancel(0xa)`，方向 `Move`；Pause8／Q0x200／R0x4000独立。原谁调用：`FUN_0042cdf0`各页、`FUN_00422c80`与`FUN_004236f0`。

```go
func ui.NewTitleController(g *game.Game, entry int) *ui.TitleController
func ui.RegisterTitlePage(page ui.TitlePage, hooks ui.TitlePageHooks)
type ui.TitlePageHooks struct {
    Calc func(*ui.TitleController, *game.Game, *input.State) error
    Draw func(*ui.TitleController, *supervisor.Supervisor)
}
func (*title.Title) SetPageHooks(page ui.TitlePage, hooks ui.TitlePageHooks)
func (*ui.TitleController) Calc(*game.Game, *input.State) error
func (*ui.TitleController) SetState(ui.TitlePage)
func (*ui.TitleController) SetSubstate(int32)
func (*ui.TitleController) Spawn(script int) *anm.VM
func (*ui.TitleController) Root(script int) *anm.VM
func (*ui.TitleController) LeaveScript(script int)
func (*ui.TitleController) Interrupt(script int, label int16)
func (*ui.TitleController) Child(root, script int) *anm.VM
func (*ui.TitleController) InterruptChild(root, script int, label int16)
func (*ui.TitleController) SetChildSprite(root, script, sprite int)
func (*ui.TitleController) DrawPorted(*supervisor.Supervisor)
func (*ui.TitleController) ShowLoading()
func (*ui.TitleController) AddFade(kind, frames int32, rgb uint32)
func ui.GlyphText(*supervisor.Supervisor, string, float32, float32, uint32)
func ui.ExtraAvailable(save *game.SaveData, shot int) bool
```

公开字段为Game/State/Substate/Timer/Selector/World/Env/File/VersionFile/Roots/Hooks，以及Character/Shot/Difficulty/PracticeStage。Selector公开Value/Previous/Count/Wrap/Disabled与Set/Move/Push/Pop。

注册 `TitlePlayerData(11)／TitleReplay(12)／TitleMusicRoom(14)／TitlePlayerDataDetail(15)／TitlePlayerDataCards(16)` 即替换旧外部场景适配器。t11菜单包可以在init里调用ui.RegisterTitlePage，避免import title形成环。原函数分别 `FUN_00431ee0／004315c0／00433ef0／00432cb0／00433570`，由 `FUN_0042cdf0` 分派，Draw hooks对应原标题产生/ASCII提交内容。页内转移只SetState／Selector.Push/Pop，不能GoTo销毁共用World。SetChildSprite对应`FUN_00449670`，不要使用RestartChild代替精灵切换。

```go
func ui.NewFade(kind, frames int32, rgb uint32) *ui.Fade
func (*ui.Fade) Calc(speed float32, exiting bool, stageFlags uint32)
func (*ui.Fade) Draw(*supervisor.Supervisor)
func (*audio.Manager) SetNativeVolumes(bgm, se uint8)
func audio.NativeVolumeDB(percent uint8, base int32, se bool) int32
func (*audio.Manager) PauseBGM()
func (*audio.Manager) ResumeBGM()
```

Fade支持kind0/2/3/5，0/3为in，2/5为out；rgb不含alpha，Timer先取值后Tick。kind1仍t7震屏，不在这里；其他种类未供接口。对应 `FUN_0043c8b0／0043c970／0043bd40／0043c230`。原t11如需全屏／场地fade可直接用0/5或3/2。

### 保留的 t5／t8 与暂停入口

```go
func selector.Launch(g *game.Game, state *game.GameState, back game.Scene) error
func stage.NewPlayback(*game.Game, *replay.Replay, int, func(int) stage.Script, game.Scene) (*stage.Playback, error)
func stage.NewDemoPlayback(*game.Game, *replay.Replay, int, func(int) stage.Script, game.Scene) (*stage.Playback, error)
func (*stage.Stage) UpdateMenu(g *game.Game, in *input.State) (stage.MenuAction, bool)
func (*stage.MenuController) TryPause(g *game.Game, in *input.State) bool
func (*stage.MenuController) OpenPause(g *game.Game)
func (*game.GameState) RetryInitialStage()
```

`Stage.OnRetry func(*game.Game,*game.GameState) game.Scene`新供请求10；`Stage.OnContinue`原签名不变，仍请求13。`Stage.SceneFlags() uint32`对应stage+0x58，bit0x10是pause freeze；Menu.State仍DAT_00477830+4。保存回放在原状态6–13，暂停没有另造保存页。`ending.Ending.After func(*game.Game) game.Scene`完全保留，普通最终通关后仍同一t5菜单；GUI+0x9eb4/scene请求及Mode/RunFlags/Practice仍由t8接口负责。

### 调用外部范围、t10未实现的原函数

- ending构造／释放 `FUN_0040b940／0040b9d0`：场景14，沿用ending.New/After；t11负责视觉与ST结束／Extra解锁写者 `0x40b6d5` 等。
- Player Data/Replay/Music Room五组函数：保留上述页hook与原页号，未复制它们的状态机。
- 原stage构造／重开 `FUN_004180e0／00417870／00417c80`：新选择写字段，仍调用已有stage.New、t5 RestartStage/t8 BeginRun，未重写关内逻辑或重复统计。
- ANM／图形 `FUN_0043e710／0043ee30／00448d00／00449470／0043bda0`：直接调用现有VM/World/Renderer；捕获`FUN_00424480`继续t5适配。

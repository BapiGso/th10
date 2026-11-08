# t3：GUI、MSG 演出、绘制节点 0x2b

更新：2026-11-07。工作树：`D:/GoProjects/th10/.tmp/codex/t3/tree`。

**状态：四阶段主要代码与相关测试已实现；完整验收未绿，不写 DONE，不宣称任务完成。** 主树只写此交付文档，所有代码仍在 t3 工作树。未运行原作、未运行 th10probe、未新增依赖、未改 golden。

## 1. 摘要

- 阶段 1：GUI 构造、侧栏数组、Power 数字、信仰／保持条调用点、滚动得分、难度／标题演出；删除旧 ebitenui HUD，0x2b 非 legacy。
- 阶段 2：Boss 名字、星数、渐增／即时扣血血条、四个有序 ARGB 色段、倒计时、Boss 箭头；347／360 写原字段；符卡名／数字／收取提示的 GUI 接口已提供。
- 阶段 3：MSG 原 World VM、两半身、表情、亮暗、出入场、名牌、曲名、op 22／23；时序测试期望完全不改；MST 繁体只覆盖演出台词。
- 阶段 4：独立 `DAT_00477840` 弹出字，723 槽／720 生成槽、calc 0x0f／draw 0x23、原数值／ARGB／上浮／寿命／距离 alpha／末尾调色。
- GUI／MSG VM 统一 `sv.ANM`，不用旧 DrawImage 叠层。字体用既有嵌入 DFYuGaSo W3／W7 的 CPU 纹理，字形／度量／光栅化／阴影近似，VM 几何与时序不近似。
- 不实现 t2 的奖励公式、收取结算、cut-in、符卡背景、震屏；不改自机 0x16、暂停 0x2e。ebitenui 仍被其他菜单使用，故保留 go.mod／go.sum。
- 原符卡数字数组属于 `DAT_004776f4`，必须画在 0x25，不能为了旧测试把它挪到 GUI 0x2b。t2 集成时按第 9 节选用 GUI 接口或其自身 VM。

## 2. 改动文件清单

新增：

- `anm/ordinal_test.go`：验证 text.anm 的负标签与原顺序索引 0～72。
- `anm/world_find_test.go`：验证首次 World 排层前的子 VM 查询及失效句柄。
- `assets/anm/ascii.anm`：th10.dat 原 ANM／纹理／脚本原样嵌入。
- `assets/anm/front.anm`：th10.dat 原 ANM／纹理／脚本原样嵌入。
- `assets/anm/front.ant`：th10c.dat 繁体演出贴图原样嵌入。
- `assets/anm/st01logo.anm`：th10.dat 原 ANM／纹理／脚本原样嵌入。
- `assets/anm/st01logo.ant`：th10c.dat 繁体演出贴图原样嵌入。
- `assets/anm/st02logo.anm`：th10.dat 原 ANM／纹理／脚本原样嵌入。
- `assets/anm/st02logo.ant`：th10c.dat 繁体演出贴图原样嵌入。
- `assets/anm/st03logo.anm`：th10.dat 原 ANM／纹理／脚本原样嵌入。
- `assets/anm/st03logo.ant`：th10c.dat 繁体演出贴图原样嵌入。
- `assets/anm/st04logo.anm`：th10.dat 原 ANM／纹理／脚本原样嵌入。
- `assets/anm/st04logo.ant`：th10c.dat 繁体演出贴图原样嵌入。
- `assets/anm/st05logo.anm`：th10.dat 原 ANM／纹理／脚本原样嵌入。
- `assets/anm/st05logo.ant`：th10c.dat 繁体演出贴图原样嵌入。
- `assets/anm/st06logo.anm`：th10.dat 原 ANM／纹理／脚本原样嵌入。
- `assets/anm/st06logo.ant`：th10c.dat 繁体演出贴图原样嵌入。
- `assets/anm/st07logo.anm`：th10.dat 原 ANM／纹理／脚本原样嵌入。
- `assets/anm/st07logo.ant`：th10c.dat 繁体演出贴图原样嵌入。
- `assets/anm/text.anm`：th10.dat 原 ANM／纹理／脚本原样嵌入。
- `assets/ecl/stage01.ect`：th10c.dat 繁体符卡名原样嵌入，只提取文字。
- `assets/ecl/stage02.ect`：th10c.dat 繁体符卡名原样嵌入，只提取文字。
- `assets/ecl/stage03.ect`：th10c.dat 繁体符卡名原样嵌入，只提取文字。
- `assets/ecl/stage04.ect`：th10c.dat 繁体符卡名原样嵌入，只提取文字。
- `assets/ecl/stage05.ect`：th10c.dat 繁体符卡名原样嵌入，只提取文字。
- `assets/ecl/stage06.ect`：th10c.dat 繁体符卡名原样嵌入，只提取文字。
- `assets/ecl/stage07.ect`：th10c.dat 繁体符卡名原样嵌入，只提取文字。
- `assets/misc/musiccmtt.txt`：繁体曲名原样嵌入；关卡内曲名目前由原 logo ANT 绘制。
- `assets/msg/st01_00.mst`：th10c.dat 对应角色、关卡的繁体台词原样嵌入。
- `assets/msg/st01_01.mst`：th10c.dat 对应角色、关卡的繁体台词原样嵌入。
- `assets/msg/st02_00.mst`：th10c.dat 对应角色、关卡的繁体台词原样嵌入。
- `assets/msg/st02_01.mst`：th10c.dat 对应角色、关卡的繁体台词原样嵌入。
- `assets/msg/st03_00.mst`：th10c.dat 对应角色、关卡的繁体台词原样嵌入。
- `assets/msg/st03_01.mst`：th10c.dat 对应角色、关卡的繁体台词原样嵌入。
- `assets/msg/st04_00.mst`：th10c.dat 对应角色、关卡的繁体台词原样嵌入。
- `assets/msg/st04_01.mst`：th10c.dat 对应角色、关卡的繁体台词原样嵌入。
- `assets/msg/st05_00.mst`：th10c.dat 对应角色、关卡的繁体台词原样嵌入。
- `assets/msg/st05_01.mst`：th10c.dat 对应角色、关卡的繁体台词原样嵌入。
- `assets/msg/st06_00.mst`：th10c.dat 对应角色、关卡的繁体台词原样嵌入。
- `assets/msg/st06_01.mst`：th10c.dat 对应角色、关卡的繁体台词原样嵌入。
- `assets/msg/st07_00.mst`：th10c.dat 对应角色、关卡的繁体台词原样嵌入。
- `assets/msg/st07_01.mst`：th10c.dat 对应角色、关卡的繁体台词原样嵌入。
- `docs/handoff/codex_t3.md`：九节交付、阶段结果与未绿项，另复制至主树同名文档。
- `entity/item/popup_test.go`：验证各道具弹出字值、色、位置及不生成分支。
- `gamedata/presentation_test.go`：汉化原样资源逐字节对照 th10c.dat。
- `gamedata/presentation.go`：演出选择繁体 ANT 贴图，逻辑仍用原版资源。
- `render/vmtext_test.go`：验证文字写入不改时序／精灵尺寸，缓存复用且原资源不变。
- `render/vmtext.go`：CPU 光栅化嵌入字体到私有 ANM 纹理，保留原 VM 几何与时间线。
- `scene/dialog/presentation_test.go`：MSG op 影响、原脚本／字体／图层绘制与时序不变护栏。
- `scene/dialog/presentation.go`：MSG 的原 World VM、表情、亮暗、出入场、名牌和曲名。
- `scene/stage/ecl/gui_names.go`：ECT 仅按子程序／宣言序号配对名字，绝不运行汉化逻辑。
- `scene/stage/ecl/gui_test.go`：347／360 字段及取参顺序、繁体名字只用于演出的测试。
- `scene/stage/gui_boss_test.go`：Boss 血条、色段顺序、警告色、星数与符卡数字测试。
- `scene/stage/gui_boss.go`：Boss 名字、星数、原色段血条、倒计时与箭头。
- `scene/stage/gui_msg.go`：为 MSG 提供 GUI 资源、World 和嵌入 ANM。
- `scene/stage/gui_spell.go`：符卡 UI 接口、十三个数字 VM 和收取提示，不计算奖励。
- `scene/stage/gui_test.go`：侧栏假设备序列、数字格式、上限和信仰遮挡测试。
- `scene/stage/gui.go`：GUI 构造、侧栏数组、得分滚动、Power、信仰与标题。
- `scene/stage/score_popup_test.go`：弹出字格式、游标、寿命、距离 alpha、调色及设备序列。
- `scene/stage/score_popup.go`：独立 DAT_00477840 管理器、calc 0x0f 与 draw 0x23。

修改：

- `anm/loader.go`：保留原文件顺序脚本表，追加顺序索引视图，不改原标签表。
- `anm/world.go`：增加按脚本号查询根对象子 VM 的 FindChild。
- `assets/assets.go`：嵌入既有 DFYuGaSo W3／W7 和曲名，GUI 不读取 OS 字体。
- `entity/item/item.go`：增加原数值、ARGB 与道具位置的弹出字回调，不改资源与运动。
- `render/text.go`：提供只读嵌入字体的 LoadGameFace。
- `scene/dialog/dialog.go`：移除旧渐变框／PNG 叠层，只保留旧台词 API 的 Line 类型。
- `scene/dialog/msg.go`：只追加 MST 繁体演出台词，保留原指令，按首个 NUL 终止文字。
- `scene/dialog/msgvm.go`：消费指令后驱动演出，不改时钟／等待／信号／音效逻辑。
- `scene/stage/chain_legacy.go`：只删除旧 nodeGUI，不改自机 0x16 或暂停 0x2e。
- `scene/stage/chain_test.go`：仅在授权 legacy 列表移出 0x2b，其余期望不改。
- `scene/stage/chain.go`：0x2b 非 legacy；0x23 弹出字和 0x25 符卡数字接原节点。
- `scene/stage/ecl/game_ops.go`：347／360 原 GUI 字段语义，以及既有 331 的全局标记清除。
- `scene/stage/ecl/interrupt.go`：现有阶段时限写入 GUI 原倒计时字段。
- `scene/stage/ecl/vm.go`：Boss HUD 补齐原字段与繁体演出名字，不改 ECL 程序。
- `scene/stage/stage.go`：GUI／弹出字／MSG 接线，按原 calc 相对次序推进演出。

删除：

- `scene/dialog/faces.go`：删除旧上半身 PNG、表情后缀猜测与语义回退。
- `scene/stage/hud.go`：删除旧 ebitenui HUD。

## 3. 移植的函数地址

| 原作 | Go |
|---|---|
| FUN_00413810／00413980／00413a20 | `NewGUI`，嵌入资源／初值／数组 |
| FUN_00413bc0 | `GUI.beginStage`，框、标签、标题、难度及六组数组 |
| FUN_00413790 | `GUI.SetLives`，只改可绘位，VM 仍更新 |
| FUN_004054b0 | `GUI.SetPower`；原 0.05 单位换算由 `powerDigits` |
| FUN_00415ae0／00414900 | `GUI.Calc`／`calcFaith`／`calcBoss`／`AfterMSG`／`calcArrow` |
| FUN_00417040 | `GUI.Numbers`，显示得分滚动，不接资源／extend 副作用 |
| FUN_00415af0／00415800 | `Stage.nodeGUI`／`GUI.Draw`／`drawBoss`／矩形调用点 |
| ECL 347，0x40fc85～0x40fc96 | `VM.addLifeMarker`／`GUI.SetHPMarker`，比例与 ARGB |
| ECL 360，0x411b8d～0x411b93 | `GUI.SetBossStars`，保留原整数 |
| FUN_004127a0，0x4128b0 | `GUI.SetBossTimer`，现有阶段时限写原 GUI 字段 |
| FUN_00409270／004091c0 | `Stage.nodeSpellHUD`／`SpellHUD.Draw`，0x25 |
| FUN_00408d60 的显示尾部 | `SpellHUD.update`，不包含公式／符卡时钟 |
| FUN_004172e0 | `GUI.ShowNotice`／`ShowCaptureBonus`，只显示 |
| FUN_00447470，0x4476b8～0x4476eb | `File.ScriptOrder`／`OrdinalScripts`；text 负标签仍保留 |
| FUN_004479d0／00447a50／00447ae0 | `render.WriteVMText`／`RasterText`，字体近似 |
| FUN_00415b00／00415e90 视觉分支 | `NewPresentation`／`Presentation.Apply` |
| FUN_004136c0 | `Presentation.Close`，删六个剩余句柄及子链 |
| FUN_00409e50 | 隐藏时中断 1，不立即删除 |
| FUN_0040c480／0040c4d0 | 中断 2／3 亮暗 |
| FUN_00449470，0x41623c／0x41624e | op 22／23 的中断 7，忽略指令 Arg |
| FUN_004497d0／00449670／004496a0 | `World.FindChild`／`Presentation.setFace` |
| FUN_0042b430／0042b4d0 | `NewScorePopups` |
| FUN_0042b9c0 | `ScorePopups.Spawn`／`Context.AddScorePopup` |
| FUN_0042b9a0／0042b6f0 | `ScorePopups.Calc`，优先级 0x0f |
| FUN_0042b9b0／0042b780 | `nodeScorePopups`／`ScorePopups.Draw`，0x23 |

证据：既有 `.tmp/ecl_dump/render6`、`r6`、`r8`、`r10`、`r11`、`bomb2`、`item`；缺失部分只读 `objdump` 汇编与 `peread` 位模式核对，输出在 `.tmp/codex/t3/evidence/`。没有启动 exe 或数据目录中的程序。

## 4. 测试

每个阶段均执行完整检查，日志 `.tmp/codex/t3/logs/phase1` 至 `phase4`。所有新增／修改的 Go 文件已 gofmt；不批量格式化范围外文件。

| 检查 | 阶段 1 | 阶段 2 | 阶段 3 | 阶段 4 |
|---|---|---|---|---|
| go build ./... | 通过 | 通过 | 通过 | 通过 |
| go vet ./... | 通过 | 通过 | 通过 | 通过 |
| go test -count=1 ./... | 未绿：Rect／smoke | 同前＋旧 0x25 列表 | 同前 | 同前＋旧 0x23 列表 |
| gofmt -l . | 基线 38 文件 | 基线 38 文件 | 37 文件 | 37 文件 |
| golangci-lint run --timeout 10m | 44 项 | 42 项 | 41 项 | 41 项 |
| WASM，非 tools 包 | 通过 | 通过 | 通过 | 通过 |

WASM 按规则命令执行。`go list ./...` 会诊断 tools/th10probe 的 Windows 包不支持 js；筛除 tools 后 `go build` 退出 0。未添加 OS 专用 GUI 代码。

最终再次执行 `go test ./scene/dialog ./entity/item ./scene/stage/ecl ./gamedata ./render ./anm -count=1`：六包全通过。stage 的全部新 GUI／Boss／弹出字测试单独跑通过；旧 MSG／ECL 时序、期望与 golden 未修改。

新增测试：

- GUI：`TestGUIArrayDrawOrder`、`TestGUINumericCaps`、`TestGUIPowerSprites`、`TestGUILivesKeepVMs`、`TestGUIFaithRectCallSites`、`TestGUIScoreRollIsVisualOnly`、`TestGUIFaithDigitsAndHysteresis`。
- Boss／符卡 UI：`TestGUIBossNames`、`TestGUIBossFillAndClear`、`TestGUIBossSegmentsInSlotOrder`、`TestGUIBossStarsAndTimer`、`TestGUIArrowThresholdAndAlpha`、`TestGUISpellDrawOrder`、`TestGUISpellNumbersAndCapturePopup`。
- ECL：`TestECL347GUIRecords`（含逆序 pop 与全局写入）、`TestECL360RawStars`、`TestSpellNamesPresentationOnly`。
- MSG：`TestMSGPortraitScriptsAndFaces`、`TestMSGSpeakerAndTextVMOperations`、`TestMSG22And23InterruptSeven`、`TestMSGHideUsesExitAndEndDeletes`、`TestMSGWorldLayerDraws`、`TestMSGTraditionalOverlayKeepsInstructions`。
- 弹出字：`TestScorePopupDigits`、`TestScorePopupCursorAndLifetime`、`TestScorePopupAlphaAndPalette`、`TestScorePopupDrawSequence`、`TestASCIIPopupGlyphDimensions`、`TestCollectionPopupValues`。
- 资源／字体：`TestTextANMOrdinalTable`、`TestWorldFindChildBeforeLayerSort`、`TestVMTextKeepsTimelineAndSpriteSize`、`TestTextByteWidth`、`TestPresentationAssetsMatchTH10C`、`TestPresentationANMUsesTraditionalTexture`。

失败与原因：

1. `anm/anmdraw.TestRect`：`rect.go:32` 基线桩恒 panic：`anmdraw: Rect not implemented`。归 t1，未改。
2. `scene/stage.TestChainDraws`：`chain_test.go:59` 旧 drawing 列表不含 0x23／0x25；输出为 `draws true, want false`。只改授权 legacy 列表移出 0x2b，未改其他期望。证据分别为 FUN_0042b4d0 注册 draw 0x23，以及 FUN_00409270→004091c0 画十三个符卡数字。
3. `TestReplayDrawSmoke` 默认与 `TH10_SMOKE=full`：都在 `th10_01/ch0` 的 GUI 矩形调用遇到共享 Rect 桩，未能跑完 24 章。完整版已实际执行，日志 `phase4/smoke_full.log`；不称通过或完整视觉已验收。
4. lint 最终 41 项来自基线代码／旧测试（errcheck 29、ineffassign 1、staticcheck 5、unused 6）。本任务新增的 unused／ineffassign 均已消除；不做无关全库修正或测试期望改动。
5. gofmt 列表为未修改的基线文件，主要含 CRLF／旧排版；最终列表见 `phase4/gofmt.log`。

开发中新增测试首次误用内部辅助函数名，已改为现有 `execGameOp` 并通过。出生帧测试曾错误假设无设备调用；取证证实 −Inf＋Inf→NaN 后，新增测试改为锁定原算术。Extra 的汉化文字首个 NUL 后有旧填充，现只截断演出字段，原指令与 View 不变。

## 5. Trace 结果

四个阶段各逐章顺序生成 c0～c5；24 次 `cmp` 全部退出 0，**六章输出始终与改动前基线逐字节相同**。未消耗游戏 RNG；演出对象用原 alternate RNG。大 trace 已逐章比较并删除，只保留日志、traces.csv 和结论。

| 章 | 四阶段与基线 cmp | 指定原作 trace，motion |
|---|---|---|
| 0 | 全部相同 | 原作／基线／t3 在第 4915 帧的既有 MSG 差分，见下段 |
| 1 | 全部相同 | MATCH：17733 帧 |
| 2 | 全部相同 | MATCH：29763 帧 |
| 3 | 全部相同 | MATCH：29469 帧 |
| 4 | 全部相同 | MATCH：39343 帧 |
| 5 | 全部相同 | MATCH：40099 帧 |

一面指定 `orig.txt` 的实际结果与规则写的 15853 一致帧不符：第 4915 帧原作为 `{entry 0 clock 34 wait 0 signal 0 at 34 op 16}`，基线／t3 为 `{entry 0 clock 35 wait 0 signal 0 at 38 op 12}`。已直接执行 `th10diff -a orig.txt -b base_c0.txt -scope motion` 复现相同差分，且 t3 与 base_c0 字节相同，确认没有下降／GUI 回归。保持既有 MSG 时序，不为了消差修改测试或 golden。

## 6. 待决与推测

- 未绿项：共享 Rect、绘制列表授权、基线格式／lint。架构师集成后须重跑全部检查与完整 smoke；目前不写 DONE。
- Ghidra 首次被拒绝写用户配置目录，已释放锁，不重试／改配置绕过沙盒。`.tmp/codex/t3/dump_requests.md` 保留原完整请求；代码已用现有 dump＋只读 objdump 补核，需架构师复核这些新地址／调用关系，尤其原作者定义的结构而非 Go 适配。
- 推测／控制器适配：`SingleStage` 代原 DAT_00474ca0 bit 0x20；`HasSeed` 代回放起始状态 DAT_00477810+0x5c，只影响练习提示／回收线提示。代码已标 inferred，待 t5 接原字段。
- 推测／控制器适配：Boss 名字原读 DAT_00474c84；344 未接，暂用既有 enemy.Type 区分中 Boss／Boss。提供 `SetBossSection`，不自行新增 344 语义。
- `GUI.Numbers` 的视觉 HiScore 与资源写入解耦；原 FUN_00417040 中 extend／记分副作用未加入，避免 GUI 改 trace。资源系统／t5 如修原高分／extend，应明确接线而非 GUI 偷发残机。
- t2 未合并：只读现有 SpellCard，历史 0／0；不在 GUI 计算奖励、发分、判收取。GUI 中的字体、数字、名牌与收取提示接口已实现，调用者见第 9 节。
- 原 GUI 最终通关标志（+0x9eb4 的 0x10／0x20、+0x9ecc）与 120 帧 scene request 尚未接入；保持已有 MSG Clear／Context.cleared 场景流程以维持基线。不能据此称 GUI 所有场景控制分支已完全移植，后续应由架构师划分 GUI 状态与场景控制器接线。
- 字体近似：W3／W7 字形、度量／字宽、抗锯齿、黑色阴影不会逐像素等于 GDI／D3DX；右对齐按原宽／半宽字符计数。VM 位置、精灵尺寸、行距 18、颜色／alpha 与出现消失时间线保留。
- 原 op 17 暗化的是文字 VM +0x4c／+0x50，不是立绘；新演出按汇编，旧 View.Dim 只作旧 API／测试兼容。
- 弹出字出生帧 F＝0，因为 calc 0x0f 先于拾取：8/F＝+Inf，第一字 −Inf 剔除，后续 −Inf＋Inf＝NaN 会送进设备。按原调用链保留，驱动像素差异需目视；不修成提前更新一帧。

## 7. 建议的 docs/fidelity.md 改动

未直接修改 fidelity。以下为完整建议状态行；在未绿项解除前状态保持待集成，不称完成：

```text
| ANM 绘制（anmdraw） | 移植（模式 0～4、6、8、9；模式 5、7 未移植；GUI 集成验收待决） | `Renderer.Draw`＝`FUN_004451c0`；现有模式实现不改。GUI 0x2b 改为非 legacy，内嵌六组数组／Boss 箭头共七类 VM 均经 `sv.ANM`；框、标签、MSG、Boss 名字／星数、标题／曲名在原 World 层 13～16 绘制。独立 ASCII 弹出字节点 0x23、符卡数字节点 0x25 按原顺序绘制。矩形仍调用归 t1 的 `Renderer.Rect`（当前基线未实现），TestReplayDrawSmoke／TH10_SMOKE=full 尚未通过，不能称完整绘制完成 | 现有 anmdraw 测试；`gui_test.go`、`gui_boss_test.go`、`score_popup_test.go`、`presentation_test.go`；全程 smoke 待集成 |
| 对话 .msg | 时序保持已移植；ANM 演出已接，字体近似，完整绘制验收待集成 | `FUN_00415e90` 原时钟、wait、signal、Ctrl、SE／BGM／clear 逻辑不变。1／2 用 pl0N／大立绘原脚本，3 用 front 0x5a，4～6 中断 1 退出，7／8 中断 2／3 亮暗并写文字偏移 8／24，12／13 按子脚本号替换原精灵，14～16 写 text.anm 字体纹理，17 实际暗化两条文字 VM，18 logo 脚本 2 曲名，19 原 Boss 名牌，22／23 中断 7。MST 仅覆盖繁体演出台词，原 MSG 指令／View 文字／测试不变。嵌入 DFYuGaSo 的 CPU 字形、字体度量、抗锯齿、阴影属于 GDI／D3DX 近似；VM 几何、颜色、行距和时序按原脚本。共享 Rect 阻塞全程 smoke | 既有 msgvm／msg／dialog／ECL 测试；`scene/dialog/presentation_test.go`、`render/vmtext_test.go` |
| Boss HUD/符卡名 | GUI 演出移植；奖励／结算待 t2，字体近似，完整验收待集成 | `FUN_00414900`：血条每帧增加 float32 0x3ccccccd、扣血即时下降；347 写四个原比例／ARGB，按槽序画累计色段，不再画白刻度；360 写星数，十个固定 World 句柄；原 Boss 名图、倒计时 front 0x50／0x51 的 7／8／9 警告中断与 ascii 箭头恢复。符卡名用 text.anm 0x48 的原时间线及字体纹理，Bonus／历史十三个 front VM 在 0x25，收取数字与 banner 用 FUN_004172e0。GUI 只读现有 SpellCard；t2 必须接奖励、结算与历史接口，当前历史 0／0。344 原名字选择字段尚未接，用既有敌人类型作过渡。字体不称逐像素相同，Rect／链表绘制规格待集成 | `gui_boss_test.go`、`ecl/gui_test.go`、`render/vmtext_test.go`；用户目视与全程 smoke 待集成 |
| 道具管理器 | 位精确（移植；一至五面运动范围无回归，道具数待 probe 采样） | `DAT_00477818`（`entity/item`）：150 个掉落槽 + 2048 个消弹槽，步长 0x3f0。生成 `FUN_0041bb00`：消弹道具（种类 8）取游标槽（占用则不生成，游标照走），延迟按上次更新以来的生成数取槽号 &3 / &7+4 / &15+8 / &31+16，进入状态 5；其余种类取首个空槽，x 钳到 [−192, 192]，火力 > 99 时 1/4→9、10/11→5；10/11 的外观同 1/4；`bullet.anm` 脚本为种类 + 0x176。更新 `FUN_0041afd0`（按槽序）：状态 5 倒数结束转 2 并启动脚本；状态 1 在自机 y < 128（且非状态 2/4）或灵击中转 3 追踪（速度灵梦 5、魔理沙 7，`0x476fa0`），否则移动、重力 0.03、开始下落后横向速度清零、竖直速度上限 2、y > 472 消失；状态 2 上升，停止上升后转 3，y > 472 消失并 rank −4；状态 3/4 追踪，速度每帧 +0.2 直到 12，决死窗口中转回 1、速度清零。自机不在状态 2 时：在拾取盒内拾取（`FUN_00425730` 尾部，±10/±12），状态 1/2 进入吸附盒（低速 ±50/±59，否则 ±20/±24）转 4，速度为追踪速度 ×0.33333334。收集（`FUN_0041be80(n)` 给信仰加 n/10，`FUN_00409d90(n)` 给得分加 n/10）：1/10 火力 +1，升级时 rank +12、刷新副机 `FUN_00426f70`、音效 0x1d；4/11 火力 +20，升级时 rank +24；2 得点，自机 y < 144 时得 信仰×10、rank +8，否则 信仰×10/2 − ftol((y−144)·(1/304)·(信仰·0.5 − 5000))、个位清零、rank +1；3 按难度 n = 5000/5000/8000/10000；5 得 信仰×10、rank +8；7 残机 +1（`FUN_004188a0`，音效 0x2c，上限 9）、rank +0x100；8 n = 10、得分 +10；9 n = 100；拾取音效 0x14。达到满火力的那次更新末尾 `FUN_0041ba50` 把场上的 1/4 号道具换成以 −π/2、2.2 抛起的 9 号道具。绘制 `FUN_0041b8e0`：场内用精灵 外观 + 0x157（原作与 +0x158 比较，永不相等，所以每帧重设 alpha 0xff）；y < 8 时画在 y 24 的箭头（外观 + 0x160），alpha = (char)ftol((y−8)·0.03125·255)，为负数时取低字节。演出：得分弹出字 `FUN_0042b9c0` 已由独立 `DAT_00477840` 的 calc 0x0f／draw 0x23 移植；Power 数字 `FUN_004054b0` 已由 GUI 更新。GUI／MSG／字体／节点的完整绘制验收仍待共享 Rect 与链表测试接线，不称完成。未移植：特效 0x189 | `entity/item/item_test.go`、`scene/stage/power_test.go`、`scene/stage/score_popup_test.go`、`entity/item/popup_test.go` |
```

按现有格式的更新记录建议：

- 2026-11-07 — T3：GUI／MSG 演出与原 draw 0x2b。GUI 框／标签／标题／难度、侧栏数字、残机、Power、信仰与保持条、Boss 名／星数／有序 ARGB 血条／倒计时／箭头由 front／ascii 与 anmdraw 绘制，删除旧 ebitenui HUD／PNG 叠层；独立 DAT_00477840 弹出字恢复 calc 0x0f／draw 0x23，符卡十三个数字恢复 draw 0x25。MSG 原两半身、表情、亮暗、退出、名牌、曲名和 op 22／23 通过原 World VM；字体纹理用既有嵌入 DFYuGaSo，GDI／D3DX 光栅化为明确近似。347／360 写原 GUI 字段，MST／ECT 只提供繁体演出文字。符卡奖励与结算仍交 t2。四阶段六章 trace 均与基线逐字节相同；指定 orig.txt 的一面 4915 MSG 差分在基线即存在。完整验收未绿：共享 Rect 桩、旧 TestChainDraws 0x23／0x25 列表、基线格式／lint；TH10_SMOKE=full 已尝试但被 Rect 阻塞。不称完整绘制完成，待集成后复验。

## 8. 需要用户目视确认的项目

**先合并 t1 的 Rect，再检查画面；当前工作树会在首次 GUI 矩形遇桩 panic。** 可运行项目现有命令 `go run ./cmd -debug-stage 1 -debug-diff normal`（可换 1～6／extra），或集成 t5 后从用户回放菜单看 th10_05。不要启动原作作自动取证。普通输入会令对话窗口时间后移；以下采样是 th10_05 各章从零计时，日志 `.tmp/codex/t3/visibility_times.log`。

| 项目 | 可见位置／约时间 | 确认内容 |
|---|---|---|
| 侧栏 | 每面 0～1 秒 | 边框、标签、难度、零补齐得分／高分、残机、Power 淡入与对齐 |
| 火力槽 | 五面约 180 秒首次拾取弹出字附近；手动一面前几波收 P | 原作这里是三位 Power 数字与点，不是另造长槽；0.05 单位、升级、灵击减 Power |
| 信仰／保持条 | 拾取道具时；五面约 180 秒附近，左下角 | 六位显示／短保持条，移到左下角按中断变暗，离开有迟滞 |
| Boss HUD | 一面约 42 秒中 Boss；五面约 103 秒符卡 | 名字图、星数、渐增白条、立即掉血、原色段顺序而非白刻度、倒数警告色／SE |
| 符卡名／奖励 | 一面约 129 秒、五面约 103 秒、六面约 137 秒首次宣言 | 名字原缩放／移动／淡化，0x25 奖励数；公式／历史 0／0 待 t2 接线 |
| 对话框 | 一面约 81 秒；六面约 61 秒 | 两行基线 y＝375／393、双方偏移、两半身、说话者亮／听者 96 色、原出入场／NUL 无尾字 |
| Boss 名牌 | 一面约 83 秒；五面约 183 秒；六面约 66 秒 | enemy ANM 名牌脚本与动画；六面／Extra 大立绘裁切／下半身 |
| 曲名 | 每面 0～5 秒关卡曲；Boss 对话结束切曲时 | logo ANT 原脚本 1／2 的曲名位置与淡入淡出，Boss 时间受对话输入影响 |
| 得分弹出字 | 六面约 28.2 秒首次；五面约 180.4 秒首次 | 上浮、字距收敛、距离 alpha、52／56 帧调色、升级特殊字形、60 帧结束，无出生帧闪烁／NaN 驱动异常 |

回放可能按住 Ctrl 快过台词；看正常演出可用 debug 入口，不按 Ctrl。没有像素 diff 或用户目视验收，不报告 L3 已完成。

## 9. 与其他任务的接口

调用范围外部分：

- `anmdraw.Renderer.Rect(v [4]float32, c gfx.Color)`＝FUN_0043bda0，归 t1。原 GUI 调用点 `0x4158fc`／`0x415937`（保持条）、`0x41598a`／`0x4159b8`（血条）、`0x415a29`（色段）。真实调用保留；未复制另一后端或吞 panic。
- FUN_00409280／00409c00／00408d60 的宣言、结算、奖励／时钟属于 t2。当前 UI 只用既有 SpellCard／BossHUDInfo，未实现 cut-in／背景／震屏。
- FUN_00417040 的资源／extend、GUI 最终通关 scene request 和 344 名字选择字段仍需控制器明确接线。这里不改 Context.cleared 的既有逻辑。
- 不碰自机 0x16、DAT_00477830、draw 0x2e。0x23／0x25 的测试 drawing 列表建议由架构师增加；t1／t5 各自负责 legacy 0x16／0x2e。

提供的接口：

| 签名 | 原作调用者／用法 |
|---|---|
| `GUI.SetPower(whole, percent int)` | FUN_004054b0，原自机死亡／灵击／拾取后调用；当前 GUI calc 读现有 State.Power |
| `GUI.SetLives(n int)` | FUN_00413790，FUN_004188a0 增残机／上限钳制后调用 |
| `GUI.SetBossStars(n int32)` | 360，保留原字段值，已接 |
| `GUI.SetHPMarker(index int, fraction float32, colour uint32)` | 347，已接；331 清四个全局记录 |
| `GUI.SetBossTimer(seconds int)` | FUN_004127a0，已接现有阶段时限 |
| `GUI.SetBossSection(n int32)` | 给名字选择提供 DAT_00474c84；344 未由 t3 实现 |
| `GUI.BeginSpellPresentation(name string)` | t2 的 FUN_00409280 名字／数字起始 GUI hook；自动禁用字段过渡驱动 |
| `GUI.UpdateSpellPresentation(bonus int64, capture bool, captures, attempts int32, frames int32)` | t2 的 FUN_00408d60 给已经算好的数据／历史／时钟 |
| `GUI.EndSpellPresentation()` | t2 FUN_00409c00 的 GUI 释放部分 |
| `GUI.ShowCaptureBonus(amount int64)` | FUN_00409c00 在 0x409cbb 调 FUN_004172e0(kind 0)；只显示，不发分 |
| `GUI.ShowNotice(kind int, amount int64)` | FUN_004172e0：0 收取、1 失败、2 满 Power、3 HiScore、4 Extend、6 Stage Clear；5 空 |
| `GUI.SetSpellName(vm *anm.VM, name string)` | t2 若保留自己的名字句柄，在 0x409681 原 FUN_00447ae0 调用处写字体纹理 |
| `GUI.UseExternalSpellPresentation()` | 上述自持 VM 模式先关闭过渡对象；此模式 t2 也需自持／接其 0x25 数字数组，不能再同时生成第二份名字／数字 |
| `Context.AddScorePopup(pos [3]float32, value int32, color uint32)`／`ScorePopups.Spawn(...)` | FUN_0042b9c0；已接道具原调用点。负值为升级字形，不是负分，不调用 AddPoints |
| `World.FindChild(root *anm.VM, script int) *anm.VM` | FUN_004497d0，首次 World.Run 前可查询，失效句柄 nil |
| `dialog.NewPresentation(c PresentationConfig) *Presentation`／`MSG.SetPresentation(p)` | FUN_00415b00 视觉构造与挂接，不推进时钟，提供嵌入资源／World 即可 |
| `File.OrdinalScripts() *File` | text.anm 的原负标签 −73～−1 对应 runtime 0～72；t2 用原始 gamedata.ANM 文件时先调用此函数 |

集成顺序：合并共享 Rect，接 t2 UI hook／0x25 所有权与历史，接 t5 的实践／回放标志和最终场景请求，授权更新 drawing 列表，然后重跑完整 suite、lint、WASM、六章 cmp、TH10_SMOKE=full 与目视。当前无 DONE。

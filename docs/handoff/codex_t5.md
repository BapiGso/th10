# t5：回放、关卡内菜单与存储

## 1. 摘要

工作树：`D:/GoProjects/th10/.tmp/codex/t5/tree`。已完整阅读 rules 与指定资料，按阶段顺序实现并执行六轮全仓检查／六章 trace。实现已交付审查，但**全仓验收未全绿**，不得视为发布验收完成。未运行原作，未修改主树源码、基线、golden 或非授权测试。

| 阶段 | 状态 |
|---|---|
| 1．编码器 | 核心完成，13 个文件逐字节相同；全仓验收有基线阻断 |
| 2．录制 | 核心完成，th10_05 六章输入、样本、章节头强测试通过；自机不透明字段留 t1 hook |
| 3．关卡内菜单与节点 0x2e | 状态 6–13、ANM 绘制、排行、名字与保存流程完成；mode 13 重开已接 |
| 4．回放存储 | 桌面原文件命名／原 .rpy／安全覆盖、WASM 会话内存后端完成；阶段全仓仍有既有阻断 |
| 5．游戏内播放 | 存储入口与共享输入／章节恢复完成；阶段全仓仍有既有阻断 |
| 6．原作存档／配置格式 | cfg 的 52 字节格式、默认值、存储与回放配置快照完成；score.dat 容器／符卡历史未做 |

## 2. 改动文件清单

### 新增

- `assets/anm/ascii.anm`：原字库与捕获背景脚本的原始嵌入资源。
- `assets/anm/front.anm`：原 game-over／clear／选择项 ANM 的原始嵌入资源。
- `docs/handoff/codex_t5.md`：本交付文档。
- `entity/player/replay.go`：已有 focus 状态的快照／恢复接口。
- `game/native_config.go`：原 52 字节 cfg、默认值与校验。
- `game/native_config_test.go`：原 cfg 全字节 roundtrip、默认值与坏输入。
- `game/native_config_desktop_test.go`：原 cfg 存储、音量编辑与不透明字节保留。
- `game/persistence_data.go`：合并两平台的进度类型，新增原排行记录布局。
- `game/persistence_data_test.go`：同分插前及排行搬移测试。
- `game/persistence_replays.go`：统一回放存储接口、槽位命名、验证、内存后端。
- `game/persistence_replays_desktop.go`：数据目录内的安全文件读取／覆盖。
- `game/persistence_replays_desktop_test.go`：文件命名、覆盖、目录与失败回收测试。
- `game/persistence_replays_wasm.go`：与现有 WASM 模式一致的页面会话内存后端。
- `game/persistence_replays_test.go`：内存别名、损坏槽位、原格式内容及越界测试。
- `game/replay_progress_test.go`：回放不得更新进度的护栏。
- `game/restart.go`：mode 13 的原作关卡重开初始化分支。
- `game/restart_test.go`：重开／续关计数与初始关卡分支。
- `input/replay.go`：共用回放输入映射、原始三字、cfg 自动低速选项。
- `input/replay_test.go`：方向冲突、边沿、原字保持与自动低速测试。
- `replay/config.go`：sim／游戏内播放共同的章节配置。
- `replay/encode.go`：纯原作文件编码器与 USER 块写出。
- `replay/encode_test.go`：13 个原回放的全文件字节强测试。
- `replay/lzss_encode.go`：原树形 LZSS 及逆加密。
- `replay/record.go`：录制器与共用记录／slow-rate 游标。
- `replay/record_test.go`：块边界、重开、重复保存与输入游标测试。
- `replay/validate.go`：用户回放进入游戏数据索引前的安全校验。
- `replay/validate_test.go`：WASM int 溢出、坏索引测试。
- `scene/menus/replay_test.go`：入口 UI 从存储列举及元数据标签测试。
- `scene/stage/menu.go`：原关卡内菜单状态 6–13、名字与保存选择。
- `scene/stage/menu_draw.go`：节点 0x2e、原小字库、捕获纹理的 ANM 绘制。
- `scene/stage/menu_test.go`：菜单门控、名字、排行、保存、非 legacy 绘制测试。
- `scene/stage/playback.go`：共用输入流的游戏内多章播放与返回。
- `scene/stage/playback_test.go`：存储播放、章头恢复、sim 等价与坏入口测试。
- `scene/stage/record.go`：章头快照／恢复、录制模式、t1 不透明字段 hook。
- `scene/stage/record_test.go`：正常、练习、Debug、回放的录制门控。
- `sim/record_test.go`：th10_05 六章三字、样本与完整章头强测试。

### 修改

- `game/config.go`：配置保留原 cfg 原字节。
- `game/game.go`：回放存储接口与原 FPS 采样格式。
- `game/persistence.go`：原 cfg 用户数据读写与回放进度隔离。
- `game/persistence_wasm.go`：进度类型共用与回放进度隔离。
- `game/state.go`：回放 mode、录制器、初始关卡、cfg 与不透明状态字段。
- `input/input.go`：保留回放三字，物理输入恢复时清除回放来源标志。
- `replay/replay.go`：完整文件头／USER／样本／不透明字段、注释勘误与安全边界。
- `scene/ending/ending.go`：仅新增结束后的流程 hook，页面视觉不变。
- `scene/menus/replay.go`：沿用已有风格，从存储列举并进入游戏内播放。
- `scene/select/select.go`：共用关卡工厂、关卡内结算返回与 mode 13 接线。
- `scene/stage/chain.go`：0x2e 正名并取消 legacy，接屏幕层的菜单对象。
- `scene/stage/chain_legacy.go`：旧暂停提示移到链外，不再冒充 0x2e。
- `scene/stage/chain_test.go`：只改授权的 legacy 列表一行，移出 0x2e。
- `scene/stage/stage.go`：录制、菜单、结束流程、配置拷贝、nil 音频接口保护。
- `sim/sim.go`：复用配置／输入流／章头恢复，增加同步录制观察者。

删除：无。

## 3. 移植的函数地址

取证勘误：`FUN_00428f60` 不是文件写出，而是管理器创建／回放加载／章节初始状态。`FUN_004236f0` 的 `0x423f16` 调用 `FUN_00429b60` 才是保存入口。

已移植：`FUN_00429b60` → `replay.Encode`；`FUN_004359b0` → `lzssCompress`；`FUN_00436000` → `lzTree.insert`；`FUN_00436210/00436260/004362b0/00436330` → `remove/unlink/replace/predecessor`；`FUN_0044b220` → `xorEncrypt`。

| 原函数 | Go |
|---|---|
| `FUN_00428f60` mode 0／1 | `NewRecorder`、`beginRecording`、`RestoreReplayChapter`、`Playback.startChapter` |
| `FUN_0042a450`（旧资料误写 0x42a460） | `ReplaySnapshot`、`StartRecording`、`RestoreReplayChapter` |
| `FUN_0042a8a0`／`FUN_004297d0` | `Recorder.Append`、`Stream.Next`、`RestoreReplay`、`ReplayRecord`、`ReplaySample` |
| `FUN_0040ac20` | 三字按下／释放沿及菜单重复键门控 |
| `FUN_00422360`／`FUN_004220e0`／`FUN_00422150` | `NewMenuController` 与阶段 calc／draw 接线 |
| `FUN_00422a90` → `FUN_004223f0` → `FUN_004236f0` | `MenuController.Calc`（end-menu 状态 6–13） |
| `FUN_00422aa0` → `FUN_00422660` | `nodeMenu` → `MenuController.Draw` |
| `FUN_004224c0` | `drawName` |
| `FUN_004231d0`／`FUN_00423370` | `OpenGameOver`／`OpenClear` |
| `FUN_00423510`／`FUN_004243f0`／`FUN_0040c4d0` | `close`、interrupt 6、`selectChoice` |
| `FUN_00424480` | 排队捕获、RGBA→BGRA→ANM 的 host 适配 |
| `FUN_00401760` | 小字库 `glyphText` 的菜单使用子集 |
| `FUN_00421fa0` | `Game.InsertRanking` |
| `FUN_004218d0` mode 13、`FUN_00417c80`、`FUN_00418a90`、`FUN_00417870` fresh | `RestartStage`、select 的 `OnContinue` |
| `FUN_00418c40`／`FUN_00420870` | `DefaultNativeConfig`／`DecodeNativeConfig` |

Ghidra 无法启动后，只用已安装的 `objdump` 只读解析 exe，输出在 `.tmp/codex/t5/evidence/`；没有向沙盒外写入、没有重试 Ghidra。既有 `ecl_dump/top/func_00429b60.c` 提供完整写出函数。

## 4. 测试

阶段 1：`go test -count=1 -v ./replay` 通过。`TestEncodeOriginalReplays` 实际读取 9 个槽位和 4 个 demo，13 个 Decode→Encode 全文件逐字节相同；再 Decode 的全部 Go 字段与源解码对象 DeepEqual。压缩从解压数据重新生成，不缓存原压缩字节。随机／零／循环数据与窗口边界、XOR 所有长度余数、新建回放、非法编码输入测试通过。

`checks_stage1.log`：build＝0，vet＝0，WASM build＝0；全仓 test＝1（既有 `anm/anmdraw.TestRect` 在 `Rect` 未实现桩 panic，属于 t1）；lint＝1（既有问题与本任务触及的旧解码器问题，后者已修）；gofmt 列出 38 个既有 CRLF 文件，未做全仓无关格式化。现有 demo 测试因旧 `.thtk/th10_dat` 路径缺失而 skip；新增测试走规则提供的 junction，全部实际执行。

阶段 2：`TestRecordTh10_05AllChapters` 实际顺序播放六章通过：所有三字输入和 slow-rate 字节、六份 452 字节章节头逐字节相同。+0x04 通过 `PrepareSave` 两次复现，未把源计数直接写入录制器。不透明自机历史／副机字节由章节恢复保留，所有已解码标量来自模拟游戏状态；实时生成不透明字段要接 t1 的 capture hook。记录完整输入游标的过场尾部，不继续世界更新、也不延长 trace。新增输入边沿、3600 帧块边界、重复保存、样本游标和模式门控测试通过。

`checks_stage2.log`：build／vet／WASM＝0，全仓 test 仍只有 t1 的 `TestRect` 阻断；lint 为 35 个既有问题，本任务新文件无报告；gofmt 的 38 个既有 CRLF 文件未改。

阶段 3：新增 `TestMenuNameAndReplaySave`（十帧门控、八字输入、退格、保存后解码）、`TestMenuNameGridWrapping`、`TestMenuRankingAndExit`、`TestMenuDrawUsesANMWithoutBackBuffer`、`TestRankingInsertTieAndShift` 均通过；授权的旧测试仅修改 `chain_test.go` legacy 列表移出 0x2e。`checks_stage3.log`：build／vet／WASM＝0，test 的唯一失败仍为 `TestRect`；lint 35 个既有问题，无本任务问题；剩余 gofmt 34 个既有 CRLF 文件。本任务触及的四个原 CRLF 文件随 gofmt 正常转 LF，未格式化无关文件。

阶段 4 核心测试：桌面 `replay/th10_NN.rpy` 命名、覆盖后新实例读回、TH10_DATA_DIR、失败不覆盖目录且不残留临时文件、内存读写不别名、25 个槽位列举、损坏槽位与越界拒绝通过。存储内容与 `replay.Encode` 逐字节相同；所有用户文件 I/O 都集中于 `game/persistence*`。

阶段 4 完整验收 `checks_stage4.log`：build／vet／WASM＝0；test 只因既有 `TestRect` 失败；lint 35 个既有报告、无本任务报告；gofmt 为 34 个既有 CRLF 文件。新增 WASM 整数溢出防护测试：u32 帧数／长度必须先按正文长度裁切，再转 int；回放索引验证、16 MiB 用户文件读取上限与 64 MiB 解压上限均是项目侧安全边界。

阶段 5 核心测试：游戏内播放从存储读出两章，逐章恢复种子、固定位置、score／faith／power／lives／rank，输入移动自机并在 EOF 返回；无录制器、无进度变化。`TestInteractivePlaybackMatchesSim` 的 1500 帧状态（位置、RNG、score／faith／power／life／rank、敌人／子弹数）逐帧相同。`TestPlaybackNeverMarksProgress` 防止 BeginRun／MarkClear／MarkGameOver 在回放中更新存档。`TestReplayMenuLoadsStorage` 实际构造既有风格 UI 并列出 25 个槽位。

`checks_stage5.log`：build／vet／WASM＝0；test 只因既有 `TestRect` 失败；lint 35 个既有报告、无本任务报告；gofmt 为 33 个既有 CRLF 文件。阶段 6 核心测试：`TestNativeConfigOriginalFileRoundtrip` 实际读取本机原 cfg，52 字节 roundtrip 相同；默认键表、600 字、版本、BGM／SE 与 0x100 默认字按 `FUN_00418c40` 取证；加载／保存只修改已编辑音量，其他原字节与 +0x30 保留。cfg 的 0x200 自动低速选项按 `FUN_004297d0` 接线，回放输入不受 host 选项影响。

### 最终全仓结果

`checks_final.log`、`trace_final.log` 是交付时最后一次真实运行：

| 检查 | 结果 |
|---|---|
| `go build ./...` | 通过，exit 0 |
| `go vet ./...` | 通过，exit 0 |
| `go test -count=1 ./...` | exit 1；唯一失败为既有 `anm/anmdraw.TestRect`，`rect.go:32` 未实现桩 panic（t1）；其余包通过 |
| `golangci-lint run` | exit 1，35 个既有报告：errcheck 26、staticcheck 4、unused 5；本任务文件无报告 |
| `gofmt -l .` | 非空，32 个未触及的既有 CRLF 文件；本任务新增／修改 Go 文件已 gofmt |
| `GOOS=js GOARCH=wasm`，排除 tools | `go build` 通过，exit 0；先在 host 下列包，避免 tools/th10probe 的 Windows-only 枚举错误 |

未删除／放宽 `TestRect` 或其他旧断言，未重生成 golden，未修改无关文件替全仓 lint／CRLF 擦尾。完整失败输出保留在任务根目录。

## 5. Trace 结果

阶段 1：六章与基线流式二进制 cmp 全部相同，字节数依次为 94186573／144171327／271969413／319443819／430006225／510140938。新生成 trace 均已删除，保留 `trace_stage1.log`。

motion：c1–c5 为 17733／29763／29469／39343／40099。规则指定的 c0 `orig.txt` 在 4915 帧含半帧 MSG 状态；用同一命令比较基线也在该帧分歧，因此不是回归。已有 settled `orig_d.txt` 与基线为 15853 帧全部一致；本阶段 c0 与基线逐字节同一，达到原有覆盖。最终验收会再次分别记录这两个输入。

阶段 2 再次生成六章，cmp 全部相同，字节数与阶段 1 相同；`orig_d.txt` 一面 15853 帧与其余五章 17733／29763／29469／39343／40099 全部一致。规定的 `orig.txt` 仍只有相同的 4915 帧采样分歧。输出 trace 已删除，保留 `trace_stage2.log`。

阶段 3 同样六章 cmp 相同，settled motion 一致帧数仍为 15853／17733／29763／29469／39343／40099，无任何下降；`trace_stage3.log` 保留结果，trace 本体已删除。

阶段 4 再跑六章，cmp 字节数与阶段 1 相同，settled motion 仍全绿（15853／17733／29763／29469／39343／40099）；保留 `trace_stage4.log`，trace 本体已删除。

阶段 5 六章 cmp 与基线逐字节相同；settled motion 一致帧数仍为 15853／17733／29763／29469／39343／40099。`trace_stage5.log` 保留结果，trace 本体已删除。

最终阶段 6 再次六章 cmp 全部相同：

| 章 | 与 base cmp 字节数 | settled motion 一致帧数 |
|---|---:|---:|
| c0 | 94186573 | 15853（`orig_d.txt`；规定的 `orig.txt` 有同一 4915 半帧问题） |
| c1 | 144171327 | 17733 |
| c2 | 271969413 | 29763 |
| c3 | 319443819 | 29469 |
| c4 | 430006225 | 39343 |
| c5 | 510140938 | 40099 |

没有 RNG、运动或 trace 字节回归。所有生成的 cN 大文件已删，只留日志；未动任何基线／原作 trace。

## 6. 待决与推测

Ghidra 首次调用被沙盒策略拒绝，未启动；按 rules 记录 `.tmp/codex/t5/dump_requests.md`，不重试、不绕过。编码、快照、菜单、重开与 cfg 后来均由既有 dump 加只读 objdump／PE 常量解析解决；剩余请求主要为 `score.dat` 原生容器／符卡历史读写。

真正未完成／有界 host 适配：

- 现场录制的 33 项自机历史、四组副机内部状态仍待 t1 capture／restore hook；强测试对此部分是在原章头恢复后保持原字，不冒充实时移植。
- 当前原作目录没有 `score.dat`；原容器校验／压缩与符卡历史持久化未完成，保留既有 JSON 进度，并给出确定的排行布局。
- 全局 slow-rate 的原 double 计数器＋0x24／＋0x2c 尚无等价 host 数据源；新现场录制的总慢放率目前为 0，逐 30 帧样本使用 host logic FPS 的原格式，首窗口用 60 是 host 适配。源回放所有原慢放字段在 Decode→Encode 中原字保持。
- 正常关间完整 fade／transition 流程仍由现有场景切换承担；验收源 transition-tail 输入完整保留，但无头世界不继续跑这些记录。
- 正常六面经现有 ending 的流程 hook，结局／Staff 页面视觉仍为其他任务；原 MSG 0x14 通关加分 `FUN_00409d90` 未在本任务移植，不改变既有 trace 计分。
- 菜单截图时机／像素、小字库子集需用户目视；原暂停 flag 未找到，暂停仍为项目侧近似。

已消除编码器待决：LZSS 的等长匹配更新、整匹配替换节点、末尾仅写 14 个零 bit 且丢弃不满字节均按汇编移植。v5 写出路径不计算校验和，保留原作 +0x10＝0x100 默认字与所有保留字。trailer 不是固定 200 字节（th10_05 为 208）。章节 +0x04 双倍是原作写出时未清零累加，见 `FUN_00429b60`；Encoder 保留源计数。

菜单：暂停 flag 仍未定位，旧暂停显示保留在链外，未再冒充 0x2e。原作截图请求由 `FUN_00424480` 排队，Ebiten 适配为节点 0x29 前复制场地 RGBA 到 BGRA 纹理，再由 ANM 绘制；复制时机／像素结果待用户目视。为冻结游戏对象而让菜单动画继续，菜单使用独立 World、共享备用 RNG；不改变无头逻辑。

mode 13 已补齐：`FUN_004218d0` 的表 `0x421b0c` 第 13 项 → `0x421ac7`：释放旧关卡，fresh-run 标记＝1，新建非回放关卡。`FUN_004175e0` 的 `0x4175e0..0x4175e9` 只递增当前关卡＋0x3c（`DAT_00474c7c`），不改最初请求关卡＋0x40（`DAT_00474c80`）；因此超过初始关卡后的 mode 13 析构会以 `0x474c40` 调 `FUN_00418a90`（`0x417d92`），计数＋0x50（`DAT_00474c90`）封顶 9，置 0x8。按 `FUN_00417870` fresh 分支重置当前关卡，已由 `GameState.RestartStage` 和 select 的 `OnContinue` 接线；初始关卡重开不计续关。`TestRestartStageOriginalFreshState` 验证此分支与封顶。原作 cfg 可选择的练习残机数尚未建模，使用已确认的默认 9。

## 7. 建议的 docs/fidelity.md 改动

建议替换的完整状态行（未直接改主账）：

```text
| .rpy 回放解析 | 位精确（解码／编码）；方向按 FUN_004250b0 | FUN_00428f60 管理器、FUN_00429b60 写出、FUN_004359b0/00436000/00436210 树形 LZSS、FUN_0044b220/0044b0d0；变长 USER、slow-rate、重复保存计数原字保持 | replay/encode_test.go：9 槽＋4 demo Decode→Encode 全文件字节相同，Decode(Encode(x)) DeepEqual |
| 回放录制/播放 | 输入与已解码章头移植；现场不透明自机状态待 t1，host 慢放统计适配 | FUN_0042a8a0/004297d0/0042a450；sim 与 stage.Playback 共用 ConfigFor／Stream／RestoreReplay，存储中的原 .rpy 可游戏内播放；DAT_00477830 end-menu 经 ANM | th10_05 六章三字／样本／452 B 章头强测试；多章存储播放、sim 1500 帧逐帧等价；六章 trace 与基线字节相同 |
| 菜单/结局/Staff | 主菜单／结局仍近似；关卡内 end-menu 状态 6–13 已移植，视觉待目视 | FUN_00422360/004223f0/00422660/004231d0/00423370；0x2e 非暂停且非 legacy；原 front.anm／ascii.anm，mode 13 重开接线 | TestMenu*、TestRankingInsertTieAndShift、TestRestartStageOriginalFreshState；用户目视 |
| 存档/配置 | 回放原格式存储、cfg 52 B 互通；score.dat 容器／符卡历史未完成，进度 JSON 保留 | game/persistence*；桌面 replay/th10_NN.rpy 安全覆盖，WASM 页面内存；FUN_00418c40/00420870，cfg＋0x30 接线；FUN_00421fa0 排行记录布局 | 存储／覆盖／损坏／越界测试；原 cfg 52 B roundtrip；WASM build |
| 无头运行器 | 位精确（确定性），支持同步录制观察者 | sim/、stage.NewHeadless；与游戏内播放共用输入源与章头恢复 | TestHeadlessDeterministic、TestRecordTh10_05AllChapters、TestInteractivePlaybackMatchesSim |
```

建议按现有更新记录格式添加：

```text
- 2026-11-07 — t5：回放写出 FUN_00429b60、原树形 LZSS FUN_004359b0（等长匹配、替换节点与截断终止位）及逆加密 FUN_0044b220；9 个用户槽位＋4 demo 全文件 Decode→Encode 字节相同。补齐原文件头／变长 USER／逐 30 帧样本，确认重复保存 +0x04 累加是原作行为。录制按 FUN_0042a8a0／FUN_0042a450，sim 同流录制 th10_05 六章输入／样本／章头字节一致；不透明自机历史／副机留 t1 hook。DAT_00477830 的 end-menu 状态 6–13、名字／排行／保存选择经原 front.anm／ascii.anm 绘制，节点 0x2e 正名并移出 legacy，旧暂停显示移到链外。统一用户数据 I/O，桌面原 replay/th10_NN.rpy、WASM 会话内存；入口菜单可游戏内多章播放并返回，不更新进度。按原 mode 13 接关卡重开；cfg 52 B 结构／默认值／+0x30／录制配置快照互通，score.dat 容器未完成。六章 trace 与基线逐字节相同，settled motion 15853/17733/29763/29469/39343/40099；orig.txt 4915 半帧 MSG 与基线同样分歧，正确 settled 文件是 orig_d.txt。build/vet/WASM 通过；全仓 test 仍有 t1 Rect 桩，lint／CRLF 有既有阻断，未削弱测试或重生成 golden。
```

## 8. 需要用户目视确认的项目

在合并树用 PowerShell 设置明确的用户数据目录，例如 `$env:TH10_DATA_DIR='D:/GoProjects/th10/.tmp/t5-userdata'`，再 `go run ./cmd`；不要在只读原数据 junction 中写入。

1．普通游戏结束：等自机 state 2 的 30 帧结束，确认 Game over 动画；Z 经门控打开排行／Continue／Save replay／Quit；选择 Save、槽位、八字名字，测试退格与最后三个特殊格。
2．练习、Extra 与普通六面通关：练习／Extra 直接开原 clear 控制器，普通六面先经现有结局页再进入；确认名字／排行、Next Scene、退出与 BGM。
3．保存后回标题选 Replay：确认数据目录有 `replay/th10_NN.rpy`，槽位显示名字／机体／难度／日期，选择后多章连续播放；X/Esc 或播放结束返回回放菜单，不加 TotalRuns／TotalClears 或高分。
4．确认截图背景没有位置偏移，原小字库、黄色选中项、闪烁／选择 ANM 与原游戏一致；Esc 旧暂停提示仍独立保留，不能据此认定原暂停 flag 已还原。
5．mode 13 重开：初始关卡不加续关计数，推进后的关卡重开计数加一并封顶 9；检查 power／lives／rank 初始化。
6．**由用户操作原作**：把复刻现场录制的原 .rpy 放入原作回放槽位播放，确认机体射击／副机与跨关状态；完成 t1 hook 后再做互通强验收。本任务未启动 th10.exe、probe 或 junction 中任何程序。

## 9. 与其他任务的接口

符卡历史由 t2 更新。本任务只负责存档格式；取得 t2 的布局后记录接线签名。不实现自机／GUI／符卡／主菜单视觉的其他任务内容。

t1：`Context.CaptureReplayPlayer func(header []byte)`、`Context.RestoreReplayPlayer func(header []byte)`，参数为完整 0x1c4 章节头，布局按 `0x42a4f0..0x42a589`／`0x42a5c3..0x42a687`。位置 +0x24/+0x28 与 focus +0x1bc 已接；+0x2c 的 33 个位置历史及 +0x134 起的四组副机坐标、目标与历史需接实时结构。`player.ReplayFocus() int32`、`RestoreReplayFocus(int32)` 只暴露已有 +0x4474，不改变自机状态机。

录制模式取证勘误：`FUN_00429610`／`FUN_00428f60` 只按 stage +0x5c 的 mode 0／1 判断，**练习也录制**；`FUN_004236f0` case 7 的练习分支跳过排行，但 case 8 同样提供回放保存。Debug 模式为项目侧例外，不自动录制。FPS 样本的 +0.5、256 截断已按原作取位；host 的第一个 FPS 窗口尚未完成时用 60 Hz 是项目侧适配，时间统计并不声称与原作墙钟相同。

菜单／结局任务：`ending.Ending.After func(*game.Game) game.Scene` 只接 post-ending 到同一关卡内排行／回放菜单，不改变结局页面视觉。正常六面经已有 ending 再打开菜单；练习与 Extra 按 MSG 0x14 直接 `OpenClear`。`Stage.OnContinue func(*game.Game, *game.GameState) game.Scene` 对应 `FUN_004218d0` 的 mode 13。`Context.Menu.State` 是原 `DAT_00477830+4`，t1 信仰矩形门控可直接读取。

原存档排行布局：`game.ScoreEntry` 对应 24 字节记录，偏移为 Score＋0（i32）、Stage＋4（u8）、Continues＋5（u8）、Name＋6（9 字节含 NUL）、Timestamp＋0x10（time32）、SlowRate＋0x14（f32）；机体块从 score 对象＋8 开始，步长 0x437c，排行从块内＋0x10 开始，难度步长 0xf0。`Game.InsertRanking` 按 `FUN_00421fa0` 处理同分插前和六字搬移；目前经原有 JSON 进度存储，不声称 score.dat 容器已完成。

cfg 接口：`game.NativeConfig.Raw [0x34]byte`、`Flags() uint32`、`SetFlags(uint32)`、`Config.NativeBytes() []byte`；`GameState.ConfigFlags` 是 cfg＋0x30 拷到 stage＋0x54 的值，供 t1 矩形门控；`GameState.ReplayConfig [0x34]byte` 按 `FUN_00428f60` 写在 global＋0x14。回放与 sim 的共同 `replay.RunConfig` 从 global.Raw 恢复同一配置。

对外可调用的接口：

- `replay.Encode(*Replay) ([]byte, error)`：纯编码、无文件 I/O；原 `FUN_00429b60` 的保存菜单调用点在 `0x423f16`。
- `replay.NewRecorder(GlobalHeader) *Recorder`、`BeginChapter(Chapter) error`、`Append(Record, byte)`、`PrepareSave() *Replay`：分别对应初始化、章节快照、每帧追加和重复保存计数。
- `replay.NewStream(Chapter) *Stream`、`Prime() Record`、`Next() (Record, byte, bool)`：sim 与游戏内播放共用。
- `Game.LoadReplay(int) (*replay.Replay, error)`、`SaveReplay(int, *replay.Replay) error`、`ListReplays() []ReplaySlot`：集中用户数据入口；列举只保留元数据，选中后读完整输入。
- `stage.NewPlayback(*game.Game, *replay.Replay, int, func(int) Script, game.Scene) (*Playback, error)`：由回放菜单入口调用，末尾参数是播放结束的返回场景。
- `select.ReplayScript(int) stage.Script`：共用现有关卡工厂，不再复制每面初始化入口。

外部原函数／接线边界：GUI／MSG 的 `FUN_00415e90` op 0x14（练习／Extra 调 `FUN_00423370`）仍由原有 MSG hook 置 cleared，再由 Stage 打开菜单；其绘制、原通关奖金 `FUN_00409d90` 留给对应任务。原自机状态保存 `0x42a4f0..0x42a589` 及恢复 `0x42a5c3..0x42a687` 的未建模字段留给 t1 hook。t2 符卡历史接口尚无原生 score.dat 布局，不能给出互通承诺；计数仍归 t2，本任务不增加符卡计数。

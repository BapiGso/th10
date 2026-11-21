# t15：WASM 运行与发布交付

更新时间：2026-11-08。工作树：`D:\GoProjects\th10\.tmp\codex\t15\tree`。

## 1．摘要

按 A→B→C→D→E 顺序进行。A 普查、B 打包／预览、C IndexedDB／原格式导入导出、D 启动门／固定 TPS／调试／音频评估、E 六章 Node 一致性与性能测量均已实施，五阶段必需完整验收全部通过。没有运行浏览器、原作、probe 或 Ghidra，没有修改资源格式、scene、th10sim、th10diff 或 trace 格式。交付文档工作树与主树各一份，任务目录写 DONE 完成时间。

明确未做：真实浏览器／手柄设备与 GPU 性能实测；冷路径偶发超过 16.7 ms 的峰值没有宣称已彻底消除。缓存预热后六章逻辑最大值均低于 16.7 ms，冷结果与复测都保留。保存调用继续使用 t5／t8 的既有离散宿主检查点，其与原 writer 的磁盘时序差别见第 9 节。

浏览器启动先运行空宿主包装帧以安装 Oto 的真实用户事件监听，之后 TPS＝0；一帧空 Draw 边界清除等待时间，用一次 SyncWithFPS tick 启动恰好一个真实第一帧，再固定 TPS＝60、关闭失焦更新，绕开桌面 Pacer。sig 的真实 DrawCount 不在门内增加。调试层显示逻辑 TPS、Draw FPS、逐帧耗时与一秒均值／最大值；默认关闭，F3 或按钮切换。

BGM 的 js 入口使用 200 ms 源缓冲，桌面保留默认值。冷启动标题曲建流／读取仅 200 ms PCM 在门内完成，没有播放器、解锁观察者或 RNG 调用；BGM 真正开始、原循环点、原淡出帧数不变。

一次 `go run ./tools/wasmbuild` 生成 dist。最终 `th10.wasm`＝174,610,642 B（166.52 MiB），标准库 gzip level 9＝103,256,693 B（98.47 MiB）；Go＝go1.27.3。`dist/` 已在原 `.gitignore` 中，未作无必要的重复修改。体积没有使用 t14 的资源瘦身成果。页面额外的 game.html／mjs／release.json 及各自 .gz 也一次生成，不需要外部解码／压缩程序。

## 2．改动文件清单

新增：

- `docs/research/wasm_release.md`：独立的阶段 A 普查、原保存时机、依赖源码行为、本机工具位置与 Node 初始化失败。
- `docs/handoff/codex_t15.md`：分阶段交付与验收记录。
- `tools/wasmbuild/main.go`：一键 js／wasm 构建、本机 Go glue 查找、可选 loopback 预览服务。
- `tools/wasmbuild/release.go`：原子文件发布、标准库 gzip 预压缩、尺寸／SHA-256 清单。
- `tools/wasmbuild/serve.go`：WASM MIME、显式／协商 gzip、HEAD、identity Range 与发布文件白名单。
- `tools/wasmbuild/release_test.go`：发布文件、gzip 往返、glue、MIME、HEAD、Range、非发布路径测试。
- `tools/wasmbuild/web/index.html`：网页外壳、下载进度、整数缩放 iframe、全屏按钮。
- `tools/wasmbuild/web/game.html`：同源游戏 iframe 文档，隔离 Ebiten body／canvas 与外页工具栏。
- `tools/wasmbuild/web/app.mjs`：下载与发布清单、完整下载后移交 ArrayBuffer、缩放及全屏。
- `tools/wasmbuild/web/player.mjs`：iframe 的 Go 实例启动、错误转发与握手。
- `tools/wasmbuild/web/helpers.mjs`：按解压后长度的流式下载与整数倍尺寸计算。
- `tools/wasmbuild/web/helpers.test.mjs`：五项 Node 网页纯函数／下载／键盘默认行为测试。
- `tools/wasmbuild/web/preview.test.mjs`：真实 loopback 服务的原始 HTTP MIME／gzip 长度与 Range 测试。
- `browserstore/store.go`：同步 Go 调用、事务提交后确认的二进制后端接口。
- `browserstore/store_js.go`：真实 IndexedDB 的打开／升级、ArrayBuffer 读写、事务完成／失败与诊断事件。
- `browserstore/store_other.go`：桌面不可用的对应实现，不替代桌面文件后端。
- `browserstore/store_other_test.go`：桌面不选浏览器后端。
- `browserstore/store_js_test.go`：js 适配器的重载／复制、等待事务提交与失败保留旧数据测试。
- `game/persistence_browser.go`：score／cfg／25 回放槽与通用二进制后端的适配，可在桌面用假后端测试。
- `game/persistence_transfer.go`：原格式严格导入与原字节导出；拒绝 score 原解析器允许跳过的坏段。
- `game/persistence_browser_test.go`：导入拒绝、真实原文件字节、重载、显式写入检查点与导出无写入。
- `game/browser_runtime_js.go`：通过启动门后只重置宿主计时窗口，不重置场景或 RNG。
- `cmd/runtime.go`：同时挡住真实 Update／Draw 的门，以及导入／首次帧互斥。
- `cmd/runtime_test.go`：无手势／音频未 ready 时零场景帧、一次开始与导入互斥测试。
- `cmd/runtime_js.go`：浏览器 Promise 导入／导出桥、音频 ready 门与正常 Quit 保存。
- `cmd/runtime_other.go`：桌面仍直接调用 Ebiten RunGame。
- `tools/wasmexec/run.cjs`：开发期 Node Go WASM 执行器，本机 Go glue 与显式工作区临时目录。
- `tools/wasmexec/testdata/indexeddb.cjs`：仅显式开启的事务事件假后端，不进入 dist、不替代浏览器数据库。
- `game/clock_js.go`：固定 60 TPS，禁止桌面 Pacer 二次过滤。
- `game/clock_other.go`：桌面保持 SyncWithFPS 与原 Pacer。
- `game/clock_js_test.go`：Node js 实测目标 TPS＝60、固定 ticks 不再次过滤。
- `cmd/metrics.go`：独立逻辑／Draw 计数、每帧墙钟耗时与窗口均值／最大值。
- `cmd/metrics_test.go`：60／120／144 Hz 计数分离与最后值／均值／最大值。
- `audio/stream_buffer.go`：宿主可选 BGM 源缓冲 API，不修改 codec 或循环点。
- `audio/stream_buffer_test.go`：默认零值不变、设置不播放、负值恢复默认。
- `tools/wasmbuild/web/player.test.mjs`：JS 页面门等待 Go 首帧通知、焦点与快捷键隔离。
- `tools/wasmexec/headless.cjs`：只满足 Ebiten 包初始化的开发期 DOM facade，刻意不提供 WebGL、RunGame 或音频设备。
- `tools/wasmperf/main.go`：Node／原生可运行的性能评估 CLI。
- `tools/wasmperf/audio.go`：18 首流式 Vorbis 的建流、不同 pull 长度与回环读取耗时。
- `tools/wasmperf/memory_js.go`：获取当前 WASM linear memory 大小。
- `tools/wasmperf/memory_other.go`：非 js 的对应实现。
- `cmd/finalscreen_js.go`：FinalScreenDrawer 强制最近邻，保留 Ebiten 的比例与居中矩阵。
- `sim/profile.go`：新 Measure 方法，只计时既有无头 step、不产生 trace I/O，报告堆与两 RNG。
- `sim/profile_test.go`：测量与普通 Run 的最终 trace／主 RNG 不变。
- `tools/wasmperf/logic.go`：通过原恢复路径与 ReplayScript 测量完整章，支持显式预热次数。
- `tools/wasmexec/compare.cjs`：分块逐字节比较，失败输出首次偏移与帧／行，不使用 epsilon。
- `tools/wasmexec/compare.test.cjs`：跨块比较、首差帧与 EOF 长度差测试。
- `tools/wasmcheck.ps1`：顺序六章 native／WASM／基线比较、RNG 核对、性能 JSON 与 Node 包测试。

修改：

- `cmd/main.go`：入口分派平台准备与运行，桌面路径保持一致。
- `game/persistence_wasm.go`：cfg 与 score 从会话内存切换为 IndexedDB。
- `game/persistence_replays_wasm.go`：回放从会话内存切换为 IndexedDB。
- `game/persistence_replays.go`：纠正 MemoryReplayStore 已不再是浏览器默认后端的注释。
- `game/game.go`：时钟分派到 js／非 js，小改动保持桌面 Pacer 原行为。
- `audio/audio.go`：新增可选 BGM 缓冲字段与 NewPlayer 后的设置，默认零值不改变桌面。

删除：无。任务目录中的验收脚本与日志不属于合并源码；dist 由工具生成且已忽略。

## 3．移植的函数地址

没有新增原作函数移植。复用 t5／t8 的 `FUN_0042b030/0042b1e0 → DecodeScoreFile/EncodeScoreFile`，`FUN_00420870 → DecodeNativeConfig`，`FUN_00429b60 → replay.Encode` 与既有回放验证；浏览器存储、网页与调试层是宿主适配，不虚构 FUN 地址。正常 Quit 的 cfg／score 对应 `FUN_00438ad0`／`FUN_0041fb50`，关卡保存仍在原 `0x417c8d` 的既有 Scene hook。

## 4．测试

阶段 A：原样 th10sim WASM 构建成功，原生 Go Node 执行器试跑退出 2，最先在 Ebiten `internal/colormode.init` 对缺失的浏览器对象取属性时 panic；尚未进入逻辑。原始输出：任务目录 `logs/A/plain-node.txt`。

阶段 A2：build、vet、`go test -p 2 -count=1 ./...`、非 tools 的 WASM build 均退出 0；gofmt 输出为空；full smoke 退出 0（163.810 秒）。阶段 A 当时没有新增测试；全部阶段都没有更改已有断言或 golden。

阶段 A 首次验收脚本在 js 环境中执行 `go list ./...`，Windows-only 的 th10probe 使 list 退出 1，尽管后续过滤 tools；这是脚本的包枚举顺序问题，不是基线代码失败。改为先在 native 环境枚举并过滤，再设 js／wasm 构建，完整重跑 A2。首轮日志保留于 `logs/A`。

阶段 B 相关检查：`go test -count=1 ./tools/wasmbuild` 退出 0（三项顶层测试及子测试）；`node --test tools/wasmbuild/web/helpers.test.mjs` 四项全部通过；实际 `-skip-build -serve 127.0.0.1:18015` 服务可访问，Node 原始 HTTP 检查一项通过（`logs/B/http-raw.txt`）。identity MIME＝application/wasm、长度＝174,428,459；协商与显式 `.gz` 的 Content-Encoding＝gzip、长度＝103,218,816；Range 返回 identity 的 wasm magic。这里验证的是 HTTP，不是浏览器绘制。

开发期首次新增 HEAD 测试发现 ServeContent 在设置 Content-Encoding 后不会自动给预压缩文件补 Content-Length，已补显式 gzip 长度并复测通过，没有改测试断言。一个后台 Start-Process 组合命令被策略拒绝、未执行；改用 CLI 前台预览服务验证，服务结束后只停止了已核对路径属于本任务 GOTMPDIR 的进程。

阶段 B 完整验收：build／vet／test／WASM 退出 0，gofmt 为空，full smoke 退出 0（144.769 秒）；六章 cmp 全部相同，motion 仍为 15853／17733／29763／29469／39343／40099。

阶段 C 相关检查：`go test -count=1 ./cmd ./game ./browserstore ./tools/wasmbuild` 全部通过；当前 cmd WASM 链接通过。新增六项 game 测试、两项 cmd 门测试与一项桌面后端测试；真实原 scoreth10.dat／th10_05.rpy 导入测试实际读取成功，原字节未改。原解析器遇坏 CR／ST 会跳过并返回默认值，显式导入额外拒绝 `Rejected`、不一致文件长度、缺失／重复段，不改变 t8 解析器行为。

`GOOS=js GOARCH=wasm go test -exec <Node executor> ./browserstore` 在显式 fake IndexedDB 下三项 js 测试全通过：新连接重载、数据不别名、Write 在 request 成功后仍等待 transaction.oncomplete、注入 QuotaExceededError 保留原回放。日志 `logs/C/node-storage.txt`。这验证实际 Go js 回调契约，不冒充真实浏览器 IndexedDB 配额／持久化实测。

阶段 C 完整验收：build／vet／test／WASM 退出 0，gofmt 为空，full smoke 退出 0（146.384 秒）；六章 cmp 相同、motion 数不降。C 当时发布体积为 wasm 174,554,156 B／gzip 103,247,117 B；最终体积将在 E 后重新报告。

阶段 D 相关检查：60／120／144 Hz 的计数器测试、首帧 clock 边界、BGM buffer opt-in 测试已通过。Node 运行实际 js 的 `TestBrowserFixedTPSHasNoSecondPacer` 通过（`logs/D/node-tps.txt`），这里确认目标时钟与无二次限速，不冒充 120／144 Hz 浏览器实机测试。

Node WASM 音频评估使用 `tools/wasmperf -kind audio -blocks 32`，18 首全部成功（`logs/D/audio.json`）。各首每种缓冲 32 次 ReadFull，500／200／100／46.44 ms PCM 块的全局最大值约 9.63／3.43／1.06／0.89 ms；冷建流峰值 40.46 ms（第一首），其余首建流峰值最高 11.19 ms；近末端 Seek 最高约 0.91 ms，跨循环点 Read 最高约 12.81 ms。因此选择 200 ms 以减少连续解码 burst，标题曲冷路径先预热；不整曲解码、不改资源编码或循环点。主线程 Go 解码／mixer 仍可能与绘制竞争，浏览器工作线程出声、爆音与 loop 必须用户试听。

阶段 D 完整验收：build／vet／test／WASM 退出 0，gofmt 为空，full smoke 退出 0（122.032 秒），六章 cmp 相同与原作 motion 数不降。

阶段 E 已通过：新增 sim 测量一致性测试；网页／比较器 Node 七项测试；最终 js 范围 `golangci-lint run ./browserstore ./game ./cmd` 为 0 issues。js 适配器三项测试在最终修改后再次通过。新 js lint 发现的忽略 channel error、无效赋值与弃用 screen filter 已修，最后改用 FinalScreenDrawer，不放宽测试。

完整 native lint 非验收项但实际运行：本树与只读 `.tmp/codex/base` 都退出 1，默认口径均 32 issues（errcheck 26／staticcheck 2／unused 4）；没有本任务新增代码报告。完整日志为 `logs/E/lint-workspace-cache.txt`、`lint-base.txt`。首次使用系统 lint cache 被拒并产生大量警告，日志保留；后来配置 `GOLANGCI_LINT_CACHE` 到任务可写目录复跑。没有越界修其他任务的旧报告，也不把整个 lint 称为全绿。

Node 非图形测试，全部 `GOOS=js GOARCH=wasm go test -p 1 -count=1 -exec <executor>`，每包独立 Node 进程，全部退出 0：`browserstore` 1.250 s、`collision` 0.247 s、`datfs` 2.261 s、`input` 1.694 s、`pacing` 1.605 s、`replay` 1.868 s、`sht` 2.459 s、`std` 2.323 s、`trace` 0.839 s、`zun` 0.866 s（命令总墙钟）。input／std 的 Ebiten 依赖只用初始化 facade，不调用绘制。gfx 的 GPU TestMain、完整 scene Draw smoke 没有在 Node 跑，不冒充浏览器绘制测试。

Node 原始执行器在 colormode 失败；第一版 facade 又在 legacy ebitenui 的 requestAnimationFrame 注册处失败。按实际依赖源码，仅加不调度的 RAF no-op；不运行注册的触摸回调、不提供 WebGL／AudioContext、不调用 RunGame，既有 th10sim 无需修改。首轮日志 `node/node-c0.txt` 保留，完整成功重跑为 `node2`。

阶段 E 最终必需完整验收（`logs/E/summary.txt`，2026-11-08 13:40:06＋01:00 完成）：`go build ./...` 退出 0（7.765 s）、`go vet ./...` 退出 0（0.983 s）、`go test -p 2 -count=1 ./...` 退出 0（89.866 s）、gofmt 输出为空、非 tools WASM build 退出 0（2.443 s）、`TH10_SMOKE=full go test -count=1 -run TestReplayDrawSmoke ./scene/stage/` 退出 0（117.236 s）。最终六章 cmp 都相同、原作 motion 都 MATCH。没有以基线失败解释任何必需验收失败。

最终真实 HTTP 服务再次验证通过（`logs/E/http-final.txt`）：identity／协商 gzip／显式 .gz 的 MIME、Content-Encoding、Content-Length 与上述最终尺寸相符，Range 返回 identity wasm magic。只验证服务，不冒充打开过浏览器。临时预览进程已按实际可执行路径核对后停止，成功大 trace 已清理。

## 5．Trace 结果

阶段 A2 六章顺序回归：c0…c5 的 `cmp.exe -s` 全部退出 0，逐字节相同。原作 motion 一致帧数分别为 15853／17733／29763／29469／39343／40099，均 MATCH、无下降。trace 比较后按规则删除，只保留日志与 SHA-256。

基线固定为主树 `.tmp/trace/d1c2/base_cN.txt`，c5 使用第三轮 MSG 修正后的版本，不使用 `base_c5.pre_t13.txt`。A2／B／C／D 每轮六章都逐字节相同，motion 始终不下降。

阶段 E Node 逐章顺序结果（`node2/summary.txt`）：

| 章 | 原生＝base | Node＝原生 | 完整逻辑帧 | trace 字节 | 原作 motion 一致帧 |
|---|---|---|---:|---:|---:|
| c0 | 相同 | 相同 | 15972 | 94186573 | 15853 |
| c1 | 相同 | 相同 | 19831 | 144171327 | 17733 |
| c2 | 相同 | 相同 | 31699 | 271969413 | 29763 |
| c3 | 相同 | 相同 | 31301 | 319443819 | 29469 |
| c4 | 相同 | 相同 | 40667 | 430006225 | 39343 |
| c5 | 相同 | 相同 | 50408 | 510168148 | 40099 |

比较器逐字节扫描，不用哈希代替比较，也没有新增浮点容差。六章主 RNG 与备用 RNG 的最终 State／Count 与独立原生 Measure 相同；显式 warmup 后再核对也相同。成功比较后删除大 trace，只保留摘要／JSON。c5 原作参考结束后的 5408 帧只是与已接受基线及 Node 相同，仍不能称已由原作独立验证。

逻辑性能，不计加载、trace 采样／格式化或输出 I/O；计时包裹与原 Run 相同的 `h.step`（输入恢复、完整无头更新），每章单独进程，未改关内算法。初次冷样本与显式一次预热结果如下；预热是重新创建同一确定性局并跑完一遍，测量局再次按原章头恢复，并非跳过输入。

| 章 | 冷均值 ms | 冷最大 ms／帧 | 预热后均值 ms | 预热后最大 ms／帧 | 冷加载 Go HeapAlloc B | 冷加载 linear memory B |
|---|---:|---|---:|---|---:|---:|
| c0 | 0.0742 | 15.753／4915 | 0.0544 | 6.758／2521 | 16234144 | 190840832 |
| c1 | 0.1221 | 12.050／2553 | 0.1374 | 10.173／16314 | 17714456 | 192413696 |
| c2 | 0.1297 | 12.506／3793 | 0.0929 | 5.249／7089 | 19093944 | 193462272 |
| c3 | 0.1940 | 24.708／8601 | 0.1240 | 7.148／26155 | 19407752 | 194510848 |
| c4 | 0.1300 | 14.193／6203 | 0.1056 | 3.364／26158 | 16700208 | 191365120 |
| c5 | 0.1465 | 11.927／3704 | 0.1405 | 6.382／40627 | 18222272 | 192937984 |

独立冷复测（`node-perf2`）c3 降为 9.615 ms，但 c2 出现 24.070 ms／3793；因此**不能保证所有冷帧都低于 16.7 ms**。缓存预热后全局最大 10.173 ms，逻辑单项有至少约 6.5 ms 余量；不能把这余量当成包含音频／GPU 的整个浏览器帧预算。HeapAlloc 是 Go 管理堆；linear memory 含嵌入的约 166 MiB 静态资源，不等于 JS／GPU 总内存。完整 HeapInuse／Sys／GC 也在 profile JSON 中。

## 6．待决与推测

浏览器实测被任务环境限制，所有音频、画面、刷新率、输入、全屏与真实 IndexedDB 行为必须由用户按第 8 节检查；Node 不替代浏览器。没有新增 dump 请求或关内逻辑推测。

性能待决：冷样本约 24 ms 的孤立峰值仍存在。其来自冷路径与／或共享机器宿主调度是**推测**，尚未区分 V8 编译、字体首次处理与线程停顿；均值与 warmup 样本不能抹掉这些峰值。浏览器 F3 的逐帧最大耗时供用户定位，资源／字体收缩由 t14 合并后复测；本任务没有为“跑绿”删除慢帧或改关内逻辑。

持久化宿主边界：仍沿用 t8／t11 已有的更早离散检查点，不能称原 writer 的逐指令磁盘时序完全相同。IndexedDB 配额、清理策略、隐私模式、多个页签最后写者覆盖不能由 fake 测试证明；同 origin／端口、单个活动页签、定期导出备份是建议。页面关闭时不依赖异步 beforeunload；正常游戏内 Quit 会等待提交。

## 7．建议的 docs/fidelity.md 改动

没有直接修改总账。建议完整状态行：

```text
| WASM 发布／预览 | 宿主工具实现；浏览器出声／像素／设备待目视 | t15 tools/wasmbuild：本机 Go glue、trimpath／s-w、stdlib gzip、进度页、整数最近邻与全屏；docs/research/wasm_release.md | 一键 dist、HTTP MIME／gzip／HEAD／Range 测试；最终 WASM 174610642 B、gzip 103256693 B；第 8 节 Chrome／Firefox／Edge 自查 |
| 浏览器用户数据 | 原字节 IndexedDB 后端实现；真实浏览器持久化待用户 | t5／t8 codec FUN_0042b030／0042b1e0／00420870／00429b60；t15 transaction.oncomplete、严格导入、不逐帧写入、正常 Quit 提交 | 假后端写入时机／重载／坏导入与 Node js 事务 abort 测试；真实原 scoreth10.dat／th10_05.rpy 导入字节相同；浏览器 reload／互通见用户自查 |
| 帧节拍（pacing） | 桌面原 Pacer 保持；浏览器固定 60 TPS，实机待用户 | FUN_00439390／DAT_00470c90 桌面实现不变；t15 js 无二次限速、音频／首帧门、逻辑 TPS 与 Draw FPS 分离 | 60／120／144 Hz 计数器与 js TPS 测试；Node 六章 trace 字节与两 RNG 相同；预热后逻辑 max 10.173 ms，冷峰值 24 ms 未收敛；F3 实机自查 |
```

建议更新记录：

```text
- 2026-11-08 — t15：先独立普查 WASM／原保存时机／Ebiten v2.10.5／Go1.27.3 工具链；新增一键 dist 与 loopback gzip／MIME 预览，640×480 整数最近邻、全屏、下载进度。js 用户数据换 IndexedDB，原 score／cfg／25 回放槽保留字节，拒绝坏导入，不逐帧写，正常 Quit 等待提交；桌面不变。启动先安装音频事件再零 TPS 等手势，空 Draw 边界后恰好一个首帧，再固定 60 TPS；F3 显示逻辑 TPS／Draw FPS／逐帧与窗口均值最大值。18 曲 Vorbis benchmark 后采用 200 ms BGM 源缓冲并首帧前预热标题曲，不改循环点／淡出。开发期 Node init-only DOM 支持原样 th10sim，无 WebGL／RunGame；六章 native＝base＝WASM 字节相同，两 RNG state/count 相同，十包 Node 测试通过。预热后逻辑最大 10.173 ms；冷 24 ms 峰值明确待决，不据均值称浏览器性能已验证。必需 full smoke／Go／WASM／gofmt 与六章回归见 t15 日志；native lint 与只读基线同为 32 项，js 自有范围 0 项。真实浏览器／原作互通／设备仍需用户。
```

## 8．需要用户目视确认的项目

没有浏览器实测。请用户在本机 PowerShell 7 中运行以下命令，然后手动打开网页：

```powershell
Set-Location -LiteralPath 'D:\GoProjects\th10\.tmp\codex\t15\tree'
$task = 'D:\GoProjects\th10\.tmp\codex\t15'
foreach ($dir in @("$task\gotmp", "$task\tmp", 'D:\GoProjects\th10\.tmp\codex\gocache')) {
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
}
$env:GOCACHE = 'D:\GoProjects\th10\.tmp\codex\gocache'
$env:GOTMPDIR = "$task\gotmp"
$env:TMP = "$task\tmp"
$env:TEMP = $env:TMP
go run ./tools/wasmbuild -serve :8080
```

浏览器打开 `http://127.0.0.1:8080/`。一条命令先构建再服务，裸 `:8080` 只绑定 loopback；已构建可用 `go run ./tools/wasmbuild -skip-build -serve :8080`。不能通过 file:// 打开页面。合并后从主树运行同一工具即可，勿在原作数据 junction 中执行工具。不要为不同测试轮次换端口，否则就是另一个存储 origin。

Chrome、Firefox、Edge **各做一遍**下表；Safari 有可用机器时再做。记录浏览器版本、显示刷新率、系统缩放／DPR，出现问题附控制台与 F3 层读数。

| 自查 | 每个浏览器具体看什么 |
|---|---|
| 下载与首帧 | 下载大 wasm 时有 MiB／百分比，不因 gzip 达到大于 100%；完成后停在“点击开始”，等十秒也不提前播放／推进 sig。控制台先有 Vorbis prewarm，真实点击或松开 Z 后才有 first game frame。sig 的 300 次真实 Draw 约五秒后标题曲出声；首次正常点击无需第二次才能正常进入游戏。 |
| 音频 | 开始后标题曲／关卡曲／Boss 曲／SE 有声音；Music Room 听至少一次 loop 接缝，关间／暂停恢复不爆音或突然重头。若等待 audio resume 不动，再次点击并查看 autoplay／AudioWorklet 错误；localhost 或 HTTPS 才能可靠用 worklet。不能把 audioReady=true 当实际扬声器有声证明。 |
| 输入与焦点 | 点击画布后方向移动、Z／X／Shift／Esc 正常；方向、Space 不滚页，F 不打开浏览器查找而切全屏，Ctrl+F／Ctrl+R 在游戏焦点内不触发默认操作。外围文件框／槽位选择仍可键盘操作。Alt+Tab、Ctrl+W、F11、Esc 退出全屏等保留快捷键无法保证取消。 |
| score 导入 | **点击开始前**先导出现有数据，再用文件选择框导入用户自己的原 `scoreth10.dat`；成功提示后 Player Data／排行／解锁来自该文件。不要运行 ScoreConverter 或原作做本任务自动检查。随后导入一份任意文本／截断 dat 必须拒绝；再次导出的存档 SHA-256 不应被这次坏导入改变。 |
| 回放导入／导出 | 选择 1–25 槽导入自己的 `.rpy`，成功后 Replay 菜单可选章播放；重新载入仍有槽。空槽导出应提示错误，坏 rpy 导入不能覆盖原槽；导出文件名为 th10_NN.rpy、原字节不重编码。原作读回复刻文件需用户在自己的可写原作目录中操作，仍不能声称自动互通已验证。 |
| 保存与 cfg 重载 | 进入新局／完成原保存检查点，等页面或控制台 `[TH10 storage] write ... committed`；Option 改音量后经其 Quit 提交 cfg。重新载入同一浏览器／URL／端口后数据与音量仍在。游戏内 Quit 后应提示已正常提交 score／cfg。刷新／强杀不能保证未完成事务，勿关闭“解析／提交”中的页面。 |
| 全屏与缩放 | 按按钮或 F 进入／退出全屏，640×480 比例不变、像素最近邻，不被拉伸；调整窗口显示 1×／2×／3×…整数倍。窄于 640 或矮于 480 的窗口保留 1×并允许外页滚动，这是不作分数缩小的明确策略。DPR／系统缩放下的清晰度需目视。 |
| TPS／耗时 | F3 或调试按钮开启；在 60、120、144 Hz 设置下前台稳定三秒后逻辑 TPS 应约 60，FPS 是 Draw／显示回调数，可为 120／144而非60。Update／Draw 显示 last／mean／max ms；首次 MSG／符卡名若 max 超 16.7，请记录帧段／控制台。耗时不含叠加层自身与 GPU 完成时间，不能只看均值。切换标签页应停止真实 Update／音频，回来等三秒重新看稳定读数。 |
| 手柄／持久保留 | 手柄仍走 Ebiten／原 cfg 绑定，实机确认方向阈值与 Z／X／低速／暂停。可点“请求保留网站数据”；浏览器拒绝保留不等于当前 IndexedDB 不可用，但清除网站数据仍会删存档。尽量只开一个活动页签，定期导出备份。 |

控制台顶层页面可执行以下只读诊断（运行时未退出时）：

```javascript
document.getElementById("game").contentWindow.th10.diagnostics()
```

关注 `started`、`audioReady`、`configuredTPS`（运行后 60、门内 0）、`focused`、`bgmBufferMs`（200）、TPS／FPS／毫秒、`goHeapBytes` 与 `linearMemoryBytes`。也可在开发者工具选择 game.html iframe 后直接调用 `th10.diagnostics()`。Chrome／Edge 的 Application、Firefox 的 Storage 中可看 IndexedDB `th10`／`files` 的 score.dat、th10.cfg 与 replay/th10_NN.rpy；不要直接编辑生产存档数据。

可复跑 Node 验证，保持按章顺序：

```powershell
pwsh -NoProfile -File tools/wasmcheck.ps1 -Baseline 'D:\GoProjects\th10\.tmp\trace\d1c2' -Out 'D:\GoProjects\th10\.tmp\codex\t15\user_node'
```

脚本只输出任务目录、成功后清理自己生成的大 trace；失败保留首差日志。依赖本机 Go、Node，不需要 Bash、浏览器、npm 依赖或原作程序。音频测量完整命令已实际使用：`node --stack-size=8192 tools/wasmexec/run.cjs <wasmperf.wasm> -kind audio -blocks 32 -out <audio.json>`；逻辑冷／预热分别加 `-kind logic -replay <file> -chapter N -frames 0` 与 `-warmup 1`，并设置 `TMPDIR`／`TH10_WASM_HEADLESS=1`。

## 9．与其他任务的接口

- t14：打包工具只构建 `./cmd`，复制构建工具链的 wasm glue，不枚举／解读 ANM、BGM 或字体资源；体积以本工作树为准。
- t16：不改变 scene 的标题／加载／保存控制器。浏览器门需挡住真实 Update／Draw，以保留 sig 的 300 次 Draw。
- t17：没有改 `tools/th10sim`、`tools/th10diff` 或 `trace/`。仅新增 `sim/profile.go`，用现有 private initializer／RestoreReplayChapter／Prime／h.step 测量并采最后状态，Run 原代码不变；Node facade 满足包初始化，不初始化图形，既有工具可原样运行。
- t5／t8：ScoreStore／ReplayStore／原 cfg 与 codec 签名不变。已查明原 score writer 仅由关卡释放 `0x417c8d`、全局释放 `0x41fc6e` 直接调用；既有新局、Game Over、Clear、音乐、排行检查点更早写宿主数据。本任务保持既有离散调用点，不越界搬动 scene；cfg Option Quit 保存也是现有宿主适配，正常游戏内 Quit 将另对接原退出保存。不能称原 writer 的逐指令磁盘时序完全相同。

新增可复用接口：`Game.ImportScore([]byte) error`、`ImportReplay(int, []byte) error`、`ExportScore() ([]byte,error)`、`ExportReplay(int) ([]byte,error)`；导入必须先校验后提交，不重编码用户原文件。score 导入成功后在首帧前调用既有 `LoadProgress()` 重绑历史，活动局不允许导入。`browserstore.Store` 是 Read／Write 的纯接口；js 实现等待事务完成，桌面适配测试不触及平台后端。`Game.InitBrowserStorage() error` 与 `StartBrowserClock()` 为 js 入口 hook，后者只清宿主 pacing／FPS 时间，不改变原 RNG、场景或 sig DrawCount。

t14 合并点：`(*audio.Manager).SetBGMBufferSize(time.Duration)`／`BGMBufferSize() time.Duration`；`audio.go` 仅增加字段与 `NewPlayer` 后、`Play` 前的可选 SetBufferSize。t14 若改 playBGM／decoder，请保留这三处小 hook，不整文件覆盖；`NewBGMStream` 原签名与资源读取路径没有改。

t16 合并点：`game.New` 的 SetTPS 与 `Game.Update` 的 Ready 条件仅改为平台函数分派；桌面实现仍为原 SyncWithFPS／Pacer。sig、标题、scene 生命周期与 ANM 没有修改，浏览器门与调试画面都在 cmd 包装器。

正常游戏内 Quit（RunGame 返回 nil）提交 cfg／score，分别对应 FUN_00438ad0 正常退出、FUN_0041fb50；不在刷新、hidden 或 beforeunload 增加不可靠异步保存。scene 保存调用与关内更新顺序均未改。

新只读测量接口：`(*sim.Runner).Measure(ch replay.Chapter, maxFrames int, memory func() uint64) (sim.Profile, error)`。memory 仅由工具读取 WASM exported mem 的 buffer 长度，nil 时不调用；方法不调用 Recorder／OnFrame、不格式化 trace、不触及原更新函数。Profile 报告两 RNG 与最终 trace 样本，单元测试及六章 native／Node 核对证明没有换逻辑路径。t17 若调整 private headless initializer／step，此方法自然复用；若重排 Run 的 restore／stop 时机，请一起复核 Measure 的边界。

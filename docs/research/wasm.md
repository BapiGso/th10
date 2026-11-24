# WASM 发布

WASM 是本项目的发布形态；桌面版同源构建。本文合并了第三轮 t15 的运行时普查（原 `wasm_release.md`）中仍然成立的事实与第五轮体积记录（原 `wasm_size_r5.md`）。旧文全文：`git show b4df88d:docs/research/wasm_release.md`。

## 0．现状

| 构建 | 说明 | 体积 |
|---|---|---|
| 发布版 | `go run ./tools/wasmbuild`，build tag `th10web`，资源走 `dist/assets/*.pak` 渐进加载 | th10.wasm 17.89 MiB／gzip 4.35 MiB；首次加载（wasm＋boot 包）约 8.6 MiB gzip；全部 27 个资源包 37.5 MB |
| 默认 js 构建 | `GOOS=js GOARCH=wasm go build ./cmd`（th10sim wasm、wasmcheck 也是这种），全部资源嵌入 | 56,075,259 B／gzip -9 39,186,507 B（第五轮测量） |

js 构建使用有损贴图（`assets/textures_js.go`，`th10extract -web-textures` 生成，可见 PSNR≥34、全部≥32、边缘≥36 dB，alpha 原字节不变），桌面使用无损贴图包。BGM 为 thbgm.dat 原 PCM 区间编码的 Ogg Vorbis（q1），循环点不变。

## 1．浏览器运行时

依据 `cmd/runtime_js.go`、`game/clock_js.go`、`cmd/finalscreen_js.go`；Ebiten v2.10.5／Oto v3.5.1。

- **首帧门**：先运行一个不调用真实场景 Update／Draw 的包装器，等用户真实点击／松键且 `audio.Context.IsReady()` 后才跑第一帧真实游戏。sig 的 300 次真实 Draw 门（`scene/loading`）因此不会在门外推进；备用 RNG（`FUN_004201b0` 的 `0x4201d0…0x4201e1`）只在 `game.New` 播种一次，等待期间不重播。
- **时钟**：开始后 `SetTPS(60)` 并 `SetRunnableOnUnfocused(false)`，不再经桌面 `pacing.Pacer` 二次筛选；切到后台停 Update 与音频，回来不补帧。桌面仍为 `SyncWithFPS`＋Pacer。
- **音频**：BGM source buffer 200 ms（`Manager.SetBGMBufferSize`），首帧前预热标题曲少量 PCM，不提前 Play、不常驻整曲 PCM，循环点与淡出帧数不变。Vorbis 解码与 mixer 仍在 WASM 主线程。
- **画面**：Ebiten 自建 canvas；页面用同源 iframe 承载，CSS 尺寸取 640×480 整数倍，`FinalScreenDrawer` 强制 `FilterNearest`。小于 640×480 时保持 1 倍并允许滚动。F 切全屏、F3 切调试层；浏览器／系统保留快捷键（Alt+Tab、Ctrl+W、F11 等）不能保证拦截。
- **诊断**：运行中在 game iframe 调 `th10.diagnostics()`，返回 started、audioReady、configuredTPS、focused、bgmBufferMs、TPS／FPS／毫秒与内存读数。

## 2．持久化

- 后端为 IndexedDB（`browserstore`）：数据库 `th10`、版本 1、对象仓 `files`；键 `score.dat`、`th10.cfg`、`replay/th10_NN.rpy`。写入等待 `transaction.oncomplete` 才报告成功，失败返回错误，不降级成会话内存。
- 文件字节与桌面相同：score 为原 `scoreth10.dat` 容器（`FUN_0042b030` 读／`FUN_0042b1e0` 写），cfg 为原 52 字节，回放为原 .rpy。页面提供导入／导出。
- 写入时机：原作 score writer 只在关卡释放 `FUN_00417c80`（`0x417c8d`）与全局释放 `FUN_0041fb50`（`0x41fc6e`）调用；cfg 在 `FUN_00438ad0` 正常退出时写。项目沿用新局、Game Over、Clear、解锁、保存菜单等离散宿主检查点，比原 writer 早，不是逐指令相同的磁盘时序；游戏内 Quit 会等待 score／cfg 提交。刷新、崩溃或强杀不保证未完成事务。
- 不同浏览器、origin、端口与隐私模式各有独立存储；清除网站数据会删除存档，导出是必要备份。

## 3．Node 执行器与等价验证

- `tools/wasmexec/run.cjs`：开发期 Node 执行器，`node --stack-size=8192 tools/wasmexec/run.cjs <binary.wasm> <args>`；需设置 `TMPDIR` 或 `GOTMPDIR`。`TH10_WASM_HEADLESS=1` 加载 `headless.cjs`（只够 Ebiten 包初始化的 DOM 外观，不调度帧、不初始化 WebGL），`TH10_WASM_FAKE_IDB=1` 加载假 IndexedDB，`TH10_WASM_ASSET_DIR` 把相对 URL 的 fetch 指向已构建的 dist（th10web 资源包测试）。
- `tools/wasmexec/compare.cjs`：两份 trace 的流式逐字节比较，打印首个差异帧／行。
- `tools/wasmcheck.ps1` 串起六章原生＝Node（＝基线）比较、RNG 状态核对与 Node 下的包测试，见 `docs/harness.md`“WASM 等价”。
- `wasmbuild` 复制**本次构建所用 Go** 的 `GOROOT` 下的 `wasm_exec.js`，不混用胶水与 wasm 的 Go 版本。
- 第三轮实测：六章原生＝基线＝Node 逐字节相同，主／备用 RNG 最终状态与计数相同。冷逻辑单帧有约 24 ms 的孤立峰值（来源未定位），一次预热后六章最大 3.4–10.2 ms；各关加载后 Go HeapAlloc 16.2–19.4 MB。浏览器 GPU／音频帧耗时仍需用户以 F3 实测。

## 4．第五轮体积：结局 CG 有损化与运行时依赖裁剪

日期：2026-11-21。基线为提交 76f4732（第四轮整合）。命令：`GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" ./cmd`；gzip 为 `gzip -9` 实测。

| 构建 | 原始字节 | gzip -9 字节 |
|---|---:|---:|
| 76f4732 | 66,746,466 | 43,554,681 |
| 第五轮 | 56,075,259 | 39,186,507 |
| 差 | −10,671,207 | −4,368,174 |

### 4.1 结局 CG 走有损编码

- 原因：`tools/th10extract/webtextures.go:chooseWebTexture` 对唯一颜色 ≤4096 的贴图直接保留无损。结局 CG 是带抖动的 RGB565（约 500 色），颜色少却压不动（flate 后仍约 186 KB／张），因此被误判。
- 改动：颜色数快捷判断只在无损压缩率 &lt; 0.2 时生效，其余交给原有 JPEG／量化候选与原质量门槛（可见 PSNR≥34、全部≥32、边缘≥36，均不变）。
- 结果：web.tpk 11,976,760 → 9,422,989 字节；结局类 5,198,001 → 2,767,423 字节，背景类也略降。桌面 lossless.tpk 与全部 ANP1 逻辑文件逐字节不变；生成结果可重复（两次运行 web.tpk 相同）。
- 对照图（误差最大的 20 张）：`.tmp/size_r5/compare/`。
- 未做：大多数结局 CG 停在 JPEG q92 才过 34 dB；若对结局类单独放宽门槛（如 30 dB），估计再省约 1–1.5 MB，需用户目视后决定。

### 4.2 运行时依赖裁剪

js 构建的 `./cmd` 依赖中已不再包含：ebitenui（及 bbcode、x/exp）、ebiten text/v2 与 go-text/typesetting（含 harfbuzz）、oksvg／rasterx、x/image/draw／tiff、ebitenutil 及其带入的 net/http、crypto/tls、x509。

- `scene/result`：原用 ebitenui 和操作系统字体（运行时读系统字体文件，违反资源规则）。它只在调用方未提供原生后续场景时作为宿主后备出现，原作无对应界面。改为 `render.RasterText`（内嵌 W3）一次绘制成整页图片，行内容不变。新增 `TestRenderDrawsTextOnPanel`。其后第五轮清理已把整个 `scene/result` 后备场景删除。
- 删除 `scene/ui/ui.go`（Theme／NewRoot／Label，仅 result 使用）与 `render/text.go`（text/v2 字体加载，仅 result 与测试使用）。字体子集测试所需的 text/v2 加载函数移到 `render/textv2_helpers_test.go`，测试本身不改。
- `cmd/runtime.go`、`game/game.go` 的调试叠加层改用新的 `render.DebugPrintAt`（x/image basicfont 7x13，字符串不变时复用图片）。仅调试显示，字形与 ebitenutil 不同。
- `go.mod`：删除 ebitenui、bbcode、x/exp；go-text/typesetting 仍被 `tools/th10extract` 与测试使用，改为直接依赖。

### 4.3 验收

`go build ./...`、`go vet ./...`、`go test -count=1 ./...` 全通过；`gofmt -l` 为空；非 tools 包 WASM 构建通过；Node 下 `gamedata`、`anm` 的 js 测试通过；`TH10_SMOKE=full` 的 `TestReplayDrawSmoke` 通过（130.9 秒）；th10_05 六章 trace 与 `base_cN` 逐字节相同。未做浏览器实测。

### 4.4 剩余可选项

- 字体 W7（797 KB）无运行时调用者，但现有像素参考测试覆盖 Bold 路径；删除需同时改测试。
- x/text 日文／GBK 解码表约 0.3 MB（桌面代码），可换成按固定语料生成的子集表；回放名等用户数据需保留后备。
- 发布工具改用 brotli 预压缩、加 wasm-opt，属于传输层。

## 5．js 发布版渐进加载资源（第五轮）

只有 wasmbuild 產出的發佈版（build tag `th10web`）不嵌資源。桌面版和預設 `GOOS=js` 構建（th10sim wasm、wasmcheck）照舊全嵌，邏輯與 trace 不受影響。

- `assets.ReadFile` / `assets.ReadTexture`：嵌入版在 `assets_embed.go`（`!th10web`）；網頁版在 `assets_web.go`（`js && th10web`）。gamedata 只走這兩個入口。
- 包格式和分包規則在 `assets/pack.go`，無 build tag，運行時和 wasmbuild 共用。
  - 包格式：`TAP1` + 條目，每條是 文件或貼圖 / 名字 / 數據。
  - `PackOf`：boot 是 sig、ascii、title、title_v、text、bullet 的 ANM，加全部 SE、misc、thbgm.fmt 和標題曲；stageNN 是該面的 ANM、ECL、STD、MSG；ending 是 e*/staff；每首 BGM 單獨一包；其餘歸 common。
  - `PackOrder`：common → 依通關順序每面的 stage 包、道中曲、Boss 曲 → ending → Extra。
- 運行時（cmd/runtime_js.go 的 `preparePlatform`）：
  - 先 fetch `assets/manifest.json` 和 boot 包（核對大小與 SHA-256，失敗重試 3 次），再建遊戲。
  - 其餘包在後台按 PackOrder 預取。
  - 讀到未到齊的包時，遊戲線程阻塞等待，並在 console 打印 `… waited N ms for asset pack X`。
  - 進度通過 `th10AssetProgress` 傳到頁面的「资源包」一行。
- wasmbuild：
  - 構建時帶 `-tags=th10web`。
  - 產出 `dist/assets/<pack>.<sha16>.pak` 和 `manifest.json`；gzip 省不到 5% 的包（Ogg）不發 .gz。
  - 預覽服務器只放行 `assets/manifest.json` 和 `assets/*.pak`；pak 帶 `immutable` 長緩存。

| 項目 | 之前（全嵌） | 現在 |
|---|---|---|
| th10.wasm | 56.08 MB / gzip 39.19 MB | 17.89 MiB / gzip 4.35 MiB |
| 首次加載（wasm + boot） | gzip 39.19 MB | 4.35 + 4.20 = **約 8.6 MiB gzip** |
| 全部資源包 | — | 27 包，37.5 MB（多數是已壓縮的 Ogg） |

- 跨包重複的貼圖約 0.33 MB，主要是 stage07 複用前幾面的貼圖。按設計允許重複。
- boot 包 4.9 MB 裡：標題曲 0.8 MB，SE 約 2 MB（音頻管理器創建時預載全部 SE，保持原行為），其餘是標題/子彈貼圖。

驗收：
- rules §6 全套（build、vet、test 42 包、gofmt、non-tools WASM、full smoke）全綠。
- 六章 trace 全部 SAME。
- `GOOS=js go build -tags th10web ./cmd` 和 vet 通過。
- 新增測試：
  - `assets/pack_test.go`：格式往返和截斷、BGM 表對照 thbgm.fmt、分包規則。
  - `tools/wasmbuild/packs_test.go`：每個嵌入文件逐字節進唯一一包，manifest 哈希正確，預覽服務器路徑白名單和緩存頭正確。
  - `assets/assets_web_test.go`（th10web 版，Node 下用 `TH10_WASM_ASSET_DIR` 偽 fetch）：202 個文件經真實 fetch 路徑讀回一致，全部 ANM 都能還原。
- 無頭 Chrome 實測：boot 包到齊，Vorbis 預熱完成，runtime-ready；其餘 26 包隨後在後台全部到齊，無錯誤。

## 音效播放卡頓（2026-11-23）

- 現象：WASM、144 Hz，Boss／中 Boss 等音效密集場景卡頓。
- 原因：每次 SE 播放都新建一個 Ebiten `audio.Player`。oto 的 mux 會為新播放器預讀最多 0.5 s PCM，混音也在 Go 裡做；WASM 只有一條主線程，這些工作都擠在 rAF 之間，把渲染擠掉了。邏輯（Update 約 0.6–2 ms）、draw 數量（每幀 16–22 次 submit）和 GC 都不是主因。
- 修法：每個 SE 表 id 一個常駐 voice，對應原作每 id 一個 DirectSound 緩衝。重播只回繞，不分配。
  - js（`audio/seweb_js.go`）：交給 Web Audio，在瀏覽器的音頻線程混音。每 id 一張固定圖（splitter → 左右 gain → merger → master），重播只是換一個 BufferSource。第一次播放時建 AudioContext 並上傳全部 wav；Safari 要在手勢裡 resume，所以掛了 keydown／pointerdown／touchend 監聽。
  - 桌面（`audio/semixer.go`）：在 Go 裡把全部 voice 混進一個常駐 Ebiten 播放器，緩衝 40 ms。
  - 排隊、聲像平均、原生音量曲線不變；BGM 不變（網頁 200 ms 緩衝）。
- Chrome 實測（th10_05 第 3 章，10000 幀上限，SyncWithFPS）：

| | 卡頓幀（間隔 >20 ms 或 Update+Draw >10 ms） | 幀間隔 p95 / p99 |
|---|---|---|
| 修前 | 371 | 18.1 ms / — |
| 修後 | 9 | 8.2 / 11.6 ms |
| 修後、靜音 SE | 8 | 8.4 / 11.9 ms |

- 剩下的少量尖峰來自關卡開始時懶建貼圖，和音效無關，未處理。144 Hz 下固定 60 TPS 的 2/3 刷新節奏不均是另一回事，不在此修。

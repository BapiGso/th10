# WASM 体积第五轮：结局 CG 有损化与运行时依赖裁剪

日期：2026-11-21。基线为提交 76f4732（第四轮整合）。命令：`GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" ./cmd`；gzip 为 `gzip -9` 实测。

| 构建 | 原始字节 | gzip -9 字节 |
|---|---:|---:|
| 76f4732 | 66,746,466 | 43,554,681 |
| 本轮 | 56,075,259 | 39,186,507 |
| 差 | −10,671,207 | −4,368,174 |

## 1．结局 CG 走有损编码

- 原因：`tools/th10extract/webtextures.go:chooseWebTexture` 对唯一颜色 ≤4096 的贴图直接保留无损。结局 CG 是带抖动的 RGB565（约 500 色），颜色少却压不动（flate 后仍约 186 KB／张），因此被误判。
- 改动：颜色数快捷判断只在无损压缩率 &lt; 0.2 时生效，其余交给原有 JPEG／量化候选与原质量门槛（可见 PSNR≥34、全部≥32、边缘≥36，均不变）。
- 结果：web.tpk 11,976,760 → 9,422,989 字节；结局类 5,198,001 → 2,767,423 字节，背景类也略降。桌面 lossless.tpk 与全部 ANP1 逻辑文件逐字节不变；生成结果可重复（两次运行 web.tpk 相同）。
- 对照图（误差最大的 20 张）：`.tmp/size_r5/compare/`。
- 未做：大多数结局 CG 停在 JPEG q92 才过 34 dB；若对结局类单独放宽门槛（如 30 dB），估计再省约 1–1.5 MB，需用户目视后决定。

## 2．运行时依赖裁剪

js 构建的 `./cmd` 依赖中已不再包含：ebitenui（及 bbcode、x/exp）、ebiten text/v2 与 go-text/typesetting（含 harfbuzz）、oksvg／rasterx、x/image/draw／tiff、ebitenutil 及其带入的 net/http、crypto/tls、x509。

- `scene/result`：原用 ebitenui 和操作系统字体（运行时读系统字体文件，违反资源规则）。它只在调用方未提供原生后续场景时作为宿主后备出现，原作无对应界面。改为 `render.RasterText`（内嵌 W3）一次绘制成整页图片，行内容不变。新增 `TestRenderDrawsTextOnPanel`。
- 删除 `scene/ui/ui.go`（Theme／NewRoot／Label，仅 result 使用）与 `render/text.go`（text/v2 字体加载，仅 result 与测试使用）。字体子集测试所需的 text/v2 加载函数移到 `render/textv2_helpers_test.go`，测试本身不改。
- `cmd/runtime.go`、`game/game.go` 的调试叠加层改用新的 `render.DebugPrintAt`（x/image basicfont 7x13，字符串不变时复用图片）。仅调试显示，字形与 ebitenutil 不同。
- `go.mod`：删除 ebitenui、bbcode、x/exp；go-text/typesetting 仍被 `tools/th10extract` 与测试使用，改为直接依赖。

## 3．验收

`go build ./...`、`go vet ./...`、`go test -count=1 ./...` 全通过；`gofmt -l` 为空；非 tools 包 WASM 构建通过；Node 下 `gamedata`、`anm` 的 js 测试通过；`TH10_SMOKE=full` 的 `TestReplayDrawSmoke` 通过（130.9 秒）；th10_05 六章 trace 与 `base_cN` 逐字节相同。未做浏览器实测。

## 4．剩余可选项

- 字体 W7（797 KB）无运行时调用者，但现有像素参考测试覆盖 Bold 路径；删除需同时改测试。
- x/text 日文／GBK 解码表约 0.3 MB（桌面代码），可换成按固定语料生成的子集表；回放名等用户数据需保留后备。
- 发布工具改用 brotli 预压缩、加 wasm-opt，属于传输层。

## 第 5 項：js 發佈版漸進加載資源（2026-11-21）

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

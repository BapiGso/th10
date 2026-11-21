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

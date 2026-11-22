# 东方风神录 th10.exe 移植（Go + Ebiten）

把「东方风神录 ～ Mountain of Faith.」v1.00a 的 `th10.exe` 逐函数移植到 Go + Ebiten。目标是与原作逐帧一致：逻辑按反编译／反汇编照搬，整数与 RNG 位精确，浮点按 x87 24 位精度用 float32 分步取整；代码注释引用原函数地址（如 `FUN_0040dc80`）与数据地址（如 `DAT_00477818`）。资源全部来自原作文件，运行时不读安装目录。

发布形态是浏览器 WASM，桌面版同源构建。各子系统的还原程度见 `docs/fidelity.md`。

## 运行（桌面）

```bash
go run ./cmd                                         # 正常启动：sig → 标题
go run ./cmd -debug-stage 1                          # 直接进入第 1 面（1–6 或 extra），灵梦 A
go run ./cmd -debug-stage 6 -debug-diff lunatic -debug-power 400
```

- `-debug-stage`：`1`–`6` 或 `extra`；不给时走正常流程。
- `-debug-diff`：`easy`／`normal`（默认）／`hard`／`lunatic`／`extra`；Extra 关总是 Extra 难度。
- `-debug-power`：开局火力，0–500（百分之一单位，500＝5.00）。
- 存档（`score.dat` 即原 `scoreth10.dat` 格式、`th10.cfg`、`replay/th10_NN.rpy`）写在用户数据目录；用 `TH10_DATA_DIR` 指定其他目录。

## WASM 发布

```bash
go run ./tools/wasmbuild                 # 构建到 dist/
go run ./tools/wasmbuild -serve :8080    # 构建并在 127.0.0.1:8080 预览
go run ./tools/wasmbuild -skip-build -serve :8080
```

- 发布版带 build tag `th10web`：wasm 不嵌资源，资源按 `assets/pack.go` 的规则分成 `dist/assets/<pack>.<sha16>.pak`，先取 `manifest.json` 与 boot 包，其余后台预取。
- 默认 `GOOS=js GOARCH=wasm go build`（不带 `th10web`）仍把全部资源嵌入，供 Node 下的 `th10sim`、`wasmcheck` 使用。
- js 构建使用有损贴图包，桌面使用无损贴图包；浏览器存档在 IndexedDB。体积、运行时与持久化细节见 `docs/research/wasm.md`。

## 资源

- `gamedata` 是唯一资源入口，底层为 `assets` 的 `ReadFile`／`ReadTexture`（嵌入版 `assets_embed.go`，网页版 `assets_web.go`）。
- `assets/` 由开发工具 `tools/th10extract` 从你自己的 v1.00a 安装生成：ANM／ECL／STD／MSG／SHT 等保持原字节，ANM 像素拆成去重贴图包，BGM 为 thbgm.dat 原 PCM 区间编码的 Ogg Vorbis（循环点不变）。
- 安装目录：`-dir` → 环境变量 `TH10_DIR` → 工作目录向上查找 `.tmp/[th10]/`。

```bash
go run ./tools/th10extract -list                        # 列出 th10.dat，标出已嵌入的文件
go run ./tools/th10extract stage01.std 'se_*.wav'      # 原样复制（path.Match 通配符）
go run ./tools/th10extract -archive th10c.dat e00.mst  # 汉化包中的演出文字
go run ./tools/th10extract -web-textures               # 重建贴图包（-pack-textures 只做无损桌面包）
go run ./tools/th10extract -pack-se                    # SE 无损压缩
go run ./tools/th10extract -fonts -font-source DIR     # 按固定文本裁剪原字体
go run ./tools/th10extract -bgm                        # 编码 BGM（需带 libvorbis 的 ffmpeg）
```

`go test ./gamedata ./audio` 在找得到原作数据时逐字节核对嵌入资源与 BGM 循环点，找不到时跳过。

## 验证

- `go test ./...`：单元与规格测试、ECL golden、无头回放；`TH10_SMOKE=full` 时跑完整绘制 smoke。
- 差分闭环：`tools/th10sim` 用回放无头生成 trace，`tools/th10probe` 只读采集原作进程，`tools/th10diff` 找首个分歧帧。用法与采集步骤见 `docs/harness.md`。
- 基准：官方回放 th10_05 六章的 trace 与 `base_c0`–`base_c5` 逐字节相同，任何改动不得改变它（除非是有依据的修正并更新基线）。
- `tools/wasmcheck.ps1`：六章原生＝Node 中 wasm 逐字节比较及 js 包测试（`docs/harness.md`“WASM 等价”）。

## 代码地图

| 路径 | 内容 |
|---|---|
| `cmd/` | 入口；桌面与 js 平台运行时 |
| `game/` | 主循环、游戏状态、模式、存档／配置／回放持久化 |
| `supervisor/`、`gfx/`、`render/` | 绘制链与 D3D9 固定管线模拟、文字光栅化 |
| `anm/`、`anm/anmdraw/` | ANM 容器、VM、World 层与各渲染模式 |
| `std/`、`std/stddraw/` | 关卡背景脚本、相机与 3D 绘制 |
| `scene/stage/`、`scene/stage/ecl/`、`scene/stage/script/` | 关卡：ECL VM、GUI、符卡、暂停／关卡内菜单；七关参数统一在 `script` 的配置表 |
| `entity/` | 敌机、敌弹、激光、道具、自机、自机子弹 |
| `scene/` 其余 | loading、title、select、ui（标题演出与 Option）、dialog（MSG）、ending（结局／Staff）、menus（Music Room／Player Data／回放） |
| `sht/`、`zun/`、`input/`、`pacing/`、`audio/` | SHT 数据、角度／计时器／RNG、输入、帧节拍、SE／BGM |
| `replay/`、`sim/`、`trace/` | .rpy 编解码、无头运行器、trace 格式 |
| `datfs/`、`assets/`、`gamedata/`、`browserstore/` | th10.dat 读取、嵌入资源与资源包、资源入口、IndexedDB |
| `tools/` | th10extract、th10sim、th10probe（Windows）、th10diff、th10scenes（关外场景查看）、wasmbuild、wasmperf、wasmexec、wasmcheck.ps1、ghidra |

## 文档

- `docs/fidelity.md`：真值台账，每个子系统的状态、依据与验收。
- `docs/open_items.md`：待 dump／目视／采集／审阅的事项。
- `docs/harness.md`：差分闭环、采集步骤、精度标准、WASM 等价。
- `docs/ecl_truth.md`、`docs/player_truth.md`：ECL 与自机的原作事实与边界。
- `docs/research/`：各轮取证（m1／m2／m4_*、faith、replays、scenes_a／b、probe_ext、th10c_dat、wasm），以及资源清单 JSON。
- `docs/study/`：弹幕游戏制作参考文章；`docs/research/ecl_dump/`：ECL 反编译摘录。

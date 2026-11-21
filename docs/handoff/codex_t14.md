# t14 资源瘦身交付记录

更新时间：2026-11-08。工作树：`D:/GoProjects/th10/.tmp/codex/t14/tree`。原作未运行，未使用 probe／Ghidra，没有新增依赖或修改主树源码。

## 1．摘要

- 阶段 A／B／C／D／E：实现及完整 rules 第 6 节验收均通过。独立普查先于压缩改动，见 `docs/research/assets_size.md`。
- 54 个 ANM／ANT 只分离 THTX 像素，非像素容器合计 790,168 字节；227 个源像素块去重后桌面包 16,223,947 字节，js 包 11,976,760 字节，67 块使用有损颜色。尺寸、格式、精灵矩形、指令和时间线不变。
- 桌面解码后逐 THTX 原字节相同，包括 alpha＝0 下的 RGB；js 仅放宽颜色，alpha 始终无损。小图、文字／数字精灵、纯色和低成本渐变保持无损。
- BGM 只有一套：18 首 Vorbis q1，双声道／44,100 Hz，共 22,560,332 字节；原循环起终点逐采样保留。37 个 SE 无损压缩到 1,585,814 字节，加载时解码／重采样，播放时直接用缓存 PCM。
- 三种字体共 1,636,312 字节，原为 8,402,116 字节；固定字符集合为 1,939，加上保留的排版／复合依赖和 .notdef。没有改变文字绘制器，不声称它因此变成原 GDI 的 L3 真值。
- 非运行时 PNG、反汇编文本、笔记迁入 docs；runtime embed 使用明确的扩展名白名单。没有修改 ANM VM、绘制管线、关内逻辑、cmd／网页、trace／sim／diff。

### 发布体积

| 构建 | 原始字节 | gzip -9 字节 |
|---|---:|---:|
| WASM | 66,704,990 | 43,575,791 |
| Windows amd64 | 60,904,960 | 48,135,559 |

命令：`go build -trimpath -ldflags='-s -w' -o <out> ./cmd`；WASM 额外设置 `GOOS=js GOARCH=wasm`。Go 1.27.3，Node 26.11.1。gzip 由 Python 标准库 level 9、mtime＝0 实测，不把 gzip 后大小算成原始 wasm。

### 改动前后

| 类别 | 改动前磁盘资源字节 | 桌面嵌入字节 | js 嵌入字节 |
|---|---:|---:|---:|
| ANM／贴图 | 107,761,889 | 17,022,822 | 12,775,635 |
| ECL | 1,352,853 | 995,964 | 995,964 |
| 字体 | 8,402,116 | 1,636,312 | 1,636,312 |
| MSG | 98,044 | 98,044 | 98,044 |
| SHT | 16,496 | 16,496 | 16,496 |
| STD | 18,864 | 18,864 | 18,864 |
| Misc | 37,540 | 37,540 | 37,540 |
| BGM（含 fmt） | 38,843,756 | 22,561,285 | 22,561,285 |
| SE | 1,875,292 | 1,585,814 | 1,585,814 |
| Study（原不嵌入） | 127,636 | 0 | 0 |

改动前磁盘表含当时目录通配符误带入的资源；study 原来没有 embed，现也移为 docs。两个 TPK 文件都在仓库，但每种构建只嵌入其中一个，不在 WASM 中藏入桌面包。完整源块清单见 `docs/research/textures_manifest.json`，字体源哈希／glyph 哈希见 `docs/research/font_subset.json`。

### 当前 WASM 占比

| 类别 | 字节 | 占原始 WASM |
|---|---:|---:|
| ANM／贴图 | 12,775,635 | 19.15% |
| ECL | 995,964 | 1.49% |
| 字体 | 1,636,312 | 2.45% |
| MSG | 98,044 | 0.15% |
| SHT | 16,496 | 0.02% |
| STD | 18,864 | 0.03% |
| Misc | 37,540 | 0.06% |
| BGM（含 fmt） | 22,561,285 | 33.82% |
| SE | 1,585,814 | 2.38% |
| Go／Ebiten 代码、运行时数据与链接开销 | 26,979,036 | 40.45% |

另用 WASM section 解析器实测 code section 为 16,265,021 字节，data section 为 50,395,673 字节。资源表是 embed 的有效文件／像素包长度，差额含其他运行时数据、元信息和链接开销，不能把差额全叫纯代码。独立 wasm embed 探针确认 common_total＝27,749,194 字节，字体均为裁剪后大小，selected variant＝web；发布 wasm 中未包含桌面 lossless.tpk。

## 2．改动文件清单

### 新增

- `anm/packed.go`：标准库资源解包／质量度量，不改变 ANM VM 或精灵表。
- `anm/packed_test.go`：资源原字节、像素、质量、私有缓存、采样长度或确定性回归测试。
- `anm/quality.go`：标准库资源解包／质量度量，不改变 ANM VM 或精灵表。
- `anm/quality_test.go`：资源原字节、像素、质量、私有缓存、采样长度或确定性回归测试。
- `assets/anm/texture_index.json`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/textures/lossless.tpk`：生成按源像素哈希去重的 build-selected TPK1 贴图包。
- `assets/textures/web.tpk`：生成按源像素哈希去重的 build-selected TPK1 贴图包。
- `assets/textures_desktop.go`：只嵌入运行时文件；按 js 标签选择像素包。
- `assets/textures_js.go`：只嵌入运行时文件；按 js 标签选择像素包。
- `audio/preload.go`：提前加载全部 SE PCM，并保留既有 BGM／SE 播放语义。
- `audio/preload_test.go`：资源原字节、像素、质量、私有缓存、采样长度或确定性回归测试。
- `docs/handoff/codex_t14.md`：资源普查、生成清单、验证证据或本任务交付说明。
- `docs/reference/decoded_textures/ascii/ascii.png`：迁自 `assets/anm/ascii/ascii.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ascii/leaf.png`：迁自 `assets/anm/ascii/leaf.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ascii/leaf.svg`：迁自 `assets/anm/ascii/leaf.svg`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ascii/leaflite.svg`：迁自 `assets/anm/ascii/leaflite.svg`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ascii/loading.png`：迁自 `assets/anm/ascii/loading.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ascii/pause.png`：迁自 `assets/anm/ascii/pause.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg1bg.png`：迁自 `assets/anm/background/stg1bg.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg1bg2.png`：迁自 `assets/anm/background/stg1bg2.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg1bg3.png`：迁自 `assets/anm/background/stg1bg3.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg1bg4.png`：迁自 `assets/anm/background/stg1bg4.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg2bg.png`：迁自 `assets/anm/background/stg2bg.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg2bg2.png`：迁自 `assets/anm/background/stg2bg2.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg3bg.png`：迁自 `assets/anm/background/stg3bg.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg3bg2.png`：迁自 `assets/anm/background/stg3bg2.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg3bg3.png`：迁自 `assets/anm/background/stg3bg3.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg3bg4.png`：迁自 `assets/anm/background/stg3bg4.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg4bg.png`：迁自 `assets/anm/background/stg4bg.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg4bg3.png`：迁自 `assets/anm/background/stg4bg3.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg4bg7.png`：迁自 `assets/anm/background/stg4bg7.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg5bg.png`：迁自 `assets/anm/background/stg5bg.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg5bg2.png`：迁自 `assets/anm/background/stg5bg2.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg6bg.png`：迁自 `assets/anm/background/stg6bg.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg6bg2.png`：迁自 `assets/anm/background/stg6bg2.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg6bg3.png`：迁自 `assets/anm/background/stg6bg3.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg6bg5.png`：迁自 `assets/anm/background/stg6bg5.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg6bg6.png`：迁自 `assets/anm/background/stg6bg6.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg7bg.png`：迁自 `assets/anm/background/stg7bg.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg7bg2.png`：迁自 `assets/anm/background/stg7bg2.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/background/stg7bg3.png`：迁自 `assets/anm/background/stg7bg3.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/bullet/etama.png`：迁自 `assets/anm/bullet/etama.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/bullet/etama2.png`：迁自 `assets/anm/bullet/etama2.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/bullet/etama3.png`：迁自 `assets/anm/bullet/etama3.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/bullet/etama6.png`：迁自 `assets/anm/bullet/etama6.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/card/cdbg01a.png`：迁自 `assets/anm/card/cdbg01a.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/card/cdbg01b.png`：迁自 `assets/anm/card/cdbg01b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/card/cdbg02a.png`：迁自 `assets/anm/card/cdbg02a.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/card/cdbg02b.png`：迁自 `assets/anm/card/cdbg02b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/card/cdbg03a.png`：迁自 `assets/anm/card/cdbg03a.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/card/cdbg03b.png`：迁自 `assets/anm/card/cdbg03b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/card/cdbg04a.png`：迁自 `assets/anm/card/cdbg04a.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/card/cdbg04b.png`：迁自 `assets/anm/card/cdbg04b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/card/cdbg05a.png`：迁自 `assets/anm/card/cdbg05a.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/card/cdbg05b.png`：迁自 `assets/anm/card/cdbg05b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/card/cdbg06a.png`：迁自 `assets/anm/card/cdbg06a.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/card/cdbg06b.png`：迁自 `assets/anm/card/cdbg06b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/card/cdbg07a.png`：迁自 `assets/anm/card/cdbg07a.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/card/cdbg07b.png`：迁自 `assets/anm/card/cdbg07b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e00a.png`：迁自 `assets/anm/ending/e00a.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e00ar.png`：迁自 `assets/anm/ending/e00ar.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e00b.png`：迁自 `assets/anm/ending/e00b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e00br.png`：迁自 `assets/anm/ending/e00br.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e01b.png`：迁自 `assets/anm/ending/e01b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e01br.png`：迁自 `assets/anm/ending/e01br.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e02b.png`：迁自 `assets/anm/ending/e02b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e02br.png`：迁自 `assets/anm/ending/e02br.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e03b.png`：迁自 `assets/anm/ending/e03b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e03br.png`：迁自 `assets/anm/ending/e03br.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e04b.png`：迁自 `assets/anm/ending/e04b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e04br.png`：迁自 `assets/anm/ending/e04br.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e05b.png`：迁自 `assets/anm/ending/e05b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e05br.png`：迁自 `assets/anm/ending/e05br.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e06b.png`：迁自 `assets/anm/ending/e06b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e06br.png`：迁自 `assets/anm/ending/e06br.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e06c.png`：迁自 `assets/anm/ending/e06c.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e06cr.png`：迁自 `assets/anm/ending/e06cr.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e07b.png`：迁自 `assets/anm/ending/e07b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e07br.png`：迁自 `assets/anm/ending/e07br.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e07c.png`：迁自 `assets/anm/ending/e07c.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e07cr.png`：迁自 `assets/anm/ending/e07cr.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e08b.png`：迁自 `assets/anm/ending/e08b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e08br.png`：迁自 `assets/anm/ending/e08br.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e08c.png`：迁自 `assets/anm/ending/e08c.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e08cr.png`：迁自 `assets/anm/ending/e08cr.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e09a.png`：迁自 `assets/anm/ending/e09a.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e09ar.png`：迁自 `assets/anm/ending/e09ar.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e09b.png`：迁自 `assets/anm/ending/e09b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e09br.png`：迁自 `assets/anm/ending/e09br.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e09c.png`：迁自 `assets/anm/ending/e09c.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e09cr.png`：迁自 `assets/anm/ending/e09cr.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e10a.png`：迁自 `assets/anm/ending/e10a.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e10ar.png`：迁自 `assets/anm/ending/e10ar.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e10b.png`：迁自 `assets/anm/ending/e10b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e10br.png`：迁自 `assets/anm/ending/e10br.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e10c.png`：迁自 `assets/anm/ending/e10c.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e10cr.png`：迁自 `assets/anm/ending/e10cr.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e11b.png`：迁自 `assets/anm/ending/e11b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e11br.png`：迁自 `assets/anm/ending/e11br.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e11c.png`：迁自 `assets/anm/ending/e11c.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/e11cr.png`：迁自 `assets/anm/ending/e11cr.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/st00.png`：迁自 `assets/anm/ending/st00.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/st01.png`：迁自 `assets/anm/ending/st01.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/st01r.png`：迁自 `assets/anm/ending/st01r.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/st02.png`：迁自 `assets/anm/ending/st02.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/st03.png`：迁自 `assets/anm/ending/st03.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/st03r.png`：迁自 `assets/anm/ending/st03r.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/st04.png`：迁自 `assets/anm/ending/st04.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/st05.png`：迁自 `assets/anm/ending/st05.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/st05r.png`：迁自 `assets/anm/ending/st05r.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/staff.png`：迁自 `assets/anm/ending/staff.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/ending/staff2.png`：迁自 `assets/anm/ending/staff2.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/enemy/enemy.png`：迁自 `assets/anm/enemy/enemy.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/dummy.png`：迁自 `assets/anm/face/dummy.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy1/ename01.png`：迁自 `assets/anm/face/enemy1/ename01.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy1/face01an_d.png`：迁自 `assets/anm/face/enemy1/face01an_d.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy1/face01an_u.png`：迁自 `assets/anm/face/enemy1/face01an_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy1/face01ct.png`：迁自 `assets/anm/face/enemy1/face01ct.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy1/face01n2_d.png`：迁自 `assets/anm/face/enemy1/face01n2_d.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy1/face01n2_u.png`：迁自 `assets/anm/face/enemy1/face01n2_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy1/face01no_d.png`：迁自 `assets/anm/face/enemy1/face01no_d.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy1/face01no_u.png`：迁自 `assets/anm/face/enemy1/face01no_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy1m/face01mct.png`：迁自 `assets/anm/face/enemy1m/face01mct.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy2/ename02.png`：迁自 `assets/anm/face/enemy2/ename02.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy2/face02an_u.png`：迁自 `assets/anm/face/enemy2/face02an_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy2/face02ct.png`：迁自 `assets/anm/face/enemy2/face02ct.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy2/face02lo_u.png`：迁自 `assets/anm/face/enemy2/face02lo_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy2/face02n2_u.png`：迁自 `assets/anm/face/enemy2/face02n2_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy2/face02no_d.png`：迁自 `assets/anm/face/enemy2/face02no_d.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy2/face02no_u.png`：迁自 `assets/anm/face/enemy2/face02no_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy2/face02pr_u.png`：迁自 `assets/anm/face/enemy2/face02pr_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy3/ename03.png`：迁自 `assets/anm/face/enemy3/ename03.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy3/face03an_u.png`：迁自 `assets/anm/face/enemy3/face03an_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy3/face03ct.png`：迁自 `assets/anm/face/enemy3/face03ct.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy3/face03lo_u.png`：迁自 `assets/anm/face/enemy3/face03lo_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy3/face03n2_u.png`：迁自 `assets/anm/face/enemy3/face03n2_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy3/face03no_d.png`：迁自 `assets/anm/face/enemy3/face03no_d.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy3/face03no_u.png`：迁自 `assets/anm/face/enemy3/face03no_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy3/face03sp_u.png`：迁自 `assets/anm/face/enemy3/face03sp_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy4/ename04.png`：迁自 `assets/anm/face/enemy4/ename04.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy4/face04ct.png`：迁自 `assets/anm/face/enemy4/face04ct.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy4/face04dp_u.png`：迁自 `assets/anm/face/enemy4/face04dp_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy4/face04lo_u.png`：迁自 `assets/anm/face/enemy4/face04lo_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy4/face04n2_u.png`：迁自 `assets/anm/face/enemy4/face04n2_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy4/face04no_d.png`：迁自 `assets/anm/face/enemy4/face04no_d.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy4/face04no_u.png`：迁自 `assets/anm/face/enemy4/face04no_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy4/face04pr_u.png`：迁自 `assets/anm/face/enemy4/face04pr_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy4/face04sp_u.png`：迁自 `assets/anm/face/enemy4/face04sp_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy5/ename05.png`：迁自 `assets/anm/face/enemy5/ename05.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy5/face05an_u.png`：迁自 `assets/anm/face/enemy5/face05an_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy5/face05ct.png`：迁自 `assets/anm/face/enemy5/face05ct.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy5/face05dp_u.png`：迁自 `assets/anm/face/enemy5/face05dp_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy5/face05lo_u.png`：迁自 `assets/anm/face/enemy5/face05lo_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy5/face05n2_u.png`：迁自 `assets/anm/face/enemy5/face05n2_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy5/face05no_d.png`：迁自 `assets/anm/face/enemy5/face05no_d.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy5/face05no_u.png`：迁自 `assets/anm/face/enemy5/face05no_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy5/face05sp_u.png`：迁自 `assets/anm/face/enemy5/face05sp_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy5/face05sw_u.png`：迁自 `assets/anm/face/enemy5/face05sw_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy6/ename06.png`：迁自 `assets/anm/face/enemy6/ename06.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy6/face06ct.png`：迁自 `assets/anm/face/enemy6/face06ct.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy6/face06dp_u.png`：迁自 `assets/anm/face/enemy6/face06dp_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy6/face06n2_u.png`：迁自 `assets/anm/face/enemy6/face06n2_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy6/face06no_d.png`：迁自 `assets/anm/face/enemy6/face06no_d.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy6/face06no_u.png`：迁自 `assets/anm/face/enemy6/face06no_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy7/ename07.png`：迁自 `assets/anm/face/enemy7/ename07.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy7/face07ct.png`：迁自 `assets/anm/face/enemy7/face07ct.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy7/face07dp_u.png`：迁自 `assets/anm/face/enemy7/face07dp_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy7/face07lo_u.png`：迁自 `assets/anm/face/enemy7/face07lo_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy7/face07no_d.png`：迁自 `assets/anm/face/enemy7/face07no_d.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy7/face07no_u.png`：迁自 `assets/anm/face/enemy7/face07no_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/enemy7/face07pr_u.png`：迁自 `assets/anm/face/enemy7/face07pr_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/pl00/face_pl00an_u.png`：迁自 `assets/anm/face/pl00/face_pl00an_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/pl00/face_pl00dp_u.png`：迁自 `assets/anm/face/pl00/face_pl00dp_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/pl00/face_pl00hp_u.png`：迁自 `assets/anm/face/pl00/face_pl00hp_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/pl00/face_pl00n2_u.png`：迁自 `assets/anm/face/pl00/face_pl00n2_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/pl00/face_pl00no_d.png`：迁自 `assets/anm/face/pl00/face_pl00no_d.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/pl00/face_pl00no_u.png`：迁自 `assets/anm/face/pl00/face_pl00no_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/pl00/face_pl00pr_u.png`：迁自 `assets/anm/face/pl00/face_pl00pr_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/pl00/face_pl00sp_u.png`：迁自 `assets/anm/face/pl00/face_pl00sp_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/pl00/face_pl00sw_u.png`：迁自 `assets/anm/face/pl00/face_pl00sw_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/pl01/face_pl01an_u.png`：迁自 `assets/anm/face/pl01/face_pl01an_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/pl01/face_pl01dp_u.png`：迁自 `assets/anm/face/pl01/face_pl01dp_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/pl01/face_pl01hp_u.png`：迁自 `assets/anm/face/pl01/face_pl01hp_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/pl01/face_pl01n2_u.png`：迁自 `assets/anm/face/pl01/face_pl01n2_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/pl01/face_pl01no_d.png`：迁自 `assets/anm/face/pl01/face_pl01no_d.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/pl01/face_pl01no_u.png`：迁自 `assets/anm/face/pl01/face_pl01no_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/pl01/face_pl01pr_u.png`：迁自 `assets/anm/face/pl01/face_pl01pr_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/pl01/face_pl01sp_u.png`：迁自 `assets/anm/face/pl01/face_pl01sp_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/face/pl01/face_pl01sw_u.png`：迁自 `assets/anm/face/pl01/face_pl01sw_u.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/front/ename.png`：迁自 `assets/anm/front/ename.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/front/front00.png`：迁自 `assets/anm/front/front00.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/front/front00s.png`：迁自 `assets/anm/front/front00s.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/front/st01logo.png`：迁自 `assets/anm/front/st01logo.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/front/st02logo.png`：迁自 `assets/anm/front/st02logo.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/front/st03logo.png`：迁自 `assets/anm/front/st03logo.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/front/st04logo.png`：迁自 `assets/anm/front/st04logo.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/front/st05logo.png`：迁自 `assets/anm/front/st05logo.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/front/st06logo.png`：迁自 `assets/anm/front/st06logo.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/front/st07logo.png`：迁自 `assets/anm/front/st07logo.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/loading/sig.png`：迁自 `assets/anm/loading/sig.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/loading/sig_r.png`：迁自 `assets/anm/loading/sig_r.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/loading/sigm.png`：迁自 `assets/anm/loading/sigm.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/player/pl00/pl00.png`：迁自 `assets/anm/player/pl00/pl00.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/player/pl01/pl01.png`：迁自 `assets/anm/player/pl01/pl01.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/stgenm/stg1enm.png`：迁自 `assets/anm/stgenm/stg1enm.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/stgenm/stg2enm.png`：迁自 `assets/anm/stgenm/stg2enm.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/stgenm/stg3enm.png`：迁自 `assets/anm/stgenm/stg3enm.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/stgenm/stg3enm2.png`：迁自 `assets/anm/stgenm/stg3enm2.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/stgenm/stg4benm.png`：迁自 `assets/anm/stgenm/stg4benm.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/stgenm/stg4enm.png`：迁自 `assets/anm/stgenm/stg4enm.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/stgenm/stg5enm.png`：迁自 `assets/anm/stgenm/stg5enm.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/stgenm/stg6enm.png`：迁自 `assets/anm/stgenm/stg6enm.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/stgenm/stg6enm2.png`：迁自 `assets/anm/stgenm/stg6enm2.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/stgenm/stg6enm3.png`：迁自 `assets/anm/stgenm/stg6enm3.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/stgenm/stg6enm4.png`：迁自 `assets/anm/stgenm/stg6enm4.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/stgenm/stg7enm.png`：迁自 `assets/anm/stgenm/stg7enm.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/stgenm/stg7enm2.png`：迁自 `assets/anm/stgenm/stg7enm2.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/stgenm/stg7enm3.png`：迁自 `assets/anm/stgenm/stg7enm3.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/title/rank00.png`：迁自 `assets/anm/title/rank00.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/title/result00.png`：迁自 `assets/anm/title/result00.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/title/select00.png`：迁自 `assets/anm/title/select00.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/title/select00b.png`：迁自 `assets/anm/title/select00b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/title/select00s.png`：迁自 `assets/anm/title/select00s.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/title/select01.png`：迁自 `assets/anm/title/select01.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/title/sl_pl00.png`：迁自 `assets/anm/title/sl_pl00.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/title/sl_pl00b.png`：迁自 `assets/anm/title/sl_pl00b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/title/sl_pl01.png`：迁自 `assets/anm/title/sl_pl01.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/title/sl_pl01b.png`：迁自 `assets/anm/title/sl_pl01b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/title/title00a.png`：迁自 `assets/anm/title/title00a.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/title/title00b.png`：迁自 `assets/anm/title/title00b.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/title/title00s.png`：迁自 `assets/anm/title/title00s.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/title/title01.png`：迁自 `assets/anm/title/title01.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/title/title_logo.png`：迁自 `assets/anm/title/title_logo.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/title/title_ver.png`：迁自 `assets/anm/title/title_ver.png`，内容不变，只作开发参考，不嵌入。
- `docs/reference/decoded_textures/title/weapon.png`：迁自 `assets/anm/title/weapon.png`，内容不变，只作开发参考，不嵌入。
- `docs/research/anm_sprite_report.md`：迁自 `assets/anm/sprite_report.md`，内容不变，只作开发参考，不嵌入。
- `docs/research/assets_characters.json`：资源普查、生成清单、验证证据或本任务交付说明。
- `docs/research/assets_size.md`：资源普查、生成清单、验证证据或本任务交付说明。
- `docs/research/ecl_dump/stage01.ecl.txt`：迁自 `assets/ecl/stage01.ecl.txt`，内容不变，只作开发参考，不嵌入。
- `docs/research/ecl_dump/stage02.ecl.txt`：迁自 `assets/ecl/stage02.ecl.txt`，内容不变，只作开发参考，不嵌入。
- `docs/research/ecl_dump/stage03.ecl.txt`：迁自 `assets/ecl/stage03.ecl.txt`，内容不变，只作开发参考，不嵌入。
- `docs/research/ecl_dump/stage04.ecl.txt`：迁自 `assets/ecl/stage04.ecl.txt`，内容不变，只作开发参考，不嵌入。
- `docs/research/ecl_dump/stage05.ecl.txt`：迁自 `assets/ecl/stage05.ecl.txt`，内容不变，只作开发参考，不嵌入。
- `docs/research/ecl_dump/stage06.ecl.txt`：迁自 `assets/ecl/stage06.ecl.txt`，内容不变，只作开发参考，不嵌入。
- `docs/research/font_subset.json`：资源普查、生成清单、验证证据或本任务交付说明。
- `docs/research/se_manifest.json`：资源普查、生成清单、验证证据或本任务交付说明。
- `docs/research/textures_manifest.json`：资源普查、生成清单、验证证据或本任务交付说明。
- `docs/study/01_弹幕游戏制作工具概述（上）.md`：迁自 `assets/study/01_弹幕游戏制作工具概述（上）.md`，内容不变，只作开发参考，不嵌入。
- `docs/study/02_弹幕游戏制作工具概述（下）.md`：迁自 `assets/study/02_弹幕游戏制作工具概述（下）.md`，内容不变，只作开发参考，不嵌入。
- `docs/study/03_弹幕游戏中的物体种类与显示次序.md`：迁自 `assets/study/03_弹幕游戏中的物体种类与显示次序.md`，内容不变，只作开发参考，不嵌入。
- `docs/study/04_同调弹幕所面对的技术问题与解决方案.md`：迁自 `assets/study/04_同调弹幕所面对的技术问题与解决方案.md`，内容不变，只作开发参考，不嵌入。
- `docs/study/05_帧率控制（表）.md`：迁自 `assets/study/05_帧率控制（表）.md`，内容不变，只作开发参考，不嵌入。
- `docs/study/06_垂直同步与全屏模式.md`：迁自 `assets/study/06_垂直同步与全屏模式.md`，内容不变，只作开发参考，不嵌入。
- `docs/study/07_帧率控制（里）.md`：迁自 `assets/study/07_帧率控制（里）.md`，内容不变，只作开发参考，不嵌入。
- `docs/study/08_音效播放.md`：迁自 `assets/study/08_音效播放.md`，内容不变，只作开发参考，不嵌入。
- `docs/study/09_游戏程序的最基本架构.md`：迁自 `assets/study/09_游戏程序的最基本架构.md`，内容不变，只作开发参考，不嵌入。
- `docs/study/10_输入采集与键映射.md`：迁自 `assets/study/10_输入采集与键映射.md`，内容不变，只作开发参考，不嵌入。
- `docs/study/11_（第十一篇）.md`：迁自 `assets/study/11_（第十一篇）.md`，内容不变，只作开发参考，不嵌入。
- `docs/study/12.5_关于正交圆判定的分析.md`：迁自 `assets/study/12.5_关于正交圆判定的分析.md`，内容不变，只作开发参考，不嵌入。
- `docs/study/12_判定形状.md`：迁自 `assets/study/12_判定形状.md`，内容不变，只作开发参考，不嵌入。
- `docs/study/13_（第十三篇）.md`：迁自 `assets/study/13_（第十三篇）.md`，内容不变，只作开发参考，不嵌入。
- `docs/study/14_（第十四篇）.md`：迁自 `assets/study/14_（第十四篇）.md`，内容不变，只作开发参考，不嵌入。
- `docs/study/15_怎样做出东方STG的手感.md`：迁自 `assets/study/15_怎样做出东方STG的手感.md`，内容不变，只作开发参考，不嵌入。
- `docs/study/16_玩家数据.md`：迁自 `assets/study/16_玩家数据.md`，内容不变，只作开发参考，不嵌入。
- `gamedata/compact_se.go`：在既有资源加载边界惰性还原，保持公开接口和私有可变副本。
- `gamedata/compact_se_test.go`：资源原字节、像素、质量、私有缓存、采样长度或确定性回归测试。
- `gamedata/packed_test.go`：资源原字节、像素、质量、私有缓存、采样长度或确定性回归测试。
- `render/font_pixels_data_test.go`：资源原字节、像素、质量、私有缓存、采样长度或确定性回归测试。
- `render/font_subset_test.go`：资源原字节、像素、质量、私有缓存、采样长度或确定性回归测试。
- `tools/th10extract/audit.go`：开发期普查、原档提取、确定性编码、裁剪或质量选择工具。
- `tools/th10extract/fontchars.go`：开发期普查、原档提取、确定性编码、裁剪或质量选择工具。
- `tools/th10extract/fonts.go`：开发期普查、原档提取、确定性编码、裁剪或质量选择工具。
- `tools/th10extract/fonts_test.go`：资源原字节、像素、质量、私有缓存、采样长度或确定性回归测试。
- `tools/th10extract/sounds.go`：开发期普查、原档提取、确定性编码、裁剪或质量选择工具。
- `tools/th10extract/texturepack.go`：开发期普查、原档提取、确定性编码、裁剪或质量选择工具。
- `tools/th10extract/texturepack_test.go`：资源原字节、像素、质量、私有缓存、采样长度或确定性回归测试。
- `tools/th10extract/webtextures.go`：开发期普查、原档提取、确定性编码、裁剪或质量选择工具。
- `tools/th10extract/webtextures_test.go`：资源原字节、像素、质量、私有缓存、采样长度或确定性回归测试。

### 修改

- `assets/anm/ascii.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/bullet.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/e00.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/e01.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/e02.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/e03.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/e04.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/e05.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/e06.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/e07.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/e08.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/e09.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/e10.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/e11.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/enemy.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/front.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/front.ant`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/pl00.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/pl01.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/sig.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/st01logo.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/st01logo.ant`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/st02logo.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/st02logo.ant`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/st03logo.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/st03logo.ant`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/st04logo.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/st04logo.ant`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/st05logo.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/st05logo.ant`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/st06logo.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/st06logo.ant`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/st07logo.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/st07logo.ant`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/staff.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/staff.ant`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/stage01.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/stage02.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/stage03.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/stage04.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/stage05.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/stage06.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/stage07.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/stgenm01.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/stgenm02.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/stgenm03.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/stgenm04.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/stgenm05.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/stgenm06.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/stgenm07.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/text.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/title.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/title.ant`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/anm/title_v.anm`：重生成 ANP1 非像素容器或惰性图片名字索引，保持原逻辑字节。
- `assets/assets.go`：只嵌入运行时文件；按 js 标签选择像素包。
- `assets/font/Bimini Bold.ttf`：裁剪首 face 的固定字符及依赖，保留 glyph id、度量、hint 和 .notdef。
- `assets/font/DFYuGaSo-W3.ttc`：裁剪首 face 的固定字符及依赖，保留 glyph id、度量、hint 和 .notdef。
- `assets/font/DFYuGaSo-W7.ttc`：裁剪首 face 的固定字符及依赖，保留 glyph id、度量、hint 和 .notdef。
- `assets/wav/bgm/th10_00.ogg`：从原 PCM 重编码 Vorbis q1，循环采样位置不变。
- `assets/wav/bgm/th10_01.ogg`：从原 PCM 重编码 Vorbis q1，循环采样位置不变。
- `assets/wav/bgm/th10_02.ogg`：从原 PCM 重编码 Vorbis q1，循环采样位置不变。
- `assets/wav/bgm/th10_03.ogg`：从原 PCM 重编码 Vorbis q1，循环采样位置不变。
- `assets/wav/bgm/th10_04.ogg`：从原 PCM 重编码 Vorbis q1，循环采样位置不变。
- `assets/wav/bgm/th10_05.ogg`：从原 PCM 重编码 Vorbis q1，循环采样位置不变。
- `assets/wav/bgm/th10_06.ogg`：从原 PCM 重编码 Vorbis q1，循环采样位置不变。
- `assets/wav/bgm/th10_07.ogg`：从原 PCM 重编码 Vorbis q1，循环采样位置不变。
- `assets/wav/bgm/th10_08.ogg`：从原 PCM 重编码 Vorbis q1，循环采样位置不变。
- `assets/wav/bgm/th10_09.ogg`：从原 PCM 重编码 Vorbis q1，循环采样位置不变。
- `assets/wav/bgm/th10_10.ogg`：从原 PCM 重编码 Vorbis q1，循环采样位置不变。
- `assets/wav/bgm/th10_11.ogg`：从原 PCM 重编码 Vorbis q1，循环采样位置不变。
- `assets/wav/bgm/th10_12.ogg`：从原 PCM 重编码 Vorbis q1，循环采样位置不变。
- `assets/wav/bgm/th10_13.ogg`：从原 PCM 重编码 Vorbis q1，循环采样位置不变。
- `assets/wav/bgm/th10_14.ogg`：从原 PCM 重编码 Vorbis q1，循环采样位置不变。
- `assets/wav/bgm/th10_15.ogg`：从原 PCM 重编码 Vorbis q1，循环采样位置不变。
- `assets/wav/bgm/th10_16.ogg`：从原 PCM 重编码 Vorbis q1，循环采样位置不变。
- `assets/wav/bgm/th10_17.ogg`：从原 PCM 重编码 Vorbis q1，循环采样位置不变。
- `assets/wav/se/se_bonus3.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_cancel00.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_cardget.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_cat00.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_ch00.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_ch01.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_damage00.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_damage01.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_enep00.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_enep01.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_extend.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_graze.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_gun00.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_hint00.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_invalid.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_item00.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_kira00.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_kira01.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_kira02.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_lazer00.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_lazer01.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_ok00.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_option.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_pause.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_pldead00.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_plst00.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_power0.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_power1.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_powerup.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_select00.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_slash.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_tan00.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_tan01.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_tan02.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_timeout.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_timeout2.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `assets/wav/se/se_water.wav`：生成无损 ZSE1／flate WAV，在加载时还原并预解码。
- `audio/audio.go`：提前加载全部 SE PCM，并保留既有 BGM／SE 播放语义。
- `gamedata/embed_test.go`：资源原字节、像素、质量、私有缓存、采样长度或确定性回归测试。
- `gamedata/gamedata.go`：在既有资源加载边界惰性还原，保持公开接口和私有可变副本。
- `gamedata/presentation.go`：在既有资源加载边界惰性还原，保持公开接口和私有可变副本。
- `gamedata/presentation_test.go`：资源原字节、像素、质量、私有缓存、采样长度或确定性回归测试。
- `tools/th10extract/main.go`：开发期普查、原档提取、确定性编码、裁剪或质量选择工具。

### 删除旧位置

- `assets/anm/ascii/ascii.png`：移至 `docs/reference/decoded_textures/ascii/ascii.png`，内容不变。
- `assets/anm/ascii/leaf.png`：移至 `docs/reference/decoded_textures/ascii/leaf.png`，内容不变。
- `assets/anm/ascii/leaf.svg`：移至 `docs/reference/decoded_textures/ascii/leaf.svg`，内容不变。
- `assets/anm/ascii/leaflite.svg`：移至 `docs/reference/decoded_textures/ascii/leaflite.svg`，内容不变。
- `assets/anm/ascii/loading.png`：移至 `docs/reference/decoded_textures/ascii/loading.png`，内容不变。
- `assets/anm/ascii/pause.png`：移至 `docs/reference/decoded_textures/ascii/pause.png`，内容不变。
- `assets/anm/background/stg1bg.png`：移至 `docs/reference/decoded_textures/background/stg1bg.png`，内容不变。
- `assets/anm/background/stg1bg2.png`：移至 `docs/reference/decoded_textures/background/stg1bg2.png`，内容不变。
- `assets/anm/background/stg1bg3.png`：移至 `docs/reference/decoded_textures/background/stg1bg3.png`，内容不变。
- `assets/anm/background/stg1bg4.png`：移至 `docs/reference/decoded_textures/background/stg1bg4.png`，内容不变。
- `assets/anm/background/stg2bg.png`：移至 `docs/reference/decoded_textures/background/stg2bg.png`，内容不变。
- `assets/anm/background/stg2bg2.png`：移至 `docs/reference/decoded_textures/background/stg2bg2.png`，内容不变。
- `assets/anm/background/stg3bg.png`：移至 `docs/reference/decoded_textures/background/stg3bg.png`，内容不变。
- `assets/anm/background/stg3bg2.png`：移至 `docs/reference/decoded_textures/background/stg3bg2.png`，内容不变。
- `assets/anm/background/stg3bg3.png`：移至 `docs/reference/decoded_textures/background/stg3bg3.png`，内容不变。
- `assets/anm/background/stg3bg4.png`：移至 `docs/reference/decoded_textures/background/stg3bg4.png`，内容不变。
- `assets/anm/background/stg4bg.png`：移至 `docs/reference/decoded_textures/background/stg4bg.png`，内容不变。
- `assets/anm/background/stg4bg3.png`：移至 `docs/reference/decoded_textures/background/stg4bg3.png`，内容不变。
- `assets/anm/background/stg4bg7.png`：移至 `docs/reference/decoded_textures/background/stg4bg7.png`，内容不变。
- `assets/anm/background/stg5bg.png`：移至 `docs/reference/decoded_textures/background/stg5bg.png`，内容不变。
- `assets/anm/background/stg5bg2.png`：移至 `docs/reference/decoded_textures/background/stg5bg2.png`，内容不变。
- `assets/anm/background/stg6bg.png`：移至 `docs/reference/decoded_textures/background/stg6bg.png`，内容不变。
- `assets/anm/background/stg6bg2.png`：移至 `docs/reference/decoded_textures/background/stg6bg2.png`，内容不变。
- `assets/anm/background/stg6bg3.png`：移至 `docs/reference/decoded_textures/background/stg6bg3.png`，内容不变。
- `assets/anm/background/stg6bg5.png`：移至 `docs/reference/decoded_textures/background/stg6bg5.png`，内容不变。
- `assets/anm/background/stg6bg6.png`：移至 `docs/reference/decoded_textures/background/stg6bg6.png`，内容不变。
- `assets/anm/background/stg7bg.png`：移至 `docs/reference/decoded_textures/background/stg7bg.png`，内容不变。
- `assets/anm/background/stg7bg2.png`：移至 `docs/reference/decoded_textures/background/stg7bg2.png`，内容不变。
- `assets/anm/background/stg7bg3.png`：移至 `docs/reference/decoded_textures/background/stg7bg3.png`，内容不变。
- `assets/anm/bullet/etama.png`：移至 `docs/reference/decoded_textures/bullet/etama.png`，内容不变。
- `assets/anm/bullet/etama2.png`：移至 `docs/reference/decoded_textures/bullet/etama2.png`，内容不变。
- `assets/anm/bullet/etama3.png`：移至 `docs/reference/decoded_textures/bullet/etama3.png`，内容不变。
- `assets/anm/bullet/etama6.png`：移至 `docs/reference/decoded_textures/bullet/etama6.png`，内容不变。
- `assets/anm/card/cdbg01a.png`：移至 `docs/reference/decoded_textures/card/cdbg01a.png`，内容不变。
- `assets/anm/card/cdbg01b.png`：移至 `docs/reference/decoded_textures/card/cdbg01b.png`，内容不变。
- `assets/anm/card/cdbg02a.png`：移至 `docs/reference/decoded_textures/card/cdbg02a.png`，内容不变。
- `assets/anm/card/cdbg02b.png`：移至 `docs/reference/decoded_textures/card/cdbg02b.png`，内容不变。
- `assets/anm/card/cdbg03a.png`：移至 `docs/reference/decoded_textures/card/cdbg03a.png`，内容不变。
- `assets/anm/card/cdbg03b.png`：移至 `docs/reference/decoded_textures/card/cdbg03b.png`，内容不变。
- `assets/anm/card/cdbg04a.png`：移至 `docs/reference/decoded_textures/card/cdbg04a.png`，内容不变。
- `assets/anm/card/cdbg04b.png`：移至 `docs/reference/decoded_textures/card/cdbg04b.png`，内容不变。
- `assets/anm/card/cdbg05a.png`：移至 `docs/reference/decoded_textures/card/cdbg05a.png`，内容不变。
- `assets/anm/card/cdbg05b.png`：移至 `docs/reference/decoded_textures/card/cdbg05b.png`，内容不变。
- `assets/anm/card/cdbg06a.png`：移至 `docs/reference/decoded_textures/card/cdbg06a.png`，内容不变。
- `assets/anm/card/cdbg06b.png`：移至 `docs/reference/decoded_textures/card/cdbg06b.png`，内容不变。
- `assets/anm/card/cdbg07a.png`：移至 `docs/reference/decoded_textures/card/cdbg07a.png`，内容不变。
- `assets/anm/card/cdbg07b.png`：移至 `docs/reference/decoded_textures/card/cdbg07b.png`，内容不变。
- `assets/anm/ending/e00a.png`：移至 `docs/reference/decoded_textures/ending/e00a.png`，内容不变。
- `assets/anm/ending/e00ar.png`：移至 `docs/reference/decoded_textures/ending/e00ar.png`，内容不变。
- `assets/anm/ending/e00b.png`：移至 `docs/reference/decoded_textures/ending/e00b.png`，内容不变。
- `assets/anm/ending/e00br.png`：移至 `docs/reference/decoded_textures/ending/e00br.png`，内容不变。
- `assets/anm/ending/e01b.png`：移至 `docs/reference/decoded_textures/ending/e01b.png`，内容不变。
- `assets/anm/ending/e01br.png`：移至 `docs/reference/decoded_textures/ending/e01br.png`，内容不变。
- `assets/anm/ending/e02b.png`：移至 `docs/reference/decoded_textures/ending/e02b.png`，内容不变。
- `assets/anm/ending/e02br.png`：移至 `docs/reference/decoded_textures/ending/e02br.png`，内容不变。
- `assets/anm/ending/e03b.png`：移至 `docs/reference/decoded_textures/ending/e03b.png`，内容不变。
- `assets/anm/ending/e03br.png`：移至 `docs/reference/decoded_textures/ending/e03br.png`，内容不变。
- `assets/anm/ending/e04b.png`：移至 `docs/reference/decoded_textures/ending/e04b.png`，内容不变。
- `assets/anm/ending/e04br.png`：移至 `docs/reference/decoded_textures/ending/e04br.png`，内容不变。
- `assets/anm/ending/e05b.png`：移至 `docs/reference/decoded_textures/ending/e05b.png`，内容不变。
- `assets/anm/ending/e05br.png`：移至 `docs/reference/decoded_textures/ending/e05br.png`，内容不变。
- `assets/anm/ending/e06b.png`：移至 `docs/reference/decoded_textures/ending/e06b.png`，内容不变。
- `assets/anm/ending/e06br.png`：移至 `docs/reference/decoded_textures/ending/e06br.png`，内容不变。
- `assets/anm/ending/e06c.png`：移至 `docs/reference/decoded_textures/ending/e06c.png`，内容不变。
- `assets/anm/ending/e06cr.png`：移至 `docs/reference/decoded_textures/ending/e06cr.png`，内容不变。
- `assets/anm/ending/e07b.png`：移至 `docs/reference/decoded_textures/ending/e07b.png`，内容不变。
- `assets/anm/ending/e07br.png`：移至 `docs/reference/decoded_textures/ending/e07br.png`，内容不变。
- `assets/anm/ending/e07c.png`：移至 `docs/reference/decoded_textures/ending/e07c.png`，内容不变。
- `assets/anm/ending/e07cr.png`：移至 `docs/reference/decoded_textures/ending/e07cr.png`，内容不变。
- `assets/anm/ending/e08b.png`：移至 `docs/reference/decoded_textures/ending/e08b.png`，内容不变。
- `assets/anm/ending/e08br.png`：移至 `docs/reference/decoded_textures/ending/e08br.png`，内容不变。
- `assets/anm/ending/e08c.png`：移至 `docs/reference/decoded_textures/ending/e08c.png`，内容不变。
- `assets/anm/ending/e08cr.png`：移至 `docs/reference/decoded_textures/ending/e08cr.png`，内容不变。
- `assets/anm/ending/e09a.png`：移至 `docs/reference/decoded_textures/ending/e09a.png`，内容不变。
- `assets/anm/ending/e09ar.png`：移至 `docs/reference/decoded_textures/ending/e09ar.png`，内容不变。
- `assets/anm/ending/e09b.png`：移至 `docs/reference/decoded_textures/ending/e09b.png`，内容不变。
- `assets/anm/ending/e09br.png`：移至 `docs/reference/decoded_textures/ending/e09br.png`，内容不变。
- `assets/anm/ending/e09c.png`：移至 `docs/reference/decoded_textures/ending/e09c.png`，内容不变。
- `assets/anm/ending/e09cr.png`：移至 `docs/reference/decoded_textures/ending/e09cr.png`，内容不变。
- `assets/anm/ending/e10a.png`：移至 `docs/reference/decoded_textures/ending/e10a.png`，内容不变。
- `assets/anm/ending/e10ar.png`：移至 `docs/reference/decoded_textures/ending/e10ar.png`，内容不变。
- `assets/anm/ending/e10b.png`：移至 `docs/reference/decoded_textures/ending/e10b.png`，内容不变。
- `assets/anm/ending/e10br.png`：移至 `docs/reference/decoded_textures/ending/e10br.png`，内容不变。
- `assets/anm/ending/e10c.png`：移至 `docs/reference/decoded_textures/ending/e10c.png`，内容不变。
- `assets/anm/ending/e10cr.png`：移至 `docs/reference/decoded_textures/ending/e10cr.png`，内容不变。
- `assets/anm/ending/e11b.png`：移至 `docs/reference/decoded_textures/ending/e11b.png`，内容不变。
- `assets/anm/ending/e11br.png`：移至 `docs/reference/decoded_textures/ending/e11br.png`，内容不变。
- `assets/anm/ending/e11c.png`：移至 `docs/reference/decoded_textures/ending/e11c.png`，内容不变。
- `assets/anm/ending/e11cr.png`：移至 `docs/reference/decoded_textures/ending/e11cr.png`，内容不变。
- `assets/anm/ending/st00.png`：移至 `docs/reference/decoded_textures/ending/st00.png`，内容不变。
- `assets/anm/ending/st01.png`：移至 `docs/reference/decoded_textures/ending/st01.png`，内容不变。
- `assets/anm/ending/st01r.png`：移至 `docs/reference/decoded_textures/ending/st01r.png`，内容不变。
- `assets/anm/ending/st02.png`：移至 `docs/reference/decoded_textures/ending/st02.png`，内容不变。
- `assets/anm/ending/st03.png`：移至 `docs/reference/decoded_textures/ending/st03.png`，内容不变。
- `assets/anm/ending/st03r.png`：移至 `docs/reference/decoded_textures/ending/st03r.png`，内容不变。
- `assets/anm/ending/st04.png`：移至 `docs/reference/decoded_textures/ending/st04.png`，内容不变。
- `assets/anm/ending/st05.png`：移至 `docs/reference/decoded_textures/ending/st05.png`，内容不变。
- `assets/anm/ending/st05r.png`：移至 `docs/reference/decoded_textures/ending/st05r.png`，内容不变。
- `assets/anm/ending/staff.png`：移至 `docs/reference/decoded_textures/ending/staff.png`，内容不变。
- `assets/anm/ending/staff2.png`：移至 `docs/reference/decoded_textures/ending/staff2.png`，内容不变。
- `assets/anm/enemy/enemy.png`：移至 `docs/reference/decoded_textures/enemy/enemy.png`，内容不变。
- `assets/anm/face/dummy.png`：移至 `docs/reference/decoded_textures/face/dummy.png`，内容不变。
- `assets/anm/face/enemy1/ename01.png`：移至 `docs/reference/decoded_textures/face/enemy1/ename01.png`，内容不变。
- `assets/anm/face/enemy1/face01an_d.png`：移至 `docs/reference/decoded_textures/face/enemy1/face01an_d.png`，内容不变。
- `assets/anm/face/enemy1/face01an_u.png`：移至 `docs/reference/decoded_textures/face/enemy1/face01an_u.png`，内容不变。
- `assets/anm/face/enemy1/face01ct.png`：移至 `docs/reference/decoded_textures/face/enemy1/face01ct.png`，内容不变。
- `assets/anm/face/enemy1/face01n2_d.png`：移至 `docs/reference/decoded_textures/face/enemy1/face01n2_d.png`，内容不变。
- `assets/anm/face/enemy1/face01n2_u.png`：移至 `docs/reference/decoded_textures/face/enemy1/face01n2_u.png`，内容不变。
- `assets/anm/face/enemy1/face01no_d.png`：移至 `docs/reference/decoded_textures/face/enemy1/face01no_d.png`，内容不变。
- `assets/anm/face/enemy1/face01no_u.png`：移至 `docs/reference/decoded_textures/face/enemy1/face01no_u.png`，内容不变。
- `assets/anm/face/enemy1m/face01mct.png`：移至 `docs/reference/decoded_textures/face/enemy1m/face01mct.png`，内容不变。
- `assets/anm/face/enemy2/ename02.png`：移至 `docs/reference/decoded_textures/face/enemy2/ename02.png`，内容不变。
- `assets/anm/face/enemy2/face02an_u.png`：移至 `docs/reference/decoded_textures/face/enemy2/face02an_u.png`，内容不变。
- `assets/anm/face/enemy2/face02ct.png`：移至 `docs/reference/decoded_textures/face/enemy2/face02ct.png`，内容不变。
- `assets/anm/face/enemy2/face02lo_u.png`：移至 `docs/reference/decoded_textures/face/enemy2/face02lo_u.png`，内容不变。
- `assets/anm/face/enemy2/face02n2_u.png`：移至 `docs/reference/decoded_textures/face/enemy2/face02n2_u.png`，内容不变。
- `assets/anm/face/enemy2/face02no_d.png`：移至 `docs/reference/decoded_textures/face/enemy2/face02no_d.png`，内容不变。
- `assets/anm/face/enemy2/face02no_u.png`：移至 `docs/reference/decoded_textures/face/enemy2/face02no_u.png`，内容不变。
- `assets/anm/face/enemy2/face02pr_u.png`：移至 `docs/reference/decoded_textures/face/enemy2/face02pr_u.png`，内容不变。
- `assets/anm/face/enemy3/ename03.png`：移至 `docs/reference/decoded_textures/face/enemy3/ename03.png`，内容不变。
- `assets/anm/face/enemy3/face03an_u.png`：移至 `docs/reference/decoded_textures/face/enemy3/face03an_u.png`，内容不变。
- `assets/anm/face/enemy3/face03ct.png`：移至 `docs/reference/decoded_textures/face/enemy3/face03ct.png`，内容不变。
- `assets/anm/face/enemy3/face03lo_u.png`：移至 `docs/reference/decoded_textures/face/enemy3/face03lo_u.png`，内容不变。
- `assets/anm/face/enemy3/face03n2_u.png`：移至 `docs/reference/decoded_textures/face/enemy3/face03n2_u.png`，内容不变。
- `assets/anm/face/enemy3/face03no_d.png`：移至 `docs/reference/decoded_textures/face/enemy3/face03no_d.png`，内容不变。
- `assets/anm/face/enemy3/face03no_u.png`：移至 `docs/reference/decoded_textures/face/enemy3/face03no_u.png`，内容不变。
- `assets/anm/face/enemy3/face03sp_u.png`：移至 `docs/reference/decoded_textures/face/enemy3/face03sp_u.png`，内容不变。
- `assets/anm/face/enemy4/ename04.png`：移至 `docs/reference/decoded_textures/face/enemy4/ename04.png`，内容不变。
- `assets/anm/face/enemy4/face04ct.png`：移至 `docs/reference/decoded_textures/face/enemy4/face04ct.png`，内容不变。
- `assets/anm/face/enemy4/face04dp_u.png`：移至 `docs/reference/decoded_textures/face/enemy4/face04dp_u.png`，内容不变。
- `assets/anm/face/enemy4/face04lo_u.png`：移至 `docs/reference/decoded_textures/face/enemy4/face04lo_u.png`，内容不变。
- `assets/anm/face/enemy4/face04n2_u.png`：移至 `docs/reference/decoded_textures/face/enemy4/face04n2_u.png`，内容不变。
- `assets/anm/face/enemy4/face04no_d.png`：移至 `docs/reference/decoded_textures/face/enemy4/face04no_d.png`，内容不变。
- `assets/anm/face/enemy4/face04no_u.png`：移至 `docs/reference/decoded_textures/face/enemy4/face04no_u.png`，内容不变。
- `assets/anm/face/enemy4/face04pr_u.png`：移至 `docs/reference/decoded_textures/face/enemy4/face04pr_u.png`，内容不变。
- `assets/anm/face/enemy4/face04sp_u.png`：移至 `docs/reference/decoded_textures/face/enemy4/face04sp_u.png`，内容不变。
- `assets/anm/face/enemy5/ename05.png`：移至 `docs/reference/decoded_textures/face/enemy5/ename05.png`，内容不变。
- `assets/anm/face/enemy5/face05an_u.png`：移至 `docs/reference/decoded_textures/face/enemy5/face05an_u.png`，内容不变。
- `assets/anm/face/enemy5/face05ct.png`：移至 `docs/reference/decoded_textures/face/enemy5/face05ct.png`，内容不变。
- `assets/anm/face/enemy5/face05dp_u.png`：移至 `docs/reference/decoded_textures/face/enemy5/face05dp_u.png`，内容不变。
- `assets/anm/face/enemy5/face05lo_u.png`：移至 `docs/reference/decoded_textures/face/enemy5/face05lo_u.png`，内容不变。
- `assets/anm/face/enemy5/face05n2_u.png`：移至 `docs/reference/decoded_textures/face/enemy5/face05n2_u.png`，内容不变。
- `assets/anm/face/enemy5/face05no_d.png`：移至 `docs/reference/decoded_textures/face/enemy5/face05no_d.png`，内容不变。
- `assets/anm/face/enemy5/face05no_u.png`：移至 `docs/reference/decoded_textures/face/enemy5/face05no_u.png`，内容不变。
- `assets/anm/face/enemy5/face05sp_u.png`：移至 `docs/reference/decoded_textures/face/enemy5/face05sp_u.png`，内容不变。
- `assets/anm/face/enemy5/face05sw_u.png`：移至 `docs/reference/decoded_textures/face/enemy5/face05sw_u.png`，内容不变。
- `assets/anm/face/enemy6/ename06.png`：移至 `docs/reference/decoded_textures/face/enemy6/ename06.png`，内容不变。
- `assets/anm/face/enemy6/face06ct.png`：移至 `docs/reference/decoded_textures/face/enemy6/face06ct.png`，内容不变。
- `assets/anm/face/enemy6/face06dp_u.png`：移至 `docs/reference/decoded_textures/face/enemy6/face06dp_u.png`，内容不变。
- `assets/anm/face/enemy6/face06n2_u.png`：移至 `docs/reference/decoded_textures/face/enemy6/face06n2_u.png`，内容不变。
- `assets/anm/face/enemy6/face06no_d.png`：移至 `docs/reference/decoded_textures/face/enemy6/face06no_d.png`，内容不变。
- `assets/anm/face/enemy6/face06no_u.png`：移至 `docs/reference/decoded_textures/face/enemy6/face06no_u.png`，内容不变。
- `assets/anm/face/enemy7/ename07.png`：移至 `docs/reference/decoded_textures/face/enemy7/ename07.png`，内容不变。
- `assets/anm/face/enemy7/face07ct.png`：移至 `docs/reference/decoded_textures/face/enemy7/face07ct.png`，内容不变。
- `assets/anm/face/enemy7/face07dp_u.png`：移至 `docs/reference/decoded_textures/face/enemy7/face07dp_u.png`，内容不变。
- `assets/anm/face/enemy7/face07lo_u.png`：移至 `docs/reference/decoded_textures/face/enemy7/face07lo_u.png`，内容不变。
- `assets/anm/face/enemy7/face07no_d.png`：移至 `docs/reference/decoded_textures/face/enemy7/face07no_d.png`，内容不变。
- `assets/anm/face/enemy7/face07no_u.png`：移至 `docs/reference/decoded_textures/face/enemy7/face07no_u.png`，内容不变。
- `assets/anm/face/enemy7/face07pr_u.png`：移至 `docs/reference/decoded_textures/face/enemy7/face07pr_u.png`，内容不变。
- `assets/anm/face/pl00/face_pl00an_u.png`：移至 `docs/reference/decoded_textures/face/pl00/face_pl00an_u.png`，内容不变。
- `assets/anm/face/pl00/face_pl00dp_u.png`：移至 `docs/reference/decoded_textures/face/pl00/face_pl00dp_u.png`，内容不变。
- `assets/anm/face/pl00/face_pl00hp_u.png`：移至 `docs/reference/decoded_textures/face/pl00/face_pl00hp_u.png`，内容不变。
- `assets/anm/face/pl00/face_pl00n2_u.png`：移至 `docs/reference/decoded_textures/face/pl00/face_pl00n2_u.png`，内容不变。
- `assets/anm/face/pl00/face_pl00no_d.png`：移至 `docs/reference/decoded_textures/face/pl00/face_pl00no_d.png`，内容不变。
- `assets/anm/face/pl00/face_pl00no_u.png`：移至 `docs/reference/decoded_textures/face/pl00/face_pl00no_u.png`，内容不变。
- `assets/anm/face/pl00/face_pl00pr_u.png`：移至 `docs/reference/decoded_textures/face/pl00/face_pl00pr_u.png`，内容不变。
- `assets/anm/face/pl00/face_pl00sp_u.png`：移至 `docs/reference/decoded_textures/face/pl00/face_pl00sp_u.png`，内容不变。
- `assets/anm/face/pl00/face_pl00sw_u.png`：移至 `docs/reference/decoded_textures/face/pl00/face_pl00sw_u.png`，内容不变。
- `assets/anm/face/pl01/face_pl01an_u.png`：移至 `docs/reference/decoded_textures/face/pl01/face_pl01an_u.png`，内容不变。
- `assets/anm/face/pl01/face_pl01dp_u.png`：移至 `docs/reference/decoded_textures/face/pl01/face_pl01dp_u.png`，内容不变。
- `assets/anm/face/pl01/face_pl01hp_u.png`：移至 `docs/reference/decoded_textures/face/pl01/face_pl01hp_u.png`，内容不变。
- `assets/anm/face/pl01/face_pl01n2_u.png`：移至 `docs/reference/decoded_textures/face/pl01/face_pl01n2_u.png`，内容不变。
- `assets/anm/face/pl01/face_pl01no_d.png`：移至 `docs/reference/decoded_textures/face/pl01/face_pl01no_d.png`，内容不变。
- `assets/anm/face/pl01/face_pl01no_u.png`：移至 `docs/reference/decoded_textures/face/pl01/face_pl01no_u.png`，内容不变。
- `assets/anm/face/pl01/face_pl01pr_u.png`：移至 `docs/reference/decoded_textures/face/pl01/face_pl01pr_u.png`，内容不变。
- `assets/anm/face/pl01/face_pl01sp_u.png`：移至 `docs/reference/decoded_textures/face/pl01/face_pl01sp_u.png`，内容不变。
- `assets/anm/face/pl01/face_pl01sw_u.png`：移至 `docs/reference/decoded_textures/face/pl01/face_pl01sw_u.png`，内容不变。
- `assets/anm/front/ename.png`：移至 `docs/reference/decoded_textures/front/ename.png`，内容不变。
- `assets/anm/front/front00.png`：移至 `docs/reference/decoded_textures/front/front00.png`，内容不变。
- `assets/anm/front/front00s.png`：移至 `docs/reference/decoded_textures/front/front00s.png`，内容不变。
- `assets/anm/front/st01logo.png`：移至 `docs/reference/decoded_textures/front/st01logo.png`，内容不变。
- `assets/anm/front/st02logo.png`：移至 `docs/reference/decoded_textures/front/st02logo.png`，内容不变。
- `assets/anm/front/st03logo.png`：移至 `docs/reference/decoded_textures/front/st03logo.png`，内容不变。
- `assets/anm/front/st04logo.png`：移至 `docs/reference/decoded_textures/front/st04logo.png`，内容不变。
- `assets/anm/front/st05logo.png`：移至 `docs/reference/decoded_textures/front/st05logo.png`，内容不变。
- `assets/anm/front/st06logo.png`：移至 `docs/reference/decoded_textures/front/st06logo.png`，内容不变。
- `assets/anm/front/st07logo.png`：移至 `docs/reference/decoded_textures/front/st07logo.png`，内容不变。
- `assets/anm/loading/sig.png`：移至 `docs/reference/decoded_textures/loading/sig.png`，内容不变。
- `assets/anm/loading/sig_r.png`：移至 `docs/reference/decoded_textures/loading/sig_r.png`，内容不变。
- `assets/anm/loading/sigm.png`：移至 `docs/reference/decoded_textures/loading/sigm.png`，内容不变。
- `assets/anm/player/pl00/pl00.png`：移至 `docs/reference/decoded_textures/player/pl00/pl00.png`，内容不变。
- `assets/anm/player/pl01/pl01.png`：移至 `docs/reference/decoded_textures/player/pl01/pl01.png`，内容不变。
- `assets/anm/sprite_report.md`：移至 `docs/research/anm_sprite_report.md`，内容不变。
- `assets/anm/stgenm/stg1enm.png`：移至 `docs/reference/decoded_textures/stgenm/stg1enm.png`，内容不变。
- `assets/anm/stgenm/stg2enm.png`：移至 `docs/reference/decoded_textures/stgenm/stg2enm.png`，内容不变。
- `assets/anm/stgenm/stg3enm.png`：移至 `docs/reference/decoded_textures/stgenm/stg3enm.png`，内容不变。
- `assets/anm/stgenm/stg3enm2.png`：移至 `docs/reference/decoded_textures/stgenm/stg3enm2.png`，内容不变。
- `assets/anm/stgenm/stg4benm.png`：移至 `docs/reference/decoded_textures/stgenm/stg4benm.png`，内容不变。
- `assets/anm/stgenm/stg4enm.png`：移至 `docs/reference/decoded_textures/stgenm/stg4enm.png`，内容不变。
- `assets/anm/stgenm/stg5enm.png`：移至 `docs/reference/decoded_textures/stgenm/stg5enm.png`，内容不变。
- `assets/anm/stgenm/stg6enm.png`：移至 `docs/reference/decoded_textures/stgenm/stg6enm.png`，内容不变。
- `assets/anm/stgenm/stg6enm2.png`：移至 `docs/reference/decoded_textures/stgenm/stg6enm2.png`，内容不变。
- `assets/anm/stgenm/stg6enm3.png`：移至 `docs/reference/decoded_textures/stgenm/stg6enm3.png`，内容不变。
- `assets/anm/stgenm/stg6enm4.png`：移至 `docs/reference/decoded_textures/stgenm/stg6enm4.png`，内容不变。
- `assets/anm/stgenm/stg7enm.png`：移至 `docs/reference/decoded_textures/stgenm/stg7enm.png`，内容不变。
- `assets/anm/stgenm/stg7enm2.png`：移至 `docs/reference/decoded_textures/stgenm/stg7enm2.png`，内容不变。
- `assets/anm/stgenm/stg7enm3.png`：移至 `docs/reference/decoded_textures/stgenm/stg7enm3.png`，内容不变。
- `assets/anm/title/rank00.png`：移至 `docs/reference/decoded_textures/title/rank00.png`，内容不变。
- `assets/anm/title/result00.png`：移至 `docs/reference/decoded_textures/title/result00.png`，内容不变。
- `assets/anm/title/select00.png`：移至 `docs/reference/decoded_textures/title/select00.png`，内容不变。
- `assets/anm/title/select00b.png`：移至 `docs/reference/decoded_textures/title/select00b.png`，内容不变。
- `assets/anm/title/select00s.png`：移至 `docs/reference/decoded_textures/title/select00s.png`，内容不变。
- `assets/anm/title/select01.png`：移至 `docs/reference/decoded_textures/title/select01.png`，内容不变。
- `assets/anm/title/sl_pl00.png`：移至 `docs/reference/decoded_textures/title/sl_pl00.png`，内容不变。
- `assets/anm/title/sl_pl00b.png`：移至 `docs/reference/decoded_textures/title/sl_pl00b.png`，内容不变。
- `assets/anm/title/sl_pl01.png`：移至 `docs/reference/decoded_textures/title/sl_pl01.png`，内容不变。
- `assets/anm/title/sl_pl01b.png`：移至 `docs/reference/decoded_textures/title/sl_pl01b.png`，内容不变。
- `assets/anm/title/title00a.png`：移至 `docs/reference/decoded_textures/title/title00a.png`，内容不变。
- `assets/anm/title/title00b.png`：移至 `docs/reference/decoded_textures/title/title00b.png`，内容不变。
- `assets/anm/title/title00s.png`：移至 `docs/reference/decoded_textures/title/title00s.png`，内容不变。
- `assets/anm/title/title01.png`：移至 `docs/reference/decoded_textures/title/title01.png`，内容不变。
- `assets/anm/title/title_logo.png`：移至 `docs/reference/decoded_textures/title/title_logo.png`，内容不变。
- `assets/anm/title/title_ver.png`：移至 `docs/reference/decoded_textures/title/title_ver.png`，内容不变。
- `assets/anm/title/weapon.png`：移至 `docs/reference/decoded_textures/title/weapon.png`，内容不变。
- `assets/ecl/stage01.ecl.txt`：移至 `docs/research/ecl_dump/stage01.ecl.txt`，内容不变。
- `assets/ecl/stage02.ecl.txt`：移至 `docs/research/ecl_dump/stage02.ecl.txt`，内容不变。
- `assets/ecl/stage03.ecl.txt`：移至 `docs/research/ecl_dump/stage03.ecl.txt`，内容不变。
- `assets/ecl/stage04.ecl.txt`：移至 `docs/research/ecl_dump/stage04.ecl.txt`，内容不变。
- `assets/ecl/stage05.ecl.txt`：移至 `docs/research/ecl_dump/stage05.ecl.txt`，内容不变。
- `assets/ecl/stage06.ecl.txt`：移至 `docs/research/ecl_dump/stage06.ecl.txt`，内容不变。
- `assets/study/01_弹幕游戏制作工具概述（上）.md`：移至 `docs/study/01_弹幕游戏制作工具概述（上）.md`，内容不变。
- `assets/study/02_弹幕游戏制作工具概述（下）.md`：移至 `docs/study/02_弹幕游戏制作工具概述（下）.md`，内容不变。
- `assets/study/03_弹幕游戏中的物体种类与显示次序.md`：移至 `docs/study/03_弹幕游戏中的物体种类与显示次序.md`，内容不变。
- `assets/study/04_同调弹幕所面对的技术问题与解决方案.md`：移至 `docs/study/04_同调弹幕所面对的技术问题与解决方案.md`，内容不变。
- `assets/study/05_帧率控制（表）.md`：移至 `docs/study/05_帧率控制（表）.md`，内容不变。
- `assets/study/06_垂直同步与全屏模式.md`：移至 `docs/study/06_垂直同步与全屏模式.md`，内容不变。
- `assets/study/07_帧率控制（里）.md`：移至 `docs/study/07_帧率控制（里）.md`，内容不变。
- `assets/study/08_音效播放.md`：移至 `docs/study/08_音效播放.md`，内容不变。
- `assets/study/09_游戏程序的最基本架构.md`：移至 `docs/study/09_游戏程序的最基本架构.md`，内容不变。
- `assets/study/10_输入采集与键映射.md`：移至 `docs/study/10_输入采集与键映射.md`，内容不变。
- `assets/study/11_（第十一篇）.md`：移至 `docs/study/11_（第十一篇）.md`，内容不变。
- `assets/study/12.5_关于正交圆判定的分析.md`：移至 `docs/study/12.5_关于正交圆判定的分析.md`，内容不变。
- `assets/study/12_判定形状.md`：移至 `docs/study/12_判定形状.md`，内容不变。
- `assets/study/13_（第十三篇）.md`：移至 `docs/study/13_（第十三篇）.md`，内容不变。
- `assets/study/14_（第十四篇）.md`：移至 `docs/study/14_（第十四篇）.md`，内容不变。
- `assets/study/15_怎样做出东方STG的手感.md`：移至 `docs/study/15_怎样做出东方STG的手感.md`，内容不变。
- `assets/study/16_玩家数据.md`：移至 `docs/study/16_玩家数据.md`，内容不变。

## 3．移植的函数地址

本任务是开发期资源格式／加载适配，没有新增原作函数移植，不能为压缩算法伪造 FUN 地址。继续复用：

| 原依据 | 现有／适配边界 |
|---|---|
| FUN_00447470／00446f40 | `anm.Load`／`Texture.Image`，原 ANM／THTX 字节与尺寸保留；新 `anm.Restore` 仅在其前还原资源 |
| FUN_004479d0／00447a50／00447ae0 | `render.RasterText`／`WriteVMText` 不改；裁剪原字体，不更换 rasterizer |
| FUN_0041f990，SE 表 0x474b40／0x4749c8 | `audio.Manager.preloadSE` 在加载时解码固定 SE，名字／47 id／音量／声像队列不变 |
| 原 thbgm.fmt，既有 BGM 播放适配 | `audio.NewBGMStream` 的 Intro／Size 语义不改；源 PCM SectionReader 恰好到 Size |

## 4．测试

新增：ANP1／TPK1 截断、尺寸／格式错配、确定性、全部 THTX 格式无损往返、透明 RGB、私有可变数据、惰性名字索引；SE 预加载 PCM／安全后备／ZSE1 截断；BGM 全曲实际解码 EOF 长度；字体 glyph／hint／layout／cmap／checksum、原字体逐像素对照与便携像素摘要；两变体非像素字节／精灵／脚本一致及 PSNR／alpha／过滤边缘门槛。

任务明确授权的原资源副本测试改为还原后比较，原计数断言和其他 golden／容差不改。无原 dat 时原档对照跳过；无原字体时直接原字体对照跳过，但由原字体生成的完整像素摘要仍运行。设置 `TH10_FONT_SOURCE` 可指定原字体目录，本机对照使用只读 base 副本。

| 阶段 | build／vet／test -count=1／gofmt／WASM | full smoke 秒 | 六章 cmp |
|---|---|---:|---|
| A | 通过 | 127.616 | 通过 |
| B | 通过 | 126.245 | 通过 |
| C | 通过 | 118.671 | 通过 |
| D | 通过 | 122.285 | 通过 |
| E | 通过 | 125.292 | 通过 |

`golangci-lint run` 已运行，旧范围告警另列；rules 第 6 节明确不把它作验收项。未修改 t16／t17 拥有的旧告警，不宣称全仓 lint 全绿。本任务新增代码告警须为零。日志在任务根 `logs/`。

开发期曾发生并已修：旧 Bimini 被 x/image sfnt 审计拒绝（审计改用已有 go-text）；js 环境先 go list 会碰 Windows probe（桌面枚举后再切 js）；Node 的 Go 退出后 timer 回调（Go Promise 结束后退出）；标准 JPEG 色度采样达不到门槛（追加分通道灰度 JPEG及量化候选，而不是放宽门槛）。

### 解码速度

三次冷加载＋解码＋ANM 解析均值，不含 GPU 上传，不是浏览器实机承诺；后续读取复用已还原字节。原始 benchmark 输出：`logs/B_decode_desktop.log`、`logs/B_decode_wasm.log`、`logs/E_decode_desktop.log`、`logs/E_decode_wasm.log`。WASM 是本机 Node／V8 的真实 wasm，不是桌面模拟。

| ANM | 最终桌面毫秒 | 最终 WASM 毫秒 |
|---|---:|---:|
| ascii | 0.538 | 1.163 |
| bullet | 5.495 | 16.557 |
| stage04 | 28.090 | 60.621 |
| stgenm06 | 16.961 | 21.993 |
| title | 45.126 | 65.234 |
| staff.ant | 34.236 | 125.777 |

SE 全 37 个 WAV 冷加载＋解码／重采样三次平均 610.330 毫秒（桌面），只在设备加载时发生；头less Manager 不解码不会播放的声音。所有 Manager 共享不可变 PCM，name map 私有。

### 可重复生成

```powershell
. 'D:\GoProjects\th10\.tmp\codex\t14\env.ps1'
go run ./tools/th10extract -web-textures -out assets -compare 'D:\GoProjects\th10\.tmp\codex\t14\compare'
go run ./tools/th10extract -pack-se -out assets
go run ./tools/th10extract -bgm -q 1 -out assets
go run ./tools/th10extract -fonts -font-source 'D:\GoProjects\th10\.tmp\codex\base\assets\font' -out assets
```

原 dat／thbgm.dat 和未裁剪字体只作开发输入，运行时、构建均不依赖原作目录或外部编码器。三档 BGM 使用同一 FFmpeg 8.1.2-full_build-www.gyan.dev 的 libvorbis：

```text
ffmpeg -hide_banner -loglevel error -y -f s16le -ar 44100 -ac 2 -i - -c:a libvorbis -q:a 1 -map_metadata -1 -flags +bitexact -fflags +bitexact <out>.ogg
```

参数中的 q 改为 0／2 即试听档；stdin 是原曲 Offset 起、恰好 Size 字节的 PCM。其后原 dat 的预留数据原来就没有编码，不虚报裁切节省。字体、PNG／flate／JPEG／量化由 Go 开发工具生成，无新增依赖。

## 5．Trace 结果

A～E 每阶段均按 c0～c5 顺序生成，用 Python 流式逐字节 cmp（不是哈希替代）与指定 base 比较，再用原 th10diff 的 motion scope 比原作；测试后删除自己的 cN.txt，只保留日志。

| 章 | 与 base | 文件字节 | motion 一致帧 |
|---|---|---:|---:|
| c0 | 逐字节相同 | 94,186,573 | 15853 |
| c1 | 逐字节相同 | 144,171,327 | 17733 |
| c2 | 逐字节相同 | 271,969,413 | 29763 |
| c3 | 逐字节相同 | 319,443,819 | 29469 |
| c4 | 逐字节相同 | 430,006,225 | 39343 |
| c5 | 逐字节相同 | 510,168,148 | 40099 |

参考没有覆盖的尾帧不宣称与原作比过；一致帧数没有下降。不修改 trace schema、采样半帧判定或 th10diff 容差。

## 6．待决与推测

没有新原作语义推测、没有 dump_requests、没有依赖 Ghidra 的阻塞。Source 字符分类是固定源码文案的保守闭包，含少量旧文案；原字体缺字现象保留 .notdef。

**体积目标未达到**：原始 WASM 66.705 MB，超过 40 MB；25 MB 更未达到。已按任务要求报告类别占比与代价，交架构师决定，不擅自降低默认 BGM 档、分辨率或桌面像素精度。

主要占比是默认 BGM q1 的 22.560 MB、12.776 MB ANM／贴图，以及代码／运行时开销。可选代价：q0 降到约 64.5 kbps，节省 4.115 MB（须试听）；更低 Vorbis 档明显牺牲高频／立体声细节，尚未作为默认；放宽颜色 PSNR 会增加边缘／笔触误差，须重生成对照与测试；t16 删除旧 UI 依赖可能降低代码／运行时数据体积，必须在合并后实测。当前 code section 单独约 16.265 MB，与 q1 BGM 合计已约 38.825 MB，还未算其余资源和运行时数据，所以保持本档与现有架构几乎没有 40 MB 余量。外置资源／流式网络加载违反本轮 embed 边界，不自行实施。

若只换 q0，当前原始 WASM 的算术体积为约 62.590 MB；这是固定同长度数据的差额计算，不伪称已构建该档发布二进制。

质量门槛固定为可见 RGB PSNR≥34 dB、全部 RGB（含透明色）≥32 dB、半 texel 线性过滤／alpha 合成边缘≥36 dB；分别对应约 5.1／6.4／4.0 的 8-bit RGB RMS 上界。alpha 原字节严格相同。误差在重新打包成原 THTX 格式之后测，不用 JPEG 解码中间态掩盖 565／4444 量化误差。

当前 67 块有损颜色实测最差：可见 34.058 dB、全部 34.060 dB、边缘 41.125 dB。没有 alpha 边缘或误差为零的值在文档清单中记为 100 dB 上限，测试仍用真实无限值。

透明色没有丢弃：JPEG 用不带 alpha 的原 RGB；alpha 单独无损，量化也不触 alpha。边缘度量读四个 texel 的颜色和 alpha 再做线性采样／合成，补充全部 RGB 门槛，避免只检查 alpha>0 而漏掉过滤渗色。

## 7．建议的 docs/fidelity.md 改动

请架构师替换／新增状态行，不直接把当前便携字体升级成原 GDI 真值：

```text
| 资源嵌入 | 运行时白名单；逻辑与桌面 THTX 原字节一致，js 颜色有界有损 | t14：ANP1／TPK1，源像素哈希去重与惰性恢复；原 dat／ANT 对照、字体裁剪、单套 q1 BGM／无损 SE；docs/research/assets_size.md、docs/handoff/codex_t14.md | 五阶段 build／vet／test／WASM／gofmt／full smoke 与六章 cmp，通过；体积目标和质量试听／目视见交付 |
| 字体资源裁剪 | 与裁剪前便携实现逐像素相同，仍非 GDI L3 | t14：1,939 字符与 glyph／GSUB／复合依赖，首 face、hint／度量不变，未知字符 .notdef | render/font_subset_test.go、tools/th10extract/fonts_test.go；16／17／20／28 px、两种字重、左右对齐与三墨色 |
- 2026-11-08 — t14 资源瘦身：原档逻辑原字节与桌面完整 THTX 原字节保留，按 ANM 首次加载恢复；js 按图选择 JPEG／分通道 JPEG／有损颜色量化，alpha 无损且质量门槛／最差二十图供目视；BGM 降为 q1，SE 无损预解码，字体裁剪与旧实现像素相同。运行时白名单，笔记／PNG 参考迁 docs。五阶段完整验收和六章 cmp 相同，motion 不降；原始／gzip 发布大小与未达体积目标的代价见 codex_t14.md。
```

## 8．需要用户目视确认的项目

- BGM：完整三档在 `D:/GoProjects/th10/.tmp/codex/t14/audition/q0`／`q1`／`q2` 的 `wav/bgm/`。用本机 ffplay 试听同一曲，或在游戏 Music Room 播默认 q1；关注高频打击乐、密集段落、立体声和循环接缝。默认约 80 kbps，不宣称已经听音确认。
- 贴图：打开 `D:/GoProjects/th10/.tmp/codex/t14/compare/index.json` 指向的 20 张图；按可见 RGB MSE 从大到小排序，左无损／中 js／右原 RGB 差异×8，透明区用棋盘格。原尺寸，无缩放。差异图也显示透明 RGB，检查轮廓、细线、文字边缘和笔触。
- 字体：运行 `go test -count=1 ./render ./tools/th10extract`，再在默认游戏流程看 MSG、结局／Staff、Music Room、符卡名与菜单；自动化已经确认裁剪前后像素相同，用户确认的是既有近似字体的观感及原缺字后备，不是裁剪引入了另一种字体。

不启动原作，不把复刻数据写入只读原作 junction。需要更多原字体对照时设置 `TH10_FONT_SOURCE` 为原文件目录；普通动态名称全部可打印 ASCII，超集合字符显示原 .notdef 方块／原缺字效果，不崩溃。

## 9．与其他任务的接口

- t15：入口仍 `./cmd`，未改 cmd／网页／存储／解锁代码。runtime 只经 gamedata embed，js 标签只选 web.tpk。资源加载没有 fetch、dat 读盘、系统字体新增探测。Node benchmark runner仅在任务根，不是 t15 的发布工具。
- t16：没有改任何 scene 文件。旧 `TH10_LEGACY_BG` 的 LoadImage 路径由生成名字索引＋原 ANM 组成，移走旧 PNG 不要求调用方改；旧 render.LoadSystemFace 的系统字体路径未扩展，活动 ANM 文字不用它，清理旧 UI 时请一并去掉此旧 API。
- t17：probe／sim／diff／trace 未改，只按原命令验收。原作数据只读，没有跑 probe／th10.exe。

保持公开签名：`gamedata.ReadFile(string) ([]byte,error)`、`ANM(string) (*anm.File,error)`、`PresentationANM(string) (*anm.File,error)`、`Image(string) (image.Image,error)`、`audio.NewManager(*audio.Context) *Manager`、`NewBGMStream(int,int) (*audio.InfiniteLoop,error)`。现有调用点：加载场景、ECL ANIM／STD、GUI／MSG／结局／菜单和音频 Manager，原调用者／地址列在第 3 节。

新增资源边界完整签名（均不是原作新函数；VM、sprite 和绘制管线无需 hook）：

```go
package anm
func Split(data []byte) ([]byte, map[string]*Texture, error)
func Restore(data []byte, readTexture func(string) ([]byte, error)) ([]byte, error)
func BuildTexturePack(blobs map[string][]byte) ([]byte, error)
func TextureBlob(data []byte, id string) ([]byte, error)
func DecodeTexture(data []byte) (*Texture, error)
func CompareTextureQuality(a, b *Texture) (TextureQuality, error)
```

# 表现层：背景（STD）、ANM、渲染的偏差审计（2026-10-26）

结论：原作道中背景是 **STD 脚本驱动的 3D 场景**（相机、FOV、线性雾，quad 内容由 ANM 脚本驱动），
项目把 PNG 当 2D 平铺。只有 stage1 用了部分 STD 数据，而且是手抄后的 2D 近似。仓库里没有任何代码
读取 `.std`，ANM 也只做了精灵切换。

## 1. 屏幕与场地

- `game/game.go:13-23`：640×480，场地 384×448，偏移 (32,16)，与原作一致。
- `stage.go:152` 把场地画到 384×448 离屏图，`:512-518` 放到 (32,16)；`front00s.png` 作为全屏边框
  （`:168-170`、`:521-523`）。
- 没有纹理过滤等渲染设置（`cmd/main.go:23` 只设窗口大小）。

## 2. ANM

- `anm/loader.go` 正确解析 entry 头、精灵表（`u32 id; f32 x,y,w,h`）、脚本与指令
  （`u16 op,size,time,mask`）；**有意跳过纹理块**，改读 `assets/anm/` 下预先导出的 PNG。
- `anm/resolve.go` 只处理 0（end）、1（stop）、3（set sprite）、4（goto）、5（loop），其余
  `default: pc++`（`:74`）。`SpriteAt(age)` 只返回精灵 id。
- 未实现：位置、旋转、缩放、alpha、颜色、定时插值（56-60）、翻转、中断标签（64）、锚点（65）、
  blend 模式（66）、render 模式（67）、图层（68）、UV 滚动（70/71）、z-write（73）、寄存器/数学指令（6-47）。
- **没有逐帧 VM**：`ScriptSprite()`（`sprite/anm.go:76-89`）每次 Draw 都从第 0 帧重放脚本（O(age)），
  且用的是敌人年龄，不是脚本被设置时刻起算的时间。
- `sprite/anm.go:68-69` 用 `SubImage` 切精灵，会被裁到纹理边界。背景精灵依赖超出纹理的 UV 环绕，
  例如 `stage01.anm` 精灵 0 为 `{x:1,y:1,w:510,h:254}`，纹理只有 256×256（横向环绕两次）。
- **bug**：`sprite/sheet.go:35-42` 的 `Sub()` 按名字缓存，而 `sprite/loaders.go:147`、`:198` 总传 `""`，
  所以 8 种中玉颜色和 16 帧大型敌人全部取到第一张切图。
- `sprite/loaders.go:36-41` 只加载 `pl00.anm`、`enemy.anm`、`stgenm01.anm`；`bullet.anm` 嵌入了但从未加载；
  `stgenm02-07.anm`、`stage0N.anm`、`front.anm`、`ascii.anm` 都不在 assets 中。
- 原作 ANM VM 已 dump：`.tmp/ecl_dump/top/func_0043ee30.c`（1927 行，92 个 case）。
  ins_66（`case 0x42`，:1594）写 2 位 blend 模式（0x30）；ins_67（`case 0x43`，:1600）写 4 位 render 模式
  （0x3c00000）。stage1 代码注释把 "ins_67(6)" 当成加法混合，可能把两者搞反了
  （`stg1bg2` 地面脚本用的是 `ins_66(1); ins_51(192)`）。blend 值 1 的确切含义待核实。

## 3. 各关背景现状

`Stage.Draw` 调用 `s.script.BgDraw(s.field)`（`stage.go:494`）。Go 源码中没有任何 3D、透视、相机、
`DrawTriangles` 或 shader 代码。

- **stage1**（`scene/stage/stage1/ecl_script.go`）：
  - 相机（:122-172）：手写关键帧，只插值 X/Y；忽略相机 Z（-470 到 -630）、朝向（ins_4/5）、
    up 向量（ins_6）、FOV（ins_7）；缓动是近似的（:184、:194）。
  - 地面（:84、:88）：两次平铺 `TileBackground`，Z=-128 层的视差系数 1.255；第 2556 帧后的相机
    X（-140）没作用到地面上。
  - 雾（:218-254）：全屏色块 + 手调 alpha。原作是按深度的线性雾：ins_8/9 设颜色与近/远距离，
    例如 `#70a0f0`，900 到 2000。
  - 云（:280-383）：STD entry 4-7 抄成 Go 字面量，带 `cloudScale=0.72`、视差 0.50/0.68、亮度 0.30 等经验系数。
- **stage2–6、Extra**：每关填一个底色，`bg1` 匀速平铺，`bg2` 以分数速度 + 低 alpha 平铺，没有用任何 STD 数据。

| 关 | 文件 | 每帧滚动 | 加载了但没画 |
|---|---|---|---|
| 2 | `stage2/stage2.go` | 2.0 | — |
| 3 | `stage3/stage3.go` | 1.8 | bg3、bg4 |
| 4 | `stage4/stage4.go` | 2.5 | bg3（stg4bg7） |
| 5 | `stage5/stage5.go` | 1.5 | —（没加载共用的 stg1bg3/4 云） |
| 6 | `stage6/stage6.go` | 1.2 | bg3、bg4、bg5 |
| Extra | `extra/extra.go` | 2.0 | bg3 |

原作脚本实际行为示例：stage4 相机倾斜、多次换雾，并通过 `ins_14(0,4)` 触发 ANM 中断；
stage6 用 up 向量 (0,0,-1)，相机在 X 和 Z 上移动，FOV 从 0.628 变到 0.98。

- **符卡背景**：`cdbg*` 从未绘制。ECL 342（`game_ops.go:376-377`）只记录了符卡名；使用 `cdbg01` 的脚本
  在 `stgenm01.anm` 中，已加载但没有任何东西去执行它。

## 4. 原作数据与工具（本地，均在 gitignore 的 `.tmp/` 下）

- `.tmp/[th10]/`：`th10.dat`（27,696,219 B）、`th10c.dat`（39,084,632 B，汉化补丁）、
  `thbgm.dat`（403,789,620 B）、`th10.exe`（v1.00a，SHA-256 与 `th10probe` 期望值一致）。
- `.tmp/th10_dat_extract/`：`th10.dat` 的解包结果，包括 `stage01-07.std`、`stage01-07.anm`、
  `stgenm01-07.anm`、`front.anm`、`ascii.anm`、`text.anm`、全部 `.msg`、`demo0-3.rpy`、`thbgm.fmt`。
- `.tmp/thtk_probe/`：`stage01-07.std.txt`。合计 35 个 entry、138 个 quad（全为 type 0）、约 429 个 face。
  用到的 STD 指令为 1–9 和 14。
- `.tmp/thtk-src/`：thtk 完整 C 源码（未编译）。STD 格式表 `thstd/thstd.c:65-106`；ANM v4 表
  `thanm/thanm.c:280-398`；THA1 解包 `thtk/thdat95.c`。

## 5. 实体精灵来源（`stage.go`）

| 实体 | 行 | 来源 | 备注 |
|---|---|---|---|
| 道具 | 543-559 | 彩色矩形 | |
| 特效 | 860-869 | 圆 | `effect.go` 没有精灵数据 |
| 激光 | 699-714 | `vector.StrokeLine` | |
| 自机 | 737-785 | ganim8 网格切 `pl00.png` | 基于时间；`anim.Update()` 在 Draw 中调用（:773） |
| 自机子弹 | 561-579 | `pl00.anm` 脚本精灵，固定 alpha 128 | |
| 副机 | 581-595 | `pl00.anm` 脚本 17，只取第 0 帧 | |
| 妖精 | 608-626 | `enemy.anm`，按 ECL 设置的脚本 id（`game_ops.go:166-188`） | 回退到硬编码 `FairyKind` 网格 |
| Boss/中 Boss | 631-651 | 总是 `stgenm01.anm` | 2–7 面用一面的文件解析脚本 id，显示错图或缺图；`AnmSlot`（`game_ops.go:164`）存了没用 |
| 敌弹 | 661-697 | `etama.png` 网格，只有 4 种形状 | `mapBulletType` 把 28 种压成 4 种 |

ECL 272/273（ANM 特效对象）为空操作（`game_ops.go:409-414`）。

## 6. 绘制顺序与混合

- 顺序硬编码在 `stage.go:494-509`：背景、道具、自机弹、敌人、敌弹、副机、自机、特效、灵击闪光、Boss HUD。
- 加法混合只出现在两处：`b.Additive` 的子弹（`stage.go:682-683`，但 ECL 路径总传 `false`，
  `bullet_ops.go:327`；激光传 `true` 却画成矢量线，标志无效），以及 stage1 的云（`ecl_script.go:16-23`）。
- 从不读取 ANM 的 blend 模式、render 模式和图层。
- `README.md:120` 描述的 `render/layer.go`（11 层渲染器）并不存在。

## 7. 还原背景所需

1. **STD 解析**：entry（位置/尺寸）、quad（ANM 脚本索引、位置、尺寸）、face/实例（位置），以及带时间戳的脚本段。
2. **STD VM**：每帧推进相机位置与定时移动（2/3）、朝向（4/5）、up 向量（6）、FOV（7）、雾与定时雾（8/9）、
   跳转（1）、ANM 中断（14），使用真实缓动（与 ECL 共用 `FUN_0044c350`）。
3. **3D 投影**：look-at 视图矩阵 + 透视投影到 384×448 场地，参数从 exe 取证；CPU 变换四角，用
   `DrawTrianglesShader` + 自定义顶点属性（u/w、v/w、1/w）做透视校正采样，`AddressRepeat` 支持 UV 环绕。
   Ebiten 默认仿射插值，大地面 quad 不做透视校正会扭曲。按 entry 顺序绘制（与原作一致，待核实）。
4. **雾**：按视深的线性雾（ins_8/9 的近/远），把颜色向雾色混合；按顶点还是按像素待取证。
5. **完整 ANM VM**（参考 `FUN_0043ee30`）：每实例状态、定时插值、UV 滚动、中断、blend/render 模式
   （3D 模式让精灵进入 STD 场景）、图层。背景、符卡背景、Boss、特效、子弹都依赖它。
6. **符卡背景**：ECL 342 启动 `stgenm0N.anm` 中的 cdbg 脚本，画在 STD 层之上。
7. **小修**：支持 UV 环绕的精灵绘制替代 `SubImage`；修 `Sheet.Sub("")` 缓存 bug；按关卡选择 Boss ANM；
   不再保留加载了却不用的纹理。

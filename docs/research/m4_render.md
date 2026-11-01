# M4 / L3 渲染架构设计（2026-11-01）

本文是 3D 背景（M4）、ANM 绘制（M3 剩余部分）和截图差分（L3）共用的基础设施设计。现状见
`presentation_gaps.md`。逻辑侧已经就绪：STD 相机 VM 与投影（`std/`，四面全章差分）、ANM VM
（`anm/vm.go`，驱动子弹与特效）。缺的是"怎么画"和"怎么验证画得对"。

分工：本文和 §2 的接口由我设计；§1 取证、§3 工具和 §4 中标注 Haiku 的实现都交给 Haiku 子代理，
我审 diff。

## 0. 约束

- 原作是 D3D9 固定管线。Ebiten（Windows 上为 DirectX 11，WASM 上为 WebGL）提供的是：
  - 2D 顶点（`DstX/DstY`）、顶点色，以及 4 个自定义 float 属性（`Vertex.Custom0–3`，v2.10 起可用）；
  - Kage 着色器、可自定义的混合方程；
  - **没有**深度缓冲，**没有**固定管线的雾、纹理阶段和裁剪（深度缓冲由 `gfx` 用离屏图模拟，见 §2.1 G3）。
- 内部图集：Ebiten 会把图片打包进图集，硬件 wrap 寻址用不了，UV 环绕必须在着色器里手工实现。
- WASM：只用 Kage 与纯 Go，绘制代码不得依赖平台 API。`anm`、`std` 两个逻辑包保持不 import ebiten
  （便于测试，也让逻辑可以在无 GPU 的 `th10sim` 里跑）。
- **像素级完全一致不现实**：光栅化规则、过滤精度、DWM 合成都有差异。L3 的目标是同帧对比的 PSNR
  阈值加差异热图，并且能把差异归因到具体的层。

## 1. 待取证（Haiku：dump 加文档，不写代码）

结果写进 `docs/research/m4_render_re.md`（简体中文）。dump 工具是 `tools/ghidra/dump.py`（可加 `--asm`）。
D3D 调用在反编译里是虚表调用 `(*(code **)(*dev + off))(dev, ...)`，`IDirect3DDevice9` 的偏移如下：

| 方法 | 偏移 |
|---|---|
| Present | 0x44 |
| BeginScene / EndScene | 0xa4 / 0xa8 |
| Clear | 0xac |
| SetTransform | 0xb0 |
| SetViewport | 0xbc |
| SetRenderState | 0xe4 |
| SetTexture | 0x104 |
| SetTextureStageState | 0x10c |
| SetSamplerState | 0x114 |
| DrawPrimitive | 0x144 |
| DrawPrimitiveUP | 0x14c |
| SetFVF | 0x164 |
| SetStreamSource | 0x190 |

常用枚举：

- **D3DRS**：ZENABLE 7、ZWRITEENABLE 14、ALPHATESTENABLE 15、SRCBLEND 19、DESTBLEND 20、
  CULLMODE 22、ZFUNC 23、ALPHAREF 24、ALPHAFUNC 25、ALPHABLENDENABLE 27、FOGENABLE 28、
  FOGCOLOR 34、FOGTABLEMODE 35、FOGSTART 36、FOGEND 37、RANGEFOGENABLE 48、LIGHTING 137、
  FOGVERTEXMODE 140、BLENDOP 171。
- **D3DBLEND**：ZERO 1、ONE 2、SRCALPHA 5、INVSRCALPHA 6、DESTCOLOR 9。
- **D3DFOG**：NONE 0、LINEAR 3。
- **D3DSAMP**：ADDRESSU 1、ADDRESSV 2、MAGFILTER 5、MINFILTER 6、MIPFILTER 7。
- **D3DTEXF**：POINT 1、LINEAR 2。
- **D3DTSS**：COLOROP 1、COLORARG1 2、COLORARG2 3、ALPHAOP 4。
- **D3DTS**：VIEW 2、PROJECTION 3、WORLD 256。

| # | 问题 | 线索 |
|---|---|---|
| R1 | 绘制链：注册函数、优先级、回调、对象，与 calc 链的对应关系 | calc 链见 `m2_motion.md` §1（`FUN_00449ae0`）；绘制回调应该由同一个调度器 `DAT_00491be4` 调度 |
| R2 | 设备与全局状态：`CreateDevice` 的参数（behavior flags 是否含 FPU_PRESERVE 0x2、后备缓冲格式）；初始化时设置的 render state、sampler state、texture stage state 默认值 | `FUN_00439890` 附近（`docs/harness.md` 的 th10.cfg 一节） |
| R3 | ANM 绘制：VM 的绘制入口，按 render mode（flags >> 22 & 0xf，ins_67）分支，每种模式的顶点构造（屏幕空间 XYZRHW 还是经变换的 XYZ）、半像素偏移、UV（精灵矩形、纹理尺寸、UV 滚动）、顶点色、blend 模式到 SRC/DEST 的映射、z-write（0x800）、过滤方式；图层（ins_68）如何决定绘制顺序 | `FUN_0043ee30` 的调用者；子弹的 6 个图层 `0x474250` |
| R4 | STD 绘制：entry、quad、face 的遍历顺序；VIEW/PROJECTION/WORLD 的设置；雾（FOGENABLE、TABLEMODE 还是 VERTEXMODE、RANGEFOG、颜色、近远）；ZENABLE/ZWRITE；op 9 雾插值（`FUN_00405040`、`FUN_004049a0`）；op 14 ANM 中断 | STD calc `FUN_00402720`、`FUN_00403c80`（`.tmp/ecl_dump/std*`） |
| R5 | 帧组合：后备缓冲尺寸、viewport 的切换（场地 32,16,384,448 与全屏）、Clear 颜色、`front.anm` 边框和 HUD 的绘制时机 | R1 的绘制链 |

## 2. 架构

### 2.1 `gfx` 包：D3D9 固定管线的最小仿真

```
gfx.Device            —— 一帧的绘制状态与提交入口，目标是 640×480 离屏图 Frame，另有同尺寸的深度图
  .SetTransform(World|View|Projection, Mat4)
  .SetViewport(Viewport{X, Y, W, H, MinZ, MaxZ})
  .SetFog(Fog{Mode, Color, Start, End, Range})  .SetFogEnable(bool)
  .SetCull(CullNone|CullCW|CullCCW)   —— 默认 CCW，与 D3D 相同
  .SetBlend(BlendMode)  —— 由 R3 的 SRC/DEST 组合枚举
  .SetTexture(*Texture)  .SetAddress(Address)  .SetFilter(Filter)  —— 纹理与采样状态分开设，与 D3D 一致（th10 换纹理不动采样状态）
  .SetZEnable(bool)  .SetZWrite(bool)  .SetZFunc(CmpLessEqual|CmpAlways)
  .SetAlphaTest(enable bool, ref uint8)  —— ALPHAFUNC 固定为 GREATEREQUAL
  .DrawTL(Primitive, []TLVertex)  —— 屏幕空间（D3DFVF_XYZRHW）：x, y, z, rhw, diffuse, specular, u, v
  .Draw3D(Primitive, []Vertex3D)  —— 物体空间（D3DFVF_XYZ）：x, y, z, diffuse, u, v，经 World·View·Proj
  .Clear(ClearTarget|ClearZBuffer, Color, z, rects...)  —— 矩形与 viewport 取交集，不带矩形时清 viewport；颜色 alpha 强制为 1
  .Flush()
gfx.Texture           —— 一张整纹理（不切 SubImage），记录宽高，UV 是纹理归一化坐标
```

**G1 已完成（2026-11-01）**：接口与契约在 `gfx/gfx.go`、`gfx/pipeline.go`、`gfx/shader.go` 的文档注释里，
规格测试在 `gfx/pipeline_test.go`（期望值手算）。以 Go 代码注释为准，本节只记设计理由。G1 定下的几点：

- **投影与逻辑位一致。** `std.Camera.Matrices()` 导出 view/proj（`std.Camera.Project` 也改用它），
  `Draw3D` 的裁剪坐标取 p·(world·view·proj) 一次乘积，透视除法与 viewport 映射按 `Project` 的分组写，
  所有乘加都包 `float32()` 防止编译器融合成 FMA。`TestProjectMatchesCamera` 检查绘制位置与 ECL 256 的投影逐位相同。
- **纹理按直通 alpha 存。** Ebiten 的图像是预乘 alpha，透明纹素的 RGB 会丢失；而 D3D 对直通颜色做双线性过滤，
  透明纹素的 RGB 会在精灵边缘显出来（原作的暗边/亮边）。所以 `NewTexture(*image.NRGBA)` 用 `WritePixels`
  原样写入字节，着色器把取到的值当直通 alpha 用。这张图只给着色器读，不能用 `DrawImage` 画。
- **剔除。** D3D 默认 `D3DCULL_CCW`，两条路径都在裁剪之后按屏幕空间绕向剔除；三角带的奇数三角形调换顶点顺序，
  整条带绕向一致。th10 是否改了剔除模式待 R2。
- **viewport 兼作裁剪框。** 批次画到 `Frame.SubImage(viewport)` 上（Ebiten 的 SubImage 保留父坐标并裁剪），
  对应 D3D9 光栅化不出 viewport。x/y 平面因此不做几何裁剪，只裁近、远平面（z ≥ 0、z ≤ w）。
- **Frame 始终不透明。** 后备缓冲是 X8R8G8B8；`Clear` 写 alpha 1，两种 blend 下 alpha 保持 1，
  预乘输出配 `BlendSourceOver`/`BlendLighter` 才与 D3D 的 SRCALPHA/INVSRCALPHA、SRCALPHA/ONE 等价。

- **两条提交路径汇成一种光栅化输入。** `Draw3D` 在 CPU 上做以下几步，产出与 `DrawTL` 相同的顶点：
  1. 变换到裁剪空间；
  2. 用 Sutherland–Hodgman 对 z ≥ 0（D3D 的近平面）做多边形裁剪，同时保证 w > 0；
  3. 透视除法；
  4. viewport 映射；
  5. 输出带 1/w 的屏幕空间顶点。

  相机与投影复用 `std/project.go` 的 float32 运算，保证逻辑侧（ECL 256、矩形消弹）和绘制侧用的是同一份矩阵。
- **着色器契约。** 全项目只有一个 Kage 着色器。

  自定义属性：

  | 属性 | 内容 |
  |---|---|
  | `Custom0` | u·q |
  | `Custom1` | v·q |
  | `Custom2` | q（1/w 或 rhw） |
  | `Custom3` | f·q（雾因子乘 q，与 uv 一样做透视校正） |

  顶点色走 `ColorR/G/B/A`。uniform：纹理尺寸、寻址模式、过滤方式、雾色、TSS 组合。

  片元计算：
  1. uv = (Custom0, Custom1) / Custom2，即透视校正；顶点雾 f = Custom3 / Custom2。
  2. 按寻址模式手工 wrap 或 clamp。
  3. 按过滤方式手工取样：point 取最近纹素，linear 取 4 点双线性；先 wrap 再取样，用 `imageSrc0Origin/Size` 换算到图集坐标。
  4. 颜色 = 纹理 × 顶点色（MODULATE；R3 如果发现其他组合，就加 uniform 分支）。
  5. 雾：mix(雾色, c, f)。
  6. 输出预乘 alpha。
- **雾。** R4 的结果：th10 用顶点雾（VERTEXMODE LINEAR、TABLEMODE NONE、无 range fog）。两种都已实现：
  - 顶点雾（VERTEXMODE）：在 CPU 上按视空间 z（RANGEFOG 时用距离）计算 f = (end − d)/(end − start)，钳到 [0,1]，放进 `Custom3`，与 D3D 一样逐顶点插值。
  - 表格雾（TABLEMODE）：在着色器里按逐像素深度计算。
- **混合。** 着色器输出预乘色：
  - SRCALPHA/INVSRCALPHA 对应 `ebiten.BlendSourceOver`；
  - SRCALPHA/ONE（加法）对应 `ebiten.BlendLighter`；
  - 其他组合用 `ebiten.Blend{}` 自定义。
- **批处理。** 纹理、混合、uniform 都不变的连续提交，合并成一次 `DrawTrianglesShader32`；任何状态变化都先 flush。
  **提交顺序严格等于原作的 draw call 顺序**（画家算法）。
- **半像素。** D3D9 的像素中心在整数坐标，Ebiten 在 +0.5。只在 `ebitenVertex`（两条路径共用的最后一步）加 0.5，其他代码不碰。
  纹理坐标两边约定相同（纹素中心在 (i+0.5)/W），不需要校正。
- **顶点色是屏幕线性插值。** 4 个自定义槽已用完，顶点色走 `ColorR..A`，没有透视校正。目前所见的绘制路径
  一个 quad 只有一种颜色，两者相同；若 R3/R4 发现透视 quad 上有渐变色，再考虑把颜色挪进自定义槽。
- **逻辑分辨率。** 整帧画在 640×480 的离屏 `Frame` 上，这是原作的后备缓冲尺寸，最后整体缩放到窗口。
  L3 截图直接读这张离屏图，不受窗口缩放影响。
- **深度缓冲与 alpha test（G3，2026-11-01 设计）。** 复核 R1–R5 时确认原作确实用了 D16 深度缓冲（m4_render_re.md §4.7）：
  pass 0–7 用 LESSEQUAL 测试并写入，层 18 和 pass 8–11 只测试不写；alpha test（GREATEREQUAL 1）决定哪些像素写深度。
  第一版"按提交顺序画、忽略 z"的做法因此作废，改为在 `gfx` 里模拟。契约在 `gfx/gfx.go` 的 `Flush`、`reserve`、`Clear`，
  `gfx/pipeline.go` 的 `conflicts`、`ebitenVertex`，`gfx/shader.go` 的 `fragment`；规格测试在 `gfx/pipeline_test.go`。设计要点：
  - **深度图。** 640×480，每像素用 R、G 两个字节存 D16 值，A = 1（Ebiten 按预乘存储，A = 1 时 RG 不变）。另有一张同尺寸的快照图。
  - **深度值放在 `SrcX`。** 4 个自定义槽已经用满；纹理由着色器自己寻址，`SrcX` 空着，而且和其他属性一样做屏幕线性插值，
    正是 D3D 对 z 的插值方式。Ebiten 会给 `SrcX` 加上 `Images[0]` 的图集原点，float32 相加会丢掉 D16 需要的精度，
    所以纹理、Frame、深度图都改为 Unmanaged（不进图集，原点 (0,0)）。这样片元的 `dstPos` 就是后备缓冲像素，着色器在同一位置读深度图。
  - **每批两趟。** 颜色趟读深度图做测试、写 Frame；批次写深度时再跑深度趟：读快照做测试，用 `BlendCopy` 写深度图，
    没通过的片元 `discard`。两趟都对"批次之前的深度"测试（Ebiten 不能读正在写的图）。
  - **拆批规则。** 既测试又写深度的批次里，任意两个三角形不能覆盖同一像素：`conflicts` 先比包围盒，再找分离边，只接触不算冲突
    （同一 quad 的两半按填充规则各得各的像素）。冲突就先 flush。这样"对批次前的深度测试"就等于 D3D 的逐三角形顺序。
    ALWAYS 加写入不需要拆批，因为 `BlendCopy` 本来就保留最后画的片元。
  - **量化。** D16 = ⌊z·65535 + 0.5⌋；alpha test 比较 ⌊a·255 + 0.5⌋ 与 ref。两处都是"四舍五入"的假设，待 L3 验证。
  - **代价。** 写深度的批次多一次 viewport 拷贝和一次 draw call。只有 STD pass 0–7 会这样；另外在第一次进关之前也会，
    因为 th10 初始化时没设 ZWRITE，取 D3D 默认值 TRUE，配 ZFUNC ALWAYS。
  - **状态拆分。** FOGENABLE 和雾参数分开设：th10 按每个 STD quad 的 render mode 切 FOGENABLE，参数不动。
    `Reset` 恢复 D3D 默认值；th10 的初始状态（FUN_00439d20）由绘制链（D1）设置。
  - **TL 顶点的雾。** 2D 批次（FVF 0x144）没有 specular。顶点雾开着时，这类顶点的雾因子暂定由调用方填 specular alpha 0xFF（即不雾化），待 L3 验证。

### 2.2 绘制链与 supervisor（D1）

契约在 `supervisor/supervisor.go`、`scene/stage/chain.go` 和 `anm/anmdraw` 的 `Reset` 里，规格测试是 `supervisor/supervisor_test.go`、
`scene/stage/chain_test.go`、`anmdraw` 的 `TestReset`。证据见 `m4_render_re.md` §1.3、§1.4、§2.4、§5.1，以及 `m4_r6.md` R6-B。

- **`supervisor` 包对应 0x491c28 的绘制部分。** 包括设备、两个相机块、当前相机（DAT_00491fac）、2D 标志（fb0）、
  FOGENABLE 缓存（DAT_00492378）、清屏色（DAT_004923a8），以及 ANM 管理器的绘制状态。
  设备和 ANM 缓存进程级唯一（`Get`）；测试用 `New` 加记录设备 `supervisortest.Rec`。
- **相机块与两种建矩阵方式。** Screen 是 0x491e94（viewport 0,0,640,480），Field 是 0x491d7c（32,16,384,448）。
  - `Flat`（FUN_00421480）用 `std.FlatMatrices`：z = 0 平面上一单位对应一像素。
  - `Perspective`（FUN_004215a0）用 `std.LookMatrices`：取块里的 look-at 相机，近远裁剪面 30/1800。
    STD 每次先把自己的相机抄进 Field 块再调用（S3）。`std.Camera.Project` 与它共用同一份相机数学。
  - 两者都先 flush ANM 批次，最后把块的 Shift 抄给 ANM。
- **一帧的顺序。**
  1. `BeginFrame`（FUN_00439390 在链前做的事）：flush，`Use(Screen)`，Is2D，雾缓存置为未知并关雾。
  2. draw 链的 41 个节点按优先级依次执行，每个节点先 `enter`（supervisor 部分），再 `draw`（关卡内容）。
  3. `EndFrame`：flush，`SetTexture(nil)`（只设设备，管理器的纹理缓存不动），`Dev.Flush`。

  有 supervisor 动作的节点：

  | 节点 | 函数 | enter |
  |---|---|---|
  | 1 | `FUN_00420000` | `Start`：`ANM.Reset`（纹理、blend、过滤、顶点类型、tint、Shift 全部复位）→ `Select(Screen)` → `Perspective(Screen)` → `SetViewport` → Is2D → `Clear(TARGET, ClearColor, 1)`。Screen 的 Facing 为零，视图矩阵退化，只影响 3D 绘制 |
  | 0xf | `0x448620` | `EnterField`：`Use(Field)`，Is2D = 0，关雾 |
  | 0x24 | `FUN_00448740` | `Unshake`：Field.Shift 与 ANM.Shift 清零，不 flush（批次里的 quad 生成时已经加过平移） |
  | 0x29、0x2a、0x2d | | `Enter2D`：`Use(Screen)`，Is2D = 1 |
  | 0x32 | `FUN_004200d0` | `Finish`：flush，Field.Shift 清零 |

  其余节点没有 supervisor 动作。0x28 的回调直接返回 1。
- **震屏。** `FUN_0043c5ca` 在 calc 阶段写 Field.Shift（DAT_00491e64/68，每轴随机取 0、+f、−f）。
  - 因此 0xf 到 0x23 的节点画在震动后的位置；0x24 以后的层 14–16 和 GUI 不震。
  - STD 会把 Field.Shift 抄进自己的相机，所以背景也震。
  - 项目原来平移整张场地图，GUI 以外全部跟着震；改后与原作一致。
  - 震屏逻辑移植之前，`Stage.Draw` 每帧在 `BeginFrame` 之前把 `ShakeOffset()` 写进 Field.Shift，代替 calc 里的写入。
- **清屏色（DAT_004923a8）。**
  - 进程开始时为 0。
  - 关卡开始时 `FUN_00417c80` 设为 0xff000000。DAT_00474ca0 bit 0 置位时设为 0，但 `Clear` 会把颜色变成不透明，画面相同。
  - STD 的雾指令（`FUN_00403c80`）把它设为雾色（S2）。
  - 深度缓冲不在 `Start` 里清，由 STD 清（S3）。
- **legacy 节点。** 还没移植的节点用 Ebiten 画进 back buffer。
  - `sv.Legacy()` 先 flush ANM 批次和设备，再返回 Frame。Ebiten 对同一张图按提交顺序执行，所以旧画法落在链里的正确位置。
  - 画的目标是 Frame 在**当前相机 viewport** 内的 SubImage，裁剪效果等同原作的 viewport。
  - 坐标沿用屏幕坐标：Ebiten 的 SubImage 与原图共用坐标系。
  - 偏移取 `sv.ANM.Shift`，也就是此刻 ANM quad 会加上的平移。

  D1a 的节点分配：

  | 节点 | 内容 | 依据 |
  |---|---|---|
  | 7 | `BgDraw` 画到离屏的场地图，再贴到 (32, 16) + Field.Shift，裁到场地区域 | STD 背景；STD 会抄 Field.Shift |
  | 9、0xb、0xd | World 层 0、1、2 | §1.3 |
  | 0xa | World 层 17、18 | `FUN_00402ca0` 先画层 17、18，再画 pass 8–11 |
  | 0xf、0x10–0x13、0x15、0x17、0x18、0x1a、0x1c、0x21、0x24、0x29、0x2a | World 层 3–16 | §1.3 |
  | 0x14 | 敌人 | |
  | 0x16 | 自机子弹、子机、自机、特效 | 这个顺序是猜的，等 R6-F 追踪调用点 |
  | 0x19 | 道具 | |
  | 0x1b | 激光 | |
  | 0x1d | 子弹 | |
  | 0x2b | Boss HUD、对话、front 框、HUD 文字 | GUI 节点；内部顺序沿用项目现状 |
  | 0x2e | 暂停遮罩 | 猜测，`FUN_00422150` 待查 |

  S3 之前，层 0–2、17、18 在 `Start` 设的 Screen 相机下画：不裁到场地，也不震。
  原作里 STD 节点会调用 `Perspective(Field)`，所以至少 ANM.Shift 会变成 Field.Shift；当前相机是否也切换，由 S3 取证。
- **D1 拆分。**
  - **D1a**：supervisor 实现、`anmdraw.Reset`，`Stage.Draw` 改为 `drawFrame` 加 legacy 节点。
  - **D1b**：World 层改用 `ANM.DrawLayer`。需要 `NewSheet`（R6-A），mode 0–3 的部分可以在 A1b 之前做。
  - **D1c**：实体节点（敌人、子弹、激光、道具、自机）改用 `anmdraw`，依据 R6-F 的调用点。
  - **D1d**：GUI，即 front.anm，以及 `FUN_00415af0` → `FUN_00415800` 的 VM 数组。
  - P2（`-shot`）排在 D1a 之后，直接截 Frame。

### 2.3 ANM 绘制

新包 `anm/anmdraw` 移植 ANM 管理器（DAT_00491c10）的绘制部分。契约在 `anm/anmdraw/anmdraw.go` 的文档注释里，以那里为准；
本节只记设计理由。

- **`Renderer` 对应管理器的绘制状态。**
  - 原作把 2D quad 攒成一个 TL 三角形列表批次。纹理、blend、过滤、顶点类型这几项状态各有一个缓存，值变了才设置设备，设置前先提交批次。
  - `Renderer` 保留同样的缓存，按同样的顺序调用设备，所以 `gfx.Device` 收到的状态序列与原作逐条相同。
  - 设备只通过一个四个方法的接口（`SetTexture`、`SetFilter`、`SetBlend`、`DrawTL`）访问，规格测试用记录调用的假设备比较调用序列。
  - 其他直接画设备的代码要先 `Flush`、画完 `Invalidate`，对应原作 `FUN_0043bda0` 与 `FUN_00439d20` 的缓存失效。D1 在原作的同一位置调用。
- **暂存 quad 跨调用保留。** 原作的暂存 quad 是全局变量（DAT_004978c0），锚点 3（ins_65）不写对应轴，沿用上一个 quad 的角点。
  `Renderer.quad` 照搬这一点。旋转路径在锚点 3 时读的是未初始化的栈，直接 panic。
- **浮点分组照抄。** 加法的结合顺序按汇编写在注释里（例如 x 锚点 1 的右边是 `((Pos.x + Offset.x) + w) + Pos2.x`），
  乘积包 `float32()`。规格测试在 2^24 附近验证分组。mode 2 的 z 读的是 Pos2.y（0x0044343b），这是 ZUN 的 bug，保留。
- **精灵。** `Sprite` 是运行时精灵记录（0x44 字节）里用到的字段：纹理、UV 矩形、宽高。
  `NewSheet` 按 R6-A 构造（`m4_r6.md`）：每个 entry 一张纹理（同名 entry 也各自一张），记录按 float32 公式计算，无半像素偏移。
  TH10 的 entry 全是 2 的幂，D3DX 不补边；格式只有 1、3、5，没有 Gray8；格式 3 的空白处是不透明黑。
- **范围。** A1a 是 render mode 0–3（2D quad）；mode 4–9 是 A1b，按 R7 拆成：mode 8（STD 背景和 stgenm04 层 17 用，S3 依赖它）先做，
  然后 mode 6/4、mode 9。R6 的更正：D3D9 在 COUNT2 下纹理矩阵的平移有效。
- `sprite.DrawVM` 迁移期保留，D1 把全部节点迁走后删除。

### 2.4 STD 场景

- `std` 补全解析：entry 和 quad（ANM 脚本号、位置、尺寸）、face（entry 号、位置）。
- 逻辑（`m4_render_re.md` §4）。契约在 `std/objects.go` 与 `std/vm.go`，以那里为准：
  - **S2a**：
    - 每个 quad 一个 `anm.VM`（数组 +0x17c，按 `FUN_00404450` 的顺序：entry 顺序，再 quad 顺序）。
    - 8 个槽位 VM（+0x180）由 op 14 启动；脚本号 < 0 时只清可见位，VM 照常运行。
    - entry 循环 `FUN_00403990`：没有运行中 quad 的 entry 被清掉，没有 quad 的 entry 在第一帧就清掉。
    - 每帧顺序：entry 循环 → 脚本 → 8 个槽位。
    - stage0N.anm 的脚本不抽随机数，所以放进 sim 不影响确定性。
  - **S2b**：op 9 的 7 分量雾插值（`FUN_00405040`、`FUN_004049a0`）。还缺 `FUN_00404d40`、`FUN_00404da0`、`FUN_00404e10`、`FUN_00463b2c` 的 dump。
  - 不移植：+0x1eec 非零时步进的三个特殊 VM（写入者未追到）、VM +0x3a0/+0x3a1 两个置 0x10 的字节（读者未知）。
  - probe 也可以加采样相机块和雾参数（stage+0x2a4c），在 L3 之前先用 L1/L2 对上。
- 绘制：新包 `std/stddraw` 设置 VIEW、PROJECTION 和雾，再按原作的遍历顺序对每个 quad 调 `anmdraw` 的 3D 模式。

## 3. L3 截图差分

### 3.1 原作侧：`th10probe -shot-frames=F1,F2,… -shot-dir=D`

- **仍然只读，且不与原作进程交互。**
  - 用 `EnumWindows` 加 `GetWindowThreadProcessId` 按 PID 找到 th10 的顶层窗口。
  - 用 `GetClientRect` 和 `ClientToScreen` 求出客户区在屏幕上的位置。这三个都是只读查询，不会给目标窗口发消息。
  - 从屏幕 DC（`GetDC(NULL)`）`BitBlt` 出这块区域，用 `CAPTUREBLT`，取的是 DWM 合成后的桌面，存成 PNG。
  - 不用 `PrintWindow`：它会给目标线程发 `WM_PRINT`，属于与进程交互。
  - 代价是原作窗口必须可见、不被遮挡，截图前检查窗口没有最小化。
  - 不发送任何输入，不写进程内存。
- **帧对齐。** 采样循环读到帧计数等于目标帧时截图，截图前后各读一次帧计数，写进文件名（`f<前>-<后>.png`）。
  画面比逻辑晚 0–2 帧（Present 队列加 DWM 合成），对齐交给比较器处理。
- **P1 已实现（2026-11-01）**：`tools/th10probe/shot.go`（帧列表解析、后台写 PNG）、`shot_windows.go`（GDI 截图）。
  只接受客户区 640×480 的窗口；每次截图都重新枚举窗口，截图帧附近的 trace 可能多丢一两帧，可以接受。
  截图失败只打印一行并跳过该帧，结束时列出没截到的帧。GDI 路径只编译过，待用户实测。
- **原作设置。** 窗口模式（th10.cfg +0x1d = 1）、32 位色（+0x1a = 0），由用户设置并运行。

### 3.2 复刻侧：游戏本体的 `-shot` 模式

Ebiten 没有真正的无头模式。所以截图由游戏本体完成，这样保证用的是同一套绘制代码：

```
th10 -replay X -chapter N -shot-frames F1,F2,… -shot-dir D
```

- 以最高速推进：每次 Update 跑多帧逻辑，只在目标帧附近 Draw。
- 每个目标帧存 F−2 到 F+2 五张截图，读 Frame 的 `ReadPixels`。
- 同时存一张**前景掩码**：只画非背景层（实体、弹幕、HUD）时的 alpha，用于给背景单独评分。
  逻辑已经位精确，所以复刻的掩码可以直接套到原作截图上。

### 3.3 比较器 `tools/th10pix`

- 对每个原作截图，在复刻的 F±2 里选 MSE 最低的一张作为对齐帧，并报告选中的偏移。偏移应当是稳定的常数，它本身也是一项检查。
- 报告内容：
  - 整体 PSNR；
  - 分区域 PSNR（场地、侧栏）；
  - 背景 PSNR（用掩码排除前景）；
  - 差异热图 PNG。
- 先只报告、不判失败。收敛后再在 `docs/harness.md` 定 L3 阈值。

## 4. 任务拆分与顺序

| # | 任务 | 执行 | 验证 |
|---|---|---|---|
| R1–R5 | §1 取证（另加 R4a–c：quad +0x06 的 VM 序号、entry +0x03 bit 0、face 遍历） | Haiku | 我审文档 |
| P1 | probe 截图（§3.1）——**已完成** | Haiku | `go vet`、单元测试；用户实测 |
| S1 | STD 完整解析（§2.4）——**已完成** | Haiku | 与 thtk 输出逐项一致：35 个 entry、138 个 quad、429 个 face |
| G1 | `gfx` 接口与着色器契约（§2.1）——**已完成** | 我 | `gfx/pipeline_test.go` 规格测试 |
| G2 | `gfx` 实现：裁剪、viewport、Kage、批处理——**已完成** | Haiku | 规格测试全过；`go test -tags gpu ./gfx/` 9 个用例用 CPU 参考光栅化逐像素比较（容差 2/255，实测最大 1.1/255），另做了 3 个变异检查 |
| G3 | `gfx` 深度缓冲与 alpha test（§2.1）——**已完成** | Haiku | `go test ./gfx/` 规格测试全过；GPU 用例覆盖遮挡（两种提交顺序，含同一批次）、深度相交、alpha test 与写深度、只测不写、Clear 矩形、偏移 viewport、D16 量化、FOGENABLE；另做变异检查。结果：规格测试全过，新增 8 个 GPU 用例，变异检查均被捕获 |
| P2 | 复刻 `-shot` 模式（§3.2） | Haiku | 截图可复现 |
| P3 | `th10pix`（§3.3）——**已完成** | Haiku | 单元测试（已知偏移、已知 PSNR） |
| A1a | `anmdraw` render mode 0–3（§2.3）——**已完成**：契约与规格测试（我），实现（Haiku）；剔除的 NaN 行为按汇编更正（`m4_render_re.md` §3.9）；ins_65 锚点 3 在 46 个 .anm 的 597 条指令中未出现 | Haiku | `go test ./anm/anmdraw/`：锚点、取整与半像素、分组、旋转、裁剪、状态缓存与提交顺序、颜色与 tint、UV 滚动、Shift、Invalidate；另对照汇编核对分组 |
| R6 | A1b 之前的取证：A 运行时精灵的构造；B 管理器 +0x5c/+0x60（Shift）的写入者；C VM +0x4c/+0x50 的写入者、flag 0x4000、bit 1；D mode 8/9；E mode 4–7；F `FUN_004451c0` 的其他调用者；G VM+0x39c 绘制前回调的写入者与语义——**已完成**（`m4_r6.md`）。B 的帧块写入者由我补上：`FUN_0043c5ca` 写 Field.Shift（§2.2） | Haiku | 我审文档 |
| R7 | render mode 4/6/8/9 的绘制路径取证（`m4_r7.md`） | Haiku | 我审文档 |
| A1b | `anmdraw` render mode 4–9（依赖 R6、R7）：先 mode 8（S3 依赖），再 6/4、9 | Haiku | 规格测试；L3：背景、子弹、特效 |
| S2a | quad 与槽位的 ANM VM、op 14、entry 循环（§2.4）：契约与规格测试（我），实现与嵌入 stage0N.anm（Haiku）——**已完成**，游戏和 sim 在关卡开始时 `Enable` | Haiku | `std/objects_test.go`：参考 VM 逐帧比较；七面实际数据：quad 顺序、四面与六面 op 14 的帧、RNG 不动。`go test ./...` 全过 |
| S2b | op 9 雾插值（依赖 dump） | Haiku | 单元测试；可选 probe 采样 |
| S3 | `stddraw`（依赖 R4、G3、A1b） | Haiku | L3：各面背景 PSNR |
| D1a | supervisor 与绘制链（§2.2）：契约与规格测试（我），实现与 `Stage.Draw` 迁移（Haiku）——**已完成**，待用户目视确认 | Haiku | `go test ./supervisor/ ./scene/stage/ ./anm/anmdraw/`；画面与迁移前相同，除了震屏只作用到 0x24 之前的节点 |
| D1b | World 层改用 `ANM.DrawLayer`（`NewSheet` 已完成） | Haiku | 规格测试；L3 |
| D1c | 实体节点改用 `anmdraw`（依赖 R6-F、A1b） | Haiku | L3 |
| D1d | GUI 与 front.anm | Haiku | L3 |

可以并行的第一批：R1–R5、P1、S1，三者改的文件互不重叠；我同时做 G1。

## 5. 风险

1. **深度缓冲**：原作确实用了深度缓冲，`gfx` 的模拟方案见 §2.1 G3。剩余风险有两点：D16 和 alpha 的舍入方式；拆批过多时的性能。
2. **过滤与取样**：手工双线性要与 D3D9 的取样点约定（纹素中心 +0.5）一致，否则整张背景会偏半个纹素。
3. **截图色彩**：16 位色模式会引入抖动，必须用 32 位；屏幕缩放必须是 100%（否则桌面截图被缩放）；
   夜间模式和 DWM 色彩管理关掉；原作的 gamma ramp 待 R2 确认。
4. **MSAA**：原作若开多重采样（R2），边缘会有差异，比较时只看背景 PSNR。

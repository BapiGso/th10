# 表现层：背景（STD）、ANM、渲染的偏差审计（2026-10-26）

当前结论：七面默认已经使用 STD 脚本驱动的 3D 场景、相机／雾与 quad ANM；gfx／anmdraw／supervisor 和
World 层表均已实现。旧“没有 STD／逐帧 ANM／3D／shader”的结论已经过时。

> 2026-11-08，t9 按 t4 第 5 阶段审计与第一轮合并代码更新现状。STD／渲染依据为 `FUN_00402720`／`FUN_00403a30`／`FUN_004451c0`；t1 自机依据 `FUN_004247f0`／`FUN_00426cf0`，t2 符卡依据 `FUN_00409280`／`FUN_00408d60`，t3 GUI／MSG 依据 `FUN_00415800`／`FUN_00415e90`，t5 菜单依据 `FUN_00422660`。原审计中的手写背景仅作 legacy 备用描述，不是默认绘制；像素与听音仍待用户。

## 1. 屏幕与场地

- `game/game.go:13-23`：640×480，场地 384×448，偏移 (32,16)，与原作一致。
- 默认整帧经 `supervisor`／`gfx` 画到 640×480 Frame，按原 draw 链切场地／全屏 viewport；原 384×448 场地图只在 legacy 背景回退时使用。
- t3 已用 front.anm 的 World VM 绘制框／标签，旧 ebitenui HUD 与 PNG 对话叠层删除；不能再把 `front00s.png` 描述为默认关卡边框。
- `gfx` 已实现 point／linear 过滤、wrap／clamp、混合、雾、透视采样、深度与 alpha test；实际 D3D9 像素差异仍待 L3。

## 2. ANM

- `anm/loader.go` 解析原容器／精灵／脚本，`anm/texture.go` 解码 THTX，嵌入 PNG 与原贴图校验；不是故意忽略纹理的旧路径。
- 完整 `anm.VM`／`World` 已执行位置、旋转、缩放、alpha／颜色、插值、中断、锚点、blend／render／图层、UV 与数学指令；`resolve.go` 的时间线子集仍是静态辅助，不能代表运行时全部能力。
- `anmdraw.NewSheet` 与 gfx 的 wrap shader 支持超范围 UV，不再把 STD quad 裁成 SubImage；旧空名字缓存／统一 stgenm01 的敌人路径已删除。
- 已实现渲染模式 0–4、6、8、9；5／7 未移植，模式 10 粒子仍 panic。ins_66 是 blend、ins_67 是 render mode，二者不混同；模式与原设备状态见 `m4_render_re.md`／`m4_r7.md`。
- 原 bullet／enemy／pl00／pl01、stage01–07／stgenm01–07 均已嵌入；t3／t5 又接 front／ascii／text／七面 logo，不能保留“front／ascii 不在 assets”说法。旧 17 文件模式调查不自动覆盖新增文件，见 `m4_modes.md`。
- t1 本体为角色级共享文件的嵌入 VM，判定点／被弹为 World VM；t2 已接环／闪光／1–6 面 cut-in，t3 GUI／MSG 和 t5 菜单也使用 VM。仍是静态图片的副机／自机子弹归第二轮 t6，cdbg 归 t7，字体光栅化仍近似。

## 3. 各关背景现状

七面默认由 `std/stddraw` 的 `Drawer.Node7`／`Drawer.Node0A` 在 0x07／0x0a 按 pass 绘制；相机、透视、雾、深度与
ANM quad 均经 `std`／`supervisor`／`gfx`，不是平铺 PNG。`BgDraw` 只在 `TH10_LEGACY_BG=1` 或缺少内嵌 STD 时回退，
以下手写滚动／云参数描述仅指这个 legacy 备用路径，不能作为当前默认背景缺口。

- **stage1**（`scene/stage/stage1/ecl_script.go`）：
  - 相机（:122-172）：手写关键帧，只插值 X/Y；忽略相机 Z（-470 到 -630）、朝向（ins_4/5）、
    up 向量（ins_6）、FOV（ins_7）；缓动是近似的（:184、:194）。
  - 地面（:84、:88）：两次平铺 `TileBackground`，Z=-128 层的视差系数 1.255；第 2556 帧后的相机
    X（-140）没作用到地面上。
  - 雾（:218-254）：全屏色块 + 手调 alpha。原作是按深度的线性雾：ins_8/9 设颜色与近/远距离，
    例如 `#70a0f0`，900 到 2000。
  - 云（:280-383）：STD entry 4-7 抄成 Go 字面量，带 `cloudScale=0.72`、视差 0.50/0.68、亮度 0.30 等经验系数。
- **stage2–6、Extra 的 legacy 背景**：底色与 bg1／bg2 平铺，不使用 STD；默认路径已经使用原 STD，表中滚动量不是默认渲染参数。

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

- **符卡背景仍未接**：t2 已移植 ECL 宣言的奖励／计时与已知 World 环／cut-in，342 不再只存名字。
  cdbg 是 `DAT_004776f4+0x10/+0x3bc` 两个内联 VM，由 `FUN_00404f30` 启动、calc 0x16 步进、draw 0x0e 绘制，不是 World。
  当前只留步进接口，没有初始化／draw；符卡计时 ≥ 60 隐藏 STD 后会缺背景。现成 dump 已补齐，第二轮 t7 处理中。

## 4. 原作数据与工具（本地，均在 gitignore 的 `.tmp/` 下）

- `.tmp/[th10]/`：`th10.dat`（27,696,219 B）、`th10c.dat`（39,084,632 B，汉化补丁）、
  `thbgm.dat`（403,789,620 B）、`th10.exe`（v1.00a，SHA-256 与 `th10probe` 期望值一致）。
- `.tmp/th10_dat_extract/`：`th10.dat` 的解包结果，包括 `stage01-07.std`、`stage01-07.anm`、
  `stgenm01-07.anm`、`front.anm`、`ascii.anm`、`text.anm`、全部 `.msg`、`demo0-3.rpy`、`thbgm.fmt`。
- `.tmp/thtk_probe/`：`stage01-07.std.txt`。合计 35 个 entry、138 个 quad（全为 type 0）、约 429 个 face。
  用到的 STD 指令为 1–9 和 14。
- `.tmp/thtk-src/`：thtk 完整 C 源码（未编译）。STD 格式表 `thstd/thstd.c:65-106`；ANM v4 表
  `thanm/thanm.c:280-398`；THA1 解包 `thtk/thdat95.c`。

## 5. 当前实体与 GUI 来源

| 实体 | 节点／路径 | 来源 | 剩余边界 |
|---|---|---|---|
| 道具 | 0x19 | item VM／bullet.anm，`FUN_0041b8e0` | 特效 0x189、直接道具采样与 L3 |
| 特效 | World 层 | 原死亡／擦弹／消弹 ANM，t1 被弹 0x162／0x163 | `entity/effect` 圆池已删除，不能再列为默认来源 |
| 激光 | 0x1b | beam／emitter VM，`FUN_0041d7c0`／`FUN_0041ea40` | 冻结位、直接激光 trace 与像素 |
| 自机 | 0x16 直接 ANM.Draw | t1 的 pl00／pl01 嵌入 VM，`FUN_00426360` | 本体已迁移；整个节点仍 legacy，矩形外部门控待 t6 |
| 自机子弹 | legacy 0x16 | 旧 SHT 脚本静态图／固定 alpha | World 双句柄、命中存活期、层 8 顺序归 t6 |
| 副机 | legacy 0x16 | 旧脚本 17 静态图 | 两个 World VM／中断／其余机体归 t6 |
| 妖精／Boss／中 Boss | World 槽 VM | `ecl/visual.go` 按 ANIM 文件表选择 enemy／stgenm0N | 已有方向／蓝色受击，原本体碰撞仍缺，L3 待确认 |
| 敌弹 | 0x1d | bullet.anm VM 与原弹种表 | 不再四种形状；冻结位与 L3 |
| GUI／符卡数字／MSG | 0x2b 数组、0x25 数字及 World 层 | t3 的 front／ascii／text／logo／立绘 VM | 真实历史／结算提示接线归 t7，字体与控制器适配待验证 |
| 关卡内菜单 | 层 15／16 与 0x2e | t5 的 front／ascii／捕获纹理 | 非暂停；捕获时机与小字库待用户 |

ECL 272／273 已由 D1c-2 生成自由 VM，依据 `FUN_0040e770` 与 `ecl/visual.go`，不是空操作。

## 6. 绘制顺序与混合

- 当前 `scene/stage/chain.go` 按原 41 个 draw 节点执行，World 层由 `sv.ANM.DrawLayer` 绘制；实体三节点直接 ANM.Draw，0x14 空桩，不额外画敌人。
- t3 的 0x23／0x25／0x2b 与 t5 的 0x2e 已接且非 legacy；当前 legacy 标记仅剩 0x16，旧暂停遮罩在链外，手写背景只是显式回退。
- blend／render／图层／UV／过滤状态按 `FUN_004451c0`／`FUN_004423e0`／`FUN_004425a0` 消费，不是只有旧两处加法混合。t1 的 Rect 在原调用点 flush／失效，依据 `FUN_0043bda0`。
- 第一轮合并后完整 smoke 已通过，不再被 Rect 桩或绘制列表阻塞；没有做像素 diff，也没有证明 cdbg 未接路径已完成。

## 7. 剩余表现层工作

1. 第二轮 t6 的副机／自机子弹、其余五机体、灵击余项与矩形门控；t7 的 cdbg／宣言 VM／Extra／337／344／符卡 GUI 接线。
2. 用户按同帧截图核对七面 STD、实体、GUI／MSG／菜单与字体；通过后再删除 `TH10_LEGACY_BG=1` 的备用背景，不因 smoke 通过自动删除。
3. 继续确认 D3DX／D16／alpha／过滤的像素舍入与捕获时机；STD 特殊路径／冻结等仍见各研究文档。
4. t8 的模式／最终通关控制与应用级备用 RNG 是接线工作，不是重新实现已完成的 STD／ANM／gfx。

已完成的 STD 解析／VM／雾、透视采样、UV 环绕、逐帧 ANM、敌机文件选择与原 draw 链不再列为待实现。

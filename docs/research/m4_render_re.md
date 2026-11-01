# M4 渲染取证：绘制链、设备、ANM/STD 绘制、帧组合（2026-11-01）

依据 th10.exe v1.00a 反编译（`tools/ghidra/dump.py`，专用项目 `%LOCALAPPDATA%/th10-ghidra-render`）。
问题定义见 `m4_render.md` §1（R1–R5），calc 链格式见 `m2_motion.md` §1。

约定：

- 证据路径都在 `.tmp/ecl_dump/` 下；`render/`…`render7/` 为本次新增，目录清单见 §8。
- `+0xNN` 为相对对象基址的字节偏移；"dword 下标" 另注。
- `D3D` 方法用 vtable 偏移表示，例如 `SetRenderState` = `0xe4`。
- 反编译里 `unaff_` / `extraout_` 等变量是寄存器传参的残留，凡涉及寄存器传参的结论都以 `.asm` 为准。
- 标注 **（推测）** 的是从代码结构推出、未逐条核实的内容；列在 §7 的开放问题里。

## 0. 结论速览

| 问题 | 结论 |
|---|---|
| R1 | 一个调度器对象 `DAT_00491be4`，内含两条侧链表：calc 链（+0x14 哨兵，`FUN_00449c00` 执行）和 draw 链（+0x38 哨兵，`FUN_00449d40` 执行），都按优先级升序。每帧先跑全部 calc，再在 BeginScene/EndScene 之间跑 draw。draw 只在帧跳过计数允许时执行。 |
| R2 | `Direct3DCreate9(0x20)`；窗口 640×480；后备缓冲 X8R8G8B8（独占全屏）或 UNKNOWN（窗口）；D16 深度缓冲；无 MSAA；先 HW 顶点处理（0x40），失败回退软件（0x20）；不设 FPU_PRESERVE。初始 render state 见 §2.4。 |
| R3 | ANM 绘制入口 `FUN_004451c0`，按 `flags>>22 & 0xf` 分 10 个分支（0–9）。2D 四角走批处理（TRIANGLELIST，28 字节 TL 顶点，FVF 0x144）。混合只改 DESTBLEND（SRCBLEND 固定 SRCALPHA）。图层由 `VM+0x20`（ins_68）决定，绘制顺序由 draw 链优先级决定。 |
| R4 | quad 的 VM 索引（`quad+0x06`）由关卡初始化函数 `FUN_00404450` 按文件顺序写入（从 0 开始，全局计数）。entry 的 bit0 也在这里置位，每帧 `FUN_00403990` 在无活动 VM 时清除。绘制：pass 0–7 在 `FUN_00402850`（优先级 7），pass 8–11 在 `FUN_00402ca0`（优先级 10），两者之间夹着游戏层 3D 层。雾为 VERTEXMODE LINEAR，近远与颜色取自相机块 +0xfc/+0x100/+0x114。 |
| R5 | 后备缓冲 640×480；帧视口 (0,0,640,480)；场地视口 (32,16,384,448)；STD 绘制用场地矩形清颜色（颜色为雾色 `+0x2b60 & 0xffffff`）并清深度；每帧一次 Present；60 fps 节流。 |

---

## 1. R1 绘制链

### 1.1 调度器、节点与运行器

- **调度器**：`FUN_00449aa0` 构造 0x48 字节对象，全局 `DAT_00491be4`（`render/func_438ad0.c`：`DAT_00491be4 = FUN_00449aa0()`）。
  - calc 链：哨兵元素在 +0x14，头指针 +0x18（`m2/func_449aa0.c`，`m2/func_449ae0.c`）。
  - draw 链：哨兵元素在 +0x38，头指针 +0x3c（`m2/func_449aa0.c` 中 `in_EAX[0xe] = in_EAX + 9`；`m2/func_449b70.c`）。
- **节点**：0x24 字节，`FUN_00449ed0(callback)` 分配（`render2/func_449ed0.c`）。
  - `+0x0` 优先级键（插入时由 EDI 写入）；
  - `+0x4` 标志：bit0 由分配置位；**bit1 = 启用**，运行器只跑 bit1 为 1 的节点；
  - `+0x8` 运行回调，返回值决定运行器行为；
  - `+0xc` 插入前回调（`FUN_00449ae0`/`FUN_00449b70` 中 `unaff_ESI[3]`），非空时先调用；
  - `+0x10` 删除回调（运行器 case 7）；
  - `+0x20` 对象指针（`this`）。
- **插入**：`FUN_00449ae0`（calc）与 `FUN_00449b70`（draw）都按 EDI（优先级）升序。从头扫描，跳过 key 小于新 key 的节点，插在第一个 key ≥ 新 key 的节点之前，所以同优先级时**后注册者排在前面**（`m2/func_449ae0.c`，`m2/func_449b70.c`；与 `m2_motion.md` §1 一致）。
- **运行器**：`FUN_00449c00`（calc，头 +0x18）与 `FUN_00449d40`（draw，头 +0x3c）共用循环体（`m2calc/func_449c40.c` 为同一循环的单独导出；`render2/func_449c00.c`、`render2/func_449d40.c`）。对每个节点：
  - `+0x8` 为空则跳过；bit1 为 0 则跳过；
  - 调用 `+0x8`，按返回值：0 → 用 `FUN_00449f60` 摘除并继续；2 → 继续；3/4/5 → 停止本轮；6 → 从头重扫；7 → 调用 `+0x10` 后继续（**（推测）** 分支细节以 `m2calc/func_449c40.c` 为准，draw 运行器未逐分支核对）。
- **摘除**：`FUN_00402440`（STD 析构，`render3/func_402440.c`）对 STD 的三个节点调用 `FUN_00449f60` 摘除。

### 1.2 calc 链（格式同 `m2_motion.md` §1）

| 优先级 | 注册函数 | 全局指针（大小） | calc 回调（+0x8） | 说明 |
|---|---|---|---|---|
| 1 | `FUN_00420470` | — | `0x41ff80` | 启动；+0xc 为 `LAB_004201b0`（插入前回调，见下方勘误） |
| 3 | `FUN_0041fac0` | `DAT_00477820` | `0x41feb0` | |
| 4 | `FUN_00401110` | `DAT_004776e0`（0x89ac） | `0x4014f0` | |
| 6 | `FUN_0042caa0` | | `0x42d2e0` | |
| 8 | `FUN_00445900` | 游戏对象 `DAT_00491c10`（0x732460） | `0x4485e0` → `FUN_00448900` | 第 19 层表的构造（§3.4） |
| 9 | `FUN_00422150` | `DAT_00477830`（0x2c8） | `0x422a90` | |
| 0xc | `FUN_00402230`（STD） | STD `DAT_004776e4` / `e8`（0x2b64） | `0x403050` → `FUN_00402720` | STD 每帧：相机、VM（§4.3）。`FUN_00402640(..., 0)` 时优先级 = EBP+0xc = 0xc（`render/func_402640.asm`） |
| 0xf | `FUN_0042b4d0` | `DAT_00477840`（0xb884） | `0x42b9a0` | |
| 0x10 | `FUN_004247f0` | 自机 `DAT_00477834`（0x4478） | `0x426500` | |
| 0x11 | `FUN_004055c0` | `DAT_004776ec`（0x48） | `0x405840` | |
| 0x12 | `FUN_0040d280` | 敌人 `DAT_00477704`（0x68） | `0x40d810` | |
| 0x13 | `FUN_0041c120` | 激光 `DAT_0047781c` | `0x41c480` | |
| 0x14 | `FUN_00405e20` | 子弹 `DAT_004776f0`（0x3e0b54） | `0x406770` | |
| 0x15 | `FUN_0041ad90` | 道具 `DAT_00477818` | `0x41ba00` | |
| 0x16 | `FUN_004089c0` | `DAT_004776f4`（0x37b0） | `0x409220` | |
| 0x17 | `FUN_0040ae70` | `DAT_004776fc` | `0x40b050` | |
| 0x18 | `FUN_00413980` | GUI `DAT_0047770c`（0x9ed0） | `0x415ae0` | 加载 `front.anm`（§5.4） |
| 0x19 | `FUN_00418e30` | `DAT_00477814` | `0x4198a0` | |
| 0x1a | `FUN_00445900` | 游戏对象 | `0x4485d0` → `FUN_00448860` | 活动 VM 步进并挂入层表（§3.4） |
| 0x1c | `FUN_0040b560` | `DAT_00477700` | `0x40ba70` | |

**勘误（对 `m2_motion.md` §1）**：优先级 1 的运行回调是 `0x41ff80`（`m2reg/func_420470.c`：`FUN_00449ed0(FUN_0041ff80)`，`+0xc = LAB_004201b0`）。`m2_motion.md` 中写的 `0x4201b0` 是插入前回调（`+0xc`），不是每帧运行的回调。`m2_motion.md` 已于 2026-11-01 更正。

### 1.3 draw 链（`FUN_00449b70`，列表 +0x38）

| 优先级 | 注册函数 | 回调（+0x8） | 目标 | 说明 |
|---|---|---|---|---|
| 1 | `FUN_00420470` | `0x420000` | | 启动：ANM 缓存复位、选 2D 帧 `0x491e94`、`FUN_004215a0`（朝向为零，退化视图）、SetViewport、`Clear(TARGET, DAT_004923a8, 1.0)`；不清深度 |
| 2 | `FUN_0041fac0` | `0x41fef0` | | |
| 3 | `FUN_0042caa0` | `0x42d2f0` | | |
| 6 | `FUN_0040b560` | `0x40ba80` | | |
| **7** | `FUN_00402230`（STD，EBP=0） | `0x403060` → `FUN_00402850` | STD | 背景 pass 0–7，清屏、相机、雾（§4.4） |
| **9** | `FUN_00445900` | `0x4485f0` → `FUN_00448980`（层 0） | 游戏 | |
| **0xa** | `FUN_00402230`（STD） | `0x403070` → `FUN_00402ca0` | STD | 先画层 17、18（`FUN_00448980`，EAX = 0x11、0x12），再画背景 pass 8–11（§4.4.2） |
| 0xb | `FUN_00445900` | `0x448600`（层 1） | 游戏 | |
| 0xd | `FUN_00445900` | `0x448610`（层 2） | 游戏 | |
| 0xe | `FUN_004089c0` | `0x409230` | | |
| **0xf** | `FUN_00445900` | `0x448620`（设相机矩阵，层 3） | 游戏 | 之后的层用相机（§3.4） |
| 0x10 | `FUN_00445900` | `0x4486a0`（层 4） | 游戏 | |
| 0x11 | `FUN_00445900` | `0x4486b0`（层 5） | 游戏 | |
| 0x12 | `FUN_00445900` | `0x4486c0`（层 6） | 游戏 | |
| 0x13 | `FUN_00445900` | `0x4486d0`（层 7） | 游戏 | |
| 0x14 | `FUN_0040d280` | `0x40d820` | 敌人 | |
| 0x15 | `FUN_00445900` | `0x4486e0`（层 8） | 游戏 | |
| 0x16 | `FUN_004247f0` | `0x426510` | 自机 | |
| 0x17 | `FUN_00445900` | `0x4486f0`（层 9） | 游戏 | |
| 0x18 | `FUN_00445900` | `0x448700`（层 10） | 游戏 | |
| 0x19 | `FUN_0041ad90` | `0x41ba30` | 道具 | |
| 0x1a | `FUN_00445900` | `0x448710`（层 11） | 游戏 | |
| 0x1b | `FUN_0041c120` | `0x41c4e0` | 激光 | |
| 0x1c | `FUN_00445900` | `0x448720`（层 12） | 游戏 | |
| 0x1d | `FUN_00405e20` | `0x4067a0` | 子弹 | |
| 0x20 | `FUN_0040ae70` | `0x40b060` | | |
| 0x21 | `FUN_00445900` | `0x448730`（层 13） | 游戏 | |
| 0x22 | `FUN_004055c0` | `0x405850` | | |
| 0x23 | `FUN_0042b4d0` | `0x42b9b0` | | |
| 0x24 | `FUN_00445900` | `0x448740`（层 14） | 游戏 | 层 14 之前把场地相机位移与 ANM 位移清零（不 flush），之后不再震动 |
| 0x25 | `FUN_004089c0` | `0x409270` | | |
| 0x26 | `FUN_00401110` | `0x401520` | | |
| 0x28 | `FUN_00420470` | `0x4200c0` | | 只返回 1 |
| **0x29** | `FUN_00445900` | `0x448770`（2D，层 15） | 游戏 | 切到 2D 帧 `0x491e94` |
| **0x2a** | `FUN_00445900` | `0x4487c0`（2D，层 16） | 游戏 | 切到 2D 帧 |
| 0x2b | `FUN_00413980`（GUI） | `0x415af0` | GUI | `front.anm` 相关（§5.4） |
| 0x2c | `FUN_00418e30` | `0x4198b0` | | |
| **0x2d** | `FUN_00445900` | `0x448810`（2D，层 0x13 = 19） | 游戏 | 第 19 层表 |
| 0x2e | `FUN_00422150` | `0x422aa0` | | |
| 0x30 | `FUN_00401110` | `0x401510` | | |
| 0x32 | `FUN_00420470` | `0x4200d0` | | flush、场地相机位移清零 |

来源：`render6/func_*.asm` 中各构造函数的 `MOV EDI,imm` + `CALL 0x00449b70`（例如 `func_405e20.asm`、`func_41c120.asm`、`func_4247f0.asm`、`func_420470.asm`）；游戏对象见 `render/func_445900.asm`（行 116–655）；STD 见 `render/func_402230.asm`（行 75–106）。

未枚举：部分管理器构造里还有 `FUN_00449b70` 调用（如 `m2reg/func_40b560.c`、`m2reg/func_41fac0.c` 中的 `0x41f990` 实际是线程创建，不是调度节点）。对节点是否真正启用（bit1）的核对只做了 STD 和游戏对象（§1.4、§4.2）。

### 1.4 每帧顺序（`FUN_00439390`，由消息循环 `FUN_00438ad0` 调用）

证据：`render/func_439390.c`、`render/func_438ad0.c`。

1. 节流：`FUN_00439540` 取时间；若与上一帧的差 < `DAT_00470c90`（= 1/60 s，`pe_read` 读出）则 `Sleep(0)` 并返回 0。
2. `DAT_00491fac = &DAT_00491e94`（2D 帧），`FUN_00421480`（2D 的 VIEW/PROJ），`SetViewport(帧+0xcc)` = (0,0,640,480)。
3. `FUN_00449c00(DAT_00491be4)`：**整条 calc 链**。返回 0 → 本帧返回 1；返回 −1 → 返回 2（退出）。
4. 帧跳过计数 `this+0x14` 每帧 +1；达到 `DAT_00491d66 + 1` 时绘制。绘制段：
   - `BeginScene`（0xa4）；
   - 游戏对象的批处理指针复位（`+0x3adac8 = 0`，`+0x72dacc = +0x72dad0 = 对象+0x3adacc`）；
   - `FUN_00449d40(DAT_00491be4)`：**整条 draw 链**；
   - `FUN_00442f50()`（批处理 flush）；`SetTexture(0,0)`（0x104）；
   - `EndScene`（0xa8）；
   - `FUN_004391f0()`：`Present`（0x44）；失败则 Reset + `FUN_00439d20`（§2.4）。

- 2D 与相机切换：`FUN_00421480`（2D）与 `FUN_004215a0`（相机）都先调用 `FUN_00442f50` flush（`supervisor/supervisor.go` 的 `Flat`、`Perspective` 注释）。
- 震屏：`FUN_0043c5ca` 在 calc 中写场地相机位移（Field.Shift），绘制链在节点 0x24 与 0x32 把它清零（`Unshake`、`Finish`）。
- 清屏色 `DAT_004923a8` 的写入者：`FUN_00417c80`（关卡开始）；STD 脚本 op 13（`FUN_00403c80`，见 §4.3 表）。

因此：VM 步进和层表构造在 calc 阶段完成，draw 阶段只读层表并发出顶点。

---

## 2. R2 设备与全局状态

### 2.1 窗口与 D3D 对象

- `Direct3DCreate9(0x20)` → `DAT_00491c2c`（`render/func_438ad0.c`）。
- 窗口：`FUN_00439730`（`render2/func_439730.c`）注册类 "BASE"（窗口过程 `FUN_004390e0`）。
  - `DAT_00491d65 == 0`：`CreateWindowEx(... 0xcf0000, 0, 0, 0x280, 0x1e0)`，即 640×480 重叠窗口；
  - 否则：样式 `0x100a0000`，位置 `(-0x80000000, -0x80000000)`，尺寸按 `GetSystemMetrics` 加边框换算成客户区 640×480。
- 设备：`FUN_00439890`（`render/func_439890.c`，`cfg2/func_439890.c`）。
  - `IDirect3D9::GetAdapterDisplayMode`（vtbl `+0x20`）；
  - `IDirect3D9::CheckDeviceFormat`（`+0x28`，参数 adapter 0、HAL、桌面格式、用途 0、资源类型 3、格式 `0x15`=A8R8G8B8），失败则 `DAT_00491d78 |= 1`；
  - `IDirect3D9::CreateDevice`（`+0x40`，adapter 0、`D3DDEVTYPE_HAL` = 1、`hFocusWindow = DAT_004924f0`、`BehaviorFlags = 0x40`，`pp = &local_48[1]`，输出 `&DAT_00491c30`）。
  - 行为标志：首选 `0x40`（D3DCREATE_HARDWARE_VERTEXPROCESSING），失败回退 `0x20`（SOFTWARE）；**不含 D3DCREATE_FPU_PRESERVE(0x2)**，与 `m2_motion.md` §2 一致。

### 2.2 D3DPRESENT_PARAMETERS

`FUN_00439890` 把结构体放在栈上（`local_48[1..14]`，14 个 dword，`local_48[4..17]` 先全部清零）。字段顺序按 D3D9 头文件：`W,H,Format,BackBufferCount,MSType,MSQ,SwapEffect,hWnd,Windowed,EnableAD,ADFormat,Flags,RefreshRate,Interval`。

| 字段 | 值 | 分支 / 来源 |
|---|---|---|
| BackBufferWidth | 640（0x280） | `local_48[1]` |
| BackBufferHeight | 480（0x1e0） | `local_48[2]` |
| BackBufferFormat | 独占全屏：`0x16` X8R8G8B8（`DAT_00491d62 == 0`）或 `0x17` R5G6B5（`DAT_00491d62 != 0`，或 `DAT_00491d78 & 1`）；窗口：`0` UNKNOWN | `DAT_00491d65` 分支，`cfg2/func_439890.c:43-56` |
| BackBufferCount | 0（清零，D3D 视为 1） | 清零 |
| MultiSampleType | 0（NONE） | 清零；另 `SetRenderState(161)=0`（§2.4） |
| SwapEffect | 1（DISCARD）；回退路径中为 3（COPY） | `local_48[7]` |
| hDeviceWindow | 0（用焦点窗口） | 清零 |
| Windowed | 1（窗口）/ 0（全屏） | `local_48[9]`，只在窗口分支置 1 |
| EnableAutoDepthStencil | 1 | `local_48[10]` |
| AutoDepthStencilFormat | `0x50` = D3DFMT_D16 | `local_48[11]` |
| Flags | 1（D3DPRESENTFLAG_LOCKABLE_BACKBUFFER） | `local_48[12]` |
| FullScreen_RefreshRateInHz | 60（独占全屏）/ 0（窗口） | `local_48[13]` |
| PresentationInterval | 独占全屏第一次尝试：1（ONE，垂直同步）；其它：0x80000000（IMMEDIATE）；窗口：0（DEFAULT） | `local_48[14]` |

- 模式选择：`DAT_00491d65 != 0` 为窗口模式（`local_48[9]=1`，格式 UNKNOWN，刷新率 0，间隔 DEFAULT）；`== 0` 为独占全屏（格式 0x16/0x17，刷新率 60）。`DAT_00491d65` 只在 `DAT_00491d78 & 0x100` 时由对话框（资源 0xcb）结果写入：`DAT_00491d65 = (结果 != IDYES)`（`cfg/func_438ad0.c:57`）。Yes → 0 → 独占全屏；No → 1 → 窗口（**（推测）** 语义按代码分支推出，未核对对话框文字）。
- 副本：14 个 dword 复制到 `DAT_00491d0c`，供 `Reset`（0x40）使用（`render/func_439890.c` 末段、`render/func_438ad0.c`、`render/func_4391f0.c`）。
- 回退路径（**（推测）** 只核了分支结构，未逐条跟踪）：第二次尝试 `0x20`；`iStack_10`（间隔）在某些分支改为 IMMEDIATE，`local_48[7] = 3`（COPY）。最终失败则返回非零。
- 丢失设备：`TestCooperativeLevel`（0xc）返回 `0x88760869`（D3DERR_DEVICENOTRESET）时调用 `FUN_00438a30`、`Reset(&DAT_00491d0c)`、`FUN_00439d20`（`render/func_438ad0.c`）。`Present` 返回 < 0 时在 `FUN_004391f0` 做同样处理。

### 2.3 CreateDevice 其它

- 设备创建后：`GetViewport`（0xc0）、`GetDeviceCaps`（0x1c）、`SetTransform(VIEW, &DAT_00491c74)`（0xb0,2）、`SetTransform(PROJ, &DAT_00491cb4)`（0xb0,3）。两者的值由 `D3DXMatrixLookAtLH` / `D3DXMatrixPerspectiveFovLH` 算出，每帧都会被覆盖（`render/func_439890.c` 第 153–155 行）。
- 未在渲染相关转储中找到 `SetGammaRamp`（vtbl `0x54`）的调用。**（推测）** 原作未改 gamma；扫描范围仅限 `.tmp/ecl_dump/render*/` 中引用设备全局的函数。
- `DAT_00491d62`（颜色深度）：初值 −1，`FUN_00439890` 把它规范为 0（32 位）或 1（16 位）（`cfg2/func_439890.c:43-56`）。与 `th10.cfg` +0x1a 的关系未核实。
- `DAT_00491d66`（跳帧）：在 `render*/` 里没有写入者，见 §7。

### 2.4 初始渲染状态（`FUN_00439d20`）

证据：`render/func_439d20.c`（设备创建后与 Reset 后都调用，`render/func_438ad0.c`、`render/func_4391f0.c`、`cfg2/func_439890.c`）。

| 状态 | 值 | 含义 |
|---|---|---|
| ZENABLE (7) | 1 | 深度测试开（具体 ZFUNC 由绘制函数设置，§4.7） |
| LIGHTING (137) | 0 | |
| CULLMODE (22) | 1 | D3DCULL_NONE |
| ALPHABLENDENABLE (27) | 1 | |
| SHADEMODE (9) | 2 | D3DSHADE_GOURAUD |
| SRCBLEND (19) | 5 | D3DBLEND_SRCALPHA（之后只有这里写它） |
| DESTBLEND (20) | 6 | D3DBLEND_INVSRCALPHA（之后会改，§3.5） |
| ZFUNC (23) | 8 | D3DCMP_ALWAYS |
| ALPHATESTENABLE (15) | 1 | |
| ALPHAREF (24) | 1 | |
| ALPHAFUNC (25) | 7 | D3DCMP_GREATEREQUAL（即 alpha ≥ 1/255 才写入） |
| FOGENABLE (28) | 1 | |
| FOGDENSITY (38) | 1.0 | |
| FOGTABLEMODE (35) | 0 | D3DFOG_NONE（表格雾关） |
| FOGVERTEXMODE (140) | 3 | D3DFOG_LINEAR（顶点雾，线性） |
| RANGEFOGENABLE (48) | 未设（0） | 雾用 z，不用距离 |
| FOGCOLOR (34) | `0xffa0a0a0` | 初值，之后被 STD 覆盖 |
| FOGSTART (36) | 1000.0（0x447a0000） | 初值，之后被 STD 覆盖 |
| FOGEND (37) | 5000.0（0x459c4000） | 初值，之后被 STD 覆盖 |
| MULTISAMPLEANTIALIAS (161) | 0 | |

纹理阶段（stage 0）：

| TSS | 值 | 含义 |
|---|---|---|
| COLOROP (1) | 4 | D3DTOP_MODULATE |
| COLORARG1 (2) | 2 | D3DTA_TEXTURE |
| COLORARG2 (3) | 3 | D3DTA_TFACTOR（批处理时由 flush 改为 DIFFUSE，§3.4） |
| ALPHAOP (4) | 4 | MODULATE |
| ALPHAARG1 (5) | 2 | TEXTURE |
| ALPHAARG2 (6) | 3 | TFACTOR（同上） |
| TEXTURETRANSFORMFLAGS (24) | 2 | D3DTTFF_COUNT2 |
| TEXCOORDINDEX (11) | 0 | |

采样器（stage 0）：ADDRESSU (1) = 1（WRAP）、ADDRESSV (2) = 1（WRAP）、ADDRESSW (3) = 3（CLAMP）、MAGFILTER (5) = 2（LINEAR）、MINFILTER (6) = 2（LINEAR）、MIPFILTER (7) = 0（NONE）。

游戏对象初值（`render/func_439d20.c` 末段）：`+0x3ada68 = 3`（混合缓存），`+0x3ada69 = +0x3ada6a = +0x3ada6c = 0xff`，`+0x3ada64 = 0`。这些是 §3 中各缓存的初值。

注意：`FUN_00439d20` 只在 init 时写了 `0xf`（ALPHATESTENABLE）、`0x18`、`0x19`、`0x16`、`0x89`、`0x26`、`0x23`、`0x8c`、`0xa1`、`0x1b`、`0x13`（SRCBLEND）。这些状态在渲染转储中没有其它写入者。

### 2.5 变换与视口

- **2D 帧结构 `0x491e94`**（`render2/func_4216f0.c` 写入默认值）：视口 `+0xcc..0xd8` = (0, 0, 640, 480)，`+0xd0`、`+0xd4`、`+0xd8` 为 y、宽、高（`0x491f60..0x491f74`）；VIEW 在 `+0x4c`，PROJ 在 `+0x8c`。
- **相机块 `0x491d7c`**（与 STD 相机结构相同，见 `m2_motion.md` §12）：视口 `+0xcc` = (32, 16, 384, 448, 0, 1)（`0x491e48..0x491e5c`），FOV `+0x48`（`0x491dc4` = 0x3f060a92，即 π/6；与 `m2_motion.md` §12 一致）；VIEW `+0x4c`，PROJ `+0x8c`，与 2D 帧的偏移一致。
- **`FUN_00421480`**（`render/func_421480.c`）：用帧结构的宽 (`+0xd4`) 与高 (`+0xd8`) 生成：
  - VIEW = `LookAtLH(eye=(w/2, h/2, d), at=(w/2, h/2, 0), up=(0,−1,0))`，其中 `d = (h/2) / tan(π/20)`；
  - PROJ = `PerspectiveFovLH(fov=0x3ea0d97c（π/10）, aspect=w/h, near=1.0, far=10000)`。
  - 然后 `SetTransform(2, VIEW)`、`SetTransform(3, PROJ)`，并把 `帧+0xe8/+0xec` 复制到游戏对象 `+0x5c/+0x60`（全局偏移，供 `FUN_00442670` 使用）。
  - `π/20` 来自双精度常量 `0x470ba0` = 0.15707963705（`pe_read` 读出，`render/func_421480.asm` 第 40 行 `FLD double ptr [0x00470ba0]`）。因此 z = 0 的平面上 1 单位 = 1 像素（**（推测）** 由公式推出，未渲染验证）。
  - up = (0,−1,0)，所以屏幕 y 向下（推测，由 LookAt 的基向量推出）。
- **相机路径** `FUN_004215a0`（STD 相机，见 `m2_motion.md` §12）同样生成 VIEW/PROJ（`std2/func_4215a0.c`）。

### 2.6 其它

- 多重采样：`MultiSampleType = 0`，无 MSAA。
- 深度：D16（`0x50`），LOCKABLE_BACKBUFFER 标志 1。
- 颜色：32 位（默认）或 16 位（`DAT_00491d62 == 1`）。

---

## 3. R3 ANM 绘制

### 3.1 VM 结构（0x3ac 字节）

证据：`anmvm/func_43ee30.c`、`render/func_4451c0.c`、`render/func_442670.c`、`render/func_444760.c`、`render4/func_448860.c`。

| 偏移 | 内容 | 证据 |
|---|---|---|
| `+0x20` | 图层（byte），由 ins_68（case 0x44）写入 | `anmvm/func_43ee30.c` `case 0x44` |
| `+0x24/+0x28/+0x2c` | 旋转角 X/Y/Z（用于 mode 8 的 WORLD，`D3DXMatrixRotationX/Y/Z`） | `render/func_444760.c` |
| `+0x2c` | 旋转角（mode 1/3 的 2D 旋转） | `render/func_4451c0.c` |
| `+0x2fc / +0x300` | 颜色 ARGB（`+0x300` 在 flags bit15 = 0x8000 时使用） | `render/func_442670.c` |
| `+0x2ff` | `+0x2fc` 颜色的 alpha 字节（小端 ARGB 的最高字节）；为 0 则不绘制，即使 0x8000 选了 `+0x300` 也看这个字节 | `render/func_4451c0.c` |
| `+0x334/338/33c` | 位置分量 A（x/y/z） | `render/func_443290.c`、`render/func_443080.c` |
| `+0x340/344/348` | 位置分量 B（STD：quad 位置 + face 位置，§4.5） | `render/func_403a30.c` |
| `+0x34c/350/354` | 位置分量 C | `render/func_443290.c` |
| `+0x35c` | flags：bit0 = 活动（ins_3 置位），bit1 = 允许绘制（写入者未追踪），bit26 (0x4000000) = 隐藏（跳过），bits 4–5 = 混合模式（ins_66），bits 22–25 = render mode（ins_67），bit15 (0x8000) = 颜色取 `+0x300`，bit31 = 点采样，bit11 (0x800) 由 ins_73 写入（取操作数最低位，§3.8） | `anmvm/func_43ee30.c`、`render/func_4451c0.c`、`render/func_4423e0.c` |
| `+0x390` | 当前指令指针（0 = 结束） | `anmvm/func_43ee30.c` |
| `+0x394` | 运行时精灵指针（0x44 字节，`FUN_0043e5a0` 设置）：`+4` 纹理；`+0x20/+0x24/+0x28/+0x2c` UV 矩形 u0/v0/u1/v1（浮点）；`+0x30` 高、`+0x34` 宽；`+0x18/+0x1c`、`+0x38/+0x3c` 未知（R6-A） | `render/func_442670.c`、`render/func_444760.c` |
| `+0x398` | 每步更新回调（`FUN_00445620`，由 `FUN_004452f0` 设置） | `render/func_4452f0.c` |
| `+0x39c` | 绘制前回调（`LAB_00445880`，由 `FUN_004452f0` 设置） | `render/func_4452f0.c` |
| `+0x3c/+0x40` | 缩放 x/y（STD 按 quad 尺寸除精灵尺寸设置） | `render/func_403a30.c` |
| `+0x4c/+0x50` | 基础宽/高（与缩放 `+0x3c/+0x40` 相乘得绘制尺寸）；精灵指令从精灵 `+0x34/+0x30` 复制，其它写入者待查（R6-C） | `render/func_443080.c`、`render/func_443290.c` |
| `+0x54/+0x58` | UV 滚动（加到 UV 上） | `render/func_442670.c`、`render/func_444760.c` |

游戏对象 `DAT_00491c10`（`FUN_00445900`，0x732460 字节）：

- 层哨兵：`game + 0x72dae4 + L·0x3ac`（L = 0..18），链表头在 `哨兵+0x1c`（`render4/func_448860.c` 初始化 19 个）。第 19 层哨兵是 `game+0x7320a8`（`render4/func_448900.c`）。
- 活动 VM 表：`game+0x72dad4`（由 `FUN_00448860` 遍历）；第 19 层的源表：`game+0x72dadc`（`FUN_00448900`）。
- 批处理：顶点缓冲在 `game+0x3adacc`（基址），计数 `+0x3adac8`，写指针 `+0x72dacc`，起点 `+0x72dad0`，flush 计数 `+0x58`（`render/func_442f50.c`）。
- 缓存：`+0x3ada60`（TFACTOR 颜色）、`+0x3ada64`（纹理）、`+0x3ada68`（混合模式）、`+0x3ada6a`（顶点类型）、`+0x3ada6c`、`+0x3ada6e`（点采样）、`+0x3ada70`（mode 8 的精灵指针）、`+0x3ada74`（stream 偏移）。
- 全局着色/淡入：`game+0x73245c`（启用）、`+0x732458`（颜色 ARGB），`+0x732459..0x73245b` 逐通道（`render/func_4423e0.c`、`render/func_442670.c`）。

### 3.2 render mode 分支（`FUN_004451c0`）

证据：`render/func_4451c0.c`、`render/func_4451c0.asm`。

前置条件：`VM.flags & 1` 且 `VM.flags & 2` 且 `VM+0x2ff != 0`，否则返回 −1。

mode = `(flags >> 22) & 0xf`：

| mode | 调用 | 顶点与图元 | 备注 |
|---|---|---|---|
| 0 | `FUN_00443080` → `FUN_00442670(this, 1)` | 轴对齐 2D；x 用 `floor(中心 − w/2)` 等吸附到整数，再 `round(·) − 0.5`（**像素吸附**） | 锚点由 flags bit18–19（x）、bit20–21（y）决定：0 中心、1 左/上、2 右/下 |
| 1 | 旋转路径（§3.2.1） | 旋转后 4 角 → `FUN_00442670(this, 0)` | 角度为 0 时改走 `FUN_00443290` |
| 2 | `FUN_00443290` → `FUN_00442670(this, 0)` | 轴对齐 2D，无吸附 | 与 mode 0 的区别是无吸附；**z 取 `+0x350`**：汇编 `0x0044343b` 确认读的是位置分量 C 的 y，ZUN 的 bug，照搬（§3.9） |
| 3 | 与 mode 1 代码相同 | 同 mode 1 | 反编译中 case 3 与 mode 1 的 fall-through 路径重复，行为相同 |
| 4 | `FUN_00443f80` | 经 `FUN_00443b60` 求角点，再 `FUN_00442670` | **（推测）** 多边形/自由顶点，未逐行核对 |
| 5 | `FUN_00444580` | 经 `FUN_00444240` 求角点，再 `FUN_00442670` | **（推测）** 同上 |
| 6 | `FUN_00443fb0` | 经 `FUN_00443b60` 求角点，再 `FUN_00442670`（两次调用） | **（推测）** 同上 |
| 7 | `FUN_004445c0(this, VM)` | 经 `FUN_00444240` 求角点，再 `FUN_00442670`（两次调用） | **（推测）** 同上 |
| 8 | `FUN_00444760` | 3D：WORLD 矩阵 + 4 个顶点（VB），`DrawPrimitive(TRIANGLESTRIP, 2 图元)` | 不进批处理（§3.3.3） |
| 9 | `FUN_00444ce0` | `DrawPrimitiveUP(TRIANGLESTRIP, n−2, 局部顶点, stride 0x1c)` | 不进批处理；`n` 是调用方给出的顶点数 |
| 10–15 | `default` | 不绘制（返回 0） | ins_67(10) 另外调用 `FUN_004452f0` |

ins_67(10)（`anmvm/func_43ee30.c` `case 0x43`）：当 `(flags & 0x3c00000) == 0x2800000` 时调用 `FUN_004452f0`。这个函数分配 0x4b0 字节，写入 `+0x398`/`+0x39c`，随机生成 31 个粒子（`render/func_4452f0.c`；由 `FUN_0044bb90` RNG 决定）。**（推测）** 它是粒子/爆发类效果的初始化，不直接绘制。

#### 3.2.1 旋转路径（mode 1 / 3）

- 角度 `+0x2c`；若为 0 则走 `FUN_00443290`。
- 否则 `fcos/fsin(+0x2c)`，四角 = 中心 + R(θ)·(±w/2, ±h/2)。锚点分别由 `flags>>18 & 3`（x）和 `flags>>20 & 3`（y）决定，取 0/1/2 对应中心、左/上、右/下（`render/func_4451c0.c` 中的 `uVar8`）。
- 四角写入 `DAT_00497914..`（同 §3.3），然后调用 `FUN_00442670(this, 0)`。
- z = `+0x354 + +0x348 + +0x33c`。

### 3.3 2D 四角发射 `FUN_00442670(this=游戏对象, param)`

证据：`render/func_442670.c`、`render/func_442fe0.c`。

1. 四角：P0 = (x0, y0)、P1 = (x1, y0)、P2 = (x0, y1)、P3 = (x1, y1)。`x0 = DAT_004978c0`、`x1 = DAT_004978dc`、`y0 = DAT_004978c4`、`y1 = DAT_004978fc`（mode 0/2 由调用方填入，角点加上 `this+0x5c/0x60`，即游戏对象的全局偏移）。
2. **半像素**：`param & 1` 时 `x = round(x) − 0.5`、`y = round(y) − 0.5`（常量 `0x4700ec` = 0.5，`pe_read` 读出）。`param & 1 == 0` 时不加偏移。原作 XYZRHW 顶点直接用设备坐标，D3D9 的像素中心在整数坐标 +0.5 处（**（推测）** 与 Ebiten 的 +0.5 约定不同，见 `m4_render.md` §2.1）。
3. **剔除**：角点外接矩形与当前帧视口 `帧+0xcc..0xd8` 相交，否则返回。2D 层剔除到 (0,0,640,480)，相机层剔除到场地 (32,16)–(416,464)。
4. **纹理**：`VM+0x394` 的纹理（`+4`）；若与游戏对象缓存 `+0x3ada64` 不同则 `SetTexture(0, tex)`（0x104）。
5. **UV**：`u0 = 精灵+0x20 + VM+0x54`，`u1 = 精灵+0x28 + VM+0x54`，`v0 = 精灵+0x24 + VM+0x58`，`v1 = 精灵+0x2c + VM+0x58`（`render/func_442670.c` 中的 `DAT_004978d4/f0/d8/10`）。
   - P0 = (u0, v0)、P1 = (u1, v0)、P2 = (u0, v1)、P3 = (u1, v1)。
6. **颜色**：`VM.flags & 0x8000 ? +0x300 : +0x2fc`（ARGB）。若 `game+0x73245c` 非零，逐通道 `c·tint >> 7` 夹到 255（`FUN_004423c0`，`render/func_4423c0.c`）。
7. **z** = `+0x354 + +0x348 + +0x33c`（mode 0/1/3）或 `+0x350 + +0x348 + +0x33c`（mode 2，见 §7）；`rhw = 1.0`（`0x4978cc` 等，`render/func_445900.c` 初始化）。
8. 状态：`FUN_004425a0`（混合、点采样，§3.5）。
9. `FUN_00442fe0(this, corners)`：把 4 角展开成 6 个顶点（P0,P1,P2,P1,P2,P3），每个 28 字节（x, y, z, rhw, diffuse, u, v），写入批处理 `game+0x72dacc`，`+0x3adac8 += 1`（`render/func_442fe0.c`）。

#### 3.3.1 顶点格式

- 2D 批处理：`FVF = 0x144` = XYZRHW(0x4) | DIFFUSE(0x40) | TEX1(0x100)，stride 0x1c（28 字节）；图元 TRIANGLELIST，每四角 6 顶点、2 个三角形。
- mode 9：同 FVF，TRIANGLESTRIP，stride 0x1c（`render/func_444ce0.c`）。
- mode 8：`FVF = 0x102` = XYZ(0x2) | TEX1(0x100)，stride 0x14，经 `SetStreamSource(0, VB, 0, 0x14)`（vtbl `0x190`）（`render/func_444760.c`）。
- 无纹理叠加（`FUN_00443bda0`、`FUN_0043bfa0`）：`FVF = 0x44` = XYZRHW | DIFFUSE，stride 0x14，TRIANGLESTRIP 2 图元（`render/func_43bda0.c`、`render/func_43bfa0.c`）。

### 3.3.2 批处理与 flush `FUN_00442f50`

证据：`render/func_442f50.c`。

- 条件：`game+0x3adac8 != 0`。
- 动作：`TSS(0,6,0)`、`TSS(0,3,0)`（ALPHAARG2 与 COLORARG2 = DIFFUSE，即顶点色）、`SetFVF(0x144)`、`DrawPrimitiveUP(TRIANGLELIST = 4, primCount = count·2, 顶点 = game+0x72dad0, stride 0x1c)`，然后起点 `+0x72dad0 = +0x72dacc`，计数清零，`+0x58 += 1`。
- 调用点：几乎所有状态变化前（`FUN_004423e0`、`FUN_004425a0`、`FUN_00442670` 等）。

### 3.3.3 3D 路径（mode 8）`FUN_00444760`

证据：`render/func_444760.c`、`render/func_444760.asm`。

- WORLD 矩阵（`SetTransform(256, &local_c0)`）：由旋转（`D3DXMatrixRotationX/Y/Z`，角度 +0x24/+0x28/+0x2c，脏位 `+0x35c & 4/8`）与位置分量（x = `+0x334+0x340+0x34c`，y = `+0x338+0x344+0x350`，z = `+0x33c+0x348+0x354`，并受锚点修正）组成。**（推测）** 具体矩阵乘法顺序未逐行核对。
- TEXTURE0 矩阵（`SetTransform(16, auStack_4c)`，`0x10`）：UV 滚动（精灵 `+0x20/+0x24` 加 VM `+0x54/+0x58`）。
- 纹理：`SetTexture(0, 精灵+4)`。
- FVF 0x102、`SetStreamSource` stride 0x14；TSS `(0,6,3)`、`(0,3,3)`（ALPHAARG2 与 COLORARG2 = TFACTOR，颜色由 `FUN_004423e0` 设置）。
- `DrawPrimitive(TRIANGLESTRIP = 5, startVertex 0, primCount 2)`（vtbl `0x144`）。

### 3.4 层、图层与绘制顺序

证据：`render4/func_448980.c`、`render4/func_448860.c`、`render4/func_448900.c`、`render2/func_4485f0.asm`…`render2/func_448810.asm`（各节点 `MOV EAX,imm`）、`render/func_439390.c`。

- **步进（calc 0x1a → `FUN_00448860`）**：遍历活动表 `game+0x72dad4`。对每个不带 bit26 的 VM：先调 `+0x398`，再 `FUN_0043ee30(VM)` 步进；返回 0（仍活）则按 `VM+0x20` 挂入层哨兵的链表末尾；否则 `FUN_00448bb0` 释放。遍历结束 `game+100 += 1`。
- **第 19 层（calc 8 → `FUN_00448900`）**：遍历 `game+0x72dadc` 的 VM，同样步进；仍活的挂入第 19 层哨兵（`game+0x7320a8+0x1c`）。`VM+0x20` 不参与分层（**（推测）** 这一层是一条独立的活动列表）。
- **绘制（draw 节点 → `FUN_00448980`，EAX = 层号）**：遍历层表，跳过 bit26；`+0x39c` 非空则先调用；然后 `FUN_004451c0(game)`（EAX = VM，ECX = game）。
- 同一层内顺序：按活动表 `game+0x72dad4` 的顺序（即 VM 的步进顺序，**（推测）** 近似创建顺序）。
- 层与相机状态（draw 链优先级 → 层）：

| 优先级 | 层 | 矩阵/视口 |
|---|---|---|
| 9 | 0 | 继承 STD draw 7 设置的相机（`FUN_00402850` 把 `DAT_00491fac` 设为相机） |
| 0xb | 1 | 同上 |
| 0xd | 2 | 同上 |
| 0xf | 3 | `DAT_00491fac = 相机`，`FUN_00421480`（相机 VIEW/PROJ），`SetViewport(相机+0xcc)` |
| 0x10–0x24 | 4–14 | 相机 |
| 0x29 | 15 | `DAT_00491fac = 2D 帧`，`FUN_00421480`，`SetViewport(2D+0xcc)`（**2D**） |
| 0x2a | 16 | 2D |
| 0x2d | 19（第 19 层表） | 2D |

- 层 0–2、4–14 与 STD 的关系：STD draw 7 结束时 `DAT_00491fac` 为相机，所以层 0–2 也用相机视口。
- 第 0x13 层与 0x2d 节点：`FUN_00448810` 的 `MOV EAX,0x13`（`render2/func_448810.asm`）。

### 3.5 混合、采样与 TFACTOR

证据：`render/func_4423e0.c`（3D 路径，被 mode 8 调用）、`render/func_4425a0.c`（2D 路径，被 `FUN_00442670` 调用）、`render/func_4423c0.c`。

- **混合模式** `b = (flags >> 4) & 3`（由 ins_66 写入，`anmvm/func_43ee30.c` `case 0x42`）。缓存 `+0x3ada68`，只在变化时设置。SRCBLEND 固定 SRCALPHA（§2.4，无其它写入）。

| b | 2D 路径 `FUN_004425a0` | 3D 路径 `FUN_004423e0` |
|---|---|---|
| 0 | DESTBLEND = 6（INVSRCALPHA） | DESTBLEND = 6 |
| 1 | DESTBLEND = 2（ONE） | DESTBLEND = 2 |
| 2 | **不改** | DESTBLEND = 2（ONE） |
| 3 | **不改** | **不改** |

- 两个函数对 b=2 的处理不一致（**（推测）** 可能是 bug 或是有意的，需核实；见 §7）。
- **点采样**：`flags` bit31 = 1 → MAG/MIN = POINT(1)；否则 LINEAR(2)。缓存 `+0x3ada6e`（`render/func_4425a0.c`、`render/func_4423e0.c`）。
- **TFACTOR**：3D 路径（`FUN_004423e0`）在颜色变化时 `SetRenderState(60, 颜色)`（缓存 `+0x3ada60`）。颜色取自 `+0x300`（bit15）或 `+0x2fc`，再经 `game+0x73245c` 调制。
- 2D 路径不设 TFACTOR，颜色用顶点色（flush 时 TSS = DIFFUSE）。

### 3.6 颜色调制（全局淡入/暗化）

- `game+0x73245c` 非零时，颜色逐通道 `c·game[+0x73245a / +0x732459 / +0x732458 / +0x73245b] >> 7`，夹到 255（`render/func_4423e0.c`、`render/func_442670.c`）。
- 写入者：STD 绘制 `FUN_00402850` 与 `FUN_00402ca0`（`render/func_402850.c`、`render/func_402ca0.c`）：`if (STD+0x1eec != 0) { game[+0x73245c] = 1; game[+0x732458] = 0xff404040; } else { game[+0x73245c] = 0; game[+0x732458] = 0x80808080; }`。即 STD 标志 `+0x1eec` 启用时，颜色按 0xff404040 调制（**（推测）** 对应符卡/暗化效果）。

### 3.7 图层、render mode 的 ANM 指令

| 指令 | 写入 | 证据 |
|---|---|---|
| ins_66（case 0x42） | `flags` bits 4–5 = 操作数 | `anmvm/func_43ee30.c` |
| ins_67（case 0x43） | `flags` bits 22–25 = 操作数；为 10 时 `FUN_004452f0` | 同上 |
| ins_68（case 0x44） | `VM+0x20` = 层号（byte） | 同上 |
| ins_73（case 0x49） | `flags` bit 0x800 = 操作数最低位（异或写入） | `anmvm/func_43ee30.c:1688` |
| ins_74（case 0x4a） | `flags` bit 0x2000 = 操作数最低位（异或写入） | `anmvm/func_43ee30.c` |

### 3.8 z 写入

- `flags` 的 0x800 由 ins_73 写入（取操作数最低位，`anmvm/func_43ee30.c:1688`）。
- 渲染转储中，`FUN_004451c0` 与 `FUN_00442670` 的路径**没有**读取 0x800，2D/3D 绘制都不写深度（ZWRITE 只在 STD 绘制中设置，§4.7）。**（推测）** 0x800 只在 STD 相关路径中有意义，或者属于尚未追踪的函数。


### 3.9 A1a 契约核对后的补充与更正（2026-11-01）

逐条核对 `render3/func_443080.asm`、`render3/func_443290.asm`、`render3/func_4451c0.asm`、`render/func_442670.asm`、`render/func_4425a0.asm` 后写成
`anm/anmdraw/anmdraw.go` 的文档注释（公式、加法分组、调用顺序以那里为准）。本节只记与上文不同或上文没写的点。

- **字段对应**：位置分量 A/B/C（`+0x334`/`+0x340`/`+0x34c`）在 `anm.VM` 里是 `Pos`/`Offset`/`Pos2`；`+0x24..+0x2c` 是 `Rot`，`+0x3c/+0x40` 是 `Scale`，
  `+0x54/+0x58` 是 `UV`，`+0x2fc`/`+0x300` 是 `Color`+`Alpha`/`Color2`+`Alpha2`，`+0x35c` 是 `Flags`。
- **锚点**（ins_65）：x = flags bit18–19，y = bit20–21，取值 0 中心、1 左/上、2 右/下。**取值 3** 时 mode 0/2 不写该轴的角点，
  暂存 quad（全局 `DAT_004978c0`）保留上一个 quad 的值；旋转路径在取值 3 时读未初始化的栈。
- **mode 0 的 floor 只在锚点 0**：`x0 = floor(中心 − w·0.5)`（`FUN_00452ff0`），锚点 1/2 不取整；之后 `FUN_00442670(·, 1)` 再做
  `FRNDINT − 0.5`，FRNDINT 按默认舍入模式，即**四舍六入五成双**（100.5 → 100）。
- **加法分组**不是统一的“A + B + C”：例如 x 锚点 1 的左边是 `(A.x + B.x) + C.x`，锚点 0 用 `(C.x + B.x) + A.x`，mode 2 的 y 锚点 1/2 用
  `(C.y + A.y) + B.y`。x87 精度为 24 位，分组不同结果可能不同，以契约为准。
- **`FUN_00442670` 的顺序**：加全局偏移 `+0x5c/+0x60` → 半像素吸附（仅 param & 1）→ UV → 剔除 → 纹理缓存 → 顶点类型缓存 `+0x3ada6a` →
  颜色 → `FUN_004425a0` → `FUN_00442fe0`。被剔除的 quad 不改任何缓存、不调设备、不进批次。
- **剔除的比较**（2026-11-01 按汇编更正）：`maxX = P0.X > P1.X ? P0.X : P1.X`，再依次换成更大的 P2.X、P3.X；`minX` 同理取更小者；
  Y 同理（`0x4427c4..0x442921`，`TEST AH,0x41`/`JNZ` 与 `TEST AH,5`/`JP`）。满足 `maxX < X`、`Y > maxY`、`X+W < minX`、`Y+H < minY`
  任一项即剔除（`0x44292c..0x4429c8`，均跳到 `0x442abb`）。视口四个 uint32 经 FILD 精确比较，X+W 按 uint32 相加。
  x87 分支只在有序结果时跳转，所以 NaN 比较为假：每次选择保留第二个操作数（首次是 P1，之后是当前界），NaN 界不剔除。
  原先写的“NaN 角点会被剔除”是错的（Ghidra 的 C 把条件写成了 `<=` 的取反形式）。项目：`anmdraw.(*Renderer).culled`。
- **统计计数（不移植）**：`FUN_004425a0` 每画一个 quad 执行 `INC [管理器+0x54]`（`0x442664`，3D 路径 `0x442592` 同），
  `FUN_00442f50` 每次非空提交 `+0x58` 加 1（`0x442fb7..0x442fd1`）。渲染转储中没有读取者。
- **缓存初值**（`FUN_00445900`）：纹理 NULL、混合 0、顶点类型 0、点采样 0；暂存 quad 的 rhw = 1，UV 为 (0,0)、(1,0)、(0,1)、(1,1)，其余为 0。
  混合缓存初值为 0 意味着第一次 b = 0 的绘制不设 DESTBLEND，依赖 `FUN_00439d20` 的初始 DESTBLEND = INVSRCALPHA。
- **缓存失效**：`FUN_0043bda0`（无纹理叠加）与 `FUN_00439d20`（初始渲染状态）把纹理缓存设为 NULL、混合缓存设为 3、顶点类型设为 0xff，
  不 flush，也不动点采样缓存 `+0x3ada6e`。混合 3 不对应任何会设置 DESTBLEND 的值，所以之后第一个 b = 0/1 的对象会重新设置。
- **绘制前回调**：`FUN_00448980` 对每个对象先调 `VM+0x39c`（若非空）再 `FUN_004451c0`。写入者与语义待查（R6-G）。

---

## 4. R4 STD 绘制

### 4.1 文件结构（对照 thtk，`.tmp/thtk-src/thstd/thstd.h`）

| 结构 | 大小 | 字段 |
|---|---|---|
| 文件头（`std_header_10_t`） | 0x90 + 名字区 | `u16 nb_objects`@0（entry 数）；`u16 nb_faces`@2（**quad 总数**）；`u32 faces_offset`@4；`u32 script_offset`@8；`char anm_name[128]`@0x10。entry 偏移表（`u32[nb_objects]`）从 @0x90 开始。 |
| entry（`std_entry_header_t`） | 0x1c | `u16 id`@0；`u8 pass`@2（`unknown` 的低字节）；`u8 flags`@3（`unknown` 的高字节；bit0、bit1 为运行时位）；`x,y,z`@4/8/0xc；`width,height,depth`@0x10/0x14/0x18。quad 列表从 @0x1c 开始，以 `s16 -1`（thtk 中为 `0x0004FFFF`）结尾。 |
| quad（`std_object_t`） | 0x1c | `u16 type`@0（全部为 0）；`u16 size`@2（记录步长，到下一个 quad）；`u16 script_index`@4；`u16 vm_index`@6（thtk 中为 `_padding`，文件里全 0，**运行时写入**）；`x,y,z`@8/0xc/0x10；`width`@0x14；`height`@0x18。 |
| face | 16 | `s16 entry_index`@0（−1 结束）；`s16`@2（未使用）；`x,y,z`@4/8/0xc（float）。 |

- 读取：`FUN_00403850`（`std2/func_403850.c`）。
  - 文件复制到 `this+0x10`；
  - ANM 文件 `FUN_00447280((flags&1)+4, 游戏对象, buf+0x10)` → `this+0x178`；
  - entry 偏移表重定位（`+= buf`）→ `this+0x14`；
  - faces `this+0x18 = buf + faces_offset`；script `this+0x1c = buf + script_offset`；
  - quad VM 数组 `this+0x17c = malloc(nb_faces × 0x3ac)`（`psVar2[1]` 即 nb_faces）。
- 析构：`FUN_00402440`（§1.1）。

### 4.2 设置：构造、关卡初始化

- 构造 `FUN_00402230(this, mode)`（`render/func_402230.c`、`render/func_402230.asm`）：
  - 创建三个节点（均先 `AND flags, ~2`，即**初始禁用**）：
    - `+0x8` = 计算节点 `0x403050`（优先级 `EBP+0xc`）；
    - `+0xc` = 绘制节点 `0x403060` → `FUN_00402850`（优先级 `EBP+7`）；
    - `+0x2a40` = 绘制节点 `0x403070` → `FUN_00402ca0`（优先级 `EBP+0xa`）。
  - `EBP` 是参数 `mode`：`FUN_00402640(obj, 0)`（`emgr/func_417870.c` 调用）时 EBP = 0，故优先级为 0xc、7、0xa。
  - 初始化相机块 `this+0x2a4c` 的默认值，并 `this+0x2a18 |= 1`。
- 工厂 `FUN_00402640`（`render/func_402640.asm`）：`operator new(0x2b64)`，然后调用构造。
- 关卡初始化 `FUN_00418190`（`emgr/func_418190.c`，由 `render/func_4180e0.c` 调用）：
  - 对 STD `DAT_004776e8` 调用 `FUN_00404450`（第 29 行）；`DAT_004776e4` 为空时走另一分支（第 30 行起）；
  - 对管理器节点 `+8`、`+0xc` 执行 `|= 2`（启用），与游戏对象相同。
- **启用**：`FUN_00404450`（`render4/func_404450.c`）：`this+8`、`this+0xc`、`this+0x2a40` 的节点 flags `|= 2`。STD 节点由此启用。

### 4.3 每帧 calc：`FUN_00402720`（节点 0xc）

证据：`render/func_402720.c`、`render/func_402720.asm`。

1. 若 `this+0x2a18 & 8` 为 0 且（`& 4` 为 0 或时间 < 0x3c）：
2. 清零位移 `+0x2b34..0x2b44`；`D3DXVec3Normalize(this+0x2a70, this+0x2a58)`（相机朝向归一化）；`this+0x1ee8 = 0x808080`。
3. **`FUN_00403990()`**（entry 循环，§4.6）。
4. **`FUN_00403c80(this)`**（STD 脚本 VM，见 `m2_motion.md` §12）。
5. 8 个槽 VM：`this+0x180 + i·0x3ac`（i = 0..7），逐个 `FUN_0043ee30`。
6. 若 `this+0x1eec != 0`：`DAT_00476f78 = 1.0`，再对 `this+0x1f08`、`+0x22b4`、`+0x2660` 三个 VM 调用 `FUN_0043ee30`（**（推测）** 这三个是特殊 VM，如符卡背景）。
   之后恢复游戏速度，并无条件 `this+0x1eec = 0`（0x0040282b）。绘制（`FUN_00402850`、`FUN_00402ca0`）在 calc 之后读这个标志，
   所以要让绘制看到非零，必须有代码在每帧的 STD calc 之后、STD 绘制之前重新写它。**（推测）** 它是逐帧的请求标志；写入者经指针间接访问，Ghidra 的偏移引用搜不到。
7. 相机块 `this+0x2a4c`（0x46 dword）复制到 `DAT_00491d7c`；`this+0x2a34 += 1`。

STD 脚本指令对绘制的影响（`render/func_403c80.c`）：

| op | 作用 |
|---|---|
| 8（`case 8`） | 立即设雾：`this+0x2b60 = 颜色（ARGB）`，`+0x2b48 = 近`（op+0xc），`+0x2b4c = 远`（op+0x10）；同时设 `+0x2b50..0x2b5c` 的 B、G、R、A 浮点 |
| 9（`case 9`） | 定时雾：`FUN_00405040`（启动插值，参数为时间与 RGBA 字节）；`this+0x16c = op+8`；之后每帧 `FUN_004049a0` 产生 7 个 dword，复制到 `this+0x2b48..0x2b60` |
| 12（`case 0xc`） | `this+0x20 = op+8`；若 `this+0x34 & 1` 为 0 则初始化 `+0x24/+0x28/+0x2c/+0x30` |
| 13（`case 0xd`） | `DAT_004923a8 = op+8`（全局） |
| 14（`case 0xe`） | 对槽 VM `idx = op+8`（地址 `this+0x180 + idx·0x3ac`）：若 `op+0xc < 0`，清除其 `+0x35c` bit0（停止）；否则 `FUN_00404f30(槽, this+0x178)` 重新初始化（`render/func_403c80.c:229-238`） |

**FUN_00404f30**（`render5/func_404f30.c`）：`FUN_00401de0` 初始化 VM，清零若干字段，设 `+0x3a1 = 0x10`、`+0x3a0 = 0x10`，`+0x38a = AX`（脚本号），然后 `FUN_0043e710(param)`。被 `FUN_00404450`（每个 quad）与 op 14 调用。

### 4.4 绘制：`FUN_00402850`（draw 7，pass 0–7）与 `FUN_00402ca0`（draw 0xa，pass 8–11）

证据：`render/func_402850.c`、`render/func_402850.asm`（调用序列）、`render/func_402ca0.c`、`render/func_402ca0.asm`。

两者共同的开头：

- `if (this+0x2a18 & 8) return 1`；
- 相机块复制到 `DAT_00491d7c`，`DAT_00491fac = 相机`，`FUN_004215a0()`（相机 VIEW/PROJ），`SetViewport(相机+0xcc)` = (32,16,384,448)（`0xbc`）；
- 雾的颜色与近远（`0x22` ← `this+0x2b60`，`0x24` ← `this+0x2b48`，`0x25` ← `this+0x2b4c`）；
- 颜色调制（§3.6）。

#### 4.4.1 `FUN_00402850`（draw 7）

调用顺序（按 `render/func_402850.asm` 的地址顺序，来自 `SetRenderState` / `Clear` / `SetViewport` 调用前的压栈值；寄存器 `EBX = 0`、`EDI = 1`）：

| 步骤 | 调用 | 参数 |
|---|---|---|
| 1 | `SetViewport`（0xbc） | 相机视口 (32,16,384,448) |
| 2 | `SetRenderState(ZWRITEENABLE 14)` | 1 |
| 3 | `SetRenderState(ZFUNC 23)` | 4（LESSEQUAL） |
| 4 | `SetRenderState(FOGCOLOR 34)`、`FOGSTART 36`、`FOGEND 37` | `this+0x2b60`、`+0x2b48`、`+0x2b4c` |
| 5 | `Clear(0 矩形, ZBUFFER, 颜色 0, z 1.0, stencil 0)`（0x40299c） | 清深度 |
| 6 | `Clear(1 矩形 = 场地 {32,16,416,464}, TARGET, 颜色, z 1.0, stencil 0)`：0x4029ec 与 0x402a38 是 if/else 两个分支，每帧只执行一次 | 颜色见 §5.2 |
| 8 | `FUN_0043c8b0(3, 0x1e, 0, 0, 0)`（0x402a55 起，**（推测）** 淡入）当 `this+0x2a18 & 4` 且 `this+0x2a20 < 0x1e` | |
| 9 | `flags |= 1`（`this+0x2a18`），`FUN_00405410(1)` | |
| 10 | 若 `this+0x514 != 0`：`ZWRITEENABLE = 0`（0x402b16）；`FUN_004451c0` ×8（STD 槽 VM `this+0x180 + i·0x3ac`，`+0x394 != 0` 才绘制，0x402b25–0x402b42）；flush；**`ZWRITEENABLE = 1`（0x402b62）**；`DAT_00491fac = 相机`、`FUN_004215a0()`（相机 VIEW/PROJ） | |
| 11 | `DAT_00492378 != 1` 时 `FOGENABLE = 1`（缓存 `DAT_00492378`） | |
| 12 | `FUN_00403a30(this, pass)` ×8，pass = `EBX`(0)、`EDI`(1)、2、3、4、5、6、7（地址 `0x402bd4..0x402c0d`） | 背景面（§4.5） |

| 13 | 收尾：`ZWRITEENABLE = 0`（0x402c6c，`PUSH EBX`）、`ZFUNC = 8` ALWAYS（0x402c89） | 之后的 draw 节点（层 0 等）不测试也不写深度 |

- 步骤 10 中 `ZWRITEENABLE` 的 0 → 1 切换是按地址顺序推出的；收尾两项已按 asm 核对（2026-11-01 复核）。

#### 4.4.2 `FUN_00402ca0`（draw 0xa）

调用序列（`render/func_402ca0.asm`）：

2026-11-01 复核（`render/func_402ca0.c` 与 `.asm`：`MOV EAX,0x11`/`0x12` 后 `CALL 0x00448980`，`PUSH 8..0xb` 后 `CALL 0x00403a30`）。按执行顺序：

| 段 | 状态 | 绘制 |
|---|---|---|
| 开头 | 相机 VIEW/PROJ、`SetViewport(相机)`；`FOGENABLE = 1` | |
| 层 17 | `ZWRITE = 0`、`ZFUNC = ALWAYS`、雾开 | `FUN_00448980`（EAX = 0x11） |
| 层 18 | `ZFUNC = LESSEQUAL`、`FOGENABLE = 0` | `FUN_00448980`（EAX = 0x12）：**与 pass 0–7 写下的深度比较，不写深度** |
| 雾参数 | `ZFUNC = LESSEQUAL`；`FOGCOLOR/START/END` ← `+0x2b60/+0x2b48/+0x2b4c` | |
| pass 8–11 | `flags & 1` 时：`ZWRITE = 0`、`FOGENABLE = 1`，ZFUNC 仍是 LESSEQUAL | `FUN_00403a30` ×4（pass 8、9、10、11），每个 quad 按 render mode 再改 FOGENABLE（§4.5）：**深度测试、不写** |
| 收尾 | `ZWRITE = 0`、`ZFUNC = ALWAYS`（两次）、`FOGENABLE = 0` | |

- 第 1 段的条件与 `FUN_00402850` 相同（`(flags & 4) == 0 || +0x2a20 < 0x3c`），pass 8–11 的条件是 `flags & 1`。
- `FUN_00402ca0` 不做 Clear。

#### 4.4.3 状态时间线的解释

- `FUN_00402850`：ZWRITE 开始为 1，槽绘制前为 0，面绘制（pass 0–7）前恢复为 1；ZFUNC 全程 LESSEQUAL；收尾 ZWRITE = 0、ZFUNC = ALWAYS。
- `FUN_00402ca0`：层 17 不测试；层 18 与 pass 8–11 用 LESSEQUAL 测试、不写。

### 4.5 面遍历与 quad 变换：`FUN_00403a30(this, pass)`

证据：`render/func_403a30.c`、`render/func_403a30.asm`、`render3/func_403090.c`。

面列表从 `this+0x18` 开始，**按文件顺序**，每项 16 字节，遇到 `entry_index < 0` 结束：

```
for face in faces (文件顺序):
    entry = entries[face.entry_index]                 (this+0x14 表)
    if (*(char*)(entry+2) != pass) continue           (pass 匹配)
    if FUN_00403090(entry, &face.pos, this+0x1ee0) == 0:   (可见)
        entry.flags |= 2
        for quad in entry (列表顺序, 直到 -1):
            vm = this+0x17c + quad.vm_index * 0x3ac
            if quad.type == 0:
                if (vm.flags & 0x3c00000) > 0xffffff:     (render mode >= 4，即 bit24/25 有值)
                    vm.pos_B = quad.pos + face.pos         (+0x340/344/348)
                    if quad.width  != 0: vm.scale_x = quad.width  / sprite.+0x34; vm.flags |= 8
                    if quad.height != 0: vm.scale_y = quad.height / sprite.+0x30; vm.flags |= 8
                FOGENABLE = ((vm.flags & 0x3c00000) == 0x2000000)   (render mode == 8，缓存 DAT_00492378)
                FUN_004451c0(game)  (ECX = game, EAX = vm)            ← 层 3D/2D 绘制
                this.+0x2a14 += 1
        this.+0x2a0c += 1
    else:
        this.+0x2a10 += 1
```

- **位置合成**：quad 位置（`quad+8/0xc/0x10`）加 face 位置（`face+4/8/0xc`）得到 VM 的 `pos_B`（+0x340..0x348）。**entry 的位置不加入**，它只用于剔除（`FUN_00403090`）。
- **尺寸**：quad 的 width/height（`+0x14/+0x18`）除以精灵尺寸（`sprite+0x34` 与 `+0x30`，**（推测）** 是精灵宽高）后得到缩放 `+0x3c/+0x40`。宽或高为 0 时保留 VM 原缩放。
- `pos_B` 与缩放只在 render mode ≥ 4 时写入：反编译条件是 `0xffffff < (flags & 0x3c00000)`（`render/func_403a30.c`），即 bit24 或 bit25 有值。mode 0–3 的 quad 不由 face 写位置，**（推测）** 其位置由 ANM 脚本自己给出；这一点与 §3.2 的 2D 分支需要一起核对。
- `FUN_004451c0` 在 `type == 0` 的 quad 上调用，位于 `pos_B` 写入的条件块之外（`render/func_403a30.c` 中 `if (sVar3 == 0)` 块内）。
- FOGENABLE：`(flags & 0x3c00000) == 0x2000000`，即 **mode 8**（0x2000000 >> 22 = 8），不是 mode 2。

#### 4.5.1 剔除 `FUN_00403090`（返回 0 = 可见）

证据：`render3/func_403090.c`、`render3/func_403090.asm`。

1. 距离测试：`|(entry.xyz + face.xyz) − (相机位置 + 相机+0x3c)|² ≤ this+0x1ee0`（`render3/func_403090.c:80-83`）。
2. 投影：`D3DXVec3ProjectArray(…, 12 个角, 视口 相机+0xcc, PROJ 相机+0x8c, VIEW 相机+0x4c)`（第 139 行），角点为 entry 的中心加上 `width/height/depth`（`entry+0x10/0x14/0x18`）的八个角。
3. 与场地矩形比较，有交则返回 0，否则返回 1（`render3/func_403090.c:280-299`）。

### 4.6 entry 循环 `FUN_00403990`（每帧，calc 中）

证据：`std2/func_403990.c`、`render/func_402720.c`。

```
for entry in entries (表顺序):
    if (entry.flags & 1):                                  (bit0 = 活动)
        活跃 = 0
        for quad in entry:
            if quad.vm.+0x390 != 0: 活跃++ 
            FUN_0043ee30(buf, vm_array, vm)               (步进该 quad 的 VM)
        if 活跃 == 0:
            entry.flags &= 0xfe                            (清 bit0)
```

- 注：循环中 `FUN_0043ee30` 为所有 quad 调用，活跃计数只看 `+0x390`（指令指针）。
- 注：`FUN_00403990` 先检查 `entry.flags & 1`，因此 bit0 为 0 的 entry 的 quad 不会步进。

### 4.7 深度与雾的最终状态

- **深度缓冲**：D16；ZENABLE 只在初始化时设为 1，之后没有写入者。ZWRITE/ZFUNC 的全部写入者是 `FUN_00402850`、`FUN_00402ca0`、`FUN_004450e0`（ZWRITE = 0）与初始化（ZFUNC = ALWAYS；ZWRITE 不设，取 D3D 默认 TRUE）。（2026-11-01 复核，全 dump 搜索 `SetRenderState(7/0xe/0x17)`。）
- 每帧的深度时间线：
  1. draw 7：`Clear(ZBUFFER, 1.0)`；8 个槽 VM 用 LESSEQUAL 测试、不写；**pass 0–7 用 LESSEQUAL 测试并写入**，按面文件顺序，互相遮挡；收尾 ALWAYS、不写。
  2. draw 9（层 0）：不测试不写。
  3. draw 0xa：层 17 不测试；**层 18、pass 8–11 用 LESSEQUAL 与 pass 0–7 的深度比较**，不写；收尾 ALWAYS、不写。
  4. 之后所有 draw 节点：ALWAYS、不写，深度无影响。
- 因此深度只由 pass 0–7 写，被 pass 0–7 自身、层 18、pass 8–11 读。alpha test（≥ 1/255）决定哪些像素写深度：半透明纹素也写。
- **雾**：
  - FOGENABLE 默认 1（§2.4），`FOGTABLEMODE = 0`（无表格雾），`FOGVERTEXMODE = 3`（LINEAR），`RANGEFOGENABLE` 未设（用 z 值）。
  - 近远：相机块 `+0xfc`（= `this+0x2b48`）、`+0x100`（= `this+0x2b4c`），颜色 `+0x114`（= `this+0x2b60`，ARGB）。与 `m2_motion.md` §12 的偏移一致。
  - 逐面开关：`FUN_00403a30` 对 render mode 8 的 quad 设 FOGENABLE = 1，其它为 0（`render/func_403a30.c`，`0x2000000` = mode 8）。
  - 当前 `FUN_00402ca0` 的 FOGENABLE 切换与 pass 有关（§4.4.2）。
- **（推测）** 顶点雾在 XYZRHW 路径（mode 0/2）里按顶点 z 插值；3D 路径（mode 8）按 WORLD 变换后的 z。

### 4.8 R4 绘制顺序（整体）

1. calc 节点 0xc（STD）→ 0x1a（游戏 VM 步进）等。
2. draw 节点 7：`FUN_00402850`：清屏、相机、pass 0–7（背景面）+ STD 槽 VM。
3. draw 节点 9：层 0。
4. draw 节点 0xa：`FUN_00402ca0`：pass 8–11 + 层绘制（夹在中间）。
5. draw 节点 0xb、0xd：层 1、2。
6. 0xf：相机矩阵 + 层 3。
7. 0x10–0x24：层 4–14（含 0x1d 子弹、0x19 道具、0x16 自机、0x14 敌人，见 §1.3）。
8. 0x29、0x2a：2D 层 15、16；0x2b：GUI；0x2d：2D 第 19 层。

### 4.9 R4a / R4b / R4c 的直接答案

**R4a：谁写 `quad+0x06`（VM 索引）？**

- 写入者：`FUN_00404450`（`render4/func_404450.c`，`render4/func_404450.asm`），由 `FUN_00418190` 调用（`emgr/func_418190.c:29`）。
- 写入语句：`psVar5[3] = (short)counter`；`counter` 从 0 开始，每个 quad 之后 `+1`。
- 顺序：外层 `local_4` 遍历 entry 表（0 → nb_objects−1，**表顺序**），内层沿 quad 链表（`psVar5 = psVar5 + psVar5[1]`，直到 `-1`）。计数跨 entry 连续，不按 entry 复位。
- 数组：`this+0x17c` 为 `nb_faces × 0x3ac`，所以 VM 槽 = `this+0x17c + counter × 0x3ac`（`asm` 中的 `IMUL EBX,EBX,0x3ac`）。
- 初始化：对每个 quad，`FUN_00404f30(this+0x178, VM槽)`（`AX = quad.script_index`，即 `quad+4`）。
- 文件中的 `quad+6` 恒为 0，所以运行时值是唯一的索引。

**R4b：谁设置/清除 entry+0x03 bit0，何时？**

- **置位**：`FUN_00404450` 在关卡初始化时对每个 entry 执行 `*(byte*)(entry+3) = 1`（`render4/func_404450.asm`：`MOV byte ptr [ECX + 0x3],0x1`）。注意是整字节赋值，不是 `|= 1`，所以会清掉 bit1 等其它位。
- **清除**：`FUN_00403990`（每帧 calc，由 `FUN_00402720` 调用）在 entry 的所有 quad VM 的 `+0x390 == 0`（指令指针为 0，即脚本结束）时执行 `&= 0xfe`（`std2/func_403990.c`）。
- **bit1（0x2）**：`FUN_00403a30` 在面可见（`FUN_00403090 == 0`）时 `|= 2`（`render/func_403a30.c`）。bit1 在 `FUN_00403990` 中不参与判断，渲染转储中也没有其它读取者（见 §7）。
- 因此：文件里的 0 不影响运行，运行时 `FUN_00404450` 先置 1，之后由脚本结束驱动清除。

**R4c：面遍历与合成**

- 遍历函数：`FUN_00403a30(this, pass)`（`render/func_403a30.c`），被 `FUN_00402850`（pass 0–7）与 `FUN_00402ca0`（pass 8–11）调用。
- 顺序：**文件顺序**（`this+0x18` 的面列表，直到 −1）。过滤条件：`entry+2 == pass`（`entry` 为 `this+0x14` 表中 `face.entry_index` 指向的项）。
- 合成：`quad.pos + face.pos` → VM `+0x340..0x348`（`pos_B`）。entry 位置不加。尺寸：`quad.width/height ÷ 精灵尺寸` → VM `+0x3c/+0x40`（缩放）。
- 之后 `FUN_004451c0` 按 render mode 绘制；mode 8 时 WORLD 矩阵由 `+0x24/28/2c`（旋转）与 `+0x334..0x354`（位置分量）构成（§3.3.3）。
- 剔除用 entry 位置与包围盒（`FUN_00403090`），不影响 `pos_B`。

---

## 5. R5 帧组合

### 5.1 后备缓冲与视口

- 后备缓冲 640×480（§2.2）。
- 帧视口 `帧+0xcc` = (0, 0, 640, 480, 0, 1)（2D，`0x491f60..0x491f74`，`render2/func_4216f0.c`）。
- 场地视口 `相机+0xcc` = (32, 16, 384, 448, 0, 1)（`0x491e48..0x491e5c`）。
- 切换点：
  - 每帧开始：`FUN_00439390` 设 2D 帧；
  - draw 7、0xa：`SetViewport(相机)`；
  - 层 3–14（节点 0xf 起）：相机；
  - 层 15、16、19（节点 0x29、0x2a、0x2d）：2D。

### 5.2 Clear

- `FUN_00402850` 每帧（STD 启用时）：
  1. `Clear(0 矩形, ZBUFFER, 颜色 0, z = 1.0, stencil 0)`（清深度）；
  2. `Clear(1 矩形 = 场地 {32, 16, 416, 464}, TARGET, 颜色 C)`，其中 C = `STD+0x2b60 & 0xffffff`（雾色，RGB），条件是 `(this+0x2a18 & 4) == 0` 或 `this+0x2a34 > 0x21`（`this+0x2a34` 为 STD 帧计数，§4.3）；否则 C = 0（黑色）；
- 场地外（左边 32 像素、右边 192 像素）不在 Clear 范围内，由前景 `front.anm`（§5.4）覆盖。
- 2D 场景（标题、菜单等）的清屏不在本次取证范围内（§7）。

### 5.3 每帧流程与节流

见 §1.4。摘要：

- 节流：距上一帧 < 1/60 s 则跳过（`DAT_00470c90`）。
- 跳帧：`DAT_00491d66 + 1` 帧绘制一次（写入者未找到，§7）。
- Present 之后，若 `_DAT_00474e36 & 0x800` 非零，则调用 `FUN_00452abc("snapshot")` 并保存 `snapshot/thNNN.bmp`（`render/func_4391f0.c`，文件名按 000–999 递增）。这是原作自带的截图机制，与 `m4_render.md` §3 的 L3 有关。

### 5.4 `front.anm` 与 HUD

- GUI 构造 `FUN_00413980`（`render6/func_413980.c`）：`FUN_00447280(6, 游戏对象, "front.anm")` → 存入 `GUI+0x9ec8`；随后注册 calc 节点 `0x415ae0`（优先级 0x18）和 draw 节点 `0x415af0`（优先级 0x2b，`render6/func_413980.asm`）。
- 因此 `front.anm` 在 draw 节点 0x2b 绘制（在层 15、16 之后、层 19 之前）。**（推测）** 具体绘制内容（边框、HUD）由 `FUN_00415af0` 决定，未追踪。
- ANM 槽 6：游戏对象 `+0x3ad06c + 6·4`（与 `FUN_00402440` 的 `DAT_00491c10 + 0x3ad06c + uVar4*4` 一致，`render3/func_402440.c`）。

### 5.5 窗口与全屏

- 窗口：客户区 640×480（§2.1），`Windowed = TRUE`，格式 UNKNOWN（用桌面格式），垂直同步 DEFAULT。
- 全屏：`Windowed = FALSE`，640×480，60 Hz，垂直同步 ONE（第一次尝试）。
- `DAT_00491d65` 决定模式，由对话框结果写入（§2.2）。

---

## 6. 与 `m4_render.md` 的对应（给 A1/S2/S3/D1 的摘要）

| 任务 | 本文档的依据 |
|---|---|
| A1 `anmdraw` | §3.2 render mode 表；§3.3 顶点与半像素；§3.4 层；§3.5 混合与采样 |
| S2 STD VM（op 9/14、quad VM） | §4.3 op 表；§4.9 R4a/R4b |
| S3 `stddraw` | §4.4 pass 与状态；§4.5 面遍历；§4.7 深度与雾 |
| D1 绘制链 | §1.2、§1.3 表（优先级、回调）；§1.4 每帧顺序 |
| G2 `gfx`（裁剪、视口、批处理） | §3.3 批处理与 flush；§2.5 矩阵与视口；§5.1 |

---

## 7. 开放问题

1. ~~mode 2 的 z 分量~~：已确认是 bug，照搬（§3.9）。原文：`FUN_00443290`（mode 2）用 `+0x350 + +0x348 + +0x33c` 作 z，其它路径（mode 0、1、3）用 `+0x354`。疑似偏移错误，需核实（`render/func_443290.c` 与 `render/func_443080.c`）。
2. **混合模式 b=2 不一致**：`FUN_00442670` 的 2D 路径（`FUN_004425a0`）对 b=2 不改 DESTBLEND，3D 路径（`FUN_004423e0`）改为 ONE。需确认 b=2 的预期效果。
3. **mode 4–7 的角点构造**：`FUN_00443b60`、`FUN_00444240` 只看了开头，未逐行核对。`render5/func_443b60.c`、`render5/func_444240.c` 已导出，但未完整阅读。
4. **STD 面位置的世界坐标来源**：`+0x334/338/33c` 的写入者未追踪（`render/func_443290.c`、`render/func_443080.c` 直接读取）。`FUN_00403a30` 只对 render mode ≥ 4 写 `pos_B`（+0x340..0x348），因此 mode 0–3 的 STD quad 位置从哪里来（ANM 脚本、还是其它函数）未确认。mode 8 的 3D 投影由 WORLD 矩阵承担（§3.3.3），而 `FUN_00403090` 只用于剔除。
5. **第 19 层表**：`FUN_00448900` 构造的列表的语义，哪些 VM 进入这里，未确认。
6. **`DAT_00491d66`（跳帧）**：渲染转储中没有写入者。配置来源未找到。
7. **`DAT_00491d62` 与 `th10.cfg +0x1a`**：`FUN_00439890` 把初值 −1 规范为 0，配置读取的位置未追踪。
8. ~~`+0x2ff` 的写入者~~：它是 `+0x2fc` 颜色的 alpha 字节，由设置颜色与 alpha 的 ANM 指令写入（§3.9）。
9. **0x800 标志的消费者**：ins_73 翻转 0x800，但渲染转储中的 2D/3D 路径没有读取它。可能在 STD 相关函数或尚未导出的函数里。
10. **bit1 的读取者**：`FUN_00403a30` 对 entry 置 `|= 2`，渲染转储中无人读取。
11. ~~STD 深度~~：已解决（§4.7）。深度只由 pass 0–7 写，被 pass 0–7、层 18、pass 8–11 读。待查的是层 17、18 里放的是什么（哪些 ANM 脚本用 ins_68 设层 17/18）。
12. **玩家和子弹的节点启用**：自机（`FUN_004247f0`）的绘制节点在构造时被清 bit1，未追踪它何时被启用；只核对了 STD 和游戏对象。
13. **`front.anm` 的绘制内容**：`FUN_00415af0`（draw 0x2b）未追踪。
14. **`FUN_00404f30` 的 AX 与 ESI**：由调用方寄存器传入（`render4/func_404450.asm` 的 `MOVSX EAX,word ptr [EDI+4]` 与 `ESI = 槽地址`），与 op 14 中的调用方式是否一致未核对。
15. **2D 投影常量的最终核对**：`π/20`（`0x470ba0`，double）已读出，但 `FUN_00421480` 的 `fVar4 / fVar5` 与 `D3DXMatrixLookAtLH` 的 eye 参数顺序是否与代码完全一致，仅由公式推出。
16. **gamma 与 MSAA**：仅做了渲染相关转储的扫描，未在整个 exe 中扫描 `SetGammaRamp`。
17. **标题与菜单的清屏**：非 STD 场景的清屏方式未追踪。
18. ~~**`m2_motion.md` §1 的勘误**~~：已于 2026-11-01 改正（§1.2）。
19. ~~**draw 链的 `FUN_00420470` 节点**~~（0x1、0x28、0x32）：已追踪，见 §1.3 表。

---

## 8. 证据目录（本次新增）

| 目录 | 内容 | 来源 |
|---|---|---|
| `.tmp/ecl_dump/render/` | `FUN_00439890`、`FUN_00439d20`、`FUN_004451c0`、`FUN_00442670`、`FUN_00442f50`、`FUN_00402850/402ca0/402720/402230/402640`、`FUN_00403a30`、`FUN_00444760/444ce0`、`FUN_004423e0`、`FUN_004425a0`、`_callers.txt`（引用设备全局 `DAT_00491c30` 的函数，`--ref-funcs`） | 初次 Ghidra 运行，`--asm --context 4` |
| `.tmp/ecl_dump/render2/` | `FUN_00439730`、`FUN_00449c00/449d40/449ed0/449f60/449aa0`、游戏对象节点回调 `0x4485d0`…`0x448810`、`FUN_00448740`、`FUN_004216f0`、`FUN_00421480`、`FUN_00438880` 等 | 第 2 次 |
| `.tmp/ecl_dump/render3/` | `FUN_00402440`、`FUN_004451c0`、`FUN_00443080/443290/443f80/444580`、`FUN_004423c0`、`FUN_00442fe0`、`FUN_00403090`、`FUN_004043f0`、`_writes.txt`（`--writes 0x6`） | 第 3 次 |
| `.tmp/ecl_dump/render4/` | `FUN_00404450`（quad 初始化）、`FUN_00448980/448860/448900`、`FUN_00448740`、`FUN_00444580/443f80/443fb0/4445c0/444760/444ce0/4450e0`、`FUN_004452f0`，`_callers.txt` | 第 4 次 |
| `.tmp/ecl_dump/render5/` | `FUN_00443b60`、`FUN_00444240`、`FUN_00404f30`、`FUN_00418190`、`FUN_00420cd0`、`FUN_0044a0c0`、`FUN_00463b2c`，`_callers.txt` | 第 5 次 |
| `.tmp/ecl_dump/render6/` | 管理器构造函数 `FUN_00401110`、`4055c0`、`405e20`、`4089c0`、`40ae70`、`40b560`、`40d280`、`413980`、`418e30`、`41ad90`、`41c120`、`41fac0`、`422150`、`4247f0`、`42b4d0`、`42caa0`、`420470`（`.asm` 中找 `MOV EDI` 与 `CALL 0x00449b70/ae0`） | 第 6 次 |
| `.tmp/ecl_dump/render7/` | `FUN_00405040`、`FUN_004049a0`、`FUN_00403c80`，`_callers.txt` | 第 7 次 |
| `.tmp/ecl_dump/render*_run*.log`、`render_run1.log` | 各次运行的日志（`render_run1.log` 为第 1 次） | |
| `.tmp/ghidra_venv/` | 专用 Python 虚拟环境（jpype 1.7.1 + pyghidra 3.2.0，来自 Ghidra 12.2 的本地 wheel） | 仅用于运行 `tools/ghidra/dump.py` |

此外引用了已有的 dump（不属于本次新增）：`.tmp/ecl_dump/std/`、`.tmp/ecl_dump/std2/`（`func_403850.c`、`func_403990.c`、`func_4215a0.c`）、`.tmp/ecl_dump/m2reg/`、`.tmp/ecl_dump/m2calc/func_449c40.c`、`.tmp/ecl_dump/anmvm/func_43ee30.c`、`.tmp/ecl_dump/cfg/func_438ad0.c`、`.tmp/ecl_dump/cfg2/func_439890.c`、`.tmp/ecl_dump/emgr/func_418190.c`、`.tmp/ecl_dump/emgr/func_417870.c`、`.tmp/ecl_dump/top/`。PE 常量读取：`.tmp/scratch/pe_read.py`。thtk 格式：`.tmp/thtk-src/thstd/thstd.h`、`thstd.c`。

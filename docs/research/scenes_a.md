# 关外场景 A 普查（t10）

日期：2026-11-08。版本：TH10 v1.00a。阶段 A 文档先于运行时代码；后续实现签名以交接第 9 节为准。

## 1．取证范围与方法

完整读取 `.tmp/codex/rules.md`、总账、harness、open_items 及 t1–t9 相关交接第 6／9 节。只读原 exe／dat／回放，不启动原作、probe 或 Ghidra，不修改关内逻辑、GUI、golden。

依据为主树 `.tmp/ecl_dump/t4_native/exe.asm`，已有 `r11`、`m5clear`、`m2reg`、`r2_ctrl`、`t5_save` 与 `round2` 的 C／ASM。选取的原汇编副本在 `.tmp/codex/t10/evidence/{switcher,title,loading,input}.asm`；PE 常量／表由 `.tmp/scripts/t10_native.py` 只读解析。补充 C 请求列在 `.tmp/codex/t10/dump_requests.md`，不等待。

## 2．监督者／游戏模式切换

| 项目 | 地址与行为 |
|---|---|
| 注册 | `FUN_00420470`：当前 `DAT_00491fb4＝−2`、请求 `DAT_00491fb8＝0`、前一个场景 `DAT_00491fbc`；calc 节点 `FUN_0041ff80`，draw 首尾 `FUN_00420000／004200d0`。 |
| 请求 | `FUN_0040ac90`，`0x40ac90–0x40acb2`：通常写请求号，`DAT_00491ff4 & 0x1000` 时写 2；`FUN_0041ff30` 是不做该覆盖的直接写入。 |
| calc／切换 | `FUN_0041ff80` 先跑声音控制与输入／资源加载队列，再在监督者＋0x648 为 0 时调用 `FUN_004218d0`。当前号相同则不切；不同则记前场景、清屏色设 `0xff000000`、执行分支，最后当前号取规范化后的请求号。 |
| 析构 | `FUN_004203f0` 释放 stage、title、loading、ending、replay 管理器。具体 `FUN_00417c80／0042cb60／0041fb50／0040b7b0／004294a0` 分工保持，不把整个 ANM 世界每帧重建。 |

`FUN_004218d0` 的表在 `0x421b0c`，场景号不是标题对象的菜单页号：

| 请求号 | 原作处理 |
|---|---|
| 0 | 规范化成 1，构造 loading（`FUN_0041fd00`）；失败请求 3。 |
| 1／2／5／6／8／9 | switch 缺省路径；1／2 为启动／系统退出状态，不伪造新画面。 |
| 3 | 全局释放并返回 calc 结果 4（退出）。 |
| 4 | 从 7 释放 stage，从 14 释放 ending；`FUN_0042cd50` 构造 title。 |
| 7 | 从 4 释放 title；fresh-run＝1；`FUN_004180e0` 构造正常关卡。 |
| 10 | 释放 stage；fresh-run＝1；回到初始关卡 `DAT_00474c80`，规范化为 7。 |
| 11 | fresh-run＝0；必要时释放旧 stage；规范化为 7，按新当前关卡构造。 |
| 12 | 从 4 释放 title；fresh-run＝1；规范化为 7，播放回放／demo。 |
| 13 | 释放 stage；fresh-run＝1；规范化为 7，沿用 t5 的 `RestartStage／OnContinue`。 |
| 14 | 必要时释放 stage，`FUN_0040b940` 构造 ending（t11）。 |
| 15 | 从 2／7／14 返回时释放相应对象；规范化为 4，标题入口原因 `DAT_00491c00＝3`（Player Data，t11）。 |

Go 保留 `game.Scene.Update／Draw` 和已有 `GoTo／SetScene`；增加带原场景号的延迟工厂与可选生命周期。既有 `ending.After`、`Stage.OnContinue`、`stage.NewPlayback` 不改签名。关间实际 ECL 与自机更新仍由原 Stage 实现，切换器不推进额外关内帧。

## 3．启动加载

| 项目 | 地址 |
|---|---|
| 构造／分配 | `FUN_0041f850／0041fd00`，对象 0x3f0 B，`DAT_00477820`。 |
| 注册／加载线程 | `FUN_0041fac0` 注册 calc 3、draw 2；线程入口 `FUN_0041f990` 加载 `sig.anm`（文件槽 1）、ascii／text／声音／score，以及 front／bullet。 |
| draw | `FUN_0041fdd0`，由 `0x41fef0` 调用：就绪时尾插 `sig.anm` 脚本 0（层 15）；ascii 脚本 6 放 `(480,392,0)`；每次 draw 递增对象＋0x3ec。 |
| calc | `0x41feb0`：加载完成后关闭启动 ASCII 的 calc／draw，清系统 0x1000，请求 4。 |
| 最短显示 | title 初始化 `FUN_0042c9f0` 的 `0x42ca22–0x42ca58` 等 loading draw 次数到 300；系统 0x80 可跳过。不是旧实现的 180 个 Update，也不是额外运行原资源目录下程序。 |
| 析构 | `FUN_0041fb50／0041fd70`：摘两个回调、释放 sig 等启动资源与线程对象。 |

嵌入资源加载是同步、跨平台的项目适配；保留 draw 计数与可验证的就绪／释放顺序，不模拟 Win32 线程或人为随机耗时。

## 4．标题对象、主菜单与演示

标题、Option、难度、角色、装备、练习关卡、回放、数据、Music Room **共用一个 0x5acc B 标题对象**，`DAT_0047784c`。`FUN_0042c920` 构造清零；`FUN_0042caa0` 注册 calc 6（`0x42d2e0 → FUN_0042cdf0`）、draw 3（`0x42d2f0 → FUN_0042d260`），加载 `title.anm` 槽 0x19 与 `title_v.anm` 槽 0x1a；`FUN_0042cb60／0042cdb0` 析构。

主要成员：＋0x1c 为页状态，＋0x20 为子状态；＋0x24／0x28 为当前／上次光标，＋0x2c 为项数；＋0xf4 为环绕，＋0xf8 与＋0xb4 为不可选项；＋0x2b0 的 ZunTimer 当前值在＋0x2b4；＋0x2c4 开始按脚本号索引根句柄。辅助函数：`FUN_0042c5c0` 换页／子状态归零，`FUN_0042c620` 换子状态，均清计时器；`FUN_0044be20／0044be70` 保存／恢复光标与项数，深度最多 16；`FUN_0044bea0` 环绕／钳位并跳过不可选项。

ANM 共用句柄接口：`FUN_0042c670` 尾插 title 脚本（层 15、备用 RNG）；`FUN_0042c6d0` 发标签 1 并清句柄；`FUN_0042c730／0042c750` 发标签 2／3；`FUN_0042c770` 发指定标签，包含根的子链。不要把 `Restart` 当 `Interrupt`。

| 页状态 | calc 地址／归属 |
|---|---|
| 0 | `FUN_0042cdf0` 的初始化：入口原因 `DAT_00491c00` 为 0→初次标题；1→主菜单；2→回放页；3→Player Data 页。非 demo 返回播放标题曲 `bgm/th10_02.wav`；demo 返回恢复旧难度并清 0x20，不重复启动曲。 |
| 1 | `FUN_0042d300`：初次标题。缺根 0x58 时生成 0x58、0x5a、title_v 0；子状态 1 在 timer >10 进入 2；按下沿 `0x160b` 进入 4、向 0x5a 发标签 6、SE 0x20；timer ≥30 换主菜单页 2。 |
| 2 | `FUN_0042d420`：八项主菜单。根 0；进入活跃状态需 timer >10，先标签 3，再 `17＋光标`；移动先标签 3，再 `7＋光标`。上／下按重复；X／Esc 先移到 Quit，已在 Quit 才退出；确认发标签 6。 |
| 3 | `FUN_0042cdf0`：请求退出；系统 0x1000 的分支保持原覆盖。 |
| 4／5 | Option／Key Config，见第 6 节。 |
| 6／7／8／9 | 难度／角色／装备／练习选关，见第 5 节。 |
| 10／13 | 原声音控制过渡（`FUN_00420c00`），不新增页面。 |
| 11／12／14／15／16 | Player Data／Replay／Music Room／其它数据页，分别 `FUN_00431ee0／004315c0／00433ef0／00432cb0／00433570`，由 t11 实现。 |

主菜单顺序为 Game Start、Extra Start、Practice Start、Replay、Player Data、Music Room、Option、Quit。确认后子状态 4 的 timer ≥20 才换页。普通／练习进入难度页，Extra 临时记旧难度并设 4；Replay→12，Player Data→11，Music Room→14，Option→4。

演示是存在的：`FUN_0042cdf0` 在页 1／2 每帧检查**当前按住** `0x160b`，没有这些键才递增 `DAT_00474ca4`，≥900 进入 demo；方向与 Shift 不清待机计数。置 `RunFlagDemo＝0x20`，清未知 bit 0x40。表 `0x476ff8` 的顺序为 demo1、demo2、demo3、demo0，`DAT_00474ca8` 循环；加载第一个有效章，恢复角色／装备／难度／初始关卡，请求 12。Go 通过 t5 的 `NewPlayback` 进入，不创建第二套输入流。

## 5．难度／自机／装备／练习选关

全部使用标题对象子状态 0／1／2／3／4；每次换页计时器清零，calc 帧尾 Tick，不提前加帧。

| 画面 | 函数、脚本与门槛 |
|---|---|
| 难度 | `FUN_00430320`：提示 0x5e、选择根 0x77（四难度）／0x78（Extra）、0x62 标题、ascii 8。入场 timer >6；上／下仅普通难度；确认标签 6 与选中子脚本 `0x6d＋难度`／0x71 标签 2，SE 10；timer ≥14 进选人。取消 SE 11，timer ≥6 回主菜单。 |
| 角色 | `FUN_004306a0`：提示 0x63，选择根 0x7d。左右重复；入场 timer >6；确认选中的 `0x79＋角色`、`0x7b＋角色` 标签 6，另一个标签 1；timer ≥14 进装备，SE 10／11／12。Extra 按 ST 六个机体解锁字节禁用整角色。 |
| 装备 | `FUN_00430a60`：提示 0x64，根 `0x96＋角色`。上下重复；入场 timer >6；Extra 各机体按对应解锁字节禁用；普通记录中通关次数为 0 的名字子脚本 `0x8a＋角色*3＋装备` 隐藏。确认发选中 `0x7e＋角色*6＋装备` 标签 6；练习 timer ==10 换练习页；普通 timer ==10 生成加载标记／黑色淡出，timer ≥40 写初始关卡 1／7、请求 7、BGM fade 参数 float32 `0x40c00000`（6）。 |
| 练习 | `FUN_00430ff0`：根 0x6a 与 `0x6b＋角色`；入场 timer >10，记住 `DAT_00491c08` 光标。确认看机体／难度／关卡记录 `+0x4e9`（Cleared 字节）；未解锁 SE 0x25，不换状态。成功 timer ==10 生成加载标记／黑色淡出，timer ≥40 写当前／初始关卡 `光标＋1`、请求 7、6 秒 BGM fade。取消 timer ≥6 回装备页。 |
| 练习文字 draw | `FUN_00431410`，由 title draw 页 9 分派。原作遍历六面，选中解锁为黄 `0xffffff00`，未解锁 `0xffdfdfdf`，非选中 `0xff808080`；确认状态按 timer%4 闪黑。坐标灵梦 `(168,216)`、魔理沙 `(296,216)`，行距取 `0x470c0c`。任务要求只显示已解锁面，Go 入口／可选列表按该要求过滤，文档明确此显示策略与原六行占位的差别。 |

解锁不能用总分／全局 ExtraUnlocked 代替：Extra 字节的原对象地址为＋0x1d888…＋0x1d88d，对应 ST 的＋0x1c…＋0x21（CR 7×0x437c，原对象前两个指针占 8 B）；练习读 t8 的 `Save.Progress[character*3+shot].Stages[difficulty][stage-1].Cleared`。原 `RecordStageClear` 同时写 Available／Cleared。没有证据把第一次练习等同 bit 8／Boss 练习。

入口创建 `Game.NewState`，普通 stage／manager mode＝0；练习只调用 t8 `GameState.Practice(stage)`；回放保持 1／1，demo 再 OR `RunFlagDemo`。数字 1–9 按住确认练习时写 `PracticeLives`（`0x43118c–0x431315`），默认 0→9 残；保留 `StartStage` 与 fresh-run，不靠 `SingleStage` 猜原 flags。

## 6．Option／手柄配置

主 Option：`FUN_0042d920`，根 script 1，六项（Hint、BGM 音量、SE 音量、Key Config、Default、Quit）。入场 timer >6；出场 timer ≥10。上下重复先于取消；X／Esc 先移到 Quit，第二次退出；Z／Enter 只激活后三项。Key Config 先标签 6，timer ≥10 换页 5 并压栈；返回先恢复第 3 项，随后 Option 子状态 0 在 `0x42d953` 又置为 0，按原指令保留。

| 配置字段 | 原地址／分支 |
|---|---|
| Hint | cfg＋0x22（`DAT_00491d6a`），0＝On、1＝Off、2＝Auto，左右环绕；`0x42dd8b／0x42e044`。对子脚本 0x1f／0x20 **设置精灵** 0x2e／0x2f（On）、0x30／0x31（Off）或 0x2c／0x2d（Auto）。`FUN_00449670` 是设置精灵，不是重启脚本。原 Hint 管理器在 `0x418e8b／0x419009` 读该字节；关内 Hint 演出不在本任务范围。 |
| BGM 音量 | cfg＋0x20，左右每次 5，钳 [0,100]；`0x42dd58／0x42e021`。 |
| SE 音量 | cfg＋0x21，同样每次 5；`0x42dd25／0x42df27`；停在该项每 60 帧试放 SE 4（`0x42dccb`）。 |
| 默认 | `0x42e2ce` 只置 BGM 100、SE 80、Hint 0；**不清整个 cfg、不把三态 Hint 缩成复选框**。 |
| 数字 | `FUN_0042e5a0`：BGM 三位子脚本 0x25–0x27、阴影 0x29–0x2b；SE 三位 0x2d–0x2f、阴影 0x31–0x33。正常精灵 `0x33＋digit`，阴影精灵 `0x3d＋digit`；低于 100 隐藏百位与其阴影、低于 10 隐藏十位与其阴影。 |

Key Config：`FUN_0042f540`，根 script 2，七项（射击／灵击／低速／暂停／快进五个手柄键，Default、Quit）；入场 timer >6、出场 ≥10。暂存五个 word 于 title＋0x59cc…＋0x59d4，原输入表 `0x474e88／8a／8c／8e／98`。取第一个按住的手柄键 0…30，经 `FUN_00430250` 写暂存，重复目标与旧键交换；数字 helper `FUN_0042f8b0`。Default 的 `0x42f810` 复制当前 live 表，不重新生成工厂默认。Quit 确认才提交 cfg＋4／6／8／0xa／0x14；Cancel 在 Quit 上退出并恢复当前表；中间四个保留键字不改。保留纯键盘菜单与无手柄输入的空操作，不添加系统特定 API。

## 7．暂停（与 t5 的状态 6–13 接续）

`FUN_004223f0` 的状态 0 在非 demo、Esc 按下沿或系统 0x10、stage 更新节点启用、stage timer >29 时调 `FUN_00422ab0`。暂停位是 **stage＋0x58 的 0x10**，不是菜单对象 State 本身；t5 后半也使用冻结位。

| 状态／函数 | 原行为 |
|---|---|
| 打开／状态 1 | `FUN_00422ab0`：timer 0，保存游戏速度＋0x2c0、速度设 1，冻结 stage；capture 区 `(32,16,384,448)`；ascii 0 背景与 front 0x79 标题（层 15、备用 RNG）；SE 0x20（se_pause.wav，表号不是 wav 表索引）；BGM Pause；回放删起始标签。 |
| 状态 2 | `FUN_00422c80`：timer >9 后项数为正常 3／回放 2，初始 0。上下按重复；选中标签 `7＋光标`。三项为 Resume、Return to Title、Retry。Z／Enter：Resume→3；Title 正常→4、回放→3；Retry→4。R（0x4000）／Q（0x200）直接进入 3 并选 Retry／Title；Esc 选 Resume 并进 3，优先级保留。 |
| 状态 3 | timer >11：光标 0 调 `FUN_00422c30`，清冻结、恢复速度／BGM；光标 1 请求 4；光标 2 请求 10（回初始关卡）。 |
| 状态 4 | timer ==20 压栈、项数 2、默认 1（No）；timer ≥30 才读上下／确认。确认→5；Esc→3（Resume）。确认框 root 0x7f 发对应标签。 |
| 状态 5 | timer ≥20：Yes 恢复压栈选项、发退出标签、进入 3；No 恢复选项回 2。 |
| 解除 | `FUN_00422c30`：清 stage 冻结位、BGM UnPause、删额外句柄、恢复游戏速度。 |

暂停本身没有新的回放保存页：保存入口在 t5 的正常 Game Over／Clear 状态 6–13。暂停返回／重开不偷偷调用编码器；回放暂停只有 Resume／Title。原 pause draw `FUN_00422660` 对 1–5 无附加文字，画面来自 front／ascii World，不能再用链外手画黑幕。

## 8．共用输入／绘制／淡入淡出接口

`FUN_0044a950` 是菜单 16 位输入快照：当前＋0、上帧＋2、重复＋4、按下沿＋6、释放沿＋8；每位有 word 计数。按住递增到 26 触发，减 8，因此**第 26、34、42…帧重复**，释放清零。`FUN_0040ace0` 返回 `(repeat|pressed)&mask`。`FUN_0040ac20` 是三字回放辅助，不能用其低位重复缺陷代替菜单采样。

键位：Z＝1、X＝2、Shift＝4、Esc＝8、上下左右＝0x10／20／40／80、Ctrl＝0x100、Q＝0x200、S＝0x400、P／Home＝0x800、Enter＝0x1000、D＝0x2000、R＝0x4000（键盘采样 `FUN_0044a5f0`）。确认按下沿掩码 0x1001；普通标题取消 0xa；暂停 Esc／Q／R 独立。两相反方向按代码的先后分别执行，不擅自 XOR 消除。手柄轴使用原四分之一范围边界（标准 −1…1 对应 ±0.5），五个键 word 来自 cfg。

绘制共用原 Supervisor：BeginFrame→Start→屏幕平面→World 各层→必要的 native 文本→Finish→EndFrame；纯无头设备由 `DrawPorted(*supervisor.Supervisor)` 入口使用。ANM 的 alpha／移动／缩放和标签负责菜单入场。选装备／练习确认还调用 `FUN_0040c540`（确保 ascii 6 加载标记），不是全屏捕获；`FUN_0043c8b0` kind 5 则创建真实黑色淡出，calc `FUN_0043c230`（优先级 0x0e）、draw `FUN_0043c1a0`（调用者 EBX＝0x2b）。Alpha 先取 `trunc(timer.F*255/duration)`、duration 后为 255，之后 Tick；timer ≥duration＋2 析构。kind 0 为相反的 `FUN_0043bd40`，Alpha＝`trunc(255−timer.F*255/duration)`，timer ≥duration 析构。全屏矩形 `(0,0,640,480)` 使用原 `Renderer.Rect`。共用签名在交接第 9 节；kind 1 的关内震屏仍属于 t7，不变更。

## 9．t11 接线约定

实现按以下方向提供接口；实现后的准确 Go 签名、调用地址与差别记在 t10 交付第 9 节（`git show b4df88d:docs/handoff/codex_t10.md`）。

- `game.SceneID` 保留原 0…15 编号；带原号的请求／延迟工厂，`GoTo／SetScene` 原签名不变；可选 SceneMode／Enter／Leave。
- `input.MenuInput` 提供 Current／Pressed／Released／Repeat 与 `Move(mask)`；从 `input.State` 只读当前帧，读取本身不推进计数。
- `scene/ui.TitleController` 提供 State／Substate／Timer／Selector／World／Env 与 `SetState`、`SetSubstate`、`Spawn`、`Interrupt`、`LeaveScript`、`Child`、`DrawPorted`；页 11／12／14／15／16 的 calc／draw 留 t11 callbacks，不复制其状态机。
- 原 `FUN_0042c5c0／0042c620／0042c670／0042c6d0／0042c770` 的 Go 共用接口给 t11；标题主菜单在 20 帧过渡完成后调用 t11 的回放／数据／Music Room 页 hook。
- `ending.Ending.After`、`Stage.OnContinue`、`stage.NewPlayback` 与 t8 的 GameState 模式／存档 API 保留；演示也走同一回放入口。

## 10．实现后复核补充

- 音量不是线性 PCM：FUN_0044d4e0（BGM）以 1－(1－p×0.01)² 乘 (baseDB＋5000)、截断后减5000；0x43e2da–0x43e31c（SE）以立方代替平方，每个id使用原表baseDB。0音量写−10000。FUN_0042e5a0 的 DAT_0049785c 额外写值没有运行时读者。
- demo 的原 FUN_00418190 在 0x4186aa 的2940帧创建60帧fade、0x4186c4 的3000帧请求4；当前按住0x160b或menu冻结位可提前退出。
- 初始Power：0x417976–0x41798c 对所有非一面设0x50（项目400），包括Extra，不限于练习。
- 原ASCII由标题draw优先级3产生、实际文字由稍后的ASCII提交；共享Go绘制接口直接提交该文字pass，不能把它画到不透明背景之前。
- 五轮完整验收已通过；准确签名、兼容边界、t11调用点均已写在主树交付文档第9节。

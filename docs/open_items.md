# 待决清单（第五轮）

更新时间：2026-11-21。第一至五轮均已提交。交付文档 `docs/handoff/` 已删除；各条的详细依据在原交付第 6、8 节，用 `git show b4df88d:docs/handoff/codex_tN.md` 查看。本清单只列未关闭事项，已关闭的列在末尾。

## 等 dump

多数只供独立复核，不阻塞代码；已有证据的事项不重复申请。

| 编号 | 来源与事项 | 当前状态 |
|---|---|---|
| D8 | t12：魔理沙 B，`th10_03` 4098…4102 公共命中／死亡路径（`.tmp/codex/t12/dump_requests.md` §1） | 合并树在证实半帧 3887 之后至原作末帧全部匹配，4100 差分已消失；待独立复核后关闭 |
| D9 | t12：魔理沙 C 槽 735／1960 动态快照（同文件 §2） | 替代回放 `th10_10` 在 5352 截断前无分歧；只在真人录制的 C 出现分歧时需要 |
| D10 | t10、t11：标题、选择、结局、Staff、Music Room、Player Data、回放详情 | 代码已由全程序 asm 闭合，dump 仅供复核 |
| D11 | t13：函数 dump 与补采数据（`.tmp/codex/t13/dump_requests.md`） | 函数 dump 供复核；补采见 R7 |

## 等用户目视／试听

用户数据放在自己指定的可写目录（`TH10_DATA_DIR`），不写原作数据 junction。

| 编号 | 来源与项目 | 怎么看（摘要） |
|---|---|---|
| V1 | t1 §8（`git show b4df88d:docs/handoff/codex_t1.md`）：自机本体、判定点、蓝闪、被弹火花 | 两角色静止／左右起动／反向／回中，Shift 判定点淡入淡出，复活／灵击无敌蓝闪，被弹 1＋32 个火花 |
| V4 | t3 §8（`git show b4df88d:docs/handoff/codex_t3.md`）：GUI、Boss HUD、MSG、字体、弹出字 | 侧栏与数字、Power 点、信仰保持条迟滞、血条色段、星数／警告色、两半身与亮暗、两行基线 375／393、名牌／曲名；弹出字出生帧 Inf／NaN 无闪烁 |
| V6 | t4 §8（`git show b4df88d:docs/handoff/codex_t4.md`）：立体声声像 | `TH10_TEST_AUDIO=1 go test -count=1 ./audio -run '^TestPlaySEPanning$' -v`：−192／0／＋192 时另一侧衰减 10 dB 仍可听到；游戏内发弹／击破声像 |
| V7 | t5 §8（`git show b4df88d:docs/handoff/codex_t5.md`）：关卡内排行／回放保存、截图背景、mode 13 | Game over／练习／Extra clear 菜单、八字名字、槽位保存后回标题播放且不加进度；场地捕获无偏移；重开续关计数封顶 9 |
| V8 | t6 §8（`git show b4df88d:docs/handoff/codex_t6.md`）：副机、射击与灵击 | 副机入场／满火力叠加／退出；魔理沙 A 拖尾、C 锁定与解除延迟；六机体射击与命中图、魔理沙 B 激光亮暗；信仰保持矩形（cfg 0x80）；`th10_05` 第 5 章 1649 帧灵击环；满火力 P 转 0x189 |
| V9 | t7 §8（`git show b4df88d:docs/handoff/codex_t7.md`）：符卡与震屏 | 宣言名字与八位奖励、历史两位数；失败横幅；三宣言牌上移与退出；cdbg 渐入与 STD 交接；Extra Boss 名字与立绘；ECL 337 震屏 |
| V10 | t8 §8（`git show b4df88d:docs/handoff/codex_t8.md`）：模式、存档与得分 | 旧 save.json 导入后 score.dat 写入与重启保持；练习后续关卡残机与 rank；extend 门槛 SE 与 One Up；六面 MSG 奖金与 120 帧后的 ending；原 scoreth10.dat 互通（需用户操作原作） |
| V11 | t10 §8（`git show b4df88d:docs/handoff/codex_t10.md`）：标题、选择、Option、暂停 | sig 与 300 次 draw；光标中断与重复；待机 demo 1／2／3／0、2940 帧 fade 与 3000 帧回标题；Option 音量／Key Config；手柄阈值（需实机）；暂停确认 No 默认、Q／R |
| V12 | t11 §8（`git show b4df88d:docs/handoff/codex_t11.md`）：结局、Staff、Music Room、Player Data、回放选单 | 十二机体 Good／Bad 与 Staff 时序；Music Room 解锁与二次试听；Player Data 统计与符卡历史；回放列表／详情／选章；可用 `go run ./tools/th10scenes -scene <name>`（ending／staff／music／data／replay） |
| V13 | t12 §8（`git show b4df88d:docs/handoff/codex_t12.md`）：魔理沙射击 | B 首按 Z 激光数与枪口跟随；A 复活后首枚火箭；C 低速锁定与 30 tick 延迟；5.00 显示与四副机上限 |
| V14 | t13 §8（`git show b4df88d:docs/handoff/codex_t13.md`）：通关、接触与震屏 | 最终 MSG op 20 后 draw 0x31 全屏黑淡入；BossDead 震屏与六面 0.25 速度 MSG；本体接触后的决死／死亡／复活 |
| V15 | t14 §8（`git show b4df88d:docs/handoff/codex_t14.md`）：BGM q1、js 有损贴图、字体观感 | Music Room 听高频与循环接缝；`.tmp/codex/t14/compare`、`.tmp/size_r5/compare` 的误差最大贴图与结局 CG；结局 CG 是否可再放宽到约 30 dB（`docs/research/wasm.md` §4.1） |
| V16 | t15 §8（`git show b4df88d:docs/handoff/codex_t15.md`）：浏览器实测 | `go run ./tools/wasmbuild -serve :8080`，Chrome／Firefox／Edge 各做：点击开始门、音频、输入与焦点、score／回放导入导出、IndexedDB 重载、全屏整数缩放、F3 下 60／120／144 Hz 的 TPS 与最大帧耗时（冷帧约 24 ms 峰值未定位）、资源包加载等待 |
| V17 | t16 §8（`git show b4df88d:docs/handoff/codex_t16.md`）：清理后的标题与菜单 | 标题→难度→三菜单的背景生命周期、回放确认淡出、结局／Staff 淡出与方向长按重复（26／34／42…帧） |

STD 背景与 pacing 的目视事项见 `docs/fidelity.md` 对应行。

## 等用户录回放或采 trace

采集步骤见 `docs/harness.md`“用户采集步骤”（t17 新探针，`git show b4df88d:docs/handoff/codex_t17.md` §8）。

| 编号 | 来源与项目 | 需要的资料 |
|---|---|---|
| R1 | t1：其余机体动态射击 | 已有 `th10_01`–`04`、`08` 与替代 C；仍缺动态射击槽位（D9）与真人录制的魔理沙 C |
| R2 | t5：复刻录制的原作互通 | 复刻生成的 `.rpy` 放入用户自己的原作槽位播放，提供结果或 trace |
| R4 | t1–t5、t17：全数值与自机状态差分 | 新探针 `-all-chapters -ext` 重采 th10_05 六章（score、faith、+0x458／+0x4310、道具）；旧 trace 的 score／faith／state 列无效 |
| R5 | M2 激光直接验收 | 同 R4 的 `-ext` 采集含激光行 |
| R6 | t2、t5、t8：Extra 与最终通关流程 | Extra 宣言、结束与六面通关加分／场景请求的 trace 或录像 |
| R7 | t12、t13：原作 trace 末帧截断 | 重采 `th10_02`、`th10_04`、替代 `th10_10`（截断于 6360／6945／5352） |
| R8 | t12：真人录制的魔理沙 C 回放与 trace | 替代回放（B 输入加 C 逻辑）不能代替 |
| R9 | t17：标题四 demo | `th10probe -demos -ext` 采 demo1／2／3／0，再与 `th10sim -demo N` 比较 |

## 推测待证与审阅注意

| 编号 | 来源与未决点 | 当前状态 |
|---|---|---|
| P7 | t5：现场 slow-rate 的 host 数据源 | 原全局 double 计数器无 host 来源，现场总慢放率为 0；逐 30 帧样本用逻辑 FPS，首窗口 60 是适配 |
| P9 | t5、t10：原暂停 flag | 暂停状态 1–5 已按 t10 移植，原 flag 未专门取证，遮罩在原 draw 链外 |
| P10 | t3：字体与弹出字驱动差异 | W3／W7 光栅化不等于 GDI／D3DX；出生帧 Inf／NaN 的像素结果可能依驱动而异 |
| P13 | t13：c5 第六面末段 | 49748 帧起的 MSG 计时修正使章末 50288→50408；原作参考只到约 45000 帧 |
| P14 | t6：`TestReimuAPowerLevelsAndBirths` 期望值 | 期望按原作算术重算（FUN_00427e90、FUN_00428b10、FUN_00428280），防回归但不是独立数值来源；审阅看公式与地址 |
| P16 | t11：汉化资源 | 结局、Staff、标题菜单文字用 th10c.dat 繁体覆盖（`.mst`、`.ant`），仅演出；原字节资源来自 th10.dat |
| P17 | t11：额外回放 catalog | `th10_ud????.rpy` 枚举后端未扩展，默认只显示 25 个原槽 |
| P20 | t15：持久化时机 | 沿用更早的离散宿主检查点，不是原 writer 逐指令磁盘时序（`docs/research/wasm.md` §2） |
| P12 | 全部轮次：lint | golangci-lint 不是验收项，不称全绿 |

## 已关闭或已定论

- D1–D5、D7，P1–P6：第二轮 t6–t8 实现并合并。
- R3：t8 用真实 `scoreth10.dat`（17655 B）完成类型视图往返与字节相同验证；双向运行时互通见 V10。
- V2、V3（t1 矩形门控、t2 宣言／cdbg／震屏）：由 t6、t7 完成，其目视并入 V8、V9。V5 并入 V4。
- P15：`TestOtherShotsRemainLegacy`／`TestReimuAPowerLevelsAndBirths` 的期望已按授权修正（`git show b4df88d:docs/handoff/codex_t6.md` §4），容差未改。
- P18：标题背景、ANM-91 与共用淡出已统一到 `scene/ui/presentation.go` 的 `TitleAnimations`。
- P19：th10diff 只保留 `-after`，`-from-frame` 已删除；新探针的 H 半帧自动 SKIP。
- 原作 trace th10_01–04、08 与替代 C 采集：用户已完成，合并树上分段续比复现。
- t8 第六面末帧分数（50288，9055940）：采用新值，`base_c5` 已更新。
- `musiccmtt.txt` 换行：以 `.gitattributes` 的 `-text` 保存原 CRLF 字节。
- 旧 `TH10_LEGACY_BG` 背景对比：旧背景已于第五轮删除，不再目视对比。

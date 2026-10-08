# 真值台账（Fidelity Ledger）

每个子系统的还原状态、依据与验收方式。**每次改动必须更新此表**；golden 快照只在有意的
语义变更后重生成，且必须在提交信息中说明原因——禁止用更新 golden 掩盖分歧。

状态枚举：

- **位精确** — 逐位/逐值与原作一致，有二进制依据
- **有界漂移** — 浮点路径，短窗口 epsilon 内一致且不发散
- **近似** — 行为形状对，数值未取证
- **未做** — 缺失或占位

依据栏：函数地址（th10.exe 反编译）、数据表地址、trace/回放编号、文档路径。

## 关卡逻辑

| 子系统 | 状态 | 依据 | 验收 |
|---|---|---|---|
| ECL 容器/指令解码 | 位精确 | `thecl10.c`、`FUN_0044fdb0/0044fe40` | `scene/stage/ecl/loader_test.go` |
| ECL 虚拟时钟 | 位精确 | Boss1/Boss1At1 Time 字段 | `TestVMClockTimingFirstWave` |
| ECL 栈机 opcode 0-93 | 位精确 | `FUN_0043ee30` | `vm_test.go` |
| ECL 敌机/移动 256-368 | 有界漂移 | `FUN_0040e770` | `TestVMAllStagesStableAndFire` |
| ECL 弹幕 400-436 | 有界漂移 | `FUN_004067d0`、`FUN_004073e0` | `emit_test.go`、golden |
| ZUN RNG | 位精确 | `FUN_0044bb90` 常量 PE 直读 | `vm.go` rotl16 实现 |
| 缓动 stepper | 近似 | `FUN_00404610`（round2 dump 已有，未实装） | 待 M1 |
| 出屏删除 | 位精确 | `FUN_0040dc80` | golden |
| 动态 rank | 位精确 | `DAT_00474c98` | golden |
| 弹种尺寸表 | 位精确 | `DAT_004741e0` PE 直读 | `_data_tables.txt` |
| ECL 全关卡稳定性 | 有界漂移 | stage01-06.ecl | `TestGoldenStages`（3600 帧 FNV） |

## 自机

| 子系统 | 状态 | 依据 | 验收 |
|---|---|---|---|
| SHT 移动参数 | 位精确 | `FUN_00426520`、`FUN_004250b0` | `sht/movement_test.go` |
| 移动边界 | 位精确 | `FUN_004250b0` | `player_test.go` |
| 灵梦 A 射击组/发弹 | 有界漂移 | `FUN_004281d0/00428160`、`pl00a.sht` | `reimu_a_test.go` |
| 灵梦 A 诱导 | 有界漂移 | `0x428ad0/00428b10/00428ce0` | `reimu_a_test.go` |
| 副机布局/插值 | 有界漂移 | `0x476f7c`、`FUN_00426f70` | `reimu_a_test.go` |
| 其余 5 机体射击 | 近似 | 旧实现，未接 SHT 组表 | 待 M1 |
| Power 资源规则 | 位精确 | `FUN_00425730`、`FUN_00418930` | `player_test.go`、`power_test.go` |
| 决死窗口 | 位精确 | `FUN_00425730` state 4 | `player_test.go` |
| 死亡 Power 损失/掉落 | 有界漂移 | `FUN_00425730` state 2 | `player_test.go`（延时/轨迹近似） |
| 灵击生命周期 | **近似** | `FUN_00405750/00405860/004055c0`（已 dump 未实装） | 待 M1（现为 300 帧占位） |
| 自机完整状态机顺序 | 近似 | `FUN_00425730` caller 链 | 待 M1 |

## 验证基础设施（M0，2026-10-04 建成）

| 组件 | 状态 | 依据 | 验收 |
|---|---|---|---|
| .rpy 回放解析 | 位精确 | `FUN_00428f60/0042a200/004297d0/0044b0d0/00435dc0` | `replay_test.go`（4 demo + trailer 交叉验证） |
| trace 格式 | — | `trace/` | `th10diff` 测试 |
| 无头运行器 | 位精确（确定性） | `sim/`、`stage.NewHeadless` | `TestHeadlessDeterministic`（逐字节一致） |
| 原作只读采集 | 待动态验证 | `tools/th10probe/` | 需运行原作（尚未跑） |
| 差分器 | — | `tools/th10diff/` | `main_test.go` |
| 差分闭环 | **未打通** | — | 原作侧 trace 尚未采集 |

## 表现层

| 子系统 | 状态 | 依据 | 验收 |
|---|---|---|---|
| ANM 容器/精灵表 | 位精确 | `anm/loader.go` | `loader_test.go` |
| ANM 时间线子集 | 有界漂移 | `anm/resolve.go`（ins_0/1/3/4/5） | `loader_test.go` |
| ANM 完整 VM（~75 op） | 未做 | — | 待 M2 |
| STD 背景 | 近似 | `stage01.std.txt` | 待 M2 |
| 子弹/敌人贴图 | 近似 | 几何 fallback + 部分 ANM | 待 M2 |
| 对话 .msg | 位精确 | `scene/dialog/msg.go` | `msg_test.go` |
| Boss HUD/符卡名 | 位精确 | ECL 357/359 + Shift-JIS | 手工核对 |

## 外围

| 子系统 | 状态 | 依据 | 验收 |
|---|---|---|---|
| Extra 关 | 近似 | 手写（`assets/ecl/stage07.ecl` 未接） | 待 M3 |
| 回放录制/播放 | 未做 | — | 待 M3 |
| 菜单/结局/Staff | 近似 | — | 待 M3 |
| 存档/配置 | 近似 | `game/persistence.go` | — |

## 更新记录

- 2026-10-04 — 建表；M0 差分脊柱建成（replay/trace/sim/th10diff/th10probe）。

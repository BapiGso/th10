# 差分验收闭环（Differential Harness）

M0 建立的"一比一还原"验证脊柱。它把还原从**手工取证**升级为**可量化验收**：
用官方回放驱动，同一输入喂原作与复刻，逐帧比对，分歧即 bug 报告。

## 数据流

```
                    .thtk/th10_dat/demoN.rpy  (官方回放，t10r 格式)
                              │
              ┌───────────────┴───────────────┐
              ▼                               ▼
   th10probe（只读采集）              th10sim（无头模拟）
   OpenProcess(VM_READ)              stage.NewHeadless + 逐帧 Restore
   ReadProcessMemory 逐帧采样          replay.Chapter.Records 逐帧喂入
              │                               │
              ▼                               ▼
        orig.trace                       sim.trace
              └───────────────┬───────────────┘
                              ▼
                        th10diff（比对）
                   首个分歧：帧号 + 实体 + 字段 + 两侧值
```

两侧产出同构的 trace（`trace` 包），所以 diff 不需要任何适配层。

## 组件

| 组件 | 路径 | 说明 |
|---|---|---|
| 回放解析 | `replay/` | t10r 容器 + 双 XOR 变换 + ZUN LZSS + 章节/输入记录/trailer |
| trace 格式 | `trace/` | 逐帧文本 schema，两侧共用；diff 按 epsilon 预算比较 |
| 无头模拟 | `sim/` + `scene/stage.NewHeadless` | 不依赖音频设备与 HUD，输入来自 `input.State.Restore` |
| 原作采集 | `tools/th10probe/` | **只读**（`PROCESS_VM_READ`），校验 exe SHA-256 与模块基址 |
| 差分器 | `tools/th10diff/` | 首分歧定位 + 前后文帧打印 |
| 模拟 CLI | `tools/th10sim/` | 回放 → trace，用于生成复刻侧基准 |

## 用法

```bash
# 复刻侧（无需原作）；验收回放见 docs/research/replays.md
go run ./tools/th10sim -replay ".tmp/[th10]/replay/th10_05.rpy" -frames 3600 -out .tmp/trace/sim.txt

# 原作侧（先启动原作并进入回放播放）
go run ./tools/th10probe -exe ".tmp/[th10]/th10.exe" -frames 3600 -out .tmp/trace/orig.txt

# 比对
go run ./tools/th10diff -a .tmp/trace/orig.txt -b .tmp/trace/sim.txt
# exit 0 = 全程在 epsilon 内；exit 1 = 有分歧（打印首个分歧帧）
```

原作侧加 `-shot-frames 1200,1300-1302 -shot-dir .tmp/shot` 可在指定帧截取游戏窗口客区，写成 `f<前>-<后>.png`（前后为截图前后读到的帧计数）。原作须为窗口模式、32 位色、窗口未被遮挡、显示缩放 100%。

### 原作配置 th10.cfg

`th10.cfg` 共 52 字节，由 `FUN_00438ad0` 读入 `DAT_00491d48` 并在退出时写回。与采集相关的字段：

| 偏移 | 含义 | 依据 |
|---|---|---|
| +0x1a | 色深：0 = 32 位（X8R8G8B8），1 = 16 位（R5G6B5），0xff = 自动 | `FUN_00439890` |
| +0x1d | 1 = 窗口模式（带边框 640×480），0 = 全屏 | `FUN_00439730`、`FUN_00439890` |
| +0x20 / +0x21 | BGM / SE 音量 | — |
| +0x30 位 0x100 | 启动时弹窗询问；答"是"（IDYES）为全屏，否则窗口模式，结果覆盖 +0x1d | `FUN_00438ad0` |

截图对比（L3）需要窗口模式。

## 精度标准（L1/L2/L3）

- **L1 位精确**：整数状态（分数/火力/残机/信仰/计时器/帧号）、RNG 序列、生成/删除决策、
  碰撞判定结果、输入采样。任何偏差 = bug。
- **L2 有界漂移**：浮点路径。原作是 x87 80 位中间精度 + float32 存储；复刻以 float32 存储、
  float64 中间运算近似 x87。验收：短窗口 epsilon ≤ 1e-3 px，且**不允许随帧数累积发散**——
  若发散说明是逻辑分歧而非精度问题，必须定位到具体帧。
- **L3 视觉一致**：M2 之后接入 ANM/STD 再按像素 diff 收敛。

## 已确认的回放格式事实

来源：th10.exe 反编译（`FUN_00428f60` 写入、`FUN_0042a200` 加载、`FUN_0042a8a0` 记录追加、
`FUN_004297d0` 逐帧消费、`FUN_0044b0d0` XOR 变换、`FUN_00435dc0` LZSS 解压）。
证据 dump 在 `.tmp/replay_truth/`。

```
文件头 36 B:
  +0x00 char[4] "t10r"        +0x04 u32 版本(=5)
  +0x0c u32 尾部偏移(文件大小-200)
  +0x1c u32 压缩长度          +0x20 u32 解压后长度
  +0x24 压缩数据
  尾部 200 B 明文元数据（USER 名 / Version / Date / Chara / Rank / Stage / Score / Slow Rate）

解密: FUN_0044b0d0(data, len, keyInc, modulus, limit)，初始密钥在 AL
  第一次 keyInc=0xe1 modulus=0x400 initKey=0xaa
  第二次 keyInc=0x7a modulus=0x80  initKey=0x3d
解压: FUN_00435dc0 = ZUN LZSS，8 KB 环形窗口
  标志位 MSB-first；1 位=字面字节，0 位=13 位窗口偏移 + 4 位长度（拷贝 长度+3 字节）
  偏移 0 终止

解压后负载:
  100 B 全局头（+0x50 角色 / +0x54 装备 / +0x58 难度 / +0x5c 关卡）
  章节块个数 = 全局头 +0x4c（大于 7 时按 6），FUN_0042a200 按此遍历:
    0x1c4 B 头 + N×6 B 输入记录 + ceil(N/30) 字节 slow-rate 采样
    章节头 +0x00 章节索引 / +0x02 RNG 种子 / +0x04 帧数（回放读取上限）
           +0x08 本块正文字节数（下一块 = 记录起点 + 该值）/ +0x14 信仰（存储值）/ +0x18 信仰保持计时（帧）/ +0x20 动态 rank
    记录 = {u16 keys, u16 aux1, u16 aux2}
  +0x04 可能大于实际存储的记录数（Royalflare ud0043 每章都是 2 倍），
  所以下一块的位置必须用 +0x08，记录数取两者较小值

输入位（keys，消费端用 0x01f7 掩码）:
  0x001 射击(Z)   0x002 灵击(X)   0x004 低速(Shift)   0x100 跳过(Ctrl)
  0x010 上   0x020 下   0x040 左   0x080 右（每个方向一位，与菜单 FUN_004236f0 相同）
  自机方向由 FUN_004250b0 按以下顺序判定，冲突组合取先命中者:
    上左 5 → 下左 7 → 上右 6 → 下右 8 → 下 2 → 上 1 → 左 3 → 右 4
  （2026-10-27 更正：此前写的"2-of-4 编码"是猜测，回放中大量单独的 0x40/0x80 与之矛盾）
```

4 个官方 demo 的 trailer 与全局头**交叉验证一致**（角色/装备/难度/关卡四项），见
`replay/replay_test.go`。

## 回归护栏

```bash
go test ./replay ./sim ./tools/th10diff ./trace   # M0 组件
go test ./scene/stage/ecl -run TestGoldenStages   # ECL 行为快照（已有）
go test ./... && go build -o .tmp/th10-debug-next.exe ./cmd
```

`TestHeadlessDeterministic` 锁定"同一输入 → 逐字节相同 trace"这一差分前提。

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
# 复刻侧（无需原作）
go run ./tools/th10sim -replay .thtk/th10_dat/demo0.rpy -frames 3600 -out .tmp/trace/sim.txt

# 原作侧（先启动原作并进入回放播放）
go run ./tools/th10probe -exe ".tmp/[th10]/th10.exe" -frames 3600 -out .tmp/trace/orig.txt

# 比对
go run ./tools/th10diff -a .tmp/trace/orig.txt -b .tmp/trace/sim.txt
# exit 0 = 全程在 epsilon 内；exit 1 = 有分歧（打印首个分歧帧）
```

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
  最多 6 个章节块: 0x1c4 B 头 + N×6 B 输入记录
    章节头 +0x00 章节索引 / +0x02 RNG 种子 / +0x04 记录数 / +0x18 起始帧 / +0x20 动态 rank
    记录 = {u16 keys, u16 aux1, u16 aux2}
  每章节后跟 ceil(N/30) 字节的 slow-rate 采样

输入位（keys，消费端用 0x01f7 掩码）:
  0x001 射击(Z)   0x002 灵击(X)   0x004 低速(Shift)   0x100 跳过(Ctrl)
  方向用 2-of-4 编码:
    0x010/0x020 = 水平/垂直轴激活；0x080/0x040 = 上或左 / 下或右
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

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

续比只用于已证实的采样半帧，不替代完整比较：

- `-from-frame N`：只比较原作参考帧号 ≥N 的帧（`-shift` 对齐在其后）。
- `-after N`：跳过原作参考帧号 ≤N 的帧。
- 两者同时给出时，跳过条件取并集。两者都不放宽 eps，也不删 M 行。

原作末行截断或短 B 行时返回读取错误（exit 2，输出 `INCOMPLETE`），不会当作 MATCH。

六章基准 `.tmp/trace/d1c2/base_cN.txt`：c5 已于 2026-11-08 更新为 t13 合并后的输出，旧版保存为 `base_c5.pre_t13.txt`。

新版探针不截图、不发窗口消息、不挂起线程、不修改原作内存。原 `-shot-frames／-shot-dir` 采集路径已禁用，传入非空值返回用法错误。L3 由用户自行目视或另行提供画面，不能用探针的数值 MATCH 冒充像素一致。

新版自动把未确认算完的样本写成 H 半帧。`th10diff` 跳过它们并报告 `SKIP`，不计入匹配帧；全部重叠样本都是半帧时返回 INCOMPLETE。旧文件没有 H，仍需对已取证的半帧使用原续比开关。

### 原作配置 th10.cfg

`th10.cfg` 共 52 字节，由 `FUN_00438ad0` 读入 `DAT_00491d48` 并在退出时写回。与采集相关的字段：

| 偏移 | 含义 | 依据 |
|---|---|---|
| +0x1a | 色深：0 = 32 位（X8R8G8B8），1 = 16 位（R5G6B5），0xff = 自动 | `FUN_00439890` |
| +0x1d | 1 = 窗口模式（带边框 640×480），0 = 全屏 | `FUN_00439730`、`FUN_00439890` |
| +0x20 / +0x21 | BGM / SE 音量 | — |
| +0x30 位 0x100 | 启动时弹窗询问；答"是"（IDYES）为全屏，否则窗口模式，结果覆盖 +0x1d | `FUN_00438ad0` |

这些配置供用户自行做 L3 目视参考；只读数值采集不要求切换窗口／全屏模式。

## 用户采集步骤

以下命令在工具合并后的项目主树、PowerShell 7 中执行。开发者本轮没有运行原作或探针；这些步骤由用户操作。不要把 trace 输出放到原作数据目录或三个只读 junction。每次使用新的时间戳目录，工具不覆盖已有采集。

### 1．准备工具和输出目录

```powershell
Set-Location -LiteralPath 'D:\GoProjects\th10'
$exe = 'D:\GoProjects\th10\.tmp\[th10]\th10.exe'
$replays = 'D:\GoProjects\th10\.tmp\[th10]\replay'
$bin = 'D:\GoProjects\th10\.tmp\capture\bin'
$run = 'D:\GoProjects\th10\.tmp\capture\t17_' + (Get-Date -Format 'yyyyMMdd_HHmmss')
New-Item -ItemType Directory -Force -Path $bin, $run | Out-Null
go build -o "$bin\th10probe.exe" ./tools/th10probe
if ($LASTEXITCODE -ne 0) { throw 'probe build failed' }
go build -o "$bin\th10sim.exe" ./tools/th10sim
if ($LASTEXITCODE -ne 0) { throw 'sim build failed' }
go build -o "$bin\th10diff.exe" ./tools/th10diff
if ($LASTEXITCODE -ne 0) { throw 'diff build failed' }
Write-Output $run
```

如果原作在别的目录，只改 `$exe／$replays`，不要改原回放内容。自行启动原作，停在回放选择画面；不要让本脚本替原作按键。先启动探针、看到 waiting，再在原作选相应回放开始播放。播放中不要按 Ctrl 快进；尽量避免暂停、窗口拖动、休眠和高负载任务。

### 2．优先：th10_05 六章扩展采集

原作选择槽位 05、从第一面开始，不能从第四面半路进入。探针一次连续采六章，跨关只换文件，不退出进程。

```powershell
$a05 = "$run\th10_05\orig"
$b05 = "$run\th10_05\sim"
New-Item -ItemType Directory -Force -Path $a05, $b05 | Out-Null
& "$bin\th10probe.exe" -exe $exe -replay "$replays\th10_05.rpy" -all-chapters -ext -frames 0 -timeout 2h -out-dir $a05 2>&1 | Tee-Object -FilePath "$run\th10_05\probe.log"
```

预计约 53～56 分钟：六章输入分别约 4分27秒／5分32秒／8分50秒／8分43秒／11分19秒／14分02秒，加载另加。输出为 `$a05\th10_05_c0.txt`…`th10_05_c5.txt`，第六面没有 45000／50288 的人为上限。扩展数值覆盖 R4，道具／激光覆盖 R5，末段为 P13 新参考；采到不等于已经比对通过。

采集完成后生成 sim，直接读取各文件 C 中的入场最高分，不需要用户手填 HUD 数字：

```powershell
& "$bin\th10sim.exe" -replay "$replays\th10_05.rpy" -all-chapters -ext -frames 0 -initial-dir $a05 -out-dir $b05 2>&1 | Tee-Object -FilePath "$run\th10_05\sim.log"
foreach ($n in 0..5) {
    & "$bin\th10diff.exe" -a "$a05\th10_05_c$n.txt" -b "$b05\th10_05_c$n.txt" -scope motion 2>&1 | Tee-Object -FilePath "$run\th10_05\motion_c$n.log"
    & "$bin\th10diff.exe" -a "$a05\th10_05_c$n.txt" -b "$b05\th10_05_c$n.txt" -scope ext 2>&1 | Tee-Object -FilePath "$run\th10_05\ext_c$n.log"
}
```

sim 不按实时 60 Hz 等待，通常几分钟内完成，具体取决于机器和磁盘。原作＋sim 六章扩展文件按数 GB 预留空间。若 ext 有首个分歧，保留原文件／日志即可；可再用 `-scope stats／items／lasers／spell` 缩小类别，不放宽 eps，不删 M 行。

### 3．补 R7：三份截断参考

依次执行下面三组，每组都先运行探针，再由用户在原作选择对应槽位。02 为灵梦 C／Lunatic／四面，04 为魔理沙 A／Lunatic／二面；10 是已有的 B 输入改 C 的替代回放，不能当作真人录制的魔理沙 C。

```powershell
$a02 = "$run\th10_02\orig"
New-Item -ItemType Directory -Force -Path $a02 | Out-Null
& "$bin\th10probe.exe" -exe $exe -replay "$replays\th10_02.rpy" -all-chapters -frames 0 -out-dir $a02 2>&1 | Tee-Object -FilePath "$run\th10_02\probe.log"
# 原作：播放槽位 02；约 1分46秒，含加载按 2～3 分钟预留。

$a04 = "$run\th10_04\orig"
New-Item -ItemType Directory -Force -Path $a04 | Out-Null
& "$bin\th10probe.exe" -exe $exe -replay "$replays\th10_04.rpy" -all-chapters -frames 0 -out-dir $a04 2>&1 | Tee-Object -FilePath "$run\th10_04\probe.log"
# 原作：播放槽位 04；约 1分56秒，含加载按 2～3 分钟预留。

$aC = "$run\th10_03_as_C\orig"
New-Item -ItemType Directory -Force -Path $aC | Out-Null
& "$bin\th10probe.exe" -exe $exe -replay "$replays\th10_10.rpy" -all-chapters -frames 0 -out-dir $aC 2>&1 | Tee-Object -FilePath "$run\th10_03_as_C\probe.log"
# 原作：播放槽位 10；现有替代在约 5352 帧结束，约 1分30秒，最多约 2 分钟。
```

新参考依次在 `th10_02\orig\th10_02_c0.txt`、`th10_04\orig\th10_04_c0.txt`、`th10_03_as_C\orig\th10_10_c0.txt`；不要覆盖旧的截断文件。

```powershell
foreach ($job in @(
    @{ Folder='th10_02'; Replay='th10_02'; Orig=$a02 },
    @{ Folder='th10_04'; Replay='th10_04'; Orig=$a04 },
    @{ Folder='th10_03_as_C'; Replay='th10_10'; Orig=$aC }
)) {
    $b = "$run\$($job.Folder)\sim"
    New-Item -ItemType Directory -Force -Path $b | Out-Null
    & "$bin\th10sim.exe" -replay "$replays\$($job.Replay).rpy" -all-chapters -frames 0 -initial-dir $job.Orig -out-dir $b 2>&1 | Tee-Object -FilePath "$run\$($job.Folder)\sim.log"
    & "$bin\th10diff.exe" -a "$($job.Orig)\$($job.Replay)_c0.txt" -b "$b\$($job.Replay)_c0.txt" -scope motion 2>&1 | Tee-Object -FilePath "$run\$($job.Folder)\motion.log"
}
```

正常新文件有 Z 帧提交与 Q 结束行；读取异常、超时、Ctrl+C 或未确认末帧会明确报告 INCOMPLETE，而不是静默输出损坏尾行。旧 trace 的已证实历史半帧不会被新工具自动补造标记；新采集使用 H，无须先猜 `-after／-from-frame`。

### 4．最后：标题 demo 自动采集

自行让原作停在初始标题或主菜单，不要继续按 Z／X／Esc／Enter／Ctrl。探针不操作窗口，等原作自身的 900 帧待机演示，按实际文件名识别正在播放的编号；原作的下一项计数器不是当前 demo 编号。

```powershell
$ad = "$run\demo\orig"
$bd = "$run\demo\sim"
New-Item -ItemType Directory -Force -Path $ad, $bd | Out-Null
& "$bin\th10probe.exe" -exe $exe -demos -ext -out-dir $ad -timeout 15m 2>&1 | Tee-Object -FilePath "$run\demo\probe.log"
```

通常 4～5 分钟采齐 demo1／2／3／0，分别为 `demoN_c0.txt`；若启动时已在某段中途，默认等下一段从头开始。重复编号不覆盖文件，采齐四个不同编号后自动退出。

```powershell
foreach ($n in 0..3) {
    & "$bin\th10sim.exe" -demo $n -ext -initial "$ad\demo${n}_c0.txt" -out "$bd\demo${n}_c0.txt" 2>&1 | Tee-Object -FilePath "$run\demo\sim$n.log"
    & "$bin\th10diff.exe" -a "$ad\demo${n}_c0.txt" -b "$bd\demo${n}_c0.txt" -scope motion 2>&1 | Tee-Object -FilePath "$run\demo\motion$n.log"
    & "$bin\th10diff.exe" -a "$ad\demo${n}_c0.txt" -b "$bd\demo${n}_c0.txt" -scope ext 2>&1 | Tee-Object -FilePath "$run\demo\ext$n.log"
}
```

`-demo N` 直接读嵌入的 `assets/misc/demoN.rpy`，默认模拟原标题演示的 3001 实帧窗口；`-frames 0` 可以回放资源的完整输入，但这不是标题实际播放的长度。

### 5．交回资料与异常处理

保留 `$run` 的 orig、sim 和全部日志，把目录路径告诉架构师。关注每章 `status=complete`、最后帧号、half／missing、平均／最大读取耗时，以及 diff 的 MATCH／SKIP／INCOMPLETE／DIVERGE。只有实际参与比较的稳定帧才计入 MATCH。分数实际字段与 HUD 滚动字段分开：第六面最后一帧可能刚加奖金，HUD 尚未滚动到总分，不要手工改 trace。

不要在采集时关闭原作进程；结束播放后探针会自行收尾。必须中断时，对探针终端按 Ctrl+C，完整已提交前缀会保留，但该段标为未完成，需要重新播放。若探针因哈希／基址／进程查询失败，保留错误，不用 `-skip-hash` 绕过。若末帧未确认，换新的输出目录重播；降低系统负载，不改容差。工具没有强制让原作等待读取的功能，也不会写入原作保存／配置文件。

## 精度标准（L1/L2/L3）

- **L1 位精确**：整数状态（分数/火力/残机/信仰/计时器/帧号）、RNG 序列、生成/删除决策、
  碰撞判定结果、输入采样。任何偏差 = bug。
- **L2 有界漂移**：浮点路径。`FUN_00439890` 不设 FPU_PRESERVE，原作游戏路径的 x87 精度控制为 24 位；
  复刻对普通算术按 float32 分步取整，三角函数／D3DX 仍有实现差异（见 `docs/research/m2_motion.md` 第 2 节）。
  验收：短窗口 epsilon ≤ 1e-3 px，且**不允许随帧数累积发散**——
  若发散说明是逻辑分歧而非精度问题，必须定位到具体帧。
- **L3 视觉一致**：M2 之后接入 ANM/STD 再按像素 diff 收敛。

## 已确认的回放格式事实

来源：th10.exe 反编译（`FUN_00428f60` 管理器创建／加载／章头初始化、`FUN_00429b60` 文件写出、`FUN_0042a200` 加载、`FUN_0042a8a0` 记录追加、
`FUN_004297d0` 逐帧消费、`FUN_0044b0d0` XOR 变换、`FUN_00435dc0` LZSS 解压）。
证据 dump 在 `.tmp/replay_truth/`。

2026-11-08，t9 按 t5 的编码器取证补正：USER／trailer 长度可变，不能固定为 200 字节；
th10_05 为 208 字节。9 个用户槽位＋4 个 demo 已在第一轮做全文件 Decode→Encode 逐字节互通，
实时自机不透明状态仍待 t6；本次文档整合未运行这些测试。

```
文件头 36 B:
  +0x00 char[4] "t10r"        +0x04 u32 版本(=5)
  +0x0c u32 USER 尾部偏移（按文件头定位，不固定为文件大小-200）
  +0x1c u32 压缩长度          +0x20 u32 解压后长度
  +0x24 压缩数据
  变长 USER 明文元数据（USER 名 / Version / Date / Chara / Rank / Stage / Score / Slow Rate）

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
  +0x04 可能大于实际存储的记录数（Royalflare ud0043 每章都是 2 倍；FUN_00429b60 重复保存累加），
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

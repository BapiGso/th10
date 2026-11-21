# t17：探针与差分工具

日期：2026-11-08。工作树：`D:/GoProjects/th10/.tmp/codex/t17/tree`。

## 1．摘要

A（普查）、B（半帧）、C（扩展字段）、D（整段／demo）、E（用户采集步骤）全部完成。最终规定验收全部通过，另跑完整绘制 smoke 与规则原样的 WASM 命令通过；DONE 时间在任务目录。

- 探针保持 VM_READ／查询权限；没有内存写入、注入、线程挂起、窗口消息、截图或原作输入。旧截图路径禁用。
- 完成见证＋前后控制块括读＋双逻辑快照；总预算 10 ms、最多四次。无法确认的样本标 H，不作正常验证。
- `-ext` 才输出 X／S／I／L；分数、最高分、信仰、火力、残机、自机状态／时钟／集中／定点位置、全部道具／敌激光、符卡／rank。整数精确，浮点预算不变。
- `-all-chapters` 按原活动章头路由，一次采整条回放；`-demos` 只观察原标题的 1／2／3／0，按实际文件名分别采四个不同编号。sim 支持嵌入 `-demo`、整段、入场最高分初值。
- 整帧单次写入，文件短写／写错回滚；每帧刷新，结束／错误／可处理终止信号收尾。C／Z／Q 检测完整帧与整段尾部，异常报 INCOMPLETE。
- 不改关内规则、RNG、时钟顺序、任何既有测试断言或 golden；默认六章输出保持基线字节。

没有运行原作、`tools/th10probe` 或 Ghidra。R4／R5／R7／P13 的原作验证仍要用户播放回放并采新文件；本任务只把采集／差分工具交齐，不宣称这些原作验证已经通过。

## 2．改动文件清单

### 新增

| 文件 | 用途 |
|---|---|
| `docs/research/probe_ext.md` | 中文普查：主循环／完成信号、字段地址与读写函数、截断证据、demo、格式与开销。 |
| `docs/handoff/codex_t17.md` | 分阶段交付与合并接口，本文件。 |
| `trace/half.go` | H 诊断标记的类型与严格解析。 |
| `trace/half_test.go` | H 往返、旧字节兼容与坏标记测试。 |
| `trace/extended.go` | 可选扩展字段、全部 I／L、S／Z 与完整性解析。 |
| `trace/extended_test.go` | 全字段往返、缺行、完整容量、标记半帧测试。 |
| `trace/capture.go` | 原子帧写入／回滚、C／Q 会话与 entry 初值。 |
| `trace/capture_test.go` | 单次大帧写、刷新、短写回滚、整尾／异常完整性测试。 |
| `tools/th10probe/native.go` | 跨平台只读内存接口、地址／控制块、基础逻辑快照、大缓冲复用。 |
| `tools/th10probe/sample.go` | 有完成见证的预算内重读，不稳定／跨帧／读错 H。 |
| `tools/th10probe/sample_test.go` | 合成字节内存的稳定、重读、GDI 静止、跨帧／读错与参数测试。 |
| `tools/th10probe/extended.go` | 原资源、自机、道具、两类激光与符卡解码。 |
| `tools/th10probe/extended_test.go` | 字段／符号宽度、链表异常、缓冲复用、不输出的敌弹仍检测、解码基准。 |
| `tools/th10probe/readonly_test.go` | AST 审计只读／查询 Windows API 和 OpenProcess 权限，不接触进程。 |
| `tools/th10probe/capture.go` | 回放章节／demo 路由、连续采集、独占新文件、结束与异常刷新。 |
| `tools/th10probe/capture_test.go` | 合成内存跨关、demo 编号、读错末帧收尾、参数／身份测试。 |
| `tools/th10probe/process_other.go` | 非 Windows 采进程明确不可用，解码／格式测试仍可构建。 |
| `tools/th10sim/options.go` | 可重复解析的 CLI：ext、demo、整段、初值／info。 |
| `tools/th10sim/extended.go` | 只观察移植变量，填新 trace，不复算数值。 |
| `tools/th10sim/extended_test.go` | 变量直读、无状态／RNG 修改、扩展不改普通帧。 |
| `tools/th10sim/capture_test.go` | 四个嵌入 demo、整段及 C 初值／参数冲突。 |
| `tools/th10diff/extended.go` | ext／stats／items／lasers／spell 精确比较和新字段诊断。 |
| `tools/th10diff/half_test.go` | 两侧 H 跳过／计数、shift／范围、零稳定样本不报 MATCH。 |
| `tools/th10diff/extended_test.go` | 每项整数、既有浮点预算、类别与旧格式兼容。 |
| `tools/th10diff/capture_test.go` | 缺 Q 的新采集不能被当作 MATCH。 |
| `entity/laser/inspect.go` | 按原链顺序只读遍历，包括待删除激光。 |
| `scene/stage/probe_view.go` | 只读原 GUI 滚动分数字段。 |

### 修改

| 文件 | 用途 |
|---|---|
| `tools/th10probe/main.go` | 原子整段入口、只读打开、参数与本进程信号收尾。 |
| `tools/th10probe/shot.go` | 删除不可达的 PNG 队列／写入；只保留已有帧列表解析，以保留旧测试。 |
| `tools/th10sim/main.go` | 复用 headless 的单章／整段／demo，C 初值注入与收尾。 |
| `tools/th10diff/main.go` | H 自动排除与计数，新 scope，错误／INCOMPLETE 报告，原续比保留。 |
| `trace/trace.go` | 接可选扩展／会话；score／faith 用 ParseInt64，旧字节不变；Flush 不因旧错误丢前缀。 |
| `sim/sim.go` | 工具可选的 ConfigureState／TraceFrame／Writer hook，nil 默认路径保留。 |
| `docs/harness.md` | 只读约束、H 说明与完整中文用户采集步骤／命令／时长。 |

### 重命名／删除

`tools/th10probe/shot_windows.go` → `tools/th10probe/process_windows.go`：原 GDI／窗口捕获整条实现被移除，改为 hash／模块／路径校验与只读进程后端。没有删除已有测试、原回放、基线或资源。所有本任务生成的大 trace 比较后已删除；草稿、脚本和日志只在 `.tmp/codex/t17/`。

## 3．移植的函数地址

本任务没有重写这些关内函数，只从它们的字段／顺序取证采样。

| 原作依据 | Go 接点 |
|---|---|
| `FUN_00439390`，0x439478／0x43951b；`FUN_00418190`，0x418779；0x417a22 注册 calc 0x0a | `reader.readControl`、`calcWitness`、`reader.sampleSettled`。 |
| `FUN_00409d90／0041be80／00418930／004188a0／00418a00／00417040` | `reader.sampleFrame／sampleExtended` 与 sim 变量观察。 |
| `FUN_004250b0／00425730／00426cf0／00428e10` | 原自机定点、状态、状态时钟、无敌时钟；不修改 Player。 |
| `FUN_0041bb00／0041afd0` | 原道具 2198 槽／Live／种类／位置／状态／时钟解码。 |
| `FUN_0041c510／0041c330／0041c5b0／0041c680／0041c8c0／0041e5c0` | 原激光链、256 上限、两个 vtable／模板位置；`Manager.Inspect` 只读。 |
| `FUN_00409280／00408d60／00409c00` | 原符卡保留 id／timer／bonus／initial／duration／flags。 |
| `FUN_0042cdf0`，0x42ce60／0x42ce9a；`FUN_00418190`，0x4186c4 | `captureInfo／demoNumber`：读真实文件名，不拿已经推进的 demo 游标作当前编号。 |
| `FUN_004297d0／0042a450／004218d0`；0x416ede／0x416ee8、0x425c93 | 活动原章节／场景请求，先提交旧章最后一帧再路由下一章／退出。 |

完整绝对地址、偏移与类型逐项见 `docs/research/probe_ext.md`。

## 4．测试

A／B／C／D 均跑完整 build、vet、`go test -count=1 ./...`、gofmt 输出为空、WASM（tools 除外）及六章差分，全部规定项通过。日志分别在主树 `.tmp/codex/t17/logs/a_verified/`、`b/`、`c/`、`d/`。E 最终两轮复验也通过，日志 `logs/final/`、`logs/final_verified/`；最后一轮 exit 0。完整 `TH10_SMOKE=full` 通过（128.128 秒），原样 GOOS=js 的 go list／tools 剔除／WASM build 通过，日志 `logs/final_extra/`。首轮外层 PowerShell 误把非验收 lint 的 LASTEXITCODE＝1 当成失败，因此未执行附加 smoke；验收 summary 本身为 PASS，脚本已改为明确返回规定项状态，附加检查随后单独执行通过。

新增测试覆盖：合成内存三种半帧结果；GDI 长停顿即使静止也不能成为正常样本；全部原扩展字段、i16 火力／激光颜色；读错、跨帧、激光坏链；完整 2198 道具／256 激光；超过打印 256 敌弹仍检查；大帧单次写、短写回滚、每帧刷新、异常前缀／完整末帧；缺 Z／缺 Q／缺整尾；两侧 H 对齐跳过、计数及 scope；旧 trace、原续比与原容差；四 demo、跨关、观测高分初值；API／权限静态审计。

`logs/d_cli/summary.txt`：四个嵌入 demo 各 3001 帧完整 ext 自比较通过；显式整段六章各三帧的路由／会话自比较通过。原作接入没有动态测试，这是硬禁止事项，不伪造通过。

合成内存基准：i7-12650H、Windows amd64、Go 环境，单次扩展解码约 91.5 微秒、5098 B／16 次分配；仅热缓存字节内存，不测原进程。

阶段 A 首次脚本在切 GOOS 后枚举全包，含原 Windows-only 工具导致 go list 失败；改在切 GOOS 前枚举并排除 tools，真实 WASM 构建通过，失败日志保留 `logs/a/`。B 后后端已有 build tag／非 Windows stub。C 新测试首轮只把已有“player state”诊断标签误写为“state”；D 合成回放首轮忘更新 Global.Chapters＝2，编码器按真实字段只写一章，均修正本任务新测试，不改既有断言。

额外 lint：已实际运行 `golangci-lint run`，缓存改用任务目录避免被拒写 LocalAppData。工具版本 2.15.0（Go 1.27.0 构建）。只读 base 同命令复现 32 条；本工作树清掉本任务／已移除截图路径问题后剩 21 条，均在其他范围（16 errcheck／2 staticcheck／3 unused），本任务目录没有新报告。它依 rules 6.5 不是验收门槛，不能称“所有 lint 全绿”。最终默认仍为 21 条；取消数量限制后为 34 条（29 errcheck／2 staticcheck／3 unused），无本任务新增报告，见 `logs/final_extra/lint_full_verified.log`。本任务所有新代码／新测试的实际 lint 报告已处理；保留未修改的 sim 调试输出／旧工具测试等原问题。

## 5．Trace 结果

| 章 | 默认输出 cmp（A／B／C／D／E 两轮） | 原作 motion 匹配帧 |
|---|---|---:|
| c0 | 六轮均 0，逐字节相同 | 15853 |
| c1 | 六轮均 0，逐字节相同 | 17733 |
| c2 | 六轮均 0，逐字节相同 | 29763 |
| c3 | 六轮均 0，逐字节相同 | 29469 |
| c4 | 六轮均 0，逐字节相同 | 39343 |
| c5 | 六轮均 0，逐字节相同 | 40099 |

c0 使用 `orig_d.txt`，其余 `orig_05_s2.txt`…`orig_05_s6.txt`；所有上述比较 exit 0，无覆盖下降。c5 原参考结束后的 5408 帧仍待用户新采集，不把 sim 自比较算成原作验证。

`-ext -hiscore 10000000` 在 C、D 各跑 c0／c5 全段：15972／50408 帧；X／I／L／S／Z 严格解析与 ext 自比较通过。c0 末分 1090、HUD 1090、最多 106 道具／14 激光；c5 末分 9055940、最多 247 道具／12 激光。c5 刚在最后一帧加最终奖金，GUI 保留滚动值 1887190；这与移植的 32 帧滚动一致，不是解码错误，也不冒称已目视原 HUD。分数／信仰／火力／残机／rank 全程在原取值范围内。大 trace 已删除，摘要 `logs/c_ext/summary.txt`、`logs/d_ext/summary.txt`。

## 6．待决与推测

- 没有 dump 请求，没有未知地址补丁；静态依据为完整 asm、已有 C／ASM 和当前逐字段 Go 移植。
- 外部只读采样不能锁住原线程，也不能补读已经释放的末帧。完成见证＋括读＋双快照仍有调度竞态；无法确认就 H／INCOMPLETE，绝不挂起／注入来“保证”。
- 三份旧尾部都恰在 4096 字节边界，旧 bufio 未刷新的解释有静态依据；具体是 Ctrl+C 还是读取异常没有现场日志。
- 开销模型估算新增双读约 0.9～2.5 ms，gap 比旧版少 1.5 ms且大缓冲复用；真实 half／missing／平均／最大耗时等待用户日志。热缓存基准不是原作性能测量。
- C 的最高分是采集入场观测初值，不是回放自带字段；用于设置 Go 状态。没有 C 的旧格式需显式 `-hiscore`，不复算未知存档。
- 原作 R4／R5／R7／P13 与四 demo 尚未新采，因而不能关闭这些验证项。强杀／断电／底层存储故障不在 defer 持久性承诺内。

## 7．建议的 `docs/fidelity.md` 改动

本任务不直接改总账。建议替换／新增完整状态行：

```text
| trace 格式 | v2 默认字节保留；可选扩展 v1 与完整采集会话 | t17：H 半帧、X/S/I/L 扩展、C/Z/Q 完整提交；2198 道具/256 激光不截样；缺尾与异常不报 MATCH | 新增 trace/工具合成测试；th10_05 六章默认 cmp 不变 |
| 原作只读采集 | 工具静态/合成验收通过，等待用户新 trace | FUN_00439390 完成见证、FUN_00418190 前段帧时钟；预算括读/双快照/H，原子帧提交与异常收尾；只 VM_READ/查询，无截图/输入/挂起；跨章与标题 demo | docs/research/probe_ext.md、docs/harness.md 用户步骤；真实 half/missing/耗时待采 |
| 差分器 | 旧规则不变；扩展字段/半帧/完整性已接 | -scope ext/stats/items/lasers/spell，整数精确与原浮点预算；H 不计匹配，零稳定样本与不完整 C/Z/Q 返回 INCOMPLETE；-after/-from-frame 保留 | 原工具测试与新增逐项/兼容测试；六章 motion 15853/17733/29763/29469/39343/40099 不下降 |
```

建议更新记录：

```text
- 2026-11-08 — t17：先完成 probe_ext 中文原作地址/主循环普查，确认关卡帧号在 calc 0x0a 提前递增，不能把帧号不变或 GDI 静止双快照当完成。增加只读完成见证、10 ms/4 次预算括读与 H，diff 自动跳过并报告，不计匹配；可选 -ext 直接读 Go/native 字段，补高分/信仰/自机时钟/全部道具/两类敌激光/符卡/rank，整数精确与原浮点预算不变。按活动原章头连续采整条回放，最后帧整块单次写、失败回滚和 C/Z/Q 检查；移除探针窗口截图，原作文件独占新建不覆盖。只观察标题 demo 真文件名，分别采四编号；sim 复用嵌入 demo/整段/C 高分初值。六章默认输出与基线逐字节一致、motion 覆盖不变，规定 Go/WASM/gofmt 验收通过；原作新 ext/末帧/demo/读取开销仍等用户播放既有回放采集，不能把合成或 sim 自比较当成原作验证。
```

## 8．需要用户目视确认的项目

完整命令、目录与时长已在 `docs/harness.md`“用户采集步骤”。优先 th10_05 六章 ext（约 53～56 分钟），再 02／04／10 替代 C（各约两分钟），最后标题四 demo（约 4～5 分钟）。不要求用户自己打游戏。

请交回新的 orig／sim／日志。看探针 status／最后帧号／half／missing／平均与最大采样时间、diff 的 MATCH／SKIP／INCOMPLETE／DIVERGE。分数与 HUD 滚动值分开观察；实际激光与道具逐项数据、六面末段及末帧需要新的原作对比。发生未完成末帧必须重播，不能改容差、删行或编造缺帧。

## 9．与其他任务的接口

### 最小关内只读改动

- `entity/laser/inspect.go`：`func (m *Manager) Inspect(fn func(Laser))`，读 `FUN_0041c330` 的原顺序，不排序、不 prep、不删、不 tick，不过滤状态 1。仅 tools/th10sim 的观察者调用，调用方不得修改对象。
- `scene/stage/probe_view.go`：`func (g *GUI) ShownScore() int32`，只读原 GUI＋0x9e78（FUN_00417040），返回存储单位。工具输出乘 10；不触及 Numbers／Calc／Draw。
- `sim/sim.go`：新增 `Runner.ConfigureState func(*game.GameState)`（构造前）、`Runner.TraceFrame func(*stage.Stage,*trace.Frame)`（既有 sample 后）、`Runner.Writer *trace.Writer`（可选原子输出）。默认 nil 仍走原 newHeadless／既有采样，六章 cmp 证实。没有另算分数／信仰／rank／奖励／时钟，也没有改 Player／Item／Laser／ECL 更新。

### th10sim 入口，供 t15／t16 合并

`main → run → parseSimOptions → simulate → replaySource → sim.ConfigFor → simulateChapter → sim.New → Runner.Run`。`scriptFactory(int) stage.Script` 仍是单独函数，显式调用 stage1–6／Extra 的 NewScript；t16 若收缩配置只合并该工厂，不覆盖整文件。demo 走 gamedata 嵌入字节、正常回放走工具文件 I/O；不在游戏运行时读磁盘。t15 的 node/JS 入口应保留 ConfigureState／TraceFrame／Writer 三 hook与默认分支。没有新增依赖。

### 共享 trace／工具 API

`trace.NewWriter(io.Writer)` 仍默认 v2；`trace.NewAtomicWriter(io.Writer)` 整帧提交；`BeginCapture(CaptureHeader) error`、`EndCapture(string) error`、`Err() error`、`Frames() int`、`LastFrame() int`；reader 的 `Capture() *CaptureHeader` 供工具入场条件。H 结构化诊断不要求旧格式改版。新 scope 不修改既有 eps／ang-eps／打印舍入余量；原两项续比开关测试与行为都保留。

未调用／实现其他范围的原作业务函数；上述都是既有移植变量或原字段的读取。主树仅复制了本交付文档，没有改主树源码、总账、基线、资源、其他任务或 .git。

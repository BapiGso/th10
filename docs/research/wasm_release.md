# WASM 运行与发布普查（t15，阶段 A）

日期：2026-11-08。普查对象是第三轮合并后的 t15 独立工作树；没有运行浏览器、原作或探针。本文先记录代码与依赖源码中的事实，后续实测结果另列，不能把源码推断写成浏览器实测。

## 1．现有入口与资源边界

- `cmd/main.go` 创建 `game.New()`，设置 `loading.New()`，最后调用 `ebiten.RunGame`。目前没有下载页、用户手势门、本地 HTTP 服务或浏览器调试面板。
- `game/game.go` 的逻辑画面为 640×480，场地为（32，16）至（416，464）。`Layout` 恒定返回 640×480。全部资源经 `gamedata` 嵌入，运行时不读取资源文件；打包工具只构建 `./cmd`，不得根据 ANM、BGM 或字体的具体存储格式打包。
- `game.New` 当前调用 `SetTPS(SyncWithFPS)`，`Game.Update` 再经 `pacing.Pacer.Ready` 选择逻辑帧。60／120／180／240 Hz 附近锁整数刷新，144 Hz 走 1／60 秒网格；跳过的 Draw 用 `keepAlive` 保留上一帧。不能据 `ActualTPS()` 断言逻辑每秒 60 次，因为它会计入被 Pacer 跳过的 Update 回调。
- `scene/loading/loading.go` 的 sig 门由 **300 次真实场景 Draw** 控制，不是等待墙钟或 300 次空 Update。浏览器启动门必须挡住 `Game.Update` 与 `Game.Draw`，不能只静音或延后标题 BGM。
- `game.New` 用 `time.Now().UnixMilli()` 低 16 位播应用级备用 RNG，原依据为 `FUN_004201b0` 的 `0x4201d0…0x4201e1`。等待用户手势不得逐帧重播 RNG。

## 2．持久化接口与原作写入时机

### 2.1．原作实际文件与调用点

依据：现成 `t5_save/func_438ad0.c`、`func_417c80.c`、`func_41fb50.c`，完整 `t4_native/exe.asm`，t5／t8／t10／t11 的交付第 6、9 节。这里的 `score.dat` 是项目后端名，**原作文件名是 `scoreth10.dat`**。

| 文件／数据 | 原作入口与磁盘写入时机 | 当前项目调用 |
|---|---|---|
| score | `FUN_0042b030` 读、`FUN_0042b1e0` 写。原 score writer 的两个直接调用是关卡释放 `FUN_00417c80` 中 `0x417c8d`，与全局加载对象释放 `FUN_0041fb50` 中 `0x41fc6e`。排行、练习分数、符卡与音乐解锁先改内存，不是每次尝试／每帧写磁盘。 | `Game.SaveProgress` 调 `SaveData.nativeBytes → EncodeScoreFile → ScoreStore.Write`。t8／t11 还保留新局、Game Over、Clear、首次音乐解锁、结局／Staff 标记、排行与保存菜单的离散宿主检查点。它们比原 writer 的直接调用更早，不能称为原作逐指令磁盘时序。 |
| cfg | `FUN_00420870` 在 `FUN_00438ad0` 的 `0x438b66` 读取并验证 52 字节 cfg；`FUN_00438ad0` 正常退出且 `local_20 != 2` 时调用 `FUN_0044b540("th10.cfg", &DAT_00491d48)` 写回。 | `Game.SaveConfig` 是后端边界，t10 在 Option 的 Quit 完成后调用；这是浏览器不能依赖关闭页回调的离散宿主适配，桌面既有行为不在本任务重排。 |
| 回放 | `FUN_0042a8a0` 逐帧记录只改内存；保存页 `FUN_004236f0` 的 `0x423f16` 调 `FUN_00429b60` 编码／写出选中槽。暂停本身没有额外保存页。 | `Game.SaveReplay(slot, r)` 先 `ValidatePlayback`、`replay.Encode`，再 `ReplayStore.Write`；正常 Game Over／Clear 菜单状态 6–13 选择槽后调用。读取／列举走 `LoadReplay`／`ListReplays`。 |

本任务保持 t5／t8 的接口与既有离散检查点，不在 scene 中搬动原统计／保存流程，也不增加逐帧写入。浏览器正常游戏内 Quit 另外提交 cfg／score，对应原退出；刷新、崩溃或系统强杀不能保证退出回调，不能等到 `beforeunload` 才保存。原 writer 与既有宿主检查点的差异会在交付第 9 节明确留给架构师，而不是暗改关内逻辑。

### 2.2．现有后端与格式

- `ScoreStore`：`Read() ([]byte, error)`、`Write([]byte) error`、`LegacyJSON() ([]byte, error)`。`DecodeScoreFile`／`EncodeScoreFile` 是 t8 的原格式 codec；TH10 v3、CR／ST unsigned checksum、原 XOR 与树形 LZSS，保留原字节。真实 17,655 字节样本已由 t8 验证。
- `ReplayStore`：`Read(slot int) ([]byte, error)`、`Write(slot int, data []byte) error`。25 槽对应 `th10_01.rpy`…`th10_25.rpy`。`replay.Decode` 与 `ValidatePlayback` 是导入边界；不合法文件不能先写后验证。
- cfg：`DecodeNativeConfig` 验证尺寸、版本与原 graphics 范围；`NativeConfig.Raw [0x34]byte` 保留未知字段，`Config.NativeBytes` 更新原百分比音量。
- 桌面 `persistence.go`／`persistence_replays_desktop.go` 用现有用户数据目录及临时文件重命名，资源读取与用户数据读写是两条边界。
- js／wasm 当前 `LoadConfig` 总是默认，`SaveConfig` 空操作；`browserScores`／`browserReplays` 是全局 MemoryStore，只有本页会话，没有刷新后保存。
- 回放播放不写共享统计、符卡历史、音乐解锁；本任务必须保留这些门控，不能因为替换后端而额外提交。

### 2.3．后端选择：IndexedDB

选 IndexedDB，不选 localStorage。localStorage 是同步字符串 API，通常只有数 MiB 配额；原二进制需 Base64，体积另增约三分之一，25 个回放槽可能超限。IndexedDB 支持 ArrayBuffer 与更大的、仍由浏览器按 origin 管理的配额，不需要新依赖。

实现约定：数据库 `th10`、版本 1、对象仓 `files`；键为 `score.dat`、`th10.cfg`、`replay/th10_NN.rpy`。Go 的同步接口内部用 `syscall/js` 注册事件，回调立即返回；调用方 goroutine 等待 **transaction.oncomplete**，不是 put request 成功就假报保存。只在原有离散保存检查点建事务，不每帧轮询或写入，不把失败降级为“已持久化”的会话内存。配额不足、禁用存储、事务 abort 要返回错误并送往控制台／页面状态。

IndexedDB 是异步宿主，浏览器关闭瞬间不保证未完成事务。页面自查要等“保存已提交”的提示后再刷新。不同浏览器、origin、端口与隐私模式有独立存储；清除网站数据会删除存档，导出仍是必要备份。

## 3．Ebiten v2.10.5 在 js 下的行为

依赖固定于 `go.mod`：Ebiten **v2.10.5**、Oto **v3.5.1**、oggvorbis **v1.0.5**、vorbis **v1.0.2**。核查的是本机 module cache 的实际源码，不引用别版本的行为。

### 3.1．音频上下文与自动播放

- `ebiten/audio/audio.go` 的 `NewContext` 仅建立 Go 对象，音频设备由 `AppendHookOnBeforeUpdateWithVMGuestInfo` 在 UI 初始化后的第一个 Update 之前创建。不能在调用 `game.New` 后立即假定 Web Audio 已存在。
- `oto/driver_js.go:newContext` 创建 `AudioContext`（后备 `webkitAudioContext`），注册 document 的 `touchend`、`keyup`、`mouseup`，在用户事件中调用 `resume()`；成功后关闭 ready channel。`audio.Context.IsReady()` 可以作为宿主门的第二个条件。
- Oto 优先用 AudioWorklet（另一个音频线程／worklet），不可用时退回 ScriptProcessor。Vorbis 解码、Go mixer 与向 worklet 送 PCM 仍经过 WASM 主线程；“有 worklet”不等于 Vorbis 在独立线程解码。
- 可行的第一帧门：先运行一个不调用真实场景 Update／Draw 的 Ebiten 包装器，让第一个音频 hook 安装；再允许 iframe 内真实点击／松键；同时满足“用户请求开始”与 `IsReady()` 后才启动真实第一帧，并重置宿主计时窗口。不能伪造 DOM 事件作为音频解锁。

### 3.2．失焦、隐藏与刷新率

- `internal/ui/ui_js.go:init` 默认 `runnableOnUnfocused: true`。**默认失焦并不会主动停止 Update**，隐藏标签页的 requestAnimationFrame 又通常被浏览器暂停／节流，两者不可混同。
- `SetRunnableOnUnfocused(false)` 后，`suspended()` 检查 `document.hasFocus()` 与 `document.hidden`；update 例程停下。独立的 100 ms ticker 检查焦点并调用音频 Suspend／Resume。这里是文档焦点，不是严格“canvas 是 activeElement”。
- 默认 js loop 用 requestAnimationFrame；显示刷新影响 Draw 回调数，不决定固定 TPS 的逻辑时钟。`internal/clock/clock.go` 的 `SetTPS(60)` 使用累积时间、四舍五入 tick 数与最多五 tick／至少 1／12 秒的 catch-up 上限；120／144 Hz 下通常多次 Draw 间运行一次 Update。
- `SyncWithFPS` 每个刷新回调必返一个 Update，项目当前又有自己的 Pacer。WASM 应显式 `SetTPS(60)` 并绕开桌面 Pacer 的二次限速；否则 60 次回调再筛一遍会少算。桌面 Pacer 保持不变。恢复焦点后不补完后台全部墙钟帧。
- 固定 TPS 是源码可验证的目标；真实浏览器是否能维持 60，要用逻辑帧计数叠加层在 60／120／144 Hz 实机验证，不能将 Node 的跑满速度当作渲染 FPS。

### 3.3．键盘、焦点、画布

- `ui_js.go:setCanvasEventHandlers` 对 canvas 的 keydown／keyup 调 `preventDefault`；keydown、mousedown、touchstart 会显式 focus canvas，blur 释放键。输入映射仍由 `input` 与 Ebiten gamepad API 负责。
- Ebiten 创建并向 `document.body` 追加 canvas，窗口尺寸来自 body；不能假定它会使用打包页自行创建的 `<canvas>`。网页采用同源 iframe 隔离游戏文档，使外围文件选择框／工具栏不被游戏的默认事件拦截。
- iframe 的 CSS 尺寸按 640×480 整数倍设置，canvas 使用最近邻；`FinalScreenDrawer` 保留 Ebiten 的比例／居中矩阵并强制 `FilterNearest`，不调用已弃用的 SetScreenFilterEnabled。小于 640×480 时保留 1 倍并允许外页滚动，不谎称分数缩小也是整数倍。
- 显式处理 Space、方向、F／F3 与游戏键的默认行为；F 切全屏，F3 切调试层。游戏焦点内拦可取消快捷键，不能保证拦截浏览器／系统保留的 Alt+Tab、Ctrl+W、F11、Esc 退出全屏等特权行为。

## 4．Vorbis 流式解码风险

`audio.NewBGMStream` 用 `vorbis.DecodeWithSampleRate(bytes.NewReader(embedded))`，再用 `audio.NewInfiniteLoopWithIntro` 保留原循环点。`vorbis.Stream.Read` 按需解码，Seek 明确是重操作。16-bit 解码路径没有整曲预解码，不能为了 WASM 把每首全展开成 PCM 加入常驻堆。

Oto `internal/mux/mux.go:defaultBufferSize` 的 source buffer 默认为 0.5 秒；这与 js driver 默认 2048 stereo frame 的 worklet／ScriptProcessor buffer 不是同一层。需要测本机 Node WASM 的建流、对应音频 buffer 的顺序 Read 与回环 Seek 耗时，再决定是否减少一次解码 burst／设置播放器 buffer；不能凭“可能卡”无条件增大缓冲。音频实际输出、爆音、浏览器 worklet 调度只能由用户试听。本任务不改 t14 的编码／资源路径，也不改变淡出帧数或原 PCM 循环点。

## 5．本机 Go 与 Node 执行器

实测命令：`go version`、`go env GOROOT GOPATH GOMODCACHE`、`node --version`。

| 项目 | 本机结果 |
|---|---|
| Go | `go1.27.3 windows/amd64`；仓库 go 指令为 `1.26.4` |
| GOROOT | `D:\ScoopApps\apps\go\current` |
| 浏览器胶水 | `D:\ScoopApps\apps\go\current\lib\wasm\wasm_exec.js`（16,992 B） |
| Node 胶水 | `D:\ScoopApps\apps\go\current\lib\wasm\wasm_exec_node.js`（1,186 B） |
| Unix 执行包装 | `D:\ScoopApps\apps\go\current\lib\wasm\go_js_wasm_exec`（603 B，Bash 脚本，本机不执行） |
| 示例页 | `D:\ScoopApps\apps\go\current\misc\wasm\wasm_exec.html`；旧位置没有运行时 JS |
| Node | `v26.11.1`，`D:\ScoopApps\apps\nodejs\current\node.exe` |

PowerShell 直接用 `node --stack-size=8192 <wasm_exec_node.js> <binary.wasm> <args>`，与 Go 包装器的 V8 stack 设置一致；每个 shell 按 rules 设置工作区 GOCACHE／GOTMPDIR／TMP／TEMP，Node 另设置 `TMPDIR`，不使用系统临时目录。发布工具必须复制 **本次构建 Go 的 GOROOT** 下的 `wasm_exec.js`，不能把胶水与 wasm 的 Go 版本混用。

## 6．阶段 A 的 Node 试跑事实

已构建原样 `GOOS=js GOARCH=wasm go build ./tools/th10sim`，用上述本机执行器运行 th10_05 第 0 章一帧。退出码 2，最先失败于：

```text
panic: syscall/js: call of Value.Get on undefined
github.com/hajimehoshi/ebiten/v2/internal/colormode.init()
.../internal/colormode/colormode_js.go:20
```

尚未进入 `sim.Runner`，不是浮点或 RNG 分歧。后续还可见 `ui_js.go` 的 document getter 绑定没有 Node guard，尽管同文件部分函数留有 Node 分支。阶段 E 将寻找**仅开发期 Node 执行器提供的最小浏览器初始化环境**，不初始化 WebGL、不调用 RunGame、不更换逻辑实现；若足够便无需改 th10sim 的图形构建路径。真实六章字节比较、逻辑性能、加载堆与非图形测试结果在完成后补录。

## 7．阶段边界

本文是先行普查，不表示 B–E 已实现或浏览器已运行。阶段 A 完整验收日志将保存在任务目录 `logs/A`；后续各阶段按同一门槛复跑并写入 `docs/handoff/codex_t15.md`。用户端浏览器检查项统一列在交付第 8 节。

补录：阶段 A2 完整 build／vet／test／WASM／gofmt／full smoke 与六章 cmp、motion 全部通过；原作一致帧数没有下降。首次脚本在 js 环境枚举 tools 时遇到 Windows-only probe，包枚举改为 native 先过滤后完整复跑，首轮日志保留。阶段 B 一键发布已产出 174,428,459 B wasm／103,218,816 B gzip，并完成真实本地 HTTP 原始头检查，结果见交付与 `logs/B`；仍没有浏览器实测。

阶段 C 完整验收与六章 cmp／motion 再次通过，真实原文件导入未改字节；实际 js IndexedDB adapter 在 Node 的显式事务假后端下验证了提交等待与 abort 保留旧数据。阶段 D 用 init-only DOM facade 运行非绘图的 js 时钟测试与 Vorbis benchmark：500 ms pull 最大 9.63 ms，200 ms 最大 3.43 ms；标题曲冷建流 40.46 ms，跨 loop Read 最大 12.81 ms。浏览器选择 200 ms source buffer，并在首个真实游戏帧前预热少量标题曲；没有提前 Play 或音乐解锁、没有整曲 PCM 常驻，也没有改变原循环点／淡出。完整数据与命令在交付及 `logs/D/audio.json`，不能据此声称浏览器音频或 GPU FPS 已验证。

最终补录：D／E 必需 build／vet／test／WASM／gofmt／full smoke 与六章基线、motion 验收全部通过。原样 th10sim 的 Node 初始化还需满足 legacy ebitenui 的 RAF 注册，开发期 facade 只返回注册值、不调度回调，仍没有图形／音频设备。六章原生＝base＝Node 逐字节相同，两 RNG 最终状态／计数相同；十个不调用图形运行器的包在 Node 测试通过。新增 sim.Measure 只计时既有 h.step，不修改 Run、关内逻辑、工具或 trace。

冷逻辑均值约 0.07–0.19 ms，但出现 24.708 ms／c3 8601、复测 24.070 ms／c2 3793 的单帧峰值；来源于冷路径与／或宿主调度是推测，未完全定位。显式同进程一次预热后六章最大值 3.364–10.173 ms，有稳态逻辑余量，不能据此保证全部冷帧或整个浏览器 GPU／音频帧都在 16.7 ms 内。各关加载后 Go HeapAlloc 16.2–19.4 MB，linear memory 190.8–194.5 MB，详细 JSON 与用户 F3 检查项见最终交付。最终发布为 wasm 174,610,642 B、gzip 103,256,693 B，仍没有使用 t14 瘦身结果或浏览器实测。

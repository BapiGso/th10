# t11：关外场景 B

## 1．摘要

工作树：`D:/GoProjects/th10/.tmp/codex/t11/tree`。2026-11-08 完成 A → B → C → D，四轮 rules 第 6 节验收全部通过。阶段 A 普查先单独成文，再按独立结局 VM、原 Staff／Music／Player Data／回放状态机移植。没有运行原作、probe 或 Ghidra，没有改关内逻辑、既有测试断言、golden、回放 codec、stage playback 或 t10 场景切换。

结局为十二机体／条件分支；Good 经原 Staff 后才进入 t5 的后续菜单，Bad 直接进入；Extra／练习不虚构结局。Music Room 使用原十八曲及八行说明、真实 ST 解锁字节与未知曲目二次确认；Player Data 使用所选机体／难度的原统计、排行与符卡历史；回放选单使用原列表／详情／选章时序和 t5 存储／播放接口。字体光栅化、持久标题背景、共用输入／淡出与额外 catalog 的集成边界见第 6／9 节。

| 阶段 | 状态 |
|---|---|
| A．普查 | 完成：独立普查、原资源提取、完整验收与六章 trace 全绿 |
| B．结局与 Staff | 完成：十二分支、独立 MSG VM、原结局图与 Staff、后续入口；完整验收、六章 trace 全绿 |
| C．Music Room 与 Player Data | 完成：原曲目说明／解锁／试听与统计／历史分页；完整验收、六章 trace 全绿 |
| D．回放选单 | 完成：原列表／详情／选章与 t5 播放／返回；最终完整验收、六章 trace 全绿 |

## 2．改动文件清单

与只读 `.tmp/codex/base` 的 SHA-256 清单对照：新增 60 个，修改 9 个，删除 0 个；详见任务根 `changes.json`。没有改 `docs/fidelity.md`。

### 新增源码与文档

- `audio/bgm_observer_test.go`：校验原 BGM 解锁请求与不解锁试听。
- `docs/research/scenes_b.md`：阶段 A 的地址、资源、全局字段与状态机普查。
- `docs/handoff/codex_t11.md`：本交接及四轮实际验收记录。
- `game/scene_progress.go`：原 ST 结局／Staff／音乐字节与应用音乐接线。
- `game/scene_progress_test.go`：原偏移、持久化和回放隔离测试。
- `scene/ending/ending_test.go`：十二分支、原 MSG、全长 Staff 与无头绘制测试。
- `scene/ending/fade.go`：结局使用的原 kind 0／5 私有矩形淡出。
- `scene/ending/msg.go`：独立结局 MSG 容器与只覆盖演出台词的繁体 MST。
- `scene/ending/msgvm.go`：`FUN_0040bd80` 的等待、资源、文字、音乐与条件图。
- `scene/menus/music_data.go`：原十八曲／八行说明与繁体说明覆盖。
- `scene/menus/native.go`：私有菜单 World、小字库、重复输入、位置补间和加载子粒子。
- `scene/menus/player_data_model.go`：原所选机体／难度统计、排行和符卡历史视图。
- `scene/menus/replay_data.go`：原回放详情、缺章、下一章得分及最终 continue 原字。
- `scene/menus/replay_native_test.go`：存储、详情、选章、播放返回及无头绘制测试。
- `scene/menus/scenes_b_test.go`：Music／Player Data 数据来源、解锁、导航与绘制测试。
- `tools/th10extract/main_test.go`：原 archive／汉化 archive 原字节提取测试。
- `tools/th10scenes/main.go`：开发期直接进入五个画面的目视工具，不运行原作。

### 新增嵌入资源

- `assets/anm/e00.anm`：灵梦 A Bad 结局原图和 ANM。
- `assets/anm/e01.anm`：灵梦 B Bad 结局原图和 ANM。
- `assets/anm/e02.anm`：灵梦 C Bad 结局原图和 ANM。
- `assets/anm/e03.anm`：魔理沙 A Bad 结局原图和 ANM。
- `assets/anm/e04.anm`：魔理沙 B Bad 结局原图和 ANM。
- `assets/anm/e05.anm`：魔理沙 C Bad 结局原图和 ANM。
- `assets/anm/e06.anm`：灵梦 A Good 结局原图和 ANM。
- `assets/anm/e07.anm`：灵梦 B Good 结局原图和 ANM。
- `assets/anm/e08.anm`：灵梦 C Good 结局原图和 ANM。
- `assets/anm/e09.anm`：魔理沙 A Good 结局原图和 ANM。
- `assets/anm/e10.anm`：魔理沙 B Good 结局原图和 ANM。
- `assets/anm/e11.anm`：魔理沙 C Good 结局原图和 ANM。
- `assets/anm/staff.anm`：原 Staff 图、条件图与时序。
- `assets/anm/staff.ant`：仅演出使用的繁体 Staff 原补丁资源。
- `assets/anm/title.anm`：原菜单子场景背景、标题、机体、箭头与曲目文字 VM。
- `assets/anm/title.ant`：菜单繁体贴图；测试确认时间线与原版相同。
- `assets/misc/musiccmt.txt`：原十八曲路径、曲目名和固定八行说明。
- `assets/msg/e00.msg`：灵梦 A Bad 原结局控制脚本。
- `assets/msg/e00.mst`：灵梦 A Bad 繁体演出台词。
- `assets/msg/e01.msg`：灵梦 B Bad 原结局控制脚本。
- `assets/msg/e01.mst`：灵梦 B Bad 繁体演出台词。
- `assets/msg/e02.msg`：灵梦 C Bad 原结局控制脚本。
- `assets/msg/e02.mst`：灵梦 C Bad 繁体演出台词。
- `assets/msg/e03.msg`：魔理沙 A Bad 原结局控制脚本。
- `assets/msg/e03.mst`：魔理沙 A Bad 繁体演出台词。
- `assets/msg/e04.msg`：魔理沙 B Bad 原结局控制脚本。
- `assets/msg/e04.mst`：魔理沙 B Bad 繁体演出台词。
- `assets/msg/e05.msg`：魔理沙 C Bad 原结局控制脚本。
- `assets/msg/e05.mst`：魔理沙 C Bad 繁体演出台词。
- `assets/msg/e06.msg`：灵梦 A Good 原结局控制脚本。
- `assets/msg/e06.mst`：灵梦 A Good 繁体演出台词。
- `assets/msg/e07.msg`：灵梦 B Good 原结局控制脚本。
- `assets/msg/e07.mst`：灵梦 B Good 繁体演出台词。
- `assets/msg/e08.msg`：灵梦 C Good 原结局控制脚本。
- `assets/msg/e08.mst`：灵梦 C Good 繁体演出台词。
- `assets/msg/e09.msg`：魔理沙 A Good 原结局控制脚本。
- `assets/msg/e09.mst`：魔理沙 A Good 繁体演出台词。
- `assets/msg/e10.msg`：魔理沙 B Good 原结局控制脚本。
- `assets/msg/e10.mst`：魔理沙 B Good 繁体演出台词。
- `assets/msg/e11.msg`：魔理沙 C Good 原结局控制脚本。
- `assets/msg/e11.mst`：魔理沙 C Good 繁体演出台词。
- `assets/msg/staff.msg`：原 Staff 全长 4960 脚本时刻与条件图。
- `assets/msg/staff.mst`：原样繁体补丁 Staff 对照数据，不执行其控制字节。

### 修改

- `assets/assets.go`：嵌入实际使用的原曲目说明，其余新增资源沿用原目录嵌入。
- `audio/audio.go`：增加原音乐观看记录回调和不解锁试听，不改 PCM／BGM 解码。
- `datfs/layout.go`：MST／ANT／ECT 提取到既有资源目录。
- `game/game.go`：New／BindState 各增加一次音乐接线，场景切换代码未改。
- `scene/ending/ending.go`：原脚本／图／Staff 取代手写三页剧情和 ebitenui。
- `scene/menus/musicroom.go`：原曲目／说明／解锁／试听状态机取代近似 List。
- `scene/menus/playerdata.go`：原统计、机体／难度／历史分页取代全局摘要。
- `scene/menus/replay.go`：原列表、详情、选章与返回取代近似 List。
- `tools/th10extract/main.go`：增加只在开发期使用的 `-archive th10c.dat`。

### 删除

删除文件：无。删除旧 `endingPages`、结局／Music／Player Data／回放的 ebitenui 叠层；不删除其他任务的资源、测试或函数。

## 3．移植的函数地址

| 原作 | Go |
|---|---|
| `FUN_0040b940`／`FUN_0040b560`，`0x40b684…0x40b6b4` | `ending.New`／`EndingIndex`／`Ending.enter` |
| `FUN_0040ba90`／`FUN_00417010` | `Ending.init`／`parseMSG`，text.anm 0x42～0x46、首 NUL XOR、MST 演出覆盖 |
| `FUN_0040bd80` | `msgVM.step`／`write`／`spawnImage`／`playBGM`，全部 0～17 分支 |
| `FUN_0040bd20`／`FUN_0040b9f0`／`FUN_00449c00` 第六返回码 | `Ending.Update`，独立时钟与已看 Staff 的十二次 calc 重跑 |
| `FUN_0040b480`／`FUN_0040c3c0` | `msgVM.reset`／`Ending.completeLoad`，原 Staff 跳转与下一 calc 的资源装载 |
| `FUN_0040c540`／`FUN_00448d50` | `Ending.showLoading`，绝对屏幕 Offset、Start 清备用 RNG 位 |
| `FUN_0040b7b0`／`FUN_0040b3a0` | `Ending.Close`，不清保留的 Stage／应用 RNG／共享历史 |
| `FUN_0043c970`／`FUN_0043bd40`／`FUN_0043c230`／`FUN_0043c1a0` | 私有 `screenFade` 的 kind 0／5、pre-tick alpha／完整两帧末尾、draw 0x31 |
| `0x40b6d5`／`0x40b6e2`／`0x420b4c`／`0x420bd7` | `ScoreFile.MarkEndingSeen`／`MarkStaffSeen`／`MarkMusicPlayed` |
| `FUN_00433ef0`／`FUN_004349e0`／`FUN_00434a20` | `MusicRoom.Update`／`Play`／`Stop`／`parseMusic`，原列表与八行文字、未知曲目二次确认 |
| `FUN_00431ee0`／`FUN_004088c0`／`FUN_00432690`／`FUN_004329f0` | `PlayerData.Update`／`spellIDs`／`SpellRows`／`Statistics`／`DrawPorted` |
| `FUN_004315c0`／`FUN_00431ba0` | `Replay.Update`／`openSlot`／`moveChapter`／`Details`／`DrawPorted` |
| `FUN_0042c670`／`FUN_00401760`／`FUN_00434a80` | 私有 `nativeMenu.spawnTitle`／`smallText`／`move` |
| `FUN_0043ee30`，`0x43ef55`／`0x43f00d` | 私有 `nativeMenu.calcLoading`，仅 ascii 脚本 8 的前插子粒子，不改通用 ANM |

## 4．测试

新增 28 个顶层测试，包含十二结局、三种 Good 难度全长 Staff、原曲目与解锁、三十组统计／真实 native backend、汇总名与所选机体计数、t5 选章／EOF／存档隔离以及五个场景的无头绘制。全部实际通过；没有修改既有断言或重生成 golden。

阶段 A 实际运行 `accept.ps1 -Phase A`：build／vet／`go test -count=1 ./...`／WASM／完整 smoke 全部 exit 0，`gofmt -l .` 输出为空。全仓测试 132.120 秒，完整 smoke 126.946 秒（命令总时长 128.441 秒）。日志为任务根 `checks_A.log`／`results_A.json`。

阶段 B 新增结局／Staff 与 ST 测试全部通过。实际完整 `accept.ps1 -Phase B`：build／vet／全仓 test／WASM／full smoke 全部 exit 0，gofmt 空；全仓 test 93.800 秒，full smoke 117.589 秒。日志 `checks_B.log`／`results_B.json`。

阶段 C 新增测试已实际通过：原十八曲与 BGM 顺序、八行繁体说明、真实 ST 字节门控、未知曲目停止／二次试听且不解锁、十行窗口／20 像素行距／26＋8 帧重复／逐两帧说明；原三十组机体／难度统计、跨 native backend 重载、汇总名／所选机体计数／不套 GUI 99 上限、历史分页与只读数据画面、Music／Player Data 无头绘制、私有加载粒子前插出生顺序；audio 观看回调与回放隔离通过。相关 lint 实际运行 `lint_C_related.log`：六项旧报告，涉及 audio／extract 的旧 Close 与 Option 的 focusIdx，未发现本任务新文件报告；完整 lint 留到收尾。

阶段 C 实际完整 `accept.ps1 -Phase C`：build／vet／全仓 test／WASM／full smoke 全部 exit 0，gofmt 空；test 86.983 秒，full smoke 116.954 秒。日志 `checks_C.log`／`results_C.json`。Title ANT 保留原时间线的新测试也实际通过。

阶段 D 新增测试已实际通过：非连续且非关卡排序的章节详情、十五帧章节输入门控、下一章／最终得分与 global＋0x60 的 continue 字段、缺章跳过、32 帧确认门槛、通过 t5 NewPlayback 的准确切片索引播放并 EOF 返回、存储重载／共享存档字节不变、坏／空槽位、回放详情无头绘制及可选 catalog hook。两个既有回放菜单测试未经修改仍通过；目视 viewer 已编译但未运行。

最终 `accept.ps1 -Phase D -Lint`：build／vet／全仓 test／WASM／full smoke 全部 exit 0，gofmt 空；全仓 test 71.334 秒，full smoke 123.907 秒（命令总时长 125.373 秒）。所有包通过；日志 `checks_D.log`／`results_D.json`。A／B／C／D 四轮完整验收均通过。

额外 `golangci-lint run` 实际 exit 1，默认显示 33 项（errcheck 26／staticcheck 2／unused 5）。只读基线同样 exit 1／33 项；默认 `max-same-issues=3` 会选出不同截取位置，因此没有用默认截取声称完全相同。按实际 `--help` 验证选项后，两树都以 `--max-issues-per-linter 0 --max-same-issues 0` 重跑：均为 51 项（44／2／5），去除行号后的完整报告一致，无新增或移除；见 `lint_full_base.log`／`lint_full_tree.log`／`measurements.json`。没有越界修 Option、probe 或旧测试的 lint。rules 第 6．5 节明确 lint 不列为验收门槛。

资源体积实测：新增 43 个原样资源共 38,782,328 B（36.986 MiB），未嵌入 title_v.anm、无用简体结局图副本或新的 BGM。原资源／繁体版本均经 gamedata 校验，原 MSG 控制字节与原 ANM 时间线测试通过。额外在只读 baseline 与 t11 各构建 `./cmd` 的 WASM：135,242,318 B → 173,829,627 B，增加 36.800 MiB；level-9 gzip 流式测量为 90,337,096 B → 102,174,476 B，增加 11.289 MiB。没有添加依赖；测量结果在 `measurements.json`。

## 5．Trace 结果

阶段 A 顺序生成并比较六章：cmp 全部 0；字节数 94186573／144171327／271969413／319443819／430006225／510140938；motion 一致帧数 15853／17733／29763／29469／39343／40099。使用规定 `orig_d.txt` 与后五章原作 trace，无任何下降。比较后删除本任务大 trace，保留 `trace_A.log`。

阶段 B 同样顺序六章 cmp 全部 0，字节数与 A 相同，motion 仍为 15853／17733／29763／29469／39343／40099，无下降；保留 `trace_B.log`，大文件已删除。

阶段 C 六章 cmp 全部 0，字节数与 A／B 相同，motion 仍为 15853／17733／29763／29469／39343／40099，无下降；`trace_C.log` 保留完整结论，大文件已删除。

最终 D 与 A／B／C 相同，六章全部逐字节相同，没有第一分歧帧：

| 章 | cmp | 输出字节 | 原作 motion 一致帧 |
|---|---:|---:|---:|
| c0 | 0 | 94186573 | 15853 |
| c1 | 0 | 144171327 | 17733 |
| c2 | 0 | 271969413 | 29763 |
| c3 | 0 | 319443819 | 29469 |
| c4 | 0 | 430006225 | 39343 |
| c5 | 0 | 510140938 | 40099 |

四轮共 24 次 cmp／motion 全部通过。所有本任务大 trace 比较后均已删除；基线／原作 trace 与 golden 不动。最终 `trace_D.log` 保留完整 th10diff 输出；c0 严格使用规则中的 `orig_d.txt`，没有删 M 行或放宽容差。

## 6．待决与推测

新增 dump 请求在 `.tmp/codex/t11/dump_requests.md`，继续用全程序 asm 工作。已确认结局 VM 不同于关内 MSG；Staff 是 Good 结局的 opcode 12 切换，不虚构 Extra 结局。原 Music Room 允许未知曲目警告后再次确认试听；解锁记录来自实际 BGM 播放，不以通关数近似。

阶段 B 已由 asm 闭合两个原 C dump 丢失的点：return 6 实际从头重跑 calc 链；opcode 11 原 float32 3 秒对应 180 音频帧。opcode 7 的原回调读已经前移 PC 的＋4 字段作为文件槽，保留此行为；全部原文件均在此放 op 8 的图槽 0。资源异步线程适配为嵌入资源在下一 calc 完成，不读 dat，不新建平台线程。汉化包没有 e07／e08／e09.ant，仅有其简体 .anm，未覆写原图或嵌入无用副本；结局文字使用繁体 MST，Staff 图使用 staff.ant。

- 必需控制流程均由现成 dump＋完整 asm／PE 表闭合；没有代码里的“推测”分支。dump_requests 中的 VM 构造／资源回调、Music 状态机、Player Data／回放 draw C dump 仅保留作架构师独立复核，不等待后才完成代码。
- 字体继续使用既有 DFYuGaSo CPU GDI／D3DX 适配，度量、抗锯齿、阴影仍不是 L3 像素精确；新增菜单字体不读系统字体。小数字来自原 ascii 精灵，不以文字字体替代。
- 持久标题背景、共用输入／淡出与 ANM-91 应在合并时由架构师与 t10 统一；本树运行原子场景背景与私有最小实现，不重写监督者或主标题。后台资源装载的下一 calc 调度是平台适配，需目视核对交接帧。
- 原回放初始化会读额外 `th10_ud????.rpy` 缓存（`0x4316cb`）；t5 公开后端仅 25 个编号槽。默认画面按原 25 行列表实现；额外 catalog 可通过 `ReplayMenuSource` 接入并已经用原 codec 做 hook 测试，但本任务不扩展 t5 的文件枚举后端。此接口边界不等同于已验证任意自命名回放文件的桌面枚举。
- 新音乐标志经原 native CR／ST 后端保存；WASM 仍是 t5／t8 的页面会话后端，不声称刷新后持久。回放不修改共享音乐／历史／统计。
- 已发生的开发期失败均已修：误传 PlaySEID 参数由实际编译器指出后改用现有单参接口；汉化包无 e07.ant，已停止提取该不存在资源，不覆写原版。最终没有未绿的 rules 第 6 节项目。

## 7．建议的 `docs/fidelity.md` 改动

未直接修改主账。建议在“外围”保留 t10／t5 的菜单状态行，另外增加以下完整行，避免把主标题／Option／暂停也误报为 t11 完成：

```text
| 结局／Staff | 原状态机与脚本移植；字形／异步交接与 L3 待确认 | FUN_0040b560/0040b9f0/0040ba90/0040bd80/0040c3c0；PE 0x474440 十二机体条件分支，Easy 或有续关为 Bad，Normal＋零续关为 Good；原 e00-e11.msg/anm，MST 只覆盖繁体台词，Good opcode 12 转 staff.msg，4840 等 Z、4960 结束；保留 t5 Ending.After，Staff return 6 重跑十二次 calc | 十二分支／原字节与繁体覆盖／wait／全长三难度 Staff／无头绘制／后续请求测试；四轮 build/vet/test/WASM/full smoke/gofmt 全绿；六章 trace 逐字节同基线；用户目视 |
| Music Room | 原数据、解锁与播放状态机移植；字形／共享背景待 L3 | FUN_00433ef0/004349e0/00434a20；原 musiccmt.txt 十八曲与八行说明，繁体 musiccmtt.txt 只覆盖文字；ST+0x26 由原 BGM ordinal 写入，未知曲目警告后可再次试听且不解锁；十行窗口、20 像素行距、每两帧说明；title/text/ascii VM | 原曲目／说明／BGM 顺序、真实 ST 解锁／持久化／回放隔离／二次试听／输入与无头绘制测试；四轮验收与六章回归全绿 |
| Player Data | 原所选机体／难度统计、排行与符卡历史移植；字形待 L3 | FUN_00431ee0/004088c0/00432690/004329f0；SaveData.Progress/Rankings 与共享 SpellHistory，六机体／五难度，Total 控制名字、所选 shot 提供计数；PE 0x4743c0 难度表，每页十卡、不套 GUI 99 上限；原小字库与 title/text ANM | 三十组数据来源、native backend 重载、名字／计数来源、分页／导航／只读与无头绘制测试；四轮验收与六章 trace 全绿 |
| 回放选单 | 原列表／详情／选章状态机移植；额外 catalog 与共享演出接线见交付 | FUN_004315c0/00431ba0；t5 ListReplays/LoadReplay/NewPlayback，不改 codec；默认 25 原槽，十五帧选章门控、32 帧确认，按实际 chapter slice index 播放；原下一章开始分与 global 最终分／continue 原字，EOF 返回同菜单并重载存储 | 缺章／非连续非排序索引、原详情数据、t5 存储／播放返回／存档隔离／坏槽／无头绘制与 catalog hook 测试；原旧测试不改，四轮验收与六章回归全绿 |
| 资源嵌入 | 原样资源；汉化仅用于演出 | gamedata、tools/th10extract 的开发期 -archive th10c.dat；t11 增 e00-e11.msg/anm/mst、staff.msg/anm/mst/ant、title.anm/ant、musiccmt.txt，原控制与时间线测试相同；运行时不读 dat，新增原资源 38782328 B | gamedata 原 archive／汉化 archive 校验、原 MSG／ANM timeline 测试与 WASM build；WASM 173829627 B，level-9 gzip 测量 102174476 B |
```

按现有格式的更新记录：

```text
- 2026-11-08 — t11 A-D：先独立普查 scenes_b.md，再按 FUN_0040b560/0040bd80 移植十二结局及原 Staff；Good 走 opcode 12，4840 等确认、4960 结束，保留 t5 post-ending 菜单。Music Room 按 FUN_00433ef0 读原十八曲与八行说明，ST 解锁取原音乐序号，未知曲目二次试听不解锁；Player Data 按 FUN_00431ee0/00432690/004329f0 读所选 CR 与共享符卡历史，不以全局摘要或 GUI 99 上限代替；回放按 FUN_004315c0/00431ba0 接 t5 存储、详情、缺章跳过、实际切片索引播放与返回。删除四场景 ebitenui 近似及手写剧情，原 ANM／MSG 加繁体演出覆盖经 gamedata 嵌入，新增资源 36.986 MiB，WASM/gzip 增 36.800/11.289 MiB。补私有菜单输入、四帧位置补间、加载 ANM-91 与淡出，持久背景／通用演出待架构师同 t10 统一；不改监督者、关内逻辑、GUI 或回放解码。新增 28 顶层测试，A-D 四轮 build/vet/test/WASM/gofmt/full smoke 全绿，六章 24 次 cmp 逐字节相同，motion 15853/17733/29763/29469/39343/40099 不降。额外 lint 默认 33／无限制 51 均在只读基线复现，无新增；字体、画面与设备试听仍需用户 L3，额外 th10_ud????.rpy 后端枚举只留 catalog 接口。
```

## 8．需要用户目视确认的项目

这些命令是用户确认入口，本任务没有运行 viewer、原作或采集器。可先在 t11 工作树执行；合并后在主树执行相同相对命令。用独立用户数据目录，不向原资源 junction 写入。

```powershell
Set-Location -LiteralPath 'D:\GoProjects\th10\.tmp\codex\t11\tree'
. 'D:\GoProjects\th10\.tmp\codex\t11\env.ps1'
$env:TH10_DATA_DIR = 'D:\GoProjects\th10\.tmp\codex\t11\manual-data'
New-Item -ItemType Directory -Path $env:TH10_DATA_DIR -Force | Out-Null
```

1．Good 结局：`go run ./tools/th10scenes -scene ending -character 0 -shot 0 -difficulty 1 -continues 0 -seen`。机体参数分别为 0／1（灵梦／魔理沙），shot 为 0～2；difficulty 为 0～3。对照三张原图、五行文字、颜色、换页、图切换与 Good → Staff。Z／Enter 推进；`-seen` 只用于检查原 Ctrl 门控，去掉它测试首次观看。正常游戏的真正入口仍是 Normal／Hard／Lunatic 六面零续关完成。
2．Bad 结局：`go run ./tools/th10scenes -scene ending -character 1 -shot 2 -difficulty 0`；再用 Normal＋`-continues 1`。对照 e00～e05 两张图及 Bad 不进入 Staff。原条件看续关计数，不看残机、分数或灵击次数。viewer 的 After 为返回标题，真正 post-ending 排行／回放保存须用正常六面 clear 的 t5 接口确认。
3．Staff：`go run ./tools/th10scenes -scene staff -difficulty 3 -seen`；改为 1／2 检查 Normal／Hard。对照 420／480、1900、1990～2065、3800、4780 的背景与 credits，Normal 图 32／Hard 33／Lunatic 34＋35，BGM、滚动层次、每十二 calc 的 Ctrl 快进。4840 时必须等待 Z／Enter，不按应一直停；确认后对照三秒音乐淡出与 4960 结束。
4．Music Room：从 `go run ./cmd` 的 Music Room 进入，或 `go run ./tools/th10scenes -scene music`。新用户目录只有标题曲已听；上下选择未知曲目，第一次 Z 停止 BGM 并显示八行警告，第二次试听；退出恢复标题曲，再进入该未听曲仍未永久解锁。完成对应游戏音乐播放后再看解锁。对照原十八曲顺序／繁体说明、十行窗口、20 像素行距、两帧一行、选择色与 x＝56 的位置、BGM 循环和停止。
5．Player Data：从主菜单进入，或 `go run ./tools/th10scenes -scene data`。可由用户把已有原 `scoreth10.dat` 复制到独立目录的 `score.dat`，不使用转换器。左右六机体、上下五难度、Z 轮转排行／符卡页、X 返回。对照十条排行、最后一位 continue、日期／时分、slow rate、所选机体的局数／时长／所选难度 clear 数；符卡名是否由汇总尝试揭示、当前机体计数及大于 99 的显示。字体与阴影是既有 CPU 近似，不要求误报逐像素精确。
6．回放：用户将既有 th10_05.rpy 等放在独立用户目录 `replay/th10_01.rpy`～`th10_25.rpy`，从主菜单进入，或 `go run ./tools/th10scenes -scene replay`。上下选择、Z 打开详情、等十五帧后选有数据章并 Z 播放；X 从详情恢复同槽游标，播放中 X／Esc 或 EOF 返回同菜单。对照 25 行原小字库、详细日期／机体／难度／slow rate、缺章灰行、下一章开始得分、32 帧淡出；返回后原 score／符卡／音乐进度应不变。原额外自命名 catalog 后端不是本树默认能力，见第 9 节。
7．实际完整流程与集成：正常六面 Good → Staff → 同 Stage 的排行／回放保存；Bad 直接后续菜单；Extra／练习直接 clear menu，不误进结局。合并 t10 后检查持久标题背景、淡入淡出、返回定位以及浏览器音频／页面会话数据。私有子场景与 shared supervisor 交接的首尾帧、字体和加载粒子须与原作目视核对。

## 9．与其他任务的接口

- 保持 `game.Scene`／`Game.GoTo`，不改 `game/game.go` 场景切换。
- 保持 t5 `Ending.After func(*game.Game) game.Scene`，Good 在 Staff 完成后调用，Bad 直接调用。
- 回放使用 `Game.ListReplays`／`LoadReplay` 与 `stage.NewPlayback`；t5 仅公开 25 个编号槽，原额外自命名文件枚举需要后端接口，t11 不绕过存储自行读文件。
- Player Data 读取 t8 `SaveData.Progress`／`Rankings`／`SpellHistory`，Music 读取 `ScoreFile.Stats` 原音乐字节；任何应用级播放记录 hook 另行列明。
- 私有菜单重复、淡入淡出、跨管理器 calc 返回 6 与加载粒子的原作地址和合并 hook 均已在本节列出；不实现 t10 的共享监督者。

阶段 B 新接口：`ending.EndingIndex(*game.GameState) (int,error)`；`ending.NewStaff(*game.GameState,game.Scene) *Staff`；`(*Ending).IsStaff()`／`ScriptName()`／`Close()`／`DrawPorted(*supervisor.Supervisor)`。`ResolveScene func(*game.Game,int32) game.Scene` 可将最终 15／系统覆盖 2 交给 t10；旧调用者继续使用 `After`／`back`。私有矩形淡出地址为 `FUN_0043c8b0/0043c970/0043bd40/0043c230/0043c1a0`，仅 kind 0／5，不含关内震屏或 RNG；合并时可由架构师与 t10 的共享演出统一。

阶段 C 音乐接线：`(*audio.Manager).SetBGMObserver(func(int))`／`PlayBGMPreview(int)`；`(*game.Game).BindMusicProgress(*GameState)`／`RecordMusicPlayed(int)`。普通 BGM 以 FUN_00420b10／00420b90 的原序号写 ST，试听只恢复原 byte 0。Game.New／BindState 已接，不需改关内声音调用；t10 标题在回放退出后可调用 `BindMusicProgress(nil)` 恢复菜单追踪，Music Room 自己已处理。所有回放共享存档仍只读。

阶段 C 场景接口保持 `NewMusicRoom(back)`／`NewPlayerData(save,back)`，新增只读曲目／统计／历史视图及 `DrawPorted(*Supervisor)`／`Close()`。每个场景的 `Background func(*Supervisor)` 可接 t10 的持久标题背景；默认已运行原子场景 0x5e／0x5f 背景，不复制主标题／选人状态机。私有 `menuRepeat`＝FUN_0040ac20／0040ace0，`move`＝FUN_00434a80（4 帧／mode 0）；ascii 脚本 8 的 ANM-91 子粒子只在本包补最小实现（0x43ef55／0x43f00d），共享 ANM 仍不变，合并时可统一。

阶段 D：`NewReplay(back)` 签名保持不变；默认 Source 是 Game 的既有 `ListReplays()`／`LoadReplay(int)`，分离 metadata 与完整输入读取；播放仍调用 `stage.NewPlayback(g,r,actualSliceIndex,select.ReplayScript,thisMenu)`。新增 `ReplayMenuSource` 可供架构师为原 th10_ud????.rpy 检索接 catalog，t5 默认后端仍仅 25 个编号槽；不绕过存储做目录或文件读取。`Details()`／`Slots()`／`DrawPorted(*Supervisor)`／`Close()` 可供 t10 使用。确认时的私有 kind 5 淡出对应 `0x4319cd`（32 帧、原 draw 0x2b），合并时与 t10 基础演出统一。

对外签名与原调用关系：

```go
ending.EndingIndex(*game.GameState) (int, error)
ending.New(*game.GameState, bool, game.Scene) *ending.Ending
ending.NewStaff(*game.GameState, game.Scene) *ending.Staff
(*ending.Ending).IsStaff() bool
(*ending.Ending).ScriptName() string
(*ending.Ending).Close()
(*ending.Ending).DrawPorted(*supervisor.Supervisor)
(*audio.Manager).SetBGMObserver(func(int))
(*audio.Manager).PlayBGMPreview(int)
(*game.Game).BindMusicProgress(*game.GameState)
(*game.Game).RecordMusicPlayed(int)
(*game.ScoreFile).EndingSeen(int) bool
(*game.ScoreFile).MarkEndingSeen(int)
(*game.ScoreFile).StaffSeen() bool
(*game.ScoreFile).MarkStaffSeen()
(*game.ScoreFile).MusicUnlocked(int) bool
(*game.ScoreFile).MarkMusicPlayed(int)
menus.NewMusicRoom(game.Scene) *menus.MusicRoom
menus.NewPlayerData(*game.SaveData, game.Scene) *menus.PlayerData
menus.NewReplay(game.Scene) *menus.Replay
(*menus.MusicRoom).Tracks() []menus.MusicTrack
(*menus.MusicRoom).TrackUnlocked(int) bool
(*menus.MusicRoom).Play(*game.Game, int) error
(*menus.MusicRoom).Stop(*game.Game)
(*menus.PlayerData).Statistics() menus.PlayerStatistics
(*menus.PlayerData).SpellRows() []menus.SpellHistoryRow
(*menus.Replay).Slots() []game.ReplaySlot
(*menus.Replay).Details() menus.ReplayDetails
```

三个菜单场景均另实现 `Close()`／`DrawPorted(*supervisor.Supervisor)`，仍实现原 `game.Scene`。原监督者 `FUN_004218d0` 创建结局，原 Title `FUN_0042cdf0` 分派三菜单；共享生命周期归 t10。Staff 是 opcode 12 的同对象段，`NewStaff` 只是 viewer／监督者可用的显式入口。

调用了但不属于本任务的函数：t5 `FUN_004296f0/0042a200` 存储检查／完整回放读取与 `FUN_00428f60` 播放入口；本树只通过 Game／NewPlayback 调用。`FUN_0043ee30/00448d00/00448e30/00448980/004451c0` 通过现有 ANM／World／Supervisor 调用；只有 ascii 脚本 8 的 91 子集在本包私有补齐。t5／t8 后续排行保存 `FUN_00423370/004236f0` 仍由 `After` 接，不重复加分或计数。共用矩形、资源装载、音乐接口、重复输入、持续背景与 draw 0x2b／0x31 的顺序由架构师合并时统一，本树没有写 `game.GoTo`、没有新增关内 RNG 消耗。

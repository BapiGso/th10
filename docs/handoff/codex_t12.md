# t12：差分修正 A（魔理沙 A／B／C 射击）

工作树：`D:/GoProjects/th10/.tmp/codex/t12/tree`。任务开始日期：2026-11-08。

## 1．摘要

本轮修正与取证循环、回归和 rules 第 6 节验收已完成。三条回放的结果如下；尚缺原作动态字段的部分明确留待补证，不冒充三条全章匹配。

| 阶段 | 结果 |
|---|---|
| 魔理沙 B | 修正首帧光束误结束，252→3793；越过两个已确认半帧后到 4100 公共命中／死亡差异，转交 t13／动态取证。 |
| 魔理沙 A | 修正 state 0 漏更新历史，真实火箭差分 3340 消失；越过七个已确认半帧后，全部完整参考帧匹配至 6944。 |
| 魔理沙 C | 首个 motion 分歧仍为 5011；前 256 弹运动一致，复刻释放的两槽缺原动态状态，不能确定哪槽多活及原因，已写 dump 请求。 |
| 火力上限 | 已核对原作与复刻均为 500（5.00），射击／副机档位封顶为 4；补 0…100 raw 的全阈值与饱和测试，没有虚构上限逻辑改动。 |
| 验收 | build／vet／全仓 test／WASM／gofmt／完整 smoke 通过；六条新回放不倒退，th10_05 六章与基线逐字节相同。 |

A 的 6945、C 的 5352 末帧行截断，不能称末帧已验证。lint 按全局 AGENTS 实际执行，默认 33 项；不限输出时基线／本树均为 51 项、报告一致，没有新增告警。rules 第 6．5 节明确不以 lint 作为 DONE 门控。

实际按 B → A → C 推进差分，随后完成完整验收、顺序重跑新六条与 th10_05 六章。所有回放 th10diff 调用均使用 `-scope motion`，没有改变比较容差。没有运行原作、probe 或 Ghidra，没有修改数据 junction、已有测试／golden、go.mod／go.sum、主树源码或 `docs/fidelity.md`。所有完整 trace、比较用参考前缀均逐条删除；只保留 diff 日志和小型取证片段。

## 2．改动文件清单

新增六个文件：

- `docs/handoff/codex_t12.md`：取证、修正、回归、待决及合并接口记录。
- `sim/marisa_shots_test.go`：真实 B 首帧／252 伤害与 A 复活后火箭命中时间线回归。
- `entity/player/marisa_power_test.go`：三型 101 个 raw 火力值的档位、组表与 5.00 饱和测试。
- `entity/player/marisa_history_test.go`：state 0 无移动／focus=true 时的历史推进与终端样本测试。
- `tools/th10diff/after_test.go`：显式比较起点、shift、漏采和后续分歧不被屏蔽的测试。
- `trace/short_bullet_test.go`：截断 B 行返回错误而不是 panic，完整行仍正常解析。

修改十四个文件：

- `entity/player/options.go`：抽出原相同历史移位队列，正常移动门控保持，供 state 0 共用。
- `entity/player/player.go`：respawn 在原位置无条件推进 33 项历史，不改时钟或移动公式。
- `game/state.go`：修正 GainPower 的上限说明，写明原 raw 0x64＝5.00；无逻辑变化。
- `sht/shot.go`：补 PowerLevel 的原除法／档位钳制地址，明确其不是资源上限；无逻辑变化。
- `scene/stage/ecl/vm.go`：提供 BindContext，在首次 calc 前安装已有管理器 hook。
- `scene/stage/stage1/ecl_script.go`：Init 提前绑定一面管理器。
- `scene/stage/stage2/stage2.go`：Init 提前绑定二面管理器。
- `scene/stage/stage3/stage3.go`：Init 提前绑定三面管理器，消除 B 首帧误结束。
- `scene/stage/stage4/stage4.go`：Init 提前绑定四面管理器。
- `scene/stage/stage5/stage5.go`：Init 提前绑定五面管理器。
- `scene/stage/stage6/stage6.go`：Init 提前绑定六面管理器。
- `scene/stage/extra/extra.go`：Init 提前绑定 Extra 管理器，保持七关同一生命周期。
- `tools/th10diff/main.go`：新增显式 `-after N`，默认行为与容差常量不变。
- `trace/trace.go`：短 B 行先检查字段数，明确拒绝损坏数据，不自动忽略。

删除文件：无。全树 SHA-256 对比仅出现上述改动，清单见 `.tmp/codex/t12/logs/source_changes.csv`；原有测试、golden、资源和依赖文件未变。

工作树外只有规则允许的草稿／日志：`.tmp/codex/t12/` 中的验证脚本、动态请求和观察者，以及 `.tmp/scripts/t12_trace.py`（只读取证、字节 cmp 和完整帧前缀复制）。交付文档另复制到主树 `docs/handoff/codex_t12.md`。

## 3．移植的函数地址

- `FUN_00417870` 的 `0x417b33／0x417b86` → `ecl.VM.BindContext` 与七关 Init，保证首次 calc 前已有管理器，不提前执行 ECL。
- `FUN_00428280` 的 `0x428328` → 现有 `Pool.ManagerPresent`，本次只修复它的绑定时机。
- `FUN_00425730` state 0 的 `0x4257b3–0x425833` → `Player.respawn` 调 `advancePositionHistory`，复活上移时无条件推进历史；与 `FUN_004250b0` 的 `0x425532–0x4255b3` 共用同一队列实现，正常移动门控保持。
- `FUN_00418930` 的 `0x41893c／0x41894b／0x418963`：raw 16 位火力钳到 `0x64`，项目值 ×5，即 500（5.00）；资源上限已经正确，没有虚构“改成 500”的逻辑改动。
- `FUN_00428160` 的 `0x428160–0x42817d`：raw／20 选档，raw／20 ≥5 时改回档 4；阈值 raw 20／40／60／80，对应 1.00／2.00／3.00／4.00；5.00 仍使用档 4，低速加 5。
- `FUN_00426520` 的 `0x4265b0–0x4265e1` 与 PE 表 `0x47476c／0x474778` 已复核：初始化 1→`0x428ad0`，更新 1→`0x428b10`、2→`0x428c20`，现有六份 SHT 解码无该处差错。

## 4．测试

新增九个测试，完整验收中全部通过：

| 测试 | 内容／修正前失败证据 |
|---|---|
| TestMarisaBLasersStartOnFirstFrame | 首帧四束光和两 World 句柄；修正前 beams=0。 |
| TestMarisaBFrame252LaserDamage | 原 frame252 enemy[2] HP=11；修正前 HP=20。 |
| TestMarisaARespawnTrailAndRocketHit | 原3339…3343 HP为767／767／766／766／765；修正前3340／3342／3343各早扣1。 |
| TestMarisaPowerCeilingAndShotThresholds | A／B／C，raw0…100，共303个火力样本；高／低速组选对，495→500和满火力饱和。 |
| TestRespawnAdvancesOptionHistoryWithoutMovement | 0／1／4副机、focus=true、无移动的state0；修正前历史头仍为40000而非本帧复活位置。 |
| TestDiffAfterKeepsLaterDivergences | cutoff只排除前缀，后续超容差仍失败，负cutoff为usage error。 |
| TestDiffAfterUsesReferenceFrameBeforeShift | cutoff使用A原帧号，与shift组合仍比对正确帧。 |
| TestDiffAfterPreservesSampleGapsAndLengthCheck | 截断前缀不计作漏采，后续漏采和复刻先结束仍按原规则处理。 |
| TestReaderRejectsTruncatedBulletRows | 末行 `B 25.`／缺type和color返回短行错误，合法行不变。 |

开发中相关包测试已通过。最终实际运行结果如下，日志在 `.tmp/codex/t12/logs/final/`：

| 检查 | exit | 结果／耗时 |
|---|---:|---|
| gofmt -l . | 0 | 输出为空，0.179秒。 |
| go build ./... | 0 | 5.433秒。 |
| go vet ./... | 0 | 2.292秒。 |
| go test -count=1 ./... | 0 | 全仓通过，90.055秒。 |
| GOOS=js GOARCH=wasm go build，排除tools | 0 | 4.835秒；先在host枚举包，避免Windows-only probe的枚举问题。 |
| TH10_SMOKE=full go test -count=1 -run TestReplayDrawSmoke ./scene/stage/ | 0 | 完整回放目录绘制通过，118.056秒。 |
| golangci-lint run --timeout 10m | 1 | 默认33项：errcheck26、staticcheck2、unused5，5.137秒。 |

没有隐藏 lint 失败。另在只读 base 和本树分别运行 `--max-same-issues 0 --max-issues-per-linter 0`，两树均为51项（errcheck44、staticcheck2、unused5）；只归一化改动导致的行／列号后，报告差集为0。默认最多显示3条同文本问题，导致两次默认报告中一个 f.Close 的显示位置不同，不是新增问题。证据为 `logs/lint_baseline_unlimited.txt`、`logs/lint_final_unlimited.txt`。没有通过修不相关文件或放宽 lint 配置掩盖它。

## 5．Trace 结果

### 六条新回放修正前后

| 回放，第0章 | 原作trace | 修正前首个motion分歧帧 | 最终原样比较首帧 | 已确认半帧之后／原因 | 不倒退 |
|---|---|---:|---:|---|---|
| th10_08 | orig_th10_08.txt | 4915 | 4915 | 同一op16半帧；属于t13回放，本任务不越过。 | 是。 |
| th10_04，A | orig_th10_04.txt | 2553 | 2553 | 3340真实差分已修；排除七个已证实半帧后，至6944完整参考末帧匹配。 | 是。 |
| th10_03，B | orig_th10_03.txt | 252 | 3793 | 排除3793／3887后到4100公共命中差分。 | 是。 |
| th10_10，C | orig_th10_03_as_C.txt | 5011 | 5011 | 未采样槽的计数差持续，原动态字段不足。 | 是。 |
| th10_02 | orig_th10_02.txt | 6039 | 6039 | player pos (0.000,480.000) != (0.000,400.000)，t13复活范围。 | 是。 |
| th10_01 | orig_th10_01.txt | 4637 | 4637 | player pos (-71.680,105.820) != (-76.180,105.820)，t13移动范围。 | 是。 |

修正前列为任务给定的分诊基线，其中B252已在本任务改动前复现；最终列是本树交付前六条全部实际重跑的结果，不把cutoff结果代入原样比较列。完整命令、原文及CSV在 `logs/traces_final/`。A累计验证6877个有效采样帧至6944（6884个采样去掉七个确认半帧）；B累计4070个有效采样帧至4099；C为4999个采样帧至5010。F0与复刻起点不对齐的帧不计入，漏采不是分歧。仅motion受验证，不宣称score／faith／自机state等未比较列一致。

复刻侧六条run实际最大火力依次为500／500／500／400／500／500；B已出现5.00，进一步确认并非被资源上限4.00截断。C没有自然升到5.00，不据此推断上限。

### 分歧修正记录

1．th10_03，252：`DIVERGE at frame 252: enemy[2] hp 11 != 20`。`FUN_00428160` 首相位发射四束激光，但复刻 `Context.ManagerOrder` 尚未绑定，`FUN_00428280` 的 no-manager 门控误结束首帧四束光，第 6 帧才重建。现场 Go 观察首帧只有主弹，250…253 帧激光 age 为 245…248；原组表应从首帧保持光束，所以伤害时钟相差 5 帧。提前 BindContext 后帧 252 造成原 SHT 的 9 点伤害，HP=11。依据 `0x417b33／0x417b86`、`0x428328`、`FUN_00427c70`（Cur≠Prev 且 Cur%4=0）。完整分诊／失败测试／观察日志在 `.tmp/codex/t12/logs/` 与 `evidence/`。

2．th10_03，3793：`DIVERGE at frame 3793: dialog {entry 0 clock 32 wait 0 signal 0 at 32 op 16} != {entry 0 clock 33 wait 0 signal 0 at 92 op 5}`。确认 probe 半帧：原 B 3792 为 clock32／op13，3793 仍 clock32 却停在同时间的 op16，3794 为 clock34／op5；原 C 相同输入、同一帧 3793 为 clock33／at92／op5，与复刻一致，3794 再次一致。op16 的 GDI 文本路径 `FUN_00415e90` 属已知慢调用窗口。证据 `m03_probe_3793.txt`、`m03_c_probe_3793.txt`、`m03_sim_3793.txt`。新增显式 `-after 3793` 继续，不删 M 行，不放宽容差。

3．th10_03，3887：`DIVERGE at frame 3887: dialog {entry 0 clock 126 wait 0 signal 0 at 126 op 16} != {entry 0 clock 127 wait 0 signal 0 at 236 op 4}`。同类半帧：原 B 3886 clock126／op12，3887 clock126／op16，3888 clock128／at236／op4；原 C 3887 clock127／at236／op4，与复刻一致，3888 再次一致。证据同名 `*_3887.txt`。显式 `-after 3887` 继续，4100 首次出现非 MSG 差分。

4．th10_03，4100（未修）：`DIVERGE at frame 4100: bullet[8] pos (-91.098,67.476) != (-94.348,69.095)`。仅第8条敌弹不同，其他13颗与敌机运动一致；差异在4101持续，原作已提交死亡、残机8，复刻仍在决死状态、残机9，不是单帧采样问题。关联公共命中／消亡移动／死亡，但缺原矩形、槽状态和时钟，细部根因未闭合，不以 `-after` 越过。已写动态请求，该自机死亡路径属于t13接口边界。

5．th10_04，2553：`DIVERGE at frame 2553: enemy count 7 != 4`。确认半帧：原2553的自机已从y339.600移到337.600，中Boss前身已移动、新9999 HP helper已创建，其他旧敌机尚未被扫掉。实际提取全部157条B行比较，2552与2553的逐行差数为0；不是仅看前8行。2554又与复刻motion一致（新Boss位移到−27.390／112.110，弹数155），随后一直匹配到3340。原作 `FUN_0040dc80／0040e770` 当帧创建新对象、执行阶段／扫除后才轮到敌弹calc，采样落在该中段，不是“射击杀多了”。证据 `m04_orig_2553.txt／m04_sim_2553.txt`、`m04_orig_2552_2553_all.txt`。未改逻辑，显式 `-after 2553` 继续。

6．th10_04，3340：`DIVERGE at frame 3340: enemy[2] hp 767 != 766`。原 3339…3343 HP 为 767／767／766／766／765；修正前为 767／766／766／765／764。原 `FUN_00425730` state 0 在 `0x4257b3–0x425833` 无条件推进 33 项历史（先重复旧终端样本，再倒序移位，写本帧复活位置），复刻仅 snapOptions，没有历史队列。这使复活后的魔理沙 A 副机偏高，首枚火箭早一帧命中，并把区域四 tick 排除相位提前。新增 `advancePositionHistory` 共用正常移动的相同循环，在 respawn 原位置调用；没有硬编码帧号、HP 或额外延迟。修正后五帧 HP 时间线通过，motion 推到5899；爆破伤害仍走既有 `FUN_00428630／00427b50`，没有改伤害或面积。失败测试与现场 AREA 日志见 `logs/m04_*_before.txt`、`evidence/m04_3334_3343.txt`。

7．th10_04，5899／6009／6072／6074／6084／6086：六处 `FUN_00415e90` op16 半帧，原文和相邻 M 行分别保存在 `logs/m04_r3…r9_*motion.txt`、`evidence/m04_*_<帧>.txt`。原／复刻的 M 字段为：5899 `clock34 wait0 at34 op16`／`clock34 wait599 at34 op10`；6009 `clock38 at38 op16`／`clock39 at42 op9`；6072 `clock110 at110 op16`／`clock111 at114 op13`；6074 `clock118 at118 op16`／`clock119 at122 op10`；6084 `clock158 at158 op16`／`clock159 at162 op10`；6086 `clock166 at166 op16`／`clock167 at170 op10`（除5899外 wait 都为0）。5899 下一帧 wait598，说明该帧 op10 已初始化600并扣到599；其他后续有效帧恢复同一时钟／指令。6073 漏采，不把一次0帧比较称为匹配证据；6071 与6075 的完整 motion 单帧对比各 MATCH 1，6074之后6075…6083、6084之后6085、6086之后6087…6944均对齐。逐次确认后增加显式 cutoff，不删除M行或改变容差。

上述六处th10diff原文：

```text
DIVERGE at frame 5899: dialog {entry 0 clock 34 wait 0 signal 0 at 34 op 16} != {entry 0 clock 34 wait 599 signal 0 at 34 op 10}
DIVERGE at frame 6009: dialog {entry 0 clock 38 wait 0 signal 0 at 38 op 16} != {entry 0 clock 39 wait 0 signal 0 at 42 op 9}
DIVERGE at frame 6072: dialog {entry 0 clock 110 wait 0 signal 0 at 110 op 16} != {entry 0 clock 111 wait 0 signal 0 at 114 op 13}
DIVERGE at frame 6074: dialog {entry 0 clock 118 wait 0 signal 0 at 118 op 16} != {entry 0 clock 119 wait 0 signal 0 at 122 op 10}
DIVERGE at frame 6084: dialog {entry 0 clock 158 wait 0 signal 0 at 158 op 16} != {entry 0 clock 159 wait 0 signal 0 at 162 op 10}
DIVERGE at frame 6086: dialog {entry 0 clock 166 wait 0 signal 0 at 166 op 16} != {entry 0 clock 167 wait 0 signal 0 at 170 op 10}
```

8．th10_04，6945（参考文件错误）：末行只有 `B 72.812 33.664 ang -0.35903 spd 3.300`，原 Reader 索引越界。本次加入短 B 行检查，现返回 `th10diff: read A: short bullet row: got 7 fields, want at least 11`，不忽略该行。原文件不改；任务目录只复制完整的0…6944参考前缀比较，`-after 6086` 为 `MATCH: 856 frames compared`，两个漏采帧、37个复刻尾帧不比较。大 trace／参考前缀均删除；6945不报已验证，已请求补完整末帧。

9．th10_10，5011（未修）：`DIVERGE at frame 5011: bullet count 867 != 866`。前256弹的运动全一致，5012…5014仍差1，不能判为半帧。复刻当帧释放735（type18，屏内消亡 VM 结束）和1960（type18，越界），这两槽未出现在原256行采样中，不能确定原作哪槽多活及原因。请求原4992…5016槽状态／VM时钟／实际速度和自机判定矩形；没有给消亡加3帧、加容差或跳过5011。C末帧5352还含截断的 `B 25.`，另请求补完整数据。

### th10_05 六章

六章逐章顺序生成，与 `D:/GoProjects/th10/.tmp/trace/d1c2/base_cN.txt` 做原生cmp，再用th10diff motion比原作，比较后立即删除。最终结果：

| 章N | 基线cmp exit／首个分歧帧 | 原作trace | 原作motion一致帧数，修正前→后 | 字节数 | 原因 |
|---|---|---|---:|---:|---|
| 0 | 0，无分歧。 | orig_d.txt | 15853→15853 | 94186573 | 不射击；两处修正不改变该章trace。 |
| 1 | 0，无分歧。 | orig_05_s2.txt | 17733→17733 | 144171327 | 不射击；历史队列不参与该灵梦C的运动结果。 |
| 2 | 0，无分歧。 | orig_05_s3.txt | 29763→29763 | 271969413 | 不射击；无魔理沙B首枪路径。 |
| 3 | 0，无分歧。 | orig_05_s4.txt | 29469→29469 | 319443819 | 不射击；trace各列逐字节不变。 |
| 4 | 0，无分歧。 | orig_05_s5.txt | 39343→39343 | 430006225 | 有死亡／复活，但历史队列不改变不射击的灵梦C逻辑。 |
| 5 | 0，无分歧。 | orig_05_s6.txt | 40099→40099 | 510140938 | 有灵击／死亡／复活；伤害、RNG、时钟未改，trace逐字节不变。 |

“修正会改变逻辑”适用于射击的魔理沙回放；不强迫不射击的th10_05产生差异。本次六章原作比较全部exit0，没有一致帧数下降；参考结束后的尾帧仍不算验证。记录在 `logs/traces_final/chapters.csv`，完整trace目录已清空。

## 6．待决与推测

动态与尾部请求已写入 `D:/GoProjects/th10/.tmp/codex/t12/dump_requests.md`：B4100公共命中／死亡字段；C5011未采样槽735／1960；A6945／C5352文件尾部补全。所有相关静态函数已查既有dump，没有重复申请已有Ghidra输出。现有probe没有指定敌弹槽的动态dump选项，不虚构CLI。没有“推测”运行时补丁；明确的未知点保持不改。下一步由t13／架构师补公共命中字段或原动态槽后继续。

## 7．建议的 `docs/fidelity.md` 改动

不直接修改总账。建议补充／拆分下列完整状态行，避免把B／C剩余公共差分写成六机体已全部动态位精确：

```text
| 魔理沙 A／B／C 射击 | 有界漂移（A有效参考motion已匹配；B／C公共命中／消亡差分待决） | 六份原SHT及 `FUN_00426520／00428160／00427e90／00428280／00428630`；t12按 `FUN_00417870` 的0x417b33／0x417b86提前绑定管理器，消除B首帧四束光误结束及5tick伤害相位偏移；按 `FUN_00425730` state0的0x4257b3–0x425833补复活历史推进，消除A3340火箭早命中。B的motion从252推到3793，两个证实半帧之后为4100；A排除七个证实半帧后至完整参考末6944匹配；C5011未采样敌弹槽差分待动态dump，A6945／C5352原文件尾行损坏不计已验证 | 九个新增测试，全仓build／vet／test／WASM／gofmt／完整smoke通过；新六条motion首分歧均不倒退，th10_05六章cmp相同；L3与B／C待决 |
| Power资源上限／射击档位 | 位精确（原上限与各阈值核对） | `FUN_00418930` 的0x41893c／0x41894b／0x418963将16位raw火力钳到0x64＝5.00，项目值乘5为500；`FUN_00428160` 的0x428160–0x42817d按raw／20选档，阈值1.00／2.00／3.00／4.00，raw／20≥5钳回档4，低速加5，5.00仍四副机；原资源上限并非4.00，现有代码已正确，t12只补依据注释与测试 | TestMarisaPowerCeilingAndShotThresholds，三型raw0…100、高／低速组与饱和边界；B复刻trace实际max_power=500 |
| 差分器／trace输入检查 | 位精确（工具规则，默认比较不变） | th10diff新增显式 `-after N`，按A原帧号排除≤N、再按shift对齐；只用于交付记录中的已证实半帧，不改eps／ang-eps／打印舍入预算，不删M行；短B行明确报格式错误，不再索引panic，也不自动忽略损坏末行 | 三项after测试与TestReaderRejectsTruncatedBulletRows；全部既有diff／trace测试通过 |
```

候选更新记录：

```text
- 2026-11-08 — t12：新魔理沙B trace在252帧HP11／20，查明首帧四束光因Go管理器hook未绑定被FUN_00428280误结束、第6帧重建，伤害相位偏5tick；按FUN_00417870首次calc前创建／复用管理器的顺序，在七关Init提前BindContext。魔理沙A2553为创建／扫敌中段半帧，继续后3340真实火箭早命中来自state0遗漏33项历史推进；按0x4257b3–0x425833移植同一队列，不改移动、伤害或时钟。A有效motion至6944（末6945损坏）、B至3793及已证实半帧后的4100公共差分，C5011未采样槽差分待动态数据。火力上限核实为5.00、射击档位封顶4，补303个raw样本；新增-after仅用于已证实半帧，短B行报错不忽略。九个新增测试和完整build／vet／test／WASM／gofmt／smoke通过；六条新回放不倒退，th10_05六章与base逐字节相同、motion为15853／17733／29763／29469／39343／40099。lint默认33项，不限输出基线／本树各51项完全相同；L3及B／C动态公共字段待补，不称三条全章匹配。
```

## 8．需要用户目视确认的项目

没有启动可见窗口或运行原作；完整draw smoke通过只证明绘制路径无panic，不是L3像素一致。

可用已有手动debug入口（本任务未执行），将 `-debug-shot=b` 换成a／c：

```powershell
go test ./cmd -run TestDebugStage -timeout 0 -args -debug-stage=3 -debug-char=marisa -debug-shot=b -debug-diff=lunatic
```

- B：首按Z应立即出现当前火力对应数量的激光（4.00／5.00为四束）；Shift／移动时枪口跟随副机，命中亮／暗切换，松Z在原burst末结束，不先闪退再等第6帧重建。
- A：高速拖尾、低速相对位置，以及死亡后复活上移期间的副机恢复；重点看第一枚复活后火箭的出生位置，不应留在旧死亡轨迹形成提前命中。
- A：火箭命中火花与32／1.4／13的区域演出；本任务未另造视觉圈，保持原ANM。
- C：低速绝对锁定、解除后的30tick延迟；本轮没有为未采样敌弹增造延寿或移动容差。
- 通过正常选人 `go run ./cmd` 收集到5.00：GUI应显示5.00并有满火力附加VM，但只有四副机，不生成“第五副机”。

用户补原作动态快照及完整末帧后，按dump_requests中的复现命令继续B／C；不能因A前缀匹配就认定6945损坏帧正确。

## 9．与其他任务的接口

t13 负责灵梦回放、MSG、自机移动／死亡／复活和震屏。本次公共改动：`func (vm *ecl.VM) BindContext(ctx *stage.Context)` 将已有 ManagerOrder／Boss0／Spell／CalcShake hook 从第一次 Update 提前到七关 Init，Update 仍重复绑定；不运行任何 ECL、移动、时钟或 RNG。目的仅是符合 `FUN_00417870` 在首次 calc 前创建／复用敌机管理器的顺序，使魔理沙 B 光束从首帧存活。t13 若修改这些 hook，应保留提前绑定，不重复加同类修补。`Pool.ManagerPresent` 的签名与 gate 条件未改。

`tools/th10diff` 仅新增 `-after N`（跳过 A 原帧号 ≤N，与 shift 独立）；默认路径与容差常量不变，所有正式对比仍为 motion。3793／3887 的证据可供 t13 复用，不能用该选项掩盖未取证的死亡差异。不修改 t10／t11 的关外场景。

另一处公共改动：`Player.respawn() bool` 在 syncPos／snapOptions 后、清弹／state0→1过渡前调用 `advancePositionHistory()`，对应 `0x4257b3–0x425833`；正常移动的 `updatePositionHistory(moved bool)` 仍只在非focus且移动时移位。t13 合并复活代码时保留此历史推进，不能把状态0的focus／移动门控套进去；过渡帧若继续高速移动，原作会再走一次正常移动历史，当前实现也如此。没有改任何死亡／无敌／移动时钟或判定条件。

`trace.parseBulletLine` 只拒绝字段不足的B行，完整行解析与运动比较不变；未自动忽略损坏末行。A使用完整参考前缀只验证到6944，补全后可直接用原文件复跑。此工具改动供t13处理相同尾部数据时复用。

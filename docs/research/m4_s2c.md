# M4 S2c 取证：STD 的 +0x1eec、+0x2a18 位、+0x2a1c 计时器的写入者（2026-11-01）

依据 th10.exe v1.00a 的 Ghidra 反编译与汇编（`tools/ghidra/dump.py` 的副本，项目 `%LOCALAPPDATA%/th10-ghidra-render`，只读；未运行游戏、未改 exe）。新增 dump：`.tmp/ecl_dump/s2c/`（清单见第 7 节）。前置：`m4_render_re.md` §4.2–§4.4（STD 构造、calc、draw）、`m4_s2b.md`（写法）。

约定：

- “STD” 指 `DAT_004776e8` 指向的对象（主 STD）；“STD2” 指 `DAT_004776e4` 指向的对象。
- 地址写作 `0x……`。证据写作 `地址`（dump 文件见同段或第 7 节）。
- 无标注者为事实（有指令地址与 dump 证据）；“（推测）”为由结构推出、未逐条验证。

---

## 0. 结论速览

| 字段 | 写入者（指令地址） | 写入的值 / 条件 | 调用链（上溯） | 置信度 |
|---|---|---|---|---|
| STD+0x1eec | **非零写入者未找到**。全程序唯一的写入是 `FUN_00402720` 末尾清零（0x40282b） | 只写 0 | 无 | 事实（搜索范围见 §1.2） |
| STD+0x2a18 bit0（值 1） | `FUN_00402230` 0x40241e 置位；`FUN_00402850` 0x402a77 置位、0x402a8f 清零；`FUN_00409c00` 0x409c18 置位；`FUN_00408d60` 0x408d7f 清零 | 见 §2 | 见 §5 | 事实 |
| STD+0x2a18 bit1（值 2） | `FUN_00404530` 0x40459f 置位（作用于 STD2）；`FUN_00402ca0` 0x402fcf 清零 | 置位：关卡首帧且 STD2 非空；清零：计时到期 | 见 §5 | 事实 |
| STD+0x2a18 bit2（值 4） | `FUN_004045b0` 0x404604 置位（作用于主 STD）；`FUN_00402ca0` 0x402fcf 清零 | 同上 | 见 §5 | 事实 |
| STD+0x2a18 bit3（值 8） | `FUN_00402ca0` 0x402fb6 置位 | bit1 已置位，且计时到期 | 无清零写入 | 事实 |
| STD+0x2a1c 计时器（ZunTimer） | Set：`FUN_00404530` 0x404581 等、`FUN_004045b0` 0x4045e6 等、`FUN_00405410(1)`（经 `FUN_00402850` 0x402a7d）；Tick：`FUN_0044bf40`（经 `FUN_00402ca0` 0x402f95） | +0x2a20 即当前值（cur）；见 §3 | 见 §5 | 事实（布局由多处互证） |
| 三个特殊 VM（+0x1f08 / +0x22b4 / +0x2660） | 构造 `FUN_00402160`（0x4021db）；析构 `FUN_00402440`（0x4025ed）；步进 `FUN_00402720`（0x4027df、0x4027f9、0x402805） | **启动点未找到**（无脚本号写入） | 见 §4 | 事实（无启动点）；“从不启动”为（推测） |

---

## 1. STD+0x1eec（每帧“特殊 VM / 暗化”请求）

### 1.1 读者与写者

- calc `FUN_00402720`（节点 0x403050，优先级 0xc）：0x4027cf `MOV EAX,[EBX+0x1eec]`。非零则步进三个特殊 VM（§4），之后 0x40282b `MOV dword ptr [EBX+0x1eec],0x0` 无条件清零。
- draw 7 `FUN_00402850`：0x402c44 `CMP [EBP+0x1eec],EBX`。非零则 `game[0x73245c]=1`、`game[0x732458]=0xff404040`（`r6/func_402850.c:173`；m4_render_re §3.6）。
- draw 0xa `FUN_00402ca0`：0x402f37 同上（`r6/func_402ca0.c:81`）。
- **写者**：只有 0x40282b 一条，值为 0（`s2c/_writes.txt` 运行 1 的 `== stores to +0x1eec ==` 段只列出 `FUN_00402720`）。

calc 末尾无条件清零，所以绘制要读到非零，写入必须发生在同一帧的 calc 之后、draw 之前。这一点与 m4_render_re §4.3 一致。

### 1.2 非零写入者的搜索范围（均未找到）

1. 任意基址的存储 `[reg+0x1eec]`（运行 1，`--writes 0x1eec`，覆盖整个 listing，包括不属于函数的指令）：只有 0x40282b。
2. 任意形式含立即数 0x1eec 的指令（LEA、ADD、PUSH、内存操作数；运行 2 `--needle 0x1eec`）：仅 4 条：0x4027cf（读）、0x40282b（清零）、0x402c44（读）、0x402f37（读）。
3. 相邻偏移的 LEA/ADD（运行 3，`--needle-list 0x1ee0 0x1ee4 0x1ee8 0x1eea 0x1eeb 0x1eed 0x1ef0`）：没有 LEA 或 ADD。读取只有 `0x403ab4`（`MOV ECX,[EBP+0x1ee0]`，`FUN_00403a30`）。存储只有：ctor `0x40235d`（`+0x1ee0 = 0x4b12a310`）；calc `0x4027a0`（`+0x1ee8 = 0x808080`）；draw `0x402fc5`（`+0x1ee8 = 0xffffff`）；draw `0x402a89`、`0x402e99`、`0x402faa`（`+0x1eeb` 字节，值 0 / 0xff）；draw `0x402c2e`（`+0x1ee4`）。没有一条覆盖 0x1eec（字节 0x1eec–0x1eef）。
4. 块复制（`REP MOVS/STOS`）。STD 相关的全部检查，目标都不覆盖 0x1eec：
   - ctor `FUN_00402230` 0x4022a0：`DAT_00491d7c` → `this+0x2a4c`（0x46 dword，`render/func_402230.c`）。
   - ctor `FUN_00402160` 0x4021fe：清零整个对象（`s2c/func_402160.c`）。
   - calc 0x402835、draw 0x4028b4、0x402d00：`this+0x2a4c` → `DAT_00491d7c`（0x46 dword；`r6/func_402850.asm`、`r6/func_402ca0.asm`）。
   - 脚本 `FUN_00403c80` 0x403f62、0x403f76、0x404339：目标为 `this+0xe8`、`this+0x104`、`this+0x2b48`（各 7 dword；`render/func_403c80.asm`）。
   - `FUN_00403850`（STD 文件装载，ctor 调用）：目标是 `malloc` 缓冲（`this+0x10`）与全局 `DAT_00497c38`（`std2/func_403850.c`）。
   - `FUN_004049a0` 0x404a71–0x404c40：写调用者提供的记录缓冲（`EBP`、`EBX`、局部；`render7/func_4049a0.asm`）。
5. 范围之外：Ghidra 未反汇编的代码（例如只经间接调用到达、未被识别的函数）不在 listing 中，上面的扫描看不到。

**（推测）** 可能的解释：(a) 写入藏在未反汇编的代码中；(b) 写入经计算地址（基址寄存器加变量偏移，立即数不是 0x1eec，也不在 0x1ee0–0x1ef0 的 LEA 中）；(c) 原作中该分支不可达（只有清零），此时三个特殊 VM 也从未启动（§4），整段是死代码。要区分三者，需要运行时对照或扩大反汇编范围，本次不做。

---

## 2. STD+0x2a18 标志位

### 2.1 写者（运行 1 的 `_writes.txt` 给出的完整列表，并核对了每一处的条件）

| 地址 | 函数 | 指令 | 写入 | 条件 | 节点 / 调用链 |
|---|---|---|---|---|---|
| 0x40241e | `FUN_00402230`（STD 构造） | `|= 1`（`render/func_402230.c`） | bit0 置位 | 无（构造时） | 工厂 `FUN_00402640`（mode 0）← `FUN_00417870` 的两个分支（`emgr/func_417870.c:108`、`:124`）← 线程 `0x417c70`（§5） |
| 0x402a77 | `FUN_00402850`（draw 7，节点 0x403060，优先级 7） | `ECX = flags|1`，写回 | bit0 置位 | `flags&4` 且 cur<0x1e（0x402a4c `CMP [2a20],0x1e`、0x402a53 `JGE`） | 同一分支还调用 `FUN_0043c8b0(3,0x1e,…)`（0x402a61，淡入）与 `FUN_00405410(1)`（0x402a7d，计时器置 1，§3）。draw 链（m4_render_re §1.4） |
| 0x402a8f | `FUN_00402850` | `EAX = flags & 0xfffffffe` | bit0 清零 | `flags&4` 且 cur≥0x1e；同时 `+0x1eeb = 0`（0x402a89） | 同上 |
| 0x402fb6 | `FUN_00402ca0`（draw 0xa，节点 0x403070，优先级 0xa） | `|= 8` | bit3 置位 | `flags&2`（0x402faa `TEST AL,2`）且计时 tick 后 cur<1（0x402f9a） | 同上；同一分支 `+0x1eeb = 0xff`（0x402faa） |
| 0x402fcf | `FUN_00402ca0` | `flags & 0xfffffff9`（清 bit1 与 bit2） | bit1、bit2 清零 | 同上（cur<1） | 同上；同一分支 `+0x1ee8 = 0xffffff`（0x402fc5） |
| 0x40459f | `FUN_00404530` | `OR [ESI+0x2a18],2` | bit1 置位 | 关卡时钟 +0x14 == 0（首帧）且 STD2 非空（0x4181e8–0x4181f0） | `FUN_00418190` 0x4181f2（ESI = STD2）←节点回调 0x4187c0（§5） |
| 0x404604 | `FUN_004045b0` | `OR [EAX+0x2a18],4`（EAX = 主 STD，0x4181f7） | bit2 置位 | 同上（STD2 非空） | `FUN_00418190` 0x4181fc（§5） |
| 0x408d7f | `FUN_00408d60`（bomb） | `AND [EAX+0x2a18],0xfffffffe`（EAX = DAT_004776e8） | bit0 清零 | 0x378c bit0 置位（符卡进行中，入口检查）且 `[0x378c 所在块+0x3738] > 0x3b` | `FUN_00409220` 每帧调用（bomb calc 节点 0x409220，优先级 0x16，`render6/func_4089c0.asm:1–4`）；入口见 §5 |
| 0x409c18 | `FUN_00409c00`（符卡结束） | `OR [EAX+0x2a18],1`（EAX = DAT_004776e8） | bit0 置位 | 0x378c bit0 置位（进入即检查，`bmgr` 块 `+0x378c`） | ECL 指令 0x157（343）：`round2/func_40e770.c:982–983`，调用点 0x410f5b（§5） |

没有其他写入：`_writes.txt` 中 `+0x2a18` 的全部存储都在上表。因此 bit4 及以上从不被写入。

### 2.2 读者（简表，供核对语义）

- calc `FUN_00402720`：0x402726 读 `flags`；bit3 置位则跳过 calc；bit2 置位时只在 cur<0x3c 时运行（`render/func_402720.c`）。
- draw 7、draw 0xa：早退 0x402858、0x402ca5（bit3）；0x4029a2、0x402a48（TEST 4）；0x402aba（TEST 1）与 0x402ad3 `JZ 0x402c34`（bit0 清零时跳过 pass 0–7，即 0x402bd4–0x402c0d 的调用段）；0x402faa（TEST 2）；pass 8–11 的门控为 bit0（`r6/func_402ca0.c:64`）。
- `FUN_00418190` 0x41862c `TEST [ESI+0x2a18],8`（ESI = STD2）：置位时释放 STD2（`FUN_00402440`；`s2c/func_418190.c:284–287`）。

### 2.3 语义（推测，未经游戏行为对照）

- bit0 = 背景可见（pass 0–11 的门控）。bit4 段内 cur≥0x1e 时清 bit0，cur<0x1e 时置 bit0。
- bit2（值 4）= 主 STD 的入场段（60 帧，前约 30 帧背景隐藏，之后淡入显示）。
- bit1（值 2）= STD2 的退场段（30 帧）；到期时置 bit3。
- bit3（值 8）= 退场完成，允许释放 STD2。

---

## 3. STD+0x2a1c 计时器（ZunTimer）

### 3.1 布局（事实，多处互证）

| 偏移 | 字段 | 证据 |
|---|---|---|
| +0x2a1c | prev（int） | `FUN_0044bf40` 首条语句 `*unaff_ESI = unaff_ESI[1]`（`s2b_aux/func_44bf40.c`） |
| +0x2a20 | **cur（int，当前值）** | `FUN_0044bf40` 末尾 `unaff_ESI[1] = trunc(acc)`；draw 中 `CMP [2a20], 0x1e/0x3c/0`（`r6/func_402850.c:33,103`、`r6/func_402ca0.c:16,61`） |
| +0x2a24 | acc（float） | `FUN_0044bf40`：`unaff_ESI[2] += n·s`（`s2b_aux/func_44bf40.c`） |
| +0x2a28 | 速度指针（→ `DAT_00476f78`，即 1.0） | `FUN_0044bf40` 读 `unaff_ESI[3]`；`FUN_00404530` 0x404571 写入 `&DAT_00476f78` |
| +0x2a2c | flags（bit0 = 已初始化） | `FUN_00405410`（`in_EAX[4]&1`，`bomb/func_405410.c`）；`FUN_00402160` 0x4021ef 清 bit0 |

独立互证：STD 脚本 op 1 用同样的布局（prev +0x38、cur +0x3c、acc +0x40、速度 +0x44、flags +0x48，`render/func_403c80.c:33–48`），ctor 也初始化 +0x38…+0x48（`render/func_402230.c`）。faith 块（`faith.md` §1）是同一结构。因此 **+0x2a20 确为当前值**（cur）。

### 3.2 Set / Reset / Tick 站点

| 操作 | 地址 | 函数 | 内容 | 条件 / 调用链 |
|---|---|---|---|---|
| Reset（初始化） | 0x404553–0x40457b | `FUN_00404530` | flags bit0 为 0 时：prev=0xfff0bdc1（即 −999999）、cur=0、acc=0、速度=&DAT_00476f78、flags|=1 | STD2，首帧（§5） |
| Reset（初始化） | 0x4045c0–0x4045e0 | `FUN_004045b0` | 同上 | 主 STD，首帧（§5） |
| Set(30) | 0x404581（cur=0x1e）、0x40458b（acc=0x41f00000 = 30.0）、0x404595（prev=0x1d） | `FUN_00404530` | | STD2，首帧 |
| Set(60) | 0x4045e6（cur=0x3c）、0x4045f0（acc=0x42700000 = 60.0）、0x4045fa（prev=0x3b） | `FUN_004045b0` | | 主 STD，首帧 |
| Set(1) | 0x402a71 `LEA EAX,[EBP+0x2a1c]`，0x402a6f `PUSH 1`，0x402a7d `CALL 0x405410` | `FUN_00405410`（`bomb/func_405410.c`） | prev=0、cur=1、acc=1.0 | draw 7，`flags&4` 且 cur<0x1e |
| Tick(−1.0) | 0x402f8a `PUSH 0xbf800000`、0x402f8f `LEA ESI,[EBP+0x2a1c]`、0x402f95 `CALL 0x44bf40` | `FUN_0044bf40`（经 `FUN_00402ca0`） | prev=cur，acc −= 速度，cur=trunc(acc) | draw 0xa；门控 0x402f82 `cur > 0`，且 flags&8 为 0（`r6/func_402ca0.c:13` 早退）。**STD 计时器的 tick 仅此一处** |

要点（事实）：STD 计时器 +0x2a1c 的唯一 tick 站点是 draw 0xa（`FUN_00402ca0` 0x402f95，LEA 与 CALL 同处）；calc `FUN_00402720` 不直接调用 `FUN_0044bf40`。`FUN_0044bf40` 还被 VM 步进函数 `FUN_0043ee30`（`render/func_43ee30.c:1628`）调用，但那作用于 VM 自身的计时器，与 +0x2a1c 无关。因此该计时器与绘制次数相同推进。
**（推测）** 若启用了跳帧（m4_render_re §1.4 的 `this+0x14`），计时也会随之变慢。

---

## 4. 三个特殊 VM（+0x1f08 / +0x22b4 / +0x2660）

- **构造**：`FUN_00402160`（STD 的 VM 部分，`s2c/func_402160.c`）`_eh_vector_constructor_iterator_(this+0x1f08, 0x3ac, 3, FUN_00402050, FUN_00401ff0)`，LEA 在 0x4021db。调用者：`FUN_00402640` 0x402675（`render/func_402640.asm`）。
- **析构**：`FUN_00402440` 0x4025ed `_eh_vector_destructor_iterator_(this+0x1f08, 0x3ac, 3, FUN_00401ff0)`（`render3/func_402440.c:74`；`s2c/func_402440.c`）。另有 SEH 解栈 `Unwind@0x4652bb`（0x4652ca `ADD EAX,0x1f08`，`s2c/func_4652bb.c`）。
- **步进**：`FUN_00402720` 0x4027df（`+0x1f08`）、0x4027f9（`+0x22b4`）、0x402805（`+0x2660`），仅 1eec≠0 时调用 `FUN_0043ee30`（`render/func_402720.c:43–47`）。
- **启动（脚本号、活动位）**：**未找到**。运行 2 的 `--needle` 显示，全程序中含 0x1f08、0x22b4、0x2660 的指令只有上面的 LEA、析构、解栈与 calc 步进；没有任何代码把这三个地址传给 VM 初始化函数（`FUN_00404f30`、`FUN_00401de0` 等）。
- 槽 VM（STD 脚本 op 14）按 `基址 + idx·0x3ac` 寻址（m4_render_re §4.3，基址 0x180，idx = op+8）。0x1f08 − 0x180 = 0x1d88，不是 0x3ac 的整数倍，因此 op 14 无法寻址三个特殊 VM（**（推测）** 基址取 0x180 时成立）。

**（推测）** 按静态证据，三个特殊 VM 没有启动路径。即使 1eec 非零，它们也只会步进未初始化的 VM。这与 §1 合起来，说明该特殊路径在 v1.00a 中可能不生效。这是待验证的假设，本次不运行游戏。

---

## 5. 调用链（上溯，附证据）

**每帧调度**：calc 链 `FUN_00449c00`、draw 链 `FUN_00449d40`（m4_render_re §1.4）。相关节点：STD calc 0xc（0x403050）；bomb calc 0x16（0x409220）；STD draw 7（0x403060）与 0xa（0x403070）；关卡节点回调 0x4187c0（优先级未核，见 §6）。

**关卡开始（首帧）**：

1. 节点回调 `FUN_004187c0`（`s2c/func_4187c0.c`：`FUN_00418190(param, param)`）。该回调由 `FUN_00417870` 注册（`emgr/func_417870.c:83`：`FUN_00449ed0(&LAB_004187c0)`）。
2. `FUN_00418190`（`s2c/func_418190.c`）：关卡时钟 `+0x14 == 0` 时（0x41819b–0x4181a2），`FUN_00404450(主 STD)`（0x4181e3）。若 `DAT_004776e4 != NULL`（0x4181e8–0x4181f0），再调用 `FUN_00404530(STD2)`（0x4181f2）与 `FUN_004045b0(主 STD)`（0x4181fc）。
3. STD2 非空的条件：`FUN_00417c80`（关卡释放）在 `DAT_00474ca0 & 2` 时执行 `DAT_004776e4 = DAT_004776e8`（`s2c/func_417c80.c` else 分支；`DAT_00474ca0` bit1 由 `DAT_00491fb8 == 0xb` 时置位，`bmgr/func_417c80.c:52–53`）。因此 STD2 是上一关保留下来的主 STD。
4. `FUN_00417870`（关卡对象创建）两个分支都只调用 `FUN_00402640(…, 0)`（`emgr/func_417870.c:108`、`:124`），即只创建主 STD。mode 1 的构造（会把 `DAT_004776e4` 设为新对象）在 dump 中没有调用者。
5. 线程：`FUN_004180e0` 新建 0x60 字节的关卡对象（`DAT_00477810`），并用 `FUN_00420ea0(&LAB_00417c70)` 起线程（`s2c/func_4180e0.c`）。线程体 `FUN_00417c70` = `FUN_00417870(DAT_00477810)`（`s2c/func_417c70.c`）。
6. 场景状态机 `FUN_004218d0` 调用 `FUN_004180e0`（0x421a14、0x421a44、0x421a76、0x421abd、0x421ae8）与释放函数 `FUN_00418150`（0x4219dd、0x421a66、0x421a86、0x421acd，`s2c/_callers.txt`）。`FUN_00418150` 调用 `FUN_00417c80`（`s2c/func_418150.c`）。
7. 状态机由 supervisor calc `FUN_0041ff80` 驱动（`systems_gaps.md`:133）。

**符卡（bomb 块 `DAT_004776f4`，`+0x378c`）**：

- ECL 指令 357 置 `+0x378c` bit0（符卡进行中；`systems_gaps.md`:79，已有结论）。
- bomb calc 节点 0x409220（优先级 0x16）每帧调用 `FUN_00408d60`（`s2c/func_409220.c`）。节点由 `FUN_004089c0` 注册（`render6/func_4089c0.asm:1–4`；调用者 `FUN_00408c90` 0x408cd8，上层未追）。
- ECL 指令 343（ClearAll）调用 `FUN_00409c00`（符卡结束；`round2/func_40e770.c:982–983`，调用点 0x410f5b，`s2c/_callers.txt`）。`FUN_0040e770` 是 ECL 指令分发（`m2_bullets.md` §2）。

---

## 6. 未解决

1. **STD+0x1eec 的非零写入者**：§1.2 的范围内未找到。原因见 §1.2 末尾的三种（推测）解释。
2. **三个特殊 VM 的启动点**：§4 未找到。
3. **节点 0x4187c0 的优先级**：`FUN_00417870` 中 `FUN_00449ae0` 之前的参数未核。因此“首帧 `FUN_00418190` 与 STD calc（0xc）的先后”未定。
4. **场景码 0xb 的含义**：推测对应 `FUN_004218d0` 的 `case 0xb`（`systems_gaps.md`:133–134），未核。
5. **bit 语义**（§2.3）未经游戏行为对照，全部为（推测）。
6. **跳帧对计时的影响**：未核（§3.2）。
7. **m4_render_re.md §4.2 的一处描述需要更正**：它写 `FUN_00418190` “由 `render/func_4180e0.c` 调用”。s2c 证据（`s2c/func_4180e0.c`）显示 `FUN_004180e0` 只起线程，真正的每帧调用者是节点回调 `FUN_004187c0`（§5）。本任务不改该文件，由文档维护者更新。

---

## 7. 运行记录与文件清单

运行（共 3 次，未超过 4 次上限；均使用 `--project "$LOCALAPPDATA/th10-ghidra-render" --exe ".tmp/[th10]/th10.exe"`）：

- 运行 1（`tools/ghidra/dump.py`）：`--writes 0x1eec --writes 0x2a18 --writes 0x2a1c --writes 0x2a20 --writes 0x2a24 --writes 0x2a28 --writes 0x2a2c --writes 0x1f08 --writes 0x22b4 --writes 0x2660 --ref-funcs 0x4776e8 --callers 0x404530 --callers 0x4045b0 --callers 0x408d60 --context 6 --asm --tag s2c` → `.tmp/ecl_dump/s2c_run1.log`。`--ref-funcs 0x4776e8` 只命中 6 个函数（ctor 0x402230、dtor 0x402440、0x408d60、0x409c00、0x417c80、0x418190）。
- 运行 2（`s2c/dump_s2c.py`，即 `tools/ghidra/dump.py` 的副本，增加了 `--needle` 选项）：`--needle 0x1eec 0x1f08 0x22b4 0x2660 0x2a18 0x2a1c 0x2a2c --callers 0x418190 0x409c00 0x402160 0x409220 0x404450 0x4180e0 --context 6 --asm 0x409220 0x402160 0x4180e0 0x404450` → `.tmp/ecl_dump/s2c_run2.log`（命中 12 个函数）。
- 运行 3（同上副本，增加 `--needle-list`）：`--needle-list 0x1ee0 0x1ee4 0x1ee8 0x1eea 0x1eeb 0x1eed 0x1ef0 REP --callers 0x418150 0x417c70 0x4089c0 --context 6 --asm 0x4187c0 0x418150 0x417c70 0x4089c0` → `.tmp/ecl_dump/s2c_run3.log`。

文件（`.tmp/ecl_dump/s2c/`）：

- `_writes.txt`：运行 1 的存储扫描（`+0x1eec`、`+0x2a18`、`+0x2a1c`、`+0x2a20`、`+0x2a24`、`+0x2a28`、`+0x2a2c`、`+0x1f08`、`+0x22b4`、`+0x2660`）。
- `_callers.txt`：运行 1–3 的调用者（追加）。
- `_needle.txt`：运行 3 的 L 行（REP 与相邻偏移）。运行 2 的 D 行曾被它覆盖。
- `_needle_run2_rebuilt.txt`：由 12 个命中函数的 `.asm` 按相同立即数规则重建的运行 2 命中列表（42 行）。
- `dump_s2c.py`：`tools/ghidra/dump.py` 的副本（仅增加 `--needle` 与 `--needle-list`；原脚本未改）。
- `func_*.c` 与 `func_*.asm`：402160、402230、402440、402720、402850、402ca0、404450、404530、4045b0、408d60、409220、409c00、417c70、417c80、418150、418190、4180e0、4187c0、4089c0、4652bb。
- 运行日志：`.tmp/ecl_dump/s2c_run1.log`、`s2c_run2.log`、`s2c_run3.log`（在 `s2c/` 之外，按任务要求）。

外部引用（既有 dump，未重跑）：`render/`、`r6/`、`render3/`、`render6/`、`render7/`、`std2/`、`emgr/`、`bmgr/`、`round2/`、`bomb/`、`s2b_aux/`、`faith/`。

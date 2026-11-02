# M4 S2b 取证：op 9 雾插值的基本运算 FUN_00404e10 / FUN_00404d40 / FUN_00404da0，及 FUN_004049a0 的调用点（2026-11-01）

依据 th10.exe v1.00a 汇编（`tools/ghidra/dump.py`，项目 `%LOCALAPPDATA%/th10-ghidra-render`）。新增 dump：`.tmp/ecl_dump/s2b/`（三个函数的 `.c`/`.asm` 与 `_callers.txt`）；另有 `.tmp/ecl_dump/s2b_aux/func_44c350.{c,asm}`，只用于确认缓动函数的调用约定。前置文档：`m4_r7.md`（D.2 雾字段、D.3 `_ftol2`）、`m4_render.md`（S2b 条目）、`m2_motion.md`（缓动表）。

约定：

- 地址写作 `0x……`。
- 寄存器与栈参数以 `.asm` 为准。`FUN_004049a0` 入口 ESP 记为 `S0`，序言 `SUB ESP,0xd4` 加三次 `PUSH` 后，函数体基准 ESP 记为 `E = S0 − 0xe0`。正文的 `E+x` 为该基准下的偏移，中途 `PUSH` 之后的 `[ESP+x]` 按 push 数换算，表中同时给出 `E+x`。
- “舍入”指 `FST`/`FSTP` 写 `float ptr` 时取整到 float32。寄存器里的 x87 值在两次舍入之间保持未舍入。
- “（推测）”标注由结构推出、未逐条核对的内容。
- `.c` 与 `.asm` 不一致处以 `.asm` 为准，见第 5.5 节。

---

## 0. 结论速览

| 问题 | 结论 |
|---|---|
| FUN_00404e10 | 6 个 float 的逐分量加法 `out = EAX + ECX`。寄存器传参（EAX、ECX 为输入，ESI 为输出），`RET` 无清栈，返回 `EAX = ESI`。输出 28 字节：6 个 float32 加 4 个字节。 |
| FUN_00404d40 | 逐分量减法 `out = ECX − EAX`。传参与输出格式同上，返回 `EAX = ESI`。 |
| FUN_00404da0 | 标量乘 `out = s × EAX[]`，`s` 经栈传入（stdcall，`RET 4`），返回 `EAX = ESI`。 |
| 舍入规律 | 三者都先在 x87 中算出未舍入结果。6 个 float 分量经 `FST/FSTP` 舍入到 float32，4 个字节则由**未舍入**结果经 `_ftol2`（`FUN_00463b2c`）截断得到。 |
| 浮点常量 | 三个函数内部没有浮点常量。 |
| 调用点 | `FUN_004049a0` 调用 `FUN_00404e10` 共 6 处、`FUN_00404da0` 共 5 处。任务清单漏了 `0x404c32`（`FUN_00404e10`）与 `0x404c25`（`FUN_00404da0`），以 `_callers.txt` 为准，见第 5.3 节。 |
| op 9 每帧 | `FUN_004049a0` 按 `counter`、`elapsed`、`mode` 生成 7 dword 的雾状态，按 `mode` 分四条路径：7（线性累加）、0x11（二阶累加）、8（三次 Hermite）、其它（缓动线性插值）。 |

---

## 1. 记录格式与 x87 舍入模型

本文的“记录”（record）为 28 字节，即 7 个 dword：

| 偏移 | 内容 |
|---|---|
| +0x00 .. +0x14 | 6 个 float32，下标 k = 0..5 |
| +0x18 .. +0x1b | 4 个字节 B0..B3，依次为分量 2、3、4、5 的截断值（`_ftol2` 结果的低 8 位） |

`FUN_00463b2c` 是 MSVC x87 `_ftol2`（`m4_r7.md` D.3 已核对）。它对 ST0 截断到零得到 int64，返回 `EDX:EAX`，调用方只取 `AL`。它**弹出** ST0。该函数对 `FST` 后的值符号判断用的是 float32 副本（`0x463b35–0x463b37`），对普通数值与截断等价（推测）。

x87 精度控制字：本文三个函数和本文涉及的调用方中没有 `FLDCW`。按 Windows 默认 53 位精度处理（推测）。在该假定下：

- float32 × float32 的乘积精确（有效位不超过 48 位），`FMUL` 的结果与 float64 乘法相同。
- float32 ± float32 的和或差在指数差不超过 29 时精确。更大时先舍入到 53 位、再由 `FST` 舍入到 24 位。这与 Go 中 `float32(float64(a) + float64(b))` 的两次舍入路径一致。

---

## 2. FUN_00404e10（加法）

### 2.1 调用约定（`.asm` 0x404e10–0x404e6b）

- 输入：`EAX` = 第一个记录 a（读 `[EAX+0..0x14]`），`ECX` = 第二个记录 b（读 `[ECX+0..0x14]`）。
- 输出：`ESI` = 输出指针，写 `[ESI+0..0x1b]`，共 28 字节。
- 返回：`EAX = ESI`，`RET`（无立即数，无栈参数）。
- 栈：只通过 `CALL 0x463b2c` 使用，ESP 平衡。返回时 x87 栈深度与入口相同。
- 寄存器：`ESI`、`EBX`、`EDI`、`EBP` 保持；`ECX`、`EDX` 被 `_ftol2` 改写；`EAX` 被设为 `ESI`。

### 2.2 计算步骤

第 1–12 步：6 次 `FLD [EAX+4k]; FADD [ECX+4k]`，在 x87 栈上得到 `s_k = a_k + b_k`（未舍入）。栈顶为 s0。

第 13–21 步（栈顶在左）：

| 指令 | 作用 | 栈 |
|---|---|---|
| `FSTP [ESI]` | out0 = f32(s0)，弹出 | s1 s2 s3 s4 s5 |
| `FSTP [ESI+4]` | out1 = f32(s1)，弹出 | s2 s3 s4 s5 |
| `FST [ESI+8]` | out2 = f32(s2)，不弹出 | s2 s3 s4 s5 |
| `FLD ST1` / `FSTP [ESI+0xc]` | out3 = f32(s3) | s2 s3 s4 s5 |
| `FLD ST2` / `FSTP [ESI+0x10]` | out4 = f32(s4) | s2 s3 s4 s5 |
| `FLD ST3` / `FSTP [ESI+0x14]` | out5 = f32(s5) | s2 s3 s4 s5 |

第 22–25 步：4 次 `CALL 0x463b2c`，无参数。

- 第 1 次弹出 ST0 = s2（**未舍入**的值），低字节写 `[ESI+0x18]`。
- 第 2、3、4 次依次弹出 s3、s4、s5，低字节写 `[ESI+0x19]`、`[ESI+0x1a]`、`[ESI+0x1b]`。
- 返回时 x87 栈为空。

要点：字节取自未舍入的 s2..s5。例如 s2 = 2.99999997（未舍入）时字节为 2，而 out2 = f32(s2) 为 3.0。

### 2.3 常量

无。

### 2.4 读写与调用

- 读：`[EAX+0..0x14]`、`[ECX+0..0x14]`。
- 写：`[ESI+0..0x1b]`。
- 调用：`FUN_00463b2c` 共 4 次。
- 寄存器副作用：`EAX`（返回 `ESI`）、`ECX`、`EDX`。

### 2.5 Go 风格伪代码

```go
// rec7 为 28 字节记录：6 个 float32 加 4 个字节。
type rec7 struct {
    F [6]float32
    B [4]byte
}

// FUN_00404e10：out = a + b。a 对应 EAX，b 对应 ECX，输出对应 ESI。
func add7(a, b *rec7) rec7 {
    var s [6]float64
    var out rec7
    for k := range s {
        s[k] = float64(a.F[k]) + float64(b.F[k]) // FLD [EAX+4k]; FADD [ECX+4k]，未舍入
    }
    for k := range s {
        out.F[k] = float32(s[k]) // FSTP/FST float ptr，舍入到 float32
    }
    for k := 0; k < 4; k++ {
        out.B[k] = byte(int64(s[k+2])) // _ftol2(未舍入值) 的低 8 位，截断到零
    }
    return out
}
```

注：Go 中 `int64(x)` 对越界或 NaN 的结果由实现定义。`_ftol2` 对这类输入返回 `0x8000000000000000`，低字节为 0。在 amd64 上 `CVTTSD2SQ` 也给出同样的值（推测）。

---

## 3. FUN_00404d40（减法）

### 3.1 调用约定（`.asm` 0x404d40–0x404d9b）

- 输入：`ECX` = 被减数 a（读 `[ECX+..]`），`EAX` = 减数 b（读 `[EAX+..]`）。
- 输出：`ESI`，写 `[ESI+0..0x1b]`。
- 返回：`EAX = ESI`，`RET`（无栈参数）。
- 寄存器副作用与 FUN_00404e10 相同。

### 3.2 计算步骤

第 1–12 步：`FLD [ECX+4k]; FSUB [EAX+4k]`，得到 `s_k = ECX_k − EAX_k`（未舍入）。

第 13–25 步与 FUN_00404e10 的 `0x404e32–0x404e69` 逐条对应（`0x404d62–0x404d9b`），舍入规则与字节截断相同。

### 3.3 常量、读写与调用

- 无浮点常量。
- 读：`[ECX+..]`、`[EAX+..]`。写：`[ESI+0..0x1b]`。
- 调用：`FUN_00463b2c` 共 4 次。

### 3.4 Go 风格伪代码

```go
// FUN_00404d40：out = ecx − eax。ecx 为 ECX（被减数），eax 为 EAX（减数）。
func sub7(ecx, eax *rec7) rec7 {
    var s [6]float64
    var out rec7
    for k := range s {
        s[k] = float64(ecx.F[k]) - float64(eax.F[k]) // FLD [ECX+4k]; FSUB [EAX+4k]
    }
    for k := range s {
        out.F[k] = float32(s[k])
    }
    for k := 0; k < 4; k++ {
        out.B[k] = byte(int64(s[k+2]))
    }
    return out
}
```

---

## 4. FUN_00404da0（标量乘）

### 4.1 调用约定（`.asm` 0x404da0–0x404e02）

- 栈参数：入口时 `[ESP+4]` = 标量 s（float32）。其读取方式为 `FLD float ptr [ESP+4]`。
- 输入：`EAX` = 记录 a。
- 输出：`ESI`，写 `[ESI+0..0x1b]`。
- 返回：`EAX = ESI`，`RET 0x4`，由被调用方清掉 s 所占的 4 字节。
- 寄存器副作用：`ECX`、`EDX` 被 `_ftol2` 改写。

### 4.2 计算步骤

第 1–12 步：6 次 `FLD [ESP+4]; FMUL [EAX+4k]`，得到 `p_k = s × a_k`（乘积精确，见第 1 节）。

第 13–25 步：与 FUN_00404e10 的存储与截断顺序完全相同（`0x404dc9–0x404dfd`）。
- 6 个 float 为 f32(p_k)。
- 4 个字节为 `_ftol2(p_2..p_5)` 的低字节，取未舍入的乘积。

### 4.3 常量、读写与调用

- 无浮点常量。
- 读：`[ESP+4]`、`[EAX+..]`。写：`[ESI+0..0x1b]`。
- 调用：`FUN_00463b2c` 共 4 次。

### 4.4 Go 风格伪代码

```go
// FUN_00404da0：out = s * a。s 来自栈 [ESP+4]，a 来自 EAX，输出对应 ESI。
func scale7(s float32, a *rec7) rec7 {
    var p [6]float64
    var out rec7
    for k := range p {
        // FLD [ESP+4]; FMUL [EAX+4k]。float32 × float32 的乘积精确。
        p[k] = float64(s) * float64(a.F[k])
    }
    for k := range p {
        out.F[k] = float32(p[k])
    }
    for k := 0; k < 4; k++ {
        out.B[k] = byte(int64(p[k+2]))
    }
    return out
}
```

---

## 5. FUN_004049a0 中的调用（op 9 每帧）

### 5.1 状态块（EBX 指向 STD+0xe8）

调用方 `FUN_00403c80`（`render7/func_403c80.asm` 0x404317–0x40433b）：

- `LEA ECX,[ESP+0x80]; PUSH ECX`：输出指针。
- `LEA EBX,[EBP+0xe8]`：状态块。
- `CALL 0x4049a0`，返回后把 `EAX` 指向的 7 个 dword 复制到 `STD+0x2b48`（`0x40432c–0x404339`）。
- 调用门控：`[STD+0x16c] != 0` 才调用（`0x40430f–0x404317`）。

状态块字段（EBX 偏移 / STD 偏移）：

| EBX | STD | 名称（推测） | 说明 |
|---|---|---|---|
| +0x00 | +0xe8 | P | 7 dword 记录，“起点”（推测） |
| +0x1c | +0x104 | Q | 7 dword 记录，“终点”（推测） |
| +0x38 | +0x120 | R1 | 7 dword 记录，切线 0（推测）。op 9 不写入 |
| +0x54 | +0x13c | R2 | 7 dword 记录，切线 1（推测）。op 9 不写入 |
| +0x70 | +0x158 | prev | 整数。本函数内只写不读 |
| +0x74 | +0x15c | counter | 整数，帧计数 |
| +0x78 | +0x160 | elapsed | float32，累计时间（帧） |
| +0x7c | +0x164 | step | 指向 float32 的指针，每次调用的步长 |
| +0x80 | +0x168 | flags | bit0 = 已初始化 |
| +0x84 | +0x16c | N | 整数，持续帧数。完成后被置 0 |
| +0x88 | +0x170 | mode | 整数，插值模式（来自 op+0xc） |

### 5.2 每次调用的流程（Go 风格伪代码）

本段由 `.asm` 0x4049a0–0x404c4d 逐条整理。三个基本运算以第 2–4 节的函数表示。

```go
// tween 为 EBX 指向的状态块。
type tween struct {
    P, Q, R1, R2 rec7
    Prev         int32    // +0x70
    Counter      int32    // +0x74
    Elapsed      float32  // +0x78
    Step         *float32 // +0x7c
    Flags        uint32   // +0x80
    N            int32    // +0x84
    Mode         int32    // +0x88
}

var defaultStep float32 = 1.0 // DAT_00476f78

// fogTween 对应 FUN_004049a0（stdcall RET 4）。返回值 = 输出记录（EAX 指向的 7 dword）。
func fogTween(s *tween) rec7 {
    if s.N > 0 { // 0x4049be JLE：N <= 0 时跳过计数，直接进入进行中分支
        s.Prev = s.Counter // 0x4049c4–0x4049ca

        step := *s.Step
        if !(step > 0.99 && step < 1.01) { // 0x4049cd–0x404a0a；NaN 也走此分支
            sum := float64(step) + float64(s.Elapsed) // FLD [ECX]; FADD [EBX+0x78]
            s.Elapsed = float32(sum)                  // FST [EBX+0x78]（保留 ST0 未舍入）
            s.Counter = int32(int64(sum))             // _ftol2 的低 32 位
        } else { // 0x4049eb–0x4049f8
            s.Counter++                                 // EDX = 旧值 + 1
            s.Elapsed = float32(float64(s.Elapsed) + 1) // FADD [0x470afc]=1.0; FSTP
        }

        if s.Counter >= s.N { // 0x404a13–0x404a1a：JL 不跳，进入完成分支
            if s.Flags&1 == 0 { // 0x404a22–0x404a41
                s.Counter = 0
                s.Prev = -999999 // 0xfff0bdc1
                s.Elapsed = 0
                s.Step = &defaultStep // 0x404a3a
                s.Flags |= 1
            }
            s.Counter = s.N                   // 0x404a4b
            s.Prev = s.N - 1                  // 0x404a4e–0x404a4f
            s.Elapsed = float32(float64(s.N)) // FILD N; FSTP  (0x404a47–0x404a52)
            s.N = 0                           // 0x404a62：MOV [EBX+0x84], ESI(=0)
            if s.Mode == 7 {                  // 0x404a6c
                return s.P // 0x404a7f：复制 EBX（P）
            }
            return s.Q // 0x404a6e：复制 EBX+0x1c（Q）
        }
    }

    // 进行中分支（0x404a8f）
    switch s.Mode {
    case 7:
        s.P = add7(&s.Q, &s.P) // 0x404aaf：EAX=Q，ECX=P 的副本
        return s.P
    case 0x11:
        newP := add7(&s.R2, &s.P) // 0x404ae3：EAX=R2，ECX=P 副本
        s.R2 = add7(&s.Q, &s.R2)  // 0x404aff：EAX=Q，ECX=R2（使用旧 R2）
        s.P = newP
        return s.P
    case 8:
        return hermite8(s)
    default:
        return easeLerp(s)
    }
}

// 模式 8：三次 Hermite（0x404b14–0x404c02 中 mode==8 的分支）。
func hermite8(s *tween) rec7 {
    // FILD N; FDIVR [EBX+0x78]：t = elapsed / N，不舍入
    t := float64(s.Elapsed) / float64(s.N)
    // FLD [0x470afc]; FSUB ST0,ST1; FSTP float32 [E+0xc]：tm1 为 float32 舍入值
    tm1 := float64(float32(t - 1))

    // 权重（0x404b3e–0x404b7a）。3.0 = [0x470bd4]，1.0 = [0x470afc]。
    f0 := float32((1 + 2*t) * tm1 * tm1) // h00 = (1+2t)(t−1)²，t−1 取 float32 舍入值
    f1 := float32((3 - 2*t) * t * t)     // h01 = (3−2t)t²
    f2 := float32((1 - t) * (1 - t) * t) // h10 = t(1−t)²
    f3 := float32(tm1 * t * t)           // h11 = (t−1)t²，t−1 取 float32 舍入值

    // 四次乘法与三次加法（顺序与 0x404b88–0x404c32 一致）
    tmp1 := scale7(f3, &s.R2) // 0x404b88
    tmp2 := scale7(f2, &s.R1) // 0x404b9b
    tmp3 := scale7(f1, &s.Q)  // 0x404bb3
    tmp4 := scale7(f0, &s.P)  // 0x404bca
    tmp5 := add7(&tmp3, &tmp4) // 0x404bd9：EAX=tmp3，ECX=tmp4
    tmp6 := add7(&tmp2, &tmp5) // 0x404beb：EAX=tmp2，ECX=tmp5
    return add7(&tmp1, &tmp6)  // 0x404c32：EAX=tmp1，ECX=tmp6
}

// 其它模式（缓动插值，0x404bfa–0x404c39）。
func easeLerp(s *tween) rec7 {
    // FUN_0044c350：EAX = mode，栈 [ESP+4]=elapsed，[ESP+8]=float32(N)，stdcall RET 8；返回 ST0。
    // 0x404bfa–0x404c07：FSTP float32 [E+0x10]，即舍入到 float32。
    e := float32(ease(s.Mode, s.Elapsed, float32(s.N)))

    d := sub7(&s.Q, &s.P)  // 0x404c19：ECX=Q，EAX=P（EBX）
    sc := scale7(e, &d)    // 0x404c25：栈=e，EAX=d
    return add7(&s.P, &sc) // 0x404c32：EAX=P，ECX=sc
}
```

注意：

- 上面的 `ease(mode, a, b)` 对应 `FUN_0044c350`。其内部 `FDIV` 计算 `a/b`，之后的分支由模式决定（`m2_motion.md` 缓动表，本文未重新核对其 x87 舍入）。
- Go 中为避免编译器把 `x*y + z` 融合成 FMA，每一步都应显式对 float64 中间值求值（上面的表达式逐步左结合，与 x87 的逐步舍入一致）。
- 完成分支中 `s.N = 0` 之后，调用方的门控会阻止下一帧的调用（`0x40430f`）。因此 `N ≤ 0` 的进行中路径只在 `N` 为负（如 op+8 为负）时才可能出现。本文未见实例（推测）。

### 5.3 调用点表（`_callers.txt` 与 `.asm`）

`E+x` 为函数体基准 ESP 下的偏移（`PUSH` 之后已修正）。

| 地址 | 目标 | 分支 | 入参（寄存器 / 栈） | 结果去向 |
|---|---|---|---|---|
| 0x404aaf | `FUN_00404e10` | 模式 7 | EAX=&Q（+0x1c）；ECX=P 副本（`E+0x1c`，由 `0x404aa2` 复制）；ESI=`E+0x38` | `E+0x38` 结果复制回 P（`0x404abd`），输出 P |
| 0x404ae3 | `FUN_00404e10` | 模式 0x11 | EAX=&R2（+0x54）；ECX=P 副本（`E+0x1c`）；ESI=`E+0x38` | 复制回 P（`0x404aea`） |
| 0x404aff | `FUN_00404e10` | 模式 0x11 | EAX=&Q；ECX=&R2（`EDI`）；ESI=`E+0x38` | 复制回 R2（`0x404b04`） |
| 0x404b88 | `FUN_00404da0` | 模式 8 | 栈=f3（`0x404b82` `PUSH EAX`，EAX=`[E+0xc]`）；EAX=&R2；ESI=`E+0x38` | tmp1 = f3·R2 |
| 0x404b9b | `FUN_00404da0` | 模式 8 | 栈=f2（`0x404b93` `PUSH ECX`，ECX=`[E+0x18]`）；EAX=&R1（+0x38）；ESI=`E+0x1c` | tmp2 = f2·R1 |
| 0x404bb3 | `FUN_00404da0` | 模式 8 | 栈=f1（`0x404ba8` `PUSH EDX`，EDX=`[E+0x14]`）；EAX=&Q；ESI=`E+0xa8` | tmp3 = f1·Q |
| 0x404bca | `FUN_00404da0` | 模式 8 | 栈=f0（`0x404bc0` `PUSH ECX`，ECX=`[E+0x10]`）；EAX=EBX（&P）；ESI=`E+0xc4` | tmp4 = f0·P |
| 0x404bd9 | `FUN_00404e10` | 模式 8 | EAX=tmp3；ECX=tmp4；ESI=`E+0x54` | tmp5 = tmp3 + tmp4 |
| 0x404beb | `FUN_00404e10` | 模式 8 | EAX=tmp2（`[E+0x14]`，已被覆写为指针）；ECX=tmp5；ESI=`E+0x8c` | tmp6 = tmp2 + tmp5 |
| 0x404c19 | `FUN_00404d40` | 其它 | ECX=&Q；EAX=EBX（&P）；ESI=`E+0x70`（`PUSH EAX` 之后显示为 `ESI,[ESP+0x74]`） | D = Q − P，返回 `EAX=E+0x70` |
| 0x404c25 | `FUN_00404da0` | 其它 | 栈=e（`0x404c0f` `PUSH EAX`，EAX=`[E+0x10]`）；EAX=D（上一条返回值）；ESI=`E+0x8c` | S = e·D |
| 0x404c32 | `FUN_00404e10` | 模式 8 与其它共用 | 模式 8：EAX=tmp1、ECX=tmp6、ESI=`E+0x70`；其它：EAX=EBX（P）、ECX=S、ESI=`E+0x54` | 模式 8：H(t)；其它：P + e·D。随后 `0x404c39` 复制到输出 |

其它与 `FUN_004049a0` 相关的调用：

- `0x404c02`：`CALL FUN_0044c350`（缓动，stdcall RET 8）。入参：EAX=mode（`[EBX+0x88]`，未经修改），栈 `[ESP+4]` = elapsed（float32），`[ESP+8]` = float32(N)（由 `FSTP` 写在原 ECX=N 的位置）。返回 ST0，`0x404c07` 以 `FSTP float32 [E+0x10]` 舍入。
- `0x463b2c`（`_ftol2`）：`0x404a05`（非整数分支），`0x404a05` 对 `elapsed + step` 截断得到 counter。

### 5.4 op 9 的初始化（调用方一侧，供移植参考）

本段只说明状态怎样被设置，细节以汇编为准。

1. `FUN_00403c80` 0x403eef–0x403f3b：从 op 记录取 R、G、B、A 四字节（`op+0x10..0x13`），用 `FILD` 转为 float，作为 `FUN_00405040` 的 arg2..arg5。arg0 = `op+0x14` 的原始 dword，arg1 = `op+0x18` 的原始 dword。
2. `0x403f40`：`CALL FUN_00405040`（stdcall，`RET 0x18`）。其输出 ESI = `[ESP+0x80]`（调用方帧内）。
3. `0x403f45–0x403f48`：`N = op+8` 写入 `STD+0x16c`。
4. `0x403f4e–0x403f78`：`mode = op+0xc` 写入 `STD+0x170`。
5. `0x403f51–0x403f62`：`P（STD+0xe8）` ← `STD+0x2b48..0x2b60` 的旧雾状态（7 dword）。
6. `0x403f64–0x403f76`：`Q（STD+0x104）` ← `[ESP+0x80]`（即 `FUN_00405040` 的输出）。
7. `0x403f86–0x403f9f`：若 `flags` bit0 为 0，则写 `step = &DAT_00476f78`，`flags |= 1`。
8. `0x403faf–0x403fbb`：每次都执行。`counter = 0`，`elapsed = 0.0`，`prev = −1`。此处 `prev` 的写入覆盖了 `0x403f93` 的 `EBX` 写入，后者无效。
9. R1（+0x38）与 R2（+0x54）不被 op 9 写入。它们的初值来源未查（见第 6 节）。

`FUN_00405040`（`render7/func_405040.asm` 0x405040–0x40509b，stdcall `RET 0x18`）：

- `[ESI+0..0x14]` = 6 个参数的原始 dword（arg0..arg5）。
- `[ESI+0x18..0x1b]` = `_ftol2(arg2..arg5)` 的低字节（`0x405044–0x405096`）。因为 arg2..arg5 为整数值 R、G、B、A，所以字节即 R、G、B、A 的顺序（推测，见第 6 节第 7 项）。
- 因此记录的 7 个 dword 与 `STD+0x2b48..0x2b60` 的字段对应（`m4_r7.md` D.2）：`[near, far, R, G, B, A, 打包 DWORD]`（推测）。

### 5.5 .c 与 .asm 不一致之处（以 .asm 为准）

1. `FUN_00404e10` / `FUN_00404d40` 的 `.c` 把 s2..s5 声明为 float 局部量，并隐去了 `_ftol2` 的 x87 参数（ST0）。`.asm` 显示该参数是 `FST` 之后仍留在 ST 栈上的**未舍入**值（第 2.2 节）。
2. `FUN_004049a0` 模式 8 的 `.c` 写成 `fVar7 = [0x1e] / (float)N`，即 t 被舍入到 float32。`.asm`（`0x404b21`）是 `FDIVR`，t 不舍入。
3. 模式 8 的 `.c` 把 `1−t`、`t−1` 等写成 float 局部量。`.asm` 只对 `t−1` 做了一次 float32 舍入（`0x404b34`），`1−t` 与 `t` 保持扩展精度。
4. `FUN_0044c350` 调用的 `.c`（`FUN_0044c350(unaff_EBX[0x1e], (float)N)`）漏写了入参 EAX = mode。`.asm` 中 `0x404c02` 前 EAX = `[EBX+0x88]`，`0x44c350` 的 `DEC EAX; CMP EAX,0xf` 读取它。
5. 复位分支的 `.c` 写 `unaff_EBX[0x1c] = -NAN`。`.asm` 实际写的是 `MOV dword ptr [EBX+0x70],0xfff0bdc1`，即整数 −999999（`.c` 的 NaN 只是 float 视图）。
6. `.c` 把 `FUN_00463b2c` 的结果以 float 形式存入 `[0x74]`。`.asm` 写入的是 `EAX`（int64 的低 32 位，`0x404a0a`）。
7. `.c` 中的 `9.80909e-45`、`2.38221e-44`、`1.12104e-44` 是整数 7、0x11、8 的 float 视图。`.asm` 中是整数比较（`CMP EAX,7` 等）。
8. `FUN_00404da0` 的 `.c` 按 float32 做乘法，`.asm` 的乘积精确，只在存储时舍入一次。两者结果相同，但 `.asm` 的字节截断用的是未舍入乘积（第 4.2 节）。

---

## 6. 无法确定的事项

1. x87 精度控制字（53 位还是 64 位）没有在本文 dump 中确认，本文假定 53 位（推测）。它只影响 t、缓动与和差的末位，不影响上面的结构。
2. 模式 0x11 的语义未闭合。进行中路径是 `P += R2`、`R2 += Q`，完成路径却输出 Q。若 Q 是终点，则 `R2 += Q` 会累积。这是否是 ZUN 的缺陷或设计，无法从代码判断。
3. 模式 7 的完成路径输出 P，不再加一次 Q。因此最终输出为 `P0 + (N−1)·Q`（由进行中路径推出，推测其设计意图未知）。
4. R1（+0x38）与 R2（+0x54）不被 op 9 写入。模式 8 与 0x11 使用它们，其初值来自哪里（STD 分配清零、上一次 tween 残留，还是其它写入者）未查。
5. 步长指针 `+0x7c`：本文已知的写入者（`0x403f9f`、`0x404a3a`）都写 `&DAT_00476f78`（值 1.0）。若没有其它写入者，非整数分支不可达。其它写入者（是否有 op 调节雾速度）未查。
6. flags bit0 的来源：op 9 初始化依赖 `STD+0x168` 的 bit0 为 0。若 STD 分配不清零则行为不同。未查。
7. 四个字节的颜色语义：调用方对 `op+0x10..0x13` 的 R、G、B、A 顺序（`0x403eef–0x403f37`）为 R、G、B、A，内存顺序即字节 0..3。与 D3DCOLOR 字节序的关系未闭合（`m4_r7.md` 无法确定 #6）。
8. `FUN_0044c350` 的缓动数学没有在本文核对，沿用 `m2_motion.md` 的表。其 x87 舍入顺序未逐条核对。
9. `_ftol2` 的符号判断用 float32 副本（`0x463b35–0x463b37`）。对普通数值与截断等价，对接近 0 的值的边界行为未验证。
10. 调用频率：`FUN_00403c80` 是否每帧调用一次 `FUN_004049a0` 未核对。本文的 counter 语义按“每次调用推进一步”描述。
11. `prev`（+0x70）在本函数内只写不读，是否有其它读者未查。
12. `N ≤ 0` 的进行中分支（`0x4049be` JLE）未见合法调用，语义仅按汇编描述。
13. 与任务清单的差异：`_callers.txt` 中 `FUN_00404e10` 有 6 处（含 `0x404c32`），`FUN_00404da0` 有 5 处（含 `0x404c25`）。本文均已列入第 5.3 节。

---

## 7. 审校更正（2026-11-01）

以下更正以本节为准，覆盖正文对应内容。

1. **x87 精度是 24 位，不是 53 位。** `CreateDevice` 不带 `D3DCREATE_FPU_PRESERVE`（`m2_motion.md` §2、`m4_render_re.md` R2），D3D9 把精度控制设为 24 位。因此：
   - FADD/FSUB/FMUL/FDIV 的结果已经是 float32 精度（指数范围仍是扩展的，只在溢出边界有差别）。第 1 节“乘积精确、和差两次舍入”的模型作废。
   - 第 2.2 节“字节取自未舍入的 s2..s5”的区分不存在：寄存器里的值与 `FST` 写出的 float32 相同，所以字节就是 `byte(int64(out.F[k+2]))`。第 2.2 节末的 2.99999997 例子不会出现。
   - 模式 8 的 t 是 float32 除法（`FDIVR` 按 24 位舍入）。第 5.5 节第 2、3 条描述的是 53 位下的差别，24 位下不成立：每一步都按 float32 舍入。
2. **步长指针指向游戏速度。** `DAT_00476f78` 是游戏速度（`std.VM.Update` 的 `gs`），不是常数 1.0。第 6 节第 5 条因此已解决：非整数分支在游戏速度偏离 1 超过 1% 时可达，与 `zun.Timer.Tick` 相同。
3. **`FUN_004049a0` 与 `FUN_00404610` 是同一个插值模板**，记录换成了 7 dword。状态块 `+0x70..+0x80` 是 `zun.Timer` 的 `{prev, cur, f, gs*, flags}`：
   - 第 5.2 节完成分支里的 flags 判断是计时器的惰性初始化，合起来就是 `zun.Timer.Set(N)`（`FUN_00405410`）。
   - op 9 初始化的第 7、8 步（第 5.4 节）合起来就是 `zun.Timer.Reset()`。
   - P、Q、R1、R2 分别对应 `zun.Interp` 的 Start、End、Tan0、Tan1。
   - 模式 0x11 就是 `zun.Interp` 的模式 17（速度加加速度）。模式 7 的完成路径返回 Start，与 `zun.Interp` 相同。
   - 第 6 节第 2、3 条不是缺陷，是模板的既有语义。
4. **Hermite 权重的分组：** h00 = `((1 + (t + t))·a)·a`，即先算 `t + t`，再加 1（0x404b28–0x404b3e）。第 5.2 节伪代码的 `(1+2*t)` 在数值上相同，但 `.c` 写成 `1 + t + t`，容易被读成 `(1 + t) + t`。
   - `FUN_00404610`（0x404804–0x40481a）和 `FUN_00412ac0`（0x412c38 起，字节 `d9 c0 d8 c1 d9 c1 d8 25 fc 0a 47 00 d9 1c 24 d9 05 fc 0a 47 00 d8 c1`）是同样的分组。
   - `zun.Interp` 按 `(1 + t) + t` 实现，这是一个逻辑层的偏差（ECL 模式 8 移动受影响），已修正（`zun.HermiteWeights`，2026-11-01）。
5. **R1、R2 的初值：** op 9 不写它们，也没有其它写入者（`FUN_00403c80` 中只有 0x404321 的调用读取）。按 STD 对象分配时清零处理，即恒为 0。
6. **雾记录初值：** `+0x2b48..+0x2b60` 来自构造函数 `FUN_00402230` 复制的 Field 相机块（`DAT_00491d7c+0xfc..+0x114`），未追。项目中按 0 处理，直到第一条 op 8 写入。
7. **op 8 同步更新四个颜色浮点**（0x403e7f–0x403ee4）：`+0x2b60` = 颜色 dword；`+0x2b50..+0x2b5c` = 颜色字节 0..3 的整数值转 float；然后写 near、far。所以记录里的浮点与打包颜色在 op 8 之后一致。
8. **字节顺序：** 记录分量 2..5 是颜色 dword 的内存字节 0..3。按 D3DCOLOR（ARGB）解读，依次是 B、G、R、A。正文称之为 R、G、B、A，只是命名问题：打包后的 dword 与 op 8 的 `u32` 读法一致。

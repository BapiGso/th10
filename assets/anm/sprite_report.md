# Sprite 资源报告

## 检测说明

- 本文只统计 `assets/anm/` 下的 PNG，共 `221` 张；同目录里的 `SVG` 等源文件不计入目录树。
- 像素尺寸由 PowerShell 直接读取 PNG `IHDR` 头得到，统一记为 `宽x高`。
- 下述“推测”项结合了文件命名、尺寸结构、当前项目代码和东方风神录原版资源拆分惯例；未反编译 `.anm` 切片脚本，因此个别细节仍保留不确定性。

## 目录树（带像素尺寸）

```text
assets/anm/
├── ascii/
│   ├── ascii.png [256x256]
│   ├── leaf.png [226x220]
│   ├── loading.png [128x128]
│   └── pause.png [256x256]
├── background/
│   ├── stg1bg.png [256x256]
│   ├── stg1bg2.png [256x256]
│   ├── stg1bg3.png [128x128]
│   ├── stg1bg4.png [128x128]
│   ├── stg2bg.png [256x256]
│   ├── stg2bg2.png [512x512]
│   ├── stg3bg.png [256x256]
│   ├── stg3bg2.png [128x128]
│   ├── stg3bg3.png [512x512]
│   ├── stg3bg4.png [256x512]
│   ├── stg4bg.png [512x512]
│   ├── stg4bg3.png [512x256]
│   ├── stg4bg7.png [384x448]
│   ├── stg5bg.png [256x256]
│   ├── stg5bg2.png [256x256]
│   ├── stg6bg.png [512x512]
│   ├── stg6bg2.png [256x256]
│   ├── stg6bg3.png [32x256]
│   ├── stg6bg5.png [256x256]
│   ├── stg6bg6.png [32x128]
│   ├── stg7bg.png [256x256]
│   ├── stg7bg2.png [256x256]
│   └── stg7bg3.png [32x256]
├── bullet/
│   ├── etama.png [256x256]
│   ├── etama2.png [256x256]
│   ├── etama3.png [128x128]
│   └── etama6.png [256x256]
├── card/
│   ├── cdbg01a.png [384x448]
│   ├── cdbg01b.png [256x256]
│   ├── cdbg02a.png [384x448]
│   ├── cdbg02b.png [256x256]
│   ├── cdbg03a.png [384x448]
│   ├── cdbg03b.png [256x256]
│   ├── cdbg04a.png [384x448]
│   ├── cdbg04b.png [256x256]
│   ├── cdbg05a.png [384x448]
│   ├── cdbg05b.png [256x256]
│   ├── cdbg06a.png [384x448]
│   ├── cdbg06b.png [128x128]
│   ├── cdbg07a.png [384x448]
│   └── cdbg07b.png [256x256]
├── ending/
│   ├── e00a.png [512x480]
│   ├── e00ar.png [128x480]
│   ├── e00b.png [512x480]
│   ├── e00br.png [128x480]
│   ├── e01b.png [512x480]
│   ├── e01br.png [128x480]
│   ├── e02b.png [512x480]
│   ├── e02br.png [128x480]
│   ├── e03b.png [512x480]
│   ├── e03br.png [128x480]
│   ├── e04b.png [512x480]
│   ├── e04br.png [128x480]
│   ├── e05b.png [512x480]
│   ├── e05br.png [128x480]
│   ├── e06b.png [512x480]
│   ├── e06br.png [128x480]
│   ├── e06c.png [512x480]
│   ├── e06cr.png [128x480]
│   ├── e07b.png [512x480]
│   ├── e07br.png [128x480]
│   ├── e07c.png [512x480]
│   ├── e07cr.png [128x480]
│   ├── e08b.png [512x480]
│   ├── e08br.png [128x480]
│   ├── e08c.png [512x480]
│   ├── e08cr.png [128x480]
│   ├── e09a.png [512x480]
│   ├── e09ar.png [128x480]
│   ├── e09b.png [512x480]
│   ├── e09br.png [128x480]
│   ├── e09c.png [512x480]
│   ├── e09cr.png [128x480]
│   ├── e10a.png [512x480]
│   ├── e10ar.png [128x480]
│   ├── e10b.png [512x480]
│   ├── e10br.png [128x480]
│   ├── e10c.png [512x480]
│   ├── e10cr.png [128x480]
│   ├── e11b.png [512x480]
│   ├── e11br.png [128x480]
│   ├── e11c.png [512x480]
│   ├── e11cr.png [128x480]
│   ├── st00.png [512x256]
│   ├── st01.png [512x480]
│   ├── st01r.png [128x480]
│   ├── st02.png [512x512]
│   ├── st03.png [512x480]
│   ├── st03r.png [128x480]
│   ├── st04.png [512x256]
│   ├── st05.png [512x480]
│   ├── st05r.png [128x480]
│   ├── staff.png [512x512]
│   └── staff2.png [256x128]
├── enemy/
│   └── enemy.png [512x512]
├── face/
│   ├── enemy1/
│   │   ├── ename01.png [128x64]
│   │   ├── face01an_d.png [256x64]
│   │   ├── face01an_u.png [256x256]
│   │   ├── face01ct.png [256x512]
│   │   ├── face01n2_d.png [256x64]
│   │   ├── face01n2_u.png [256x256]
│   │   ├── face01no_d.png [256x64]
│   │   └── face01no_u.png [256x256]
│   ├── enemy1m/
│   │   └── face01mct.png [256x512]
│   ├── enemy2/
│   │   ├── ename02.png [128x64]
│   │   ├── face02an_u.png [256x256]
│   │   ├── face02ct.png [256x512]
│   │   ├── face02lo_u.png [256x256]
│   │   ├── face02n2_u.png [256x256]
│   │   ├── face02no_d.png [256x64]
│   │   ├── face02no_u.png [256x256]
│   │   └── face02pr_u.png [256x256]
│   ├── enemy3/
│   │   ├── ename03.png [128x64]
│   │   ├── face03an_u.png [256x256]
│   │   ├── face03ct.png [256x512]
│   │   ├── face03lo_u.png [256x256]
│   │   ├── face03n2_u.png [256x256]
│   │   ├── face03no_d.png [256x64]
│   │   ├── face03no_u.png [256x256]
│   │   └── face03sp_u.png [256x256]
│   ├── enemy4/
│   │   ├── ename04.png [128x64]
│   │   ├── face04ct.png [256x512]
│   │   ├── face04dp_u.png [256x256]
│   │   ├── face04lo_u.png [256x256]
│   │   ├── face04n2_u.png [256x256]
│   │   ├── face04no_d.png [256x64]
│   │   ├── face04no_u.png [256x256]
│   │   ├── face04pr_u.png [256x256]
│   │   └── face04sp_u.png [256x256]
│   ├── enemy5/
│   │   ├── ename05.png [128x64]
│   │   ├── face05an_u.png [256x256]
│   │   ├── face05ct.png [256x512]
│   │   ├── face05dp_u.png [256x256]
│   │   ├── face05lo_u.png [256x256]
│   │   ├── face05n2_u.png [256x256]
│   │   ├── face05no_d.png [256x64]
│   │   ├── face05no_u.png [256x256]
│   │   ├── face05sp_u.png [256x256]
│   │   └── face05sw_u.png [256x256]
│   ├── enemy6/
│   │   ├── ename06.png [128x64]
│   │   ├── face06ct.png [512x512]
│   │   ├── face06dp_u.png [512x256]
│   │   ├── face06n2_u.png [512x256]
│   │   ├── face06no_d.png [512x128]
│   │   └── face06no_u.png [512x256]
│   ├── enemy7/
│   │   ├── ename07.png [128x64]
│   │   ├── face07ct.png [256x512]
│   │   ├── face07dp_u.png [256x256]
│   │   ├── face07lo_u.png [256x256]
│   │   ├── face07no_d.png [256x64]
│   │   ├── face07no_u.png [256x256]
│   │   └── face07pr_u.png [256x256]
│   ├── pl00/
│   │   ├── face_pl00an_u.png [256x256]
│   │   ├── face_pl00dp_u.png [256x256]
│   │   ├── face_pl00hp_u.png [256x256]
│   │   ├── face_pl00n2_u.png [256x256]
│   │   ├── face_pl00no_d.png [256x64]
│   │   ├── face_pl00no_u.png [256x256]
│   │   ├── face_pl00pr_u.png [256x256]
│   │   ├── face_pl00sp_u.png [256x256]
│   │   └── face_pl00sw_u.png [256x256]
│   ├── pl01/
│   │   ├── face_pl01an_u.png [256x256]
│   │   ├── face_pl01dp_u.png [256x256]
│   │   ├── face_pl01hp_u.png [256x256]
│   │   ├── face_pl01n2_u.png [256x256]
│   │   ├── face_pl01no_d.png [256x64]
│   │   ├── face_pl01no_u.png [256x256]
│   │   ├── face_pl01pr_u.png [256x256]
│   │   ├── face_pl01sp_u.png [256x256]
│   │   └── face_pl01sw_u.png [256x256]
│   └── dummy.png [8x8]
├── front/
│   ├── ename.png [128x256]
│   ├── front00.png [512x512]
│   ├── front00s.png [640x480]
│   ├── st01logo.png [128x512]
│   ├── st02logo.png [128x512]
│   ├── st03logo.png [128x512]
│   ├── st04logo.png [128x512]
│   ├── st05logo.png [128x512]
│   ├── st06logo.png [160x512]
│   └── st07logo.png [128x512]
├── loading/
│   ├── sig.png [512x480]
│   ├── sig_r.png [128x480]
│   └── sigm.png [640x480]
├── player/
│   ├── pl00/
│   │   └── pl00.png [256x256]
│   └── pl01/
│       └── pl01.png [256x256]
├── stgenm/
│   ├── stg1enm.png [256x256]
│   ├── stg2enm.png [256x256]
│   ├── stg3enm.png [256x256]
│   ├── stg3enm2.png [256x256]
│   ├── stg4benm.png [256x256]
│   ├── stg4enm.png [256x256]
│   ├── stg5enm.png [256x256]
│   ├── stg6enm.png [256x256]
│   ├── stg6enm2.png [256x256]
│   ├── stg6enm3.png [128x64]
│   ├── stg6enm4.png [256x256]
│   ├── stg7enm.png [256x256]
│   ├── stg7enm2.png [256x256]
│   └── stg7enm3.png [256x256]
└── title/
    ├── rank00.png [512x512]
    ├── result00.png [256x256]
    ├── select00.png [512x480]
    ├── select00b.png [128x480]
    ├── select00s.png [640x480]
    ├── select01.png [256x512]
    ├── sl_pl00.png [256x256]
    ├── sl_pl00b.png [256x256]
    ├── sl_pl01.png [256x256]
    ├── sl_pl01b.png [256x256]
    ├── title_logo.png [512x256]
    ├── title_ver.png [64x16]
    ├── title00a.png [512x480]
    ├── title00b.png [128x480]
    ├── title00s.png [640x480]
    ├── title01.png [512x512]
    └── weapon.png [512x512]
```

## player/（2 张）

### `pl00/pl00.png`

- 尺寸: `256x256`
- 帧大小: `30x46`（ANM 实际裁切）；按 `32x48` 单元格排布，四周约有 `1px` 留白。
- 布局: 顶部角色动画区为 `8列 x 3行`；下半区是非统一的 shot/option/effect 区。
- 动画序列: 正面 `8帧`；左/右转各 `8帧`；另有回正过渡脚本（推测）。
- 备注: 共 `52` 个 sprite 定义，其中角色本体是前 `24` 帧，和原版 `256x256 / 8列 / 32x48` 惯例基本一致。

### `pl01/pl01.png`

- 尺寸: `256x256`
- 帧大小: `30x46`（ANM 实际裁切）；按 `32x48` 单元格排布。
- 布局: 顶部角色动画区同样是 `8列 x 3行`；下半区为混合区，包含 `32x16`、`32x32`、`64x64`，以及一条非常规的 `480x14` 长条效果。
- 动画序列: 正面 `8帧`；左/右转各 `8帧`；回正时复用反向帧序列（推测）。
- 备注: 共 `45` 个 sprite 定义；角色本体仍是标准的 `24` 帧顶区。

### 小结

- 两张玩家图都严格落在风神录玩家 sheet 惯例上: `256x256` 底图，角色本体按 `8列 x 3行` 排列，单元格约 `32x48`，实际有效裁切是 `30x46`。
- 当前项目的 `sprite/player.go` 只切取顶部 `8x3` 行走图；下半区的 shot/option/effect 资源仍未接线。

## bullet/（4 张）

### `etama.png`

- 尺寸: `256x256`
- 帧大小: 混合图集，主要有 `8x8`、`14x14`、`14x16`、`16x14`、`16x16`、`30x30`。
- 布局: 以 `16列` 小弹网格为主，底部还有 `8列` 的 `30x30` 大弹区和 `8x8` 微弹区。
- 包含弹型: 推测包括微弹、小玉、米弹/针弹、中玉、大玉等基础弹幕。
- 备注: 共 `256` 个 sprite 定义，是标准基础弹幕总图集，符合风神录常见 `256x256` 子弹 atlas 习惯。

### `etama2.png`

- 尺寸: `256x256`
- 帧大小: 混合 `16x16`、`30x30`、`32x32`、`64x64`、`128x128`。
- 布局: 顶部有一整行 `16x16`；中部有若干 `64x64` 大块；右下有一块 `128x128`；下部还有 `30x30`、`32x32`、`16x16` 补充区。
- 包含弹型: 推测是特殊大弹、环形弹、蓄力/爆发类特效，不是规则单一弹幕网格。
- 备注: 共 `54` 个 sprite 定义，且有少量 ANM 重复引用同一贴图区块。

### `etama3.png`

- 尺寸: `128x128`
- 帧大小: 可见为 `14x128` 的纵向长条；另有一条 ANM 声明的特殊 `15x448` 长条。
- 布局: 不是常规行列式 spritesheet，而是纵向条带布局；图像检测能清楚看到 `3` 条窄竖带。
- 包含弹型: 推测用于激光/beam 本体或长条尾迹纹理。
- 备注: `15x448` 已超出 PNG 本身高度，说明渲染端很可能会对这条纹理做纵向重复或拉伸。

### `etama6.png`

- 尺寸: `256x256`
- 帧大小: 以 `30x30` 为主，另有 `32x32` 和 `64x64`。
- 布局: `8列 x 5行` 的 `30x30` 主区，外加 `1` 行 `32x32`，底部还有 `4` 张 `64x64` 大图。
- 包含弹型: 推测为大玉、符卡弹和大型命中特效。
- 备注: 共 `52` 个 sprite 定义，是明显偏“大型弹体/特效”的补充图集。

### 小结

- `etama.png` 是基础弹幕总图集；`etama2.png` 和 `etama6.png` 偏特殊大弹/特效；`etama3.png` 更像激光条纹理。
- 当前项目只加载 `etama.png`，并且仅取出米弹、小玉和一部分中玉切片；`TypeLarge` 仍暂时复用 `Middle`，`TypeLaser` 也还没有对上 `etama3.png`。

## enemy/（1 张）

### `enemy.png`

- 尺寸: `512x512`
- 帧大小: 以 `32x32` 为主，另有 `64x64`。
- 布局: 主杂兵区主要是 `12列` 的 `32x32` 网格；右侧和底部有 `64x64` 的大型敌人/特效区。
- 动画序列: 多套杂兵动画，脚本模式看起来是常见的 `12帧一组`，即正面 `4帧` 加左右偏航/回正序列（推测）。
- 包含敌型: 多种杂兵/妖精动画帧，以及若干中大型敌人帧。
- 备注: 共 `124` 个 sprite 定义，其中 `112` 个是 `32x32`，`12` 个是 `64x64`；和“enemy.png 存放多类杂兵动画”的原版习惯一致。

### 小结

- 这是当前项目唯一接入的通用敌人 atlas。
- 运行时目前只明确使用了小妖精序列；Boss 和大敌人仍以几何占位为主。

## stgenm/（14 张）

- 这一组是 stage-specific enemy atlas，用来补足 `enemy/enemy.png` 无法覆盖的关卡专属敌人、中 Boss、Boss 本体及附件。
- 总体规律很清楚: 除 `stg6enm3.png` 外几乎全是 `256x256`，说明原版依然偏向“小图集按关拆分”；出现多张文件的关卡，通常意味着 Boss 本体、附件和特殊姿势较多。
- `stg1enm.png` `256x256`: 已分析；可视上符合 `4列 x 4行` 的 `64x64` 网格，Stage 1 专用中型敌人/场景敌人 atlas。
- `stg2enm.png` `256x256`: Stage 2 专属敌人/Boss atlas，尺寸与 Stage 1 相同，推测仍以规则大格切片为主。
- `stg3enm.png` `256x256`: Stage 3 主 atlas。
- `stg3enm2.png` `256x256`: Stage 3 第二张补充 atlas，通常意味着 Boss 或附件动画较多。
- `stg4enm.png` `256x256`: Stage 4 通常场景敌人/中 Boss atlas。
- `stg4benm.png` `256x256`: 文件名里的 `b` 很像 `boss`，大概率是 Stage 4 Boss 专用 sheet。
- `stg5enm.png` `256x256`: Stage 5 专属敌人/Boss atlas。
- `stg6enm.png` `256x256`: Stage 6 主 atlas。
- `stg6enm2.png` `256x256`: Stage 6 第二张补充 atlas。
- `stg6enm3.png` `128x64`: 这一张是明显异类，更像小型附件条、手臂/神具部件或局部特效 strip，而不是完整角色表。
- `stg6enm4.png` `256x256`: Stage 6 第四张补充 atlas。
- `stg7enm.png` `256x256`: Extra Stage 主 atlas。
- `stg7enm2.png` `256x256`: Extra Stage 第二张补充 atlas。
- `stg7enm3.png` `256x256`: Extra Stage 第三张补充 atlas。
- 结论: `stgenm/` 明显是“每关专属角色表”的目录；Stage 6 与 Extra 的资源最重，和原作后期 Boss 角色/附件更多的直觉一致。

## loading/（3 张）

- `sig.png` `512x480`: 加载画面的主图，名字很像 `signature`，可理解为“加载画面签名/插画主体”。
- `sig_r.png` `128x480`: 与 `sig.png` 正好拼成 `640x480`，明显是右侧补条。
- `sigm.png` `640x480`: 已合成好的满屏版本；`m` 可以理解为 merged/main composite。
- 这组命名与 `title/`、`ending/` 的全屏资源拆分方式一致，符合老式 `512 + 128 = 640` 的横向拼接套路。
- 当前加载场景只使用了 `sig.png`，没有拼 `sig_r.png`，也没有直接使用 `sigm.png`。

## title/（17 张）

- 这一目录同时覆盖标题画面、选人画面、难度/排行/结算等前端 UI。
- 标题主画面:
  - `title00a.png` `512x480` + `title00b.png` `128x480` = 标题背景拆分版。
  - `title00s.png` `640x480` = 标题背景合成版。
  - `title_logo.png` `512x256` = 游戏 logo。
  - `title_ver.png` `64x16` = 版本号小标签。
  - `title01.png` `512x512` = 标题菜单用的 UI atlas，可能包含菜单框、提示、装饰和光效切片。
- 角色/装备/难度选择:
  - `select00.png` `512x480` + `select00b.png` `128x480` = 选人背景拆分版。
  - `select00s.png` `640x480` = 选人背景合成版。
  - `select01.png` `256x512` = 选人界面的 UI atlas，可能含游标、框体、说明栏等。
  - `sl_pl00.png`、`sl_pl00b.png`、`sl_pl01.png`、`sl_pl01b.png` 各 `256x256`: 两位自机的选人立绘/补充层，`b` 很可能是第二层、替换层或额外说明面板。
  - `weapon.png` `512x512`: 从命名上看最像 shot type / weapon 选择用图集，应该对应 Type A/B/C 的图标和说明部件。
- 排行/结算:
  - `rank00.png` `512x512`: 大概率是 rank / difficulty / result 类 UI 字样和牌面资源。
  - `result00.png` `256x256`: 结算界面专用 sheet。
- 结论: `title/` 基本包办了标题、选人、难度、排行、结算这整条前端链路；当前项目这些场景仍是文字+色块占位，因此整目录实际尚未接线。

## front/（10 张）

- `st01logo.png` 到 `st07logo.png`: 都是纵向长图（Stage 6 为 `160x512`，其余多为 `128x512`），很像每关开场飞入的 stage logo / chapter title。
- `ename.png` `128x256`: 从命名看是通用的敌方名字框或前景层名字 UI 容器。
- `front00.png` `512x512`: 前景层总图集，可能包含关卡 title card、Boss 名字框、HUD 装饰或场景前景碎片。
- `front00s.png` `640x480`: 已合成的满屏前景图，可能用于 stage intro 或某种过场前景叠层。
- 结论: `front/` 更像“关卡前景演出资源”而不是标题 UI；它和 `background/`、`card/` 一起构成场景演出的非实体层。

## ascii/（4 张）

- `ascii.png` `256x256`: 位图字库，通常会承载英数字、HUD 标签、菜单文字等。
- `leaf.png` `226x220`: 枫叶贴图；当前加载场景已经把它当作落叶粒子使用。
- `loading.png` `128x128`: 从命名看应是 loading 字样、图标或转场用的小图集。
- `pause.png` `256x256`: 暂停界面图集，通常会包含 `Pause`、`Retry`、`Quit` 之类菜单素材。
- 结论: `ascii/` 兼有通用字库和少量 UI 特效；当前除了 `leaf.png` 以外都还没接线。

## background/（23 张）

- 这组文件名与关卡编号完全对应，`stg1` 到 `stg6` 是正篇六关，`stg7` 对应 Extra。
- Stage 1:
  - `stg1bg.png` `256x256`
  - `stg1bg2.png` `256x256`
  - `stg1bg3.png` `128x128`
  - `stg1bg4.png` `128x128`
- Stage 2:
  - `stg2bg.png` `256x256`
  - `stg2bg2.png` `512x512`
- Stage 3:
  - `stg3bg.png` `256x256`
  - `stg3bg2.png` `128x128`
  - `stg3bg3.png` `512x512`
  - `stg3bg4.png` `256x512`
- Stage 4:
  - `stg4bg.png` `512x512`
  - `stg4bg3.png` `512x256`
  - `stg4bg7.png` `384x448`
- Stage 5:
  - `stg5bg.png` `256x256`
  - `stg5bg2.png` `256x256`
- Stage 6:
  - `stg6bg.png` `512x512`
  - `stg6bg2.png` `256x256`
  - `stg6bg3.png` `32x256`
  - `stg6bg5.png` `256x256`
  - `stg6bg6.png` `32x128`
- Extra:
  - `stg7bg.png` `256x256`
  - `stg7bg2.png` `256x256`
  - `stg7bg3.png` `32x256`
- 结构解读:
  - `256x256`、`128x128`、`32x256` 这类尺寸很像可重复平铺的地面/墙面/柱条/栅格纹理。
  - `512x512`、`512x256`、`256x512` 更像大块远景或中景纹理。
  - `384x448` 的 `stg4bg7.png` 接近游戏场大小，可能是 Stage 4 某个非平铺的特殊前景/中景板。
- 当前各关 `BgDraw` 都是程序化线条和色块，`background/` 目录尚未被实际加载。

## card/（14 张）

- `cdbg01` 到 `cdbg07` 一共 `7` 组，数量正好对应 Stage 1 到 Stage 6 再加 Extra Boss。
- 每组都分为 `a` / `b` 两张:
  - `cdbg01a.png` 到 `cdbg07a.png` 全是 `384x448`，这正好接近游戏场区域，明显像符卡背景主板。
  - `cdbg01b.png` 到 `cdbg07b.png` 多为 `256x256`，更像叠加层、旋转纹理、噪声光纹或中心法阵；只有 `cdbg06b.png` 是 `128x128`，属于特例。
- 可推测用途:
  - `01` 对应 Stage 1 Boss
  - `02` 对应 Stage 2 Boss
  - `03` 对应 Stage 3 Boss
  - `04` 对应 Stage 4 Boss
  - `05` 对应 Stage 5 Boss
  - `06` 对应 Stage 6 Boss
  - `07` 对应 Extra Boss
- 当前项目已有符卡逻辑和名称字段，但尚未实现符卡背景演出，因此整组资源未接线。

## ending/（53 张）

- 这组资源的命名非常成体系，明显是结局 CG + staff roll 资源。
- `e00` 到 `e11` 一共 `12` 组，对应 TH10 的 `2` 名自机 × `3` 种装备 × `good/bad` 两类结局，数量上完全对得上。
- `eNN[a|b|c].png`:
  - 主图基本都是 `512x480`。
  - 对应的 `eNN[a|b|c]r.png` 都是 `128x480`，显然是右侧补条。
  - 也就是说每张完整结局页实际上是 `640x480 = 512 + 128` 的拼接结构。
  - 并不是每个 ending route 都有完整的 `a/b/c` 三页，说明不同路线页数不同。
- `st00.png`、`st01.png`、`st01r.png`、`st02.png`、`st03.png`、`st03r.png`、`st04.png`、`st05.png`、`st05r.png`:
  - 这一组更像 staff roll 的背景页、标题页或过渡页。
  - `st01` / `st03` / `st05` 仍是标准 `512+128` 拆分的全屏页。
  - `st00`、`st04` 是 `512x256`，像横向横幅或章节标题板。
  - `st02` 是 `512x512`，可能是较大的通用背景 atlas。
- `staff.png` `512x512` + `staff2.png` `256x128`:
  - 大概率是 staff roll 文字、装饰、logo、分隔牌之类的独立图集。
- 当前 `scene/ending` 仍是 TODO，因此 `ending/` 全目录未接线。

## face/（76 张）

- 目录结构:
  - `pl00`、`pl01` 是两位自机对话头像。
  - `enemy1` 到 `enemy7` 对应 Stage 1 到 Stage 6 再加 Extra 的敌方头像。
  - `enemy1m` 是 Stage 1 的特殊分支目录，只含一张 `face01mct.png`，大概率是另一位秋姐妹或 midboss 专用 cut-in。
  - `dummy.png` `8x8` 是占位图。
- 尺寸规律:
  - 大多数头像上半部分是 `256x256`，下半条是 `256x64`，拼起来像一张 `256x320` 的立绘组合。
  - `faceNNct.png` 普遍是 `256x512`，像完整 cut-in / tall portrait。
  - `enemy6` 是唯一明显放大的套装，`face06ct.png` 为 `512x512`，`*_u` 为 `512x256`，`*_d` 为 `512x128`，说明 final boss 头像规格更大。
  - `enameNN.png` 都是 `128x64`，像名字牌。
- 命名规则推测:
  - `face_pl00...` / `face_pl01...`: 自机头像。
  - `face01...` 到 `face07...`: 敌方头像。
  - `_u`: 立绘上半部分。
  - `_d`: 立绘下半条。
  - `ct`: 完整 cut-in / 长立绘。
  - `enameNN`: 对应角色名字贴图。
  - 表情缩写按命名大致可读为: `no` 正常，`an` 生气，`n2` 第二常态/严肃，`dp` 低落，`hp` 高兴，`lo` 低气压/困惑，`pr` 得意，`sp` 惊讶或特殊，`sw` 微笑；其中只有 `no` 最稳，其余都属于保守推测。
- 当前对话系统只画文本框，不显示任何 `face/` 头像，因此整个目录尚未接线。

## 项目接线状态

- 打包状态: `assets/assets.go` 已经 embed 了整个 `anm/` 目录，所以所有 PNG 都会随程序打包；但“被 embed”不等于“被场景实际加载”。
- 当前直接加载到运行时的 PNG 只有 `6` 张:
  - `anm/loading/sig.png`
  - `anm/ascii/leaf.png`
  - `anm/player/pl00/pl00.png`
  - `anm/player/pl01/pl01.png`
  - `anm/bullet/etama.png`
  - `anm/enemy/enemy.png`
- 已接线但只用到局部切片:
  - `player/`: 只切顶部 `8x3` 行走图；下半区的 shot/option/effect 未用。
  - `bullet/etama.png`: 只取了米弹、小玉和一部分中玉；`Large` 仍暂时复用 `Middle`，激光也还没对上 `etama3.png`。
  - `enemy/enemy.png`: loader 会切完整的小妖精/大敌人区域，但运行时目前只明确用到小妖精动画；Boss 和大敌人仍以矩形占位。
- 已存在场景但仍是占位实现:
  - `scene/title` 和 `scene/select` 仍用 `DebugPrintAt` + `vector` 绘制文本与色块，尚未接 `title/` 图集。
  - `scene/dialog` 只画文本框，尚未接 `face/`、`front/ename.png` 或各 `enameNN.png`。
  - `scene/stage/stage.go` 明写“以下绘制全部用几何占位，后续替换为贴图”；各关 `BgDraw` 也都是程序化线条/色块，因此 `background/`、`front/`、`card/` 尚未接线。
  - `scene/result` 与 `scene/ending` 都还是 TODO，因此 `title/result00.png` 和整个 `ending/` 目录仍未使用。
- 完全未接入或未直接加载的整类资源:
  - `stgenm/`
  - `background/`
  - `card/`
  - `face/`
  - `front/`
  - `title/`
  - `ending/`
  - `loading/sig_r.png`、`loading/sigm.png`
  - `ascii/ascii.png`、`ascii/loading.png`、`ascii/pause.png`
  - `bullet/etama2.png`、`bullet/etama3.png`、`bullet/etama6.png`
- 特别说明:
  - 当前 loading 场景只居中绘制 `sig.png`，没有拼 `sig_r.png`，也没有直接使用 `sigm.png`，所以左右两侧仍是黑底。
  - 结论上可以把项目现状理解为“核心战斗实体 atlas 已有最小接线，前端 UI / 对话 / 背景 / 符卡 / 结局资源基本还停留在待接入状态”。

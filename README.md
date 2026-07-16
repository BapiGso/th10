# 东方风神录 Ebiten 复刻 — 设计文档

## 一、项目定位

用 Go + Ebiten 复刻「东方风神录 ~ Mountain of Faith」（东方Project 第十弹）的核心玩法。
不追求 1:1 还原，而是以原作为蓝本，构建一个可扩展的弹幕 STG 框架。

> 「东方风神录　～ Mountain of Faith.」为更加直接的弹幕射击游戏。
> 在不知何时起到处都是妖怪的博丽神社附近，神明们的弹幕苏醒了。

通常认为，从本作起的一系列东方作品，和红魔乡到本作之前的东方作品有着一定的差异性。
东方风神录的历史背景与部分角色是以日本神话中的诹访大战以及诹访神社的相关神明为基础进行改编而成的。

---

## 一.五、游戏设定

### 自机角色

本作包含 **2 位自机角色 × 3 种装备 = 6 个机体**。
游戏对话根据角色而不同，ending 根据机体（装备）而不同。

| 角色 | 称号 | 装备 A | 装备 B | 装备 C |
|---|---|---|---|---|
| 博丽灵梦 | 乐园的巫女 | 灵符（诱导弹） | 梦符（前方集中） | 神符（巫女敏符） |
| 雾雨魔理沙 | 普通的魔法使 | 魔符（魔法导弹） | 恋符（Master Spark） | 星符（星屑光线） |

- `Character`：0 = 灵梦，1 = 魔理沙
- `ShotType`：0 = A, 1 = B, 2 = C

### 游戏系统特色

本作在游戏系统上进行了较大程度的简化：
- **取消** bomb 道具、擦弹奖励（风神录的擦弹不影响得分）、蓝点计数、结算页面
- **灵击系统**：替代传统 Bomb，消耗信仰点数发动
- **信仰系统**：信仰值与剧情相关，影响得分倍率；残机以得分获得，得分受信仰系统影响
- **Power 系统**：火力 0.00 ~ 5.00（内部 0~500），满火力后 P 道具转化为得分道具

### 关卡结构（六面 + Extra）

| 关卡 | 道中曲 | BOSS 曲 | BOSS | 称号 |
|---|---|---|---|---|
| Stage 1 | 眷爱众生之神 ～ Romantic Fall | 会受稻田姬的斥责啦 | 秋静叶（道中）/ 秋穰子 | 寂寞与终焉的象征 / 丰裕与收成的象征 |
| Stage 2 | 厄神降临之路 ～ Dark Road | 命运的阴暗面 | 键山雏 | 秘神流雏 |
| Stage 3 | 众神眷恋的幻想乡 | 芥川龙之介的河童 ～ Candid Friend | 河城荷取 | 超妖怪弹头 |
| Stage 4 | Fall of Fall ～ 秋意渐浓之瀑 | 妖怪之山 ～ Mysterious Mountain | 犬走椛（道中）/ 射命丸文 | 基层警卫天狗 / 最接近村落的天狗 |
| Stage 5 | 少女曾见的日本原风景 | 信仰是为了虚幻之人 | 东风谷早苗 | 被祭拜的风之人 |
| Stage 6 | 御柱的墓场 ～ Grave of Being | 神圣庄严的古战场 ～ Suwa Foughten Field | 八坂神奈子 | 山与湖的化身 |
| Extra | 明日之盛，昨日之俗 | Native Faith | 洩矢诹访子 | 土著神的顶点 |

- Stage 1 道中 BOSS：秋静叶
- Stage 4 道中 BOSS：犬走椛
- Extra 道中：八坂神奈子（再登场）

### 难度

| 值 | 难度 |
|---|---|
| 0 | Easy |
| 1 | Normal |
| 2 | Hard |
| 3 | Lunatic |
| 4 | Extra（通关 Normal+ 后解锁） |

---

## 二、技术选型

| 层            | 选择              | 理由                           |
|--------------|-----------------|------------------------------|
| 语言           | Go 1.26         | 编译快、跨平台、GC 可控                |
| 渲染 / 窗口 / 输入 | Ebiten v2       | Go 生态唯一成熟 2D 引擎，内置 60TPS 定时器 |
| 音频           | ebiten/v2/audio | 支持 WAV SE + OGG/WAV BGM 无缝循环 |
| 资源嵌入         | embed.FS        | 单二进制分发，无需外部文件                |

**不引入额外框架。** 弹幕逻辑、碰撞、渲染分层、输入映射全部自研——
这也是文章系列的核心观点：引擎只提供渲染/音频/输入，弹幕游戏的灵魂必须自己写。

---

## 三、主循环架构（文章第 9 篇）

```
┌──────────────────────────────────┐
│           ebiten.RunGame         │
│  ┌────────────────────────────┐  │
│  │  Update()  — 60 TPS 固定  │  │
│  │  1. input.Global.Update()  │  │  ← 输入采集（第 10 篇）
│  │  2. scene.Update()         │  │  ← 数据处理
│  │     player.Update()        │  │
│  │     enemies.Update()       │  │
│  │     bullets.Update()       │  │
│  │     collision check        │  │
│  │     items.Update()         │  │
│  │     effects.Update()       │  │
│  └────────────────────────────┘  │
│  ┌────────────────────────────┐  │
│  │  Draw()    — VSync 或自由  │  │
│  │  renderer.Flush(screen)    │  │  ← 分层渲染（第 3 篇）
│  └────────────────────────────┘  │
└──────────────────────────────────┘
```

**铁律**：`Draw` 只读取状态，`Update` 只修改状态。二者绝不交叉。
这保证了暂停时画面静止、Replay 可重现。

---

## 四、项目结构

```
th10/
├── cmd/main.go                  # 入口
├── game/game.go                 # Game 主循环 + Scene 接口 + 全局常量
│
├── input/input.go               # 键映射 + JustPressed/IsPressed/Snapshot
│                                  支持 Replay 录制/回放（第 10 篇）
│
├── render/layer.go              # 分层渲染器（第 3 篇）
│                                  11 层，Additive 与 AlphaBlend 分组
│
├── collision/collision.go       # 碰撞检测（第 12/12.5 篇）
│                                  圆-圆 / 矩形 / 正交圆 / 边界检测
│
├── danmaku/emitter.go           # 弹幕发射器：Ring / Aimed / Spiral / Random
│
├── entity/
│   ├── player/player.go         # 自机（移动/射击/决死/动画状态机）
│   ├── bullet/bullet.go         # 敌弹 + 对象池（2048 容量）
│   ├── playerbullet/             # 自机子弹 + 对象池
│   ├── enemy/enemy.go           # 敌人（路径移动/受击/掉落）
│   ├── item/item.go             # 道具（重力抛物/磁吸/自动回收线）
│   └── effect/effect.go         # 特效（爆炸/擦弹/消弹）+ 对象池
│
├── scene/
│   ├── loading/                 # 少女祈祷中…
│   ├── title/                   # 标题 + 主菜单
│   ├── select/                  # 角色/难度选择
│   ├── stage/stage.go           # 关卡通用逻辑（暂停/对话触发/Boss 战切换）
│   │   └── stage1/              # Stage 1 脚本（敌人编排 + 弹幕时间轴）
│   ├── dialog/                  # 剧情对话
│   ├── result/                  # 结算
│   └── ending/                  # ED
│
├── audio/audio.go               # BGM / SE 管理（第 8 篇）
├── assets/                      # embed 资源（图片/音频/字体）
└── data/                        # 存档 / 配置（第 16 篇）
```

---

## 五、核心系统设计

### 5.1 输入系统（第 10 篇）

```
键盘/手柄 → 键映射层 → KeyPacket（bool 位掩码）→ 游戏逻辑
```

- `IsPressed(key)`：持续按住（移动/射击）
- `JustPressed(key)`：边沿触发（菜单确认/Bomb）
- `HoldFrames(key)`：连续帧数（菜单长按加速）
- `Snapshot() / Restore()`：Replay 录制/回放只需每帧存 16bit

### 5.2 碰撞系统（第 12 / 12.5 篇）

采用**标准圆判定**（不用正交圆，文章 12.5 篇已论证其非线性缺陷）。

判定半径参考值（第 15 篇）：

| 对象 | 半径 |
|---|---|
| 自机 | 2 px |
| 米弹 | 1 px |
| 小玉 | 2 px |
| 中玉 | 6 px |
| 大玉 | 10 px |

9 组碰撞对（第 11 篇）：

1. 敌弹 vs 自机 → 被弹/擦弹
2. 敌体 vs 自机 → 被弹
3. Boss 体 vs 自机 → 被弹
4. 自机弹 vs 杂兵 → 伤害
5. 自机弹 vs Boss → 伤害
6. Bomb vs 敌弹 → 消弹
7. Bomb vs 杂兵 → 伤害
8. Bomb vs Boss → 伤害
9. 道具 vs 自机 → 拾取

### 5.3 渲染分层（第 3 篇）

11 层，从底到顶：

```
0  Background         AlphaBlend
1  Item               AlphaBlend
2  PlayerBullet       AlphaBlend
3  Player             AlphaBlend
4  Enemy              AlphaBlend
5  Bullet (normal)    AlphaBlend
6  Bullet (additive)  Additive ← 高光弹
7  Hitbox             Additive ← 判定点
8  Effect             Additive ← 特效
9  Foreground         AlphaBlend
10 UI / HUD           AlphaBlend
```

连续 Additive 层不需要切换混合模式，最多只有 3 次切换。

### 5.4 对象池

弹幕游戏同屏子弹可达 2000+。如果每颗子弹都 `new`，GC 压力会导致掉帧。

方案：预分配固定大小切片，用 `Active` 标志位管理生命周期。

| 池 | 容量 | 说明 |
|---|---|---|
| 敌弹 Pool | 2048 | 覆盖最密弹幕 |
| 自机弹 Pool | 256 | 足够双线 + Option |
| 道具 Pool | 256 | Boss 击破大量掉落 |
| 特效 Pool | 512 | 消弹/爆炸 |

### 5.5 弹幕发射器

将"发射参数"与"子弹运动"分离：

- **Emitter**：持有位置 + 子弹池引用，提供 `Ring / Aimed / Spiral / Random` 等原语
- **Stage 脚本**：在特定帧调用 Emitter 的方法组合弹幕

```go
// Stage1 Boss 第一张符卡示例伪码
if boss.Age%8 == 0 {
    emitter.SetPos(boss.X, boss.Y)
    emitter.Ring(24, 2.5, float64(boss.Age)*0.05, bullet.TypeSmall, 1)
}
```

子弹自身支持 `AccelS`（速度加速度）和 `AccelA`（角度加速度），可做变速弹和弧线弹。

### 5.6 自机手感（第 15 篇）

| 参数          | 值            |
|-------------|--------------|
| 高速移动        | 4.5 px/帧（灵梦） |
| 低速移动        | 2 px/帧       |
| 射击间隔        | 3 帧/发        |
| 复活无敌        | 300 帧（5 秒）   |
| Bomb 无敌     | 60 帧         |
| 决死窗口        | 16 帧         |
| 行走图正面/侧身帧间隔 | 8 帧          |
| 行走图转身过渡帧间隔  | 6 帧          |

自机状态机：`Normal → Dead → (决死Bomb?) → Respawn → Normal`

### 5.7 音频（第 8 篇）

- **BGM**：WAV 解码 + `InfiniteLoopWithIntro` 实现无缝循环，需预设 loopStart / loopEnd
- **SE**：每次播放创建新 Player 实例（允许重叠播放）
- SE 全部 embed 为 WAV，启动时不额外加载

### 5.8 信仰系统（风神录独有）

- 信仰值初始 50000，上限 100000
- 击破敌人、收集道具可增加信仰值
- 信仰值随时间缓慢衰减
- 得分倍率 = 信仰值 / 基准值，直接影响最终得分
- 被弹时信仰值大幅下降
- 灵击（Bomb）消耗信仰值而非独立的 Bomb 道具数

### 5.9 存档数据（第 16 篇）

```go
type SaveData struct {
    HighScores  [5][2][3][10]ScoreEntry // [难度(含Extra)][角色][装备][Top10]
    SpellCards  [5][2][]SpellRecord     // [难度][角色][符卡ID]
    PlayStats   [5][2][3]PlayStat       // [难度][角色][装备]
    Unlocks     uint64                  // 位掩码（Extra解锁等）
    DefaultName string
}
```

JSON 存储，启动时加载，通关/中途退出时写入。

---

## 六、场景流转

```
Loading → Title ─┬→ Game Start → Select(角色/装备/难度) → Stage1 → (Dialog) → Stage1 Boss
                 │                                      → Stage2 → ... → Stage6 Boss
                 │                                      → Ending → Result → Title
                 ├→ Extra Start → Select(角色/装备) → Extra Stage → Extra Boss
                 │                                  → Ending → Result → Title
                 ├→ Practice → Select(角色/装备/难度/关卡) → 单关 → Result → Title
                 ├→ Replay → 选择记录 → 回放 → Title
                 ├→ Player Data → 查看成绩/符卡 → Title
                 ├→ Music Room → 试听BGM → Title
                 ├→ Option(Hint/BGM Vol/SE Vol/Key Config/Default/Quit) → Title
                 └→ Quit
```

每个 Scene 实现 `Update(g *Game) error` + `Draw(screen *ebiten.Image)`。
通过 `g.GoTo(nextScene)` 切换，下一帧生效，避免当前帧状态不一致。

---

## 七、帧率与同步（第 5/6/7 篇）

- Ebiten 内置 60 TPS 固定逻辑帧，渲染帧独立，天然符合文章推荐的「绝对帧率控制」
- 不依赖 VSync 做帧率控制（VSync 增加输入延迟）
- 如需 BGM 同步：每秒对比 `audioPlayer.Position()` 与 `frameCount/60`，偏差 > 50ms 则修正

---

## 八、性能目标

参考文章第 1/2 篇的基准：同屏 6000 对象 @60FPS 即可满足弹幕游戏需求。

Ebiten 在现代硬件上轻松达到此目标，前提是：
1. 对象池避免 GC 抖动
2. 减少混合模式切换（分层渲染）
3. 碰撞检测先做矩形粗判，再做圆形精判

---

## 九、开发路线

1. **Phase 1 — 可操控自机** ✅（已完成骨架）
   - 自机移动 / 射击 / 判定点 / 行走图动画
2. **Phase 2 — 单面弹幕**
   - Stage 1 背景滚动 + 杂兵编排 + 基础弹幕
3. **Phase 3 — Boss 战**
   - 符卡系统 / 血条 / 时间限制 / 对话
4. **Phase 4 — 完整一面**
   - 道具 / 得分 / 擦弹 / SE / BGM
5. **Phase 5 — 系统完善**
   - 标题 / 选人 / 结算 / 存档 / Replay
6. **Phase 6 — 多面扩展**
   - Stage 2~6 + Extra

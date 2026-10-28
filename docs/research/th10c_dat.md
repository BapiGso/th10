# th10c.dat（2009 年汉化补丁）内容调查（2026-10-27）

用户决定对话默认使用中文**繁体**（2026-10-27），文本来源是 `.tmp/[th10]/th10c.dat`。该文件与 `th10chs.exe`
（简体）、`th10cht.exe`（繁体）配套，可用 `datfs` 直接读取（THA1，190 个文件）。

## 与 th10.dat 的对比

| 类别 | 结论 |
|---|---|
| 84 个文件 | 与 th10.dat 逐字节相同 |
| 54 个同名不同内容 | 简体版：`.msg` 全部（GBK 编码，XOR 方式与原版相同）、`musiccmt.txt`、`stage01–07.ecl`、`front.anm`、`title.anm`、`staff.anm`、`stgenm01–07.anm`、`st01–07logo.anm`、`e07–e09.anm` |
| 52 个只在 c 中 | 繁体版：`.mst`（对应 `.msg`）、`.ect`（对应 `.ecl`）、`.ant`（对应 `.anm`）、`musiccmtt.txt` |

即简繁两套都在同一个 dat 里：简体沿用原扩展名，繁体把扩展名最后一个字母改成 `t`。

## ECL：不能拿来跑逻辑

- `stage01/07` 的 `.ecl`/`.ect`：子程序数与指令完全相同，只有符卡名字符串不同。
- 简体 `stage04.ecl`、`stage06.ecl`：头部子程序数翻倍（0x34 → 0x68），文件大小约为原版两倍，
  现有 loader 解析失败（truncated）。
- 繁体 `stage06.ect`：`BossCard2` 比原版少一条指令。

**结论**：游戏逻辑始终使用 th10.dat 的原版 ECL；汉化 ECL 只作为符卡名的来源，
按子程序名加 357/358/359 的出现顺序对应取字符串。

## 用于 M5/M6 的资源

- 对话：使用繁体 `stNN_0M.mst`、结局 `eNN.mst`、`staff.mst`（简体 `.msg` 不用）。
- 音乐室：`musiccmt.txt`（简）/ `musiccmtt.txt`（繁）。
- 汉化贴图：`front`（HUD 文字）、`title`（标题菜单）、`stNNlogo`（关卡标题）、
  `stgenmNN`（符卡相关）、`staff` 的 `.anm/.ant`。
- 嵌入方式与原版资源相同：用 `tools/th10extract` 原样提取，并加入 `gamedata` 校验
  （需要让校验测试同时对照 th10c.dat）。

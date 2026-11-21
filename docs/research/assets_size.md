# 资源体积普查（t14，阶段 A）

日期：2026-11-08。此普查在贴图、音频和字体改动前独立完成。MB 使用十进制，MiB 使用二进制。未运行原作。

## 1．方法与基线

普查命令：`go run ./tools/th10extract -out assets -audit D:/GoProjects/th10/.tmp/codex/t14/audit_baseline.json`。逐项解析嵌入目录中的原 ANM／ANT、THTX、WAV、TTC／TTF、ECL／STD／MSG，BGM 用本机 FFmpeg 8.1.2 的 ffprobe 检查。颜色统计包含透明像素下的 RGB，不忽略 alpha＝0；哈希包含 THTX 格式、尺寸和完整原始像素。

发布构建命令：`$env:GOOS='js'; $env:GOARCH='wasm'; go build -trimpath -ldflags='-s -w' -o <out> ./cmd`。gzip 由 Python 标准库 `gzip.compress(data, compresslevel=9, mtime=0)` 实测，不是估算。

| 二进制 | 原始字节 | gzip -9 字节 |
|---|---:|---:|
| WASM | 174,428,427 | 102,448,368 |
| Windows amd64 | 175,175,168 | 102,826,313 |

## 2．资源类别

下表是磁盘资源总量。当前 embed 的目录通配符另见第 4 节，`study` 不在原 embed 中，但属于非运行时笔记。

| 类别 | 文件数 | 字节 | MB | 占磁盘资源 |
|---|---:|---:|---:|---:|
| anm | 278 | 107,761,889 | 107.762 | 67.97% |
| ecl | 21 | 1,352,853 | 1.353 | 0.85% |
| font | 3 | 8,402,116 | 8.402 | 5.30% |
| misc | 6 | 37,540 | 0.038 | 0.02% |
| msg | 54 | 98,044 | 0.098 | 0.06% |
| sht | 6 | 16,496 | 0.016 | 0.01% |
| std | 7 | 18,864 | 0.019 | 0.01% |
| study | 17 | 127,636 | 0.128 | 0.08% |
| wav | 56 | 40,719,048 | 40.719 | 25.68% |
| 合计 | 448 | 158,534,486 | 158.534 | 100% |

ANM／ANT 中共有 318 个有像素的 THTX 块，原始像素 77,467,904 字节，按格式＋尺寸＋原字节去重后为 227 块。目录中的 PNG 又重复保存了其中大量贴图。

## 3．逐贴图清单

格式：1＝BGRA8888，3＝RGB565，5＝ARGB4444，7＝Gray8。`#` 后是 ANM entry 的零基下标；尺寸是 THTX 的实际尺寸，不是精灵矩形或 ANM 的逻辑尺寸。相同列列出其他完全相同的 entry；`—` 表示没有重复。

| ANM／entry | 名称 | 格式 | 尺寸 | 原始字节 | 唯一颜色 | 全透明比例 | 完全相同 |
|---|---|---:|---|---:|---:|---:|---|
| ascii.anm#0 | ascii/ascii.png | 5 | 256×256 | 131,072 | 1,092 | 67.47% | — |
| ascii.anm#1 | ascii/loading.png | 5 | 128×128 | 32,768 | 164 | 76.51% | — |
| bullet.anm#0 | bullet/etama.png | 1 | 256×256 | 262,144 | 21,893 | 36.45% | — |
| bullet.anm#1 | bullet/etama6.png | 1 | 256×256 | 262,144 | 15,056 | 31.10% | — |
| bullet.anm#2 | bullet/etama2.png | 5 | 256×256 | 131,072 | 3,044 | 39.74% | — |
| bullet.anm#3 | bullet/etama3.png | 5 | 128×128 | 32,768 | 1,139 | 32.59% | — |
| e00.anm#0 | ending/e00a.png | 3 | 512×480 | 491,520 | 512 | 0.00% | e01.anm#0, e02.anm#0, e03.anm#0, e04.anm#0, e05.anm#0, e06.anm#0, e07.anm#0, e08.anm#0, e11.anm#0 |
| e00.anm#1 | ending/e00ar.png | 3 | 128×480 | 122,880 | 157 | 0.00% | e01.anm#1, e02.anm#1, e03.anm#1, e04.anm#1, e05.anm#1, e06.anm#1, e07.anm#1, e08.anm#1, e11.anm#1 |
| e00.anm#2 | ending/e00b.png | 3 | 512×480 | 491,520 | 536 | 0.00% | — |
| e00.anm#3 | ending/e00br.png | 3 | 128×480 | 122,880 | 155 | 0.00% | — |
| e01.anm#0 | ending/e00a.png | 3 | 512×480 | 491,520 | 512 | 0.00% | e00.anm#0, e02.anm#0, e03.anm#0, e04.anm#0, e05.anm#0, e06.anm#0, e07.anm#0, e08.anm#0, e11.anm#0 |
| e01.anm#1 | ending/e00ar.png | 3 | 128×480 | 122,880 | 157 | 0.00% | e00.anm#1, e02.anm#1, e03.anm#1, e04.anm#1, e05.anm#1, e06.anm#1, e07.anm#1, e08.anm#1, e11.anm#1 |
| e01.anm#2 | ending/e01b.png | 3 | 512×480 | 491,520 | 538 | 0.00% | — |
| e01.anm#3 | ending/e01br.png | 3 | 128×480 | 122,880 | 157 | 0.00% | — |
| e02.anm#0 | ending/e00a.png | 3 | 512×480 | 491,520 | 512 | 0.00% | e00.anm#0, e01.anm#0, e03.anm#0, e04.anm#0, e05.anm#0, e06.anm#0, e07.anm#0, e08.anm#0, e11.anm#0 |
| e02.anm#1 | ending/e00ar.png | 3 | 128×480 | 122,880 | 157 | 0.00% | e00.anm#1, e01.anm#1, e03.anm#1, e04.anm#1, e05.anm#1, e06.anm#1, e07.anm#1, e08.anm#1, e11.anm#1 |
| e02.anm#2 | ending/e02b.png | 3 | 512×480 | 491,520 | 543 | 0.00% | — |
| e02.anm#3 | ending/e02br.png | 3 | 128×480 | 122,880 | 160 | 0.00% | — |
| e03.anm#0 | ending/e00a.png | 3 | 512×480 | 491,520 | 512 | 0.00% | e00.anm#0, e01.anm#0, e02.anm#0, e04.anm#0, e05.anm#0, e06.anm#0, e07.anm#0, e08.anm#0, e11.anm#0 |
| e03.anm#1 | ending/e00ar.png | 3 | 128×480 | 122,880 | 157 | 0.00% | e00.anm#1, e01.anm#1, e02.anm#1, e04.anm#1, e05.anm#1, e06.anm#1, e07.anm#1, e08.anm#1, e11.anm#1 |
| e03.anm#2 | ending/e03b.png | 3 | 512×480 | 491,520 | 541 | 0.00% | — |
| e03.anm#3 | ending/e03br.png | 3 | 128×480 | 122,880 | 327 | 0.00% | e04.anm#3, e05.anm#3 |
| e04.anm#0 | ending/e00a.png | 3 | 512×480 | 491,520 | 512 | 0.00% | e00.anm#0, e01.anm#0, e02.anm#0, e03.anm#0, e05.anm#0, e06.anm#0, e07.anm#0, e08.anm#0, e11.anm#0 |
| e04.anm#1 | ending/e00ar.png | 3 | 128×480 | 122,880 | 157 | 0.00% | e00.anm#1, e01.anm#1, e02.anm#1, e03.anm#1, e05.anm#1, e06.anm#1, e07.anm#1, e08.anm#1, e11.anm#1 |
| e04.anm#2 | ending/e04b.png | 3 | 512×480 | 491,520 | 542 | 0.00% | — |
| e04.anm#3 | ending/e04br.png | 3 | 128×480 | 122,880 | 327 | 0.00% | e03.anm#3, e05.anm#3 |
| e05.anm#0 | ending/e00a.png | 3 | 512×480 | 491,520 | 512 | 0.00% | e00.anm#0, e01.anm#0, e02.anm#0, e03.anm#0, e04.anm#0, e06.anm#0, e07.anm#0, e08.anm#0, e11.anm#0 |
| e05.anm#1 | ending/e00ar.png | 3 | 128×480 | 122,880 | 157 | 0.00% | e00.anm#1, e01.anm#1, e02.anm#1, e03.anm#1, e04.anm#1, e06.anm#1, e07.anm#1, e08.anm#1, e11.anm#1 |
| e05.anm#2 | ending/e05b.png | 3 | 512×480 | 491,520 | 543 | 0.00% | — |
| e05.anm#3 | ending/e05br.png | 3 | 128×480 | 122,880 | 327 | 0.00% | e03.anm#3, e04.anm#3 |
| e06.anm#0 | ending/e00a.png | 3 | 512×480 | 491,520 | 512 | 0.00% | e00.anm#0, e01.anm#0, e02.anm#0, e03.anm#0, e04.anm#0, e05.anm#0, e07.anm#0, e08.anm#0, e11.anm#0 |
| e06.anm#1 | ending/e00ar.png | 3 | 128×480 | 122,880 | 157 | 0.00% | e00.anm#1, e01.anm#1, e02.anm#1, e03.anm#1, e04.anm#1, e05.anm#1, e07.anm#1, e08.anm#1, e11.anm#1 |
| e06.anm#2 | ending/e06b.png | 3 | 512×480 | 491,520 | 493 | 0.00% | — |
| e06.anm#3 | ending/e06br.png | 3 | 128×480 | 122,880 | 172 | 0.00% | — |
| e06.anm#4 | ending/e06c.png | 3 | 512×480 | 491,520 | 483 | 0.00% | — |
| e06.anm#5 | ending/e06cr.png | 3 | 128×480 | 122,880 | 305 | 0.00% | — |
| e07.anm#0 | ending/e00a.png | 3 | 512×480 | 491,520 | 512 | 0.00% | e00.anm#0, e01.anm#0, e02.anm#0, e03.anm#0, e04.anm#0, e05.anm#0, e06.anm#0, e08.anm#0, e11.anm#0 |
| e07.anm#1 | ending/e00ar.png | 3 | 128×480 | 122,880 | 157 | 0.00% | e00.anm#1, e01.anm#1, e02.anm#1, e03.anm#1, e04.anm#1, e05.anm#1, e06.anm#1, e08.anm#1, e11.anm#1 |
| e07.anm#2 | ending/e07b.png | 3 | 512×480 | 491,520 | 443 | 0.00% | — |
| e07.anm#3 | ending/e07br.png | 3 | 128×480 | 122,880 | 176 | 0.00% | — |
| e07.anm#4 | ending/e07c.png | 3 | 512×480 | 491,520 | 452 | 0.00% | — |
| e07.anm#5 | ending/e07cr.png | 3 | 128×480 | 122,880 | 243 | 0.00% | — |
| e08.anm#0 | ending/e00a.png | 3 | 512×480 | 491,520 | 512 | 0.00% | e00.anm#0, e01.anm#0, e02.anm#0, e03.anm#0, e04.anm#0, e05.anm#0, e06.anm#0, e07.anm#0, e11.anm#0 |
| e08.anm#1 | ending/e00ar.png | 3 | 128×480 | 122,880 | 157 | 0.00% | e00.anm#1, e01.anm#1, e02.anm#1, e03.anm#1, e04.anm#1, e05.anm#1, e06.anm#1, e07.anm#1, e11.anm#1 |
| e08.anm#2 | ending/e08b.png | 3 | 512×480 | 491,520 | 458 | 0.00% | — |
| e08.anm#3 | ending/e08br.png | 3 | 128×480 | 122,880 | 175 | 0.00% | — |
| e08.anm#4 | ending/e08c.png | 3 | 512×480 | 491,520 | 466 | 0.00% | — |
| e08.anm#5 | ending/e08cr.png | 3 | 128×480 | 122,880 | 172 | 0.00% | — |
| e09.anm#0 | ending/e09a.png | 3 | 512×480 | 491,520 | 342 | 0.00% | — |
| e09.anm#1 | ending/e09ar.png | 3 | 128×480 | 122,880 | 172 | 0.00% | — |
| e09.anm#2 | ending/e09b.png | 3 | 512×480 | 491,520 | 513 | 0.00% | — |
| e09.anm#3 | ending/e09br.png | 3 | 128×480 | 122,880 | 174 | 0.00% | — |
| e09.anm#4 | ending/e09c.png | 3 | 512×480 | 491,520 | 487 | 0.00% | — |
| e09.anm#5 | ending/e09cr.png | 3 | 128×480 | 122,880 | 314 | 0.00% | — |
| e10.anm#0 | ending/e10a.png | 3 | 512×480 | 491,520 | 447 | 0.00% | — |
| e10.anm#1 | ending/e10ar.png | 3 | 128×480 | 122,880 | 218 | 0.00% | — |
| e10.anm#2 | ending/e10b.png | 3 | 512×480 | 491,520 | 475 | 0.00% | — |
| e10.anm#3 | ending/e10br.png | 3 | 128×480 | 122,880 | 174 | 0.00% | — |
| e10.anm#4 | ending/e10c.png | 3 | 512×480 | 491,520 | 503 | 0.00% | — |
| e10.anm#5 | ending/e10cr.png | 3 | 128×480 | 122,880 | 173 | 0.00% | — |
| e11.anm#0 | ending/e00a.png | 3 | 512×480 | 491,520 | 512 | 0.00% | e00.anm#0, e01.anm#0, e02.anm#0, e03.anm#0, e04.anm#0, e05.anm#0, e06.anm#0, e07.anm#0, e08.anm#0 |
| e11.anm#1 | ending/e00ar.png | 3 | 128×480 | 122,880 | 157 | 0.00% | e00.anm#1, e01.anm#1, e02.anm#1, e03.anm#1, e04.anm#1, e05.anm#1, e06.anm#1, e07.anm#1, e08.anm#1 |
| e11.anm#2 | ending/e11b.png | 3 | 512×480 | 491,520 | 498 | 0.00% | — |
| e11.anm#3 | ending/e11br.png | 3 | 128×480 | 122,880 | 325 | 0.00% | — |
| e11.anm#4 | ending/e11c.png | 3 | 512×480 | 491,520 | 427 | 0.00% | — |
| e11.anm#5 | ending/e11cr.png | 3 | 128×480 | 122,880 | 226 | 0.00% | — |
| enemy.anm#0 | enemy/enemy.png | 5 | 512×512 | 524,288 | 8,390 | 60.69% | — |
| front.anm#0 | front/front00.png | 5 | 512×512 | 524,288 | 1,242 | 26.08% | front.ant#0 |
| front.anm#1 | ascii/pause.png | 5 | 256×256 | 131,072 | 561 | 51.47% | — |
| front.anm#2 | front/ename.png | 5 | 128×256 | 65,536 | 18 | 85.63% | front.ant#2 |
| front.ant#0 | front/front00.png | 5 | 512×512 | 524,288 | 1,242 | 26.08% | front.anm#0 |
| front.ant#1 | ascii/pause.png | 5 | 256×256 | 131,072 | 453 | 59.43% | — |
| front.ant#2 | front/ename.png | 5 | 128×256 | 65,536 | 18 | 85.63% | front.anm#2 |
| pl00.anm#0 | player/pl00/pl00.png | 5 | 256×256 | 131,072 | 3,731 | 49.89% | — |
| pl00.anm#1 | face/pl00/face_pl00no_u.png | 1 | 256×256 | 262,144 | 7,429 | 57.69% | — |
| pl00.anm#2 | face/pl00/face_pl00n2_u.png | 1 | 256×256 | 262,144 | 7,738 | 57.22% | — |
| pl00.anm#3 | face/pl00/face_pl00hp_u.png | 1 | 256×256 | 262,144 | 7,604 | 57.21% | — |
| pl00.anm#4 | face/pl00/face_pl00an_u.png | 1 | 256×256 | 262,144 | 7,445 | 56.86% | — |
| pl00.anm#5 | face/pl00/face_pl00sw_u.png | 1 | 256×256 | 262,144 | 7,651 | 57.59% | — |
| pl00.anm#6 | face/pl00/face_pl00dp_u.png | 1 | 256×256 | 262,144 | 8,701 | 55.80% | — |
| pl00.anm#7 | face/pl00/face_pl00pr_u.png | 1 | 256×256 | 262,144 | 7,548 | 53.34% | — |
| pl00.anm#8 | face/pl00/face_pl00sp_u.png | 1 | 256×256 | 262,144 | 7,743 | 56.61% | — |
| pl00.anm#9 | face/pl00/face_pl00no_d.png | 1 | 256×64 | 65,536 | 2,994 | 37.40% | — |
| pl01.anm#0 | player/pl01/pl01.png | 5 | 256×256 | 131,072 | 2,788 | 58.07% | — |
| pl01.anm#1 | face/pl01/face_pl01no_u.png | 1 | 256×256 | 262,144 | 11,448 | 41.24% | — |
| pl01.anm#2 | face/pl01/face_pl01n2_u.png | 1 | 256×256 | 262,144 | 12,069 | 39.89% | — |
| pl01.anm#3 | face/pl01/face_pl01hp_u.png | 1 | 256×256 | 262,144 | 11,793 | 40.40% | — |
| pl01.anm#4 | face/pl01/face_pl01an_u.png | 1 | 256×256 | 262,144 | 11,189 | 41.22% | — |
| pl01.anm#5 | face/pl01/face_pl01sw_u.png | 1 | 256×256 | 262,144 | 11,590 | 40.87% | — |
| pl01.anm#6 | face/pl01/face_pl01dp_u.png | 1 | 256×256 | 262,144 | 11,732 | 38.64% | — |
| pl01.anm#7 | face/pl01/face_pl01pr_u.png | 1 | 256×256 | 262,144 | 11,517 | 38.60% | — |
| pl01.anm#8 | face/pl01/face_pl01sp_u.png | 1 | 256×256 | 262,144 | 11,741 | 39.66% | — |
| pl01.anm#9 | face/pl01/face_pl01no_d.png | 1 | 256×64 | 65,536 | 1,312 | 29.85% | — |
| sig.anm#0 | loading/sig.png | 1 | 512×480 | 983,040 | 20,274 | 0.00% | — |
| sig.anm#1 | loading/sig_r.png | 1 | 128×480 | 245,760 | 8,148 | 0.00% | — |
| st01logo.anm#0 | front/st01logo.png | 5 | 128×512 | 131,072 | 166 | 79.67% | — |
| st01logo.ant#0 | front/st01logo.png | 5 | 128×512 | 131,072 | 185 | 82.32% | — |
| st02logo.anm#0 | front/st02logo.png | 5 | 128×512 | 131,072 | 172 | 83.06% | — |
| st02logo.ant#0 | front/st02logo.png | 5 | 128×512 | 131,072 | 165 | 84.97% | — |
| st03logo.anm#0 | front/st03logo.png | 5 | 128×512 | 131,072 | 173 | 78.59% | — |
| st03logo.ant#0 | front/st03logo.png | 5 | 128×512 | 131,072 | 166 | 79.53% | — |
| st04logo.anm#0 | front/st04logo.png | 5 | 128×512 | 131,072 | 171 | 84.32% | — |
| st04logo.ant#0 | front/st04logo.png | 5 | 128×512 | 131,072 | 159 | 85.80% | — |
| st05logo.anm#0 | front/st05logo.png | 5 | 128×512 | 131,072 | 176 | 80.20% | — |
| st05logo.ant#0 | front/st05logo.png | 5 | 128×512 | 131,072 | 166 | 82.32% | — |
| st06logo.anm#0 | front/st06logo.png | 5 | 160×512 | 163,840 | 173 | 77.85% | — |
| st06logo.ant#0 | front/st06logo.png | 5 | 160×512 | 163,840 | 170 | 76.66% | — |
| st07logo.anm#0 | front/st07logo.png | 5 | 128×512 | 131,072 | 175 | 76.28% | — |
| st07logo.ant#0 | front/st07logo.png | 5 | 128×512 | 131,072 | 169 | 81.51% | — |
| staff.anm#0 | ending/st00.png | 1 | 512×256 | 524,288 | 11,398 | 0.00% | staff.ant#0 |
| staff.anm#1 | ending/staff.png | 5 | 512×512 | 524,288 | 67 | 82.97% | staff.ant#1 |
| staff.anm#2 | ending/staff2.png | 5 | 256×128 | 65,536 | 116 | 65.55% | — |
| staff.anm#3 | ending/st01.png | 3 | 512×480 | 491,520 | 26 | 0.00% | staff.ant#3 |
| staff.anm#4 | ending/st01r.png | 3 | 128×480 | 122,880 | 23 | 0.00% | staff.ant#4 |
| staff.anm#5 | ending/st02.png | 3 | 512×512 | 524,288 | 219 | 0.00% | staff.ant#5 |
| staff.anm#6 | ending/st03.png | 3 | 512×480 | 491,520 | 30 | 0.00% | staff.ant#6 |
| staff.anm#7 | ending/st03r.png | 3 | 128×480 | 122,880 | 30 | 0.00% | staff.ant#7 |
| staff.anm#8 | ending/st04.png | 3 | 512×256 | 262,144 | 363 | 0.00% | staff.ant#8 |
| staff.anm#9 | ending/st05.png | 3 | 512×480 | 491,520 | 61 | 0.00% | staff.ant#9 |
| staff.anm#10 | ending/st05r.png | 3 | 128×480 | 122,880 | 58 | 0.00% | staff.ant#10 |
| staff.ant#0 | ending/st00.png | 1 | 512×256 | 524,288 | 11,398 | 0.00% | staff.anm#0 |
| staff.ant#1 | ending/staff.png | 5 | 512×512 | 524,288 | 67 | 82.97% | staff.anm#1 |
| staff.ant#2 | ending/staff2.png | 5 | 256×128 | 65,536 | 168 | 66.91% | — |
| staff.ant#3 | ending/st01.png | 3 | 512×480 | 491,520 | 26 | 0.00% | staff.anm#3 |
| staff.ant#4 | ending/st01r.png | 3 | 128×480 | 122,880 | 23 | 0.00% | staff.anm#4 |
| staff.ant#5 | ending/st02.png | 3 | 512×512 | 524,288 | 219 | 0.00% | staff.anm#5 |
| staff.ant#6 | ending/st03.png | 3 | 512×480 | 491,520 | 30 | 0.00% | staff.anm#6 |
| staff.ant#7 | ending/st03r.png | 3 | 128×480 | 122,880 | 30 | 0.00% | staff.anm#7 |
| staff.ant#8 | ending/st04.png | 3 | 512×256 | 262,144 | 363 | 0.00% | staff.anm#8 |
| staff.ant#9 | ending/st05.png | 3 | 512×480 | 491,520 | 61 | 0.00% | staff.anm#9 |
| staff.ant#10 | ending/st05r.png | 3 | 128×480 | 122,880 | 58 | 0.00% | staff.anm#10 |
| stage01.anm#0 | background/stg1bg.png | 5 | 256×256 | 131,072 | 73 | 0.00% | — |
| stage01.anm#1 | background/stg1bg2.png | 1 | 256×256 | 262,144 | 13,159 | 16.60% | — |
| stage01.anm#2 | background/stg1bg3.png | 1 | 128×128 | 65,536 | 249 | 40.19% | stage05.anm#2 |
| stage01.anm#3 | background/stg1bg4.png | 1 | 128×128 | 65,536 | 256 | 56.94% | stage05.anm#3 |
| stage02.anm#0 | background/stg2bg.png | 1 | 256×256 | 262,144 | 6,397 | 0.00% | — |
| stage02.anm#1 | background/stg2bg2.png | 1 | 512×512 | 1,048,576 | 55,855 | 30.19% | — |
| stage03.anm#0 | background/stg3bg.png | 1 | 256×256 | 262,144 | 34,774 | 12.05% | — |
| stage03.anm#1 | background/stg3bg2.png | 1 | 128×128 | 65,536 | 7,499 | 0.00% | — |
| stage03.anm#2 | background/stg3bg3.png | 1 | 512×512 | 1,048,576 | 15,915 | 0.00% | — |
| stage03.anm#3 | background/stg3bg4.png | 1 | 256×512 | 524,288 | 20,358 | 53.06% | — |
| stage04.anm#0 | background/stg4bg.png | 1 | 512×512 | 1,048,576 | 57,064 | 0.00% | — |
| stage04.anm#1 | background/stg4bg3.png | 1 | 512×256 | 524,288 | 22,989 | 0.00% | — |
| stage04.anm#2 | background/stg4bg7.png | 1 | 384×448 | 688,128 | 6,172 | 0.00% | — |
| stage05.anm#0 | background/stg5bg.png | 1 | 256×256 | 262,144 | 22,103 | 0.00% | — |
| stage05.anm#1 | background/stg5bg2.png | 1 | 256×256 | 262,144 | 4,626 | 0.00% | — |
| stage05.anm#2 | background/stg1bg3.png | 1 | 128×128 | 65,536 | 249 | 40.19% | stage01.anm#2 |
| stage05.anm#3 | background/stg1bg4.png | 1 | 128×128 | 65,536 | 256 | 56.94% | stage01.anm#3 |
| stage06.anm#0 | background/stg6bg.png | 3 | 512×512 | 524,288 | 2,274 | 0.00% | — |
| stage06.anm#1 | background/stg6bg2.png | 1 | 256×256 | 262,144 | 3,881 | 0.00% | — |
| stage06.anm#2 | background/stg6bg3.png | 5 | 32×256 | 16,384 | 29 | 0.00% | — |
| stage06.anm#3 | background/stg6bg5.png | 1 | 256×256 | 262,144 | 184 | 0.00% | — |
| stage06.anm#4 | background/stg6bg6.png | 5 | 32×128 | 8,192 | 16 | 83.11% | — |
| stage07.anm#0 | background/stg7bg.png | 1 | 256×256 | 262,144 | 30,213 | 0.12% | — |
| stage07.anm#1 | background/stg7bg2.png | 1 | 256×256 | 262,144 | 3,235 | 0.00% | — |
| stage07.anm#2 | background/stg7bg3.png | 5 | 32×256 | 16,384 | 41 | 0.00% | — |
| stgenm01.anm#0 | stgenm/stg1enm.png | 5 | 256×256 | 131,072 | 2,125 | 76.70% | — |
| stgenm01.anm#1 | face/enemy1/face01no_u.png | 1 | 256×256 | 262,144 | 11,969 | 56.93% | — |
| stgenm01.anm#2 | face/enemy1/face01n2_u.png | 1 | 256×256 | 262,144 | 12,288 | 56.20% | — |
| stgenm01.anm#3 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm01.anm#4 | face/enemy1/face01an_u.png | 1 | 256×256 | 262,144 | 12,083 | 56.93% | — |
| stgenm01.anm#5 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm01.anm#6 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm01.anm#7 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm01.anm#8 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm01.anm#9 | face/enemy1/face01no_d.png | 1 | 256×64 | 65,536 | 3,234 | 20.99% | stgenm01.anm#10, stgenm01.anm#12 |
| stgenm01.anm#10 | face/enemy1/face01n2_d.png | 1 | 256×64 | 65,536 | 3,234 | 20.99% | stgenm01.anm#9, stgenm01.anm#12 |
| stgenm01.anm#11 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm01.anm#12 | face/enemy1/face01an_d.png | 1 | 256×64 | 65,536 | 3,234 | 20.99% | stgenm01.anm#9, stgenm01.anm#10 |
| stgenm01.anm#13 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm01.anm#14 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm01.anm#15 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm01.anm#16 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm01.anm#17 | card/cdbg01a.png | 5 | 384×448 | 344,064 | 449 | 8.16% | — |
| stgenm01.anm#18 | card/cdbg01b.png | 5 | 256×256 | 131,072 | 7 | 0.00% | — |
| stgenm01.anm#19 | face/enemy1/ename01.png | 5 | 128×64 | 16,384 | 308 | 57.85% | — |
| stgenm01.anm#20 | face/enemy1/face01ct.png | 5 | 256×512 | 262,144 | 1,546 | 52.82% | — |
| stgenm01.anm#21 | face/enemy1m/face01mct.png | 5 | 256×512 | 262,144 | 1,923 | 53.60% | — |
| stgenm02.anm#0 | stgenm/stg2enm.png | 5 | 256×256 | 131,072 | 674 | 83.37% | — |
| stgenm02.anm#1 | face/enemy2/face02no_u.png | 1 | 256×256 | 262,144 | 14,078 | 55.27% | — |
| stgenm02.anm#2 | face/enemy2/face02n2_u.png | 1 | 256×256 | 262,144 | 14,198 | 55.27% | — |
| stgenm02.anm#3 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm02.anm#4 | face/enemy2/face02an_u.png | 1 | 256×256 | 262,144 | 13,886 | 54.79% | — |
| stgenm02.anm#5 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm02.anm#6 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm02.anm#7 | face/enemy2/face02pr_u.png | 1 | 256×256 | 262,144 | 14,001 | 55.27% | — |
| stgenm02.anm#8 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm02.anm#9 | face/enemy2/face02lo_u.png | 1 | 256×256 | 262,144 | 14,719 | 54.99% | — |
| stgenm02.anm#10 | face/enemy2/face02no_d.png | 1 | 256×64 | 65,536 | 5,510 | 12.52% | — |
| stgenm02.anm#11 | card/cdbg02a.png | 5 | 384×448 | 344,064 | 16 | 0.00% | — |
| stgenm02.anm#12 | card/cdbg02b.png | 5 | 256×256 | 131,072 | 39 | 0.00% | — |
| stgenm02.anm#13 | face/enemy2/ename02.png | 5 | 128×64 | 16,384 | 318 | 61.01% | — |
| stgenm02.anm#14 | face/enemy2/face02ct.png | 5 | 256×512 | 262,144 | 1,374 | 53.45% | — |
| stgenm03.anm#0 | stgenm/stg3enm.png | 5 | 256×256 | 131,072 | 1,391 | 82.79% | — |
| stgenm03.anm#1 | stgenm/stg3enm2.png | 5 | 256×256 | 131,072 | 1,521 | 82.60% | — |
| stgenm03.anm#2 | face/enemy3/face03no_u.png | 1 | 256×256 | 262,144 | 28,678 | 41.52% | — |
| stgenm03.anm#3 | face/enemy3/face03n2_u.png | 1 | 256×256 | 262,144 | 28,775 | 41.52% | — |
| stgenm03.anm#4 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm03.anm#5 | face/enemy3/face03an_u.png | 1 | 256×256 | 262,144 | 27,438 | 40.85% | — |
| stgenm03.anm#6 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm03.anm#7 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm03.anm#8 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm03.anm#9 | face/enemy3/face03sp_u.png | 1 | 256×256 | 262,144 | 28,284 | 40.65% | — |
| stgenm03.anm#10 | face/enemy3/face03lo_u.png | 1 | 256×256 | 262,144 | 29,011 | 40.76% | — |
| stgenm03.anm#11 | face/enemy3/face03no_d.png | 1 | 256×64 | 65,536 | 11,836 | 19.33% | — |
| stgenm03.anm#12 | card/cdbg03a.png | 5 | 384×448 | 344,064 | 64 | 0.00% | — |
| stgenm03.anm#13 | card/cdbg03b.png | 5 | 256×256 | 131,072 | 106 | 35.63% | — |
| stgenm03.anm#14 | face/enemy3/ename03.png | 5 | 128×64 | 16,384 | 368 | 53.19% | — |
| stgenm03.anm#15 | face/enemy3/face03ct.png | 5 | 256×512 | 262,144 | 2,539 | 50.02% | — |
| stgenm04.anm#0 | stgenm/stg4enm.png | 5 | 256×256 | 131,072 | 2,026 | 77.97% | — |
| stgenm04.anm#1 | stgenm/stg4benm.png | 5 | 256×256 | 131,072 | 751 | 78.85% | — |
| stgenm04.anm#2 | face/enemy4/face04no_u.png | 1 | 256×256 | 262,144 | 17,616 | 44.53% | — |
| stgenm04.anm#3 | face/enemy4/face04n2_u.png | 1 | 256×256 | 262,144 | 17,663 | 44.53% | — |
| stgenm04.anm#4 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm04.anm#5 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm04.anm#6 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm04.anm#7 | face/enemy4/face04dp_u.png | 1 | 256×256 | 262,144 | 17,794 | 43.25% | — |
| stgenm04.anm#8 | face/enemy4/face04pr_u.png | 1 | 256×256 | 262,144 | 17,799 | 42.39% | — |
| stgenm04.anm#9 | face/enemy4/face04sp_u.png | 1 | 256×256 | 262,144 | 17,785 | 42.80% | — |
| stgenm04.anm#10 | face/enemy4/face04lo_u.png | 1 | 256×256 | 262,144 | 17,404 | 44.14% | — |
| stgenm04.anm#11 | face/enemy4/face04no_d.png | 1 | 256×64 | 65,536 | 3,661 | 23.50% | — |
| stgenm04.anm#12 | card/cdbg04a.png | 5 | 384×448 | 344,064 | 363 | 0.00% | — |
| stgenm04.anm#13 | card/cdbg04b.png | 5 | 256×256 | 131,072 | 304 | 18.38% | — |
| stgenm04.anm#14 | face/enemy4/ename04.png | 5 | 128×64 | 16,384 | 236 | 51.88% | — |
| stgenm04.anm#15 | face/enemy4/face04ct.png | 5 | 256×512 | 262,144 | 2,362 | 53.51% | — |
| stgenm05.anm#0 | stgenm/stg5enm.png | 5 | 256×256 | 131,072 | 1,791 | 78.23% | — |
| stgenm05.anm#1 | face/enemy5/face05no_u.png | 1 | 256×256 | 262,144 | 21,388 | 47.89% | — |
| stgenm05.anm#2 | face/enemy5/face05n2_u.png | 1 | 256×256 | 262,144 | 21,525 | 47.89% | — |
| stgenm05.anm#3 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm05.anm#4 | face/enemy5/face05an_u.png | 1 | 256×256 | 262,144 | 21,516 | 46.04% | — |
| stgenm05.anm#5 | face/enemy5/face05sw_u.png | 1 | 256×256 | 262,144 | 21,400 | 47.20% | — |
| stgenm05.anm#6 | face/enemy5/face05dp_u.png | 1 | 256×256 | 262,144 | 21,810 | 45.96% | — |
| stgenm05.anm#7 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm05.anm#8 | face/enemy5/face05sp_u.png | 1 | 256×256 | 262,144 | 21,488 | 46.64% | — |
| stgenm05.anm#9 | face/enemy5/face05lo_u.png | 1 | 256×256 | 262,144 | 21,538 | 47.89% | — |
| stgenm05.anm#10 | face/enemy5/face05no_d.png | 1 | 256×64 | 65,536 | 11,328 | 0.55% | — |
| stgenm05.anm#11 | card/cdbg05a.png | 5 | 384×448 | 344,064 | 491 | 0.00% | — |
| stgenm05.anm#12 | card/cdbg05b.png | 5 | 256×256 | 131,072 | 20 | 21.92% | stgenm07.anm#29 |
| stgenm05.anm#13 | face/enemy5/ename05.png | 5 | 128×64 | 16,384 | 297 | 51.07% | — |
| stgenm05.anm#14 | face/enemy5/face05ct.png | 5 | 256×512 | 262,144 | 3,539 | 29.40% | — |
| stgenm06.anm#0 | stgenm/stg6enm.png | 5 | 256×256 | 131,072 | 3,225 | 56.61% | stgenm07.anm#3 |
| stgenm06.anm#1 | stgenm/stg6enm2.png | 1 | 256×256 | 262,144 | 10,104 | 23.90% | — |
| stgenm06.anm#2 | stgenm/stg6enm3.png | 5 | 128×64 | 16,384 | 131 | 75.29% | — |
| stgenm06.anm#3 | stgenm/stg6enm4.png | 5 | 256×256 | 131,072 | 402 | 46.42% | — |
| stgenm06.anm#4 | face/enemy6/face06no_u.png | 1 | 512×256 | 524,288 | 23,958 | 57.71% | stgenm07.anm#18 |
| stgenm06.anm#5 | face/enemy6/face06n2_u.png | 1 | 512×256 | 524,288 | 24,173 | 57.71% | — |
| stgenm06.anm#6 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm06.anm#7 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm06.anm#8 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm06.anm#9 | face/enemy6/face06dp_u.png | 1 | 512×256 | 524,288 | 24,229 | 57.71% | stgenm07.anm#23 |
| stgenm06.anm#10 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm06.anm#11 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm06.anm#12 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm06.anm#13 | face/enemy6/face06no_d.png | 1 | 512×128 | 262,144 | 8,688 | 53.49% | stgenm07.anm#27 |
| stgenm06.anm#14 | card/cdbg06a.png | 5 | 384×448 | 344,064 | 473 | 0.00% | stgenm07.anm#28 |
| stgenm06.anm#15 | card/cdbg06b.png | 5 | 128×128 | 32,768 | 64 | 5.43% | — |
| stgenm06.anm#16 | face/enemy6/ename06.png | 5 | 128×64 | 16,384 | 194 | 53.47% | — |
| stgenm06.anm#17 | face/enemy6/face06ct.png | 5 | 512×512 | 524,288 | 1,904 | 58.22% | stgenm07.anm#30 |
| stgenm07.anm#0 | stgenm/stg7enm.png | 5 | 256×256 | 131,072 | 1,980 | 72.87% | — |
| stgenm07.anm#1 | stgenm/stg7enm3.png | 5 | 256×256 | 131,072 | 3,734 | 69.23% | — |
| stgenm07.anm#2 | stgenm/stg7enm2.png | 5 | 256×256 | 131,072 | 823 | 64.33% | — |
| stgenm07.anm#3 | stgenm/stg6enm.png | 5 | 256×256 | 131,072 | 3,225 | 56.61% | stgenm06.anm#0 |
| stgenm07.anm#4 | face/enemy7/face07no_u.png | 1 | 256×256 | 262,144 | 26,002 | 40.94% | — |
| stgenm07.anm#5 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm07.anm#6 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm07.anm#7 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm07.anm#8 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm07.anm#9 | face/enemy7/face07dp_u.png | 1 | 256×256 | 262,144 | 25,970 | 40.27% | — |
| stgenm07.anm#10 | face/enemy7/face07pr_u.png | 1 | 256×256 | 262,144 | 26,141 | 40.94% | — |
| stgenm07.anm#11 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm07.anm#12 | face/enemy7/face07lo_u.png | 1 | 256×256 | 262,144 | 26,968 | 40.38% | — |
| stgenm07.anm#13 | face/enemy7/face07no_d.png | 1 | 256×64 | 65,536 | 11,103 | 26.01% | — |
| stgenm07.anm#14 | card/cdbg07a.png | 5 | 384×448 | 344,064 | 225 | 0.00% | — |
| stgenm07.anm#15 | card/cdbg07b.png | 5 | 256×256 | 131,072 | 416 | 80.76% | — |
| stgenm07.anm#16 | face/enemy7/ename07.png | 5 | 128×64 | 16,384 | 264 | 54.05% | — |
| stgenm07.anm#17 | face/enemy7/face07ct.png | 5 | 256×512 | 262,144 | 1,891 | 52.09% | — |
| stgenm07.anm#18 | face/enemy6/face06no_u.png | 1 | 512×256 | 524,288 | 23,958 | 57.71% | stgenm06.anm#4 |
| stgenm07.anm#19 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm07.anm#20 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm07.anm#21 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm07.anm#22 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#24, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm07.anm#23 | face/enemy6/face06dp_u.png | 1 | 512×256 | 524,288 | 24,229 | 57.71% | stgenm06.anm#9 |
| stgenm07.anm#24 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#25, stgenm07.anm#26 |
| stgenm07.anm#25 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#26 |
| stgenm07.anm#26 | face/dummy.png | 1 | 8×8 | 256 | 1 | 0.00% | stgenm01.anm#3, stgenm01.anm#5, stgenm01.anm#6, stgenm01.anm#7, stgenm01.anm#8, stgenm01.anm#11, stgenm01.anm#13, stgenm01.anm#14, stgenm01.anm#15, stgenm01.anm#16, stgenm02.anm#3, stgenm02.anm#5, stgenm02.anm#6, stgenm02.anm#8, stgenm03.anm#4, stgenm03.anm#6, stgenm03.anm#7, stgenm03.anm#8, stgenm04.anm#4, stgenm04.anm#5, stgenm04.anm#6, stgenm05.anm#3, stgenm05.anm#7, stgenm06.anm#6, stgenm06.anm#7, stgenm06.anm#8, stgenm06.anm#10, stgenm06.anm#11, stgenm06.anm#12, stgenm07.anm#5, stgenm07.anm#6, stgenm07.anm#7, stgenm07.anm#8, stgenm07.anm#11, stgenm07.anm#19, stgenm07.anm#20, stgenm07.anm#21, stgenm07.anm#22, stgenm07.anm#24, stgenm07.anm#25 |
| stgenm07.anm#27 | face/enemy6/face06no_d.png | 1 | 512×128 | 262,144 | 8,688 | 53.49% | stgenm06.anm#13 |
| stgenm07.anm#28 | card/cdbg06a.png | 5 | 384×448 | 344,064 | 473 | 0.00% | stgenm06.anm#14 |
| stgenm07.anm#29 | card/cdbg05b.png | 5 | 256×256 | 131,072 | 20 | 21.92% | stgenm05.anm#12 |
| stgenm07.anm#30 | face/enemy6/face06ct.png | 5 | 512×512 | 524,288 | 1,904 | 58.22% | stgenm06.anm#17 |
| title.anm#0 | title/title01.png | 5 | 512×512 | 524,288 | 31 | 53.02% | title.ant#0 |
| title.anm#1 | title/title00a.png | 1 | 512×480 | 983,040 | 107,796 | 0.00% | title.ant#1 |
| title.anm#2 | title/title00b.png | 1 | 128×480 | 245,760 | 15,554 | 0.00% | title.ant#2 |
| title.anm#3 | title/title_logo.png | 5 | 512×256 | 262,144 | 628 | 61.07% | title.ant#3 |
| title.anm#4 | title/select00.png | 1 | 512×480 | 983,040 | 26,468 | 0.00% | title.ant#4 |
| title.anm#5 | title/select00b.png | 1 | 128×480 | 245,760 | 11,216 | 0.00% | title.ant#5 |
| title.anm#6 | title/select01.png | 1 | 256×512 | 524,288 | 13,314 | 45.73% | — |
| title.anm#7 | title/rank00.png | 1 | 512×512 | 1,048,576 | 6,106 | 53.65% | — |
| title.anm#8 | title/sl_pl00.png | 1 | 256×256 | 262,144 | 9,051 | 73.46% | title.ant#8 |
| title.anm#9 | title/sl_pl01.png | 1 | 256×256 | 262,144 | 7,079 | 67.58% | title.ant#9 |
| title.anm#10 | title/sl_pl00b.png | 5 | 256×256 | 131,072 | 566 | 38.33% | — |
| title.anm#11 | title/sl_pl01b.png | 5 | 256×256 | 131,072 | 529 | 40.61% | — |
| title.anm#12 | title/weapon.png | 5 | 512×512 | 524,288 | 1,116 | 47.29% | — |
| title.anm#13 | title/result00.png | 5 | 256×256 | 131,072 | 48 | 77.35% | — |
| title.ant#0 | title/title01.png | 5 | 512×512 | 524,288 | 31 | 53.02% | title.anm#0 |
| title.ant#1 | title/title00a.png | 1 | 512×480 | 983,040 | 107,796 | 0.00% | title.anm#1 |
| title.ant#2 | title/title00b.png | 1 | 128×480 | 245,760 | 15,554 | 0.00% | title.anm#2 |
| title.ant#3 | title/title_logo.png | 5 | 512×256 | 262,144 | 628 | 61.07% | title.anm#3 |
| title.ant#4 | title/select00.png | 1 | 512×480 | 983,040 | 26,468 | 0.00% | title.anm#4 |
| title.ant#5 | title/select00b.png | 1 | 128×480 | 245,760 | 11,216 | 0.00% | title.anm#5 |
| title.ant#6 | title/select01.png | 1 | 256×512 | 524,288 | 11,758 | 47.57% | — |
| title.ant#7 | title/rank00.png | 1 | 512×512 | 1,048,576 | 1,739 | 60.68% | — |
| title.ant#8 | title/sl_pl00.png | 1 | 256×256 | 262,144 | 9,051 | 73.46% | title.anm#8 |
| title.ant#9 | title/sl_pl01.png | 1 | 256×256 | 262,144 | 7,079 | 67.58% | title.anm#9 |
| title.ant#10 | title/sl_pl00b.png | 5 | 256×256 | 131,072 | 667 | 37.50% | — |
| title.ant#11 | title/sl_pl01b.png | 5 | 256×256 | 131,072 | 749 | 41.00% | — |
| title.ant#12 | title/weapon.png | 5 | 512×512 | 524,288 | 627 | 53.48% | — |
| title.ant#13 | title/result00.png | 5 | 256×256 | 131,072 | 54 | 79.41% | — |
| title_v.anm#0 | title/title_ver.png | 5 | 64×16 | 2,048 | 14 | 76.66% | — |

## 4．真正运行时的文件与依据

- `scene/loading/loading.go` 加载 `sig.anm`、`ascii.anm`；`scene/ui/title_controller.go` 加载 `title.anm`、`title_v.anm`、`ascii.anm`。菜单三页的 `scene/menus/native.go` 通过 PresentationANM 使用 `title.ant` 和 `text.anm`。
- `scene/stage/gui.go`／`gui_msg.go`／`gui_boss.go`／`menu.go` 使用 `front.anm`／`front.ant`、`text.anm`、`ascii.anm`、`st01logo`～`st07logo` 的原版和 ANT；`entity/player/anm.go` 按角色选择 `pl00.anm`／`pl01.anm`，子弹和灵击使用 `bullet.anm`。
- 关卡 ECL 的 ANIM 表加载 `enemy.anm` 和对应 `stgenmNN.anm`；STD 头部的 ANM 名加载 `stageNN.anm`。表格中给出全部实际引用。
- `scene/ending/ending.go`／`msg.go`：十二机体分支需要 `e00`～`e11.msg`／`.mst`／`.anm`；Good 的 op 12 切 Staff，加载 `staff.msg`／`.mst`／`.anm`／`.ant`。文件名引用来自原 MSG，不依据猜测删结局图。
- `audio/se.go` 的 `seWav` 表有 37 个文件，47 个 sound id 全部在此表映射；ECL 336、弹幕模板、SHT 和菜单发 sound id。37 个 SE 文件都由可调用的 id 覆盖，不能仅按某份回放出现的 id 删除。
- BGM 共 18 首；Music Room 可试听全部曲目，`audio.BGMTracks` 与 `misc/musiccmt.txt`／`musiccmtt.txt` 对应。四个 `misc/demo0`～`demo3.rpy` 是标题待机演示，实际运行时必需。
- 原版 ECL 由 `scene/stage/ecl/vm.go` 执行；`.ect` 只由 `gui_names.go` 读符卡名，不能替换逻辑。全套原版／繁体 MSG 均用于演出。六个 SHT 和七个 STD 均为实际资源。
- 原 `//go:embed anm ecl msg sht std ...` 会把 `assets/anm/sprite_report.md`、六个 `assets/ecl/stageNN.ecl.txt` 反汇编文本和全部预解码 PNG 一并带入。报告／反汇编文本应移到 `docs/research/`；17 篇 `assets/study` 笔记应移到 `docs/study/`。这些不是运行时资源。
- `render.LoadImage` 的关卡旧背景 PNG 路径和 `sprite.Load` 的 `bullet/etama.png` 可以由对应 ANM 的 THTX 组成；无需额外嵌入 PNG。前者属于 t16 将删除的旧绘制路径，本任务只保持兼容资源入口。未被任何运行时路径引用的合成标题／边框、旧立绘 PNG 不应嵌入。

### ECL／STD 的文件名引用

| 源文件 | 引用 |
|---|---|
| default.ecl | enemy.anm |
| stage01.ecl | enemy.anm, stgenm01.anm, default.ecl |
| stage02.ecl | enemy.anm, stgenm02.anm, default.ecl |
| stage03.ecl | enemy.anm, stgenm03.anm, default.ecl |
| stage04.ecl | enemy.anm, stgenm04.anm, default.ecl |
| stage05.ecl | enemy.anm, stgenm05.anm, default.ecl |
| stage06.ecl | enemy.anm, stgenm06.anm, default.ecl |
| stage07.ecl | enemy.anm, stgenm07.anm, default.ecl |
| stage01.std | stage01.anm |
| stage02.std | stage02.anm |
| stage03.std | stage03.anm |
| stage04.std | stage04.anm |
| stage05.std | stage05.anm |
| stage06.std | stage06.anm |
| stage07.std | stage07.anm |

### 结局／Staff MSG 的明文文件名引用

| 源文件 | 引用 |
|---|---|
| e00.msg | bgm/th10_13.wav, e00.anm |
| e01.msg | bgm/th10_13.wav, e01.anm |
| e02.msg | bgm/th10_13.wav, e02.anm |
| e03.msg | bgm/th10_13.wav, e03.anm |
| e04.msg | bgm/th10_13.wav, e04.anm |
| e05.msg | bgm/th10_13.wav, e05.anm |
| e06.msg | bgm/th10_13.wav, e06.anm, staff.msg |
| e07.msg | bgm/th10_13.wav, e07.anm, staff.msg |
| e08.msg | bgm/th10_13.wav, e08.anm, staff.msg |
| e09.msg | bgm/th10_13.wav, e09.anm, staff.msg |
| e10.msg | bgm/th10_13.wav, e10.anm, staff.msg |
| e11.msg | bgm/th10_13.wav, e11.anm, staff.msg |
| staff.msg | bgm/th10_14.wav, staff.anm |

## 5．BGM

18 首合计 2289.057 秒（38.151 分钟），Ogg 共 38,842,803 字节，按文件字节／原始时长算平均 135.75 kbps。全部为 Vorbis、双声道、44,100 Hz。

循环点在原样嵌入的 `wav/bgm/thbgm.fmt`。每条 52 字节：＋0x18 Intro、＋0x1c Size 为 16-bit stereo PCM 字节偏移，除 4 得采样帧；播放 `[0, Size)` 后重复 `[Intro, Size)`。＋0x14 Unknown 是原 dat 的保留／占位空间，不是播放终点。

现有提取器已经只用 `BGMTrack.PCM` 的 SectionReader 读取 Size 字节，因此当前 Ogg 没有保留循环终点之后的无用 PCM。不可把 thbgm.dat 的曲目间预留间隔算进音频或改变循环点。

| 曲目文件（fmt 顺序） | 秒 | Ogg 字节 | 实际 kbps | Intro 采样帧 | End 采样帧 | 原 dat 预留值 |
|---|---:|---:|---:|---:|---:|---:|
| th10_02.wav | 84.326 | 1,444,716 | 137.06 | 986,112 | 3,718,784 | 16,454,400 |
| th10_00.wav | 136.081 | 2,374,884 | 139.62 | 679,680 | 6,001,152 | 26,335,268 |
| th10_01.wav | 44.606 | 762,282 | 136.72 | 60,800 | 1,967,104 | 8,203,492 |
| th10_03.wav | 121.388 | 2,137,484 | 140.87 | 162,944 | 5,353,216 | 21,888,220 |
| th10_04.wav | 105.752 | 1,732,699 | 131.08 | 127,616 | 4,663,680 | 19,191,488 |
| th10_05.wav | 182.309 | 3,172,643 | 139.22 | 786,432 | 8,039,808 | 34,076,912 |
| th10_06.wav | 146.200 | 2,467,858 | 135.04 | 436,736 | 6,447,424 | 27,197,884 |
| th10_07.wav | 204.022 | 3,476,250 | 136.31 | 427,008 | 8,997,376 | 39,112,424 |
| th10_08.wav | 83.659 | 1,419,849 | 135.78 | 413,184 | 3,689,344 | 16,144,736 |
| th10_09.wav | 183.011 | 3,198,358 | 139.81 | 133,120 | 8,070,784 | 36,342,052 |
| th10_10.wav | 178.186 | 3,006,657 | 134.99 | 641,280 | 7,857,984 | 35,280,000 |
| th10_11.wav | 67.641 | 1,124,588 | 133.01 | 549,376 | 2,982,976 | 13,696,116 |
| th10_12.wav | 162.842 | 2,647,428 | 130.06 | 1,271,488 | 7,181,312 | 29,758,300 |
| th10_15.wav | 200.544 | 3,421,203 | 136.48 | 103,488 | 8,844,000 | 37,219,160 |
| th10_16.wav | 172.557 | 2,920,696 | 135.41 | 220,296 | 7,609,744 | 32,181,632 |
| th10_13.wav | 87.426 | 1,453,588 | 133.01 | 2,001,664 | 3,855,488 | 16,813,024 |
| th10_14.wav | 90.293 | 1,513,734 | 134.12 | 2,782,849 | 3,981,929 | 20,550,740 |
| th10_17.wav | 38.215 | 567,886 | 118.88 | 469,810 | 1,685,296 | 8,582,772 |

## 6．SE

37 个文件共 1,875,292 字节，均为 RIFF／WAVE PCM（format tag＝1），混用 8／16 bit、mono／stereo、11,025～44,100 Hz。运行时 `audio.Manager.loadSEPCM` 在首次加载时用 Ebiten WAV 解码／重采样，并缓存 16-bit stereo PCM；后续播放直接读 PCM。若改为压缩 SE，需要继续在加载时完成解码，不允许在播放时新增解码延迟。

| SE | 文件字节 | PCM 字节 | Hz | 声道 | 位数 |
|---|---:|---:|---:|---:|---:|
| se_bonus3.wav | 14,800 | 14,756 | 11025 | 1 | 16 |
| se_cancel00.wav | 13,652 | 13,608 | 22050 | 1 | 8 |
| se_cardget.wav | 222,864 | 222,820 | 22050 | 2 | 16 |
| se_cat00.wav | 84,260 | 84,104 | 22050 | 2 | 8 |
| se_ch00.wav | 87,564 | 87,422 | 22050 | 1 | 16 |
| se_ch01.wav | 60,286 | 60,126 | 44100 | 1 | 16 |
| se_damage00.wav | 6,772 | 6,614 | 22050 | 1 | 16 |
| se_damage01.wav | 27,028 | 26,872 | 22050 | 1 | 16 |
| se_enep00.wav | 115,584 | 115,456 | 44100 | 2 | 16 |
| se_enep01.wav | 230,300 | 230,144 | 22050 | 2 | 16 |
| se_extend.wav | 70,188 | 70,144 | 22050 | 2 | 8 |
| se_graze.wav | 4,542 | 4,410 | 22050 | 1 | 16 |
| se_gun00.wav | 134,694 | 134,536 | 22050 | 2 | 8 |
| se_hint00.wav | 5,420 | 5,376 | 22050 | 1 | 8 |
| se_invalid.wav | 62,950 | 62,824 | 22050 | 2 | 16 |
| se_item00.wav | 7,378 | 7,246 | 22050 | 1 | 16 |
| se_kira00.wav | 35,222 | 35,066 | 22050 | 1 | 8 |
| se_kira01.wav | 21,940 | 21,784 | 22050 | 1 | 8 |
| se_kira02.wav | 25,782 | 25,624 | 22050 | 1 | 16 |
| se_lazer00.wav | 22,182 | 22,050 | 22050 | 1 | 8 |
| se_lazer01.wav | 24,182 | 24,023 | 22050 | 1 | 8 |
| se_ok00.wav | 10,876 | 10,832 | 22050 | 1 | 8 |
| se_option.wav | 26,504 | 26,460 | 22050 | 1 | 16 |
| se_pause.wav | 133,676 | 133,632 | 22050 | 2 | 16 |
| se_pldead00.wav | 60,852 | 60,808 | 22050 | 2 | 8 |
| se_plst00.wav | 3,596 | 3,552 | 22050 | 1 | 8 |
| se_power0.wav | 88,052 | 87,422 | 22050 | 1 | 16 |
| se_power1.wav | 30,412 | 30,367 | 22050 | 1 | 8 |
| se_powerup.wav | 77,278 | 77,120 | 22050 | 2 | 16 |
| se_select00.wav | 1,146 | 1,102 | 22050 | 1 | 8 |
| se_slash.wav | 54,782 | 54,738 | 22050 | 1 | 16 |
| se_tan00.wav | 15,100 | 15,056 | 22050 | 1 | 8 |
| se_tan01.wav | 15,100 | 15,056 | 22050 | 1 | 8 |
| se_tan02.wav | 11,708 | 11,663 | 22050 | 1 | 8 |
| se_timeout.wav | 19,390 | 19,232 | 22050 | 1 | 16 |
| se_timeout2.wav | 19,388 | 19,232 | 22050 | 1 | 16 |
| se_water.wav | 29,842 | 29,798 | 44100 | 1 | 8 |

## 7．字体与文字绘制

- 原字体：Bimini Bold 46,488 字节／218 glyphs；DFYuGaSo-W3 4,189,740 字节、DFYuGaSo-W7 4,165,888 字节，各首 face 8,987 glyphs。TTC 的其他 face 没有调用者。
- 实际 ANM 文字路径是 `render.WriteVMText → RasterText`：CPU `golang.org/x/image/font/opentype`、72 DPI、HintingFull、先 `(1,1)` 黑影再画彩色字，基线取 Ascent.Ceil；尺寸、精灵表和 ANM VM 不变。MSG／结局／Staff／曲目说明／Player Data 使用 W3、16 px；符卡名 W3、默认 17 px，按原 SJIS 字节宽右对齐。颜色由调用点或 MSG 设置，不改变字形。
- W7 只在 `RasterText` 的 Bold 分支和 `LoadGameFace(..., true)` 中引用；当前场景调用没有 Bold＝true。为了保持字体加载／绘制 API 的样式能力，可以裁剪该首 face，而不是留下整个 TTC。
- `render.LoadTextFace`／`LoadHUDFace` 是 Ebiten text/v2 的 GoTextFace 路径。Bimini 是 ASCII HUD／旧 UI 的后备。旧 `scene/result` 的 DefaultTheme 用 LoadSystemFace（28／20／16 px）；主流程已经使用 ANM 菜单，小字来自 ascii.anm 精灵，不是字体。旧系统字体路径的清理需与 t16 协调，不能在字体瘦身中偷偷改变既有绘制器。

字符普查并集为 1939 个可打印 Unicode 字符。包括全部原版和繁体数据（原版用于后备）、95 个可打印 ASCII，以及 runtime Go 字符串字面量的保守闭包。统计不含 NUL／换行，不把 GBK／SJIS 原字节误当 Unicode。

| 来源 | 不去重到其他类别时的字符数 |
|---|---:|
| ASCII | 95 |
| Ending | 1043 |
| MSG | 1096 |
| Music | 833 |
| Source | 783 |
| Spell | 393 |

| 字体首 face | 实际集合映射到的唯一 glyph | 原字体无字形的字符数 |
|---|---:|---:|
| Bimini Bold.ttf | 95 | 0 |
| DFYuGaSo-W3.ttc | 1694 | 245 |
| DFYuGaSo-W7.ttc | 1694 | 245 |

W3／W7 对一些保守 Source 闭包中的简体字和少量繁体／符号本来就缺字；这些字符当前使用原 `.notdef`，不应在瘦身中悄悄补字或改变像素。裁剪继续保留同一个 `.notdef`；完整缺字清单写入字体产物清单。

Source 包含菜单、Music Room 未解锁警告、角色／关卡名及少量尚存的旧固定文案，属于保守上界，不能宣称其中每一个字符都会在正常流程被画出。完整有序字符集和分类集合见本普查配套 `docs/research/assets_characters.json`。字体裁剪测试必须覆盖所有集合、16／17 px、左右对齐与 Bold API；不会用截图容差掩盖变化。

## 8．现有编码器

`tools/th10extract/main.go:encodeBGM` 调用 `ffmpeg` 的 `libvorbis`，默认 `-q 4`，没有先重采样：

```text
ffmpeg -hide_banner -loglevel error -y -f s16le -ar 44100 -ac 2 -i - -c:a libvorbis -q:a 4 -map_metadata -1 -flags +bitexact -fflags +bitexact <out>.ogg
```

stdin 是 thbgm.dat 从 Offset 起、恰好 Size 字节的 PCM。完整 thbgm.fmt 原样复制，编码器延迟不能通过改循环点补偿。现有 `TestBGMLoopSeam` 和 `TestBGMMatchesThbgm` 已检验输出长度、循环接缝和原作采样对齐，后续降低码率仍须原样通过。

## 9．实施方向

桌面：ANM 逻辑原字节分段保存，THTX 像素去重、逐块比较 PNG／滤波 flate 实际大小。按 ANM 第一次加载时解码；图片索引不能通过启动时加载全部 ANM 来建立。WASM：只替换有收益的大幅手绘图编码，尺寸、格式、精灵坐标不变。字体：保留首 face 的实际字符及复合／排版依赖，保持 glyph id 和 hint 程序，避免改绘制器。BGM：先实测三个低质量档，保留 Ogg，SE 优先比较无损 flate 与原 WAV，避免新增播放延迟。

普查本身不改变运行时资源。阶段完成情况、后续实测、验收及需试听／目视项目写入 `docs/handoff/codex_t14.md`。

import { downloadWASM, integerSize, mebibytes, warmBootPack } from "./helpers.mjs";

const stage = document.getElementById("stage");
const frame = document.getElementById("game");
const status = document.getElementById("status");
const progress = document.getElementById("progress");
const transfer = document.getElementById("transfer");
let playerReady = false;
let downloaded = null;
let runtimeReady = false;
let running = false;
let busy = false;
let limits = { scoreLimit: 2 * 1048576, replayLimit: 16 * 1048576 };
const dataButtons = ["import-score", "export-score", "export-cfg", "import-replay", "export-replay"];
const storageStatus = document.getElementById("storage-status");
const assetStatus = document.getElementById("asset-status");
const slot = document.getElementById("slot");
for (let n = 1; n <= 25; n++) slot.add(new Option(String(n).padStart(2, "0"), String(n)));

function updateDataButtons() {
    for (const id of dataButtons) document.getElementById(id).disabled = !runtimeReady || busy || (running && id.startsWith("import"));
}

function report(message, error = false) {
    status.textContent = message;
    status.classList.toggle("error", error);
    console[error ? "error" : "info"]("[TH10]", message);
}

function resize() {
    const size = integerSize(stage.clientWidth, stage.clientHeight);
    frame.style.width = `${size.width}px`;
    frame.style.height = `${size.height}px`;
    document.getElementById("scale").textContent = `640 × 480 / ${size.scale}× / DPR ${window.devicePixelRatio}`;
}
new ResizeObserver(resize).observe(stage);
document.addEventListener("fullscreenchange", resize);
resize();

async function fullscreen() {
    try {
        if (document.fullscreenElement) await document.exitFullscreen();
        else await stage.requestFullscreen();
        resize();
        frame.contentWindow.focus();
        frame.contentWindow.postMessage({ type: "focus" }, location.origin);
    } catch (error) { report(`全屏失败：${error.message}`, true); }
}
document.getElementById("fullscreen").addEventListener("click", fullscreen);
document.getElementById("debug").addEventListener("click", () => frame.contentWindow.postMessage({ type: "debug" }, location.origin));

function launch() {
    if (!playerReady || !downloaded) return;
    const buffer = downloaded.buffer;
    downloaded = null;
    report("下载完成，初始化 WASM 并载入启动资源包…");
    frame.contentWindow.postMessage({ type: "wasm", buffer }, location.origin, [buffer]);
}

window.addEventListener("message", event => {
    if (event.source !== frame.contentWindow || event.origin !== location.origin) return;
    const message = event.data;
    if (message?.type === "player-ready") { playerReady = true; launch(); }
    else if (message?.type === "fullscreen") { void fullscreen(); }
    else if (message?.type === "status") { report(message.text, message.error); }
    else if (message?.type === "runtime-ready") {
        runtimeReady = true;
        limits = message.limits;
        updateDataButtons();
        document.getElementById("debug").disabled = false;
        report("初始化完成。可先导入存档／回放，再点击游戏内“点击开始”。");
    }
    else if (message?.type === "started") {
        running = true;
        updateDataButtons();
        report("音频已解锁，游戏第一帧开始。点击画布获得键盘焦点。");
    }
    else if (message?.type === "storage") {
        const e = message.event;
        if (e.error) { storageStatus.textContent = `存储失败，未确认保存：${e.error}`; report(storageStatus.textContent, true); }
        else if (e.operation === "write") storageStatus.textContent = `已提交 ${e.key}（${e.bytes} 字节）；同一网址重新载入可读取。请定期导出备份。`;
        else if (e.operation === "open") storageStatus.textContent = "IndexedDB 已连接。导入只允许在点击开始前；请选择槽位后导入／导出回放。";
    }
    else if (message?.type === "assets") {
        const e = message.event;
        assetStatus.textContent = e.loadedPacks >= e.totalPacks
            ? `资源包全部就绪（${e.totalPacks} 个，${mebibytes(e.totalBytes)} MiB）`
            : `资源包 ${e.loadedPacks}/${e.totalPacks} · ${mebibytes(e.loadedBytes)} / ${mebibytes(e.totalBytes)} MiB（后台预取，最近：${e.pack}）`;
    }
    else if (message?.type === "debug-state") { document.getElementById("debug").setAttribute("aria-pressed", String(Boolean(message.enabled))); }
    else if (message?.type === "metrics") {
        const m = message.metrics;
        transfer.textContent = `${m.tps.toFixed(1)} TPS / ${m.fps.toFixed(1)} FPS · Update ${m.updateLastMs.toFixed(3)} ms · Draw ${m.drawLastMs.toFixed(3)} ms`;
    }
    else if (message?.type === "exited") { runtimeReady = false; updateDataButtons(); document.getElementById("debug").disabled = true; }
});
frame.addEventListener("load", () => frame.contentWindow.postMessage({ type: "hello" }, location.origin));
frame.contentWindow.postMessage({ type: "hello" }, location.origin);

async function load() {
    try {
        if (location.protocol === "file:") throw new Error("请通过 wasmbuild -serve 预览，不要直接打开本地 HTML。");
        void warmBootPack();
        const response = await fetch("release.json", { cache: "no-cache" });
        if (!response.ok) throw new Error(`发布清单 HTTP ${response.status}`);
        const release = await response.json();
        report(`下载 WASM（gzip 约 ${mebibytes(release.gzipBytes)} MiB；进度按解压后字节计）…`);
        downloaded = await downloadWASM(`th10.wasm?v=${encodeURIComponent(release.wasmSHA256)}`, release.wasmBytes, (bytes, total) => {
            progress.value = bytes / total;
            transfer.textContent = `${mebibytes(bytes)} / ${mebibytes(total)} MiB · ${Math.floor(bytes * 100 / total)}%`;
        });
        launch();
    } catch (error) { report(`启动失败：${error.message}`, true); }
}
void load();

async function importFile(kind, file) {
    if (!file) return;
    try {
        const limit = kind === "score" ? limits.scoreLimit : limits.replayLimit;
        if (file.size > limit) throw new Error(`文件超过 ${mebibytes(limit)} MiB 上限，未读取也未覆盖。`);
        const target = kind === "score" ? "现有存档" : `回放槽 ${slot.value}`;
        if (!confirm(`导入 ${file.name} 将替换${target}。建议先导出备份；只有格式校验通过才会写入。继续？`)) return;
        busy = true;
        updateDataButtons();
        report(`解析原作格式并提交${target}…`);
        // Use the iframe's typed-array realm for syscall/js's Uint8Array check.
        const data = new frame.contentWindow.Uint8Array(await file.arrayBuffer());
        await frame.contentWindow.th10.importFile(kind, Number(slot.value), data);
        report(`已导入 ${file.name}；原字节已提交，点击开始即可使用。`);
    } catch (error) { report(`导入被拒绝，现有数据未被无效文件覆盖：${error.message}`, true); }
    finally { busy = false; updateDataButtons(); }
}
document.getElementById("import-score").addEventListener("click", () => document.getElementById("score-file").click());
document.getElementById("import-replay").addEventListener("click", () => document.getElementById("replay-file").click());
for (const [id, kind] of [["score-file", "score"], ["replay-file", "replay"]]) {
    document.getElementById(id).addEventListener("change", event => {
        void importFile(kind, event.target.files[0]);
        event.target.value = "";
    });
}

async function exportFile(kind) {
    try {
        busy = true;
        updateDataButtons();
        const result = await frame.contentWindow.th10.exportFile(kind, Number(slot.value));
        const url = URL.createObjectURL(new Blob([result.data], { type: "application/octet-stream" }));
        const link = document.createElement("a");
        link.href = url;
        link.download = result.name;
        document.body.append(link);
        link.click();
        link.remove();
        setTimeout(() => URL.revokeObjectURL(url), 60000);
        report(`已导出 ${result.name}（${result.data.byteLength} 字节）；导出不会额外写存档。`);
    } catch (error) { report(`导出失败：${error.message}`, true); }
    finally { busy = false; updateDataButtons(); }
}
for (const [id, kind] of [["export-score", "score"], ["export-replay", "replay"], ["export-cfg", "cfg"]]) {
    document.getElementById(id).addEventListener("click", () => void exportFile(kind));
}
document.getElementById("persist").addEventListener("click", async () => {
    try {
        if (!navigator.storage?.persist) throw new Error("浏览器未提供 StorageManager.persist；请用导出文件备份。");
        const granted = await navigator.storage.persist();
        storageStatus.textContent = granted ? "浏览器已同意保留此网站数据；清除网站数据仍会删除存档。" : "浏览器未同意长期保留；当前 IndexedDB 可用，但请定期导出备份。";
    } catch (error) { storageStatus.textContent = error.message; }
});

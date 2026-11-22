import { preventGameDefault, isStartKey, installAudioQueue } from "./helpers.mjs";

// Before Go creates its audio context: a deeper worklet queue rides out
// heavy frames instead of dropping audio (see AUDIO_QUEUE_FRAMES).
installAudioQueue(globalThis.AudioWorklet);

const boot = document.getElementById("boot");
const GAME_CANVAS = "canvas:not(#loading)"; // Ebiten's canvas, not the loading screen's
const start = document.getElementById("start");
let launched = false;
let ready = false;
let running = false;

function post(message) { parent.postMessage(message, location.origin); }
function report(text, error = false) { post({ type: "status", text, error }); }

window.th10StorageEvent = event => post({ type: "storage", event });
window.th10Fatal = text => report(`运行时错误：${text}`, true);
window.th10RuntimeReady = limits => {
    ready = true;
    document.getElementById("boot-text").textContent = "";
    document.getElementById("boot-note").textContent = "点击开始前可导入存档；此时游戏尚未运行第一帧。";
    start.disabled = false;
    boot.classList?.add("ready");
    document.querySelector(GAME_CANVAS)?.addEventListener("pointerdown", event => event.target.focus({ preventScroll: true }));
    post({ type: "runtime-ready", limits });
};
window.th10Started = () => {
    running = true;
    boot.hidden = true;
    document.querySelector(GAME_CANVAS)?.focus({ preventScroll: true });
    post({ type: "started" });
};
window.th10Metrics = metrics => post({ type: "metrics", metrics });
window.th10AssetProgress = event => {
    if (event.error) { report(`资源包 ${event.pack} 下载失败：${event.error}`, true); return; }
    post({ type: "assets", event });
    if (!ready) document.getElementById("boot-text").textContent = `资源 ${event.pack}…`;
};
function toggleDebug() {
    if (!ready) return;
    const enabled = window.th10?.toggleDebug();
    post({ type: "debug-state", enabled });
    report(`调试叠加层${enabled ? "已开启" : "已关闭"}；耗时是 Update／Draw 调用墙钟时间，不是 GPU 完成时间。`);
}
function requestStart() {
    if (!ready || running) return;
    const error = window.th10?.start();
    if (error) report(`暂不能开始：${error}`, true);
    else document.getElementById("boot-note").textContent = "等待音频 resume；若仍停在这里，请再次点击或按键。";
}
boot.addEventListener("click", requestStart); // #start's clicks bubble here too // the whole loading screen; browsers need one real gesture for audio
document.addEventListener("keyup", event => {
    if (ready && !running && isStartKey(event)) requestStart();
});

window.addEventListener("error", event => report(`游戏错误：${event.message}`, true));
window.addEventListener("unhandledrejection", event => report(`游戏异步错误：${event.reason}`, true));
window.addEventListener("keydown", event => {
    if (preventGameDefault(event)) event.preventDefault();
    if (event.code === "F3") {
        event.preventDefault();
        if (!event.repeat) toggleDebug();
        return;
    }
    if (event.code === "KeyF" && !event.ctrlKey && !event.altKey && !event.metaKey) {
        event.preventDefault();
        if (!event.repeat) post({ type: "fullscreen" });
    }
}, { capture: true });

window.addEventListener("message", async event => {
    if (event.source !== parent || event.origin !== location.origin) return;
    if (event.data?.type === "hello") { post({ type: "player-ready" }); return; }
    if (event.data?.type === "debug") { toggleDebug(); return; }
    if (event.data?.type === "focus") { document.querySelector(GAME_CANVAS)?.focus({ preventScroll: true }); return; }
    if (event.data?.type !== "wasm" || launched) return;
    launched = true;
    try {
        if (!(event.data.buffer instanceof ArrayBuffer)) throw new Error("Missing WASM bytes");
        document.getElementById("boot-text").textContent = "初始化…";
        const go = new Go();
        go.exit = code => {
            ready = false;
            post({ type: "exited" });
            report(code === 0 ? "游戏已正常退出并提交存档／cfg；重新载入可再次开始。" : `Go 退出码 ${code}；查看控制台并重新载入。`, code !== 0);
        };
        const result = await WebAssembly.instantiate(event.data.buffer, go.importObject);
        const memory = result.instance.exports.mem;
        window.th10WASMMemoryBytes = () => memory.buffer.byteLength;
        // The long-running Go promise resolves only when the game exits.
        void go.run(result.instance).catch(error => report(`WASM 运行失败：${error.message}`, true));
    } catch (error) { report(`WASM 初始化失败：${error.message}`, true); }
});
post({ type: "player-ready" });

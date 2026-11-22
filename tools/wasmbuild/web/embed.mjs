// 只有游戏画面的宿主页（供博客等页面用 iframe 嵌入）：下载 WASM 交给 game.html，
// 按整数倍铺满窗口，F 键全屏。下载进度、存档导入导出等完整界面见 index.html + app.mjs。
import { downloadWASM, integerSize, warmBootPack } from "./helpers.mjs";

const frame = document.getElementById("game");
let playerReady = false;
let downloaded = null;

const post = (message, transfer) => frame.contentWindow.postMessage(message, location.origin, transfer);

function resize() {
    const size = integerSize(innerWidth, innerHeight);
    frame.style.width = `${size.width}px`;
    frame.style.height = `${size.height}px`;
}
addEventListener("resize", resize);
resize();

function launch() {
    if (!playerReady || !downloaded) return;
    const buffer = downloaded.buffer;
    downloaded = null;
    post({ type: "wasm", buffer }, [buffer]);
}

async function fullscreen() {
    try {
        if (document.fullscreenElement) await document.exitFullscreen();
        else await document.documentElement.requestFullscreen();
        frame.contentWindow.focus();
        post({ type: "focus" });
    } catch (error) { console.error("[TH10]", error); }
}

addEventListener("message", event => {
    if (event.source !== frame.contentWindow || event.origin !== location.origin) return;
    const message = event.data;
    if (message?.type === "player-ready") { playerReady = true; launch(); }
    else if (message?.type === "fullscreen") void fullscreen();
    else if (message?.type === "status") console[message.error ? "error" : "info"]("[TH10]", message.text);
});
frame.addEventListener("load", () => post({ type: "hello" }));
post({ type: "hello" });

void warmBootPack();
try {
    const response = await fetch("release.json", { cache: "no-cache" });
    if (!response.ok) throw new Error(`发布清单 HTTP ${response.status}`);
    const release = await response.json();
    downloaded = await downloadWASM(`th10.wasm?v=${encodeURIComponent(release.wasmSHA256)}`, release.wasmBytes, () => {});
    launch();
} catch (error) { console.error("[TH10] 启动失败：", error); }

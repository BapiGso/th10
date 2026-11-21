export function integerSize(width, height) {
    const available = Math.min(width / 640, height / 480);
    const scale = Number.isFinite(available) ? Math.max(1, Math.floor(available)) : 1;
    return { scale, width: 640 * scale, height: 480 * scale };
}

export function mebibytes(bytes) {
    return (bytes / 1048576).toFixed(1);
}

const gameKeys = new Set([
    "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "Space", "Enter", "Escape", "Home",
    "KeyZ", "KeyX", "KeyR", "KeyQ", "KeyS", "KeyP", "KeyD", "KeyF", "F3",
    "ShiftLeft", "ShiftRight", "ControlLeft", "ControlRight",
    ...Array.from({ length: 10 }, (_, n) => `Numpad${n}`),
    ...Array.from({ length: 9 }, (_, n) => `Digit${n + 1}`)
]);

export function preventGameDefault(event) { return gameKeys.has(event.code); }
export function isStartKey(event) { return gameKeys.has(event.code) && !["KeyF", "F3", "ControlLeft", "ControlRight"].includes(event.code) && !event.ctrlKey && !event.altKey && !event.metaKey; }

export async function downloadWASM(url, expectedBytes, onProgress, fetcher = fetch) {
    if (!Number.isSafeInteger(expectedBytes) || expectedBytes <= 0 || expectedBytes > 512 * 1048576) {
        throw new Error("Invalid release size; rebuild dist with wasmbuild.");
    }
    const response = await fetcher(url, { cache: "no-cache" });
    if (!response.ok) throw new Error(`WASM download failed: HTTP ${response.status}`);
    onProgress(0, expectedBytes);
    if (!response.body?.getReader) {
        const data = new Uint8Array(await response.arrayBuffer());
        if (data.byteLength !== expectedBytes) throw new Error("WASM length differs from release.json.");
        onProgress(data.byteLength, expectedBytes);
        return data;
    }
    // Fetch transparently decompresses gzip. Content-Length is wire size, not this size.
    const data = new Uint8Array(expectedBytes);
    const reader = response.body.getReader();
    let offset = 0;
    try {
        for (;;) {
            const { done, value } = await reader.read();
            if (done) break;
            if (offset + value.byteLength > expectedBytes) throw new Error("WASM is larger than release.json.");
            data.set(value, offset);
            offset += value.byteLength;
            onProgress(offset, expectedBytes);
        }
        if (offset !== expectedBytes) throw new Error("WASM download was truncated; rebuild or retry.");
        return data;
    } catch (error) {
        await reader.cancel(error).catch(() => {});
        throw error;
    } finally {
        reader.releaseLock();
    }
}

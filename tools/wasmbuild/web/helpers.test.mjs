import test from "node:test";
import assert from "node:assert/strict";
import { downloadWASM, integerSize, preventGameDefault, isStartKey, warmBootPack, enlargeOtoWorklet, installAudioQueue } from "./helpers.mjs";
import { resolveObjectURL } from "node:buffer";

test("integer viewport never uses fractional logical scaling", () => {
    assert.deepEqual(integerSize(1920, 1080), { scale: 2, width: 1280, height: 960 });
    assert.deepEqual(integerSize(2560, 1920), { scale: 4, width: 2560, height: 1920 });
    assert.deepEqual(integerSize(375, 650), { scale: 1, width: 640, height: 480 });
    assert.equal(integerSize(0, 0).scale, 1);
    assert.equal(integerSize(NaN, 800).scale, 1);
});

function streamed(chunks, status = 200) {
    return async () => new Response(new ReadableStream({
        start(controller) {
            chunks.forEach(chunk => controller.enqueue(new Uint8Array(chunk)));
            controller.close();
        }
    }), { status, headers: { "Content-Encoding": "gzip", "Content-Length": "2" } });
}

test("download progress counts decoded bytes, not compressed Content-Length", async () => {
    const progress = [];
    const data = await downloadWASM("test.wasm", 5, (n, total) => progress.push([n, total]), streamed([[0, 1], [2, 3, 4]]));
    assert.deepEqual([...data], [0, 1, 2, 3, 4]);
    assert.deepEqual(progress, [[0, 5], [2, 5], [5, 5]]);
});

test("download rejects overflow, truncation, HTTP errors and invalid manifests", async () => {
    await assert.rejects(downloadWASM("x", 1, () => {}, streamed([[1, 2]])), /larger/);
    await assert.rejects(downloadWASM("x", 3, () => {}, streamed([[1, 2]])), /truncated/);
    await assert.rejects(downloadWASM("x", 3, () => {}, streamed([[1]], 404)), /HTTP 404/);
    await assert.rejects(downloadWASM("x", 0, () => {}, streamed([])), /Invalid release/);
    await assert.rejects(downloadWASM("x", 513 * 1048576, () => {}, streamed([])), /Invalid release/);
});

test("non-streaming fallback still validates the exact size", async () => {
    const fetcher = async () => ({ ok: true, body: null, arrayBuffer: async () => new Uint8Array([1, 2]).buffer });
    assert.deepEqual([...(await downloadWASM("x", 2, () => {}, fetcher))], [1, 2]);
    await assert.rejects(downloadWASM("x", 3, () => {}, fetcher), /length differs/);
});

test("game keys prevent scroll/find/reload defaults; host shortcuts do not start play", () => {
    for (const code of ["ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "Space", "KeyF", "F3", "KeyR"]) {
        assert.equal(preventGameDefault({ code }), true);
    }
    assert.equal(preventGameDefault({ code: "KeyF", ctrlKey: true }), true);
    assert.equal(isStartKey({ code: "KeyZ" }), true);
    assert.equal(isStartKey({ code: "Space" }), true);
    assert.equal(isStartKey({ code: "KeyF" }), false);
    assert.equal(isStartKey({ code: "F3" }), false);
    assert.equal(isStartKey({ code: "KeyR", ctrlKey: true }), false);
});

test("boot pack warm-up reads the manifest's boot pack and never throws", async () => {
    const requests = [];
    const fetcher = async url => {
        requests.push(url);
        if (url === "assets/manifest.json") return Response.json({ packs: { boot: { file: "boot.0123.pak" } } });
        return new Response(new Uint8Array([1, 2, 3]));
    };
    assert.equal(await warmBootPack(fetcher), true);
    assert.deepEqual(requests, ["assets/manifest.json", "assets/boot.0123.pak"]);
    assert.equal(await warmBootPack(async () => new Response("", { status: 404 })), false);
    assert.equal(await warmBootPack(async () => { throw new Error("offline"); }), false);
});

const otoWorklet = "class OtoWorkletProcessor extends AudioWorkletProcessor {\n\tconstructor() {\n\t\tsuper();\n\t\tthis.bufferSize_ = 2048;\n\t}\n}\nregisterProcessor('oto-worklet-processor', OtoWorkletProcessor);\n";

test("oto worklet queue is enlarged, and unknown modules are left alone", () => {
    assert.match(enlargeOtoWorklet(otoWorklet, 8192), /this\.bufferSize_ = 8192;/);
    assert.equal(enlargeOtoWorklet("registerProcessor('other', X);"), null);
    assert.equal(enlargeOtoWorklet(otoWorklet.replace("this.bufferSize_ = 2048;", "")), null);
});

test("installed audio queue loads the patched module and falls back on failure", async () => {
    const loaded = [];
    class FakeWorklet { async addModule(url) { loaded.push(url.startsWith("blob:") ? await resolveObjectURL(url).text() : url); } }
    assert.equal(installAudioQueue(FakeWorklet, async () => new Response(otoWorklet), 6000), true);
    await new FakeWorklet().addModule("blob:oto");
    assert.match(loaded[0], /this\.bufferSize_ = 6000;/);

    class Other { async addModule(url) { loaded.push(url); } }
    const warn = console.warn;
    console.warn = () => {};
    try {
        installAudioQueue(Other, async () => { throw new Error("offline"); });
        await new Other().addModule("blob:x");
    } finally { console.warn = warn; }
    assert.equal(loaded[1], "blob:x");
    assert.equal(installAudioQueue(undefined), false);
});

import test from "node:test";
import assert from "node:assert/strict";
import http from "node:http";

function request(base, path, method = "GET", headers = {}) {
    return new Promise((resolve, reject) => {
        const req = http.request(new URL(path, base), { method, headers }, response => {
            const chunks = [];
            response.on("data", chunk => chunks.push(chunk));
            response.on("end", () => resolve({ status: response.statusCode, headers: response.headers, body: Buffer.concat(chunks) }));
        });
        req.on("error", reject);
        req.end();
    });
}

test("live preview exposes raw gzip MIME/length and identity ranges", { skip: !process.env.TH10_PREVIEW_URL }, async () => {
    const base = new URL(process.env.TH10_PREVIEW_URL);
    assert.ok(["127.0.0.1", "localhost", "[::1]"].includes(base.hostname), "preview tests are loopback-only");
    const release = JSON.parse((await request(base, "/release.json")).body);
    for (const [path, headers, encoding, size] of [
        ["/th10.wasm", { "Accept-Encoding": "identity" }, undefined, release.wasmBytes],
        ["/th10.wasm", { "Accept-Encoding": "gzip" }, "gzip", release.gzipBytes],
        ["/th10.wasm.gz", {}, "gzip", release.gzipBytes]
    ]) {
        const response = await request(base, path, "HEAD", headers);
        assert.equal(response.status, 200);
        assert.equal(response.headers["content-type"], "application/wasm");
        assert.equal(response.headers["content-encoding"], encoding);
        assert.equal(Number(response.headers["content-length"]), size);
        assert.equal(response.headers.vary, "Accept-Encoding");
        console.info(path, encoding ?? "identity", size, "bytes");
    }
    const range = await request(base, "/th10.wasm", "GET", { "Accept-Encoding": "gzip", Range: "bytes=0-3" });
    assert.equal(range.status, 206);
    assert.equal(range.headers["content-encoding"], undefined);
    assert.deepEqual(range.body, Buffer.from([0, 97, 115, 109]));
});

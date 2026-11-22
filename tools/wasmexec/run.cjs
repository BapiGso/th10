// Development-only Node executor. This file is never included in dist.
"use strict";

const fs = require("node:fs");
const path = require("node:path");
const { execFileSync } = require("node:child_process");

if (process.argv.length < 3) {
    console.error("usage: node --stack-size=8192 tools/wasmexec/run.cjs binary.wasm [arguments]");
    process.exit(2);
}
const temp = process.env.TMPDIR || process.env.GOTMPDIR;
if (!temp) throw new Error("Set TMPDIR or GOTMPDIR to a workspace directory before running WASM.");

globalThis.require = require;
globalThis.fs = fs;
globalThis.path = path;
globalThis.TextEncoder ??= require("node:util").TextEncoder;
globalThis.TextDecoder ??= require("node:util").TextDecoder;
globalThis.performance ??= require("node:perf_hooks").performance;
globalThis.crypto ??= require("node:crypto").webcrypto;

if (process.env.TH10_WASM_FAKE_IDB === "1") require("./testdata/indexeddb.cjs");
if (process.env.TH10_WASM_HEADLESS === "1") require("./headless.cjs");
// th10web asset packs: serve fetch(relative URL) from a built dist directory.
if (process.env.TH10_WASM_ASSET_DIR) {
    const root = path.resolve(process.env.TH10_WASM_ASSET_DIR);
    globalThis.fetch = async (url, options = {}) => {
        const file = path.resolve(root, String(url).split("?")[0]);
        if (!file.startsWith(root + path.sep) || !fs.existsSync(file)) return { ok: false, status: 404 };
        const data = fs.readFileSync(file);
        // Subresource Integrity, as a browser checks it: a mismatch rejects.
        if (options.integrity && options.integrity !== "sha256-" + require("node:crypto").createHash("sha256").update(data).digest("base64")) {
            throw new TypeError(`integrity mismatch for ${url}`);
        }
        return { ok: true, status: 200, arrayBuffer: async () => data.buffer.slice(data.byteOffset, data.byteOffset + data.byteLength) };
    };
}

const goroot = process.env.GOROOT || execFileSync("go", ["env", "GOROOT"], { encoding: "utf8" }).trim();
const runtime = ["lib", "misc"].map(dir => path.join(goroot, dir, "wasm", "wasm_exec.js")).find(file => fs.existsSync(file));
if (!runtime) throw new Error(`wasm_exec.js not found in ${goroot}`);
require(runtime);

const go = new Go();
go.argv = process.argv.slice(2);
go.env = { ...process.env, TMPDIR: temp };
go.exit = process.exit;
let memory;
globalThis.th10WASMMemoryBytes = () => memory?.buffer.byteLength ?? 0;
WebAssembly.instantiate(fs.readFileSync(process.argv[2]), go.importObject).then(result => {
    memory = result.instance.exports.mem;
    process.on("exit", code => {
        if (code === 0 && !go.exited) {
            go._pendingEvent = { id: 0 };
            go._resume();
        }
    });
    return go.run(result.instance);
}).catch(error => {
    console.error(error);
    process.exit(1);
});

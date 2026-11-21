// Exact streaming byte comparison, with first differing frame/line context.
"use strict";
const fs = require("node:fs");

if (process.argv.length !== 4) {
    console.error("usage: node tools/wasmexec/compare.cjs native.txt wasm.txt");
    process.exit(2);
}
const descriptors = process.argv.slice(2).map(file => fs.openSync(file, "r"));
const sizes = descriptors.map(fd => fs.fstatSync(fd).size);
const buffers = [Buffer.alloc(1048576), Buffer.alloc(1048576)];
let offset = 0;
let different = -1;
try {
    while (offset < Math.max(...sizes)) {
        const counts = descriptors.map((fd, n) => fs.readSync(fd, buffers[n], 0, buffers[n].length, offset));
        const common = Math.min(...counts);
        if (!buffers[0].subarray(0, common).equals(buffers[1].subarray(0, common))) {
            let at = 0;
            while (at < common && buffers[0][at] === buffers[1][at]) at++;
            different = offset + at;
            break;
        }
        if (counts[0] !== counts[1]) { different = offset + common; break; }
        offset += common;
    }
    if (different < 0) {
        console.log(JSON.stringify({ equal: true, bytes: sizes[0] }));
    } else {
        const context = descriptors.map((fd, n) => {
            const begin = Math.max(0, different - 131072);
            const buffer = Buffer.alloc(Math.min(262144, sizes[n] - begin));
            const count = fs.readSync(fd, buffer, 0, buffer.length, begin);
            const text = buffer.subarray(0, count).toString("utf8");
            const at = different - begin;
            const preceding = text.slice(0, at);
            const frames = [...preceding.matchAll(/(?:^|\n)F (\d+) /g)];
            const lineBegin = preceding.lastIndexOf("\n") + 1;
            const end = text.indexOf("\n", at);
            return { frame: Number(frames.at(-1)?.[1] ?? 0), line: text.slice(lineBegin, end < 0 ? undefined : end) };
        });
        console.log(JSON.stringify({ equal: false, offset: different, sizes, native: context[0], wasm: context[1] }));
        process.exitCode = 1;
    }
} finally {
    descriptors.forEach(fd => fs.closeSync(fd));
}

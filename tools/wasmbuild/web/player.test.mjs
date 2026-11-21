import test from "node:test";
import assert from "node:assert/strict";

test("page gate waits for Go's first-frame signal and isolates shortcuts", async t => {
    const globals = ["window", "document", "parent", "location"];
    const saved = globals.map(name => Object.getOwnPropertyDescriptor(globalThis, name));
    t.after(() => globals.forEach((name, i) => {
        if (saved[i]) Object.defineProperty(globalThis, name, saved[i]);
        else delete globalThis[name];
    }));
    const windowEvents = new Map();
    const documentEvents = new Map();
    const messages = [];
    let startCalls = 0;
    let focused = 0;
    let debug = false;
    const elements = new Map();
    for (const name of ["boot", "start", "boot-text", "boot-note"]) {
        elements.set(name, { hidden: false, disabled: true, addEventListener() {} });
    }
    const canvas = { focus() { focused++; }, addEventListener() {} };
    globalThis.window = {
        addEventListener(name, body) { windowEvents.set(name, body); },
        th10: { start() { startCalls++; return ""; }, toggleDebug() { debug = !debug; return debug; } }
    };
    globalThis.document = {
        getElementById(name) { return elements.get(name); },
        querySelector() { return canvas; },
        addEventListener(name, body) { documentEvents.set(name, body); }
    };
    globalThis.location = { origin: "http://127.0.0.1:8080" };
    globalThis.parent = { postMessage(message) { messages.push(message); } };
    await import(`./player.mjs?test=${Date.now()}`);

    documentEvents.get("keyup")({ code: "KeyZ" });
    assert.equal(startCalls, 0);
    window.th10RuntimeReady({ scoreLimit: 1, replayLimit: 2 });
    documentEvents.get("keyup")({ code: "KeyZ" });
    assert.equal(startCalls, 1);
    assert.equal(elements.get("boot").hidden, false, "requesting start must not hide the gate early");

    let prevented = false;
    windowEvents.get("keydown")({ code: "F3", repeat: false, preventDefault() { prevented = true; } });
    assert.equal(prevented, true);
    assert.equal(debug, true);
    documentEvents.get("keyup")({ code: "F3" });
    assert.equal(startCalls, 1, "debug shortcut must not advance the game");
    assert.equal(messages.some(message => message.type === "started"), false);

    window.th10Started();
    assert.equal(elements.get("boot").hidden, true);
    assert.equal(focused, 1);
    documentEvents.get("keyup")({ code: "KeyZ" });
    assert.equal(startCalls, 1, "gameplay keyup restarted the runtime");
    await windowEvents.get("message")({ source: parent, origin: "https://invalid.example", data: { type: "debug" } });
    assert.equal(debug, true, "foreign messages were accepted");
});

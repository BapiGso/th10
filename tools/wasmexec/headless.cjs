// Init-only DOM facade for Ebiten packages used by logic/audio tools in Node.
// Deliberately no scheduled frames or WebGL renderer: RunGame must not work.
"use strict";

class Element {
    constructor() {
        this.style = {};
        this.clientWidth = 640;
        this.clientHeight = 480;
    }
    appendChild() {}
    setAttribute() {}
    addEventListener() {}
    removeEventListener() {}
    focus() {}
    getContext() { throw new Error("headless Node tools must never initialize WebGL"); }
}
class Document {
    constructor() {
        this.body = new Element();
        this.head = new Element();
        this.documentElement = new Element();
    }
    get hidden() { return false; }
    hasFocus() { return true; }
    createElement() { return new Element(); }
    addEventListener() {}
    removeEventListener() {}
}
globalThis.window = { addEventListener() {}, removeEventListener() {}, devicePixelRatio: 1 };
globalThis.Document = Document;
globalThis.document = new Document();
globalThis.screen = { width: 640, height: 480, availWidth: 640, availHeight: 480 };
globalThis.navigator ??= {};
// Ebiten's internal/ui (ui_js.go) looks up requestAnimationFrame at package
// init. Headless tools never run a frame, so the stub schedules nothing.
globalThis.requestAnimationFrame = () => 0;
console.error("[th10 wasmexec] init-only DOM; no WebGL, RunGame, or audio device");

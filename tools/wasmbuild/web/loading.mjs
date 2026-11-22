// The original's loading screen ("少女祈祷中"), drawn while the WASM downloads
// and until Go's first frame: sig.jpg is sig.anm scripts 0/1, loading.png
// ascii.anm's loading sheet, and the marker follows ascii.anm script 6 at
// (480, 392) with its children 3, 4 and 5, frame by frame at 60 Hz. Go's own
// loading scene then plays the same screen again from black.

const SPRITES = { // ascii.anm sprites 232-234 on loading.png
    shoujo: [1, 1, 126, 30],
    now: [1, 33, 126, 30],
    leaf: [1, 65, 30, 30]
};
const ORIGIN = [480, 392]; // FUN_0040c540 / scene/loading: ascii script 6 offset

// zun.Ease modes 0 (linear), 4 (ease out) and 9 (ease in-out), FUN_0044c350.
function ease(mode, t) {
    if (mode === 4) return 1 - (1 - t) * (1 - t);
    if (mode === 9) {
        const u = t + t;
        return u >= 1 ? (2 - (2 - u) * (2 - u)) * 0.5 : u * u * 0.5;
    }
    return t;
}
const tween = (from, to, frame, frames, mode = 0) => from + (to - from) * ease(mode, Math.min(Math.max(frame / frames, 0), 1));

// Script 3 / 4: fade in, then an endless 255 <-> 128 alpha pulse (mode 9).
function pulse(frame, start, half) {
    if (frame < start) return 255;
    const at = (frame - start) % (half * 2);
    return at < half ? tween(255, 128, at, half, 9) : tween(128, 255, at - half, half, 9);
}
const shoujoAlpha = c => c < 8 ? tween(0, 255, c, 8) : pulse(c, 40, 20);
const nowAlpha = c => c < 8 ? 0 : c < 16 ? tween(0, 255, c - 8, 8) : pulse(c, 68, 30);
const nowX = c => c < 8 ? 128 : tween(128, 48, c - 8, 4, 4);

// Script 5, one leaf: F0..F3 from the signed RNG (10012), 60 frames long.
function newLeaf() {
    const r = () => Math.random() * 2 - 1;
    const x0 = r() * 96 + 16, y0 = r() * 16 - 32;
    const x1 = r() * 32 + x0, y1 = r() * 16 + 48;
    const scale = r() * 0.8 + 0.2;
    const spin = r() * Math.PI / 32;
    const angle = r() * Math.PI;
    const grey = ((Math.trunc(r() * 192) + 63) & 255) / 255; // op 18 then op 52, uint8
    return { age: 0, x0, y0, x1, y1, scale, spin, angle, grey };
}

export function startLoading(canvas, boot) {
    const ctx = canvas.getContext("2d");
    const sig = new Image();
    const sheet = new Image();
    sig.src = "sig.jpg";
    sheet.src = "loading.png";
    let frame = 0;
    let last = performance.now();
    let leaves = [];
    const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches;

    function sprite(name, x, y, alpha, scale = 1, angle = 0) {
        const [sx, sy, w, h] = SPRITES[name];
        ctx.save();
        ctx.globalAlpha = alpha;
        ctx.translate(x, y);
        if (angle) ctx.rotate(angle);
        if (scale !== 1) ctx.scale(scale, scale);
        ctx.drawImage(sheet, sx, sy, w, h, -w / 2, -h / 2, w, h);
        ctx.restore();
    }

    function step() {
        frame++;
        if (frame >= 2 && frame % 2 === 0 && !reduced) leaves.unshift(newLeaf()); // script 6: op 92 at t=1, every 2 frames
        for (const leaf of leaves) leaf.age++;
        leaves = leaves.filter(leaf => leaf.age < 60);
    }

    function draw() {
        ctx.globalCompositeOperation = "source-over";
        ctx.globalAlpha = 1;
        ctx.fillStyle = "#000";
        ctx.fillRect(0, 0, 640, 480);
        if (sig.complete && sig.naturalWidth) {
            ctx.globalAlpha = Math.min(frame / 10, 1); // op 57: colour 0 -> 255 in 10 frames
            ctx.drawImage(sig, 0, 0);
        }
        if (!sheet.complete || !sheet.naturalWidth || frame < 2) return;
        const c = frame - 2; // children start on script 6's frame 1
        const [ox, oy] = ORIGIN;
        // Op 92 puts leaves at the list head, so they draw first; additive.
        ctx.globalCompositeOperation = "lighter";
        for (const leaf of leaves) {
            const a = leaf.age < 10 ? leaf.age / 10 : leaf.age < 50 ? 1 : (60 - leaf.age) / 10;
            sprite("leaf", ox + tween(leaf.x0, leaf.x1, leaf.age, 60), oy + tween(leaf.y0, leaf.y1, leaf.age, 60),
                a * leaf.grey, leaf.scale, leaf.angle + leaf.spin * leaf.age);
        }
        ctx.globalCompositeOperation = "source-over";
        sprite("now", ox + nowX(c), oy + 16, nowAlpha(c) / 255);
        sprite("shoujo", ox, oy, shoujoAlpha(c) / 255);
    }

    function tick(now) {
        if (boot.hidden) return; // Go's first frame took over
        // Fixed 60 Hz steps; a hidden tab does not fast-forward afterwards.
        let steps = Math.min(Math.floor((now - last) / (1000 / 60)), 4);
        last += steps * (1000 / 60);
        if (now - last > 100) last = now;
        while (steps-- > 0) step();
        draw();
        requestAnimationFrame(tick);
    }
    requestAnimationFrame(tick);
}

const canvas = document.getElementById("loading");
const boot = document.getElementById("boot");
if (canvas && boot) startLoading(canvas, boot);

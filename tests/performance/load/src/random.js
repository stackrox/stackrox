// Mulberry32 - deterministic 32-bit PRNG suitable for k6 (no crypto/Math.random needed).
function mulberry32(seed) {
    let s = seed | 0;
    return function () {
        s = (s + 0x6d2b79f5) | 0;
        let t = Math.imul(s ^ (s >>> 15), 1 | s);
        t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
        return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    };
}

export function createRng(vuId, iterationId) {
    const seed = 12345 + vuId * 1000 + iterationId;
    return mulberry32(seed);
}

export function randomBetween(rng, min, max) {
    return min + rng() * (max - min);
}

export function randomInt(rng, min, max) {
    return Math.floor(randomBetween(rng, min, max + 1));
}

export function shuffle(rng, arr) {
    const result = [...arr];
    for (let i = result.length - 1; i > 0; i--) {
        const j = Math.floor(rng() * (i + 1));
        [result[i], result[j]] = [result[j], result[i]];
    }
    return result;
}

export function pickWeighted(rng, items) {
    const selected = [];
    for (const item of items) {
        if (rng() < item.weight) {
            selected.push(item);
        }
    }
    return shuffle(rng, selected);
}

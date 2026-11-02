// Âm thanh tổng hợp bằng WebAudio: không cần tải file, bật sau lần chạm đầu tiên của người chơi.
let ctx = null, master = null;
let volume = 0.6, muted = false;

export function setAudio(v, m) {
  volume = v; muted = m;
  if (master) master.gain.value = muted ? 0 : volume;
}

function ensure() {
  if (ctx) return ctx.state === 'running';
  const AC = window.AudioContext || window.webkitAudioContext;
  if (!AC) return false;
  ctx = new AC();
  master = ctx.createGain();
  master.gain.value = muted ? 0 : volume;
  master.connect(ctx.destination);
  return ctx.state === 'running';
}
// Trình duyệt chỉ cho phát sau tương tác: mở khoá ở lần bấm/phím đầu tiên.
['pointerdown', 'keydown'].forEach((ev) => window.addEventListener(ev, () => { ensure(); ctx?.resume?.(); }, { passive: true }));

function tone(freq, dur, { type = 'sine', gain = 0.3, slide = 0, delay = 0 } = {}) {
  const t0 = ctx.currentTime + delay;
  const o = ctx.createOscillator(), g = ctx.createGain();
  o.type = type;
  o.frequency.setValueAtTime(freq, t0);
  if (slide) o.frequency.exponentialRampToValueAtTime(Math.max(30, freq + slide), t0 + dur);
  g.gain.setValueAtTime(0.0001, t0);
  g.gain.exponentialRampToValueAtTime(gain, t0 + 0.01);
  g.gain.exponentialRampToValueAtTime(0.0001, t0 + dur);
  o.connect(g); g.connect(master);
  o.start(t0); o.stop(t0 + dur + 0.02);
}

function noise(dur, { gain = 0.25, freq = 800, delay = 0 } = {}) {
  const t0 = ctx.currentTime + delay;
  const buf = ctx.createBuffer(1, Math.ceil(ctx.sampleRate * dur), ctx.sampleRate);
  const d = buf.getChannelData(0);
  for (let i = 0; i < d.length; i++) d[i] = (Math.random() * 2 - 1) * (1 - i / d.length);
  const src = ctx.createBufferSource(), f = ctx.createBiquadFilter(), g = ctx.createGain();
  src.buffer = buf; f.type = 'lowpass'; f.frequency.value = freq; g.gain.value = gain;
  src.connect(f); f.connect(g); g.connect(master);
  src.start(t0);
}

const SFX = {
  click: () => tone(660, 0.06, { type: 'triangle', gain: 0.12 }),
  door: () => { tone(330, 0.12, { type: 'triangle', gain: 0.15 }); tone(495, 0.15, { type: 'triangle', gain: 0.12, delay: 0.08 }); },
  dig: () => { noise(0.18, { gain: 0.35, freq: 500 }); noise(0.15, { gain: 0.25, freq: 400, delay: 0.22 }); },
  pop: () => tone(520, 0.14, { type: 'sine', gain: 0.25, slide: 400 }),
  rare: () => [784, 988, 1175, 1568].forEach((f, i) => tone(f, 0.25, { type: 'triangle', gain: 0.18, delay: i * 0.07 })),
  coin: () => { tone(988, 0.08, { type: 'square', gain: 0.08 }); tone(1319, 0.18, { type: 'square', gain: 0.08, delay: 0.07 }); },
  eat: () => [0, 0.12, 0.24].forEach((d) => noise(0.07, { gain: 0.3, freq: 2200, delay: d })),
  level: () => [523, 659, 784, 1047].forEach((f, i) => tone(f, 0.22, { type: 'triangle', gain: 0.2, delay: i * 0.1 })),
  tap: () => tone(900, 0.04, { type: 'square', gain: 0.06 }),
  win: () => [659, 784, 1047].forEach((f, i) => tone(f, 0.3, { type: 'triangle', gain: 0.2, delay: i * 0.12 })),
  lose: () => tone(330, 0.5, { type: 'sawtooth', gain: 0.1, slide: -200 }),
  notify: () => { tone(880, 0.1, { gain: 0.15 }); tone(1175, 0.14, { gain: 0.12, delay: 0.1 }); },
  crow: () => { tone(500, 0.25, { type: 'sawtooth', gain: 0.08, slide: 300 }); tone(800, 0.5, { type: 'sawtooth', gain: 0.08, slide: -350, delay: 0.25 }); },
  fail: () => tone(260, 0.25, { type: 'sawtooth', gain: 0.1, slide: -120 }),
  honk: () => { tone(440, 0.12, { type: 'square', gain: 0.07 }); tone(440, 0.14, { type: 'square', gain: 0.07, delay: 0.17 }); },
  bus: () => { tone(180, 0.4, { type: 'sawtooth', gain: 0.05, slide: 70 }); noise(0.3, { gain: 0.12, freq: 300 }); },
  whoosh: () => noise(0.18, { gain: 0.14, freq: 1200 }),
  speech: () => tone(700, 0.05, { type: 'triangle', gain: 0.05 }),
};

export function sfx(name) {
  if (muted || !ensure() || !SFX[name]) return;
  try { SFX[name](); } catch { /* âm thanh không được làm hỏng gameplay */ }
}

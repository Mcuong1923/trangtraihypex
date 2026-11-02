// Mô phỏng giằng co giun (PLAN_LOGIC_GAME 4.2): bản sao 1-1 của server/dig.go, không phụ thuộc trình duyệt
// để TestTugMatchesClient (server/tug_parity_test.go) chạy được bằng Node và giữ hai bên khớp nhau.
export const TUG_STEP = 0.05, SLIP_DROP = 0.25, SLIP_RESET = 0.3;
export const r3 = (v) => Math.round(v * 1000) / 1000;

// mulberry32: cùng dãy số với tugRand bên Go.
export function mulberry32(a) {
  return () => {
    a |= 0; a = (a + 0x6D2B79F5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

export function tugBursts(seed, p) {
  const r = mulberry32(seed >>> 0), out = [];
  let t = 0.6 + r() * 0.6;
  while (t < p.maxSec) {
    const dur = 0.4 + r() * 0.8, str = p.burstMin + r() * (p.burstMax - p.burstMin);
    out.push({ at: r3(t), dur: r3(dur), str: r3(str) });
    t += dur;
    if (r() < p.double) {
      const d2 = 0.3 + r() * 0.4;
      out.push({ at: r3(t + 0.15), dur: r3(d2), str: r3(str * 0.9) });
      t += 0.15 + d2;
    }
    t += p.gap * 0.5 + r() * p.gap;
  }
  return out;
}

export function burstForce(bursts, t) {
  let f = 0;
  for (const b of bursts) if (t >= b.at && t < b.at + b.dur && b.str > f) f = b.str;
  return f;
}

// Một bước mô phỏng (dùng chung cho chơi trực tiếp và mô phỏng lại toàn bộ).
export function stepTug(p, s, t, bursts) {
  const f = burstForce(bursts, t);
  if (s.hold) {
    s.prog += p.pull * TUG_STEP * (1 - 0.6 * s.tension);
    s.tension += (p.baseStrain + p.burstStrain * f) * TUG_STEP;
  } else {
    s.prog -= p.retreat * (f > 0 ? 1.6 : 1) * TUG_STEP;
    s.tension -= p.relax * TUG_STEP;
  }
  s.tension = Math.max(0, s.tension);
  s.prog = Math.max(0, Math.min(1, s.prog));
  s.force = f;
  if (s.tension >= 1) {
    s.slips++; s.slipped = true;
    s.prog = Math.max(0, s.prog - SLIP_DROP);
    s.tension = SLIP_RESET;
    if (p.canEscape && s.slips >= p.maxSlips) return { escaped: true, end: r3(t + TUG_STEP) };
  }
  if (s.prog >= 1) return { caught: true, end: r3(t + TUG_STEP) };
  return null;
}

export function simulateTug(p, bursts, events) {
  const s = { prog: 0, tension: 0, slips: 0, hold: false };
  let ei = 0;
  const steps = Math.ceil(p.maxSec / TUG_STEP);
  for (let i = 0; i < steps; i++) {
    const t = i * TUG_STEP;
    while (ei < events.length && events[ei][0] <= t + 1e-9) s.hold = !!events[ei++][1];
    const done = stepTug(p, s, t, bursts);
    if (done) return { ...done, slips: s.slips };
  }
  return { caught: !p.canEscape, escaped: p.canEscape, slips: s.slips, end: p.maxSec };
}

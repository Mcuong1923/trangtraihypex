// Đào giun v2 (PLAN_LOGIC_GAME mục 4): mổ đất → giun ló → cận cảnh → giằng co.
// Mô phỏng nằm ở tug.js (bản sao 1-1 của server/dig.go); máy chủ chạy lại đúng mô phỏng đó với dòng thời
// gian giữ/thả mà client gửi lên để chốt kết quả.
import { G, $, esc, toast, showResult, applyMe, rtag, wormById } from './ui.js';
import { sfx } from './audio.js';
import { TUG_STEP, r3, tugBursts, burstForce, stepTug } from './tug.js';

const T = window.THREE;

// ---------- Màn chơi ----------

const RARITY = ['Thường', 'Khá hiếm', 'Hiếm', 'Sử thi', 'Huyền thoại'];
const ESCAPE_LINES = ['Nó chuồn rồi! Lần sau đừng giật mạnh thế.', 'Giun trơn như lươn, tuột mất tiêu!', 'Hụt rồi! Thả ra khi dây đỏ nhé.'];
let wormLine = null;

function wormTube(color, rare) {
  const m = new T.MeshStandardMaterial({ color, roughness: 0.55, emissive: rare ? new T.Color(color).multiplyScalar(0.35) : 0x000000 });
  const mesh = new T.Mesh(new T.BufferGeometry(), m);
  mesh.castShadow = true;
  return mesh;
}

// Thân giun là ống cong nối miệng hố với mỏ gà; ló ra theo tiến độ, rung theo độ căng.
function shapeWorm(mesh, chick, s, t) {
  const ry = chick.rotation.y, fx = Math.sin(ry), fz = Math.cos(ry);
  const hole = new T.Vector3(chick.position.x + fx * 0.75, 0.02, chick.position.z + fz * 0.75);
  const beak = new T.Vector3(chick.position.x + fx * 0.52, 0.72 - s.pose * 0.18, chick.position.z + fz * 0.52);
  const out = 0.25 + s.prog * 0.75, wob = (s.force > 0 ? 0.09 : 0.03) + s.tension * 0.05;
  const pts = [];
  for (let i = 0; i <= 6; i++) {
    const k = i / 6, p = hole.clone().lerp(beak, k * out + (1 - out) * 0);
    p.y = 0.02 + (beak.y - 0.02) * Math.sin(k * Math.PI / 2) * out + Math.sin(t * 22 + i) * wob * Math.sin(k * Math.PI);
    p.x += Math.cos(ry) * Math.sin(t * 17 + i * 1.3) * wob; p.z -= Math.sin(ry) * Math.sin(t * 17 + i * 1.3) * wob;
    pts.push(p);
  }
  mesh.geometry.dispose();
  mesh.geometry = new T.TubeGeometry(new T.CatmullRomCurve3(pts), 24, 0.05 + 0.012 * (s.tier || 0), 8, false);
}

// Bảng giằng co dùng chung cho đào giun và câu cá (fish.js).
export function overlay(w, p, exhausted, hint) {
  let el = $('tug');
  if (!el) { el = document.createElement('div'); el.id = 'tug'; $('hud').appendChild(el); }
  const hearts = p.canEscape ? `<span class="tug-slips" id="tugSlips">${'❤'.repeat(p.maxSlips)}</span>` : '';
  el.className = 'tug';
  el.innerHTML = `
    <div class="tug-head"><b style="color:${w.color}">${esc(w.name)}</b>${rtag(w)}${hearts}<span class="tug-time" id="tugTime"></span></div>
    <div class="tug-bar"><span>Kéo</span><div class="track"><div class="fill prog" id="tugProg"></div></div></div>
    <div class="tug-bar"><span>Độ căng</span><div class="track tension"><div class="fill ten" id="tugTen"></div></div></div>
    <button class="tug-btn" id="tugBtn">GIỮ ĐỂ KÉO <kbd>Space</kbd></button>
    <div class="tug-hint">${hint || `${exhausted ? '<b>Kiệt sức</b>: chỉ đào được giun thường. ' : ''}Giữ để kéo · thấy đỏ thì <b>thả</b> · giun quẫy là lúc nguy hiểm`}</div>`;
  el.hidden = false;
  return el;
}

export function pop(text, color, ms) {
  let el = $('tugPop');
  if (!el) { el = document.createElement('div'); el.id = 'tugPop'; $('hud').appendChild(el); }
  el.innerHTML = text; el.style.setProperty('--c', color || '#fff6e4');
  el.className = 'tug-pop show';
  clearTimeout(el._t); el._t = setTimeout(() => (el.className = 'tug-pop'), ms);
}

// startDig: E ở ụ đất. Máy chủ gieo loài giun + seed; client mổ, cho giun ló, rồi chơi giằng co.
export async function startDig(item) {
  if (G.busy) return;
  G.busy = true;
  const res = await G.net.act('dig_start', { id: item.id });
  if (!showResult(res, { quiet: true })) { G.busy = false; return; }
  const d = res.data, w = d.worm, p = d.params, tier = d.tier, me = G.self.mesh;
  const bursts = tugBursts(d.seed, p);
  // Quay mặt về ụ đất rồi mổ.
  me.rotation.y = Math.atan2(item.x - G.self.pos.x, item.z - G.self.pos.z);
  G.fx?.('dig', me, { dur: d.peckMs / 1000 });
  sfx('dig');
  const tugAt = performance.now() + d.peckMs + d.revealMs;
  setTimeout(() => {
    pop(`<small>Có gì đó ngọ nguậy…!</small><b>🪱 ${esc(w.name)}</b><span>${RARITY[tier]}</span>`, w.color, Math.max(900, d.revealMs + 500));
    sfx(tier >= 2 ? 'rare' : 'pop');
    G.world.controls.closeup({ x: G.self.pos.x, z: G.self.pos.z, ry: me.rotation.y });
  }, d.peckMs);

  const s = { prog: 0, tension: 0, slips: 0, hold: false, force: 0, pose: 0, tier };
  const events = [];
  let step = 0, done = null, live = true, ui = null, tube = null;
  const holdOn = (v) => {
    if (!live || v === s.wantHold) return;
    s.wantHold = v;
    const t = r3(Math.max(0, (performance.now() - tugAt) / 1000));
    const last = events[events.length - 1];
    if (last && last[0] === t) last[1] = v ? 1 : 0; else events.push([t, v ? 1 : 0]);
    ui?.querySelector('#tugBtn')?.classList.toggle('on', v);
  };
  const key = (e) => { if (e.code === 'Space' && !e.repeat) { e.preventDefault(); holdOn(e.type === 'keydown'); } };
  const finish = async () => {
    live = false;
    window.removeEventListener('keydown', key, true); window.removeEventListener('keyup', key, true);
    const r = await G.net.act('dig_finish', { id: d.id, ev: events });
    if (!r.ok && /chưa chịu ra/.test(r.err || '')) { // đồng hồ client nhanh hơn máy chủ: chờ rồi gửi lại
      await new Promise((ok) => setTimeout(ok, 700));
      return finishRetry(r);
    }
    end(r);
  };
  const finishRetry = async () => end(await G.net.act('dig_finish', { id: d.id, ev: events }));
  const end = (r) => {
    G.busy = false;
    G.world.controls.closeup(null);
    me.userData.tug = null;
    if (tube) { tube.parent?.remove(tube); tube.geometry.dispose(); }
    if (ui) ui.hidden = true;
    if (r.me) applyMe(r.me);
    if (!r.ok) { showResult(r); return; }
    if (r.data.caught) {
      G.fx?.('dug', me, { color: w.color, rare: w.rare });
      onLoot(r.data);
    } else {
      sfx('fail');
      toast(`🪱 ${w.name} chuồn mất! ${ESCAPE_LINES[(Math.random() * ESCAPE_LINES.length) | 0]} (+${r.data.xp} XP an ủi)`, 'warn');
    }
  };
  const cancel = () => { if (!live) return; live = false; G.net.raw({ t: 'act', a: 'dig_cancel' }); end({ ok: true, data: { caught: false, xp: 0 } }); };
  G.cancelDig = cancel;

  function frame() {
    if (!live) return;
    requestAnimationFrame(frame);
    const now = performance.now(), el = (now - tugAt) / 1000;
    if (el < 0) return;
    if (!ui) {
      ui = overlay(w, p, d.exhausted);
      const btn = ui.querySelector('#tugBtn');
      btn.onpointerdown = (e) => { e.preventDefault(); btn.setPointerCapture?.(e.pointerId); holdOn(true); };
      btn.onpointerup = btn.onpointercancel = () => holdOn(false);
      window.addEventListener('keydown', key, true); window.addEventListener('keyup', key, true);
      tube = wormTube(w.color, w.rare); G.world.scene.add(tube);
      me.userData.tug = s;
    }
    // Chạy mô phỏng theo đúng các bước 50 ms như máy chủ; sự kiện có hiệu lực từ bước kế tiếp.
    let ei = 0;
    while (!done && step * TUG_STEP <= el) {
      const t = step * TUG_STEP;
      for (ei = 0; ei < events.length; ei++) if (events[ei][0] <= t + 1e-9) s.hold = !!events[ei][1];
      done = stepTug(p, s, t, bursts) || (step + 1 >= Math.ceil(p.maxSec / TUG_STEP) ? { caught: !p.canEscape, escaped: p.canEscape, end: p.maxSec } : null);
      if (s.slipped) { s.slipped = false; sfx('fail'); ui.classList.add('slip'); setTimeout(() => ui?.classList.remove('slip'), 300); const h = $('tugSlips'); if (h) h.textContent = '❤'.repeat(Math.max(0, p.maxSlips - s.slips)) + '🤍'.repeat(Math.min(p.maxSlips, s.slips)); }
      step++;
    }
    s.pose += ((s.hold ? 1 : 0) - s.pose) * 0.2;
    $('tugProg').style.width = (s.prog * 100).toFixed(1) + '%';
    const ten = $('tugTen'); ten.style.width = (Math.min(1, s.tension) * 100).toFixed(1) + '%';
    ten.style.background = s.tension > 0.75 ? '#d94d42' : s.tension > 0.45 ? '#e8a33a' : '#5aa05a';
    // Báo trước cơn quẫy 0,3 giây để kịp thả nút.
    ui.classList.toggle('warn', burstForce(bursts, el + 0.3) > 0);
    $('tugTime').textContent = Math.max(0, p.maxSec - el).toFixed(1) + 's';
    shapeWorm(tube, me, s, now / 1000);
    if (done) { s.prog = done.caught ? 1 : s.prog; finish(); }
  }
  requestAnimationFrame(frame);
}

function onLoot(data) {
  const loot = data.loot, w = wormById(loot.worm);
  G.showLoot?.(loot, data.slips === 0, [...(data.notes || []), `+${data.xp} XP khi đào`]);
}

// Câu cá ở cầu ao (PLAN_LOGIC_GAME 16): quăng cần → phao nổi → cá cắn câu (phao chìm) → giằng co bằng đúng
// mô phỏng tug.js như đào giun. Máy chủ gieo loài cá và seed lúc thả câu, chạy lại mô phỏng để chốt.
import { G, toast, showResult, applyMe } from './ui.js';
import { sfx } from './audio.js';
import { TUG_STEP, r3, tugBursts, burstForce, stepTug } from './tug.js';
import { overlay, pop } from './dig.js';
import { mat } from './models.js';

const T = window.THREE;
const RARITY = ['Thường', 'Khá hiếm', 'Hiếm', 'Sử thi', 'Huyền thoại'];
const FISH_COLOR = ['#7fa6b8', '#d9894a', '#6d6a5e', '#f2c33a'];
const CAST = 3.2; // phao cách gà 3,2 m

function fishMesh(color) {
  const g = new T.Group();
  const body = new T.Mesh(new T.SphereGeometry(0.16, 12, 8), mat(new T.Color(color).getHex())); body.scale.set(0.7, 0.6, 1.4); g.add(body);
  const tail = new T.Mesh(new T.ConeGeometry(0.12, 0.2, 4), mat(new T.Color(color).multiplyScalar(0.8).getHex())); tail.rotation.x = Math.PI / 2; tail.position.z = -0.28; g.add(tail);
  return g;
}

export async function startFish(item) {
  if (G.busy) return;
  G.busy = true;
  const res = await G.net.act('fish_start', { id: item.id });
  if (!showResult(res, { quiet: true })) { G.busy = false; return; }
  const d = res.data, p = d.params, tier = d.tier, me = G.self.mesh, info = { name: d.fish.name, color: FISH_COLOR[tier] || '#7fa6b8', rarity: RARITY[tier] };
  const bursts = tugBursts(d.seed, p);
  // Quay mặt ra giữa ao (tâm ao ở toạ độ khu 0, −2).
  const pond = G.world.zones.park?.group.localToWorld(new T.Vector3(0, 0, -2)) || new T.Vector3(G.self.pos.x, 0, G.self.pos.z - 4);
  me.rotation.y = Math.atan2(pond.x - G.self.pos.x, pond.z - G.self.pos.z);
  const fx = Math.sin(me.rotation.y), fz = Math.cos(me.rotation.y);
  const water = new T.Vector3(G.self.pos.x + fx * CAST, 0.08, G.self.pos.z + fz * CAST);
  // Cần câu cầm ở cánh, dây nối đầu cần với phao.
  const rod = new T.Mesh(new T.CylinderGeometry(0.015, 0.03, 1.7, 6), mat(0x8a5a30));
  rod.position.set(0.28, 0.85, 0.35); rod.rotation.x = 0.9; me.add(rod);
  const bobber = new T.Group();
  const top = new T.Mesh(new T.SphereGeometry(0.07, 10, 8, 0, Math.PI * 2, 0, Math.PI / 2), new T.MeshStandardMaterial({ color: 0xd8342b }));
  const bot = new T.Mesh(new T.SphereGeometry(0.07, 10, 8, 0, Math.PI * 2, Math.PI / 2, Math.PI / 2), new T.MeshStandardMaterial({ color: 0xf4f1e6 }));
  bobber.add(top, bot); bobber.position.copy(water); G.world.scene.add(bobber);
  const line = new T.Line(new T.BufferGeometry().setFromPoints([new T.Vector3(), new T.Vector3()]), new T.LineBasicMaterial({ color: 0xf5f0e0 }));
  line.frustumCulled = false; G.world.scene.add(line);
  const fish = fishMesh(info.color); fish.visible = false; G.world.scene.add(fish);
  G.fx?.('cast', me, { ms: d.biteMs });
  sfx('pop');
  G.world.controls.closeup({ x: G.self.pos.x, z: G.self.pos.z, ry: me.rotation.y, far: CAST });
  const tugAt = performance.now() + d.biteMs + d.revealMs;
  let bitten = false;
  setTimeout(() => {
    if (!live) return; // lượt đã bị huỷ trước khi cá cắn câu
    bitten = true;
    pop(`<small>Phao chìm rồi! Cá cắn câu!</small><b>🎣 ${info.name}</b><span>${RARITY[tier]}</span>`, info.color, Math.max(900, d.revealMs + 600));
    sfx(tier >= 2 ? 'rare' : 'pop');
  }, d.biteMs);

  const s = { prog: 0, tension: 0, slips: 0, hold: false, force: 0, pose: 0, tier };
  const events = [];
  let step = 0, done = null, live = true, ui = null;
  const holdOn = (v) => {
    if (!live || v === s.wantHold) return;
    s.wantHold = v;
    // Làm tròn mili giây có thể rơi vào bước 50 ms đã mô phỏng xong → máy chủ áp sự kiện sớm hơn một bước (L14): đẩy qua bước đó.
    let t = r3(Math.max(0, (performance.now() - tugAt) / 1000));
    const lastSim = (step - 1) * TUG_STEP;
    if (step > 0 && t <= lastSim + 1e-9) t = r3(lastSim + 0.001);
    const last = events[events.length - 1];
    if (last && last[0] === t) last[1] = v ? 1 : 0; else events.push([t, v ? 1 : 0]);
    ui?.querySelector('#tugBtn')?.classList.toggle('on', v);
  };
  const key = (e) => { if (e.code === 'Space' && !e.repeat) { e.preventDefault(); holdOn(e.type === 'keydown'); } };
  const cleanup = () => {
    me.remove(rod); G.world.scene.remove(bobber, line, fish); line.geometry.dispose();
    window.removeEventListener('keydown', key, true); window.removeEventListener('keyup', key, true);
    G.world.controls.closeup(null);
    me.userData.tug = null;
    if (ui) ui.hidden = true;
    G.busy = false;
  };
  const end = (r) => {
    cleanup();
    if (r.me) applyMe(r.me);
    if (!r.ok) { showResult(r); return; }
    if (r.data.caught) {
      sfx('coin'); G.fx?.('fish_catch', me, { tier });
      toast(`🎣 Câu được ${d.fish.emo} ${info.name}! +${r.data.xp} XP · cá đã vào kho (bán ở Quầy thức ăn hoặc ăn cho no)`, 'good');
      for (const n of r.data.notes || []) toast(n, 'good');
    } else {
      sfx('fail');
      toast(`🐟 ${info.name} giật đứt mồi rồi lặn mất! (+${r.data.xp} XP an ủi)`, 'warn');
    }
  };
  // Máy chủ đã huỷ lượt câu (đổi phòng, chuyển vùng, nối lại): chỉ dọn màn giằng co, không gửi gì.
  G.abortTug = () => { if (!live) return; live = false; end({ ok: false, err: 'Lượt bị huỷ vì đổi phòng hoặc mất kết nối giữa chừng.' }); };
  // Bỏ cần giữa chừng: máy chủ tính như cá sổng (vẫn hồi câu, tốn sức).
  const cancel = () => {
    if (!live) return;
    live = false;
    G.net.act('fish_cancel').then((r) => { if (r.me) applyMe(r.me); });
    cleanup();
    toast('Đã bỏ cần — cá lặn mất.', 'warn');
  };
  G.cancelDig = cancel;
  const finish = async () => {
    live = false;
    let r = await G.net.act('fish_finish', { id: d.id, ev: events });
    if (!r.ok && /chưa chịu lên/.test(r.err || '')) { await new Promise((ok) => setTimeout(ok, 700)); r = await G.net.act('fish_finish', { id: d.id, ev: events }); }
    end(r);
  };

  const tip = new T.Vector3();
  function frame() {
    if (!live) return;
    requestAnimationFrame(frame);
    const now = performance.now(), el = (now - tugAt) / 1000, t = now / 1000;
    rod.updateMatrixWorld(true);
    tip.set(0, 0.85, 0).applyMatrix4(rod.matrixWorld);
    if (el < 0) {
      // Chờ cá: phao nhấp nhô, lúc cá cắn thì chìm hẳn.
      bobber.position.y = bitten ? -0.06 + Math.sin(t * 30) * 0.03 : 0.08 + Math.sin(t * 2.4) * 0.025;
      line.geometry.setFromPoints([tip, bobber.position]);
      return;
    }
    if (!ui) {
      ui = overlay(info, p, d.exhausted, `${d.exhausted ? '<b>Kiệt sức</b>: chỉ câu được cá nhỏ. ' : ''}Giữ để kéo cá vào · dây căng đỏ thì <b>thả</b> · cá quẫy là lúc dễ đứt`);
      const btn = ui.querySelector('#tugBtn');
      btn.onpointerdown = (e) => { e.preventDefault(); btn.setPointerCapture?.(e.pointerId); holdOn(true); };
      btn.onpointerup = btn.onpointercancel = () => holdOn(false);
      ui.querySelector('#tugCancel').onclick = cancel;
      window.addEventListener('keydown', key, true); window.addEventListener('keyup', key, true);
      bobber.visible = false; fish.visible = true;
      me.userData.tug = s;
    }
    let ei = 0;
    while (!done && step * TUG_STEP <= el) {
      const st = step * TUG_STEP;
      for (ei = 0; ei < events.length; ei++) if (events[ei][0] <= st + 1e-9) s.hold = !!events[ei][1];
      done = stepTug(p, s, st, bursts) || (step + 1 >= Math.ceil(p.maxSec / TUG_STEP) ? { caught: !p.canEscape, escaped: p.canEscape, end: p.maxSec } : null);
      if (s.slipped) { s.slipped = false; sfx('fail'); ui.classList.add('slip'); setTimeout(() => ui?.classList.remove('slip'), 300); const h = document.getElementById('tugSlips'); if (h) h.textContent = '❤'.repeat(Math.max(0, p.maxSlips - s.slips)) + '🤍'.repeat(Math.min(p.maxSlips, s.slips)); }
      step++;
    }
    s.pose += ((s.hold ? 1 : 0) - s.pose) * 0.2;
    rod.rotation.x = 0.9 - s.pose * 0.35 - s.tension * 0.15;
    // Cá bị kéo dần từ chỗ phao vào chân cầu, quẫy theo độ căng và cơn giật.
    const k = Math.min(1, s.prog), wob = (s.force > 0 ? 0.25 : 0.08) + s.tension * 0.1;
    fish.position.set(water.x - fx * CAST * 0.8 * k + Math.cos(me.rotation.y) * Math.sin(t * 9) * wob, 0.02 + k * 0.5 + Math.abs(Math.sin(t * 13)) * wob * 0.4, water.z - fz * CAST * 0.8 * k - Math.sin(me.rotation.y) * Math.sin(t * 9) * wob);
    fish.rotation.y = me.rotation.y + Math.PI + Math.sin(t * 15) * wob * 2;
    line.geometry.setFromPoints([tip, fish.position]);
    document.getElementById('tugProg').style.width = (s.prog * 100).toFixed(1) + '%';
    const ten = document.getElementById('tugTen'); ten.style.width = (Math.min(1, s.tension) * 100).toFixed(1) + '%';
    ten.style.background = s.tension > 0.75 ? '#d94d42' : s.tension > 0.45 ? '#e8a33a' : '#5aa05a';
    ui.classList.toggle('warn', burstForce(bursts, el + 0.3) > 0);
    document.getElementById('tugTime').textContent = Math.max(0, p.maxSec - el).toFixed(1) + 's';
    if (done) { s.prog = done.caught ? 1 : s.prog; finish(); }
  }
  requestAnimationFrame(frame);
}

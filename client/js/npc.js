// Làng sống phía client (PLAN_LOGIC_GAME 5–9): dựng và nội suy NPC di động do máy chủ điều khiển (xe buýt,
// xe ôm, xe trên đường, cameo), hiện bong bóng thoại, dấu "!" hướng dẫn, thẻ mời xe ôm, câu hỏi của NPC,
// bảng bến xe buýt và trận biểu diễn ở Sân tỷ thí. Mọi quyết định (ai nói gì, giá vé, chở đi đâu) ở máy chủ.
import { G, $, esc, toast, bubble, showResult, openSheet, closeSheet } from './ui.js';
import { buildChicken, dress, buildBus, buildVehicle, buildDog, mat, label } from './models.js';
import * as V from './v11.js';
import * as MAP from './map.js';
import { mulberry32 } from './tug.js';
import { sfx } from './audio.js';

const T = window.THREE;
G.actors = new Map();
const ROLE_LIFT = { tai_xe: 1.5, xe_om: 0.55, shipper: 0.55, dap_xe: 0.45, thieu_gia: 0.45, van_truot: 0.15 };

// ---------- Mô hình ----------

function chicken(look = {}) {
  const c = buildChicken({ color: look.color || 'vang', comb: look.comb || 'don' });
  const eq = {};
  for (const slot of ['head', 'eye', 'neck', 'body']) if (look[slot] && G.catalog.cosmetics.some((x) => x.id === look[slot])) eq[slot] = look[slot];
  dress(c, eq, { shirt: look.shirt || 'none' }, G.catalog.cosmetics);
  if (look.scale) c.scale.setScalar(+look.scale);
  return c;
}

function skateboard() {
  const g = new T.Group();
  const deck = new T.Mesh(new T.BoxGeometry(0.34, 0.05, 1.0), mat(0xd8574c)); deck.position.y = 0.1; g.add(deck);
  for (const sx of [-1, 1]) for (const sz of [-1, 1]) {
    const w = new T.Mesh(new T.CylinderGeometry(0.06, 0.06, 0.05, 10), mat(0xf2ead2)); w.rotation.z = Math.PI / 2; w.position.set(sx * 0.15, 0.05, sz * 0.36); g.add(w);
  }
  return g;
}

// Đèn pha bật khi trời tối (PLAN_LOGIC_GAME 9.1): bóng đèn, quầng sáng và vệt sáng trên mặt đường phía trước.
let glowMat = null;
function glowMaterial() {
  if (!glowMat) {
    const cv = document.createElement('canvas'); cv.width = cv.height = 64;
    const c = cv.getContext('2d'), gr = c.createRadialGradient(32, 32, 0, 32, 32, 32);
    gr.addColorStop(0, 'rgba(255,244,200,1)'); gr.addColorStop(0.3, 'rgba(255,226,150,.5)'); gr.addColorStop(1, 'rgba(255,210,120,0)');
    c.fillStyle = gr; c.fillRect(0, 0, 64, 64);
    glowMat = new T.SpriteMaterial({ map: new T.CanvasTexture(cv), blending: T.AdditiveBlending, depthWrite: false, transparent: true });
  }
  return glowMat;
}
function headlights(root, ride) {
  root.updateMatrixWorld(true);
  const box = new T.Box3().setFromObject(root), size = box.getSize(new T.Vector3());
  const g = new T.Group(), n = ride === 'xe_dap' || ride === 'xe_may' ? 1 : 2, big = ride === 'bus';
  for (let i = 0; i < n; i++) {
    const lamp = new T.Mesh(new T.SphereGeometry(big ? 0.13 : 0.08, 8, 6), new T.MeshBasicMaterial({ color: 0xfff3c4 }));
    lamp.position.set(n === 1 ? 0 : (i ? 1 : -1) * size.x * 0.3, box.min.y + size.y * (big ? 0.22 : 0.42), box.max.z + 0.03);
    const glow = new T.Sprite(glowMaterial()); glow.scale.setScalar(big ? 1.7 : 1.1); glow.position.copy(lamp.position);
    g.add(lamp, glow);
  }
  const beam = new T.Mesh(new T.CircleGeometry(1, 20), new T.MeshBasicMaterial({ color: 0xffe9a8, transparent: true, opacity: 0.16, depthWrite: false }));
  beam.rotation.x = -Math.PI / 2; beam.scale.set(n === 1 ? 0.8 : 1.5, big ? 3.4 : 2.4, 1); beam.position.set(0, 0.05, box.max.z + (big ? 3.4 : 2.3));
  g.add(beam);
  g.visible = false;
  root.add(g);
  return g;
}

function buildActor(info) {
  const look = info.look || {};
  const root = new T.Group();
  let rider = null;
  if (info.ride === 'bus') {
    const bus = buildBus(); bus.scale.setScalar(1.5); root.add(bus);
    rider = chicken({ color: look.color, head: 'mu_luoi' }); rider.scale.setScalar(0.8); rider.position.set(-0.55, 1.25, 2.55); root.add(rider);
    // Vài khách NPC ngồi sẵn cho có không khí (không chiếm ghế người chơi).
    const r = mulberry32(info.id.length * 977 + info.id.charCodeAt(1));
    for (let i = 0; i < 3; i++) { const q = chicken({ color: ['vang', 'trang', 'nau', 'kem'][(r() * 4) | 0] }); q.scale.setScalar(0.72); q.position.set(i % 2 ? 0.55 : -0.55, 1.2, 0.6 - i * 1.4); root.add(q); }
  } else {
    rider = chicken(look); root.add(rider);
    if (['xe_may', 'xe_dap', 'o_to'].includes(info.ride)) {
      V.setVehicle(rider, info.ride);
      if (info.ride === 'o_to' && look.car) rider.userData.vehicle.traverse((o) => { if (o.isMesh && o.material?.color && o.material.color.getHex() !== 0x2e302b && o.geometry.parameters?.width > 1) o.material = mat(parseInt(look.car.slice(1), 16)); });
    }
    if (info.ride === 'van_truot') { const sk = skateboard(); rider.add(sk); rider.userData.skate = true; }
    if (look.mic) { const mic = new T.Mesh(new T.CylinderGeometry(0.03, 0.03, 0.28, 8), mat(0x2b2b2b)); mic.position.set(0.42, 0.85, 0.32); mic.rotation.x = -0.6; rider.add(mic); const head = new T.Mesh(new T.SphereGeometry(0.06, 10, 8), mat(0x555555)); head.position.set(0.42, 0.98, 0.42); rider.add(head); }
    if (look.tray) { const tray = new T.Mesh(new T.CylinderGeometry(0.32, 0.28, 0.08, 14), mat(0xb88b54)); tray.position.set(0, 0.95, 0.55); rider.add(tray); for (let i = 0; i < 3; i++) { const b = new T.Mesh(new T.SphereGeometry(0.08, 8, 6), mat([0xe7b23e, 0x87a650, 0xe7b23e][i])); b.position.set(-0.12 + i * 0.12, 1.02, 0.55); rider.add(b); } }
  }
  root.userData.rider = rider;
  if (['bus', 'xe_may', 'xe_dap', 'o_to'].includes(info.ride)) root.userData.lights = headlights(root, info.ride);
  const tag = label(`${info.name} · NPC`, { bg: 'rgba(89,54,29,.78)', scale: 0.42 });
  tag.position.y = (ROLE_LIFT[info.role] || 0) + 2.15; tag.visible = false; root.add(tag); root.userData.tag = tag;
  return root;
}

function spawn(info) {
  if (G.actors.has(info.id)) return G.actors.get(info.id);
  const mesh = buildActor(info);
  G.world.scene.add(mesh);
  const a = { info, mesh, target: new T.Vector3(), ry: 0, mv: 0, phase: Math.random() * 6, state: '' , fresh: true };
  G.actors.set(info.id, a);
  return a;
}

function despawn(id) {
  const a = G.actors.get(id);
  if (!a) return;
  G.world.scene.remove(a.mesh);
  G.actors.delete(id);
}

export function onWelcome(list) {
  for (const id of [...G.actors.keys()]) despawn(id);
  for (const info of list || []) spawn(info);
  stopShow();
}
export const onJoin = (m) => spawn(m.a);
export const onLeave = (m) => despawn(m.id);

// Dòng vị trí trong snapshot: [id, x, z, ry, đang chạy, trạng thái].
export function onSnap(rows) {
  for (const [id, x, z, ry, mv, st] of rows || []) {
    const a = G.actors.get(id);
    if (!a) continue;
    a.target.set(x, 0, z); a.ry = ry; a.mv = mv; a.state = st;
    if (a.fresh) { a.mesh.position.set(x, 0, z); a.mesh.rotation.y = ry; a.fresh = false; }
  }
}

const lerpAngle = (a, b, t) => { let d = ((b - a + Math.PI) % (Math.PI * 2)) - Math.PI; if (d < -Math.PI) d += Math.PI * 2; return a + d * t; };

export function update(dt, me) {
  const k = 1 - Math.exp(-dt * 10), night = (G.world.nightK || 0) > 0.45;
  for (const a of G.actors.values()) {
    if (a.mesh.userData.lights) a.mesh.userData.lights.visible = night;
    a.mesh.position.lerp(a.target, k);
    a.mesh.rotation.y = lerpAngle(a.mesh.rotation.y, a.ry, 0.2);
    const r = a.mesh.userData.rider, p = r?.userData.parts;
    if (!p) continue;
    if (a.info.ride === 'walk') {
      if (a.mv) { a.phase += dt * 11; p.legs[0].rotation.x = Math.sin(a.phase) * 0.6; p.legs[1].rotation.x = -Math.sin(a.phase) * 0.6; p.rig.position.y = Math.abs(Math.sin(a.phase * 2)) * 0.05; }
      else { p.legs[0].rotation.x *= 0.8; p.legs[1].rotation.x *= 0.8; p.rig.position.y = 0; }
    } else if (a.info.ride !== 'bus') {
      p.rig.position.y = r.userData.skate ? 0.12 + (a.mv ? Math.sin(performance.now() * 0.01) * 0.02 : 0) : V.rideHeight(a.info.ride);
      p.legs[0].rotation.x = p.legs[1].rotation.x = r.userData.skate ? 0 : -1.1;
      if (r.userData.skate && a.mv) { a.phase += dt * 4; p.rig.rotation.z = Math.sin(a.phase) * 0.08; }
    }
    // Nhãn "· NPC" chỉ hiện khi ở gần để người chơi biết đây không phải người thật.
    const tag = a.mesh.userData.tag;
    tag.visible = me && a.mesh.position.distanceTo(me) < 9;
  }
  updateMarker(dt);
  updateShow(dt);
  updateRideCard();
}

// ---------- Ai đang nói ----------

const RESIDENT = { ga_don_khach: 'Gà Đón Khách', quay_an: 'Gà Bán Thóc', quay_do: 'Gà Thợ May' };
const meshCache = new Map();
export function itemMesh(id) {
  if (meshCache.has(id) && meshCache.get(id).parent) return meshCache.get(id);
  for (const z of Object.values(G.world.zones)) {
    if (RESIDENT[id]) {
      const r = z.scenery?.ambient?.find((x) => x.name === RESIDENT[id]);
      if (r) { meshCache.set(id, r.model); return r.model; }
      continue;
    }
    for (const c of z.group.children) if (c.userData.item?.id === id) { meshCache.set(id, c); return c; }
  }
  return null;
}

function speakerMesh(id) {
  return G.actors.get(id)?.mesh || itemMesh(id) || (id === G.pid ? G.self?.mesh : G.players.get(id)?.mesh) || null;
}

export function onSay(m) {
  const me = !!m.to && m.to === G.pid;
  const mesh = m.remote ? null : speakerMesh(m.id);
  if (mesh) {
    const a = G.actors.get(m.id);
    bubble(mesh, m.text, { name: m.name, me, ttl: m.ttl || 4, lift: a ? ROLE_LIFT[a.info.role] || 0 : 0 });
  }
  if (me) {
    if (!mesh) toast(`🐔 ${m.name}: ${m.text}`, 'npc');
    G.chatLog?.({ kind: 'npc', name: m.name, text: m.text });
    sfx('speech');
  }
}

// ---------- Hướng dẫn: dấu "!" trên NPC của bước kế ----------

let marker = null;
function makeMarker() {
  const cv = document.createElement('canvas'); cv.width = cv.height = 128;
  const c = cv.getContext('2d');
  c.fillStyle = '#ffcf3f'; c.strokeStyle = '#7a4a12'; c.lineWidth = 8;
  c.beginPath(); c.arc(64, 64, 52, 0, Math.PI * 2); c.fill(); c.stroke();
  c.fillStyle = '#5a3410'; c.font = 'bold 84px system-ui'; c.textAlign = 'center'; c.textBaseline = 'middle'; c.fillText('!', 64, 70);
  const sp = new T.Sprite(new T.SpriteMaterial({ map: new T.CanvasTexture(cv), depthTest: false, toneMapped: false }));
  sp.scale.setScalar(0.85); sp.renderOrder = 9;
  return sp;
}
export function onMark(m) {
  G.tutStep = m.step ? m : null;
  const card = $('tutCard');
  if (!m.step) { card.hidden = true; marker?.parent?.remove(marker); return; }
  card.hidden = false;
  card.innerHTML = `<small>HƯỚNG DẪN</small><b>${esc(m.text)}</b>`;
  marker ||= makeMarker();
  marker.userData.npc = m.npc; marker.parent?.remove(marker);
  if (m.area && m.area !== G.zone && G.worldMap && MAP.currentDestination() !== m.area) MAP.setDestination(m.area, 'Đi theo dấu chỉ đường tới bước hướng dẫn tiếp theo.');
}
function updateMarker() {
  if (!marker || !G.tutStep) return;
  if (!marker.parent) { const host = itemMesh(marker.userData.npc); if (host) { host.add(marker); marker.position.set(0, 2.9 / (host.scale.y || 1), 0); } }
  if (marker.parent) marker.position.y = (2.9 + Math.sin(performance.now() * 0.004) * 0.12) / (marker.parent.scale.y || 1);
}

// ---------- Xe ôm: thẻ mời ----------

let offer = null, offerTimer = 0;
export function onOffer(m) {
  offer = m;
  const card = $('offerCard');
  card.innerHTML = `<div><small>XE ÔM MỜI</small><b>🛵 ${esc(m.driver)}</b><span>${m.short} thóc nếu ≤ ${m.shortM} m, ${m.long} thóc nếu xa hơn</span></div>
    <button class="btn hot" id="offerYes">Đi <kbd>E</kbd></button><button class="btn sec" id="offerNo">Không <kbd>X</kbd></button>`;
  card.hidden = false;
  $('offerYes').onclick = acceptOffer;
  $('offerNo').onclick = declineOffer;
  clearTimeout(offerTimer); offerTimer = setTimeout(() => endOffer(m.id), m.ttl * 1000);
  sfx('honk');
}
export function onOfferEnd(m) { endOffer(m.id); }
function endOffer(id) { if (offer && (!id || offer.id === id)) { offer = null; $('offerCard').hidden = true; } }
export function hasOffer() { return !!offer; }
export function acceptOffer() {
  if (!offer) return;
  const o = offer;
  endOffer();
  const fareFor = (id) => { const s = MAP.etaSeconds(id); return s != null && s * 4.6 > o.shortM ? o.long : o.short; };
  MAP.pickDestination((id) => `🛵 Chở tới đây · khoảng ${fareFor(id)} thóc`, async (id) => {
    const r = await G.net.act('xeom_accept', { id: o.actor, to: id });
    showResult(r);
  }, () => G.net.act('xeom_decline', { id: o.actor }), 'xe ôm chở tới khu đó');
}
export function declineOffer() { if (!offer) return; const o = offer; endOffer(); G.net.act('xeom_decline', { id: o.actor }); }

// ---------- Câu hỏi của NPC (shipper hỏi đường, nhà báo phỏng vấn, bán rong) ----------

export function onAsk(m) {
  const card = $('askCard');
  card.innerHTML = `<small>${esc(m.name)} hỏi</small><b>${esc(m.text)}</b><div class="ask-choices">${m.choices.map((c, i) => `<button class="btn sec" data-ask="${i}">${esc(c)}</button>`).join('')}</div>`;
  card.hidden = false;
  const t = setTimeout(() => (card.hidden = true), (m.ttl || 20) * 1000);
  card.querySelectorAll('[data-ask]').forEach((b) => (b.onclick = async () => {
    clearTimeout(t); card.hidden = true;
    const r = await G.net.act('npc_choice', { id: m.id, i: +b.dataset.ask });
    showResult(r);
    if (r.ok && r.data?.loot) G.net.act('show', { id: r.data.loot }); // trả lời phỏng vấn xong thì khoe thẻ giun
  }));
}

// ---------- Xe buýt vòng làng ----------

export function nearestStop() {
  const rn = G.worldMap?.roads;
  if (!rn || !G.self || G.myCarrier) return null;
  return rn.stops.find((s) => Math.hypot(s.pos.x - G.self.pos.x, s.pos.z - G.self.pos.z) < 4.5) || null;
}

const mmss = (s) => `${Math.floor(s / 60)}:${String(Math.round(s % 60)).padStart(2, '0')}`;
export async function openBusStop() {
  const r = await G.net.act('bus_stops');
  if (!showResult(r, { quiet: true })) return;
  const d = r.data, here = d.stops[d.here];
  if (!here) { toast('Đứng gần biển 🚌 ở cổng mỗi khu để đón xe.', 'warn'); return; }
  const tok = d.tokens > 0 ? `🎫 Còn <b>${d.tokens}/${G.catalog.maxBusTokens}</b> lượt miễn phí` : `🎫 Hết lượt miễn phí · lượt mới sau <b>${mmss(d.nextToken)}</b> · vé <b>${d.fare} thóc</b>`;
  const ready = here.eta === 0;
  openSheet('bus', `🚌 Bến ${esc(here.title)}`, `
    <p class="mut" style="margin-top:0">${ready ? '<b style="color:var(--good)">Xe đang dừng ở bến — lên ngay!</b>' : `Xe tới sau <b>${Math.ceil(here.eta)} giây</b>. Xe chạy vòng làng liên tục, dừng ở cổng mọi khu.`}</p>
    <p>${tok}</p>
    ${d.stops.filter((s, i) => i !== d.here).map((s) => `<div class="row"><div class="ico" style="background:#e3eef5">🅑</div><div class="meta"><div class="nm">${esc(s.title)}</div></div>
      <div class="act"><button class="mini ${ready ? 'hot' : 'sec'}" data-board="${s.id}" ${ready ? '' : 'disabled'}>Lên xe tới đây</button></div></div>`).join('')}
    <button class="btn block sec" data-board="" ${ready ? '' : 'disabled'}>Lên xe đi vòng, tự xuống</button>
    <div class="note">Lượt miễn phí dùng chung cho xe buýt vòng làng và xe liên trại, tối đa ${G.catalog.maxBusTokens} lượt, hồi 1 lượt mỗi ${Math.round(G.catalog.balance.tokenEverySec / 60)} phút.</div>`);
  document.querySelectorAll('[data-board]').forEach((b) => (b.onclick = async () => {
    const res = await G.net.act('bus_board', { to: b.dataset.board });
    if (showResult(res)) { G.myBusDest = b.dataset.board; closeSheet(); sfx('bus'); }
  }));
  if (!ready) setTimeout(() => { if (G.modal === 'bus') openBusStop(); }, Math.min(5000, Math.max(800, here.eta * 1000)));
}

// Khách lên / xuống xe NPC.
export function onCarrier(m) {
  const mesh = m.pid === G.pid ? G.self?.mesh : G.players.get(m.pid)?.mesh;
  if (mesh) mesh.userData.carrier = m.carrier ? { id: m.carrier, kind: m.kind, seat: m.seat } : null;
  if (m.pid === G.pid) { G.myCarrier = m.carrier ? { id: m.carrier, kind: m.kind } : null; if (!m.carrier) G.myBusDest = null; }
}

function updateRideCard() {
  const card = $('rideCard'), c = G.myCarrier;
  if (!c && G.myRideOf) { // ngồi ghế khách trên xe người chơi khác: server có seat_leave, cần nút để xuống
    const k = 'seat' + G.myRideOf;
    if (card.dataset.k !== k) {
      card.dataset.k = k;
      card.innerHTML = `<small>ĐI NHỜ XE</small><b>🚗 Đang ngồi xe của ${esc(G.players.get(G.myRideOf)?.name || 'bạn')}</b><button class="btn sec" id="rideOff">Xuống xe</button>`;
      $('rideOff').onclick = async () => showResult(await G.net.act('seat_leave'));
    }
    card.hidden = false;
    return;
  }
  if (!c) { if (!card.hidden) { card.hidden = true; card.dataset.k = ''; } return; }
  const a = G.actors.get(c.id);
  const html = c.kind === 'bus'
    ? `<small>XE BUÝT VÒNG LÀNG</small><b>🚌 ${G.myBusDest ? 'Xuống ở bến ' + esc(G.zones[G.myBusDest]?.title || '') : 'Đi vòng làng'}</b><button class="btn sec" id="rideOff">Xuống ở bến kế</button>`
    : `<small>XE ÔM</small><b>🛵 ${esc(a?.info.name || 'Xe ôm')} đang chở bạn</b><button class="btn sec" id="rideOff">Xuống đây</button>`;
  if (card.dataset.k !== c.kind + (G.myBusDest || '')) {
    card.dataset.k = c.kind + (G.myBusDest || '');
    card.innerHTML = html;
    $('rideOff').onclick = async () => showResult(await G.net.act(c.kind === 'bus' ? 'bus_alight' : 'xeom_leave'));
  }
  card.hidden = false;
}

// ---------- Trận biểu diễn ở Sân tỷ thí ----------

let show = null;
// Sân khấu: giữa Sân tỷ thí, hoặc bãi cỏ trống giữa ao và lối vào Ao và vườn khi có ngày hội (PLAN_LOGIC_GAME 9.3).
const STAGE = { arena: [0, 0], park: [6, 13] };
function arenaPoint(x, z) {
  const where = show?.m.where || 'arena', zone = G.world.zones[where], [ox, oz] = STAGE[where] || [0, 0];
  return zone ? zone.group.localToWorld(new T.Vector3(ox + x, 0, oz + z)) : null;
}
export function onScene(m) {
  const where = m.where || 'arena';
  if (m.kind !== 'arena_show' || !G.world.zones[where]) return;
  stopShow();
  const rnd = mulberry32(m.seed >>> 0), f = [];
  const looks = [{ color: 'trang', neck: 'no_do' }, { color: 'nau', neck: 'khan_ran' }];
  for (let i = 0; i < 2; i++) { const c = chicken(looks[i]); G.world.scene.add(c); f.push(c); }
  // Lịch các pha mổ từ seed: mọi client dựng cùng một trận.
  const hits = [];
  for (let t = 5; t < 51; t += 1.1 + rnd() * 0.7) hits.push({ t, who: rnd() < 0.5 ? 0 : 1 });
  show = { m, f, hits, t0: performance.now() / 1000 - (m.since || 0), said: new Set() };
  // Ngày hội: Chó Mực đứng ở sân tỷ thí nên sân khấu ao có một bạn chó trọng tài riêng.
  if (where !== 'arena') {
    const ref = buildDog(); ref.position.copy(arenaPoint(0, -2.6)); ref.lookAt(arenaPoint(0, 0)); G.world.scene.add(ref);
    show.ref = ref; f.push(ref);
  }
}
export function onSceneEnd(m) { if (show && (!m.id || show.m.id === m.id)) stopShow(); }
function stopShow() { if (!show) return; for (const c of show.f) G.world.scene.remove(c); show = null; }

function updateShow() {
  if (!show) return;
  const t = performance.now() / 1000 - show.t0, m = show.m, [a, b] = show.f;
  if (t > m.dur) { stopShow(); return; }
  const near = Math.min(1, Math.max(0, (t - 0.5) / 3));
  const pa = arenaPoint(-3.6 + 2.3 * near, 0.2), pb = arenaPoint(3.6 - 2.3 * near, 0.2);
  a.position.copy(pa); b.position.copy(pb);
  a.lookAt(pb.x, 0, pb.z); b.lookAt(pa.x, 0, pa.z);
  for (const c of [a, b]) { const p = c.userData.parts; p.rig.rotation.set(0, 0, 0); p.rig.position.y = 0; p.head.rotation.x = 0; p.wings[0].rotation.z = p.wings[1].rotation.z = 0; }
  // Pha mổ: kẻ tấn công lao tới 0,6 m rồi lùi, người bị mổ giật lùi.
  for (const h of show.hits) {
    const u = t - h.t;
    if (u < 0 || u > 0.5 || t > 52) continue;
    const atk = show.f[h.who], def = show.f[1 - h.who], s = Math.sin(u / 0.5 * Math.PI);
    const dir = new T.Vector3().subVectors(def.position, atk.position).setY(0).normalize();
    atk.position.addScaledVector(dir, s * 0.6); atk.userData.parts.head.rotation.x = s * 0.9;
    def.position.addScaledVector(dir, s * 0.25); def.userData.parts.rig.position.y = s * 0.15;
  }
  if (t > 52) {
    const win = show.f[m.winner], lose = show.f[1 - m.winner], u = Math.min(1, (t - 52) / 0.3);
    lose.userData.parts.rig.rotation.z = (Math.PI / 2) * u * (t < 59 ? 1 : Math.max(0, 1 - (t - 59)));
    lose.userData.parts.rig.position.y = 0.3 * u * (t < 59 ? 1 : 0);
    win.userData.parts.wings[0].rotation.z = -1.1; win.userData.parts.wings[1].rotation.z = 1.1;
    win.userData.parts.rig.position.y = Math.abs(Math.sin(t * 6)) * 0.25;
    if (!show.said.has('ko')) { show.said.add('ko'); bubble(lose, '💫', { emo: true, ttl: 6 }); }
  }
  // Lời trọng tài, đấu sĩ, khán giả.
  const crowd = G.world.zones[m.where || 'arena']?.scenery?.ambient || [];
  for (const [i, c] of m.cues.entries()) {
    if (show.said.has(i) || t < c.t) continue;
    show.said.add(i);
    let who = null, name = '';
    if (c.who === 'ref') { who = show.ref || itemMesh('cho_muc'); name = 'Chó Mực'; }
    else if (c.who === 'f0' || c.who === 'f1') { who = show.f[+c.who[1]]; name = m.names[+c.who[1]]; }
    else if (crowd.length) { const r = crowd[(+c.who.slice(1)) % crowd.length]; who = r.model; name = r.name; }
    if (who && G.self && who.getWorldPosition(new T.Vector3()).distanceTo(G.self.pos) < 45) bubble(who, c.text, { name, ttl: 3.6 });
  }
}

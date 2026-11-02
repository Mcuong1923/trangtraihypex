// Đất, nhà, vật liệu, trồng cây và cài đặt NPC (PLAN_LOGIC_GAME 11, 13, 14, 5.2). Máy chủ quyết định giá,
// thời gian và tài sản; ở đây chỉ hiện bảng và gửi ý định.
import { G, $, esc, openSheet, closeSheet, toast, showResult } from './ui.js';
import { mat } from './models.js';

const T = window.THREE;
const act = (a, args) => G.net.act(a, args);
const fmt = (n) => Number(n).toLocaleString('vi-VN');
const itemOf = (id) => G.catalog.items.find((x) => x.id === id);
const need = (items) => Object.entries(items || {}).map(([id, n]) => `${itemOf(id)?.emo || ''} ${n} ${itemOf(id)?.name || id}`).join(', ');
const have = (items) => Object.entries(items || {}).every(([id, n]) => (G.me.items?.[id] || 0) >= n);
const MAT = { rom: 'Rơm', go: 'Gỗ', gach: 'Gạch' };
const npcBox = (face, bg, who, text) => `<div class="npcbox"><div class="npcface" style="background:${bg}">${face}</div><div><div class="who">${esc(who)}</div><p>${text}</p></div></div>`;

// ---------- Ông Chủ Đầu Tư: bán lô đất ----------

export async function dialogInvestor() {
  const r = await act('land_info');
  if (!showResult(r, { quiet: true })) return;
  const d = r.data, me = G.me;
  const lots = d.tiers.map((t, i) => {
    const owned = i <= d.tier;
    return `<div class="row"><div class="ico" style="background:#e8f0dc">📐</div><div class="meta"><div class="nm">${esc(t.name)} ${i === d.tier ? '<span class="rtag" style="background:var(--good)">Đang có</span>' : ''}</div>
      <div class="sub">${t.halfX * 2} × ${t.halfZ * 2} m · ${t.cost ? '🌾 ' + fmt(t.cost) : 'miễn phí'} · cần cấp ${t.level}</div></div>
      <div class="act">${owned ? '' : `<button class="mini hot" data-lot="${i}" ${me.level < t.level ? 'disabled' : ''}>Mua lô này</button>`}</div></div>`;
  }).join('');
  const places = d.places.map((p) => `<button class="tab" data-place="${p.id}" aria-selected="${p.id === d.place}">${esc(p.name)}${p.cost ? ` +${p.cost}` : ''}</button>`).join('');
  let place = d.place;
  openSheet('investor', '🐓 Văn phòng đất nền · Ông Chủ Đầu Tư', `
    ${npcBox('🐓', '#f0e2b0', 'Ông Chủ Đầu Tư', me.thoc < d.tiers[Math.min(d.tiers.length - 1, d.tier + 1)].cost ? 'Chưa đủ thóc à? Không sao, đào thêm vài con giun vàng là có ngay!' : 'Đất ven ao, view triệu đô, sổ đỏ… à sổ rơm chính chủ! Mua ngay kẻo hết lô đẹp!')}
    <h2>Vị trí <span class="mut" style="font-weight:600">· chỉ đổi cảnh quanh nhà riêng</span></h2><div class="tabs">${places}</div>
    <h2>Lô đất</h2>${lots}
    <button class="btn block sec" id="placeOnly" ${place === d.place ? 'disabled' : ''}>Chỉ đổi vị trí</button>
    <div class="note">Nhà riêng là phòng của riêng bạn nên không bao giờ hết lô. Mua xong, vào nhà (biển “Chuồng riêng & thăm bạn”) rồi gặp Bò Mộc để xây nhà bằng rơm, gỗ hoặc gạch.</div>`);
  document.querySelectorAll('[data-place]').forEach((b) => (b.onclick = () => {
    place = b.dataset.place;
    document.querySelectorAll('[data-place]').forEach((x) => x.setAttribute('aria-selected', x === b));
    $('placeOnly').disabled = place === d.place;
  }));
  const buy = async (tier) => { const res = await act('land_buy', { i: tier, mode: place }); if (showResult(res)) dialogInvestor(); };
  document.querySelectorAll('[data-lot]').forEach((b) => (b.onclick = () => buy(+b.dataset.lot)));
  $('placeOnly').onclick = () => buy(d.tier);
}

// ---------- Bò Mộc ở Xóm: tiệm vật liệu ----------

export function dialogMaterials() {
  const me = G.me;
  const rows = G.catalog.items.filter((it) => it.shop === 'vatlieu').map((it) => `<div class="row"><div class="ico" style="background:#f1e3c6">${it.emo}</div>
    <div class="meta"><div class="nm">${esc(it.name)} <span class="mut">· kho ${me.items?.[it.id] || 0}</span></div><div class="sub">🌾 ${it.buy} / món</div></div>
    <div class="act">${[1, 5, 10].map((n) => `<button class="mini ${n === 10 ? 'hot' : 'sec'}" data-mat="${it.id}" data-n="${n}" ${me.thoc < it.buy * n ? 'disabled' : ''}>+${n}</button>`).join('')}</div></div>`).join('');
  const designs = G.catalog.houseDesigns.map((d) => `<li><b>${esc(d.name)}</b> (cần ${esc(G.catalog.landTiers[d.minTier].name)}): ${Object.entries(d.recipes).map(([m, r]) => `${MAT[m]} — 🌾 ${fmt(r.thoc)} + ${need(r.items)}, ${r.minutes} phút`).join('; ')}</li>`).join('');
  openSheet('materials', '🐮 Bò Mộc · tiệm vật liệu', `
    ${npcBox('🐮', '#dfe9f5', 'Bò Mộc', 'Rơm thì nhanh, gỗ thì chắc, gạch thì sang — còn thóc thì… phải có! Gom đủ vật liệu rồi vào nhà gọi tôi xây.')}
    <p class="mut">Ví 🌾 ${fmt(me.thoc)} · kho ${me.storageCount}/${me.storageCap}</p>${rows}
    <h2>Nhà xây được</h2><ul class="rewardlist">${designs}</ul>
    <div class="note">Vật liệu còn có trong quà lên cấp, điểm danh, và khi đào ở ụ “Gốc cây” (gỗ) hoặc “Đất ẩm” (đá).</div>`);
  document.querySelectorAll('[data-mat]').forEach((b) => (b.onclick = async () => { b.disabled = true; if (showResult(await act('item_buy', { id: b.dataset.mat, i: +b.dataset.n }))) dialogMaterials(); }));
}

// ---------- Xây nhà trong nhà riêng (gọi từ hộp thoại Bò Mộc của social.js) ----------

export function houseSection(h) {
  const me = G.me, c = h.coop;
  if (c && c.state === 'building') return `<p>🔨 Bò Mộc đang xây <b>${esc(designName(c.design))}</b> bằng ${MAT[c.material]?.toLowerCase()} — còn <b id="buildLeft">${Math.ceil(h.buildLeft)}</b> giây. Bạn đi chơi, tôi vẫn làm.</p>`;
  let html = '';
  if (c && c.state === 'done') {
    html += `<p>✅ <b>${esc(designName(c.design))}</b> bằng ${MAT[c.material]?.toLowerCase()} · ngủ ổ trong nhà hồi <b>+${h.regen} độ no/phút</b> · tối đa <b>${h.cap}</b> khách.</p>`;
    const d = G.catalog.houseDesigns.find((x) => x.id === c.design);
    const ups = Object.entries(d.recipes).filter(([m]) => ({ rom: 0, go: 1, gach: 2 })[m] > ({ rom: 0, go: 1, gach: 2 })[c.material]);
    html += ups.map(([m, r]) => {
      const cur = d.recipes[c.material], diff = {};
      for (const [id, n] of Object.entries(r.items)) if (n - (cur.items[id] || 0) > 0) diff[id] = n - (cur.items[id] || 0);
      return `<div class="row"><div class="ico" style="background:#f1e3c6">⬆️</div><div class="meta"><div class="nm">Nâng lên ${MAT[m]}</div><div class="sub">🌾 ${fmt(Math.floor(r.thoc / 2))} + ${need(diff) || 'không cần thêm vật liệu'}</div></div>
        <div class="act"><button class="mini hot" data-upmat="${m}" ${me.thoc < r.thoc / 2 || !have(diff) ? 'disabled' : ''}>Nâng cấp</button></div></div>`;
    }).join('');
  }
  const options = G.catalog.houseDesigns.filter((d) => !c || d.id !== c.design).flatMap((d) => Object.entries(d.recipes).map(([m, r]) => ({ d, m, r })));
  if (h.tier < 1) return html + '<p class="mut">Mua lô đất ở Văn phòng đất nền của Ông Chủ Đầu Tư (cạnh cổng xóm) rồi tôi mới xây được.</p>';
  html += `<p class="mut" style="margin-bottom:4px">Màu mái: ${Object.entries(G.catalog.roofs).map(([k, v]) => `<label style="margin-right:8px"><input type="radio" name="roof" value="${k}" ${k === (c?.roof || 'do') ? 'checked' : ''}> ${esc(v)}</label>`).join('')}</p>`;
  html += options.map(({ d, m, r }) => {
    const ok = h.tier >= d.minTier && me.thoc >= r.thoc && have(r.items);
    return `<div class="row"><div class="ico" style="background:#e8f0dc">🏠</div><div class="meta"><div class="nm">${esc(d.name)} · ${MAT[m]}</div>
      <div class="sub">🌾 ${fmt(r.thoc)} + ${need(r.items)} · ${r.minutes} phút${h.tier < d.minTier ? ` · cần ${esc(G.catalog.landTiers[d.minTier].name)}` : ''}</div></div>
      <div class="act"><button class="mini ${ok ? 'hot' : 'sec'}" data-build="${d.id}" data-mat="${m}" ${ok ? '' : 'disabled'}>Xây</button></div></div>`;
  }).join('');
  return html;
}
const designName = (id) => G.catalog.houseDesigns.find((d) => d.id === id)?.name || 'Chuồng';

export function bindHouseSection(refresh) {
  document.querySelectorAll('[data-build]').forEach((b) => (b.onclick = async () => {
    const roof = document.querySelector('input[name="roof"]:checked')?.value || 'do';
    b.disabled = true;
    if (showResult(await act('home_build', { id: roof, to: b.dataset.build, mode: b.dataset.mat }))) refresh();
  }));
  document.querySelectorAll('[data-upmat]').forEach((b) => (b.onclick = async () => { b.disabled = true; if (showResult(await act('home_upgrade_house', { mode: b.dataset.upmat }))) refresh(); }));
}

// ---------- Hạt giống & thu mua nông sản ở Quầy thức ăn ----------

export function seedsSection() {
  const me = G.me;
  const seeds = G.catalog.items.filter((it) => it.shop === 'thucan').map((it) => {
    const sd = G.catalog.seeds.find((s) => s.seed === it.id), crop = itemOf(sd?.crop);
    return `<div class="row"><div class="ico" style="background:#e8f0dc">🌱</div><div class="meta"><div class="nm">${esc(it.name)} <span class="mut">· kho ${me.items?.[it.id] || 0}</span></div>
      <div class="sub">🌾 ${it.buy} · lớn ${Math.round(sd.sec / 60)} phút → ${sd.yield} ${esc(crop?.name || '')}</div></div>
      <div class="act"><button class="mini sec" data-seed="${it.id}" ${me.thoc < it.buy ? 'disabled' : ''}>Mua 1</button></div></div>`;
  }).join('');
  const produce = G.catalog.items.filter((it) => it.group === 'nongsan' && it.sell > 0 && (me.items?.[it.id] || 0) > 0).map((it) => `<div class="row"><div class="ico" style="background:#fff0cf">${it.emo}</div>
    <div class="meta"><div class="nm">${esc(it.name)} ×${me.items[it.id]}</div><div class="sub">thu ${it.sell} thóc / món</div></div>
    <div class="act"><button class="mini good" data-sellitem="${it.id}">Bán +${it.sell}</button></div></div>`).join('');
  return `<h2>Hạt giống</h2>${seeds}${produce ? `<h2>Thu mua nông sản</h2>${produce}` : ''}<div class="note">Gieo ở 12 luống cộng đồng bên phải ao (Ao và vườn) hoặc luống rau trong nhà riêng. Tưới một lần để cây lớn nhanh hơn.</div>`;
}
export function bindSeeds(refresh) {
  document.querySelectorAll('[data-seed]').forEach((b) => (b.onclick = async () => { b.disabled = true; if (showResult(await act('item_buy', { id: b.dataset.seed, i: 1 }))) refresh(); }));
  document.querySelectorAll('[data-sellitem]').forEach((b) => (b.onclick = async () => { b.disabled = true; if (showResult(await act('item_sell', { id: b.dataset.sellitem, i: 1 }))) refresh(); }));
}

// ---------- Luống rau ----------

const ago = (t) => (Date.parse(t) - Date.now()) / 1000;
export function dialogBed(it) {
  const key = it.id.startsWith('home:') ? it.id : 'park:' + it.id;
  const pl = G.me.beds?.[key];
  const seeds = G.catalog.seeds.filter((s) => (G.me.items?.[s.seed] || 0) > 0);
  let body;
  if (!pl) {
    const holding = Object.keys(G.me.beds || {}).find((k) => k.startsWith('park:'));
    body = key.startsWith('park:') && holding ? '<p class="mut">Bạn đang giữ một luống cộng đồng khác. Thu hoạch luống đó trước nhé.</p>'
      : seeds.length ? `<p>Chọn hạt để gieo:</p>${seeds.map((s) => `<button class="btn block sec" data-plant="${s.seed}">🌱 ${esc(itemOf(s.seed).name)} ×${G.me.items[s.seed]} · ${Math.round(s.sec / 60)} phút</button>`).join('')}
        ${(G.me.items?.phan_u || 0) > 0 ? '<label class="mut"><input type="checkbox" id="fert"> Bón 1 phân ủ (+1 nông sản)</label>' : ''}`
        : '<p class="mut">Bạn chưa có hạt giống. Mua ở Quầy thức ăn (Chợ thóc).</p>';
  } else {
    const left = ago(pl.ready), wilt = ago(pl.wilt), crop = itemOf(G.catalog.seeds.find((s) => s.seed === pl.seed)?.crop);
    body = left > 0 ? `<p>🌱 ${esc(itemOf(pl.seed).name)} đang lớn — chín sau <b>${Math.ceil(left / 60)} phút</b>.</p>${pl.watered ? '<p class="mut">Đã tưới.</p>' : '<button class="btn block" id="water">💧 Tưới nước (nhanh hơn 25%)</button>'}`
      : wilt > 0 ? `<p>${crop?.emo || '🌿'} ${esc(crop?.name || '')} chín rồi! Thu trong ${Math.ceil(wilt / 60)} phút kẻo héo.</p><button class="btn block hot" id="harvest">🧺 Thu hoạch</button>`
        : `<p>🥀 Cây héo mất rồi. Thu dọn để lấy 1 phân ủ.</p><button class="btn block sec" id="harvest">Thu dọn</button>`;
  }
  openSheet('bed', key.startsWith('park:') ? '🌱 Luống cộng đồng' : '🌱 Luống rau nhà', body);
  document.querySelectorAll('[data-plant]').forEach((b) => (b.onclick = async () => { if (showResult(await act('plant', { id: it.id, mode: b.dataset.plant, eat: !!$('fert')?.checked }))) closeSheet(); }));
  if ($('water')) $('water').onclick = async () => { if (showResult(await act('water', { id: it.id }))) closeSheet(); };
  if ($('harvest')) $('harvest').onclick = async () => { if (showResult(await act('harvest', { id: it.id }))) closeSheet(); };
}

// Cây của mình trên các luống: mầm → bụi → chín (đúng loại nông sản) → héo.
const plants = new Map();
let plantAt = 0;
export function updatePlants() {
  if (!G.me || performance.now() - plantAt < 1500) return;
  plantAt = performance.now();
  const beds = G.me.beds || {}, seen = new Set();
  for (const [key, pl] of Object.entries(beds)) {
    const pos = bedPos(key);
    if (!pos) continue;
    seen.add(key);
    const total = (Date.parse(pl.ready) - Date.parse(pl.at)) / 1000, grown = (Date.now() - Date.parse(pl.at)) / 1000;
    const stage = Date.now() > Date.parse(pl.wilt) ? 'wilt' : grown >= total ? 'ripe' : grown > total * 0.35 ? 'bush' : 'sprout';
    const cur = plants.get(key);
    if (cur && cur.stage === stage) continue;
    if (cur) G.world.scene.remove(cur.mesh);
    const mesh = plantMesh(pl.seed, stage); mesh.position.set(pos.x, 0.32, pos.z); G.world.scene.add(mesh);
    plants.set(key, { stage, mesh });
  }
  for (const [key, p] of plants) if (!seen.has(key)) { G.world.scene.remove(p.mesh); plants.delete(key); }
}
function bedPos(key) {
  if (key.startsWith('park:')) { const it = G.zones?.park?.items?.find((i) => i.id === key.slice(5)); return G.worldMap && it ? it : null; }
  if (G.zone !== 'home' || !G.home) return null;
  const o = G.home.objects.find((x) => 'home:' + x.id === key);
  return o ? { x: o.x, z: o.z } : null;
}
function plantMesh(seed, stage) {
  const g = new T.Group();
  const leaf = mat(stage === 'wilt' ? 0x8a6a3a : 0x5f9a3a);
  if (stage === 'sprout') { for (let i = 0; i < 3; i++) { const s = new T.Mesh(new T.ConeGeometry(0.05, 0.25, 5), leaf); s.position.set((i - 1) * 0.12, 0.12, 0); s.rotation.z = (i - 1) * 0.4; g.add(s); } return g; }
  const bush = new T.Mesh(new T.IcosahedronGeometry(stage === 'wilt' ? 0.22 : 0.3, 1), leaf); bush.position.y = 0.22; bush.scale.y = 0.75; g.add(bush);
  if (stage === 'ripe') {
    const crop = { hat_cai: [0x8fc25a, 0.18, 'sphere'], hat_ca_rot: [0xe9832a, 0.08, 'cone'], hat_bi_ngo: [0xe9833a, 0.3, 'sphere'], hat_hoa: [0xe2577a, 0.1, 'sphere'] }[seed] || [0xe7b23e, 0.15, 'sphere'];
    for (let i = 0; i < (seed === 'hat_bi_ngo' ? 1 : 3); i++) {
      const m = new T.Mesh(crop[2] === 'cone' ? new T.ConeGeometry(crop[1], 0.3, 8) : new T.SphereGeometry(crop[1], 12, 10), mat(crop[0]));
      m.position.set((i - 1) * 0.2, crop[2] === 'cone' ? 0.1 : 0.2 + crop[1] * 0.5, 0.12); if (crop[2] === 'cone') m.rotation.x = Math.PI; g.add(m);
    }
  }
  return g;
}

// ---------- Cài đặt NPC (PLAN_LOGIC_GAME 5.2) ----------

export function prefsSection() {
  const p = G.me.prefs || {};
  const ch = p.chatter || 'full';
  return `<h2>NPC trong làng</h2><div class="tabs">${[['full', 'Đầy đủ'], ['normal', 'Vừa'], ['low', 'Ít']].map(([k, v]) => `<button class="tab" data-chatter="${k}" aria-selected="${ch === k}">${v}</button>`).join('')}</div>
    <label class="mut" style="display:block;margin-top:6px"><input type="checkbox" id="npcTease" ${p.noTease ? '' : 'checked'}> Cho NPC trêu mình (thiếu gia đi ô tô, hai mẹ con gà…)</label>
    <p class="mut">Mức “Ít” chỉ giữ lời hướng dẫn, lời mời xe ôm và lời của chủ quán.</p>`;
}
export function bindPrefs() {
  const save = async (chatter) => showResult(await act('prefs', { mode: chatter, eat: $('npcTease').checked }), { quiet: true });
  document.querySelectorAll('[data-chatter]').forEach((b) => (b.onclick = () => { document.querySelectorAll('[data-chatter]').forEach((x) => x.setAttribute('aria-selected', x === b)); save(b.dataset.chatter); }));
  $('npcTease').onchange = () => save(document.querySelector('[data-chatter][aria-selected="true"]')?.dataset.chatter || 'full');
}

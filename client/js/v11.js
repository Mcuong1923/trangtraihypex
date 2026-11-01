// Giao diện V1.1/V1.2: trạm ấp trứng, bến xe & Ven Suối, xe riêng, giao dịch, hợp đồng thợ, Hội Gà.
// Mọi quyết định do máy chủ; các bảng chỉ gửi ý định và hiển thị kết quả.
import { G, $, esc, openSheet, setSheetBody, closeSheet, toast, showResult, wormById } from './ui.js';
import { buildVehicle, buildChick } from './models.js';
import { sfx } from './audio.js';

const T = window.THREE;
const act = (a, args) => G.net.act(a, args);
const raw = (m) => G.net.raw(m);
const fmt = (n) => Number(n).toLocaleString('vi-VN');
const vehicleDef = (id) => G.catalog.vehicles.find((v) => v.id === id);
const nearKind = (kind) => (G.zones?.[G.zone]?.items || []).some((it) => it.kind === kind && Math.hypot(G.self.pos.x - it.x, G.self.pos.z - it.z) < it.r);

// ---------- Hiển thị xe, gà con, tên hội ----------

// Gắn xe dưới chân người lái (xe là con của mesh gà nên đi và xoay theo).
export function setVehicle(mesh, kind) {
  if (mesh.userData.vehicle) { mesh.remove(mesh.userData.vehicle); mesh.userData.vehicle = null; }
  mesh.userData.riding = kind || null;
  if (!kind) return;
  const v = buildVehicle(kind);
  v.position.y = 0;
  mesh.add(v);
  mesh.userData.vehicle = v;
}
export const rideHeight = (kind) => ({ xe_dap: 0.45, xe_may: 0.55, o_to: 0.45 }[kind] || 0);

const pets = new Map(); // mesh chủ → { chick, name }
export function setPet(ownerMesh, pet) {
  const cur = pets.get(ownerMesh);
  if (cur && (!pet || cur.color !== pet.color || cur.name !== pet.name)) { G.world.scene.remove(cur.chick); pets.delete(ownerMesh); }
  if (!pet || pets.has(ownerMesh) || G.hidePets && ownerMesh !== G.self?.mesh) return;
  const chick = buildChick(pet.color);
  chick.position.copy(ownerMesh.position).add(new T.Vector3(0.8, 0, -0.6));
  G.world.scene.add(chick);
  pets.set(ownerMesh, { chick, color: pet.color, name: pet.name, phase: Math.random() * 6 });
}
export function dropPet(ownerMesh) { const p = pets.get(ownerMesh); if (p) { G.world.scene.remove(p.chick); pets.delete(ownerMesh); } }
export function clearPets(except) { for (const m of [...pets.keys()]) if (m !== except) dropPet(m); }

// Gà con đi theo chủ, giữ khoảng cách và nhún nhảy (mục 12).
export function updatePets(dt) {
  for (const [owner, p] of pets) {
    if (!owner.parent) { dropPet(owner); continue; }
    const back = new T.Vector3(Math.sin(owner.rotation.y + 2.4) * 0.9, 0, Math.cos(owner.rotation.y + 2.4) * 0.9);
    const target = owner.position.clone().add(back);
    const d = p.chick.position.distanceTo(target);
    p.chick.position.lerp(target, Math.min(1, dt * (d > 4 ? 10 : 3.5)));
    if (d > 0.08) { p.phase += dt * 14; p.chick.rotation.y = Math.atan2(target.x - p.chick.position.x, target.z - p.chick.position.z); }
    p.chick.position.y = d > 0.08 ? Math.abs(Math.sin(p.phase)) * 0.08 : 0;
  }
}

// ---------- Trạm ấp trứng ----------

export function dialogDuck() {
  const me = G.me, bal = G.catalog.balance;
  const claimable = G.catalog.eggMilestones.filter((lv) => me.level >= lv && !me.eggsClaimed?.[lv]).length;
  const inc = me.incubation;
  const hatchLeft = inc ? Math.max(0, me.hatchIn - since(me._at)) : 0;
  const pets = (me.companions || []).map((c) => {
    const lv = c.affinity >= 150 ? 'Thân thiết' : c.affinity >= 50 ? 'Thân' : 'Mới quen';
    const on = me.following === c.id;
    return `<div class="row"><div class="ico" style="background:${c.color}55">🐥</div>
      <div class="meta"><div class="nm">${esc(c.name)} ${on ? '<span class="rtag" style="background:var(--good)">Đang đi theo</span>' : ''}</div>
        <div class="sub">Thân thiết ${c.affinity} · ${lv}</div></div>
      <div class="act"><button class="mini ${on ? 'sec' : 'hot'}" data-follow="${on ? '' : c.id}">${on ? 'Để ở nhà' : 'Đi theo'}</button>
        <button class="mini sec" data-care="${c.id}" data-mode="pet">Vuốt ve</button><button class="mini sec" data-care="${c.id}" data-mode="feed">Cho ăn</button></div></div>`;
  }).join('');
  openSheet('duck', '🦆 Trạm ấp trứng · Vịt Bông', `
    <div class="npcbox"><div class="npcface" style="background:#f5f2e9">🦆</div><div><div class="who">Vịt Bông</div>
      <p>Trứng đầu tiên cô tặng qua nhiệm vụ ở cấp ${G.catalog.eggMilestones.join(', ')} — không bán trứng ngẫu nhiên đâu nhé. Ấp ${dur(bal.hatchSec)}, cháu đi chơi hay offline trứng vẫn ấm.</p></div></div>
    <div class="kvs" style="margin:12px 0"><b>Trứng đang có</b><span>🥚 ${me.eggs || 0}</span><b>Bạn đồng hành</b><span>${(me.companions || []).length}/${G.catalog.companionMax}</span></div>
    <div class="choices">
      ${claimable ? `<button class="btn" id="eggClaim">🎁 Nhận ${claimable} quả trứng mốc cấp</button>` : ''}
      ${inc ? (hatchLeft > 0 ? `<div class="note">🔥 Trứng đang ấp — còn <b id="hatchLeft">${Math.ceil(hatchLeft)}</b> giây.</div>`
        : `<div class="row"><div class="meta"><label class="nm" for="petName">Trứng nở rồi! Đặt tên cho gà con</label><input type="text" id="petName" maxlength="14" placeholder="vd: Bông Con" style="margin-top:6px"></div><div class="act"><button class="mini hot" id="hatchBtn" data-id="${inc.id}">Đón gà con</button></div></div>`)
        : (me.eggs > 0 ? '<button class="btn sec" id="incBtn">🪺 Đặt trứng vào ổ ấp</button>' : '')}
    </div>
    <h2>Bạn đồng hành</h2>${pets || '<p class="mut">Chưa có gà con nào. Lên cấp 5 để nhận quả trứng đầu tiên.</p>'}
    <p class="mut">Bỏ game không làm bạn đồng hành ốm hay mất đi. Chăm sóc mỗi ngày để tăng thân thiết.</p>`);
  if ($('eggClaim')) $('eggClaim').onclick = async () => { if (showResult(await act('eggs_claim'))) dialogDuck(); };
  if ($('incBtn')) $('incBtn').onclick = async () => { if (showResult(await act('incubate'))) dialogDuck(); };
  if ($('hatchBtn')) $('hatchBtn').onclick = async () => {
    const r = await act('hatch', { id: $('hatchBtn').dataset.id, text: $('petName').value || 'Gà Con' });
    if (showResult(r)) { sfx('rare'); dialogDuck(); }
  };
  document.querySelectorAll('[data-follow]').forEach((b) => (b.onclick = async () => { if (showResult(await act('follow', { id: b.dataset.follow }), { quiet: true })) dialogDuck(); }));
  document.querySelectorAll('[data-care]').forEach((b) => (b.onclick = async () => { if (showResult(await act('care', { id: b.dataset.care, mode: b.dataset.mode }))) dialogDuck(); }));
  if (inc && hatchLeft > 0) {
    const end = performance.now() + hatchLeft * 1000;
    (function tick() { const el = $('hatchLeft'); if (!el) return; const l = (end - performance.now()) / 1000; if (l <= 0) return dialogDuck(); el.textContent = Math.ceil(l); requestAnimationFrame(tick); })();
  }
}

// ---------- Bến xe & vùng ----------

export function onBus(m) { G.bus = m.bus; G.busAt = performance.now(); if (G.modal === 'bus') openBus(); }
const since = (t) => (performance.now() - (t || performance.now())) / 1000;
const dur = (sec) => (sec >= 3600 ? `${Math.round(sec / 3600)} giờ` : sec >= 60 ? `${Math.round(sec / 60)} phút` : `${Math.round(sec)} giây`);

export function openBus() {
  const me = G.me, region = G.region;
  const to = region === 'hypex' ? 'vensuoi' : 'hypex';
  const reg = G.catalog.regions[to];
  const busRaw = G.bus?.[to];
  const bus = busRaw && { ...busRaw, departIn: Math.max(0, busRaw.departIn - since(G.busAt)) };
  const bk = me.booking;
  const atStop = nearKind('bus_stop');
  openSheet('bus', '🚌 Trạm xe buýt Chim Chíp', `
    <div class="row"><div class="ico" style="background:#fff0cf">🗺️</div><div class="meta"><div class="nm">Tuyến ${esc(G.catalog.regions[region].title)} → ${esc(reg.title)}</div>
      <div class="sub">Tuyến gần · ${G.catalog.busSeats} ghế · chờ tối đa ${G.catalog.balance.busWaitSec} giây · cần cấp ${reg.level}</div></div></div>
    <div class="kvs" style="margin:10px 0"><b>Lượt miễn phí hôm nay</b><span>${me.tripsLeft}/${G.catalog.freeTrips} (reset 0:00)</span>
      <b>Giá khi hết lượt</b><span>🌾 ${G.catalog.busFare}</span>
      <b>Chuyến sắp chạy</b><span>${bus ? `còn <b id="busLeft">${Math.ceil(bus.departIn)}</b> giây · ${bus.seats}/${G.catalog.busSeats} ghế đã giữ` : 'Chưa có — đặt vé để mở chuyến'}</span></div>
    ${bk ? `<div class="note">🎫 Vé ${bk.free ? 'miễn phí' : bk.paid + ' thóc'} đi ${esc(G.catalog.regions[bk.to].title)}. Đứng ở bến chờ xe; rời bến trước giờ chạy sẽ tự huỷ và hoàn.</div>
      <button class="btn sec block" id="cancelBk">Huỷ vé</button>`
      : `<button class="btn block" id="bookBtn" ${!atStop || me.level < reg.level ? 'disabled' : ''}>${me.tripsLeft > 0 ? 'Đặt vé (dùng lượt miễn phí)' : `Đặt vé · ${G.catalog.busFare} thóc`}</button>
        ${atStop ? '' : '<p class="mut">Đứng cạnh Trạm xe buýt để đặt vé.</p>'}${me.level < reg.level ? `<p class="mut">Cần cấp ${reg.level} để tới ${esc(reg.title)}.</p>` : ''}`}
    <p class="mut">Không có ghế thì không trừ tiền hay lượt. Huỷ trước giờ chạy được hoàn đúng một lần. Về sảnh chuồng Hypex luôn miễn phí.</p>`);
  if ($('bookBtn')) $('bookBtn').onclick = async () => { $('bookBtn').disabled = true; if (showResult(await act('book'))) openBus(); };
  if ($('cancelBk')) $('cancelBk').onclick = async () => { if (showResult(await act('cancel_booking'))) openBus(); };
  if (bus) {
    const end = performance.now() + bus.departIn * 1000;
    (function tick() { const el = $('busLeft'); if (!el) return; el.textContent = Math.max(0, Math.ceil((end - performance.now()) / 1000)); requestAnimationFrame(tick); })();
  }
}

export function onTrip(m) {
  closeSheet();
  const el = $('tripOverlay');
  el.innerHTML = `<div class="card" style="text-align:center"><div style="font-size:3rem">${m.vehicle === 'xe_may' ? '🛵' : m.vehicle === 'o_to' ? '🚗' : '🚌'}</div><h1>Đang tới ${esc(m.title)}…</h1><p class="mut">Đường làng rợp bóng tre, ngồi yên nhé!</p></div>`;
  el.hidden = false;
  sfx('door');
  setTimeout(() => (el.hidden = true), 1800);
}

export function dialogBird() {
  openSheet('npc', 'Chim Chíp', `<div class="npcbox"><div class="npcface" style="background:#dbeefa">🐦</div><div><div class="who">Chim Chíp</div>
    <p>Chíp chíp! Xe buýt đi trại Ven Suối chạy liên tục, mỗi ngày có ${G.catalog.freeTrips} lượt miễn phí. Ven Suối có giun bùn, giun ngọc và cả giun rồng hiếm lắm đó! Có xe máy hay ô tô thì đi Đường liên trại, chở bạn cùng đi.</p></div></div>
    <div class="choices"><button class="choice" id="toBus">🚌 Xem tuyến & đặt vé</button><button class="choice" id="byeBird">Cảm ơn Chíp</button></div>`);
  $('toBus').onclick = openBus;
  $('byeBird').onclick = closeSheet;
}

export function dialogFrog() {
  const vs = G.catalog.worms.filter((w) => w.region === 'vensuoi');
  openSheet('npc', 'Ếch Xanh', `<div class="npcbox"><div class="npcface" style="background:#dff0d4">🐸</div><div><div class="who">Ếch Xanh</div>
    <p>Ộp! Đồng Ven Suối toàn bùn non, giun ở đây khác hẳn trại Hypex. Đào ở bãi bồi hay bùn ven suối đều được.</p></div></div>
    <div class="tblwrap"><table class="t" style="margin-top:12px"><tr><th>Giun đặc sản</th><th>Độ hiếm</th><th>Tỷ lệ</th><th>Bán</th><th>XP</th></tr>
    ${vs.map((w) => `<tr><td>${esc(w.name)}</td><td>${esc(w.rarity)}</td><td>${(w.prob * 100).toLocaleString('vi-VN')}%</td><td>${w.sell}</td><td>${w.xp}</td></tr>`).join('')}</table></div>
    <p class="mut">Muốn về trại Hypex: đi xe buýt ở bến, hoặc bấm “Về sảnh chuồng” (miễn phí).</p>`);
}

// ---------- Xe riêng ----------

export function dialogGoat() {
  const me = G.me;
  openSheet('goat', '🐐 Cửa hàng xe · Dê Thợ Máy', `
    <p class="mut" style="margin-top:0">Xe chỉ để đi lại và chở bạn; không húc, không gây sát thương. Không lái trong sảnh, khu đào, nhà riêng hay sân PK.</p>
    ${G.catalog.vehicles.map((v) => {
      const owned = me.ownedVehicles?.[v.id];
      return `<div class="row"><div class="ico" style="background:#f1e3c6">${v.emo}</div>
        <div class="meta"><div class="nm">${esc(v.name)} ${owned ? '<span class="rtag" style="background:var(--good)">Đã có</span>' : ''}</div>
          <div class="sub">${v.seats} ghế (chở ${v.seats - 1}) · nhanh ×${v.speed} · ${v.road ? 'đi được đường liên trại' : 'chỉ đi trong trại'} · cần cấp ${v.level} · ${v.phase}</div></div>
        <div class="act">${owned ? '' : `<button class="mini hot" data-buyv="${v.id}" ${me.level < v.level || me.thoc < v.cost ? 'disabled' : ''}>🌾 ${fmt(v.cost)}</button>`}</div></div>`;
    }).join('')}
    <p class="mut">Mua xong, ra <b>Điểm đỗ xe</b> để lấy xe.</p>`);
  document.querySelectorAll('[data-buyv]').forEach((b) => (b.onclick = async () => { b.disabled = true; if (showResult(await act('vehicle_buy', { id: b.dataset.buyv }))) { sfx('coin'); dialogGoat(); } }));
}

export function parkingMenu() {
  const me = G.me;
  const out = G.myRide;
  const owned = G.catalog.vehicles.filter((v) => me.ownedVehicles?.[v.id]);
  openSheet('parking', '🅿️ Điểm đỗ xe', out ? `
    <p>Bạn đang lái <b>${esc(vehicleDef(out)?.name)}</b>.</p>
    <div class="choices"><button class="btn sec" id="parkIn">Cất xe (khách xuống an toàn)</button></div>` : `
    ${owned.map((v) => `<div class="row"><div class="ico" style="background:#f1e3c6">${v.emo}</div><div class="meta"><div class="nm">${esc(v.name)}</div><div class="sub">${v.seats} ghế</div></div>
      <div class="act"><button class="mini hot" data-out="${v.id}">Lấy xe</button></div></div>`).join('') || '<p class="mut">Bạn chưa có xe. Ghé Cửa hàng xe của Dê Thợ Máy.</p>'}`);
  if ($('parkIn')) $('parkIn').onclick = async () => { if (showResult(await act('vehicle_in'))) closeSheet(); };
  document.querySelectorAll('[data-out]').forEach((b) => (b.onclick = async () => { if (showResult(await act('vehicle_out', { id: b.dataset.out }))) closeSheet(); }));
}

export function roadMenu() {
  const out = G.myRide;
  const v = out && vehicleDef(out);
  const to = G.region === 'hypex' ? 'vensuoi' : 'hypex';
  openSheet('road', '🛣️ Đường liên trại', `
    <p>Đi ${esc(G.catalog.regions[to].title)} bằng xe riêng: cả nhóm vào cùng một bản sao chuồng, ghế được giữ nguyên.</p>
    ${!v ? '<p class="mut">Lấy xe máy hoặc ô tô ở điểm đỗ trước.</p>' : !v.road ? '<p class="mut">Xe đạp chỉ đi trong trại; cần xe máy hoặc ô tô.</p>'
      : '<button class="btn block" id="driveGo">Lên đường</button>'}
    <p class="mut">Mỗi người trên xe phải đủ cấp vào vùng đích (cấp ${G.catalog.regions[to].level}). Đi xe riêng không tốn vé.</p>`);
  if ($('driveGo')) $('driveGo').onclick = async () => { const r = await act('drive_region'); if (!showResult(r)) return; };
}

export function onSeatInvite(m) {
  inviteBar(`🚗 ${m.from} mời bạn lên ${m.vehicle}`, 20, 'Lên xe', async () => showResult(await act('seat_accept', { target: m.driver })));
}

// ---------- Giao dịch ----------

export function onTradeInvite(m) {
  inviteBar(`🤝 ${m.from} mời bạn giao dịch`, 30, 'Mở', async () => showResult(await act('trade_accept', { id: m.id }), { quiet: true }));
}

let tradeDraft = null;
export function onTrade(t) {
  G.trade = t;
  if (!tradeDraft || tradeDraft.id !== t.id) tradeDraft = { id: t.id, thoc: t.offer[t.you].thoc || 0, worms: { ...(t.offer[t.you].worms || {}) } };
  const me = G.me, you = t.you, other = 1 - you;
  const offerText = (o) => [o.thoc ? `🌾 ${fmt(o.thoc)}` : '', ...Object.entries(o.worms || {}).filter(([, n]) => n > 0).map(([id, n]) => `${n} ${esc(wormById(id)?.name || id)}`)].filter(Boolean).join(' · ') || 'Chưa đưa gì';
  const mine = G.catalog.worms.filter((w) => (me.worms[w.id] || 0) > 0);
  openSheet('trade', `🤝 Giao dịch với ${esc(t.names[other])}`, `
    <div class="row"><div class="meta"><div class="nm">${esc(t.names[other])} đưa</div><div class="sub">${offerText(t.offer[other])}</div></div>
      <div class="act">${t.locked[other] ? '<span class="rtag" style="background:var(--good)">Đã khoá</span>' : '<span class="rtag" style="background:#8a8a8a">Đang chọn</span>'}${t.confirmed[other] ? ' <span class="rtag" style="background:var(--accent)">Đã xác nhận</span>' : ''}</div></div>
    <h2>Bạn đưa</h2>
    <div class="row"><div class="meta"><label class="nm" for="trThoc">Thóc (ví ${fmt(me.thoc)})</label></div><div class="act"><input type="number" id="trThoc" min="0" max="${me.thoc}" value="${tradeDraft.thoc}" style="width:110px;padding:8px;border-radius:10px;border:2px solid var(--line)"></div></div>
    ${mine.map((w) => `<div class="row"><div class="meta"><div class="nm">${esc(w.name)}</div><div class="sub">Đang có ${me.worms[w.id]}</div></div>
      <div class="act"><input type="number" data-trw="${w.id}" min="0" max="${me.worms[w.id]}" value="${tradeDraft.worms[w.id] || 0}" style="width:80px;padding:8px;border-radius:10px;border:2px solid var(--line)"></div></div>`).join('') || '<p class="mut">Không có giun để đưa.</p>'}
    <p class="mut">Chỉ trao đổi giun tự đào và thóc. Sửa đề nghị thì cả hai phải khoá lại. Cả hai khoá rồi xác nhận, máy chủ kiểm lại và đổi cùng lúc.</p>
    <div class="cart"><button class="mini sec" id="trUpdate">Cập nhật đề nghị</button>
      <button class="mini good" id="trLock" ${t.locked[you] ? 'disabled' : ''}>${t.locked[you] ? 'Đã khoá' : 'Khoá đề nghị'}</button>
      <button class="mini hot" id="trConfirm" ${!(t.locked[0] && t.locked[1]) || t.confirmed[you] ? 'disabled' : ''}>Xác nhận</button>
      <button class="mini sec" id="trCancel">Huỷ</button></div>`,
    { close: () => { if (G.trade) raw({ t: 'act', a: 'trade_cancel', id: G.trade.id, rid: 'tc' + Date.now() }); } });
  const read = () => {
    tradeDraft.thoc = Math.max(0, Math.floor(+$('trThoc').value || 0));
    document.querySelectorAll('[data-trw]').forEach((i) => (tradeDraft.worms[i.dataset.trw] = Math.max(0, Math.floor(+i.value || 0))));
  };
  $('trUpdate').onclick = async () => { read(); showResult(await act('trade_offer', { id: t.id, i: tradeDraft.thoc, offer: tradeDraft.worms }), { quiet: true }); };
  $('trLock').onclick = async () => showResult(await act('trade_lock', { id: t.id }), { quiet: true });
  $('trConfirm').onclick = async () => showResult(await act('trade_confirm', { id: t.id }));
  $('trCancel').onclick = () => closeSheet();
}
export function onTradeEnd(m) {
  if (m.reason === 'done') { toast('Giao dịch thành công!', 'good'); sfx('coin'); } else if (m.reason) toast(m.reason, 'warn');
  G.trade = null; tradeDraft = null;
  if (G.modal === 'trade') { const keep = G.trade; closeSheet(); G.trade = keep; }
}

// ---------- Hợp đồng thợ xây ----------

export function contractSection() {
  const h = G.home;
  if (!h || h.tier < 1 || h.coop) return '';
  return `<h2>Thuê thợ là người chơi</h2>
    <p class="mut">Đăng hợp đồng xây chuồng theo mẫu: thóc được giữ ký quỹ, thợ (cấp ${G.catalog.contractLevel}+) vào chuồng thi công, máy chủ tự nghiệm thu rồi trả công. Quá hạn chưa xong thì tự hoàn.</p>
    <div class="row"><div class="meta"><label class="nm" for="ctPrice">Tiền công (200–5.000 thóc)</label>
      <select id="ctRoof" style="margin-top:6px;padding:8px;border-radius:10px;border:2px solid var(--line)">${Object.entries(G.catalog.roofs).map(([k, v]) => `<option value="${k}">${esc(v)}</option>`).join('')}</select></div>
      <div class="act"><input type="number" id="ctPrice" min="200" max="5000" value="400" style="width:100px;padding:8px;border-radius:10px;border:2px solid var(--line)"><button class="mini hot" id="ctPost">Đăng</button></div></div>`;
}
export function bindContractSection(refresh) {
  if (!$('ctPost')) return;
  $('ctPost').onclick = async () => { if (showResult(await act('contract_create', { id: $('ctRoof').value, i: +$('ctPrice').value }))) refresh(); };
}

export function openContracts() {
  raw({ t: 'contracts' });
  openSheet('contracts', '🔨 Hợp đồng xây dựng', '<p class="mut">Đang tải…</p>');
}
export function onContracts(m) {
  G.contracts = m.contracts;
  if (G.modal !== 'contracts') return;
  const st = { open: 'Chờ thợ', taken: 'Đang làm', done: 'Đã xong', expired: 'Quá hạn · đã hoàn', cancelled: 'Đã huỷ · đã hoàn' };
  setSheetBody(`<p class="mut" style="margin-top:0">Nghề thợ xây mở từ cấp ${G.catalog.contractLevel}. Uy tín của bạn: <b>${G.me.reputation || 0}</b> (mỗi cặp chủ–thợ chỉ tính một lần mỗi tuần).</p>
    ${m.contracts.map((k) => {
      const mineOwner = k.owner === m.me, mineWorker = k.worker === m.me;
      return `<div class="row"><div class="ico" style="background:#f1e3c6">🏠</div>
        <div class="meta"><div class="nm">Chuồng của ${esc(k.ownerName)} · ${esc(G.catalog.roofs[k.roof])}</div>
          <div class="sub">🌾 ${fmt(k.price)} · ${st[k.state]} · tiến độ ${k.progress}/${k.need}${k.workerName ? ' · thợ ' + esc(k.workerName) : ''} · hạn ${new Date(k.deadline).toLocaleString('vi-VN')}</div></div>
        <div class="act">
          ${k.state === 'open' && !mineOwner ? `<button class="mini hot" data-take="${k.id}" ${G.me.level < G.catalog.contractLevel ? 'disabled' : ''}>Nhận việc</button>` : ''}
          ${k.state === 'taken' && mineWorker ? `<button class="mini good" data-goto="${k.owner}">Tới chuồng</button><button class="mini sec" data-quit="${k.id}">Bỏ việc</button>` : ''}
          ${k.state === 'open' && mineOwner ? `<button class="mini sec" data-quit="${k.id}">Huỷ, hoàn ký quỹ</button>` : ''}
        </div></div>`;
    }).join('') || '<div class="empty">Chưa có hợp đồng nào đang mở.</div>'}`);
  document.querySelectorAll('[data-take]').forEach((b) => (b.onclick = async () => { b.disabled = true; showResult(await act('contract_take', { id: b.dataset.take })); openContracts(); }));
  document.querySelectorAll('[data-quit]').forEach((b) => (b.onclick = async () => { showResult(await act('contract_cancel', { id: b.dataset.quit })); openContracts(); }));
  document.querySelectorAll('[data-goto]').forEach((b) => (b.onclick = () => { closeSheet(); G.goVisit({ t: 'visit', owner: b.dataset.goto }, 'Chuồng cần thi công'); }));
}

// Thợ đứng cạnh nền chuồng của chủ hợp đồng thì có nút thi công.
export function workPrompt() {
  const h = G.home;
  if (!h || G.zone !== 'home' || h.coop || h.tier < 1) return null;
  const k = (G.contracts || []).find((c) => c.state === 'taken' && c.worker === G.me.id && c.owner === h.owner);
  if (!k) return null;
  const [cx, cz] = h.coopCell;
  return Math.hypot(G.self.pos.x - cx, G.self.pos.z - (cz - 0.5)) < 3 ? k : null;
}
export async function doWork(k) {
  const r = await act('contract_work', { id: k.id });
  if (showResult(r)) { sfx('dig'); raw({ t: 'contracts' }); }
}

// ---------- Hội Gà ----------

export function openGuild() {
  raw({ t: 'guild' });
  openSheet('guild', '🏰 Hội Gà', '<p class="mut">Đang tải…</p>');
}
let guildTab = 'info';
const guildChat = [];
export function onGuildChat(m) {
  guildChat.push(m);
  if (guildChat.length > 80) guildChat.shift();
  if (G.modal === 'guild' && guildTab === 'chat') renderGuild();
  else if (!m.system) toast(`🏰 ${m.name}: ${m.text}`);
}
export function onGuild(m) { G.guild = m; if (G.modal === 'guild') renderGuild(); }
export function onGuildList(m) { G.guildList = m.guilds; if (G.modal === 'guild') renderGuild(); }
export function onGuildInvite(m) { inviteBar(`🏰 ${m.from} mời bạn vào ${m.name}`, 60, 'Vào hội', () => raw({ t: 'guild_join', id: m.id })); }

function renderGuild() {
  const data = G.guild || {};
  const g = data.guild, me = G.me;
  if (!g) {
    if (!G.guildList) raw({ t: 'guild_list' });
    setSheetBody(`<p class="mut" style="margin-top:0">Hội Gà là nhóm tối đa ${G.catalog.guildMax} người: chat hội xuyên phòng, quỹ chung cho công trình và thi đua theo tuần. Không cướp nhà, không chiếm đất.</p>
      <h2>Các hội</h2>${(G.guildList || []).map((x) => `<div class="row"><div class="ico" style="background:${x.Color}33">${x.Emblem}</div>
        <div class="meta"><div class="nm">${esc(x.Name)}</div><div class="sub">${x.Members}/${G.catalog.guildMax} thành viên · ${x.JoinMode === 'open' ? 'vào tự do' : 'cần duyệt'} · điểm tuần ${x.Score}</div></div>
        <div class="act"><button class="mini hot" data-gjoin="${x.ID}">${x.JoinMode === 'open' ? 'Vào hội' : 'Xin vào'}</button></div></div>`).join('') || '<p class="mut">Chưa có hội nào.</p>'}
      <h2>Lập hội mới</h2>
      ${me.level < G.catalog.guildLevel ? `<p class="mut">Cần cấp ${G.catalog.guildLevel} và ${fmt(G.catalog.guildCost)} thóc.</p>` : `
      <div class="row"><div class="meta"><input type="text" id="gName" maxlength="14" placeholder="Tên hội"></div></div>
      <div class="chips" id="gEmb">${data.emblems.map((e, i) => `<button class="chip" data-emb="${e}" aria-pressed="${i === 0}">${e}</button>`).join('')}</div>
      <div class="swatches" style="margin-top:8px" id="gCol">${data.colors.map((c, i) => `<button class="sw" data-col="${c}" style="background:${c}" aria-pressed="${i === 0}"></button>`).join('')}</div>
      <button class="btn block" id="gCreate" ${me.thoc < G.catalog.guildCost ? 'disabled' : ''}>Lập hội · 🌾 ${fmt(G.catalog.guildCost)}</button>`}`);
    document.querySelectorAll('[data-gjoin]').forEach((b) => (b.onclick = () => raw({ t: 'guild_join', id: b.dataset.gjoin })));
    const pick = (sel) => document.querySelectorAll(sel).forEach((b) => (b.onclick = () => { document.querySelectorAll(sel).forEach((x) => x.setAttribute('aria-pressed', x === b)); }));
    pick('[data-emb]'); pick('[data-col]');
    if ($('gCreate')) $('gCreate').onclick = async () => {
      const emb = document.querySelector('[data-emb][aria-pressed=true]')?.dataset.emb, col = document.querySelector('[data-col][aria-pressed=true]')?.dataset.col;
      if (showResult(await act('guild_create', { text: $('gName').value, id: emb, mode: col }))) { G.guildList = null; openGuild(); }
    };
    return;
  }
  const myRole = g.members[data.me]?.role;
  const boss = myRole === 'owner' || myRole === 'vice';
  const roleName = { owner: 'Chủ hội', vice: 'Phó hội', member: 'Thành viên' };
  const tabs = `<div class="tabs">${[['info', 'Hội'], ['members', `Thành viên (${Object.keys(g.members).length})`], ['fund', 'Quỹ & công trình'], ['chat', 'Chat hội']].map(([k, l]) => `<button class="tab" data-gt="${k}" aria-selected="${guildTab === k}">${l}</button>`).join('')}</div>`;
  let body = '';
  if (guildTab === 'info') {
    const score = g.week ? Object.values(g.weekScore || {}).reduce((a, b) => a + b, 0) : 0;
    body = `<div class="row"><div class="ico" style="background:${g.color}33;font-size:2rem">${g.emblem}</div><div class="meta"><div class="nm" style="color:${g.color}">${esc(g.name)}</div>
      <div class="sub">${roleName[myRole]} · ${g.joinMode === 'open' ? 'vào tự do' : 'cần duyệt'} · điểm thi đua tuần ${score} (thắng PK)</div></div></div>
      ${g.transfer === data.me ? '<button class="btn block" id="gTakeOwner">Nhận chức chủ hội</button>' : ''}
      <h2>Bảng thông báo</h2>${(g.board || []).slice().reverse().map((p) => `<div class="cm"><b>${esc(p.by)}</b>: ${esc(p.text)} <span class="mut">${new Date(p.at).toLocaleString('vi-VN')}</span></div>`).join('') || '<p class="mut">Chưa có thông báo.</p>'}
      ${boss ? `<form class="chatin" id="gPostF"><input type="text" id="gPost" maxlength="200" placeholder="Đăng thông báo cho hội"><button class="mini hot">Đăng</button></form>
        <div class="tabs" style="margin-top:10px">${['approval', 'open'].map((md) => `<button class="tab" data-gmode="${md}" aria-selected="${g.joinMode === md}">${md === 'open' ? 'Vào tự do' : 'Cần duyệt'}</button>`).join('')}</div>` : ''}
      ${myRole === 'vice' ? '<button class="btn sec block" id="gClaim">Nhận quyền khi chủ hội vắng lâu</button>' : ''}
      <button class="btn block" id="gHall">🏰 Vào Nhà Hội</button>
      <button class="btn ghost block" id="gLeave">Rời hội</button>`;
  } else if (guildTab === 'members') {
    const reqs = Object.entries(g.requests || {});
    body = (boss && reqs.length ? `<h2 style="margin-top:0">Đơn xin vào</h2>${reqs.map(([id, n]) => `<div class="row"><div class="meta"><div class="nm">${esc(n)}</div></div><div class="act"><button class="mini good" data-gap="${id}">Duyệt</button><button class="mini sec" data-gde="${id}">Từ chối</button></div></div>`).join('')}` : '') +
      `<h2${boss && reqs.length ? '' : ' style="margin-top:0"'}>Thành viên</h2>` + Object.entries(g.members).sort((a, b) => ({ owner: 0, vice: 1, member: 2 }[a[1].role] - { owner: 0, vice: 1, member: 2 }[b[1].role])).map(([id, mm]) => `
      <div class="row"><div class="meta"><div class="nm">${esc(mm.name)} <span class="rtag" style="background:${mm.role === 'owner' ? 'var(--accent)' : mm.role === 'vice' ? 'var(--grain)' : '#8a8a8a'}">${roleName[mm.role]}</span></div>
        <div class="sub">Góp ${fmt(mm.contributed)} thóc · thắng tuần ${(g.weekScore || {})[id] || 0}</div></div>
        <div class="act">${myRole === 'owner' && id !== data.me ? `<button class="mini sec" data-grole="${id}" data-r="${mm.role === 'vice' ? 'member' : 'vice'}">${mm.role === 'vice' ? 'Hạ phó' : 'Lên phó'}</button><button class="mini sec" data-gtr="${id}">Chuyển chủ</button>` : ''}
          ${boss && id !== data.me && mm.role !== 'owner' && (myRole === 'owner' || mm.role === 'member') ? `<button class="mini sec" data-gkick="${id}">Mời ra</button>` : ''}</div></div>`).join('') +
      (boss ? `<form class="chatin" id="gInvF"><input type="text" id="gInv" maxlength="7" placeholder="Mã người chơi để mời" style="text-transform:uppercase"><button class="mini hot">Mời</button></form>` : '');
  } else if (guildTab === 'fund') {
    body = `<div class="kvs"><b>Quỹ hội</b><span>🌾 ${fmt(g.fund)}</span></div>
      <div class="coinbtns">${[50, 100, 500].map((n) => `<button class="mini hot" data-ggive="${n}" ${me.thoc < n ? 'disabled' : ''}>Góp ${n}</button>`).join('')}</div>
      <p class="mut">Quỹ chỉ chi cho công trình đã định nghĩa; không ai rút được về ví cá nhân.</p>
      ${data.projects.map((p) => `<div class="row"><div class="ico" style="background:#fff0cf">${g.projects?.[p.id] ? '✅' : '🏗️'}</div><div class="meta"><div class="nm">${esc(p.name)}</div><div class="sub">${esc(p.perk)} · 🌾 ${fmt(p.cost)}</div></div>
        <div class="act">${g.projects?.[p.id] ? '' : boss ? `<button class="mini hot" data-gproj="${p.id}" ${g.fund < p.cost ? 'disabled' : ''}>Xây</button>` : ''}</div></div>`).join('')}
      <h2>Sổ quỹ</h2>${(g.ledger || []).slice(-15).reverse().map((l) => `<div class="cm">${new Date(l.at).toLocaleDateString('vi-VN')} · ${esc(l.who)} · ${esc(l.reason)} ${l.delta ? `<b style="color:${l.delta < 0 ? 'var(--bad)' : 'var(--good)'}">${l.delta > 0 ? '+' : ''}${l.delta}</b>` : ''}</div>`).join('')}`;
  } else {
    body = `<div id="chatlog">${guildChat.map((c) => c.system ? `<div class="cm sys">${esc(c.text)}</div>` : `<div class="cm"><span class="au" style="color:${g.color}">${esc(c.name)}:</span> ${esc(c.text)}</div>`).join('') || '<p class="mut">Chat hội tới mọi thành viên đang online, ở bất kỳ chuồng nào.</p>'}</div>
      <form class="chatin" id="gChatF"><input type="text" id="gChat" maxlength="120" placeholder="Nhắn cả hội…" autocomplete="off"><button class="mini hot">Gửi</button></form>`;
  }
  setSheetBody(tabs + body);
  document.querySelectorAll('[data-gt]').forEach((b) => (b.onclick = () => { guildTab = b.dataset.gt; renderGuild(); }));
  const on = (sel, f) => document.querySelectorAll(sel).forEach((b) => (b.onclick = (e) => { e.preventDefault(); f(b); }));
  on('[data-gap]', (b) => raw({ t: 'guild_approve', id: b.dataset.gap }));
  on('[data-gde]', (b) => raw({ t: 'guild_deny', id: b.dataset.gde }));
  on('[data-grole]', (b) => raw({ t: 'guild_role', id: b.dataset.grole, mode: b.dataset.r }));
  on('[data-gtr]', (b) => raw({ t: 'guild_transfer', id: b.dataset.gtr }));
  on('[data-gkick]', (b) => raw({ t: 'guild_kick', id: b.dataset.gkick }));
  on('[data-gmode]', (b) => raw({ t: 'guild_mode', mode: b.dataset.gmode }));
  on('[data-gproj]', (b) => raw({ t: 'guild_project', id: b.dataset.gproj }));
  on('[data-ggive]', async (b) => { b.disabled = true; if (showResult(await act('guild_give', { i: +b.dataset.ggive }))) raw({ t: 'guild' }); });
  if ($('gTakeOwner')) $('gTakeOwner').onclick = () => raw({ t: 'guild_transfer_accept' });
  if ($('gClaim')) $('gClaim').onclick = () => raw({ t: 'guild_claim' });
  if ($('gLeave')) $('gLeave').onclick = () => raw({ t: 'guild_leave' });
  if ($('gHall')) $('gHall').onclick = () => { closeSheet(); G.goVisit({ t: 'guild_hall' }, 'Nhà Hội Gà'); };
  if ($('gPostF')) $('gPostF').onsubmit = (e) => { e.preventDefault(); raw({ t: 'guild_post', text: $('gPost').value }); };
  if ($('gInvF')) $('gInvF').onsubmit = (e) => { e.preventDefault(); raw({ t: 'guild_invite', id: $('gInv').value.trim() }); $('gInv').value = ''; };
  if ($('gChatF')) { $('gChatF').onsubmit = (e) => { e.preventDefault(); const t = $('gChat').value.trim(); if (t) raw({ t: 'guild_chat', text: t }); $('gChat').value = ''; }; const log = $('chatlog'); log.scrollTop = log.scrollHeight; }
}

// ---------- Thanh lời mời dùng chung ----------

let barTimer = null;
function inviteBar(text, ttl, yesLabel, onYes) {
  const bar = $('inviteBar');
  let left = Math.round(ttl);
  bar.hidden = false;
  bar.dataset.id = '';
  bar.innerHTML = `<span class="t">${esc(text)} (<span id="ib2">${left}</span>s)</span><button class="mini good" id="ib2Yes">${yesLabel}</button><button class="mini sec" id="ib2No">Bỏ qua</button>`;
  const hide = () => { clearInterval(barTimer); bar.hidden = true; };
  $('ib2Yes').onclick = () => { hide(); onYes(); };
  $('ib2No').onclick = hide;
  clearInterval(barTimer);
  barTimer = setInterval(() => { left--; if (left <= 0) hide(); else if ($('ib2')) $('ib2').textContent = left; }, 1000);
  sfx('notify');
}

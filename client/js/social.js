// Bạn bè, tin riêng, mời chuồng/nhà (mục 11) và nhà riêng (mục 9.1). Server quyết định mọi quyền.
import { G, $, esc, openSheet, setSheetBody, closeSheet, toast, showResult, homeLeft, slowGuard } from './ui.js';
import { ghostDecor } from './world.js';
import * as V from './v11.js';
import * as L from './life.js';

const raw = (m) => G.net.raw(m);
const act = (a, args) => G.net.act(a, args);
const fmt = (n) => Number(n).toLocaleString('vi-VN');
const decorById = (id) => G.catalog.decors.find((d) => d.id === id);

// ---------- Bạn bè ----------

let friendsTab = 'friends';
export function openFriends(tab) {
  if (tab) friendsTab = tab;
  raw({ t: 'social' });
  openSheet('friends', 'Bạn bè', '<p class="mut" data-loading>Đang tải…</p>');
  slowGuard('friends', () => raw({ t: 'social' }));
}

export function onSocial(view) {
  G.social = view;
  const unread = view.friends.reduce((s, f) => s + f.unread, 0) + view.reqIn.length;
  $('friendsDot').hidden = !unread;
  $('friendsDot').textContent = unread > 9 ? '9+' : unread || '';
  if (G.modal === 'friends') renderFriends();
}

function renderFriends() {
  const v = G.social;
  const tabs = [['friends', `Bạn bè (${v.friends.length})`], ['requests', `Lời mời${v.reqIn.length ? ` (${v.reqIn.length})` : ''}`], ['near', 'Quanh đây'], ['settings', 'Cài đặt']];
  let body = '';
  if (friendsTab === 'friends') {
    const list = v.friends.slice().sort((a, b) => b.online - a.online || a.name.localeCompare(b.name));
    body = `<form class="chatin" id="addForm" style="margin:0 0 12px"><input type="text" id="addCode" maxlength="7" placeholder="Nhập mã bạn bè, vd: K7QM2D" autocomplete="off" style="text-transform:uppercase"><button class="mini hot" type="submit">Kết bạn</button></form>` +
      (list.map((f) => `<div class="row"><div class="ico" style="background:${f.online ? '#dff0d4' : '#eee3cf'}">${f.online ? '🐥' : '💤'}</div>
        <div class="meta"><div class="nm">${esc(f.name)} <span class="mut" title="Mã bạn bè">Mã bạn bè: #${esc(f.code)}</span>${f.unread ? ` <span class="rtag" style="background:var(--bad)">${f.unread} tin</span>` : ''}</div>
          <div class="sub">${f.online ? (f.where ? 'Đang ở ' + esc(f.where) : 'Đang online') : 'Offline'}</div></div>
        <div class="act"><button class="mini hot" data-dm="${f.id}" data-name="${esc(f.name)}">Nhắn</button>
          <button class="mini sec" data-more="${f.id}" data-name="${esc(f.name)}" data-online="${f.online ? 1 : 0}">⋯</button></div></div>`).join('') ||
        '<div class="empty">Chưa có bạn nào. Gửi mã của bạn cho người quen, hoặc kết bạn với người chơi quanh đây.</div>');
  } else if (friendsTab === 'requests') {
    body = (v.reqIn.map((f) => `<div class="row"><div class="ico" style="background:#fff0cf">👋</div><div class="meta"><div class="nm">${esc(f.name)} <span class="mut" title="Mã bạn bè">Mã bạn bè: #${esc(f.code)}</span></div><div class="sub">Muốn kết bạn với bạn</div></div>
      <div class="act"><button class="mini good" data-resp="${f.id}" data-on="1">Đồng ý</button><button class="mini sec" data-resp="${f.id}" data-on="0">Từ chối</button></div></div>`).join('') || '<div class="empty">Không có lời mời mới.</div>') +
      (v.reqOut.length ? `<h2>Đã gửi</h2>` + v.reqOut.map((f) => `<div class="row"><div class="ico" style="background:#eee3cf">⏳</div><div class="meta"><div class="nm">${esc(f.name)}</div><div class="sub">Đang chờ đồng ý</div></div></div>`).join('') : '');
  } else if (friendsTab === 'near') {
    const near = [...G.players.entries()];
    body = `<p class="mut" style="margin:0 0 10px">Những chú gà đang cùng khu với bạn.</p>` + (near.map(([pid, p]) => {
      const isFriend = v.friends.some((f) => f.name === p.name && f.code);
      return `<div class="row"><div class="ico" style="background:#f1e3c6">🐔</div><div class="meta"><div class="nm">${esc(p.name)}</div><div class="sub">Cấp ${p.level}</div></div>
        <div class="act">${isFriend ? '' : `<button class="mini good" data-addpid="${pid}">Kết bạn</button>`}
          <button class="mini sec" data-tradepid="${pid}">Giao dịch</button>
          ${isFriend && (G.me.items?.bo_hoa || 0) > 0 ? `<button class="mini good" data-flowerpid="${pid}">💐 Tặng hoa</button>` : ''}
          ${G.myRide ? (p.mesh?.userData.seated === G.pid ? `<button class="mini sec" data-kickpid="${pid}">Mời xuống xe</button>` : `<button class="mini sec" data-seatpid="${pid}">Mời lên xe</button>`) : ''}
          ${G.myGuild ? `<button class="mini sec" data-guildpid="${pid}">Mời vào hội</button>` : ''}
          <button class="mini sec" data-report="${pid}" data-name="${esc(p.name)}">Báo cáo</button>
          <button class="mini sec" data-blockpid="${pid}" data-name="${esc(p.name)}">Chặn</button></div></div>`;
    }).join('') || '<div class="empty">Chưa có ai khác ở khu này.</div>');
  } else {
    body = `<div class="row"><div class="ico" style="background:#f0e3ff">🆔</div><div class="meta"><div class="nm">Mã của bạn: <span style="font-size:1.2rem;letter-spacing:.08em">${esc(v.code)}</span></div><div class="sub">Gửi mã này để bạn bè tìm bạn, kể cả khi trùng tên.</div></div>
        <div class="act"><button class="mini sec" id="copyCode">Chép</button></div></div>
      <label class="row" style="cursor:pointer"><input type="checkbox" id="pvPresence" ${v.privacy.showPresence ? 'checked' : ''}><span class="meta"><span class="nm">Cho bạn bè thấy tôi đang ở chuồng nào</span></span></label>
      <label class="row" style="cursor:pointer"><input type="checkbox" id="pvPK" ${v.privacy.allowPK ? 'checked' : ''}><span class="meta"><span class="nm">Nhận lời mời PK</span></span></label>
      <label class="row" style="cursor:pointer"><input type="checkbox" id="pvQuiet" ${v.privacy.quietMail ? 'checked' : ''}><span class="meta"><span class="nm">Giảm thư Chim Chíp</span><span class="sub" style="display:block">Chỉ nhận thư công trình và hệ thống, bỏ thư khách ghé thăm, lời mời kết bạn</span></span></label>
      <h2>Đã chặn</h2>${v.blocks.map((f) => `<div class="row"><div class="ico" style="background:#f6d6cf">🚫</div><div class="meta"><div class="nm">${esc(f.name)}</div></div><div class="act"><button class="mini sec" data-unblock="${f.id}">Bỏ chặn</button></div></div>`).join('') || '<p class="mut">Chưa chặn ai.</p>'}
      <p class="mut">Chặn một người sẽ chặn tin nhắn, lời mời kết bạn, PK, lời mời về nhà và chat của họ với bạn.</p>`;
  }
  setSheetBody(`<div class="tabs">${tabs.map(([k, l]) => `<button class="tab" data-ft="${k}" aria-selected="${friendsTab === k}">${l}</button>`).join('')}</div>${body}`);
  bindFriends();
}

function bindFriends() {
  document.querySelectorAll('[data-ft]').forEach((b) => (b.onclick = () => { friendsTab = b.dataset.ft; renderFriends(); }));
  const add = $('addForm');
  if (add) add.onsubmit = (e) => { e.preventDefault(); const c = $('addCode').value.trim(); if (c) raw({ t: 'friend_add', id: c }); $('addCode').value = ''; };
  document.querySelectorAll('[data-dm]').forEach((b) => (b.onclick = () => openDM(b.dataset.dm, b.dataset.name)));
  document.querySelectorAll('[data-more]').forEach((b) => (b.onclick = () => friendMenu(b.dataset.more, b.dataset.name, b.dataset.online === '1')));
  document.querySelectorAll('[data-resp]').forEach((b) => (b.onclick = () => raw({ t: 'friend_respond', id: b.dataset.resp, on: b.dataset.on === '1' })));
  document.querySelectorAll('[data-addpid]').forEach((b) => (b.onclick = () => { b.disabled = true; raw({ t: 'friend_add', target: b.dataset.addpid }); }));
  document.querySelectorAll('[data-flowerpid]').forEach((b) => (b.onclick = async () => { b.disabled = true; if (showResult(await act('gift_flower', { target: b.dataset.flowerpid })) && G.modal === 'friends') renderFriends(); }));
  document.querySelectorAll('[data-tradepid]').forEach((b) => (b.onclick = async () => { closeSheet(); showResult(await act('trade_invite', { target: b.dataset.tradepid })); }));
  document.querySelectorAll('[data-seatpid]').forEach((b) => (b.onclick = async () => { showResult(await act('seat_invite', { target: b.dataset.seatpid })); }));
  document.querySelectorAll('[data-kickpid]').forEach((b) => (b.onclick = async () => { if (showResult(await act('seat_kick', { target: b.dataset.kickpid }), { quiet: true }) && G.modal === 'friends') renderFriends(); }));
  document.querySelectorAll('[data-guildpid]').forEach((b) => (b.onclick = () => raw({ t: 'guild_invite', target: b.dataset.guildpid })));
  document.querySelectorAll('[data-blockpid]').forEach((b) => (b.onclick = () => confirmBox(`Chặn ${b.dataset.name}?`, 'Bạn sẽ không nhận tin, lời mời, PK hay chat của người này.', () => raw({ t: 'block', target: b.dataset.blockpid, on: true }))));
  document.querySelectorAll('[data-report]').forEach((b) => (b.onclick = () => reportMenu(b.dataset.report, b.dataset.name)));
  document.querySelectorAll('[data-unblock]').forEach((b) => (b.onclick = () => raw({ t: 'block', id: b.dataset.unblock, on: false })));
  if ($('copyCode')) $('copyCode').onclick = () => { navigator.clipboard?.writeText(G.social.code).then(() => toast('Đã chép mã ' + G.social.code, 'good'), () => toast('Mã của bạn: ' + G.social.code)); };
  const pv = () => raw({ t: 'privacy', presence: $('pvPresence').checked, allowPK: $('pvPK').checked, on: $('pvQuiet').checked });
  if ($('pvPresence')) { $('pvPresence').onchange = pv; $('pvPK').onchange = pv; $('pvQuiet').onchange = pv; }
}

function friendMenu(id, name, online) {
  const choices = [
    ['💬 Nhắn tin', () => openDM(id, name)],
    ...(online && G.room?.kind === 'public' ? [['🏠 Mời vào chuồng ' + esc(G.room.name), () => raw({ t: 'invite_room', id })]] : []),
    ...(online ? [['🏡 Mời về chuồng riêng của tôi', () => raw({ t: 'home_invite', id })]] : []),
    ['🚪 Thăm chuồng riêng của ' + esc(name), () => goVisit({ t: 'visit', owner: id }, 'Chuồng của ' + name)],
    ['➖ Huỷ kết bạn', () => confirmBox(`Huỷ kết bạn với ${name}?`, '', () => raw({ t: 'friend_remove', id }))],
    ['🚫 Chặn', () => confirmBox(`Chặn ${name}?`, 'Huỷ kết bạn và chặn mọi tương tác.', () => raw({ t: 'block', id, on: true }))],
  ];
  openSheet('friendmenu', esc(name), `<div class="choices">${choices.map(([t], i) => `<button class="choice" data-i="${i}">${t}</button>`).join('')}</div>`);
  document.querySelectorAll('.choice').forEach((b) => (b.onclick = () => { closeSheet(); choices[+b.dataset.i][1](); }));
}

// Hộp xác nhận ngay trong trang (viewer không hiện confirm()).
export function confirmBox(title, text, yes) {
  openSheet('confirm', esc(title), `${text ? `<p>${esc(text)}</p>` : ''}<div class="choices"><button class="btn" id="cYes">Đồng ý</button><button class="btn sec" id="cNo">Thôi</button></div>`);
  $('cYes').onclick = () => { closeSheet(); yes(); };
  $('cNo').onclick = closeSheet;
}

const REASONS = [['spam', 'Spam / làm phiền'], ['ngon_tu', 'Ngôn từ xấu'], ['ten', 'Tên không phù hợp'], ['quay_roi', 'Quấy rối'], ['gian_lan', 'Gian lận']];
function reportMenu(pid, name) {
  openSheet('report', `Báo cáo ${esc(name)}`, `<p class="mut" style="margin-top:0">Máy chủ đính kèm các dòng chat gần nhất của người này làm bằng chứng.</p>
    <div class="choices">${REASONS.map(([k, l]) => `<button class="choice" data-r="${k}">${l}</button>`).join('')}</div>`);
  document.querySelectorAll('[data-r]').forEach((b) => (b.onclick = () => { closeSheet(); raw({ t: 'report', target: pid, reason: b.dataset.r }); }));
}

// ---------- Thư Chim Chíp ----------

export function openMail() {
  openSheet('mail', '📮 Thư Chim Chíp', '<p class="mut" data-loading>Đang tải…</p>');
  raw({ t: 'mail' });
  slowGuard('mail', () => raw({ t: 'mail' }));
}
const MAIL_ICON = { visit: '🚪', build: '🔨', friend: '👥', hatch: '🐣', system: '📢' };
export function onMail(m) {
  if (G.modal !== 'mail') return;
  setSheetBody(((m.mail || []).slice().reverse().map((x) => `<div class="row" style="${x.read ? 'opacity:.75' : ''}"><div class="ico" style="background:#fff0cf">${MAIL_ICON[x.kind] || '✉️'}</div>
    <div class="meta"><div class="nm">${esc(x.text)}</div><div class="sub">${new Date(x.at).toLocaleString('vi-VN')}</div></div></div>`).join('') ||
    '<div class="empty">Chưa có thư. Chim Chíp sẽ báo khi bạn bè ghé chuồng, công trình xong hay có lời mời kết bạn.</div>') +
    '<p class="mut">Muốn ít thư hơn? Bật “Giảm thư Chim Chíp” trong Bạn bè → Cài đặt.</p>');
}

// ---------- Tin riêng ----------

export function openDM(id, name) {
  G.dmWith = { id, name };
  openSheet('dm', `💬 ${esc(name)}`, `<div id="chatlog"><p class="mut" data-loading>Đang tải…</p></div>
    <form class="chatin" id="dmForm"><input type="text" id="dmMsg" maxlength="200" placeholder="Nhắn riêng…" autocomplete="off"><button class="mini hot" type="submit">Gửi</button></form>`,
    { close: () => { G.dmWith = null; } });
  raw({ t: 'dm_history', id });
  slowGuard('dm', () => raw({ t: 'dm_history', id }));
  $('dmForm').onsubmit = (e) => { e.preventDefault(); const t = $('dmMsg').value.trim(); if (t) raw({ t: 'dm', id, text: t }); $('dmMsg').value = ''; };
  setTimeout(() => $('dmMsg')?.focus(), 50);
}
const dmLine = (m) => `<div class="cm"><span class="au" style="color:${m.from === G.me.id ? 'var(--accent)' : 'var(--wood)'}">${m.from === G.me.id ? 'Bạn' : esc(G.dmWith?.name || '')}:</span> ${esc(m.text)} <span class="mut">${new Date(m.at).toLocaleTimeString('vi-VN', { hour: '2-digit', minute: '2-digit' })}</span></div>`;
export function onDMHistory(m) {
  if (G.modal !== 'dm' || G.dmWith?.id !== m.with) return;
  const log = $('chatlog');
  log.innerHTML = m.msgs.map(dmLine).join('') || '<p class="mut">Chưa có tin nào. Chào một câu đi!</p>';
  log.scrollTop = log.scrollHeight;
  raw({ t: 'social' }); // cập nhật số tin chưa đọc
}
export function onDM(m) {
  if (G.modal === 'dm' && G.dmWith?.id === m.with) {
    const log = $('chatlog');
    if (log.querySelector('p.mut')) log.innerHTML = '';
    log.insertAdjacentHTML('beforeend', dmLine(m.msg));
    log.scrollTop = log.scrollHeight;
    return;
  }
  if (m.msg.from !== G.me.id) {
    toast(`💬 ${m.name}: ${m.msg.text}`);
    raw({ t: 'social' });
  }
}

// ---------- Lời mời chuồng / nhà ----------

let barTimer = null;
function inviteBar(text, ttl, yesLabel, onYes) {
  const bar = $('inviteBar');
  const end = performance.now() + ttl * 1000; // hạn tuyệt đối: tab nền bị giãn timer vẫn hết hạn đúng lúc
  let left = Math.round(ttl);
  bar.hidden = false;
  bar.dataset.id = '';
  bar.innerHTML = `<span class="t">${esc(text)}${ttl ? ` (<span id="ibLeft">${left}</span>s)` : ''}</span><button class="mini good" id="ibYes">${yesLabel}</button><button class="mini sec" id="ibNo">Bỏ qua</button>`;
  const hide = () => { clearInterval(barTimer); bar.hidden = true; };
  $('ibYes').onclick = () => { const ok = !ttl || performance.now() < end; hide(); if (ok) onYes(); else toast('Lời mời đã hết hạn', 'warn'); };
  $('ibNo').onclick = hide;
  clearInterval(barTimer);
  barTimer = setInterval(() => { left = Math.ceil((end - performance.now()) / 1000); if (left <= 0) hide(); else if ($('ibLeft')) $('ibLeft').textContent = left; }, 1000);
}
export function onRoomInvite(m) { inviteBar(`🏠 ${m.from} mời bạn vào ${m.roomName}`, 60, 'Đến', () => raw({ t: 'switch_room', room: m.room })); }
export function onHomeInvite(m) { inviteBar(`🏡 ${m.from} mời bạn về chuồng riêng`, m.ttl, 'Đến chơi', () => goVisit({ t: 'visit', owner: m.owner, token: m.token }, `Chuồng của ${m.from}`)); }

// Coop gates and the guild hall open from Xóm Rơm Vàng. Elsewhere a visit becomes a route there,
// and the visit is offered again on arrival (the server re-checks access at the gate).
export function goVisit(msg, label) {
  if (G.zone === 'neighborhood' || G.room?.kind !== 'public') { raw(msg); return; }
  G.pendingVisit = { msg, label, until: Date.now() + 110000 };
  G.navigate?.('neighborhood', `${label} nằm ở Xóm Rơm Vàng. Đi theo chỉ đường để tới cổng.`);
}
export function arrivedLane() {
  const v = G.pendingVisit;
  if (!v || G.zone !== 'neighborhood') return;
  G.pendingVisit = null;
  if (Date.now() < v.until) inviteBar(`${v.label} · bạn đã tới Xóm Rơm Vàng`, 60, 'Vào ngay', () => raw(v.msg));
}
G.goVisit = goVisit;

// ---------- Bảng chỉ đường về chuồng riêng (sảnh) ----------

export function dialogHomeSign() {
  openSheet('homes', 'Chuồng riêng', `<div class="choices"><button class="btn" id="goMine">🏡 Về chuồng của tôi</button></div>
    <h2>Thăm chuồng khác</h2><div id="homeList"><p class="mut" data-loading>Đang tải…</p></div>`);
  $('goMine').onclick = () => { closeSheet(); goVisit({ t: 'visit', owner: 'me' }, 'Chuồng của bạn'); };
  raw({ t: 'homes' });
  slowGuard('homes', () => raw({ t: 'homes' }));
}
const ACCESS_NAME = { private: 'Riêng tư', friends: 'Bạn bè', public: 'Công khai' };
export function onHomes(m) {
  if (G.modal !== 'homes') return;
  $('homeList').innerHTML = m.homes.map((h) => `<div class="row"><div class="ico" style="background:#f1e3c6">🏡</div>
    <div class="meta"><div class="nm">Chuồng của ${esc(h.name)}</div><div class="sub">${esc(h.why)} · ${ACCESS_NAME[h.access] || ''}</div></div>
    <div class="act"><button class="mini hot" data-visit="${h.id}">Thăm</button></div></div>`).join('') ||
    '<p class="mut">Chưa có chuồng nào mở cho bạn. Kết bạn để thăm chuồng của nhau, hoặc chờ ai đó mở cửa công khai.</p>';
  document.querySelectorAll('[data-visit]').forEach((b) => (b.onclick = () => { closeSheet(); goVisit({ t: 'visit', owner: b.dataset.visit }, 'Chuồng bạn chọn'); }));
}

// ---------- Trong chuồng riêng ----------

const isOwner = () => G.home && G.me && G.home.owner === G.me.id;

export function dialogCow() {
  const h = G.home;
  if (!h) { L.dialogMaterials(); return; } // Bò Mộc ở Xóm Rơm Vàng: tiệm vật liệu
  if (!isOwner()) {
    openSheet('npc', 'Bò Mộc', `<div class="npcbox"><div class="npcface" style="background:#dfe9f5">🐮</div><div><div class="who">Bò Mộc</div><p>Chuồng của ${esc(h.ownerName)} đó, đẹp không? Muốn có chuồng thế này thì về nhà mình gọi tôi nhé.</p></div></div>`);
    return;
  }
  const t = h.tierInfo, me = G.me;
  const building = h.coop && h.coop.state === 'building';
  const shop = G.catalog.decors.map((d) => `<button class="cos" data-buyd="${d.id}" ${me.thoc < d.cost ? 'disabled' : ''}><span class="ci">${d.emo}</span><span class="cn">${esc(d.name)}</span><span class="mut"><b>🌾 ${d.cost}</b>${h.storage[d.id] ? ` · kho ${h.storage[d.id]}` : ''}</span></button>`).join('');
  const mats = G.catalog.items.filter((it) => it.group === 'vatlieu').map((it) => `${it.emo}${me.items?.[it.id] || 0}`).join(' ');
  openSheet('cow', 'Bò Mộc · xây dựng & trang trí', `
    <p class="mut" style="margin-top:0">Đất: <b>${esc(t.name)}</b> (${t.halfX * 2} × ${t.halfZ * 2} m) · ví 🌾 ${fmt(me.thoc)} · cấp ${me.level}</p>
    <p class="mut">Vật liệu trong kho: ${mats} <button class="mini sec" id="matShop">Mua vật liệu</button></p>
    <h2>Nhà</h2>${L.houseSection(h)}
    <div class="note">Muốn lô rộng hơn hoặc đổi vị trí (ven ao, đồi thông)? Gặp Ông Chủ Đầu Tư ở Văn phòng đất nền cạnh cổng Xóm Rơm Vàng.</div>
    ${V.contractSection()}<h2>Đồ trang trí <span class="mut" style="font-weight:600">· mua vào kho rồi tự đặt</span></h2><div class="grid">${shop}</div>`);
  V.bindContractSection(dialogCow);
  L.bindHouseSection(dialogCow);
  $('matShop').onclick = L.dialogMaterials;
  document.querySelectorAll('[data-buyd]').forEach((b) => (b.onclick = async () => { b.disabled = true; if (showResult(await act('home_buy', { id: b.dataset.buyd }))) dialogCow(); }));
  if (building) {
    const end = performance.now() + homeLeft(h, 'buildLeft') * 1000;
    (function tick() { const el = $('buildLeft'); if (!el) return; el.textContent = Math.max(0, Math.ceil((end - performance.now()) / 1000)); requestAnimationFrame(tick); })();
  }
}

export function openHomeBoard() {
  const h = G.home;
  const guests = [...G.players.entries()];
  if (!isOwner()) {
    openSheet('homeboard', `Chuồng của ${esc(h.ownerName)}`, `<p>Đất: <b>${esc(h.tierInfo.name)}</b> · ${h.objects.length} món trang trí${h.coop?.state === 'done' ? ` · có ${esc(G.catalog.houseDesigns.find((d) => d.id === h.coop.design)?.name || 'chuồng')} ${G.catalog.materialNames[h.coop.material] || ''}` : ''}.</p>
      <p class="mut">Chủ chuồng có thể mời khách ra về. Muốn ra ngoài thì đi tới cổng chuồng để ra Xóm Rơm Vàng.</p>`);
    return;
  }
  openSheet('homeboard', 'Bảng nhà', `
    <h2 style="margin-top:0">Ai được vào chuồng?</h2>
    <div class="tabs">${['private', 'friends', 'public'].map((a) => `<button class="tab" data-acc="${a}" aria-selected="${h.access === a}">${ACCESS_NAME[a]}</button>`).join('')}</div>
    <p class="mut">Riêng tư: chỉ khách có lời mời. Bạn bè: bạn bè vào tự do. Công khai: hiện trong danh sách “chuồng mở cửa”. Tối đa ${h.cap || G.catalog.homeCap} người (rơm 8 · gỗ 12 · gạch 16).</p>
    <h2>Bữa tiệc sân nhà</h2>
    ${homeLeft(h, 'partyLeft') > 0 ? `<p>🎉 Tiệc đang mở — còn khoảng ${Math.ceil(homeLeft(h, 'partyLeft') / 60)} phút. Ai cũng ghé được (trừ người bị chặn), tối đa ${h.cap || G.catalog.homeCap} người.</p>`
      : `<p class="mut">Mở tiệc để chuồng hiện trong danh sách “đang mở tiệc”; khách ăn mừng ở Bàn tiệc và chụp ảnh. Cần đặt ít nhất một Bàn tiệc.</p>
      <div class="coinbtns"><button class="mini hot" data-party="30">Mở tiệc 30 phút</button><button class="mini hot" data-party="60">Mở tiệc 60 phút</button></div>`}
    <h2>Trang trí</h2>
    <div class="row"><div class="ico" style="background:#f0e3ff">🪴</div><div class="meta"><div class="nm">${h.objects.length}/${G.catalog.homeObjLimit} món đã đặt</div><div class="sub">Kho: ${Object.entries(h.storage).filter(([, n]) => n > 0).map(([k, n]) => `${decorById(k)?.emo || ''}×${n}`).join(' ') || 'trống'}</div></div>
      <div class="act"><button class="mini hot" id="decoBtn">Sắp xếp đồ</button></div></div>
    <h2>Khách đang ở đây</h2>
    ${guests.map(([pid, p]) => `<div class="row"><div class="ico" style="background:#f1e3c6">🐔</div><div class="meta"><div class="nm">${esc(p.name)}</div></div><div class="act"><button class="mini sec" data-kick="${pid}">Mời ra về</button></div></div>`).join('') || '<p class="mut">Chưa có khách. Mời bạn bè từ bảng Bạn bè.</p>'}`);
  document.querySelectorAll('[data-acc]').forEach((b) => (b.onclick = async () => { if (showResult(await act('home_access', { id: b.dataset.acc }), { quiet: true })) toast('Quyền vào: ' + ACCESS_NAME[b.dataset.acc], 'good'); openHomeBoard(); }));
  document.querySelectorAll('[data-kick]').forEach((b) => (b.onclick = async () => { b.disabled = true; showResult(await act('home_kick', { target: b.dataset.kick })); }));
  $('decoBtn').onclick = openDecorate;
  document.querySelectorAll('[data-party]').forEach((b) => (b.onclick = async () => { if (showResult(await act('home_party', { i: +b.dataset.party }))) openHomeBoard(); }));
}

// ---------- Chế độ sắp xếp đồ ----------

export function openDecorate() {
  const h = G.home;
  const stock = Object.entries(h.storage).filter(([, n]) => n > 0);
  const placed = h.objects.map((o) => `<div class="row"><div class="ico" style="background:#f1e3c6">${decorById(o.kind)?.emo || '❔'}</div><div class="meta"><div class="nm">${esc(decorById(o.kind)?.name || o.kind)}</div><div class="sub">ô (${o.x}, ${o.z}) · xoay ${o.rot * 90}°</div></div>
    <div class="act"><button class="mini sec" data-rot="${o.id}">Xoay</button><button class="mini sec" data-pick="${o.id}">Cất</button></div></div>`).join('');
  openSheet('decorate', 'Sắp xếp đồ', `
    <p class="mut" style="margin-top:0">Chọn món trong kho rồi đi tới chỗ muốn đặt: bóng mờ hiện ngay trước mặt gà. Không được chắn lối vào hay đặt lên nền chuồng.</p>
    <h2>Trong kho</h2><div class="grid">${stock.map(([k, n]) => `<button class="cos" data-place="${k}"><span class="ci">${decorById(k)?.emo}</span><span class="cn">${esc(decorById(k)?.name)}</span><span class="mut">×${n}</span></button>`).join('') || '<span class="mut">Kho trống — mua thêm ở Bò Mộc.</span>'}</div>
    <h2>Đã đặt</h2>${placed || '<p class="mut">Chưa đặt món nào.</p>'}`);
  document.querySelectorAll('[data-place]').forEach((b) => (b.onclick = () => { closeSheet(); startPlacing(b.dataset.place); }));
  document.querySelectorAll('[data-pick]').forEach((b) => (b.onclick = async () => { b.disabled = true; if (showResult(await act('home_pickup', { id: b.dataset.pick }))) openDecorate(); }));
  document.querySelectorAll('[data-rot]').forEach((b) => (b.onclick = async () => {
    const o = G.home.objects.find((x) => x.id === b.dataset.rot);
    b.disabled = true;
    if (showResult(await act('home_move', { id: o.id, x: o.x, z: o.z, i: o.rot + 1 }), { quiet: true })) openDecorate();
  }));
}

function startPlacing(kind) {
  stopPlacing();
  const ghost = ghostDecor(kind);
  G.world.scene.add(ghost);
  G.placing = { kind, rot: 0, ghost, cell: [0, 0] };
  const bar = $('decorBar');
  bar.hidden = false;
  bar.innerHTML = `<span class="t">${decorById(kind).emo} Đặt ${esc(decorById(kind).name)} — đi tới chỗ muốn đặt</span>
    <button class="mini sec" id="dbRot">⟳ Xoay (R)</button><button class="mini hot" id="dbPut">Đặt ở đây (E)</button><button class="mini sec" id="dbStop">Xong</button>`;
  $('dbRot').onclick = rotatePlacing;
  $('dbPut').onclick = commitPlacing;
  $('dbStop').onclick = stopPlacing;
}
export function rotatePlacing() { if (G.placing) G.placing.rot = (G.placing.rot + 1) % 4; }
export async function commitPlacing() {
  const pl = G.placing;
  if (!pl) return;
  const [x, z] = pl.cell;
  const r = await act('home_place', { id: pl.kind, x, z, i: pl.rot });
  if (!showResult(r, { quiet: true })) return;
  toast('Đã đặt ' + decorById(pl.kind).name, 'good');
  if (!(G.home.storage[pl.kind] > 0)) stopPlacing();
}
export function stopPlacing() {
  if (!G.placing) return;
  G.world.scene.remove(G.placing.ghost);
  G.placing = null;
  $('decorBar').hidden = true;
}
// Gọi mỗi khung hình: đặt bóng mờ vào ô ngay trước mặt gà.
export function updatePlacing() {
  const pl = G.placing;
  if (!pl) return;
  const S = G.self, ry = S.mesh.rotation.y;
  const x = Math.round(S.pos.x + Math.sin(ry) * 1.3), z = Math.round(S.pos.z + Math.cos(ry) * 1.3);
  pl.cell = [x, z];
  pl.ghost.position.set(x, 0, z);
  pl.ghost.rotation.y = -pl.rot * Math.PI / 2;
}

// Thành phần giao diện dùng chung và trạng thái client G (chỉ là bản hiển thị của dữ liệu máy chủ).

export const G = {
  net: null, world: null, catalog: null, me: null, pid: null,
  zone: null, zones: null, room: null,
  self: null,            // { mesh, pos, ry, srv }
  players: new Map(),    // pid → { mesh, name, tag, char, equipped, target, ry, mv, stage }
  chat: [], chatUnread: 0,
  modal: null,           // tên bảng đang mở
  busy: false,           // đang chờ server cho thao tác chặn di chuyển (đào)
  match: null,
};

export const $ = (id) => document.getElementById(id);
export const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));

const modalRoot = $('modalRoot'), sheet = $('sheet');
let onClose = null;
$('scrim').onclick = () => closeSheet();
window.addEventListener('keydown', (e) => { if (e.key === 'Escape' && G.modal) closeSheet(); });

export function openSheet(name, title, html, { closable = true, close } = {}) {
  if (onClose) { const f = onClose; onClose = null; f(); }
  G.modal = name;
  sheet.innerHTML = `<div class="sheet-h"><h3>${title}</h3>${closable ? '<button class="x" id="xBtn" aria-label="Đóng">✕</button>' : ''}</div><div class="sheet-b" id="sheetB">${html}</div>`;
  if (closable) $('xBtn').onclick = () => closeSheet();
  $('scrim').onclick = closable ? () => closeSheet() : null;
  onClose = close || null;
  modalRoot.classList.add('on');
  return sheet;
}

export function setSheetBody(html) { const b = $('sheetB'); if (b) b.innerHTML = html; return b; }

export function closeSheet() {
  if (onClose) { const f = onClose; onClose = null; f(); }
  G.modal = null;
  modalRoot.classList.remove('on');
}

export function toast(text, kind = '') {
  const el = document.createElement('div');
  el.className = 'toast ' + kind;
  el.textContent = text;
  $('toasts').appendChild(el);
  while ($('toasts').children.length > 4) $('toasts').firstChild.remove();
  setTimeout(() => { el.style.opacity = '0'; setTimeout(() => el.remove(), 400); }, 2800);
}

// Kết quả hành động: hiện thông báo, ghi chú phụ (nhiệm vụ, kỷ lục) và cập nhật bản nhìn tài khoản.
export function showResult(res, { quiet = false } = {}) {
  if (res.me) applyMe(res.me);
  if (!res.ok) { toast(res.err || 'Không thực hiện được.', 'warn'); return false; }
  if (res.msg && !quiet) toast(res.msg, 'good');
  for (const n of res.data?.notes || []) toast(n, 'good');
  return true;
}

let meListeners = [];
export function onMe(fn) { meListeners.push(fn); }
export function applyMe(me) {
  me._at = performance.now(); // mốc nhận dữ liệu: các bộ đếm (trứng nở…) tự trừ thời gian đã trôi
  G.me = me;
  meListeners.forEach((f) => f(me));
}

// Bong bóng chat / biểu cảm gắn với một mesh trong cảnh.
const bubbles = [];
export function bubble(mesh, text, { emo = false, ttl = 3.2 } = {}) {
  if (!mesh) return;
  for (const b of bubbles) if (b.mesh === mesh && b.emo === emo) b.ttl = 0;
  const el = document.createElement('div');
  el.className = 'bubble' + (emo ? ' emo' : '');
  el.textContent = text;
  $('bubbles').appendChild(el);
  bubbles.push({ el, mesh, ttl, emo });
}
const v = new window.THREE.Vector3();
export function updateBubbles(dt, camera) {
  for (let i = bubbles.length - 1; i >= 0; i--) {
    const b = bubbles[i];
    b.ttl -= dt;
    if (b.ttl <= 0 || !b.mesh.parent) { b.el.remove(); bubbles.splice(i, 1); continue; }
    b.mesh.getWorldPosition(v);
    v.y += b.emo ? 2.75 : 2.3;
    v.project(camera);
    if (v.z > 1) { b.el.style.display = 'none'; continue; }
    b.el.style.display = '';
    b.el.style.left = ((v.x * 0.5 + 0.5) * window.innerWidth) + 'px';
    b.el.style.top = ((-v.y * 0.5 + 0.5) * window.innerHeight) + 'px';
    b.el.style.opacity = Math.min(1, b.ttl * 2);
  }
}

export function wormById(id) { return G.catalog.worms.find((w) => w.id === id); }
export function foodById(id) { return G.catalog.foods.find((f) => f.id === id); }
export function cosById(id) { return G.catalog.cosmetics.find((c) => c.id === id); }
export const RARITY_COLOR = { 'Thường': '#8a8a8a', 'Khá hiếm': '#3f86c9', 'Hiếm': '#13a08a', 'Sử thi': '#c9901a', 'Huyền thoại': '#b24bd6' };
export const rtag = (w) => `<span class="rtag" style="background:${RARITY_COLOR[w.rarity]}">${esc(w.rarity)}</span>`;
export const SLOT_LABEL = { head: 'Đầu', eye: 'Mắt', neck: 'Cổ', body: 'Thân', back: 'Lưng' };
export const STAGE_NAME = ['Bình thường', 'Sắp đói', 'Đói', 'Đói kéo dài', 'Kiệt sức'];

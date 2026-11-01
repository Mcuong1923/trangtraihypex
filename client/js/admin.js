// Trang quản trị (mục 19). Mọi quyền do máy chủ kiểm tra; trang chỉ ẩn các tab vai trò không dùng được.
const $ = (id) => document.getElementById(id);
const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const KEY = 'hypex_admin_token';
let token = null, role = null, tab = 'overview';
const store = { get: (k) => { try { return localStorage.getItem(k); } catch { return null; } }, set: (k, v) => { try { v ? localStorage.setItem(k, v) : localStorage.removeItem(k); } catch {} } };

async function call(method, path, body) {
  const r = await fetch(path, { method, headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + token }, body: body ? JSON.stringify(body) : undefined });
  const j = await r.json().catch(() => ({ err: 'Phản hồi không hợp lệ.' }));
  if (!r.ok) throw new Error(j.err || 'Lỗi ' + r.status);
  return j;
}

$('lf').onsubmit = async (e) => {
  e.preventDefault();
  const r = await fetch('/api/login', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username: $('lu').value, password: $('lp').value }) }).then((x) => x.json());
  if (r.err) { $('lerr').textContent = r.err; return; }
  token = r.token;
  start().catch((err) => { $('lerr').textContent = err.message; token = null; });
};
$('logout').onclick = () => { store.set(KEY, null); location.reload(); };
$('refresh').onclick = () => render();

async function start() {
  const me = await call('GET', '/api/admin/me');
  role = me.role;
  store.set(KEY, token);
  $('login').hidden = true; $('adminApp').hidden = false;
  $('who').textContent = `${me.username} · vai trò ${{ mod: 'kiểm duyệt', econ: 'kinh tế', admin: 'quản trị viên' }[role]}`;
  const can = (p) => role === 'admin' || role === p;
  const tabs = [['overview', 'Phòng & người chơi', true], ['reports', 'Báo cáo', can('mod')], ['moderate', 'Kiểm duyệt', can('mod')], ['announce', 'Thông báo & bảo trì', can('mod')], ['ledger', 'Sổ cái', can('econ')], ['audit', 'Nhật ký quản trị', role === 'admin']];
  $('tabs').innerHTML = tabs.filter((t) => t[2]).map(([k, l]) => `<button class="tab" data-t="${k}" aria-selected="${tab === k}">${l}</button>`).join('');
  document.querySelectorAll('[data-t]').forEach((b) => (b.onclick = () => { tab = b.dataset.t; document.querySelectorAll('[data-t]').forEach((x) => x.setAttribute('aria-selected', x === b)); render(); }));
  render();
  setInterval(() => { if (tab === 'overview') render(); }, 10000);
}

async function render() {
  const v = $('view');
  try {
    const o = await call('GET', '/api/admin/overview');
    const m = o.metrics;
    $('kpis').innerHTML = [['Đang online', m.online], ['Phòng mở', m.rooms], ['Tick p95', m.tickP95ms.toFixed(2) + ' ms'], ['Tick tối đa', m.tickMaxms.toFixed(2) + ' ms'], ['Gói bị bỏ', m.sendDrops], ['Ngày hội', `${o.festival.total}/${o.festival.goal}`]]
      .map(([l, x]) => `<div class="kpi"><span>${l}</span><b>${esc(x)}</b></div>`).join('') + (o.maintenance ? '<div class="kpi" style="border-color:var(--bad)"><span>Trạng thái</span><b style="color:var(--bad)">Bảo trì</b></div>' : '');
    if (tab === 'overview') {
      v.innerHTML = `<div class="panel"><h2>Phòng đang mở</h2><div class="tblwrap"><table class="t"><tr><th>Phòng</th><th>Loại</th><th>Sĩ số</th><th>Người chơi</th></tr>
        ${(o.rooms || []).map((r) => `<tr><td>${esc(r.Name)}</td><td>${r.Kind === 'home' ? 'Nhà riêng' : 'Chuồng chung'}</td><td>${r.Count}</td>
          <td>${(r.Players || []).map((p) => `${esc(p.Name)} <span class="mut">#${esc(p.Code)} · cấp ${p.Level} · ${esc(p.Zone)}${p.Gone ? ' · mất kết nối' : ''}</span>`).join('<br>') || '—'}</td></tr>`).join('') || '<tr><td colspan="4">Chưa có phòng nào.</td></tr>'}
        </table></div></div>`;
    } else if (tab === 'reports') {
      const { reports } = await call('GET', '/api/admin/reports');
      v.innerHTML = `<div class="panel"><h2>Hàng đợi báo cáo</h2>${(reports || []).map((r) => `<div class="row" style="align-items:flex-start">
        <div class="meta"><div class="nm"><span class="pill ${r.status}">${{ open: 'Đang mở', actioned: 'Đã xử lý', dismissed: 'Bỏ qua' }[r.status]}</span> ${esc(r.name)} · ${esc(r.reason)}</div>
          <div class="sub">${new Date(r.at).toLocaleString('vi-VN')} · ${esc(r.room || '')}${r.handler ? ' · xử lý bởi ' + esc(r.handler) + (r.note ? ': ' + esc(r.note) : '') : ''}</div>
          ${(r.evidence || []).map((e) => `<div class="ev">“${esc(e.text)}” — ${new Date(e.at).toLocaleTimeString('vi-VN')} · ${esc(e.id)}</div>`).join('') || '<div class="ev">Không có dòng chat nào trong nhật ký phòng.</div>'}</div>
        ${r.status === 'open' ? `<div class="act" style="flex-direction:column;align-items:stretch">
          <input placeholder="Ghi chú" data-note="${r.id}" style="padding:7px;border-radius:8px;border:2px solid var(--line)">
          <button class="mini hot" data-rs="${r.id}" data-a="mute">Khoá chat 30 phút</button>
          <button class="mini hot" data-rs="${r.id}" data-a="ban">Khoá TK 1 ngày</button>
          <button class="mini sec" data-rs="${r.id}" data-a="dismiss">Bỏ qua</button></div>` : ''}</div>`).join('') || '<p class="mut">Chưa có báo cáo.</p>'}</div>`;
      document.querySelectorAll('[data-rs]').forEach((b) => (b.onclick = async () => {
        const note = document.querySelector(`[data-note="${b.dataset.rs}"]`).value;
        try { await call('POST', '/api/admin/reports/resolve', { id: b.dataset.rs, action: b.dataset.a, minutes: b.dataset.a === 'ban' ? 1440 : 30, note: note || 'Vi phạm quy tắc cộng đồng' }); render(); } catch (err) { alertBox(err.message); }
      }));
    } else if (tab === 'moderate') {
      v.innerHTML = `<div class="panel"><h2>Kiểm duyệt theo mã người chơi</h2>
        <form class="form" id="mf"><div><label class="l" for="mc">Mã người chơi</label><input id="mc" placeholder="K7QM2D" style="text-transform:uppercase"></div>
        <div><label class="l" for="ma">Hành động</label><select id="ma"><option value="mute">Khoá chat</option><option value="unmute">Mở khoá chat</option><option value="ban">Khoá tài khoản</option><option value="unban">Mở khoá tài khoản</option><option value="kick">Mời ra khỏi game</option></select></div>
        <div><label class="l" for="mm">Số phút</label><input id="mm" type="number" value="60" min="1" style="width:90px"></div>
        <div style="flex:1"><label class="l" for="mr">Lý do (bắt buộc)</label><input id="mr" placeholder="vd: spam chat"></div>
        <button class="btn">Áp dụng</button></form><p id="mres" class="mut"></p></div>`;
      $('mf').onsubmit = async (e) => {
        e.preventDefault();
        try { await call('POST', '/api/admin/moderate', { code: $('mc').value.trim(), action: $('ma').value, minutes: +$('mm').value, reason: $('mr').value }); $('mres').textContent = 'Đã áp dụng và ghi nhật ký.'; } catch (err) { $('mres').textContent = err.message; }
      };
    } else if (tab === 'announce') {
      v.innerHTML = `<div class="panel"><h2>Thông báo toàn hệ thống</h2><form class="form" id="af"><div style="flex:1"><label class="l" for="at">Nội dung</label><input id="at" maxlength="160"></div><button class="btn">Gửi</button></form><p id="ares" class="mut"></p></div>
        ${role === 'admin' ? `<div class="panel"><h2>Bảo trì</h2><p class="mut">Bật bảo trì sẽ chặn người chơi mới vào game; người đang chơi giữ nguyên phiên và tài sản.</p>
        <form class="form" id="bf"><div style="flex:1"><label class="l" for="bt">Lời nhắn</label><input id="bt" placeholder="vd: nâng cấp máy chủ 15 phút"></div>
        <button class="btn" data-on="1">Bật bảo trì</button><button class="btn sec" data-on="0">Tắt bảo trì</button></form><p id="bres" class="mut"></p></div>` : ''}`;
      $('af').onsubmit = async (e) => { e.preventDefault(); try { await call('POST', '/api/admin/announce', { text: $('at').value }); $('ares').textContent = 'Đã gửi tới mọi chuồng.'; $('at').value = ''; } catch (err) { $('ares').textContent = err.message; } };
      if ($('bf')) document.querySelectorAll('#bf [data-on]').forEach((b) => (b.onclick = async (e) => { e.preventDefault(); try { await call('POST', '/api/admin/maintenance', { on: b.dataset.on === '1', text: $('bt').value }); $('bres').textContent = b.dataset.on === '1' ? 'Đã bật bảo trì.' : 'Đã tắt bảo trì.'; render(); } catch (err) { $('bres').textContent = err.message; } }));
    } else if (tab === 'ledger') {
      v.innerHTML = `<div class="panel"><h2>Truy vết giao dịch thóc</h2><form class="form" id="lgf"><div><label class="l" for="lc">Mã người chơi</label><input id="lc" style="text-transform:uppercase"></div><button class="btn">Xem sổ cái</button></form><div id="lgv"></div></div>`;
      $('lgf').onsubmit = async (e) => {
        e.preventDefault();
        try {
          const r = await call('GET', '/api/admin/ledger?code=' + encodeURIComponent($('lc').value.trim()));
          $('lgv').innerHTML = `<h2>${esc(r.name)}</h2><div class="tblwrap"><table class="t"><tr><th>Thời điểm</th><th>Lý do</th><th>Mã nghiệp vụ</th><th>Biến động</th><th>Số dư</th></tr>
            ${(r.entries || []).slice().reverse().map((x) => `<tr><td>${new Date(x.at).toLocaleString('vi-VN')}</td><td>${esc(x.reason)}</td><td>${esc(x.ref)}</td><td style="color:${x.delta < 0 ? 'var(--bad)' : 'var(--good)'}">${x.delta > 0 ? '+' : ''}${x.delta}</td><td>${x.balance}</td></tr>`).join('') || '<tr><td colspan="5">Chưa có giao dịch.</td></tr>'}</table></div>`;
        } catch (err) { $('lgv').innerHTML = `<p class="err">${esc(err.message)}</p>`; }
      };
    } else if (tab === 'audit') {
      const { lines } = await call('GET', '/api/admin/audit');
      v.innerHTML = `<div class="panel"><h2>Nhật ký quản trị</h2><div class="tblwrap"><table class="t"><tr><th>Thời điểm</th><th>Nhân sự</th><th>Hành động</th><th>Đối tượng</th><th>Chi tiết</th></tr>
        ${(lines || []).slice().reverse().map((l) => `<tr><td>${new Date(l.at).toLocaleString('vi-VN')}</td><td>${esc(l.staff)}</td><td>${esc(l.action)}</td><td>${esc(l.target || '')}</td><td>${esc(l.detail || '')}</td></tr>`).join('')}</table></div></div>`;
    }
  } catch (err) {
    v.innerHTML = `<div class="panel"><p class="err">${esc(err.message)}</p></div>`;
  }
}

function alertBox(text) { const p = document.createElement('p'); p.className = 'err'; p.textContent = text; $('view').prepend(p); setTimeout(() => p.remove(), 4000); }

token = store.get(KEY);
if (token) start().catch(() => { token = null; store.set(KEY, null); });

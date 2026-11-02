// Các bảng tác vụ. Mọi nút gọi server qua G.net.act; client không tự cộng/trừ tài sản.
import { G, $, esc, openSheet, setSheetBody, closeSheet, toast, showResult, applyMe, wormById, foodById, cosById, rtag, RARITY_COLOR, SLOT_LABEL, STAGE_NAME } from './ui.js';
import { dress } from './models.js';
import { sfx } from './audio.js';
import * as V from './v11.js';
import * as L from './life.js';

const act = (a, args) => G.net.act(a, args);
const fmt = (n) => Number(n).toLocaleString('vi-VN');
const hasWorms = () => G.catalog.worms.some((w) => (G.me.worms[w.id] || 0) > 0);
// Đứng gần một điểm tương tác của khu hiện tại? (G.nearest bị xoá khi đang mở bảng nên tính trực tiếp.)
const nearKind = (kind) => (G.zones?.[G.zone]?.items || []).some((it) => it.kind === kind && Math.hypot(G.self.pos.x - it.x, G.self.pos.z - it.z) < it.r);

// ---------- NPC ----------

function npc(face, bg, who, text, choices) {
  openSheet('npc', 'Trò chuyện', `
    <div class="npcbox"><div class="npcface" style="background:${bg}">${face}</div>
      <div><div class="who">${esc(who)}</div><p>${text}</p></div></div>
    <div class="choices">${choices.map((c, i) => `<button class="choice" data-i="${i}">${c.t}</button>`).join('')}</div>`);
  document.querySelectorAll('.choice').forEach((b) => (b.onclick = () => { const c = choices[+b.dataset.i]; if (c.keep) { c.fn(); return; } closeSheet(); c.fn?.(); }));
}

// Lần đầu gặp Cụ Cục: dặn luật 4 trang ngắn, có nút bỏ qua (PLAN_LOGIC_GAME 6.1 bước 1–2).
const ELDER_RULES = [
  ['🐔', 'Ơ kìa, gà mới! <b>{ten}</b> đấy hả? Chào mừng tới Trại Hypex — nơi giun béo, thóc đầy, và chưa ai bị làm gà rán. Lại đây cụ dặn mấy câu!'],
  ['🍚', 'Luật số một: bụng đói thì mổ chậm. Thanh <b>Độ no</b> góc trên bên trái tụt dần, hết là chân run như cầy sấy. Đầy bụng chơi được chừng 15–20 phút.<div class="bar" style="margin-top:8px"><div class="track"><div class="fill mana" style="width:62%"></div></div></div>'],
  ['🪱', 'Luật số hai: đói thì ăn, nghèo thì đào, lười thì… vẫn phải đào. Ở ụ đất bấm <b>E</b>, giun ló đầu thì <b>giữ nút để kéo</b>, dây đỏ thì thả. Hết sức vẫn đào được giun thường, ăn sống cũng no.'],
  ['🌾', 'Cụ cho {ten} <b>100 thóc</b> làm vốn. Tiêu cho khéo, đừng mua hết mũ! Giờ giữ <kbd>Shift</kbd> (hoặc nút 🏃) chạy ra cửa — Gà Đón Khách chờ ở đó.'],
];
function elderRules(page = 0) {
  const me = G.me, [emo, text] = ELDER_RULES[page], last = page === ELDER_RULES.length - 1;
  npc('🐔', '#f0e2b0', `Gà Cụ Cục · ${page + 1}/${ELDER_RULES.length}`, `${emo} ${text.replaceAll('{ten}', esc(me.char.name))}`, [
    last ? { t: 'Dạ, cháu đi chạy đây! 🏃', fn: null } : { t: 'Tiếp ›', keep: true, fn: () => elderRules(page + 1) },
    { t: '<span class="mut">Tôi biết chơi rồi (bỏ qua hướng dẫn)</span>', keep: true, fn: skipTutorial },
  ]);
}
function skipTutorial() {
  npc('🐔', '#f0e2b0', 'Gà Cụ Cục', 'Bỏ qua thì cháu vẫn giữ 100 thóc khởi nghiệp, nhưng không nhận thưởng hướng dẫn (thóc, cám, trứng) nữa đâu nhé. Chắc chưa?', [
    { t: 'Chắc, cháu tự khám phá!', fn: async () => showResult(await act('tutorial_skip')) },
    { t: 'Thôi, cụ dặn tiếp đi', keep: true, fn: () => elderRules(1) },
  ]);
}

export async function dialogElder(item) {
  const first = !G.me.tutorial?.met_elder && !G.me.tutorialClaimed;
  const res = await act('talk', { id: item.id });
  if (!showResult(res, { quiet: true })) return;
  if (first) { elderRules(0); return; }
  const me = G.me;
  const tutDone = G.catalog.tutorial.every((s) => me.tutorial[s.id]);
  const choices = [];
  if (me.rescue) choices.push({ t: '🧹 Tiếp tục việc “Dọn ổ rơm”', fn: () => startRescue() });
  else if (me.rescueEligible) choices.push({ t: `🧹 Nhận việc “Dọn ổ rơm” (+${G.catalog.balance.rescueReward} thóc)`, fn: () => startRescue() });
  if (me.rationReady) choices.push({ t: '🍚 Xin cụ một khẩu phần cám (mỗi ngày một lần)', fn: async () => showResult(await act('ration')) });
  if (!me.tutorialClaimed && tutDone) choices.push({ t: `🎁 Nhận thưởng hướng dẫn (+${G.catalog.balance.tutorialReward} thóc, 3 cám, 1 trứng)`, fn: claimTutorial });
  choices.push({ t: '📋 Xem nhiệm vụ & hướng dẫn', fn: openQuests });
  choices.push({ t: '🏠 Đổi chuồng / xem các chuồng', fn: openRooms });
  choices.push({ t: '❓ Cách chơi', fn: openHelp });
  choices.push({ t: 'Tạm biệt cụ', fn: null });
  let text;
  if (me.stage >= 2) text = 'Ối, cháu đói lả rồi! Có giun trong túi thì <b>ăn sống</b> ngay, chẳng tốn đồng nào. Hết sạch thì ra vườn đào — kiệt sức vẫn đào được giun thường. Cụ còn để dành cho cháu một khẩu phần cám mỗi ngày đấy.';
  else if (!tutDone) text = `Chào ${esc(me.char.name)}! Đây là sảnh chuồng chung, ai vào trại cũng xuất phát ở đây. Ra cửa chính là sân làng; đi đường đất phía tây tới <b>Vườn giun</b> đào giun, rồi mang ra <b>Chợ thóc</b> bán lấy thóc, nhớ ăn cho no. Lạc đường thì bấm <b>M</b> mở bản đồ. Xong mấy việc trong bảng nhiệm vụ thì quay lại cụ thưởng cho.`;
  else if (!me.tutorialClaimed) text = 'Giỏi lắm! Cháu đã biết đào, bán, ăn và làm quen. Đây, phần thưởng của cụ.';
  else text = 'Trại hôm nay đông vui ghê. Cứ đào, cứ khoe, cứ quậy — miễn đừng để bụng đói nhé!';
  npc('🐔', '#f0e2b0', 'Gà Cụ Cục', text, choices);
}

async function claimTutorial() { showResult(await act('claim_tutorial')); }

export async function dialogPig(item) {
  await act('talk', { id: item.id });
  npc('🐷', '#f3b4bb', 'Lợn Ủn',
    'Đào bới là nghề của tui! Tới ụ đất bấm <b>E</b>: gà mổ đất, giun ló đầu thì <b>giữ nút để kéo</b>. Thanh độ căng đỏ thì <b>thả ra</b> kẻo tuột. Giun hiếm quẫy mạnh, tuột 3 lần là nó chuồn! Hết sức vẫn đào được giun thường, ăn sống một con là có sức ngay.',
    [{ t: '📖 Xem sổ giun', fn: openBook }, { t: '📋 Nhiệm vụ ngày', fn: openQuests }, { t: 'Để tui đào tiếp', fn: null }]);
}

// ---------- Đào giun ----------
// Luồng đào mổ → giun ló → giằng co nằm ở dig.js; ở đây chỉ còn thẻ chiến lợi phẩm.
G.showLoot = (...a) => showLoot(...a);

function showLoot(loot, perfect, notes) {
  const w = wormById(loot.worm);
  sfx(w.rare ? 'rare' : 'pop');
  openSheet('loot', 'Chiến lợi phẩm', `
    <div class="lootcard">
      <div class="big ${w.rare ? 'shine' : ''}">🪱</div>
      <div class="nm" style="color:${w.color}">${esc(w.name)}</div>
      ${rtag(w)}
      <div class="kvs" style="margin:12px auto 0;max-width:280px;text-align:left">
        <b>Chiều dài</b><span>${loot.size} cm${G.me.records[w.id] === loot.size ? ' 🏅' : ''}</span>
        <b>Bán NPC</b><span>🌾 ${w.sell}</span>
        <b>Chế biến & ăn</b><span>+${w.xp} XP · +${w.mana} mana · phí ${w.fee}</span>
        <b>Giằng co</b><span>${perfect ? 'Không tuột lần nào ✔' : 'Có tuột nhưng vẫn kéo được'}</span>
        <b>Mã lần đào</b><span style="font-size:.7rem;overflow-wrap:anywhere">${esc(loot.id)}</span>
      </div>
      ${(notes || []).map((n) => `<p class="mut" style="margin:8px 0 0">✨ ${esc(n)}</p>`).join('')}
      <div class="choices">
        ${w.rare ? '<button class="btn" id="showBtn">📣 Khoe với chuồng</button>' : ''}
        <button class="btn sec" id="okBtn">Đào tiếp</button>
      </div>
    </div>`);
  $('okBtn').onclick = closeSheet;
  if (w.rare) $('showBtn').onclick = async () => { const r = await act('show', { id: loot.id }); if (showResult(r)) closeSheet(); };
}

// ---------- Túi đồ ----------

export function openBag(tab = 'worm') {
  const me = G.me;
  const worms = G.catalog.worms.filter((w) => (me.worms[w.id] || 0) > 0);
  const foods = G.catalog.foods.filter((f) => (me.food[f.id] || 0) > 0);
  if (tab === 'store') return openStore();
  const body = tab === 'worm'
    ? (worms.map((w) => wormRow(w, false)).join('') || '<div class="empty">Chưa có giun. Ra <b>Vườn giun</b> đào nhé! 🪱</div>') +
      `<div class="note">Giun tự đào có 3 cách dùng: <b>bán</b> ở Sạp thu mua (Chợ thóc), <b>chế biến &amp; ăn</b> ngay tại đây (trừ phí, nhận XP + mana), hoặc <b>giữ</b> để sưu tầm. Mỗi con chỉ bán hoặc ăn một lần.</div>`
    : (foods.map((f) => `<div class="row"><div class="ico" style="background:#fff0cf">${f.emo}</div>
        <div class="meta"><div class="nm">${esc(f.name)} ×${me.food[f.id]}</div><div class="sub">+${f.mana} mana · 0 XP · không bán lại</div></div>
        <div class="act"><button class="mini hot" data-eatfood="${f.id}" ${me.mana >= 100 ? 'disabled' : ''}>Ăn</button></div></div>`).join('') ||
      '<div class="empty">Chưa có thức ăn dự trữ. Mua ở <b>Quầy thức ăn</b> tại Chợ thóc.</div>');
  openSheet('bag', 'Túi đồ', `
    ${bagTabs(tab)}
    ${manaLine()}<p class="mut" style="margin:-6px 0 10px">Túi giun ${me.bagCount}/${me.bagCap}${me.bagCount >= me.bagCap ? ' — <b style="color:var(--bad)">đầy, bán hoặc ăn bớt để đào tiếp</b>' : ''}</p>${body}`);
  document.querySelectorAll('[data-tab]').forEach((b) => (b.onclick = () => openBag(b.dataset.tab)));
  bindWormButtons(() => openBag('worm'));
  document.querySelectorAll('[data-eatfood]').forEach((b) => (b.onclick = async () => { b.disabled = true; showResult(await act('eat_food', { id: b.dataset.eatfood })); openBag('food'); }));
}

const bagTabs = (tab) => `<div class="tabs"><button class="tab" data-tab="worm" aria-selected="${tab === 'worm'}">🪱 Giun</button><button class="tab" data-tab="food" aria-selected="${tab === 'food'}">🍱 Thức ăn</button><button class="tab" data-tab="store" aria-selected="${tab === 'store'}">📦 Kho</button></div>`;
const GROUP = { vatlieu: 'Vật liệu xây nhà', hatgiong: 'Hạt giống', nongsan: 'Nông sản', trung: 'Trứng' };
// Kho đồ (PLAN_LOGIC_GAME 11): vật liệu, hạt giống, nông sản, trứng.
function openStore() {
  const me = G.me, items = G.catalog.items.filter((it) => (me.items?.[it.id] || 0) > 0);
  const groups = Object.keys(GROUP).map((g) => {
    const rows = items.filter((it) => it.group === g);
    if (!rows.length) return '';
    return `<h3 class="grp">${GROUP[g]}</h3>` + rows.map((it) => `<div class="row"><div class="ico" style="background:#fff0cf">${it.emo}</div>
      <div class="meta"><div class="nm">${esc(it.name)} ×${me.items[it.id]}</div><div class="sub">${it.mana ? `ăn +${it.mana} độ no · ` : ''}${it.sell ? `bán ${it.sell} thóc ở Quầy thức ăn` : it.group === 'trung' ? 'đem tới Vịt Bông ở Trạm ấp để ấp' : it.group === 'hatgiong' ? 'gieo ở luống Ao và vườn hoặc luống rau nhà riêng' : 'dùng để xây nhà với Bò Mộc'}</div></div>
      <div class="act">${it.mana ? `<button class="mini hot" data-eatitem="${it.id}" ${me.mana >= 100 ? 'disabled' : ''}>Ăn</button>` : ''}</div></div>`).join('');
  }).join('');
  openSheet('bag', 'Túi đồ', `${bagTabs('store')}${manaLine()}<p class="mut" style="margin:-6px 0 10px">Kho ${me.storageCount}/${me.storageCap} (trứng không tính)</p>
    ${groups || '<div class="empty">Kho trống. Quà lên cấp, điểm danh và tiệm vật liệu của Bò Mộc sẽ lấp đầy chỗ này.</div>'}`);
  document.querySelectorAll('[data-tab]').forEach((b) => (b.onclick = () => openBag(b.dataset.tab)));
  document.querySelectorAll('[data-eatitem]').forEach((b) => (b.onclick = async () => { b.disabled = true; showResult(await act('eat_item', { id: b.dataset.eatitem })); openStore(); }));
}

// Quà lên cấp (PLAN_LOGIC_GAME 12): Cụ Cục chúc mừng, bảng liệt kê quà đã vào ví/kho.
export function showLevelReward(level, rw) {
  const list = [`🌾 ${rw.thoc} thóc`, ...Object.entries(rw.food || {}).map(([id, n]) => `${foodById(id)?.emo || '🍱'} ${n} ${foodById(id)?.name || id}`),
    ...Object.entries(rw.items || {}).map(([id, n]) => { const it = G.catalog.items.find((x) => x.id === id); return `${it?.emo || '🎁'} ${n} ${it?.name || id}`; })];
  if (rw.title) list.push('🏅 Danh hiệu mới');
  sfx('level');
  openSheet('reward', `Lên cấp ${level}!`, `<div class="lootcard"><div class="big">🎁</div><div class="nm">Quà cấp ${level}</div>
    <ul class="rewardlist">${list.map((x) => `<li>${esc(x)}</li>`).join('')}</ul>${rw.unlock ? `<p class="mut">Mở khoá: <b>${esc(rw.unlock)}</b></p>` : ''}
    <button class="btn block" id="rwOk">Cảm ơn Cụ Cục!</button></div>`);
  $('rwOk').onclick = closeSheet;
}

function manaLine() {
  const me = G.me;
  return `<p class="mut" style="margin:0 0 10px">Mana ${Math.ceil(me.mana)}/100 · ${STAGE_NAME[me.stage]} · 🌾 ${fmt(me.thoc)} thóc</p>`;
}

function wormRow(w, atShop) {
  const me = G.me, n = me.worms[w.id] || 0;
  const canEat = me.thoc >= w.fee && me.mana < 100;
  return `<div class="row">
    <div class="ico" style="background:${w.color}33">🪱</div>
    <div class="meta"><div class="nm">${esc(w.name)} ×${n} ${rtag(w)}</div>
      <div class="sub">Bán ${w.sell} · ăn +${w.xp} XP/+${w.mana} mana · phí chế biến ${w.fee}</div></div>
    <div class="act">
      ${atShop ? `<button class="mini good" data-sell="${w.id}">Bán +${w.sell}</button>${n > 1 ? `<button class="mini sec" data-sellall="${w.id}">Bán hết (${n})</button>` : ''}` : ''}
      <button class="mini hot" data-eat="${w.id}" ${canEat ? '' : 'disabled'} title="${me.mana >= 100 ? 'Độ no đã đầy' : me.thoc < w.fee ? 'Thiếu thóc trả phí' : ''}">Chế biến &amp; ăn −${w.fee}</button>
      ${atShop ? '' : `<button class="mini sec" data-raw="${w.id}" ${me.mana >= 100 ? 'disabled' : ''} title="Không tốn thóc, không cộng XP">Ăn sống +${Math.round(w.mana * (G.catalog.balance.rawEatRatio || 0.6))}</button>`}
    </div></div>`;
}

function bindWormButtons(refresh) {
  document.querySelectorAll('[data-eat]').forEach((b) => (b.onclick = async () => {
    b.disabled = true;
    const r = await act('eat_worm', { id: b.dataset.eat });
    if (showResult(r) && r.data?.levelUp) G.fx?.('levelup', G.self.mesh);
    refresh();
  }));
  document.querySelectorAll('[data-raw]').forEach((b) => (b.onclick = async () => { b.disabled = true; showResult(await act('eat_raw', { id: b.dataset.raw })); refresh(); }));
  document.querySelectorAll('[data-sell]').forEach((b) => (b.onclick = async () => { b.disabled = true; if (showResult(await act('sell', { id: b.dataset.sell }))) sfx('coin'); refresh(); }));
  document.querySelectorAll('[data-sellall]').forEach((b) => (b.onclick = async () => {
    b.disabled = true;
    const id = b.dataset.sellall;
    let total = 0;
    while ((G.me.worms[id] || 0) > 0) {
      const r = await act('sell', { id });
      if (r.me) applyMe(r.me);
      if (!r.ok) { toast(r.err, 'warn'); break; }
      total += wormById(id).sell;
      for (const n of r.data?.notes || []) toast(n, 'good');
    }
    if (total) { toast(`Đã bán: +${total} thóc`, 'good'); sfx('coin'); }
    refresh();
  }));
}

// ---------- Chợ ----------

export function openSell() {
  const worms = G.catalog.worms.filter((w) => (G.me.worms[w.id] || 0) > 0);
  openSheet('sell', 'Sạp thu mua giun · Lợn Ủn', `${manaLine()}
    ${worms.map((w) => wormRow(w, true)).join('') || '<div class="empty">Túi chưa có giun để bán. Ra <b>Vườn giun</b> đào đã nhé!</div>'}
    <div class="note">Giá theo độ hiếm. Thức ăn mua ở quầy là loại riêng, không bán lại được.</div>`);
  bindWormButtons(openSell);
}

export function openFood() {
  const me = G.me;
  openSheet('food', 'Quầy thức ăn', `${manaLine()}
    ${G.catalog.foods.map((f) => `<div class="row"><div class="ico" style="background:#fff0cf">${f.emo}</div>
      <div class="meta"><div class="nm">${esc(f.name)}</div><div class="sub">🌾 ${f.cost} · +${f.mana} mana · 0 XP${me.food[f.id] ? ` · đang có ${me.food[f.id]}` : ''}</div></div>
      <div class="act"><button class="mini hot" data-buy="${f.id}" data-eat="1" ${me.thoc < f.cost || me.mana >= 100 ? 'disabled' : ''}>Mua &amp; ăn</button>
        <button class="mini sec" data-buy="${f.id}" ${me.thoc < f.cost ? 'disabled' : ''}>Mua cất túi</button></div></div>`).join('')}
    <div class="note">Ăn đưa mana tối đa 100, phần dư mất. Thức ăn là chi phí sinh hoạt để tránh phạt đói; mana không tự hồi khi đăng nhập, đổi chuồng hay qua ngày (trừ khi ngủ ổ trong nhà đã xây).</div>
    ${L.seedsSection()}`);
  document.querySelectorAll('[data-buy]').forEach((b) => (b.onclick = async () => {
    b.disabled = true;
    showResult(await act('buy_food', { id: b.dataset.buy, eat: !!b.dataset.eat }));
    openFood();
  }));
  L.bindSeeds(openFood);
}

// Quầy thời trang: thử trên gà 3D (chỉ xem trước), giỏ hàng hiện tổng thóc trước khi mua.
let tryOn = {}, fashionFilter = 'all', fashionSort = 'price';
export function openFashion() {
  G.spinSelf = true;
  renderFashion();
}
function renderFashion() {
  const me = G.me;
  let items = G.catalog.cosmetics.filter((c) => fashionFilter === 'all' || c.slot === fashionFilter);
  items = items.slice().sort((a, b) => (fashionSort === 'price' ? a.cost - b.cost : a.theme.localeCompare(b.theme)));
  const cart = Object.values(tryOn).filter((id) => id && !me.owned[id]).map(cosById);
  const sum = cart.reduce((s, c) => s + c.cost, 0);
  const tabs = [['all', 'Tất cả'], ...G.catalog.slots.map((s) => [s, SLOT_LABEL[s]])];
  openSheet('fashion', 'Quầy thời trang', `
    <p class="mut" style="margin:0 0 8px">Bấm để <b>thử</b> lên gà (xoay 360° phía sau). Chỉ đổi ngoại hình, không cộng chỉ số. Váy và vest cùng slot Thân nên thay nhau.</p>
    <div class="tabs">${tabs.map(([k, l]) => `<button class="tab" data-f="${k}" aria-selected="${fashionFilter === k}">${l}</button>`).join('')}
      <button class="tab" id="sortBtn" aria-selected="false">Sắp: ${fashionSort === 'price' ? 'giá' : 'chủ đề'} ⇅</button></div>
    <div class="grid">${items.map((c) => {
      const owned = me.owned[c.id], on = me.equipped[c.slot] === c.id, trying = tryOn[c.slot] === c.id;
      return `<button class="cos ${owned ? 'owned' : ''} ${on ? 'on' : ''} ${trying ? 'try' : ''}" data-try="${c.id}">
        <span class="ci">${c.emo}</span><span class="cn">${esc(c.name)}</span>
        <span class="mut">${SLOT_LABEL[c.slot]} · ${esc(c.theme)}${c.season ? (G.catalog.season?.id === c.season ? ' · <b style="color:var(--accent)">chỉ bán trong mùa</b>' : ' · hết mùa') : ''}</span>
        <span class="mut"><b>${owned ? (on ? 'Đang mặc' : 'Đã sở hữu') : '🌾 ' + fmt(c.cost)}</b></span></button>`;
    }).join('')}</div>
    <div class="cart"><span class="sum">${cart.length ? `Giỏ: ${cart.length} món · 🌾 ${fmt(sum)}` : 'Chưa chọn món mới'} <span class="mut">(ví ${fmt(me.thoc)})</span></span>
      <button class="mini sec" id="resetTry">Bỏ thử</button>
      <button class="mini hot" id="buyCart" ${!cart.length || sum > me.thoc ? 'disabled' : ''}>Mua &amp; mặc</button>
      ${Object.keys(tryOn).some((s) => tryOn[s] && me.owned[tryOn[s]]) ? '<button class="mini good" id="wearOwned">Mặc món đã có</button>' : ''}</div>`,
    { close: () => { tryOn = {}; G.spinSelf = false; redressSelf(); } });
  preview();
  document.querySelectorAll('[data-f]').forEach((b) => (b.onclick = () => { fashionFilter = b.dataset.f; renderFashionKeep(); }));
  $('sortBtn').onclick = () => { fashionSort = fashionSort === 'price' ? 'theme' : 'price'; renderFashionKeep(); };
  document.querySelectorAll('[data-try]').forEach((b) => (b.onclick = () => {
    const c = cosById(b.dataset.try);
    tryOn[c.slot] = tryOn[c.slot] === c.id ? undefined : c.id;
    renderFashionKeep();
  }));
  $('resetTry').onclick = () => { tryOn = {}; renderFashionKeep(); };
  $('buyCart').onclick = async () => {
    $('buyCart').disabled = true;
    for (const c of cart) {
      const r = await act('buy_cos', { id: c.id });
      if (!showResult(r)) break;
    }
    await wearTried();
    tryOn = {};
    renderFashionKeep();
  };
  if ($('wearOwned')) $('wearOwned').onclick = async () => { await wearTried(); tryOn = {}; renderFashionKeep(); };
}
// Vẽ lại bảng sẽ chạy callback đóng của bảng cũ (xoá thử đồ, dừng xoay) nên khôi phục sau đó.
function renderFashionKeep() { const keep = tryOn; renderFashion(); tryOn = keep; G.spinSelf = true; preview(); }
async function wearTried() {
  for (const slot in tryOn) {
    const id = tryOn[slot];
    if (id && G.me.owned[id] && G.me.equipped[slot] !== id) showResult(await act('equip', { slot, id }), { quiet: true });
  }
}
function preview() {
  const eq = { ...G.me.equipped };
  for (const s in tryOn) if (tryOn[s]) eq[s] = tryOn[s];
  dress(G.self.mesh, eq, G.me.char, G.catalog.cosmetics);
}
export function redressSelf() { if (G.self) dress(G.self.mesh, G.me.equipped, G.me.char, G.catalog.cosmetics); }

// ---------- Tủ đồ ----------

export function openWardrobe() {
  const me = G.me;
  const slots = G.catalog.slots.map((slot) => {
    const owned = G.catalog.cosmetics.filter((c) => c.slot === slot && me.owned[c.id]);
    return `<h2 style="margin:12px 0 6px">${SLOT_LABEL[slot]}</h2><div class="grid">
      <button class="cos ${!me.equipped[slot] ? 'on' : ''}" data-slot="${slot}" data-id=""><span class="ci">${slot === 'body' && me.char.shirt !== 'none' ? '👕' : '🚫'}</span><span class="cn">${slot === 'body' && me.char.shirt !== 'none' ? 'Áo khởi điểm' : 'Không mặc'}</span></button>
      ${owned.map((c) => `<button class="cos owned ${me.equipped[slot] === c.id ? 'on' : ''}" data-slot="${slot}" data-id="${c.id}"><span class="ci">${c.emo}</span><span class="cn">${esc(c.name)}</span></button>`).join('')}
      ${owned.length ? '' : '<span class="mut" style="align-self:center">Chưa có món nào — ghé Quầy thời trang.</span>'}</div>`;
  }).join('');
  const presets = me.outfits.map((o, i) => `<div class="row"><div class="ico" style="background:#f0e3ff">${i + 1}</div>
    <div class="meta"><div class="nm">Bộ phối ${i + 1}</div><div class="sub">${o ? Object.values(o).map((id) => cosById(id)?.emo || '').join(' ') || 'Không đồ' : 'Trống'}</div></div>
    <div class="act"><button class="mini sec" data-save="${i}">Lưu bộ đang mặc</button><button class="mini hot" data-load="${i}" ${o ? '' : 'disabled'}>Mặc</button></div></div>`).join('');
  openSheet('wardrobe', 'Tủ đồ', `<p class="mut" style="margin:0">Trang bị món đã sở hữu. Lưu tối đa 3 bộ phối sẵn; không đổi đồ trong trận PK.</p>
    <h2>Bộ phối sẵn</h2>${presets}${slots}`);
  document.querySelectorAll('[data-slot]').forEach((b) => (b.onclick = async () => { showResult(await act('equip', { slot: b.dataset.slot, id: b.dataset.id }), { quiet: true }); openWardrobe(); }));
  document.querySelectorAll('[data-save]').forEach((b) => (b.onclick = async () => { if (showResult(await act('outfit_save', { i: +b.dataset.save }), { quiet: true })) toast('Đã lưu bộ phối ' + (+b.dataset.save + 1), 'good'); openWardrobe(); }));
  document.querySelectorAll('[data-load]').forEach((b) => (b.onclick = async () => { showResult(await act('outfit_load', { i: +b.dataset.load }), { quiet: true }); openWardrobe(); }));
}

// ---------- Sổ giun ----------

let bookTab = 'worms';
export function openBook(tab) {
  if (typeof tab === 'string') bookTab = tab;
  const me = G.me;
  const tabs = `<div class="tabs">${[['worms', 'Sổ giun'], ['discover', 'Khám phá & danh hiệu']].map(([k, l]) => `<button class="tab" data-bt="${k}" aria-selected="${bookTab === k}">${l}</button>`).join('')}</div>`;
  if (bookTab === 'discover') {
    const z = { lobby: 'Sảnh chuồng', garden: 'Vườn giun', market: 'Chợ thóc', arena: 'Sân tỷ thí', home: 'Chuồng riêng' };
    const n = { ga_cu_cuc: 'Gà Cụ Cục', lon_un: 'Lợn Ủn', cho_muc: 'Chó Mực', bo_moc: 'Bò Mộc' };
    const chip = (ok, t) => `<span class="chip" style="cursor:default;${ok ? 'background:var(--good);color:#fff;border-color:var(--good)' : 'opacity:.6'}">${ok ? '✔ ' : ''}${esc(t)}</span>`;
    openSheet('book', 'Sổ khám phá', `${tabs}
      <h2 style="margin-top:0">Khu đã tới</h2><div class="chips">${Object.entries(z).map(([k, t]) => chip(me.discover['zone:' + k], t)).join('')}</div>
      <h2>NPC đã gặp</h2><div class="chips">${Object.entries(n).map(([k, t]) => chip(me.discover['npc:' + k], t)).join('')}</div>
      <h2>Danh hiệu</h2>
      ${G.catalog.titles.map((t) => {
        const got = me.titles?.includes(t.id), on = me.title === t.id;
        return `<div class="row" style="${got ? '' : 'opacity:.6'}"><div class="ico" style="background:${got ? '#fff0cf' : '#eee3cf'}">${got ? '🏅' : '🔒'}</div>
          <div class="meta"><div class="nm">${esc(t.name)}</div><div class="sub">${esc(t.hint)}</div></div>
          <div class="act">${got ? `<button class="mini ${on ? 'sec' : 'hot'}" data-title="${on ? '' : t.id}">${on ? 'Bỏ đeo' : 'Đeo'}</button>` : ''}</div></div>`;
      }).join('')}`);
    document.querySelectorAll('[data-bt]').forEach((b) => (b.onclick = () => openBook(b.dataset.bt)));
    document.querySelectorAll('[data-title]').forEach((b) => (b.onclick = async () => { b.disabled = true; showResult(await act('title', { id: b.dataset.title }), { quiet: true }); openBook('discover'); }));
    return;
  }
  const got = G.catalog.worms.filter((w) => me.collection[w.id]).length;
  const rows = G.catalog.worms.map((w) => {
    const seen = me.collection[w.id];
    return `<div class="row" style="${seen ? '' : 'opacity:.55'}"><div class="ico" style="background:${w.color}33">${seen ? '🪱' : '❔'}</div>
      <div class="meta"><div class="nm">${seen ? esc(w.name) : '???'} ${rtag(w)}</div>
        <div class="sub">${seen ? `Kỷ lục ${me.records[w.id]} cm · đang giữ ${me.worms[w.id] || 0}` : `Chưa từng đào · tỷ lệ ${(w.prob * 100).toLocaleString('vi-VN')}%`}</div></div></div>`;
  }).join('');
  const recent = (me.loots || []).slice(-8).reverse().map((l) => {
    const w = wormById(l.worm);
    return `<div class="row"><div class="ico" style="background:${w.color}33">🪱</div><div class="meta"><div class="nm">${esc(w.name)} · ${l.size} cm ${rtag(w)}</div>
      <div class="sub">${new Date(l.at).toLocaleTimeString('vi-VN')} · ${l.perfect ? 'canh chuẩn' : 'canh tạm'}${l.shared ? ' · đã khoe' : ''}</div></div>
      ${w.rare && !l.shared ? `<div class="act"><button class="mini hot" data-show="${l.id}">Khoe</button></div>` : ''}</div>`;
  }).join('');
  openSheet('book', 'Sổ giun', `${tabs}<p class="mut" style="margin:0 0 10px">Đã khám phá <b>${got}/${G.catalog.worms.length}</b> loài. Sổ ghi “đã từng đào”, kể cả con đã ăn hoặc bán.</p>
    ${rows}
    <div class="kvs" style="margin:12px 0"><b>Lượt đào có thưởng</b><span>${me.stats.dug} (canh chuẩn ${me.stats.perfect})</span><b>Đã bán</b><span>${me.stats.sold}</span><b>Đã chế biến & ăn</b><span>${me.stats.eaten}</span><b>Lượt khoe</b><span>${me.stats.shows}</span><b>PK</b><span>${me.stats.pkWins} thắng / ${me.stats.pkPlayed} trận</span></div>
    ${recent ? `<h2>Lần đào gần đây</h2>${recent}` : ''}`);
  document.querySelectorAll('[data-show]').forEach((b) => (b.onclick = async () => { b.disabled = true; showResult(await act('show', { id: b.dataset.show })); openBook(); }));
  document.querySelectorAll('[data-bt]').forEach((b) => (b.onclick = () => openBook(b.dataset.bt)));
}

// ---------- Nhiệm vụ & chuồng ----------

let questTab = 'quests';
export function openQuests(tab) {
  if (typeof tab === 'string') questTab = tab;
  if (questTab === 'contracts') { questTab = 'quests'; return V.openContracts(); }
  const tabs = `<div class="tabs">${[['quests', 'Nhiệm vụ'], ['orders', 'Bảng việc làng'], ['weekly', 'Xếp hạng tuần'], ['festival', 'Ngày hội'], ['contracts', 'Hợp đồng xây']].map(([k, l]) => `<button class="tab" data-qt="${k}" aria-selected="${questTab === k}">${l}</button>`).join('')}</div>`;
  if (questTab === 'orders') return openOrders(tabs);
  if (questTab === 'weekly') return openWeekly(tabs);
  if (questTab === 'festival') return openFestival(tabs);
  openQuestList(tabs);
}
function bindQuestTabs() { document.querySelectorAll('[data-qt]').forEach((b) => (b.onclick = () => openQuests(b.dataset.qt))); }

// Bảng xếp hạng tuần (PLAN_LOGIC_GAME 16): chỉ thưởng danh hiệu cho người đứng đầu khi tuần mới bắt đầu.
async function openWeekly(tabs) {
  openSheet('quests', 'Nhiệm vụ & bảng tin', `${tabs}<p class="mut">Đang tải bảng xếp hạng…</p>`);
  bindQuestTabs();
  const r = await act('weekly');
  if (G.modal !== 'quests' || questTab !== 'weekly' || !showResult(r, { quiet: true })) return;
  const d = r.data, me = G.me;
  const fmtV = (b, e) => b.unit === 'cm' ? `${e.value.toFixed(1)} cm${e.note ? ' · ' + esc(e.note) : ''}` : `${e.value} ${b.unit}`;
  const boards = d.boards.map((b) => {
    const rows = b.entries.map((e, i) => `<div class="row${e.id === me.id ? ' me' : ''}"><div class="ico" style="background:${['#f5d76e', '#dfe3e8', '#e8c4a0'][i] || '#f1e3c6'}">${['🥇', '🥈', '🥉'][i] || i + 1}</div>
      <div class="meta"><div class="nm">${esc(e.name)}${e.id === me.id ? ' <span class="mut">(bạn)</span>' : ''}</div><div class="sub">${fmtV(b, e)}</div></div></div>`).join('');
    const win = d.winners?.[b.id];
    return `<h2>${esc(b.name)} <span class="mut" style="font-weight:600">· hạng nhất nhận “${esc(b.title)}”</span></h2>
      ${rows || '<p class="mut">Tuần này chưa có ai. Ra tay ngay!</p>'}
      ${win ? `<p class="mut">Tuần trước: 🏆 <b>${esc(win.name)}</b> (${fmtV(b, win)})</p>` : ''}`;
  }).join('');
  openSheet('quests', 'Nhiệm vụ & bảng tin', `${tabs}
    <p class="mut" style="margin-top:0">Tuần ${esc(d.week)} · tính theo giờ Việt Nam, chốt sáng thứ Hai. PK chỉ tính trận với người thật. Không có thưởng thóc — chỉ có danh hiệu và tiếng tăm!</p>${boards}`);
  bindQuestTabs();
}

function openOrders(tabs) {
  const me = G.me;
  const atBoard = nearKind('board');
  openSheet('quests', 'Nhiệm vụ & bảng tin', `${tabs}
    <p class="mut" style="margin-top:0">Đơn hàng của các NPC, đổi mới lúc 0:00. Mỗi đơn nhận một lần trong ngày; luôn có một đơn giun đất cho người mới và một nhiệm vụ đặc biệt của NPC.</p>
    ${me.orders.map((o) => {
      const item = o.item && G.catalog.items.find((x) => x.id === o.item);
      const name = item ? item.name : wormById(o.worm).name, have = item ? me.items?.[o.item] || 0 : me.worms[o.worm] || 0;
      const gift = Object.entries(o.gift || {}).map(([id, n]) => { const it = G.catalog.items.find((x) => x.id === id); return `${it?.emo || ''}${n} ${esc(it?.name || id)}`; }).join(' · ');
      return `<div class="q ${o.done ? 'done' : ''}"><div class="box">${o.done ? '✔' : ''}</div>
        <div style="flex:1"><div class="qt">${item || o.gift ? '⭐ ' : ''}${esc(o.text)}</div>${o.line ? `<div class="qs">“${esc(o.line)}”</div>` : ''}<div class="qs">${item ? 'Trong kho' : 'Trong túi'}: ${have}/${o.n} ${esc(name)}${gift ? ` · thưởng thêm ${gift}` : ''}</div></div>
        <span class="qr">🌾 ${o.reward}</span>
        ${o.done ? '' : `<button class="mini hot" data-order="${o.id}" ${have < o.n || !atBoard ? 'disabled' : ''} style="margin-left:6px">Giao</button>`}</div>`;
    }).join('')}
    ${atBoard ? '' : '<p class="mut">Đứng cạnh <b>Bảng tin</b> ở sảnh để giao hàng.</p>'}`);
  bindQuestTabs();
  document.querySelectorAll('[data-order]').forEach((b) => (b.onclick = async () => { b.disabled = true; if (showResult(await act('order', { id: b.dataset.order }))) sfx('coin'); openQuests('orders'); }));
}

function openFestival(tabs) {
  const f = G.festival || { name: 'Ngày hội', goal: 1, total: 0 };
  const pct = Math.min(100, (f.total / f.goal) * 100);
  const atBoard = nearKind('board');
  openSheet('quests', 'Nhiệm vụ & bảng tin', `${tabs}
    <h2 style="margin-top:0">🏮 ${esc(f.name)}</h2>
    <p class="mut">Cả cộng đồng cùng góp thóc. Đủ mục tiêu thì sảnh của <b>mọi chuồng</b> treo đèn lồng; người góp từ 100 thóc nhận danh hiệu “Nhà hảo tâm”. Góp là tự nguyện, mục tiêu có trần.</p>
    <div class="digbar" style="height:26px"><div class="fill xp" style="width:${pct}%"></div></div>
    <p style="text-align:center;font-weight:800;margin:0">${fmt(f.total)} / ${fmt(f.goal)} thóc${f.unlocked ? ' · Đã đủ! 🎉' : ''}</p>
    <p class="mut" style="text-align:center">Bạn đã góp ${fmt(G.me.festival || 0)} thóc.</p>
    ${f.unlocked ? '' : `<div class="coinbtns">${G.catalog.festivalAmounts.map((n) => `<button class="mini hot" data-give="${n}" ${!atBoard || G.me.thoc < n ? 'disabled' : ''}>Góp ${n}</button>`).join('')}</div>
      ${atBoard ? '' : '<p class="mut" style="text-align:center">Đứng cạnh <b>Bảng tin</b> ở sảnh để góp.</p>'}`}`);
  bindQuestTabs();
  document.querySelectorAll('[data-give]').forEach((b) => (b.onclick = async () => { b.disabled = true; if (showResult(await act('festival_give', { i: +b.dataset.give }))) sfx('coin'); openQuests('festival'); }));
}

function openQuestList(tabs) {
  const me = G.me, bal = G.catalog.balance;
  const tut = G.catalog.tutorial.map((s) => `<div class="q ${me.tutorial[s.id] ? 'done' : ''}"><div class="box">${me.tutorial[s.id] ? '✔' : ''}</div><div><div class="qt">${esc(s.text)}</div></div></div>`).join('');
  const tutDone = G.catalog.tutorial.every((s) => me.tutorial[s.id]);
  const daily = (me.daily || []).map((q) => `<div class="q ${q.done ? 'done' : ''}"><div class="box">${q.done ? '✔' : ''}</div>
    <div><div class="qt">${esc(q.text)}</div><div class="qs">${Math.min(q.cur, q.need)}/${q.need}</div></div><span class="qr">🌾 ${q.reward}</span></div>`).join('');
  openSheet('quests', 'Nhiệm vụ & bảng tin', `${tabs}
    <h2 style="margin-top:0">Hướng dẫn 15 phút đầu ${me.tutorialClaimed ? '<span class="rtag" style="background:var(--good)">Đã nhận thưởng</span>' : ''}</h2>${tut}
    ${!me.tutorialClaimed ? `<p class="mut">${tutDone ? `Xong rồi! Gặp <b>Gà Cụ Cục</b> ở sảnh để nhận <b>${bal.tutorialReward} thóc</b>.` : `Hoàn thành cả ${G.catalog.tutorial.length} bước để nhận ${bal.tutorialReward} thóc (một lần).`}</p>` : ''}
    <h2>Nhiệm vụ ngày <span class="mut" style="font-weight:600">· reset 0:00 giờ Việt Nam</span></h2>${daily}
    <div class="kvs" style="margin-top:12px"><b>Lượt đào có thưởng hôm nay</b><span>${me.dailyDigs}/${bal.digDailyLimit}</span><b>Hồi đất giữa 2 phần thưởng</b><span>${bal.digCooldownSec} giây</span></div>`);
  bindQuestTabs();
}

export function openRooms(list) {
  if (!list) { G.net.raw({ t: 'rooms' }); openSheet('rooms', 'Các chuồng công cộng', '<p class="mut">Đang tải…</p>'); return; }
  if (G.modal !== 'rooms') return;
  setSheetBody(`<p class="mut" style="margin:0 0 10px">Mọi chuồng có cùng bố cục và tiện ích. Đổi chuồng không reset nhiệm vụ, hồi đào hay phần thưởng; mỗi chuồng tối đa ${G.catalog.roomCap} người.</p>
    ${list.map((r) => `<div class="row"><div class="ico" style="background:#f1e3c6">🏠</div>
      <div class="meta"><div class="nm">${esc(r.name)}</div><div class="sub">${r.count}/${r.cap} gà${r.here ? ' · bạn đang ở đây' : ''}</div></div>
      <div class="act">${r.here ? '' : `<button class="mini hot" data-room="${r.id}" ${r.count >= r.cap ? 'disabled' : ''}>${r.count >= r.cap ? 'Đầy' : 'Đổi sang'}</button>`}</div></div>`).join('')}
    <button class="btn sec block" data-room="new">Mở chuồng mới</button>`);
  document.querySelectorAll('[data-room]').forEach((b) => (b.onclick = () => { closeSheet(); G.net.raw({ t: 'switch_room', room: b.dataset.room }); }));
}

// ---------- Cứu đói ----------

export async function startRescue() {
  const r = await act('rescue_start');
  if (!showResult(r, { quiet: true })) return;
  const { job, sec, elapsed } = r.data;
  const end = performance.now() + Math.max(0, sec - elapsed) * 1000;
  let live = true;
  openSheet('rescue', 'Việc cứu đói · Dọn ổ rơm', `
    <p class="mut" style="margin:0">Phụ Gà Cụ Cục dọn ổ rơm để nhận <b>${G.catalog.balance.rescueReward} thóc</b> — đủ mua một khẩu phần cám +20 mana. Đóng bảng vẫn giữ việc; quay lại cụ để nhận công.</p>
    <div class="digbar" style="height:24px"><div class="fill mana" id="rf" style="width:0%"></div></div>
    <p class="mut" id="rt" style="text-align:center">🧹 Đang dọn…</p>`, { close: () => { live = false; } });
  (function frame() {
    if (!live) return;
    const left = (end - performance.now()) / 1000;
    $('rf').style.width = Math.min(100, (1 - left / sec) * 100) + '%';
    if (left > 0) { $('rt').textContent = `🧹 Đang dọn… còn ${Math.ceil(left)} giây`; requestAnimationFrame(frame); return; }
    live = false;
    act('rescue_finish', { id: job }).then((res) => {
      showResult(res);
      if (res.ok) setSheetBody(`<div class="lootcard"><div class="big">🧹</div><div class="nm">Xong việc!</div><p>+${G.catalog.balance.rescueReward} thóc. Ra <b>Chợ thóc → Quầy thức ăn</b> mua khẩu phần cám nhé.</p><button class="btn block" id="rOk">Đi thôi</button></div>`), $('rOk').onclick = closeSheet;
    });
  })();
}

// ---------- Chat ----------

export function chatLine(m) {
  if (m.kind === 'system') return `<div class="cm sys">${esc(m.text)}</div>`;
  // Lời NPC nói với mình: nhãn 🐔 NPC để không lẫn với người chơi thật (PLAN_LOGIC_GAME 1.3).
  if (m.kind === 'npc') return `<div class="cm npc"><span class="au" style="color:#8a5a30">🐔 ${esc(m.name)} <span class="npctag">NPC</span>:</span> ${esc(m.text)}</div>`;
  const self = m.pid === G.pid;
  const who = `<span class="au" style="color:${self ? 'var(--accent)' : 'var(--wood)'}">${esc(m.name)}<span class="mut">#${esc(m.tag)}</span>:</span>`;
  if (m.kind === 'show') {
    const c = m.card;
    return `<div class="cm">${who} khoe chiến lợi phẩm<br><span class="showcard"><span style="font-size:1.5rem">🪱</span><span><b style="color:${c.color}">${esc(c.name)}</b> <span class="rtag" style="background:${RARITY_COLOR[c.rarity]}">${esc(c.rarity)}</span><br><span class="mut">${c.size} cm · ${esc(c.zone)} · ${new Date(c.at).toLocaleTimeString('vi-VN')}</span></span></span></div>`;
  }
  return `<div class="cm">${who} ${esc(m.text)}</div>`;
}

export function openChat() {
  G.chatUnread = 0; $('chatDot').hidden = true;
  openSheet('chat', `Chat chuồng · ${esc(G.room?.name || '')}`, `
    <div id="chatlog">${G.chat.map(chatLine).join('')}</div>
    <form class="chatin" id="chatForm"><input type="text" id="chatMsg" maxlength="120" placeholder="Nhắn với cả chuồng…" autocomplete="off"><button class="mini hot" type="submit">Gửi</button></form>
    <p class="mut" style="margin:8px 0 0">Mọi người trong chuồng đọc được; bong bóng hiện trên đầu gà cho người cùng khu. Nhấn <kbd>Enter</kbd> để mở chat nhanh.</p>`);
  const log = $('chatlog'); log.scrollTop = log.scrollHeight;
  const inp = $('chatMsg');
  $('chatForm').onsubmit = (e) => { e.preventDefault(); const t = inp.value.trim(); if (t) { G.net.raw({ t: 'chat', text: t }); inp.value = ''; } };
  setTimeout(() => inp.focus(), 50);
}

export function appendChat(m) {
  G.chat.push(m);
  if (G.chat.length > 120) G.chat.shift();
  if (G.modal === 'chat') { const log = $('chatlog'); log.insertAdjacentHTML('beforeend', chatLine(m)); log.scrollTop = log.scrollHeight; }
  else if (m.kind !== 'system') { G.chatUnread++; $('chatDot').hidden = false; }
}

// ---------- Hướng dẫn ----------

export function openHelp() {
  const b = G.catalog.balance, c = G.catalog;
  const perMin = b.manaDrainPerSec * 60, fullMin = Math.round(100 / perMin);
  openSheet('help', 'Cách chơi', `
    <p style="margin-top:0"><b>Di chuyển:</b> <kbd>W</kbd><kbd>A</kbd><kbd>S</kbd><kbd>D</kbd> / phím mũi tên, điện thoại dùng joystick trái. <b>Chạy nhanh:</b> giữ <kbd>Shift</kbd> hoặc nút 🏃 — nhanh ×${c.sprintMult} trong ${c.staminaSec} giây, hụt hơi thì nghỉ 5 giây rồi chạy tiếp. <b>Đi tới chỗ chỉ định:</b> click/chạm mặt đất. <b>Camera:</b> kéo để xoay, lăn chuột hoặc hai ngón để zoom; cây và mái che tự mờ khi chắn tầm nhìn. <b>Bản đồ:</b> <kbd>M</kbd>. <b>Tương tác:</b> lại gần rồi nhấn <kbd>E</kbd>. <b>Chat:</b> <kbd>Enter</kbd>.</p>
    <h2>Vòng chơi</h2>
    <p>Gà mới nhận <b>${b.starterThoc || 100} thóc khởi nghiệp</b>; Cụ Cục ở cuối sảnh dẫn từng bước. Ra <b>Vườn giun</b> đào → bán ở <b>Chợ thóc</b> lấy 🌾 hoặc ăn để có XP và độ no → lên cấp nhận thóc, vật liệu, hạt giống, trứng → mua đất, xây nhà, nuôi bạn đồng hành. Ra <b>Sân tỷ thí</b> xem gà NPC đấu hoặc PK vui với bạn.</p>
    <h2>Đào giun</h2>
    <p>Nhấn <kbd>E</kbd> ở ụ đất: gà mổ đất, giun ló lên rồi camera kéo lại gần. <b>Giữ nút</b> (hoặc <kbd>Space</kbd>) để kéo, thấy dây <b>đỏ</b> hay giun quẫy thì <b>thả</b> cho dịu. Giun Hiếm trở lên kéo lâu hơn và <b>tuột 3 lần là chạy mất</b> — lượt đó không tính vào hạn đào trong ngày, vẫn có chút XP an ủi. Bỏ dở giữa chừng cũng tính như giun thoát.</p>
    <h2>Độ no và không bao giờ kẹt</h2>
    <div class="tblwrap"><table class="t"><tr><th>Trạng thái</th><th>Điều kiện</th><th>Hệ quả</th></tr>
      <tr><td>Bình thường</td><td>Độ no ≥ 20</td><td>—</td></tr>
      <tr><td>Sắp đói</td><td>1–19</td><td>Thanh đổi màu, NPC nhắc ăn</td></tr>
      <tr><td>Đói</td><td>= 0</td><td>Đếm ngược tới phạt</td></tr>
      <tr><td>Đói kéo dài</td><td>${Math.round(b.hungerSlowSec / 60)} phút online ở 0</td><td>Đi chậm 15%, ít thể lực chạy</td></tr>
      <tr><td>Kiệt sức</td><td>${Math.round(b.hungerExhaustSec / 60)} phút online ở 0</td><td>Vẫn đào được (chỉ giun thường/khá hiếm, mổ chậm), vẫn ăn, bán, mua</td></tr></table></div>
    <p class="mut">Độ no tụt ${perMin.toLocaleString('vi-VN', { maximumFractionDigits: 1 })} điểm/phút khi online (đầy bụng dùng được khoảng ${fullMin} phút) và ${b.manaPerDig} điểm mỗi lượt đào; ngồi ổ rơm tụt chậm hơn, ngủ ổ trong nhà đã xây thì hồi lại. Hết thóc và đồ ăn: <b>ăn sống</b> giun ngay trong túi, hoặc gặp Cụ Cục xin <b>một phần cám</b> mỗi ngày.</p>
    <h2>Đi lại</h2>
    <p><b>Xe buýt vòng làng</b> do NPC lái chạy liên tục qua ${G.worldMap?.roads?.stops?.length || 9} bến ở cổng các khu: mỗi người có ${c.maxBusTokens} lượt miễn phí, hồi 1 lượt mỗi 5 phút; hết lượt trả ${c.villageFare} thóc. <b>Xe ôm</b> dọc đường sẽ mời bạn: chọn điểm đến trên bản đồ, 1–2 thóc tuỳ xa gần (trời mưa +1 thóc khăn lau). Đứng ở bến nhấn <kbd>E</kbd> để xem xe sắp tới.</p>
    <h2>Nhà, vườn và bạn đồng hành</h2>
    <p><b>Ông Chủ Đầu Tư</b> ở cổng Xóm Rơm Vàng bán lô đất (cạnh đường, ven ao, đồi thông). <b>Bò Mộc</b> bán rơm, gỗ, gạch… và xây nhà theo thiết kế: nhà rơm ấm ít, gỗ ấm vừa, gạch ấm nhất (hồi độ no khi ngủ ổ, chứa nhiều khách hơn). Mua hạt giống ở Quầy thức ăn, gieo ở luống cộng đồng (Ao và vườn) hoặc luống rau nhà, tưới một lần cho nhanh, thu trước khi héo. Trứng ấp ở Trạm ấp nở ra bạn đồng hành, mỗi loài một kỹ năng (đánh hơi giun, mặc cả giá vé, cõng chạy…).</p>
    <h2>Làng sống</h2>
    <p>Một ngày trong làng dài ${Math.round((c.gameDay || 2880) / 60)} phút: ban đêm đèn lồng thắp, Giun phát sáng ngoi lên nhiều hơn. Thỉnh thoảng có mưa nhẹ: ụ đất hồi nhanh hơn 30%. NPC chào hỏi, trêu nhẹ, rủ đi xe — chỉnh mức trò chuyện và tắt trêu trong Cài đặt. Bảng việc làng có nhiệm vụ của NPC; bảng <b>Xếp hạng tuần</b> trao danh hiệu cho người giỏi nhất.</p>
    <h2>Bảng giun</h2>
    <div class="tblwrap"><table class="t"><tr><th>Giun</th><th>Độ hiếm</th><th>Tỷ lệ</th><th>Bán</th><th>XP</th><th>Độ no</th><th>Phí</th></tr>
      ${G.catalog.worms.map((w) => `<tr><td>${esc(w.name)}</td><td>${esc(w.rarity)}</td><td>${(w.prob * 100).toLocaleString('vi-VN')}%</td><td>${w.sell}</td><td>${w.xp}</td><td>${w.mana}</td><td>${w.fee}</td></tr>`).join('')}</table></div>
    ${b.name !== 'live' ? `<div class="note"><b>Máy chủ đang chạy bộ cân bằng “${esc(b.name)}”</b> (hồi đất ${b.digCooldownSec} giây) để thử nhanh. Chơi thật dùng bộ “live”: độ no khoảng 15–20 phút, hồi đất 8 giây.</div>` : ''}
    <button class="btn block" id="helpOk">Chơi thôi!</button>`);
  $('helpOk').onclick = closeSheet;
}

// ---------- PK ----------

export function dialogPK(item) {
  const others = [...G.players.entries()];
  npc('🐶', '#8a7a70', 'Chó Mực (trọng tài)',
    'PK ở đây chỉ để vui: thua thì ngã lăn quay rồi đứng dậy, không mất đồ, không mất cấp, không có thóc cược. Chọn đối thủ đi!',
    [
      { t: '⚡ Mổ nhanh với Gà Máy — bấm nhiều nhất trong 8 giây', fn: () => pkBot('peck') },
      { t: '🪙 Úp ngửa với Gà Máy — 3 lượt tung đồng xu', fn: () => pkBot('coin') },
      ...others.map(([pid, p]) => ({ t: `🤝 Mời ${esc(p.name)}#${esc(p.tag)}…`, keep: true, fn: () => pkInviteMenu(pid, p) })),
      ...(others.length ? [] : [{ t: '<span class="mut">Chưa có ai khác ở Sân tỷ thí để mời. Rủ bạn vào cùng chuồng nhé!</span>', keep: true, fn: () => {} }]),
      { t: 'Để lúc khác', fn: null },
    ]);
}
async function pkBot(mode) { showResult(await act('pk_bot', { mode }), { quiet: true }); }
function pkInviteMenu(pid, p) {
  npc('🐶', '#8a7a70', 'Chó Mực', `Mời <b>${esc(p.name)}</b> chơi chế độ nào? Lời mời hết hạn sau 20 giây.`, [
    { t: '⚡ Mổ nhanh', fn: async () => showResult(await act('pk_invite', { target: pid, mode: 'peck' })) },
    { t: '🪙 Úp ngửa', fn: async () => showResult(await act('pk_invite', { target: pid, mode: 'coin' })) },
    { t: 'Thôi', fn: null },
  ]);
}

let inviteTimer = null;
export function showInvite(m) {
  const bar = $('inviteBar');
  let left = Math.round(m.ttl);
  bar.hidden = false;
  const render = () => {
    bar.innerHTML = `<span class="t">⚔️ ${esc(m.from)} mời bạn ${esc(m.modeName)} (${left}s)</span><button class="mini good" id="invYes">Nhận</button><button class="mini sec" id="invNo">Từ chối</button>`;
    $('invYes').onclick = async () => { hideInvite(); const r = await act('pk_accept', { id: m.id }); showResult(r, { quiet: true }); };
    $('invNo').onclick = async () => { hideInvite(); await act('pk_decline', { id: m.id }); };
  };
  render();
  clearInterval(inviteTimer);
  inviteTimer = setInterval(() => { left--; if (left <= 0) hideInvite(); else bar.querySelector('.t').textContent = `⚔️ ${m.from} mời bạn ${m.modeName} (${left}s)`; }, 1000);
  bar.dataset.id = m.id;
}
export function hideInvite(id) {
  const bar = $('inviteBar');
  if (id && bar.dataset.id !== id) return;
  clearInterval(inviteTimer); bar.hidden = true;
}

export const PK = {
  // Nối lại giữa trận: dựng lại bảng đúng giai đoạn với tỷ số và thời gian còn lại từ máy chủ (mục 10.2).
  resume(m) {
    PK.start({ ...m, countdown: m.state === 'countdown' ? m.countLeft : 0.01 });
    PK.score(m);
    if (m.state !== 'play') return;
    PK.go({ id: m.id });
    const mt = G.match;
    if (m.mode === 'peck') {
      mt.goAt = performance.now() - (m.duration - m.playLeft) * 1000;
    } else if (m.waiting) {
      PK.round({ id: m.id, round: m.round, chooser: m.chooser });
    }
    toast('Đã nối lại trận đấu', 'good');
  },
  start(m) {
    hideInvite();
    G.match = { ...m, a: 0, b: 0, phase: 'countdown', t0: performance.now() };
    const [na, nb] = m.names;
    const me = m.you, opp = 1 - me;
    openSheet('pk', `PK · ${esc(m.modeName)}`, `
      <div class="pkstage">
        <div class="vs"><div class="p"><div>${esc(m.names[me])} (bạn)</div><div class="sc" id="scMe">0</div></div><div style="font-weight:900">VS</div><div class="p"><div>${esc(m.names[opp])}</div><div class="sc" id="scOp">0</div></div></div>
        <div id="pkBody"><div class="pkbig" id="pkCount">${Math.ceil(m.countdown)}</div><p class="mut">${m.mode === 'peck' ? 'Chuẩn bị bấm thật nhanh!' : 'Hai bên luân phiên chọn mặt, máy chủ tung đồng xu.'}</p></div>
      </div>`, { closable: false });
    void na; void nb;
    const tick = () => {
      if (!G.match || G.match.id !== m.id || G.match.phase !== 'countdown') return;
      const left = m.countdown - (performance.now() - G.match.t0) / 1000;
      const el = $('pkCount'); if (el) el.textContent = Math.max(1, Math.ceil(left));
      requestAnimationFrame(tick);
    };
    tick();
  },
  go(m) {
    const mt = G.match; if (!mt || mt.id !== m.id) return;
    mt.phase = 'play'; mt.goAt = performance.now();
    if (mt.mode === 'peck') {
      $('pkBody').innerHTML = `<p class="mut">Còn <b id="pkTime">${mt.duration.toFixed(1)}</b> giây — mổ đi!</p><button class="pktap" id="pkTap">MỔ!</button>`;
      const tap = () => { if (mt.phase === 'play') { G.net.raw({ t: 'act', a: 'pk_tap', id: mt.id }); sfx('tap'); } };
      $('pkTap').onpointerdown = (e) => { e.preventDefault(); tap(); };
      mt.keyTap = (e) => { if (e.code === 'Space' || e.key === 'e') { e.preventDefault(); if (!e.repeat) tap(); } };
      window.addEventListener('keydown', mt.keyTap);
      const timer = () => {
        if (G.match !== mt || mt.phase !== 'play') return;
        const left = mt.duration - (performance.now() - mt.goAt) / 1000;
        const el = $('pkTime'); if (el) el.textContent = Math.max(0, left).toFixed(1);
        if (left <= 0) { const b = $('pkTap'); if (b) { b.disabled = true; b.textContent = 'Hết giờ'; } }
        requestAnimationFrame(timer);
      };
      timer();
    } else {
      $('pkBody').innerHTML = '<div class="pkbig" id="coin">🪙</div><p class="mut" id="coinTxt">Chuẩn bị…</p><div class="coinbtns" id="coinBtns"></div>';
    }
  },
  score(m) {
    const mt = G.match; if (!mt || mt.id !== m.id) return;
    const s = [m.a, m.b];
    $('scMe') && ($('scMe').textContent = s[mt.you]);
    $('scOp') && ($('scOp').textContent = s[1 - mt.you]);
  },
  round(m) {
    const mt = G.match; if (!mt || mt.id !== m.id) return;
    $('coin').textContent = '🪙';
    const mine = m.chooser === mt.you;
    $('coinTxt').innerHTML = `Lượt ${m.round + 1}/${mt.rounds} — ${mine ? '<b>bạn chọn mặt</b>, đối thủ nhận mặt còn lại' : 'đối thủ đang chọn mặt…'}`;
    $('coinBtns').innerHTML = mine ? '<button class="btn" data-side="ngua">🙂 Ngửa</button><button class="btn sec" data-side="up">🥚 Úp</button>' : '';
    document.querySelectorAll('[data-side]').forEach((b) => (b.onclick = async () => {
      document.querySelectorAll('[data-side]').forEach((x) => (x.disabled = true));
      const r = await act('pk_pick', { id: mt.id, side: b.dataset.side });
      if (!r.ok) showResult(r);
    }));
  },
  flip(m) {
    const mt = G.match; if (!mt || mt.id !== m.id) return;
    $('coinBtns').innerHTML = '';
    $('coin').textContent = '🌀';
    const name = { ngua: 'Ngửa 🙂', up: 'Úp 🥚' };
    setTimeout(() => {
      if (G.match !== mt) return;
      $('coin').textContent = m.result === 'ngua' ? '🙂' : '🥚';
      $('coinTxt').innerHTML = `Bạn chọn <b>${name[m.picks[mt.you]]}</b> · ra <b>${name[m.result]}</b> → ${m.winner === mt.you ? '<b style="color:var(--good)">bạn thắng lượt</b>' : '<b style="color:var(--bad)">đối thủ thắng lượt</b>'}`;
      PK.score(m);
    }, 600);
  },
  end(m) {
    const mt = G.match; if (!mt || mt.id !== m.id) return;
    if (mt.keyTap) window.removeEventListener('keydown', mt.keyTap);
    mt.phase = 'end';
    PK.score(m);
    const s = [m.a, m.b];
    const win = m.winner === mt.you, tie = m.winner === -1;
    if (!m.cancelled) sfx(win ? 'win' : tie ? 'pop' : 'lose');
    setTimeout(() => {
      if (G.match !== mt) return;
      G.match = null;
      openSheet('pkend', 'Kết quả PK', `<div class="pkstage">
        <div class="pkbig">${m.cancelled ? '🚫' : tie ? '🤝' : win ? '🏆' : '💫'}</div>
        <h2 style="margin:4px 0">${m.cancelled ? 'Trận bị huỷ' : tie ? 'Hoà!' : win ? 'Bạn thắng!' : 'Bạn thua — ngã lăn quay 😵'}</h2>
        <p class="mut">Tỷ số ${s[mt.you]} – ${s[1 - mt.you]}. ${esc(m.reason || '')}</p>
        <p class="mut">${mt.mode === 'coin' ? 'Úp ngửa là chế độ may rủi, không tính xếp hạng kỹ năng. ' : ''}Không mất đồ, không mất cấp, không thưởng thóc theo thắng trận.</p>
        <button class="btn block" id="pkOk">Đóng</button></div>`);
      $('pkOk').onclick = closeSheet;
    }, mt.mode === 'coin' ? 900 : 300);
  },
};

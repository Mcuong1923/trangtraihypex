// Điều phối client: đăng nhập, tạo gà, nhận trạng thái từ máy chủ Go, hiển thị 3D và gửi input.
import { Net, api } from './net.js';
import { World, fitLabel } from './world.js';
import { buildChicken, dress, label, wormMesh, mat, FEATHER, SHIRT } from './models.js';
import { G, $, esc, openSheet, closeSheet, toast, applyMe, srvNow, homeLeft, onMe, bubble, updateBubbles, STAGE_NAME } from './ui.js';
import * as P from './panels.js';
import * as SO from './social.js';
import { sfx, setAudio } from './audio.js';
import * as V from './v11.js';
import {bindWorldInput} from './input.js';
import {findPath,moveSafe,walkable,areaAt} from './navigation.js';
import * as MAP from './map.js';
import * as D from './dig.js';
import * as NPC from './npc.js';
import * as SB from './arena-scoreboard.js';
import * as L from './life.js';
import * as F from './fish.js';

const T = window.THREE;
const TOKEN_KEY = 'hypex_token';
const store = {
  get(k) { try { return localStorage.getItem(k); } catch { return null; } },
  set(k, v) { try { v == null ? localStorage.removeItem(k) : localStorage.setItem(k, v); } catch {} },
};

const screens = ['authScreen', 'charScreen', 'loadScreen', 'recoveryScreen'];
function showScreen(id) { screens.forEach((s) => ($(s).hidden = s !== id)); }
function loading(title, sub) { $('loadTitle').textContent = title; $('loadSub').innerHTML = sub || ''; showScreen('loadScreen'); }

G.net = new Net();
G.world = new World($('app'));
MAP.initMap();
// Chân dung gà trên HUD: tô màu lông theo gà của người chơi.
const hexStr = (n) => '#' + (n >>> 0).toString(16).padStart(6, '0');
function shadeHex(hex, f) {
  const n = typeof hex === 'number' ? hex : parseInt(String(hex).replace('#', ''), 16);
  return `rgb(${Math.round(((n >> 16) & 255) * f)},${Math.round(((n >> 8) & 255) * f)},${Math.round((n & 255) * f)})`;
}
function syncPortrait(color) {
  const el = document.querySelector('.player-portrait');
  if (!el) return;
  const hex = FEATHER[color] ?? FEATHER.vang;
  el.style.setProperty('--feather', hexStr(hex));
  el.style.setProperty('--feather-shade', shadeHex(hex, 0.82));
}
// Hook xem trạng thái khi phát triển cục bộ; server vẫn quyết định mọi thứ nên không mở đường gian lận.
if (['localhost', '127.0.0.1'].includes(location.hostname)) window.G_dbg = () => G;

// ============ Đăng nhập ============
let registerMode = false, recoverMode = false;
function setAuthTab(reg) {
  registerMode = reg;
  recoverMode = false;
  $('recoverFields').hidden = true;
  $('forgotBtn').hidden = false;
  $('passIn').placeholder = 'Ít nhất 6 ký tự';
  document.querySelector('label[for=passIn]').textContent = 'Mật khẩu';
  $('tabLogin').setAttribute('aria-selected', !reg);
  $('tabRegister').setAttribute('aria-selected', reg);
  $('authBtn').textContent = reg ? 'Tạo tài khoản & vào trại 🐣' : 'Vào trại 🐣';
  $('passIn').autocomplete = reg ? 'new-password' : 'current-password';
  $('authErr').textContent = '';
}
$('tabLogin').onclick = () => setAuthTab(false);
$('tabRegister').onclick = () => setAuthTab(true);
$('forgotBtn').onclick = () => {
  recoverMode = true;
  $('recoverFields').hidden = false;
  $('forgotBtn').hidden = true;
  document.querySelector('label[for=passIn]').textContent = 'Mật khẩu mới';
  $('passIn').autocomplete = 'new-password';
  $('authBtn').textContent = 'Đặt lại mật khẩu';
  $('authErr').textContent = '';
};
$('authForm').onsubmit = async (e) => {
  e.preventDefault();
  $('authBtn').disabled = true;
  const body = { username: $('userIn').value, password: $('passIn').value };
  let path = registerMode ? '/api/register' : '/api/login';
  if (recoverMode) { path = '/api/recover'; body.code = $('recIn').value; }
  const r = await api(path, body);
  $('authBtn').disabled = false;
  if (r.err) { $('authErr').textContent = r.err; return; }
  store.set(TOKEN_KEY, r.token);
  if (r.recovery) { showRecovery(r.recovery, r.token); return; }
  loading('Đang mở cổng trại…', 'Kết nối máy chủ');
  G.net.connect(r.token);
};
// Mã khôi phục chỉ hiện một lần sau khi đăng ký hoặc đặt lại mật khẩu (mục 14).
function showRecovery(code, token) {
  $('recCode').textContent = code;
  showScreen('recoveryScreen');
  $('recCopy').onclick = () => navigator.clipboard?.writeText(code).then(() => toast('Đã chép mã khôi phục', 'good'), () => {});
  $('recDone').onclick = () => { loading('Đang mở cổng trại…', 'Kết nối máy chủ'); G.net.connect(token); };
}

// ============ Tạo gà ============
const draft = { name: '', color: 'vang', comb: 'don', tail: 'quat', eyes: 'tron', shirt: 'none', sex: 'an' };
let pv = null;
function initCreator(catalog) {
  showScreen('charScreen');
  if (pv) return;
  const names = { comb: { don: 'Mào đơn', kep: 'Mào kép', ro: 'Mào rô' }, sex: { nam: 'Nam', nu: 'Nữ', khac: 'Khác', an: 'Không hiển thị' },
    tail: { quat: 'Đuôi quạt', nhon: 'Đuôi nhọn', xoan: 'Đuôi xoăn' }, eyes: { tron: 'Tròn', to: 'Mắt to', cuoi: 'Cười', mi: 'Mi cong', sao: 'Ngôi sao', ngai: 'Ngái ngủ' } };
  const pick = (container, key, values, render) => {
    container.innerHTML = '';
    values.forEach((v) => {
      const b = document.createElement('button');
      b.type = 'button';
      render(b, v);
      b.setAttribute('aria-pressed', draft[key] === v);
      b.onclick = () => { draft[key] = v; container.querySelectorAll('button').forEach((x) => x.setAttribute('aria-pressed', x === b)); rebuild(); };
      container.appendChild(b);
    });
  };
  const hex = (n) => '#' + n.toString(16).padStart(6, '0');
  pick($('colorSw'), 'color', catalog.colors, (b, v) => { b.className = 'sw'; b.style.background = hex(FEATHER[v]); b.title = v; b.setAttribute('aria-label', 'Màu ' + v); });
  pick($('combCh'), 'comb', catalog.combs, (b, v) => { b.className = 'chip'; b.textContent = names.comb[v]; });
  pick($('tailCh'), 'tail', catalog.tails, (b, v) => { b.className = 'chip'; b.textContent = names.tail[v]; });
  pick($('eyeCh'), 'eyes', catalog.eyes, (b, v) => { b.className = 'chip'; b.textContent = names.eyes[v]; });
  pick($('shirtSw'), 'shirt', catalog.shirts, (b, v) => { b.className = 'sw'; b.style.background = SHIRT[v] == null ? 'repeating-linear-gradient(45deg,#eee,#eee 6px,#ccc 6px,#ccc 12px)' : hex(SHIRT[v]); b.setAttribute('aria-label', v === 'none' ? 'Không áo' : 'Áo ' + v); });
  pick($('sexCh'), 'sex', catalog.sexes, (b, v) => { b.className = 'chip'; b.textContent = names.sex[v]; });
  $('nameIn').oninput = (e) => (draft.name = e.target.value);

  const wrap = $('previewWrap');
  const r = new T.WebGLRenderer({ antialias: true, alpha: true });
  r.setPixelRatio(Math.min(devicePixelRatio, 2));
  r.setSize(wrap.clientWidth, wrap.clientHeight); wrap.appendChild(r.domElement);
  const sc = new T.Scene();
  sc.add(new T.HemisphereLight(0xffffff, 0x886644, 1.1));
  const dl = new T.DirectionalLight(0xffffff, 0.8); dl.position.set(3, 5, 4); sc.add(dl);
  const cam = new T.PerspectiveCamera(40, wrap.clientWidth / wrap.clientHeight, 0.1, 50);
  cam.position.set(0, 1.3, 3.4); cam.lookAt(0, 0.85, 0);
  pv = { r, sc, cam, chk: null, spin: 0.012 };
  let dragX = null;
  r.domElement.addEventListener('pointerdown', (e) => { dragX = e.clientX; pv.spin = 0; });
  window.addEventListener('pointermove', (e) => { if (dragX != null && pv.chk) { pv.chk.rotation.y += (e.clientX - dragX) * 0.012; dragX = e.clientX; } });
  window.addEventListener('pointerup', () => { dragX = null; });
  rebuild();
  (function spin() { if ($('charScreen').hidden) return requestAnimationFrame(spin); if (pv.chk) pv.chk.rotation.y += pv.spin; pv.r.render(pv.sc, pv.cam); requestAnimationFrame(spin); })();
}
function rebuild() {
  if (pv.chk) pv.sc.remove(pv.chk);
  pv.chk = buildChicken({ color: draft.color, comb: draft.comb, tail: draft.tail, eyes: draft.eyes });
  dress(pv.chk, {}, { shirt: draft.shirt }, []);
  pv.chk.position.y = -0.3;
  pv.sc.add(pv.chk);
}
$('createBtn').onclick = () => {
  $('charErr').textContent = '';
  G.net.raw({ t: 'create_char', name: draft.name, color: draft.color, comb: draft.comb, tail: draft.tail, eyes: draft.eyes, shirt: draft.shirt, sex: draft.sex });
};

// ============ Người chơi trong cảnh ============
function makeAvatar(p) {
  const mesh = buildChicken({ color: p.char.color, comb: p.char.comb, tail: p.char.tail, eyes: p.char.eyes });
  dress(mesh, p.equipped, p.char, G.catalog.cosmetics);
  setNameTag(mesh, tagText(p.name, p.level, p.title, p.guild), false);
  G.world.scene.add(mesh);
  return mesh;
}
const tagText = (name, level, title, guild) => `${guild ? '[' + guild + '] ' : ''}${name} · Cấp ${level}${title ? ' · ' + title : ''}`;
function setNameTag(mesh, text, self) {
  if (mesh.userData.tag) mesh.remove(mesh.userData.tag);
  const t = label(text, { bg: self ? 'rgba(224,89,43,.85)' : 'rgba(61,43,26,.72)', scale: 0.5 });
  t.position.y = 2.0; mesh.add(t); mesh.userData.tag = t;
}

function addPlayer(p) {
  if (p.pid === G.pid || G.players.has(p.pid)) return;
  const mesh = makeAvatar(p);
  mesh.position.set(p.x, 0, p.z);
  mesh.rotation.y = p.ry;
  G.players.set(p.pid, { ...p, mesh, target: new T.Vector3(p.x, 0, p.z), mv: 0, phase: Math.random() * 6 });
  setGone(mesh, p.gone);
  V.setVehicle(mesh, p.vehicle);
  mesh.userData.seated = p.rideOf || null;
  mesh.userData.carrier = p.carrier ? { id: p.carrier, kind: G.actors.get(p.carrier)?.info.kind || 'bus', seat: p.carSeat } : null;
 mesh.userData.nest=p.nest||null;
  V.setPet(mesh, p.pet);
}
// Gà của người đang mất kết nối: mờ đi kèm biểu tượng sóng yếu, chờ nối lại.
function setGone(mesh, gone) {
  mesh.userData.gone = gone;
  mesh.traverse((o) => {
    if (!o.isMesh) return;
    if (gone && !o.userData.solidMat) { o.userData.solidMat = o.material; o.material = o.material.clone(); o.material.transparent = true; o.material.opacity = 0.4; }
    if (!gone && o.userData.solidMat) { o.material = o.userData.solidMat; delete o.userData.solidMat; }
  });
  bubble(mesh, gone ? '📶' : '👋', { emo: true, ttl: gone ? 999 : 0.8 }); // bong bóng mới thay bong bóng cũ cùng loại
}

function removePlayer(pid) {
  const p = G.players.get(pid);
  if (!p) return;
  V.dropPet(p.mesh);
  G.world.scene.remove(p.mesh);
  G.players.delete(pid);
}
function clearPlayers() { for (const pid of [...G.players.keys()]) removePlayer(pid); }

function meshOf(pid) { return pid === G.pid ? G.self?.mesh : G.players.get(pid)?.mesh; }

function setZone(zone, pos, ry, players) {
  G.zone = zone;
 G.walkPath=[];G.tapAction=null;G.routeAt=0;
  G.world.show(zone, pos, ry);
  clearPlayers();
  players.forEach(addPlayer);
  G.self.pos.set(pos.x, 0, pos.z);
  G.self.srv.copy(G.self.pos);
  G.self.mesh.rotation.y = ry;
  G.self.mesh.position.copy(G.self.pos);
  $('homeBtn').hidden = true;
  announceArea(zone);
}
// Walking across an area boundary on the seamless map: only the title, camera presets and hints change.
function announceArea(zone) {
  G.zone = zone;
  G.world.setArea(zone);
  $('currentZoneName').textContent = G.zones[zone].title;
  const el = $('areaName');
  el.textContent = G.zones[zone].title;
  el.style.opacity = '1';
  clearTimeout(el._t);
  el._t = setTimeout(() => (el.style.opacity = '0'), 2200);
  SO.arrivedLane();
}
// The ground the chicken walks on: the whole region map, or the single home/guild hall.
const walkZone = () => G.worldMap || G.zones[G.zone];
// Interaction points of every area (all in map coordinates).
const homeBeds = () => G.zone === 'home' && G.home && G.home.owner === G.me?.id ? (G.home.objects || []).filter((o) => o.kind === 'luong_rau' || o.kind === 'luong_ca_rot').map((o) => ({ kind: 'bed', id: 'home:' + o.id, label: 'Luống rau nhà', x: o.x, z: o.z, r: 1.6 })) : [];
const allItems = () => G.worldMap ? Object.values(G.zones).flatMap((z) => z.items || []) : [...(G.zones[G.zone]?.items || []), ...homeBeds()];

// ============ Tin nhắn từ máy chủ ============
const N = G.net;
N.on('open', () => { $('netDot').classList.remove('off'); });
N.on('close', () => {
  $('netDot').classList.add('off');
  if (!$('hud').hidden) toast('Mất kết nối, đang nối lại…', 'warn');
});
N.on('auth_fail', (m) => { store.set(TOKEN_KEY, null); N.stop(); showScreen('authScreen'); $('authErr').textContent = m.err; });
N.on('need_char', (m) => initCreator(m.catalog));
N.on('char_err', (m) => ($('charErr').textContent = m.err));
N.on('kicked', (m) => {
  sfx('lose');
  N.stop();
  $('hud').hidden = true;
  closeSheet();
  loading('Đã ngắt phiên này', `${esc(m.reason)}<br><button class="btn" id="takeBack" style="margin-top:14px">Chơi tiếp ở thiết bị này</button>`);
  $('takeBack').onclick = () => { loading('Đang vào lại…'); N.connect(store.get(TOKEN_KEY)); };
});

N.on('welcome', (m) => {
  G.abortTug?.(); // lượt đào/câu dở bị máy chủ huỷ khi đổi phòng hoặc nối lại
  SB.onBoard({ state: 'idle' }); // bảng điểm của phòng cũ không còn đúng; phòng mới gửi lại ngay sau welcome nếu có trận
  const first = !G.catalog;
  G.catalog = m.catalog;
  G.zones = m.zones;
  G.worldMap = m.world || null; // bản đồ liền của trại; null trong nhà riêng / Nhà Hội
  G.pid = m.pid;
  G.room = m.room;
  G.world.build(m.zones, m.catalog.cosmetics, m.world);
  setClock(m.clock);
  SO.stopPlacing();
  G.home = m.home || null;
  if (G.home) G.home._at = performance.now();
  if (m.home) G.world.setHome(m.zones.home, m.home, m.catalog.cosmetics);
  if (m.guildhall) G.world.setGuildHall(m.zones.guildhall, m.guildhall, m.catalog.cosmetics);
  if (G.self) G.world.scene.remove(G.self.mesh);
  const mesh = buildChicken({ color: m.me.char.color, comb: m.me.char.comb, tail: m.me.char.tail, eyes: m.me.char.eyes });
  G.world.scene.add(mesh);
  syncPortrait(m.me.char.color);
  G.self = { mesh, pos: new T.Vector3(), srv: new T.Vector3(), phase: 0 };
  applyMe(m.me);
  dress(mesh, m.me.equipped, m.me.char, G.catalog.cosmetics);
  setNameTag(mesh, tagText(m.me.char.name, m.me.level, titleName(m.me.title), m.self?.guild), true);
  G.region = m.region || 'hypex';
  G.bus = m.bus || {};
  G.busAt = performance.now();
  G.myGuild = m.self?.guild || '';
  G.myRide = m.self?.vehicle || null;
  G.myRideOf = m.self?.rideOf || null;
  NPC.onWelcome(m.actors);
  G.myCarrier = m.self?.carrier ? { id: m.self.carrier, kind: G.actors.get(m.self.carrier)?.info.kind || 'bus' } : null;
  G.chatLog = P.appendChat;
  V.clearPets();
  V.setVehicle(mesh, G.myRide);
  mesh.userData.seated = G.myRideOf;
 mesh.userData.nest=m.self?.nest||null;
  V.setPet(mesh, m.self?.pet);
  G.festival = m.festival;
  G.world.setFestival(m.festival?.unlocked);
  setZone(m.zone, m.pos, m.ry, m.players);
  updateRoom();
  G.net.raw({ t: 'social' });
  G.match = null;
  closeSheet();
  showScreen(null);
  $('hud').hidden = false;
  if (first) {
    P.appendChat({ kind: 'system', text: `Chào mừng ${m.me.char.name} tới ${m.room.name}!` });
    if (!store.get('hypex_help_seen')) { store.set('hypex_help_seen', '1'); setTimeout(P.openHelp, 400); }
  } else if (G.net.dropped) {
    // Chỉ báo sau khi nối lại mạng thật, không báo mỗi lần đổi phòng (B13).
    toast('Đã kết nối lại — bạn ở sảnh ' + m.room.name, 'good');
  }
  G.net.dropped = false;
});

N.on('nest',m=>{const mesh=meshOf(m.pid);if(mesh){mesh.userData.nest=m.nest||null;if(m.nest){mesh.userData.anim=null;if(m.pid===G.pid){G.walkPath=[];G.self.pos.set(m.x,0,m.z);G.self.srv.copy(G.self.pos);}}}});
N.on('zone', (m) => { setZone(m.zone, m.pos, m.ry, m.players); sfx('door'); });
N.on('maintenance', (m) => { N.stop(); loading('Trại đang bảo trì', `${esc(m.text || 'Quay lại sau ít phút nhé.')}<br><button class="btn" id="retryMaint" style="margin-top:14px">Thử lại</button>`); $('retryMaint').onclick = () => { loading('Đang mở cổng trại…'); N.connect(store.get(TOKEN_KEY)); }; });
N.on('pgone', (m) => { const p = G.players.get(m.pid); if (p) { p.gone = m.gone; setGone(p.mesh, m.gone); } });
N.on('pk_resume', P.PK.resume);
N.on('me', (m) => applyMe(m.me));
N.on('bus', V.onBus);
N.on('trip', V.onTrip);
N.on('seat_invite', V.onSeatInvite);
N.on('trade_invite', V.onTradeInvite);
N.on('trade', V.onTrade);
N.on('trade_end', V.onTradeEnd);
N.on('contracts', V.onContracts);
N.on('guild', V.onGuild);
N.on('guild_list', V.onGuildList);
N.on('guild_chat', V.onGuildChat);
N.on('guild_invite', V.onGuildInvite);
N.on('pveh', (m) => {
  const mesh = meshOf(m.pid);
  if (!mesh) return;
  V.setVehicle(mesh, m.vehicle);
  if (m.pid === G.pid) G.myRide = m.vehicle || null;
  // Khách trên ghế: đánh dấu tư thế ngồi.
  (m.seats || []).forEach((pid, i) => { const q = meshOf(pid); if (q && i > 0) q.userData.seated = m.pid; });
});
N.on('pride', (m) => {
  const mesh = meshOf(m.pid);
  if (mesh) mesh.userData.seated = m.rideOf || null;
  if (m.pid === G.pid) G.myRideOf = m.rideOf || null;
});
N.on('mail_new', (m) => { $('mailDot').hidden = !m.unread; $('mailDot').textContent = m.unread > 9 ? '9+' : m.unread; sfx('notify'); toast('📮 Có thư Chim Chíp mới'); });
N.on('mail', SO.onMail);
N.on('festival', (m) => { G.festival = m.festival; G.world.setFestival(m.festival.unlocked); if (G.modal === 'quests') P.openQuests('festival'); });
N.on('room', (m) => { G.room = m.room; updateRoom(); });
N.on('rooms', (m) => P.openRooms(m.rooms));
N.on('pjoin', (m) => addPlayer(m.p));
N.on('pleave', (m) => removePlayer(m.pid));
N.on('plook', (m) => {
  if (m.pid === G.pid) {
    if (G.modal !== 'fashion') dress(G.self.mesh, m.equipped, G.me.char, G.catalog.cosmetics);
    V.setPet(G.self.mesh, m.pet);
    if (m.guild !== undefined && m.guild !== G.myGuild) { G.myGuild = m.guild; setNameTag(G.self.mesh, tagText(G.me.char.name, G.me.level, titleName(G.me.title), G.myGuild), true); }
    return;
  }
  const p = G.players.get(m.pid);
  if (p) {
    p.equipped = m.equipped; dress(p.mesh, m.equipped, p.char, G.catalog.cosmetics);
    if (m.title !== undefined && (m.title !== p.title || m.level !== p.level || m.guild !== p.guild)) { p.title = m.title; p.level = m.level; p.guild = m.guild; setNameTag(p.mesh, tagText(p.name, p.level, p.title, p.guild), false); }
    V.setPet(p.mesh, m.pet);
  }
});
N.on('pstage', (m) => { const p = G.players.get(m.pid); if (p) p.stage = m.stage; });
N.on('snap', (m) => {
  if (m.a) NPC.onSnap(m.a);
  m.pids.forEach((pid, i) => {
    const [x, z, ry, mv] = m.p[i];
    if (pid === G.pid) { G.self.srv.set(x, 0, z); return; }
    const p = G.players.get(pid);
    if (p) { p.target.set(x, 0, z); p.ry = ry; p.mv = mv; }
  });
});
N.on('vit', (m) => {
  if (!G.me) return;
  const prev = G.me.stage;
  G.me.mana = m.mana; G.me.stage = m.stage; G.me.zeroElapsed = m.zero;
  updateHUD(G.me);
  if (m.stage !== prev) hungerToast(prev, m.stage);
});
N.on('toast', (m) => toast(m.text, m.kind));
// Server đang giữ gà ở cổng khu đầy: trong lúc còn hiệu lực, dự đoán cục bộ không bước vào khu đó.
N.on('area_full', (m) => { G.areaBlock = { area: m.area, until: performance.now() + 700 }; });
// Thể lực chạy nhanh (PLAN_LOGIC_GAME 3): máy chủ tính, client hiện vòng quanh nút 🏃.
N.on('stam', (m) => { G.stam = m; updateSprint(); });
// Làng sống (PLAN_LOGIC_GAME 5–9).
N.on('aj', NPC.onJoin); N.on('al', NPC.onLeave); N.on('say', NPC.onSay); N.on('mark', NPC.onMark);
N.on('offer', NPC.onOffer); N.on('offer_end', NPC.onOfferEnd); N.on('npcask', NPC.onAsk);
N.on('scene', NPC.onScene); N.on('scene_end', NPC.onSceneEnd); N.on('pcar', NPC.onCarrier);
N.on('reward', (m) => P.showLevelReward?.(m.level, m.reward));
N.on('chat', (m) => {
  P.appendChat(m);
  if (m.kind === 'chat' && (G.worldMap ? meshOf(m.pid)?.position.distanceTo(G.self.pos) < 30 : m.zone === G.zone)) bubble(meshOf(m.pid), m.text);
});
N.on('emote', (m) => playEmote(meshOf(m.pid), m.e));
N.on('fx', (m) => fx(m.fx, meshOf(m.pid), m));
N.on('social', (m) => SO.onSocial(m.social));
N.on('dm', (m) => { if (m.msg.from !== G.me?.id) sfx('notify'); SO.onDM(m); });
N.on('dm_history', SO.onDMHistory);
N.on('homes', SO.onHomes);
N.on('room_invite', SO.onRoomInvite);
N.on('home_invite', SO.onHomeInvite);
N.on('home', (m) => {
  G.home = m.home;
  if (G.home) G.home._at = performance.now();
  G.zones.home = m.zone;
  G.world.setHome(m.zone, m.home, G.catalog.cosmetics);
  if (G.modal === 'cow') SO.dialogCow();
  if (G.modal === 'decorate') SO.openDecorate();
});
N.on('clock', (m) => setClock(m.clock));
N.on('showcase', (m) => { G.showLots = m.lots; G.world.setShowcase(m.lots); });
N.on('fbeds', (m) => { G.friendBeds = m.beds; });
// Chip 🎫 lượt xe miễn phí (PLAN_LOGIC_GAME 7.2): tự đếm lượt hồi giữa hai lần máy chủ gửi "me".
setInterval(() => {
  const me = G.me, el = $('ticketChip');
  if (!me || !el || !G.catalog || me.busTokens == null) return;
  const max = G.catalog.maxBusTokens || 2, every = G.catalog.balance.tokenEverySec || 300;
  let n = me.busTokens, left = (me.nextTokenIn || 0) - (performance.now() - (me._at || 0)) / 1000;
  if (n < max && me.nextTokenIn > 0 && left <= 0) { n = Math.min(max, n + 1 + Math.floor(-left / every)); left = every - (-left % every); }
  el.hidden = !G.worldMap;
  el.textContent = `🎫 ${n}/${max}` + (n < max ? ` · ${Math.floor(left / 60)}:${String(Math.floor(left % 60)).padStart(2, '0')}` : '');
}, 1000);
// Ngày/đêm và mưa (PLAN_LOGIC_GAME 16): máy chủ báo, chip trên ví nhắc luật đang có tác dụng.
function setClock(c) {
  if (!c) return;
  G.clock = c;
  G.world.setClock(c);
  const chip = $('skyChip');
  chip.textContent = (c.night ? '🌙' : '☀️') + (c.rain ? ' 🌧' : '');
  chip.title = [c.night ? 'Đêm: đèn lồng thắp, Giun phát sáng ngoi lên nhiều hơn' : 'Ban ngày', c.rain ? 'Mưa nhẹ: ụ đất hồi nhanh hơn 30%, xe ôm phụ thu 1 thóc' : ''].filter(Boolean).join(' · ');
}
N.on('pk_invite', P.showInvite);
N.on('pk_invite_gone', (m) => P.hideInvite(m.id));
N.on('pk_start', P.PK.start);
N.on('pk_go', P.PK.go);
N.on('pk_score', P.PK.score);
N.on('pk_round', P.PK.round);
N.on('pk_flip', P.PK.flip);
N.on('pk_end', P.PK.end);
N.on('pk_board', SB.onBoard); // bảng điểm 3D ở Sân tỷ thí, cả phòng cùng thấy

function hungerToast(prev, st) {
  if (st === 1 && prev === 0) toast('Sắp đói! Mở túi hoặc ghé Quầy thức ăn.', 'warn');
  if (st === 2) toast('Gà đói rồi! Ăn ngay kẻo bị phạt.', 'warn');
  if (st === 3) toast('Đói kéo dài: đi chậm 15%, đào chậm hơn.', 'bad');
  if (st === 4) toast('Kiệt sức! Không đào hay nhận PK được. Hãy ăn để mana đạt 20.', 'bad');
  if (st === 0 && prev >= 2) toast('No rồi! Đã gỡ hình phạt đói.', 'good');
}

// ============ HUD ============
let digReadyAt = 0;
function updateHUD(me) {
  $('lvTxt').textContent = 'Cấp ' + me.level + (me.level >= G.catalog.levelCap ? ' · tối đa' : '');
  $('thocTxt').textContent = me.thoc.toLocaleString('vi-VN');
  $('xpFill').style.width = (me.xpNeed ? (me.xp / me.xpNeed) * 100 : 100) + '%';
  $('xpLab').textContent = me.xpNeed ? `${me.xp}/${me.xpNeed}` : 'MAX';
  const mf = $('manaFill');
  mf.style.width = me.mana + '%';
  mf.style.background = me.stage === 0 ? 'var(--mana)' : me.mana > 0 ? 'var(--warn)' : 'var(--bad)';
  const b = G.catalog.balance;
  let lab = `${Math.ceil(me.mana)}/100`;
  if (me.stage === 2) lab += ` · phạt sau ${Math.max(0, Math.ceil(b.hungerSlowSec - me.zeroElapsed))}s`;
  else if (me.stage === 3) lab += ` · kiệt sức sau ${Math.max(0, Math.ceil(b.hungerExhaustSec - me.zeroElapsed))}s`;
  else if (me.stage) lab += ' · ' + STAGE_NAME[me.stage];
  $('manaLab').textContent = lab;
  $('manaBar').classList.toggle('hungry', me.stage >= 2);
  const tutPending = !me.tutorialClaimed && G.catalog.tutorial.every((s) => me.tutorial[s.id]);
  $('questDot').hidden = !(tutPending || (me.daily || []).some((q) => !q.done && q.cur > 0));
}
let lastLevel = null, lastTitle = null;
const titleName = (id) => (G.catalog?.titles || []).find((t) => t.id === id)?.name || '';
onMe((me) => {
  digReadyAt = performance.now() + me.digReadyIn * 1000;
  updateHUD(me);
  if (lastLevel != null && me.level > lastLevel && G.self) {
    fx('levelup', G.self.mesh);
    toast(`🎉 Lên cấp ${me.level}!`, 'good');
  }
  if (G.self && (me.level !== lastLevel || me.title !== lastTitle)) setNameTag(G.self.mesh, tagText(me.char.name, me.level, titleName(me.title), G.myGuild), true);
  lastTitle = me.title;
  lastLevel = me.level;
});
function updateRoom() { $('roomTxt').textContent = `${G.room.name} · ${G.room.count} người trong kênh`; $('playerName').textContent=G.me?.char?.name||'Gà con'; }

// ============ Input ============
const keys = {};
window.addEventListener('keydown', (e) => {
  if (['INPUT','TEXTAREA'].includes(e.target.tagName)||e.target.isContentEditable) return;
  const k = e.key.toLowerCase();
  if(k==='m'&&!e.repeat&&!$('hud').hidden){e.preventDefault();if(G.modal==='map')closeSheet();else MAP.openMap();return;}
  if (G.modal) return;
  keys[k] = true;
  if (G.placing && k === 'r') { SO.rotatePlacing(); return; }
  if (G.placing && (k === 'e' || k === ' ')) { e.preventDefault(); if (!e.repeat) SO.commitPlacing(); return; }
  if (k === 'x' && NPC.hasOffer()) { e.preventDefault(); NPC.declineOffer(); return; }
  if (k === 'e' && NPC.hasOffer()) { e.preventDefault(); if (!e.repeat) NPC.acceptOffer(); return; }
  if (k === 'e' || k === ' ') { e.preventDefault(); if (!e.repeat) interact(); }
  if (k === 'enter' && !$('hud').hidden) { e.preventDefault(); P.openChat(); }
});
window.addEventListener('keyup', (e) => { keys[e.key.toLowerCase()] = false; });
window.addEventListener('blur', () => { for (const k in keys) keys[k] = false; });

const joy = { x: 0, y: 0, id: null };
const joyEl = $('joy'), nub = $('joyNub');
// Nút 🏃: giữ để chạy (cảm ứng và chuột); Shift trên bàn phím làm cùng việc.
{
  const sb = $('sprintBtn');
  const set = (v) => (e) => { e.preventDefault(); G.runHeld = v; if (v) sb.setPointerCapture?.(e.pointerId); };
  sb.addEventListener('pointerdown', set(true)); sb.addEventListener('pointerup', set(false)); sb.addEventListener('pointercancel', set(false));
}
joyEl.addEventListener('pointerdown', (e) => { joy.id = e.pointerId; joyEl.setPointerCapture(e.pointerId); moveJoy(e); });
joyEl.addEventListener('pointermove', moveJoy);
const endJoy = () => { joy.id = null; joy.x = joy.y = 0; nub.style.transform = ''; };
joyEl.addEventListener('pointerup', endJoy);
joyEl.addEventListener('pointercancel', endJoy);
function moveJoy(e) {
  if (e.pointerId !== joy.id) return;
  const r = joyEl.getBoundingClientRect();
  let dx = e.clientX - (r.left + r.width / 2), dy = e.clientY - (r.top + r.height / 2);
  const d = Math.hypot(dx, dy), max = 44;
  if (d > max) { dx = (dx / d) * max; dy = (dy / d) * max; }
  nub.style.transform = `translate(${dx}px,${dy}px)`;
  joy.x = dx / max; joy.y = dy / max;
}

const canvas = G.world.renderer.domElement;
const raycaster=new T.Raycaster(), pointer=new T.Vector2();
bindWorldInput(canvas,{camera:G.world.controls,enabled:()=>!!G.self&&!G.modal&&!G.busy&&!G.myRideOf&&!G.placing,tap:(x,y)=>{
 pointer.set(x/window.innerWidth*2-1,-y/window.innerHeight*2+1);raycaster.setFromCamera(pointer,G.world.camera);
 const point=new T.Vector3();if(!raycaster.ray.intersectPlane(new T.Plane(new T.Vector3(0,1,0),0),point))return;
 const zone=walkZone();let dest={x:point.x,z:point.z};G.tapAction=null;
 let it=allItems().filter(i=>Math.hypot(i.x-point.x,i.z-point.z)<(i.kind==='nest'?1.2:1.8)).sort((a,b)=>Math.hypot(a.x-point.x,a.z-point.z)-Math.hypot(b.x-point.x,b.z-point.z))[0];
 if(it){if(Math.hypot(G.self.pos.x-it.x,G.self.pos.z-it.z)<it.r){nearest=it;interact();return;}
  let best=null;for(let a=0;a<Math.PI*2;a+=Math.PI/8){const p={x:it.x+Math.cos(a)*Math.max(.6,it.r-.45),z:it.z+Math.sin(a)*Math.max(.6,it.r-.45)};if(walkable(zone,p.x,p.z)&&(!best||Math.hypot(p.x-G.self.pos.x,p.z-G.self.pos.z)<best.d)){best={...p,d:Math.hypot(p.x-G.self.pos.x,p.z-G.self.pos.z)};}}
  if(best){dest=best;G.tapAction=it;}
 }
 G.walkPath=findPath(zone,G.self.pos,dest);if(G.walkPath.length)G.world.markDestination(dest);else toast('Chọn một điểm trên lối đi.');
}});
$('resetCamera').onclick=()=>G.world.controls.reset();
$('socialCamera').onclick=()=>G.world.controls.toggleSocial();
$('toggleHUD').onclick=()=>document.body.classList.toggle('quiet-hud');
$('btnMore').onclick=()=>document.body.classList.toggle('tools-expanded');
$('quickChat').onsubmit=e=>{e.preventDefault();const input=$('quickChatInput');if(input.value.trim()){G.net.raw({t:'chat',text:input.value.trim()});input.value='';input.blur();}};

$('interact').onclick = () => interact();
$('homeBtn').onclick = async () => { const r = await G.net.act('go', { to: 'lobby', id: 'home' }); if (!r.ok) toast(r.err, 'warn'); };
$('roomChip').onclick = () => P.openRooms();
$('btnQuest').onclick = P.openQuests;
$('btnBag').onclick = () => P.openBag();
$('btnBook').onclick = P.openBook;
$('btnWard').onclick = P.openWardrobe;
// Cửa hàng: ở xa thì chỉ đường ra Chợ thóc; mua/bán vẫn phải đứng gần quầy tương ứng.
$('btnShop').onclick = () => {
  if (!G.me) return;
  if (G.zone === 'market') { toast('Bạn đang ở Chợ thóc — ghé các quầy để mua bán.'); return; }
  if (!G.worldMap) { toast('Ra khỏi chuồng rồi mới đi chợ được.', 'warn'); return; }
  MAP.setDestination('market', 'Đã ghim Chợ thóc — đi theo dấu chỉ đường.');
};
$('btnChat').onclick = P.openChat;
$('btnHelp').onclick = P.openHelp;
$('btnFriends').onclick = () => SO.openFriends();
$('btnSettings').onclick = openSettings;
$('btnSound').onclick = () => { settings.muted = !settings.muted; applySettings(); syncSoundIcon(); toast(settings.muted ? 'Đã tắt âm thanh' : 'Đã bật âm thanh'); };
$('btnMail').onclick = () => { $('mailDot').hidden = true; SO.openMail(); };
$('btnGuild').onclick = V.openGuild;
// Chụp ảnh cảnh hiện tại (mục 11: bữa tiệc sân nhà và chụp ảnh).
$('btnPhoto').onclick = () => {
  const w = G.world;
  w.renderer.render(w.scene, w.camera);
  const url = w.renderer.domElement.toDataURL('image/png');
  sfx('door');
  openSheet('photo', '📸 Ảnh chụp', `<img src="${url}" alt="Ảnh chụp cảnh trong game" style="width:100%;border-radius:12px;border:2px solid var(--line)">
    <a class="btn block" href="${url}" download="hypex-${Date.now()}.png" style="display:block;text-align:center;text-decoration:none">Lưu ảnh</a>`);
};
document.addEventListener('click', (e) => { if (e.target.closest('button')) sfx('click'); }, true);

// ============ Cài đặt (lưu trên thiết bị) ============
const settings = (() => { try { return JSON.parse(store.get('hypex_settings')) || {}; } catch { return {}; } })();
settings.quality ||= matchMedia('(pointer:coarse)').matches ? 'low' : 'mid';
settings.reduced ??= matchMedia('(prefers-reduced-motion: reduce)').matches;
settings.volume ??= 0.6;
settings.muted ??= false;
function applySettings() {
  G.world.setQuality(settings.quality);
  G.world.reducedMotion = G.reducedMotion = settings.reduced;
  setAudio(settings.volume, settings.muted);
  store.set('hypex_settings', JSON.stringify(settings));
}
function syncSoundIcon() {
  const use = $('btnSound')?.querySelector('use');
  if (use) use.setAttribute('href', settings.muted ? '#i-sound-off' : '#i-sound');
}
applySettings();
syncSoundIcon();
function openSettings() {
  const q = [['low', 'Thấp', 'Không bóng đổ, ưu tiên điện thoại'], ['mid', 'Vừa', 'Cân bằng'], ['high', 'Cao', 'Bóng đổ nét, máy mạnh']];
  openSheet('settings', 'Cài đặt', `
    <h2 style="margin-top:0">Đồ hoạ</h2>
    <div class="tabs">${q.map(([k, l]) => `<button class="tab" data-q="${k}" aria-selected="${settings.quality === k}" title="${l}">${l}</button>`).join('')}</div>
    <p class="mut">${q.find((x) => x[0] === settings.quality)[2]}.</p>
    <label class="row" style="cursor:pointer"><input type="checkbox" id="setReduced" ${settings.reduced ? 'checked' : ''}><span class="meta"><span class="nm">Giảm chuyển động</span><span class="sub" style="display:block">Camera không trôi, NPC và gà không nhún, tắt hiệu ứng chuyển cảnh</span></span></label>
    <h2>Âm thanh</h2>
    <div class="row"><div class="meta"><label class="nm" for="setVol">Âm lượng</label><input type="range" id="setVol" min="0" max="1" step="0.05" value="${settings.volume}" style="width:100%;accent-color:var(--accent)"></div>
      <label class="act" style="gap:6px;align-items:center;display:flex"><input type="checkbox" id="setMute" ${settings.muted ? 'checked' : ''} style="width:22px;height:22px;accent-color:var(--accent)">Tắt tiếng</label></div>
    ${L.prefsSection()}
    <h2>Riêng tư</h2>
    <button class="choice" id="setPrivacy" style="width:100%">👥 Hiện vị trí với bạn bè, nhận lời mời PK, danh sách đã chặn…</button>
    <h2>Điều khiển</h2>
    <p class="mut" style="margin:0"><kbd>W</kbd><kbd>A</kbd><kbd>S</kbd><kbd>D</kbd>/mũi tên: đi · click mặt đất: đi tới đó · kéo chuột: xoay ngang và ngẩng/cúi camera · lăn chuột: zoom · <kbd>M</kbd>: bản đồ · <kbd>E</kbd>: tương tác · <kbd>Enter</kbd>: chat · <kbd>R</kbd>: xoay đồ khi trang trí. Điện thoại: joystick trái, chạm đất để đi, kéo cảnh để xoay, hai ngón để zoom, nút Tương tác.</p>
    <h2>Hỗ trợ</h2>
    <button class="choice" id="safeReturn" style="width:100%">🛟 Về sảnh an toàn — khi bị kẹt hoặc đói lả, không tốn vé</button>
    <h2>Tài khoản</h2>
    <button class="btn sec block" id="logoutBtn" style="margin-top:0">Đăng xuất</button>`);
  document.querySelectorAll('[data-q]').forEach((b) => (b.onclick = () => { settings.quality = b.dataset.q; applySettings(); openSettings(); }));
  $('setReduced').onchange = (e) => { settings.reduced = e.target.checked; document.documentElement.classList.toggle('reduce', settings.reduced); applySettings(); };
  L.bindPrefs();
  $('setPrivacy').onclick = () => SO.openFriends('settings');
  $('safeReturn').onclick = () => SO.confirmBox('Về sảnh an toàn?', 'Gà sẽ về cửa sảnh chuồng cộng đồng. Chỉ dùng khi bị kẹt hoặc cần cứu đói; đi lại thường ngày hãy dùng bản đồ.', async () => { const r = await G.net.act('go', { to: 'lobby', id: 'home' }); if (!r.ok) toast(r.err, 'warn'); });
  $('setVol').oninput = (e) => { settings.volume = +e.target.value; applySettings(); };
  $('setVol').onchange = () => sfx('coin');
  $('setMute').onchange = (e) => { settings.muted = e.target.checked; applySettings(); };
  $('logoutBtn').onclick = () => { store.set(TOKEN_KEY, null); G.net.stop(); closeSheet(); $('hud').hidden = true; showScreen('authScreen'); };
}
document.documentElement.classList.toggle('reduce', !!settings.reduced);

const EMOTES = [['chao', '👋', 'Chào'], ['vui', '😄', 'Vui'], ['buon', '😢', 'Buồn'], ['ngac_nhien', '😮', 'Ngạc nhiên'], ['mua', '💃', 'Múa'], ['gay', '📣', 'Gáy'], ['an_mung', '🎉', 'Ăn mừng'], ['ngoi', '🪑', 'Ngồi'], ['ngu', '💤', 'Ngủ']];
$('emotes').innerHTML = EMOTES.map(([id, e, n]) => `<button class="iconbtn" data-emote="${id}" title="${n}" aria-label="${n}">${e}</button>`).join('');
$('emotes').querySelectorAll('[data-emote]').forEach((b) => (b.onclick = () => G.net.raw({ t: 'emote', e: b.dataset.emote })));

// ============ Tương tác ============
let nearest = null;
function findNearest() {
  if (!G.zones || G.modal) return null;
  let best = null, bd = 1e9;
  for (const it of allItems()) {
    const d = Math.hypot(G.self.pos.x - it.x, G.self.pos.z - it.z);
    if (d < it.r && d < bd) { bd = d; best = it; }
  }
  return best;
}
function partyPrompt() {
  const h = G.home;
  if (!h || G.zone !== 'home' || !(homeLeft(h, 'partyLeft') > 0)) return false;
  return h.objects.some((o) => o.kind === 'ban_tiec' && Math.hypot(G.self.pos.x - o.x, G.self.pos.z - o.z) < 2);
}

function promptText(it) {
  if (it.kind === 'dig') {
    const left = (digReadyAt - performance.now()) / 1000;
    return left > 0 ? `Đất đang hồi (${Math.ceil(left)}s)` : `Đào giun · ${it.label}`;
  }
  if (it.kind === 'nest') return G.self.mesh.userData.nest===it.id?'Đứng dậy':'Ngồi ổ rơm';
  if (it.kind === 'npc_guide') return 'Gặp Gà Đón Khách';
  if (it.kind === 'gate') return it.label;
  if (it.kind === 'pk') return 'Đấu PK';
  if (it.kind === 'trough') return 'Máng ăn';
  if (it.kind === 'home_exit') return 'Ra Xóm Rơm Vàng';
  if (it.kind === 'home_sign') return 'Chuồng riêng & thăm bạn';
  if (it.kind === 'home_board') return 'Bảng nhà';
  if (it.kind === 'bus_stop') return 'Trạm xe buýt';
  if (it.kind === 'guild_board') return 'Bảng hội';
  if (it.kind === 'parking') return G.myRide ? 'Cất xe' : 'Điểm đỗ xe';
  if (it.kind === 'road') return 'Đường liên trại';
  if (it.kind === 'bed') { const pl = G.me?.beds?.[it.id.startsWith('home:') ? it.id : 'park:' + it.id]; return pl ? (srvNow() >= Date.parse(pl.ready) ? '🧺 Thu hoạch' : '🌱 Chăm cây') : '🌱 Gieo hạt · ' + it.label; }
  if (it.kind === 'npc_cow' && G.zone !== 'home') return 'Tiệm vật liệu · Bò Mộc';
  if (it.kind === 'fish') { const left = (G.me?.fishReadyIn || 0) - (performance.now() - (G.me?._at || 0)) / 1000; return left > 0 ? `Cá còn lặn (${Math.ceil(left)}s)` : '🎣 Thả câu'; }
  if (it.kind === 'showcase_lot') { const lot = G.showLots?.[+it.id.split('_')[1] - 1]; return lot ? (lot.mine ? '🏠 Nhà của bạn' : `🚪 Thăm nhà ${lot.name}`) : '🏡 Nhà mẫu · Chủ Đầu Tư'; }
  return 'Gặp ' + it.label;
}
function interact() {
  if (G.modal) return;
  const work = V.workPrompt();
  if (work) { V.doWork(work); return; }
  if (partyPrompt()) { G.net.act('party_cheer').then((r) => !r.ok && toast(r.err, 'warn')); return; }
  if (!nearest) { if (NPC.nearestStop()) NPC.openBusStop(); return; }
  const it = nearest;
  switch (it.kind) {
    case 'nest': G.net.act(G.self.mesh.userData.nest?'stand':'sit',{id:it.id}).then(r=>!r.ok&&toast(r.err,'warn'));break;
    case 'npc_guide': openSheet('guide','Chào mừng về làng',`<p>Cụ Cục đang ở trên bục cuối sảnh. Chọn một ổ rơm để ngồi cùng mọi người.</p><p>Kéo để xoay góc nhìn, lăn chuột để zoom. Chạm xuống đất hoặc dùng WASD để đi.</p><p>Qua cửa chính là sân làng. Mở bản đồ để tìm đường tới vườn giun, chợ và bến xe.</p><button class="btn block" id="guideMap">Mở bản đồ làng ↗</button>`);$('guideMap').onclick=()=>MAP.openMap();break;
    case 'gate': G.net.act('go', { to: it.to }).then((r) => !r.ok && toast(r.err, 'warn')); break;
    case 'dig': D.startDig(it); break;
    case 'npc_elder': P.dialogElder(it); break;
    case 'npc_pig': P.dialogPig(it); break;
    case 'board': P.openQuests(); break;
    case 'trough': toast('Máng ăn chung để trang trí — hồi mana cần thức ăn đã mua.', ''); P.openBag('food'); break;
    case 'shop_sell': P.openSell(); break;
    case 'shop_food': P.openFood(); break;
    case 'shop_fashion': P.openFashion(); break;
    case 'pk': P.dialogPK(it); break;
    case 'home_sign': SO.dialogHomeSign(); break;
    case 'home_exit': G.net.act('go', { to: 'lobby' }); break;
    case 'npc_cow': SO.dialogCow(); break;
    case 'npc_investor': L.dialogInvestor(); break;
    case 'bed': L.dialogBed(it); break;
    case 'fish': F.startFish(it); break;
    case 'showcase_lot': L.dialogLot(it); break;
    case 'home_board': SO.openHomeBoard(); break;
    case 'npc_duck': V.dialogDuck(); break;
    case 'guild_board': V.openGuild(); break;
    case 'npc_bird': V.dialogBird(); break;
    case 'npc_goat': V.dialogGoat(); break;
    case 'npc_frog': V.dialogFrog(); break;
    case 'bus_stop': V.openBus(); break;
    case 'parking': V.parkingMenu(); break;
    case 'road': V.roadMenu(); break;
  }
}

// ============ Hiệu ứng & biểu cảm ============
const effects = [];
function fx(kind, mesh, opt = {}) {
  if (!mesh) return;
  const a = (k, dur) => (mesh.userData.anim = { k, t: 0, dur });
  switch (kind) {
    case 'dig': a('dig', opt.dur || (opt.peck ? opt.peck / 1000 : 0.9)); dirt(mesh); break;
    case 'escape': dirt(mesh); bubble(mesh, '💨', { emo: true, ttl: 1.2 }); break;
    case 'goose': bubble(mesh, '🪿 Cưỡi ngỗng!', { ttl: 2.5 }); a('hop', 0.8); break;
    case 'buddy': bubble(mesh, '💞', { emo: true, ttl: 1.6 }); break;
    case 'cast': bubble(mesh, '🎣', { emo: true, ttl: 1.4 }); break;
    case 'fish_catch': a('hop', 0.7); bubble(mesh, opt.tier >= 2 ? '🐟✨' : '🐟', { emo: true, ttl: 1.8 }); break;
    case 'fish_end': bubble(mesh, '💨', { emo: true, ttl: 1 }); break;
    case 'dug': {
      a('hop', 0.6); dirt(mesh);
      const w = wormMesh(opt.color || '#d98a8a');
      w.position.set(0, 1.5, 0.3); mesh.add(w);
      effects.push({ obj: w, parent: mesh, t: 0, dur: 1.2, update: (e) => { w.position.y = 1.5 + e.t * 0.6; w.rotation.y += 0.15; } });
      break;
    }
    case 'show': {
      a('crow', 2.4);
      const w = wormMesh(opt.color);
      w.position.set(0.35, 1.9, 0.25); w.scale.setScalar(1.6); mesh.add(w);
      const light = new T.PointLight(parseInt(String(opt.color).slice(1), 16), 2.2, 5);
      light.position.set(0, 2.3, 0); mesh.add(light);
      effects.push({ obj: w, parent: mesh, t: 0, dur: 3, update: (e) => { w.rotation.z = Math.sin(e.t * 8) * 0.4; } });
      effects.push({ obj: light, parent: mesh, t: 0, dur: 3, update: (e) => { light.intensity = 2.2 + Math.sin(e.t * 10) * 0.8; } });
      bubble(mesh, '✨', { emo: true, ttl: 2.5 });
      break;
    }
    case 'levelup': stars(mesh, 0xffe066); a('hop', 0.8); bubble(mesh, '⭐', { emo: true }); if (mesh === G.self?.mesh) sfx('level'); break;
    case 'win': a('cheer', 1.8); bubble(mesh, '🏆', { emo: true }); break; // đấu sĩ thắng dang cánh (PLAN_LOGIC_GAME 9.3)
    case 'tie': bubble(mesh, '🤝', { emo: true }); break;
    case 'party': confetti(mesh); a('party', 1.6); if (mesh === G.self?.mesh) sfx('win'); break;
    case 'ko': a('ko', 2.2); bubble(mesh, '💫', { emo: true, ttl: 2.2 }); koStars(mesh); break;
    case 'eat': a('eat', 1.0); if (mesh === G.self?.mesh) sfx('eat'); break;
  }
}
G.fx = fx;
function playEmote(mesh, e) {
  if (!mesh) return;
  const map = { chao: ['wave', '👋', 1.2], vui: ['hop', '😄', 1], buon: ['droop', '😢', 1.6], ngac_nhien: ['jolt', '😮', 0.6], mua: ['spin', '💃', 1.6], gay: ['crow', '📣', 1.4], an_mung: ['party', '🎉', 1.6], ngoi: ['sit', '🪑', 6], ngu: ['sleep', '💤', 8] };
  const [k, emo, dur] = map[e] || [];
  if (!k) return;
  mesh.userData.anim = { k, t: 0, dur };
  bubble(mesh, emo, { emo: true, ttl: dur + 0.6 });
  if (e === 'gay') { bubble(mesh, 'Ò ó o ooo!', { ttl: 1.6 }); if (mesh === G.self?.mesh || G.players.size < 8) sfx('crow'); }
}
function dirt(mesh) {
  for (let i = 0; i < 10; i++) {
    const c = new T.Mesh(new T.BoxGeometry(0.08, 0.08, 0.08), mat(0x6e4a28));
    const v = new T.Vector3((Math.random() - 0.5) * 3, 2 + Math.random() * 2, 1 + Math.random() * 1.5);
    c.position.set(mesh.position.x, 0.3, mesh.position.z);
    G.world.scene.add(c);
    effects.push({ obj: c, parent: G.world.scene, t: 0, dur: 0.8, update: (e, dt) => { v.y -= 9 * dt; c.position.addScaledVector(v, dt); } });
  }
}
function confetti(mesh) {
  const cols = [0xd64b3a, 0xe7c23a, 0x3f86c9, 0x4e9a4e, 0xe2577a];
  for (let i = 0; i < 24; i++) {
    const c = new T.Mesh(new T.BoxGeometry(0.08, 0.02, 0.12), mat(cols[i % cols.length]));
    const v = new T.Vector3((Math.random() - 0.5) * 4, 4 + Math.random() * 3, (Math.random() - 0.5) * 4);
    c.position.set(mesh.position.x, 1.6, mesh.position.z);
    G.world.scene.add(c);
    effects.push({ obj: c, parent: G.world.scene, t: 0, dur: 1.6, update: (e, dt) => { v.y -= 7 * dt; c.position.addScaledVector(v, dt); c.rotation.x += dt * 8; } });
  }
}
// Bụi sau chân khi chạy nhanh.
function dust(mesh) {
  const c = new T.Mesh(new T.SphereGeometry(0.12, 6, 4), mat(0xcdb48a, { transparent: true, opacity: 0.6 }));
  c.position.set(mesh.position.x - Math.sin(mesh.rotation.y) * 0.4, 0.12, mesh.position.z - Math.cos(mesh.rotation.y) * 0.4);
  G.world.scene.add(c);
  effects.push({ obj: c, parent: G.world.scene, t: 0, dur: 0.5, update: (e) => { c.scale.setScalar(1 + e.t * 3); c.material.opacity = 0.6 * (1 - e.t / 0.5); } });
}
// Nút 🏃 và vòng thể lực (máy chủ gửi tỉ lệ 0..1 và số giây khoá khi hụt hơi).
function updateSprint() {
  const st = G.stam || { s: 1, lock: 0 }, btn = $('sprintBtn');
  if (!btn) return;
  btn.style.setProperty('--stam', Math.round((st.lock > 0 ? 1 - st.lock / 5 : st.s) * 100));
  btn.classList.toggle('locked', st.lock > 0);
  btn.classList.toggle('none', (st.max ?? 6) <= 0);
  $('sprintTxt').textContent = st.lock > 0 ? `Hụt hơi ${st.lock.toFixed(0)}s` : (st.max ?? 6) <= 0 ? 'Đói quá' : 'Chạy';
}
// KO: 5 ngôi sao vàng xoay quanh đầu gà đang nằm nghiêng ~2 giây (bám theo vị trí đầu thật của rig).
function koStars(mesh) {
  const head = mesh.userData.parts?.head, at = new T.Vector3();
  for (let i = 0; i < 5; i++) {
    const s = new T.Mesh(new T.OctahedronGeometry(0.12), mat(0xffd84a));
    const a0 = (i / 5) * Math.PI * 2;
    G.world.scene.add(s);
    effects.push({ obj: s, parent: G.world.scene, t: 0, dur: 2, update: (e) => {
      if (head) head.getWorldPosition(at); else at.set(mesh.position.x, 1.2, mesh.position.z);
      const a = a0 + e.t * 5;
      s.position.set(at.x + Math.cos(a) * 0.45, at.y + 0.35 + Math.sin(e.t * 9 + i) * 0.05, at.z + Math.sin(a) * 0.45);
      s.rotation.y += 0.2;
    } });
  }
}
function stars(mesh, color) {
  for (let i = 0; i < 10; i++) {
    const s = new T.Mesh(new T.OctahedronGeometry(0.09), mat(color));
    const a0 = (i / 10) * Math.PI * 2;
    mesh.add(s);
    effects.push({ obj: s, parent: mesh, t: 0, dur: 1, update: (e) => { const a = a0 + e.t * 4; s.position.set(Math.cos(a) * 0.8, 0.6 + e.t * 1.5, Math.sin(a) * 0.8); } });
  }
}

function animate(mesh, dt, moving, phase, stage) {
  const p = mesh.userData.parts;
  const rig = p.rig;
  rig.position.y = 0; rig.rotation.set(0, 0, 0);
  p.head.rotation.x = stage >= 2 ? 0.35 : 0; // gà đói cúi đầu
  p.wings[0].rotation.z = p.wings[1].rotation.z = 0;
  if (moving) {
    const run = mesh.userData.running;
    p.legs[0].rotation.x = Math.sin(phase) * (run ? 0.95 : 0.6);
    p.legs[1].rotation.x = -Math.sin(phase) * (run ? 0.95 : 0.6);
    rig.position.y = Math.abs(Math.sin(phase * 2)) * (run ? 0.1 : 0.05);
    if (run) { rig.rotation.x = 0.22; p.wings[0].rotation.z = -0.55 - Math.abs(Math.sin(phase)) * 0.3; p.wings[1].rotation.z = -p.wings[0].rotation.z; }
    else { p.wings[0].rotation.z = -Math.sin(phase) * 0.2; p.wings[1].rotation.z = Math.sin(phase) * 0.2; }
  } else {
    p.legs[0].rotation.x *= 0.8; p.legs[1].rotation.x *= 0.8;
    if (!G.reducedMotion) rig.position.y = Math.sin(performance.now() * 0.002 + phase) * 0.015;
  }
  const ride = mesh.userData.riding || (mesh.userData.seated && meshOf(mesh.userData.seated)?.userData.riding);
  if (ride) {
    rig.position.y = V.rideHeight(ride);
    p.legs[0].rotation.x = p.legs[1].rotation.x = -1.1;
  }
  if(mesh.userData.nest&&!moving){rig.position.y=-.08;p.legs[0].rotation.x=p.legs[1].rotation.x=-1.2;}
  const car = mesh.userData.carrier;
  if (car) { rig.position.y = car.kind === 'bus' ? 0.95 : V.rideHeight('xe_may') + 0.08; p.legs[0].rotation.x = p.legs[1].rotation.x = -1.1; return; }
  const tug = mesh.userData.tug;
  if (tug) {
    // Giằng co: cúi đầu ngậm giun, giữ thì ngả người kéo, căng quá thì rung.
    const shake = tug.tension > 0.7 ? Math.sin(performance.now() * 0.06) * 0.05 : 0;
    p.head.rotation.x = 0.75 - tug.pose * 0.45; rig.rotation.x = 0.25 - tug.pose * 0.45 + shake;
    p.legs[0].rotation.x = p.legs[1].rotation.x = tug.pose * 0.35;
    p.wings[0].rotation.z = -0.4 - tug.pose * 0.5; p.wings[1].rotation.z = -p.wings[0].rotation.z;
    return;
  }
  const an = mesh.userData.anim;
  if (!an) return;
  an.t += dt;
  const u = an.t / an.dur, t = an.t;
  switch (an.k) {
    case 'hop': rig.position.y = Math.abs(Math.sin(t * Math.PI * 3)) * 0.45; break;
    case 'jolt': rig.position.y = Math.sin(Math.min(1, u) * Math.PI) * 0.35; break;
    case 'wave': p.wings[1].rotation.z = 0.9 + Math.sin(t * 16) * 0.5; break;
    case 'droop': p.head.rotation.x = 0.55; rig.rotation.x = 0.12; break;
    case 'spin': rig.rotation.y = u * Math.PI * 4; p.wings[0].rotation.z = -0.8; p.wings[1].rotation.z = 0.8; break;
    case 'cheer': rig.position.y = Math.abs(Math.sin(t * Math.PI * 2.5)) * 0.35; p.head.rotation.x = -0.35; p.wings[0].rotation.z = -1.15; p.wings[1].rotation.z = 1.15; break;
    case 'party': rig.rotation.y = u * Math.PI * 2; rig.position.y = Math.abs(Math.sin(t * Math.PI * 3)) * 0.4; p.wings[0].rotation.z = -1; p.wings[1].rotation.z = 1; break;
    case 'crow': p.head.rotation.x = -0.55; p.wings[0].rotation.z = -0.6 - Math.abs(Math.sin(t * 14)) * 0.5; p.wings[1].rotation.z = -p.wings[0].rotation.z; break;
    case 'dig': p.head.rotation.x = 0.45 + Math.sin(t * 22) * 0.35; rig.rotation.x = 0.18; break;
    case 'eat': p.head.rotation.x = 0.5 + Math.abs(Math.sin(t * 14)) * 0.4; break;
    case 'sit': case 'sleep': {
      // Ngồi: hạ thân, gập chân; ngủ thêm cúi đầu. Đi lại thì đứng dậy ngay.
      if (moving) { mesh.userData.anim = null; break; }
      rig.position.y = -0.22;
      p.legs[0].rotation.x = p.legs[1].rotation.x = -1.2;
      if (an.k === 'sleep') { p.head.rotation.x = 0.6; rig.position.y = -0.24 + Math.sin(t * 2) * 0.01; }
      break;
    }
    case 'ko': {
      const fall = Math.min(1, t / 0.25), rise = u > 0.8 ? (u - 0.8) / 0.2 : 0;
      rig.rotation.z = (Math.PI / 2) * fall * (1 - rise);
      rig.position.y = 0.3 * fall * (1 - rise);
      break;
    }
  }
  if (an.t >= an.dur) mesh.userData.anim = null;
}

// ============ Vòng lặp ============
const clock = new T.Clock();
let lastSent = { x: 0, z: 0, at: 0 };
let stopResend = 0;
function lerpAngle(a, b, t) { let d = ((b - a + Math.PI) % (Math.PI * 2)) - Math.PI; if (d < -Math.PI) d += Math.PI * 2; return a + d * t; }

function loop() {
  requestAnimationFrame(loop);
  // Chặn 0,1s: đủ tránh nhảy cóc khi tab treo, nhưng khung chậm vẫn dự đoán bám tốc độ của server.
  const dt = Math.min(clock.getDelta(), 0.1);
  if (!G.self) { G.world.renderer.render(G.world.scene, G.world.camera); return; }
  const S = G.self, me = G.me;
  if (S.mesh.userData.anim && (S.mesh.userData.anim.k === 'sit' || S.mesh.userData.anim.k === 'sleep') && (keys.w || keys.a || keys.s || keys.d || joy.x || joy.y)) S.mesh.userData.anim = null;

  // Hướng input theo camera: lên = đi xa camera.
  let ix = 0, iz = 0;
  if (!G.modal && !G.busy && !G.myRideOf && !G.myCarrier) {
    if (keys.w || keys.arrowup) iz -= 1;
    if (keys.s || keys.arrowdown) iz += 1;
    if (keys.a || keys.arrowleft) ix -= 1;
    if (keys.d || keys.arrowright) ix += 1;
    ix += joy.x; iz += joy.y;
  }
  const mag = Math.min(1, Math.hypot(ix, iz));
  const yaw = G.world.camYaw, cy = Math.cos(yaw), sy = Math.sin(yaw);
  let wx = 0, wz = 0;
  if (mag > 0.12) {
    const l = Math.hypot(ix, iz);
    const nx = ix / l, nz = iz / l;
    wx = (nx * cy + nz * sy) * mag;
    wz = (-nx * sy + nz * cy) * mag;
  }
  if(mag>.12){G.walkPath=[];G.tapAction=null;}
  if(!G.modal&&!G.busy&&!G.myRideOf&&G.walkPath?.length&&mag<=.12){
   let next=G.walkPath[0];if(Math.hypot(next.x-S.pos.x,next.z-S.pos.z)<.45){G.walkPath.shift();next=G.walkPath[0];}
   if(next){const d=Math.hypot(next.x-S.pos.x,next.z-S.pos.z);wx=(next.x-S.pos.x)/d;wz=(next.z-S.pos.z)/d;}
   else if(G.tapAction){const it=G.tapAction;G.tapAction=null;if(Math.hypot(it.x-S.pos.x,it.z-S.pos.z)<it.r){nearest=it;interact();}}
  }
  const moving = Math.hypot(wx,wz) > 0.12;
  const now = performance.now();
  // Chạy nhanh: giữ Shift hoặc nút 🏃; máy chủ quyết định có chạy được không (thể lực, đói, xe).
  const wantRun = !!(keys.shift || G.runHeld) && !G.busy && !G.myRideOf && !G.myCarrier;
  // Gửi input khi đổi hướng/đổi chạy hoặc 10 lần/giây khi đang đi; server tự dừng nếu 300ms không có input.
  // Giới hạn ≤20 gói/giây (server bỏ quá ~30/s); dừng và đổi hướng mạnh luôn gửi ngay (B8).
  const dIn = Math.hypot(wx - lastSent.x, wz - lastSent.z);
  const wasMoving = Math.hypot(lastSent.x, lastSent.z) > 0.12;
  const stopNow = !moving && wasMoving;
  const sendIn = stopNow || wantRun !== lastSent.run || dIn > 0.4
    || (dIn > 0.05 && now - lastSent.at >= 50) || (moving && now - lastSent.at > 100);
  if (sendIn) {
    G.net.raw({ t: 'in', x: +wx.toFixed(3), z: +wz.toFixed(3), run: wantRun });
    lastSent = { x: wx, z: wz, at: now, run: wantRun };
    clearTimeout(stopResend);
    if (!moving) { // gửi lại lệnh dừng một lần nữa phòng khi gói đầu bị mất
      const run = wantRun;
      stopResend = setTimeout(() => { if (!G.net) return; if (Math.hypot(lastSent.x, lastSent.z) <= 0.12) G.net.raw({ t: 'in', x: 0, z: 0, run }); }, 150);
    }
  }
  if (moving && stopResend) { clearTimeout(stopResend); stopResend = 0; }
  const veh = G.myRide && G.catalog.vehicles.find((v) => v.id === G.myRide);
  const running = moving && wantRun && !veh && me.stage < 3 && !(G.stam && (G.stam.lock > 0 || G.stam.s <= 0.01)) && !S.mesh.userData.nest;
  S.mesh.userData.running = running;
  // Dự đoán cục bộ cùng luật với server, rồi hiệu chỉnh theo snapshot.
  if (moving) {
    const stageF = me.stage >= 4 ? 0.75 : me.stage >= 3 ? 0.85 : 1;
    const spd = (G.catalog.walkSpeed || 4.6) * stageF * (veh ? (me.stage >= 4 ? 0 : veh.speed) : 1) * (running ? G.catalog.sprintMult || 1.65 : 1) * (1 + (G.walkBonus || 0)) * dt;
    const pos=moveSafe(walkZone(),S.pos,S.pos.x+wx*spd,S.pos.z+wz*spd);
    const blk=G.areaBlock&&G.areaBlock.until>now&&G.worldMap?G.areaBlock:null;
    if(blk&&areaAt(G.zones,G.worldMap,pos.x,pos.z)===blk.area)S.pos.lerp(S.srv,Math.min(1,dt*10)); // khu đầy: đứng ở cổng như server
    else{S.pos.x=pos.x;S.pos.z=pos.z;}
    S.mesh.rotation.y = lerpAngle(S.mesh.rotation.y, Math.atan2(wx, wz), 0.25);
    S.phase += dt * (running ? 18 : 12);
    if (running && Math.random() < dt * 8) dust(S.mesh);
  }
  const err = S.srv.distanceTo(S.pos);
  if (G.myRideOf || G.myCarrier) S.pos.lerp(S.srv, Math.min(1, dt * 12)); // ngồi xe: vị trí ghế do server đặt
  else if (err > 2.5) S.pos.copy(S.srv);
  else if (!moving || err > 1.2) S.pos.lerp(S.srv, Math.min(1, dt * (moving ? 5 : 10)));
  S.mesh.position.copy(S.pos);
  if (G.worldMap) { const area = areaAt(G.zones, G.worldMap, S.pos.x, S.pos.z); if (area && area !== G.zone && !(G.areaBlock?.area === area && G.areaBlock.until > now)) announceArea(area); }
  if (G.spinSelf) S.mesh.rotation.y += dt * 1.2;
  animate(S.mesh, dt, moving, S.phase, me.stage);

  const k = 1 - Math.exp(-dt * 12);
  const camPos = G.world.camera.position;
  for (const p of G.players.values()) {
    const far = p.mesh.position.distanceToSquared(camPos) > 26 * 26;
    if (far !== p.far) { p.far = far; p.mesh.userData.parts.detail?.forEach((o) => (o.visible = !far)); }
    p.mesh.position.lerp(p.target, k);
    p.mesh.rotation.y = lerpAngle(p.mesh.rotation.y, p.ry, 0.2);
    if (p.mv) p.phase += dt * (p.mv === 2 ? 18 : 12);
    p.mesh.userData.running = p.mv === 2;
    animate(p.mesh, dt, !!p.mv, p.phase, p.stage);
  }
  for (let i = effects.length - 1; i >= 0; i--) {
    const e = effects[i];
    e.t += dt;
    e.update(e, dt);
    if (e.t >= e.dur) { e.parent.remove(e.obj); effects.splice(i, 1); }
  }

  SO.updatePlacing();
  V.updatePets(dt);
  NPC.update(dt, S.pos);
  SB.update();
  L.updatePlants();
  nearest = G.nearest = findNearest();
  const btn = $('interact');
  const work = !G.modal && V.workPrompt();
  const party = !G.modal && !work && partyPrompt();
  if (party) { btn.hidden = false; $('interactTxt').textContent = '🎉 Ăn mừng ở bàn tiệc'; }
  else if (work) { btn.hidden = false; $('interactTxt').textContent = `🔨 Thi công (${work.progress}/${work.need})`; }
  else if (nearest) { btn.hidden = false; $('interactTxt').textContent = promptText(nearest); }
  else if (!G.modal && NPC.nearestStop()) { btn.hidden = false; $('interactTxt').textContent = `🚌 Bến xe buýt · ${NPC.nearestStop().title}`; }
  else btn.hidden = true;

  // At most 8 name plates: own (outside the social view) plus the nearest players.
  const cam = G.world.camera;
  if (S.mesh.userData.tag) { S.mesh.userData.tag.visible = G.world.controls.mode !== 'social'; if (S.mesh.userData.tag.visible) fitLabel(S.mesh.userData.tag, cam); }
  const near = [...G.players.values()].sort((a, b) => a.mesh.position.distanceToSquared(S.pos) - b.mesh.position.distanceToSquared(S.pos));
  near.forEach((p, i) => { const tag = p.mesh.userData.tag; if (!tag) return; tag.visible = i < 7 && p.mesh.position.distanceTo(S.pos) < 12; if (tag.visible) fitLabel(tag, cam); });
  G.world.actors = [S.pos, ...near.slice(0, 20).map(p => p.mesh.position)];
  G.world.update(dt, S.pos, moving, G.myRide);
  MAP.updateRoute();
  updateBubbles(dt, G.world.camera);
}
loop();
setInterval(() => G.net.ping(), 5000);
setInterval(() => { if (G.me && G.me.stage >= 2) updateHUD(G.me); }, 1000);

// ============ Khởi động ============
const saved = store.get(TOKEN_KEY);
if (saved) { loading('Đang mở cổng trại…', 'Kết nối máy chủ'); G.net.connect(saved); }
else showScreen('authScreen');

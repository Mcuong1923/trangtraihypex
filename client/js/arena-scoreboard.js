// Bảng điểm gỗ 3D trên tường sau Sân tỷ thí (PLAN_GIAO_DIEN_BO_2 B02-04, PLAN_LOGIC_GAME 9.3).
// Máy chủ gửi `pk_board` cho cả phòng (pk_scoreboard.go) nên khán giả thấy cùng tỉ số và kết quả với
// hai đấu sĩ. Ưu tiên hiển thị: trận thật > trận biểu diễn của NPC > bảng trống.
import { G } from './ui.js';

const T = window.THREE;
const W = 512, H = 256;
let board = null;  // { mesh, ctx, tex }
let real = null;   // gói pk_board gần nhất (null hoặc state "idle" = không có trận thật)
let realEnds = 0;  // giây (performance.now) hết đếm ngược / hết hiệp
let show = null;   // trận biểu diễn: { names, a, b, winner, ko }
let drawn = '';    // khoá nội dung đã vẽ, để chỉ vẽ lại khi đổi

export function onBoard(m) {
  real = m.state === 'idle' ? null : m;
  realEnds = performance.now() / 1000 + (m.left || 0);
}

// npc.js báo tỉ số trận biểu diễn; null khi trận biểu diễn kết thúc.
export function setExhibition(x) { show = x; }

// Gọi mỗi khung hình: dựng bảng khi Sân tỷ thí đã có (hoặc được dựng lại) và vẽ lại khi nội dung đổi.
export function update() {
  const zone = G.world?.zones?.arena;
  if (!zone?.group) return;
  if (!board) build(zone.group); // nhóm của Sân tỷ thí chỉ dựng một lần cho cả phiên
  const view = content();
  const key = JSON.stringify(view);
  if (key !== drawn) { drawn = key; paint(view); }
}

function build(parent) {
  const cv = document.createElement('canvas'); cv.width = W; cv.height = H;
  const tex = new T.CanvasTexture(cv);
  if (T.SRGBColorSpace) tex.colorSpace = T.SRGBColorSpace;
  const g = new T.Group();
  // Hai cột gỗ và khung, mặt bảng quay ra cổng sân (+z), treo cao trên tường rào để không che Chó Mực.
  const wood = new T.MeshStandardMaterial({ color: 0x5b3d22, roughness: 0.9 });
  for (const x of [-2.05, 2.05]) { const post = new T.Mesh(new T.BoxGeometry(0.22, 4.7, 0.22), wood); post.position.set(x, 2.35, 0); g.add(post); }
  const frame = new T.Mesh(new T.BoxGeometry(4.15, 2.2, 0.16), wood); frame.position.set(0, 3.55, -0.02); g.add(frame);
  const face = new T.Mesh(new T.PlaneGeometry(3.9, 1.95), new T.MeshBasicMaterial({ map: tex, toneMapped: false }));
  face.position.set(0, 3.55, 0.07); g.add(face);
  const roof = new T.Mesh(new T.BoxGeometry(4.5, 0.12, 0.5), new T.MeshStandardMaterial({ color: 0x8a5a2b, roughness: 1 }));
  roof.position.set(0, 4.72, 0.05); g.add(roof);
  g.position.set(0, 0, -8.7);
  parent.add(g);
  board = { mesh: g, ctx: cv.getContext('2d'), tex };
  drawn = '';
}

// Nội dung cần vẽ, đã làm tròn giây để không vẽ lại mỗi khung hình.
function content() {
  if (real) {
    const m = real, left = Math.max(0, Math.ceil(realEnds - performance.now() / 1000));
    let status = '', ko = false;
    if (m.state === 'countdown') status = left > 0 ? `Chuẩn bị… ${left}` : 'Bắt đầu!';
    else if (m.state === 'play') status = m.mode === 'coin' ? `Lượt ${Math.min(m.round + 1, m.rounds)}/${m.rounds}` : `Còn ${left} giây`;
    else if (m.cancelled) status = 'Trận bị huỷ';
    else if (m.winner < 0) status = 'HÒA!';
    else if (m.forfeit) status = `${m.names[m.winner]} thắng (đối thủ bỏ trận)`;
    else { ko = true; status = `${m.names[m.winner]} thắng`; }
    return { title: (m.modeName || 'PK').toUpperCase(), names: m.names, a: m.a, b: m.b, status, ko, winner: m.winner };
  }
  if (show) {
    return { title: 'TRẬN BIỂU DIỄN', names: show.names, a: show.a, b: show.b, ko: show.ko,
      status: show.ko ? `${show.names[show.winner]} thắng` : 'Cổ vũ đi nào!', winner: show.ko ? show.winner : -1 };
  }
  return { title: 'SÂN TỶ THÍ', idle: true, status: 'Gặp Chó Mực để thách đấu' };
}

function fitFont(c, text, weight, size, maxW) {
  let s = size;
  do { c.font = `${weight} ${s}px system-ui, "Segoe UI", sans-serif`; } while (c.measureText(text).width > maxW && --s > 14);
}

function paint(v) {
  const c = board.ctx;
  // Ván gỗ sẫm với vân ngang, viền sáng.
  c.fillStyle = '#4e3420'; c.fillRect(0, 0, W, H);
  for (let y = 0; y < H; y += 32) { c.fillStyle = y % 64 ? '#553823' : '#4a311d'; c.fillRect(0, y, W, 30); }
  c.strokeStyle = '#c9a46d'; c.lineWidth = 8; c.strokeRect(6, 6, W - 12, H - 12);
  c.textAlign = 'center'; c.textBaseline = 'middle';
  c.fillStyle = '#f4e6c4'; fitFont(c, v.title, 'bold', 30, W - 60); c.fillText(v.title, W / 2, 34);
  if (v.idle) {
    c.fillStyle = '#ffd75e'; fitFont(c, '⚔️', 'bold', 72, 200); c.fillText('⚔️', W / 2, 118);
  } else {
    const [na, nb] = v.names;
    for (const [i, name, x] of [[0, na, 120], [1, nb, W - 120]]) {
      c.fillStyle = v.winner === i ? '#ffd75e' : '#f4e6c4';
      fitFont(c, name, 'bold', 28, 200); c.fillText(name, x, 82);
    }
    c.fillStyle = '#ffd75e'; c.font = 'bold 84px system-ui, "Segoe UI", sans-serif';
    c.fillText(String(v.a), 120, 148); c.fillText(String(v.b), W - 120, 148);
    if (!v.ko) { c.fillStyle = '#f4e6c4'; c.font = 'bold 60px system-ui, sans-serif'; c.fillText(':', W / 2, 144); }
    else { // chữ KO! đỏ giữa hai tỉ số khi có người thắng
      c.save(); c.translate(W / 2, 146); c.rotate(-0.12);
      c.fillStyle = '#d64b3a'; c.strokeStyle = '#fff3d6'; c.lineWidth = 6;
      c.font = 'bold 54px system-ui, sans-serif'; c.strokeText('KO!', 0, 0); c.fillText('KO!', 0, 0);
      c.restore();
    }
  }
  c.fillStyle = v.ko ? '#ffd75e' : '#f4e6c4'; fitFont(c, v.status, 'bold', 30, W - 60); c.fillText(v.status, W / 2, 214);
  board.tex.needsUpdate = true;
}

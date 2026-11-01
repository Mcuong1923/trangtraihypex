// Dựng cảnh 3D các khu từ dữ liệu zone của máy chủ. Vị trí tương tác lấy từ server;
// phần trang trí (ổ rơm, bao thóc, sạp…) là của client theo bộ ảnh minh hoạ.
import { mat, buildPig, buildDog, buildElder, buildCow, buildCoop, buildDecor, buildDuck, buildBird, buildGoat, buildFrog, buildBus, buildVehicle, label } from './models.js';

const T = window.THREE;

export class World {
  constructor(container) {
    this.renderer = new T.WebGLRenderer({ antialias: true });
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    this.renderer.shadowMap.enabled = true;
    this.renderer.shadowMap.type = T.PCFSoftShadowMap;
    // Giữ output tuyến tính: màu vật liệu khai báo theo hex sRGB như bảng màu ảnh minh hoạ,
    // bật sRGBEncoding sẽ chuyển gamma hai lần và làm cảnh bạc trắng.
    container.appendChild(this.renderer.domElement);

    this.scene = new T.Scene();
    this.scene.background = new T.Color(0xbcd089);
    this.scene.fog = new T.Fog(0xbcd089, 34, 72);
    this.camera = new T.PerspectiveCamera(50, 1, 0.1, 200);
    // yaw 0 = camera đứng phía lối vào (+z) nhìn vào trong, như góc chụp của bộ ảnh minh hoạ.
    this.camYaw = 0; this.camTargetYaw = 0; this.camDist = 15;

    // Tổng cường độ ~1 trên mặt hướng lên để giữ đúng tông đất nâu/cỏ xanh của ảnh minh hoạ.
    const sun = new T.DirectionalLight(0xfff2d0, 0.72);
    sun.position.set(14, 26, 10); sun.castShadow = true;
    sun.shadow.mapSize.set(2048, 2048);
    Object.assign(sun.shadow.camera, { left: -26, right: 26, top: 26, bottom: -26, near: 1, far: 80 });
    this.scene.add(sun, new T.HemisphereLight(0xdfeaff, 0x8a6b3a, 0.38), new T.AmbientLight(0xffffff, 0.1));

    this.sun = sun;
    this.reducedMotion = false;
    this.zones = {};   // id → { group, data, npcs: [] }
    this.current = null;
    this.resize();
    window.addEventListener('resize', () => this.resize());
  }

  // Mức đồ hoạ (mục 14, 18.1): giảm mật độ điểm ảnh và bóng đổ trên máy yếu.
  setQuality(q) {
    const dpr = window.devicePixelRatio || 1;
    this.renderer.setPixelRatio(q === 'low' ? 1 : q === 'mid' ? Math.min(dpr, 1.5) : Math.min(dpr, 2));
    this.renderer.shadowMap.enabled = q !== 'low';
    this.sun.castShadow = q !== 'low';
    const size = q === 'high' ? 2048 : 1024;
    if (this.sun.shadow.mapSize.x !== size) { this.sun.shadow.mapSize.set(size, size); this.sun.shadow.map?.dispose(); this.sun.shadow.map = null; }
    this.scene.traverse((o) => { if (o.material) o.material.needsUpdate = true; });
    this.resize();
  }

  resize() {
    const w = window.innerWidth, h = window.innerHeight;
    this.renderer.setSize(w, h);
    this.camera.aspect = w / h;
    this.camera.updateProjectionMatrix();
  }

  // Dựng các khu chưa có. Khu "home" khác nhau theo từng nhà nên dựng qua setHome().
  build(zones, cosmetics) {
    for (const id in zones) {
      if (id === 'home' || id === 'guildhall' || this.zones[id]) continue;
      const data = zones[id];
      const group = new T.Group();
      group.visible = false;
      const z = { group, data, npcs: [] };
      this.zones[id] = z;
      this.scene.add(group);
      decor[id]?.(group, data);
      fence(group, data.halfX * 2 + 1.2, data.halfZ * 2 + 1.2);
      for (const it of data.items) this.placeItem(z, it, cosmetics);
    }
  }

  placeItem(z, it, cosmetics) {
    const g = z.group;
    let obj;
    switch (it.kind) {
      case 'npc_elder': obj = buildElder(cosmetics); break;
      case 'npc_pig': case 'shop_sell': obj = buildPig(); break;
      case 'pk': obj = buildDog(); break;
      case 'npc_cow': obj = buildCow(); break;
      case 'npc_duck': obj = buildDuck(); break;
      case 'npc_bird': obj = buildBird(); break;
      case 'npc_goat': obj = buildGoat(); break;
      case 'npc_frog': obj = buildFrog(); break;
      case 'bus_stop': obj = busStop(); break;
      case 'parking': obj = parkingSign(); break;
      case 'road': obj = roadSign(); break;
      case 'dig': obj = digMound(it.label); break;
      case 'gate': case 'board': case 'shop_food': case 'shop_fashion': case 'trough': case 'home_sign': case 'home_exit': case 'home_board': case 'guild_board': obj = signpost(); break;
      default: obj = new T.Group();
    }
    obj.position.set(it.x, 0, it.z);
    if (it.kind.startsWith('npc') || it.kind === 'pk' || it.kind === 'shop_sell') { obj.rotation.y = it.kind === 'npc_cow' ? -0.6 : 0; if (it.kind !== 'npc_bird') z.npcs.push(obj); }
    const lab = label(`${it.emo} ${it.label}`, it.kind === 'dig' ? { bg: 'rgba(92,58,31,.75)', scale: 0.62 } : {});
    lab.position.set(0, it.kind === 'dig' ? 1.25 : (it.kind === 'npc_elder' ? 1.95 : 2.35), 0);
    obj.add(lab);
    obj.userData.item = it;
    g.add(obj);
  }

  // Dựng lại khu nhà riêng: đất theo hạng, chuồng/khung đang xây, nền chuồng, đồ đã đặt.
  setHome(data, home, cosmetics) {
    const wasCurrent = this.current && this.current === this.zones.home;
    if (this.zones.home) this.scene.remove(this.zones.home.group);
    const group = new T.Group();
    group.visible = wasCurrent;
    const z = { group, data, npcs: [], objects: new Map() };
    this.zones.home = z;
    this.scene.add(group);
    const W = data.halfX * 2 + 1.2, D = data.halfZ * 2 + 1.2;
    ground(group, W, D, 0xb08a55);
    tufts(group, 8, data.halfX, data.halfZ);
    fence(group, W, D);
    // Lối đất nện từ cổng vào như ảnh "Khu nhà bạn bè".
    const path = new T.Mesh(new T.BoxGeometry(1.6, 0.02, 3), mat(0x9a7446));
    path.position.set(0, 0.01, data.halfZ - 1); path.receiveShadow = true; group.add(path);
    const [cx, cz] = home.coopCell;
    if (home.coop) {
      const coop = buildCoop(home.coop.roof, home.coop.state);
      coop.position.set(cx, 0, cz - 0.5);
      group.add(coop);
      if (home.coop.state === 'building') {
        const l = label('🔨 Đang xây…', { bg: 'rgba(47,95,158,.85)', scale: 0.6 });
        l.position.set(cx, 2.4, cz - 0.5); group.add(l);
      }
    } else if (home.tier >= 1) {
      const pad = new T.Mesh(new T.BoxGeometry(3, 0.04, 2), mat(0x8f6a3e));
      pad.position.set(cx, 0.02, cz - 0.5); group.add(pad);
      const l = label('Nền chuồng', { bg: 'rgba(92,58,31,.7)', scale: 0.5 });
      l.position.set(cx, 0.6, cz - 0.5); group.add(l);
    }
    for (const o of home.objects) {
      const m = buildDecor(o.kind, home.ownerName);
      m.position.set(o.x, 0, o.z);
      m.rotation.y = -o.rot * Math.PI / 2;
      m.userData.obj = o;
      group.add(m);
      z.objects.set(o.id, m);
    }
    for (const it of data.items) this.placeItem(z, it, cosmetics);
    if (wasCurrent) this.current = z;
  }

  // Nhà Hội theo ảnh "Sân họp gà mộc mạc": bàn họp trải bản vẽ, vòng ổ rơm, lán vật liệu, bảng hội.
  setGuildHall(data, hall, cosmetics) {
    if (this.zones.guildhall) this.scene.remove(this.zones.guildhall.group);
    const group = new T.Group();
    group.visible = false;
    const z = { group, data, npcs: [] };
    this.zones.guildhall = z;
    this.scene.add(group);
    const W = data.halfX * 2 + 1.2, D = data.halfZ * 2 + 1.2;
    ground(group, W, D, 0xa8865a);
    tufts(group, 14, data.halfX, data.halfZ);
    fence(group, W, D);
    // Bàn họp và bản vẽ chuồng mẫu.
    box(group, 3.2, 0.14, 2.0, 0x8a5a30, 0, 0.75, -0.6);
    for (const [x, zz] of [[-1.4, -1.4], [1.4, -1.4], [-1.4, 0.2], [1.4, 0.2]]) box(group, 0.14, 0.75, 0.14, 0x6b4423, x, 0.37, zz);
    const paper = new T.Mesh(new T.BoxGeometry(1.6, 0.02, 1.1), mat(0xe8dcc0)); paper.position.set(0, 0.83, -0.6); group.add(paper);
    for (const [x, zz] of [[-1.1, -1.2], [1.0, -0.2]]) { const st = new T.Mesh(new T.DodecahedronGeometry(0.16), mat(0x9a9a9a)); st.position.set(x, 0.92, zz); group.add(st); }
    for (let i = 0; i < 6; i++) { const a = (i / 6) * Math.PI * 2; nest(group, Math.sin(a) * 2.9, -0.6 + Math.cos(a) * 2.2); }
    // Lán vật liệu phía sau, đống gỗ và đá, xe cút kít.
    box(group, 6, 0.15, 2.4, 0x8a5a30, 0, 0.2, -7);
    for (const sx of [-1, 1]) box(group, 0.2, 2.4, 0.2, 0x6b4423, sx * 2.8, 1.2, -6);
    for (let i = 0; i < 4; i++) box(group, 1.6, 0.18, 0.3, 0xa47447, -1.6, 0.4 + i * 0.18, -7 + (i % 2) * 0.2);
    for (let i = 0; i < 4; i++) { const st = new T.Mesh(new T.DodecahedronGeometry(0.3), mat(0x9a9a9a)); st.position.set(1.4 + (i % 2) * 0.5, 0.45, -7 + Math.floor(i / 2) * 0.5); group.add(st); }
    for (let i = 0; i < 5; i++) box(group, 2.2, 0.2, 0.3, 0x9c6b36, 7.2, 0.12 + (i % 3) * 0.2, 1.6 + i * 0.35);
    box(group, 1.3, 0.3, 0.8, 0x7b4f2d, -7.4, 0.5, 1.2);
    const wh = new T.Mesh(new T.TorusGeometry(0.28, 0.07, 8, 14), mat(0x2b2b2b)); wh.position.set(-6.7, 0.3, 1.2); wh.rotation.y = Math.PI / 2; group.add(wh);
    hay(group, -8.6, -6.4, 1.0);
    for (let i = 0; i < 3; i++) box(group, 0.55, 0.45, 0.42, 0xe8dcc0, -8.8, 0.25 + (i > 1 ? 0.42 : 0), -4 + (i % 2) * 0.6);
    barrel(group, -7.8, -4.4);
    const p = hall.projects || {};
    if (p.nha_hoi_2) { box(group, 6.4, 0.14, 2.8, 0xb8433a, 0, 2.6, -6.6); box(group, 3.6, 0.3, 1.4, 0x8a5a30, 0, 0.35, 5.4); }
    if (p.co_hoi) {
      const col = parseInt(String(hall.color || '#d64b3a').slice(1), 16);
      for (const sx of [-1, 1]) { box(group, 0.1, 3.4, 0.1, 0x5c3a1f, sx * 4.2, 1.7, -3.6); const f = new T.Mesh(new T.BoxGeometry(1.2, 0.8, 0.04), mat(col)); f.position.set(sx * 4.2 + sx * 0.62, 3.0, -3.6); group.add(f); }
    }
    if (p.den_hoi) for (const [x, zz] of [[-5, -2], [5, -2], [-5, 3.5], [5, 3.5]]) {
      box(group, 0.08, 2.0, 0.08, 0x5c3a1f, x, 1.0, zz);
      const l = new T.Mesh(new T.SphereGeometry(0.24, 10, 8), mat(0xe2452e, { emissive: 0x8a1f0a })); l.position.set(x, 2.15, zz); group.add(l);
    }
    const sign = label(`${hall.emblem || '🏰'} ${hall.name}`, { bg: 'rgba(61,43,26,.86)', scale: 0.9 });
    sign.position.set(0, 2.6, -0.6); group.add(sign);
    for (const it of data.items) this.placeItem(z, it, cosmetics);
  }

  // Đèn lồng và cờ đuôi nheo ở sảnh khi cộng đồng góp đủ cho ngày hội (mục 11).
  setFestival(on) {
    const lobby = this.zones.lobby;
    if (!lobby) return;
    if (lobby.festival) { lobby.group.remove(lobby.festival); lobby.festival = null; }
    if (!on) return;
    const g = new T.Group();
    const colors = [0xd64b3a, 0xe7c23a, 0x3f86c9, 0x4e9a4e];
    for (const z of [-3.6, 3.2]) {
      const rope = new T.Mesh(new T.BoxGeometry(22, 0.03, 0.03), mat(0x5c3a1f)); rope.position.set(0, 3.1, z); g.add(rope);
      for (let i = 0; i < 12; i++) {
        const f = new T.Mesh(new T.ConeGeometry(0.22, 0.4, 3), mat(colors[i % 4])); f.rotation.x = Math.PI; f.position.set(-10 + i * 1.8, 2.85, z); g.add(f);
      }
    }
    for (const [x, z] of [[-6, -5], [6, -5], [-11, 5], [11, 5], [-6, 6.5], [6, 6.5]]) {
      const post = new T.Mesh(new T.BoxGeometry(0.1, 2.2, 0.1), mat(0x5c3a1f)); post.position.set(x, 1.1, z); g.add(post);
      const l = new T.Mesh(new T.SphereGeometry(0.28, 10, 8), mat(0xe2452e, { emissive: 0x8a1f0a })); l.scale.y = 1.25; l.position.set(x, 2.35, z); g.add(l);
      const cap = new T.Mesh(new T.CylinderGeometry(0.12, 0.16, 0.08, 8), mat(0xe7c23a)); cap.position.set(x, 2.72, z); g.add(cap);
    }
    lobby.group.add(g);
    lobby.festival = g;
  }

  show(id) {
    if (this.current) this.current.group.visible = false;
    this.current = this.zones[id];
    this.current.group.visible = true;
    this.camTargetYaw = this.camYaw = 0;
  }

  update(dt, target) {
    this.camYaw += (this.camTargetYaw - this.camYaw) * (this.reducedMotion ? 1 : Math.min(1, dt * 6));
    const tx = target ? target.x : 0, tz = target ? target.z : 0;
    // Góc nhìn chéo cao ~55° như bộ ảnh minh hoạ; điểm nhìn đặt trước gà một chút để thấy nhiều sân phía trước.
    const fx = -Math.sin(this.camYaw) * 2, fz = -Math.cos(this.camYaw) * 2;
    this.camera.position.set(tx + Math.sin(this.camYaw) * this.camDist * 0.72, 4 + this.camDist * 0.95, tz + Math.cos(this.camYaw) * this.camDist * 0.72);
    this.camera.lookAt(tx + fx, 0.5, tz + fz);
    const t = performance.now() * 0.002;
    if (this.current && !this.reducedMotion) this.current.npcs.forEach((n, i) => { n.position.y = Math.abs(Math.sin(t + i)) * 0.04; });
    this.renderer.render(this.scene, this.camera);
  }
}

// ---------- Trang trí ----------

function ground(g, w, d, hex) {
  const m = new T.Mesh(new T.BoxGeometry(w, 0.4, d), mat(hex));
  m.position.y = -0.2; m.receiveShadow = true; g.add(m);
  const grass = new T.Mesh(new T.BoxGeometry(w + 40, 0.38, d + 40), mat(0x9fbf63));
  grass.position.y = -0.23; grass.receiveShadow = true; g.add(grass);
}

function tufts(g, n, hx, hz) {
  for (let i = 0; i < n; i++) {
    const t = new T.Group();
    for (let k = 0; k < 3; k++) {
      const b = new T.Mesh(new T.ConeGeometry(0.04, 0.3, 4), mat(0x6fa03a));
      b.position.set((Math.random() - 0.5) * 0.2, 0.15, (Math.random() - 0.5) * 0.2); b.rotation.z = (Math.random() - 0.5) * 0.4;
      t.add(b);
    }
    t.position.set((Math.random() - 0.5) * hx * 2, 0, (Math.random() - 0.5) * hz * 2);
    g.add(t);
  }
}

function box(g, w, h, d, hex, x, y, z, ry = 0) {
  const m = new T.Mesh(new T.BoxGeometry(w, h, d), mat(hex));
  m.position.set(x, y, z); m.rotation.y = ry; m.castShadow = true; m.receiveShadow = true; g.add(m);
  return m;
}

function hay(g, x, z, s = 1) {
  const m = new T.Mesh(new T.ConeGeometry(1 * s, 1.7 * s, 10), mat(0xd9b65a));
  m.position.set(x, 0.85 * s, z); m.castShadow = true; g.add(m);
}

function barrel(g, x, z, top = 0xb59a6a) {
  const b = new T.Mesh(new T.CylinderGeometry(0.38, 0.34, 0.75, 12), mat(0x7b4f2d));
  b.position.set(x, 0.38, z); b.castShadow = true; g.add(b);
  const t = new T.Mesh(new T.CylinderGeometry(0.33, 0.33, 0.04, 12), mat(top));
  t.position.set(x, 0.76, z); g.add(t);
}

function nest(g, x, z) {
  const n = new T.Mesh(new T.TorusGeometry(0.5, 0.18, 8, 16), mat(0xd9b65a));
  n.rotation.x = Math.PI / 2; n.position.set(x, 0.18, z); n.castShadow = true; g.add(n);
  const floor = new T.Mesh(new T.CircleGeometry(0.42, 14), mat(0xb8913f));
  floor.rotation.x = -Math.PI / 2; floor.position.set(x, 0.04, z); g.add(floor);
}

function fence(g, w, d) {
  const hw = w / 2, hd = d / 2;
  const side = (x1, z1, x2, z2, gapCenter) => {
    const len = Math.hypot(x2 - x1, z2 - z1), ang = Math.atan2(z2 - z1, x2 - x1);
    const segs = gapCenter == null ? [[0, 1]] : [[0, 0.5 - 1.6 / len], [0.5 + 1.6 / len, 1]];
    for (const [a, b] of segs) {
      const sx = x1 + (x2 - x1) * a, sz = z1 + (z2 - z1) * a, ex = x1 + (x2 - x1) * b, ez = z1 + (z2 - z1) * b;
      const l = Math.hypot(ex - sx, ez - sz);
      for (const y of [0.32, 0.74]) box(g, l, 0.36, 0.14, 0x7b4f2d, (sx + ex) / 2, y, (sz + ez) / 2, -ang);
      const n = Math.max(1, Math.round(l / 2.2));
      for (let i = 0; i <= n; i++) box(g, 0.24, 1.15, 0.24, 0x5c3a1f, sx + (ex - sx) * (i / n), 0.57, sz + (ez - sz) * (i / n));
    }
  };
  side(-hw, -hd, hw, -hd);
  side(hw, -hd, hw, hd);
  side(hw, hd, -hw, hd, 0); // lối vào phía camera
  side(-hw, hd, -hw, -hd);
}

function busStop() {
  const g = new T.Group();
  for (const sx of [-1, 1]) box(g, 0.12, 2.2, 0.12, 0x5c3a1f, sx * 1.1, 1.1, -0.4);
  box(g, 2.6, 0.1, 1.2, 0x3f86c9, 0, 2.25, -0.1);
  box(g, 2.0, 0.12, 0.45, 0x9c6b36, 0, 0.5, -0.45);
  box(g, 0.7, 0.9, 0.06, 0xf4f0e6, 1.4, 1.5, -0.4);
  return g;
}
function parkingSign() {
  const g = new T.Group();
  box(g, 0.1, 1.6, 0.1, 0x9a9a9a, 0, 0.8, 0);
  box(g, 0.6, 0.6, 0.06, 0x3f86c9, 0, 1.7, 0);
  for (let i = -1; i <= 1; i++) box(g, 0.08, 0.02, 2.4, 0xf4f0e6, i * 1.6 + 1.6, 0.02, 0.8);
  return g;
}
function roadSign() {
  const g = new T.Group();
  box(g, 0.12, 1.6, 0.12, 0x6b4423, 0, 0.8, 0);
  box(g, 1.3, 0.4, 0.08, 0x4e9a4e, 0.2, 1.5, 0);
  return g;
}

function signpost() {
  const g = new T.Group();
  box(g, 0.16, 1.4, 0.16, 0x6b4423, 0, 0.7, 0);
  box(g, 1.2, 0.6, 0.08, 0x9c6b36, 0, 1.5, 0);
  return g;
}

function digMound(soil) {
  const g = new T.Group();
  const hex = soil === 'Đất ẩm' ? 0x4f3520 : soil === 'Gốc cây' ? 0x7a5531 : 0x6e4a28;
  const dirt = new T.Mesh(new T.ConeGeometry(0.85, 0.5, 12), mat(hex));
  dirt.position.y = 0.22; dirt.castShadow = true; dirt.receiveShadow = true; g.add(dirt);
  const hole = new T.Mesh(new T.CircleGeometry(0.38, 14), mat(0x2e1f12));
  hole.rotation.x = -Math.PI / 2; hole.position.y = 0.48; g.add(hole);
  for (let i = 0; i < 6; i++) {
    const a = Math.random() * Math.PI * 2, r = 0.9 + Math.random() * 0.4;
    const c = new T.Mesh(new T.DodecahedronGeometry(0.09 + Math.random() * 0.06), mat(hex));
    c.position.set(Math.cos(a) * r, 0.05, Math.sin(a) * r); g.add(c);
  }
  g.userData.dirt = dirt;
  return g;
}

// Mô hình xem trước khi đặt đồ trong nhà (bán trong suốt, đổi màu theo hợp lệ).
export function ghostDecor(kind) {
  const g = buildDecor(kind);
  g.traverse((o) => { if (o.isMesh) { o.material = o.material.clone(); o.material.transparent = true; o.material.opacity = 0.55; o.castShadow = false; } });
  return g;
}

function road(g, x, z, w, d, ry = 0) {
  const r = new T.Mesh(new T.BoxGeometry(w, 0.04, d), mat(0x6f6a62));
  r.position.set(x, 0.02, z); r.rotation.y = ry; r.receiveShadow = true; g.add(r);
  for (let i = -Math.floor(d / 2) + 1; i < d / 2; i += 2) {
    const l = new T.Mesh(new T.BoxGeometry(0.12, 0.05, 0.8), mat(0xf4f0e6));
    l.position.set(x, 0.03, z + i); g.add(l);
  }
}
function field(g, x, z, w, d) {
  const f = new T.Mesh(new T.BoxGeometry(w, 0.05, d), mat(0x9cbf4a)); f.position.set(x, 0.02, z); g.add(f);
  for (let i = 0; i < w * d / 3; i++) {
    const s = new T.Mesh(new T.ConeGeometry(0.06, 0.4, 4), mat(0xc8b64a));
    s.position.set(x + (Math.random() - 0.5) * w, 0.2, z + (Math.random() - 0.5) * d); g.add(s);
  }
}
function stream(g, x, w, d) {
  const s = new T.Mesh(new T.BoxGeometry(w, 0.05, d), mat(0x4aa3c9, { roughness: 0.25 }));
  s.position.set(x, 0.03, 0); s.receiveShadow = true; g.add(s);
  for (let i = 0; i < 14; i++) {
    const reed = new T.Mesh(new T.CylinderGeometry(0.03, 0.03, 0.8, 4), mat(0x6f8f3a));
    reed.position.set(x + (Math.random() < 0.5 ? -1 : 1) * (w / 2 + 0.2), 0.4, (Math.random() - 0.5) * d); g.add(reed);
  }
}

const decor = {
  hatchery(g, z) {
    ground(g, z.halfX * 2 + 1.2, z.halfZ * 2 + 1.2, 0xc9a978);
    tufts(g, 8, z.halfX, z.halfZ);
    // Dãy ổ ấp có trứng dưới đèn sưởi (ảnh "Trạm ấp trứng gà").
    for (let i = 0; i < 4; i++) {
      const x = -3 + i * 2.4, zz = -5;
      nest(g, x, zz);
      for (let k = 0; k < 3; k++) { const e = new T.Mesh(new T.SphereGeometry(0.13, 10, 8), mat(0xf6ecd6)); e.scale.y = 1.25; e.position.set(x + (k - 1) * 0.18, 0.2, zz + (k % 2) * 0.12); g.add(e); }
      box(g, 0.06, 1.6, 0.06, 0x5c3a1f, x, 0.8, zz - 0.7);
      const lamp = new T.Mesh(new T.ConeGeometry(0.22, 0.25, 10, 1, true), mat(0xe7a73a, { emissive: 0x7a3a0a }));
      lamp.position.set(x, 1.55, zz - 0.35); lamp.rotation.x = Math.PI; g.add(lamp);
    }
    box(g, 2.4, 1.6, 0.6, 0x8a5a30, 6.4, 0.8, -6);
    for (let r = 0; r < 3; r++) for (let c = 0; c < 4; c++) { const e = new T.Mesh(new T.SphereGeometry(0.1, 8, 6), mat(0xf6ecd6)); e.position.set(5.6 + c * 0.5, 0.45 + r * 0.45, -5.65); g.add(e); }
    hay(g, -8, -6, 0.9); barrel(g, 7.6, 3);
  },
  station(g, z) {
    ground(g, z.halfX * 2 + 1.2, z.halfZ * 2 + 1.2, 0xb59a6a);
    road(g, 0, -1.5, 3, z.halfZ * 2 + 1.2);
    field(g, -7, -5, 5, 4); field(g, 7.5, -5.5, 4, 3);
    const bus = buildBus(); bus.position.set(4.6, 0, -4.2); g.add(bus);
    g.userData.bus = bus;
    tufts(g, 10, z.halfX, z.halfZ);
  },
  garage(g, z) {
    ground(g, z.halfX * 2 + 1.2, z.halfZ * 2 + 1.2, 0xa8a196);
    box(g, 7, 2.6, 0.3, 0x8a5a30, 0, 1.3, -7.6);
    box(g, 7.4, 0.15, 2.4, 0xd64b3a, 0, 2.7, -6.6);
    const show = [['xe_dap', -5.5], ['xe_may', -2], ['o_to', 2.2]];
    for (const [k, x] of show) { const v = buildVehicle(k); v.position.set(x, 0, -5.2); v.rotation.y = 0.4; g.add(v); }
    barrel(g, 7.5, -6.5, 0x2b2b2b); barrel(g, 8.1, -5.7, 0x2b2b2b);
    tufts(g, 6, z.halfX, z.halfZ);
  },
  vs_station(g, z) {
    ground(g, z.halfX * 2 + 1.2, z.halfZ * 2 + 1.2, 0xa9a36a);
    road(g, 0, 0, 3, z.halfZ * 2 + 1.2);
    field(g, 7, -4, 4, 5);
    const bus = buildBus(); bus.position.set(4.6, 0, -4.2); g.add(bus);
    g.userData.bus = bus;
    tufts(g, 12, z.halfX, z.halfZ);
  },
  vensuoi(g, z) {
    ground(g, z.halfX * 2 + 1.2, z.halfZ * 2 + 1.2, 0x9c8a55);
    // Dòng suối chảy dọc đồng, cầu gỗ bắc ngang (ảnh "Trang trại gà ven suối").
    stream(g, 9.4, 2.4, z.halfZ * 2 + 1.2);
    box(g, 3.4, 0.15, 1.4, 0x8a5a30, 9.4, 0.35, -5);
    for (const sx of [-1, 1]) box(g, 3.4, 0.4, 0.08, 0x6b4423, 9.4, 0.6, -5 + sx * 0.66);
    const mud = new T.Mesh(new T.CircleGeometry(4.5, 20), mat(0x5e4329)); mud.rotation.x = -Math.PI / 2; mud.position.set(-3, 0.01, -3.2); g.add(mud);
    tufts(g, 18, z.halfX, z.halfZ);
    for (const [x, zz] of [[-9.5, -7], [-9.5, 6], [4, -8]]) { const t = new T.Group(); box(t, 0.3, 1.2, 0.3, 0x6b4423, 0, 0.6, 0); const c = new T.Mesh(new T.IcosahedronGeometry(1.0), mat(0x3f7d3a)); c.position.y = 1.7; t.add(c); t.position.set(x, 0, zz); g.add(t); }
  },
  lobby(g, z) {
    ground(g, z.halfX * 2 + 1.2, z.halfZ * 2 + 1.2, 0xcdae7a);
    tufts(g, 16, z.halfX, z.halfZ);
    // Sân khấu gỗ của Gà trưởng trại + cầu đậu thấp hai bên (ảnh "Sảnh nông trại gà").
    // Chừa hành lang 2m phía sau sân khấu để đi tới cổng "Ra Chợ thóc".
    box(g, 6, 0.3, 2, 0x8a5a30, 0, 0.15, -5.4);
    for (const sx of [-1, 1]) box(g, 1, 0.15, 0.6, 0x7b4f2d, sx * 3.4, 0.08, -4.6);
    for (const sx of [-1, 1]) for (let i = 0; i < 2; i++) box(g, 3.2, 0.12, 0.12, 0x6b4423, sx * 10.4, 0.55 + i * 0.5, -5 + i * 1.6);
    // Dãy ổ rơm chung (tiện ích ngồi nghỉ, không phải nhà riêng).
    for (let r = 0; r < 2; r++) for (let c = 0; c < 5; c++) nest(g, -5 + c * 2.5, -2.4 + r * 2.4);
    hay(g, -10.6, -7.6, 1.1); hay(g, -9.2, -6.4, 0.8);
    for (let i = 0; i < 3; i++) box(g, 0.55, 0.5, 0.42, 0xe8dcc0, -11.3, 0.25 + (i > 1 ? 0.45 : 0), -1.4 + (i % 2) * 0.6, 0.2);
    barrel(g, -11.2, -3, 0xe7a73a); barrel(g, 11.3, -6.6);
    box(g, 0.7, 0.3, 2.6, 0x6b4423, -10.6, 0.25, 2.5);
    box(g, 0.5, 0.12, 2.4, 0xe7a73a, -10.6, 0.43, 2.5);
    box(g, 0.7, 0.3, 3.2, 0x6b4423, 10.8, 0.25, 1.5);
    const water = new T.Mesh(new T.BoxGeometry(0.5, 0.06, 3), mat(0x4aa3c9, { roughness: 0.3 }));
    water.position.set(10.8, 0.42, 1.5); g.add(water);
    // Biển hiệu Hypex.
    const sign = label('🐔 Chuồng gà Hypex', { bg: 'rgba(224,89,43,.92)', scale: 1.2 });
    sign.position.set(0, 2.4, -5.4); g.add(sign);
  },
  garden(g, z) {
    ground(g, z.halfX * 2 + 1.2, z.halfZ * 2 + 1.2, 0xb79a63);
    tufts(g, 14, z.halfX, z.halfZ);
    // Vùng đất ẩm sẫm màu ở góc phải như ảnh minh hoạ.
    const wet = new T.Mesh(new T.CircleGeometry(3.2, 20), mat(0x5e4329));
    wet.rotation.x = -Math.PI / 2; wet.position.set(6.2, 0.01, -1.8); g.add(wet);
    hay(g, -8.6, -7, 1);
    const stump = new T.Mesh(new T.CylinderGeometry(0.75, 0.9, 0.9, 12), mat(0x6b4423));
    stump.position.set(-0.5, 0.45, -6.3); stump.castShadow = true; g.add(stump);
    const ring = new T.Mesh(new T.CylinderGeometry(0.74, 0.74, 0.02, 12), mat(0xcfa877));
    ring.position.set(-0.5, 0.91, -6.3); g.add(ring);
    // Giá treo xẻng.
    box(g, 2.2, 0.12, 0.12, 0x6b4423, -9.3, 1.3, -2.5, Math.PI / 2);
    for (let i = 0; i < 3; i++) box(g, 0.08, 1.1, 0.25, 0x9a9a9a, -9.3, 0.7, -3.3 + i * 0.7);
    barrel(g, -9.2, 0.2, 0x8ec6d9); barrel(g, 8.6, -6.5, 0x8ec6d9);
  },
  market(g, z) {
    ground(g, z.halfX * 2 + 1.2, z.halfZ * 2 + 1.2, 0xc6a976);
    tufts(g, 12, z.halfX, z.halfZ);
    const stall = (x, roof, goods) => {
      box(g, 3, 0.18, 1.3, 0x8a5a30, x, 0.95, -6.2);
      for (const sx of [-1, 1]) { box(g, 0.16, 0.95, 0.16, 0x6b4423, x + sx * 1.3, 0.47, -6.2); box(g, 0.14, 2.2, 0.14, 0x6b4423, x + sx * 1.4, 1.1, -5.6); box(g, 0.14, 2.2, 0.14, 0x6b4423, x + sx * 1.4, 1.1, -6.9); }
      box(g, 3.3, 0.12, 1.7, roof, x, 2.25, -6.25);
      goods(x);
    };
    stall(-7, 0x9c6b36, (x) => { box(g, 0.7, 0.25, 0.5, 0x7b4f2d, x - 0.7, 1.17, -6.2); box(g, 0.6, 0.12, 0.4, 0xc86f6f, x - 0.7, 1.32, -6.2); for (let i = 0; i < 2; i++) box(g, 0.5, 0.4, 0.4, 0xe8dcc0, x + 0.8, 1.25 + i * 0.4, -6.2); });
    stall(0, 0xb5743c, (x) => {
      const bowl = (bx, col, s) => { const b = new T.Mesh(new T.CylinderGeometry(0.28 * s, 0.2 * s, 0.16, 12), mat(0x9c6b36)); b.position.set(bx, 1.12, -6.2); g.add(b); const f = new T.Mesh(new T.CylinderGeometry(0.25 * s, 0.25 * s, 0.04, 12), mat(col)); f.position.set(bx, 1.2, -6.2); g.add(f); };
      bowl(x - 0.9, 0xe7a73a, 1.4); bowl(x - 0.1, 0x8a6b3a, 1); bowl(x + 0.5, 0x6fa03a, 1); bowl(x + 1.05, 0xc9a35a, 1);
    });
    stall(7, 0x8a5a30, (x) => {
      box(g, 2.6, 0.06, 0.06, 0x6b4423, x, 2.0, -5.9);
      box(g, 0.6, 0.6, 0.06, 0x3f86c9, x - 0.7, 1.6, -5.9); box(g, 0.6, 0.6, 0.06, 0xd64b3a, x, 1.6, -5.9);
      const hat1 = new T.Mesh(new T.CylinderGeometry(0.3, 0.3, 0.04, 14), mat(0xd9b65a)); hat1.position.set(x + 0.8, 1.08, -6.2); g.add(hat1);
      const hat2 = new T.Mesh(new T.CylinderGeometry(0.3, 0.3, 0.04, 14), mat(0xd64b3a)); hat2.position.set(x + 0.3, 1.08, -6.1); g.add(hat2);
    });
    hay(g, -10, -7, 0.9); barrel(g, -10.2, -4.5);
  },
  arena(g, z) {
    ground(g, z.halfX * 2 + 1.2, z.halfZ * 2 + 1.2, 0xb98a52);
    tufts(g, 10, z.halfX, z.halfZ);
    const sand = new T.Mesh(new T.CircleGeometry(4.2, 36), mat(0xcaa86a));
    sand.rotation.x = -Math.PI / 2; sand.position.y = 0.01; sand.receiveShadow = true; g.add(sand);
    const ring = new T.Mesh(new T.TorusGeometry(4.2, 0.22, 8, 40), mat(0xe8e0cc));
    ring.rotation.x = Math.PI / 2; ring.position.y = 0.06; g.add(ring);
    for (let i = 0; i < 8; i++) { const a = (i / 8) * Math.PI * 2; box(g, 0.2, 0.7, 0.2, 0x5c3a1f, Math.cos(a) * 4.6, 0.35, Math.sin(a) * 4.6); }
    // Băng ghế khán giả.
    for (const sx of [-1, 1]) box(g, 0.7, 0.4, 5, 0x8a5a30, sx * 7.8, 0.2, 0);
    for (let i = 0; i < 3; i++) { const f = new T.Mesh(new T.ConeGeometry(0.3, 0.5, 3), mat([0xd64b3a, 0xe7c23a, 0x3f86c9][i])); f.rotation.x = Math.PI; f.position.set(-3 + i * 3, 2.6, -8.4); g.add(f); }
    box(g, 9, 0.04, 0.04, 0x5c3a1f, 0, 2.85, -8.4);
  },
};

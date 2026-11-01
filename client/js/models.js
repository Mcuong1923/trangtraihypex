// Mô hình low-poly theo bộ ảnh minh hoạ: gà đầu to, thân tròn, mào đỏ, chân cam.
const T = window.THREE;

export const FEATHER = { vang: 0xf4cf3f, kem: 0xf0e2b0, cam: 0xe9933a, nau: 0xb5743c, trang: 0xf5f2e9, xam: 0xb9b3a6 };
export const SHIRT = { none: null, xanh: 0x3f86c9, do: 0xd64b3a, luc: 0x4e9a4e, tim: 0x8a6fc0 };

const matCache = new Map();
export function mat(hex, opts) {
  const key = hex + '|' + (opts ? JSON.stringify(opts) : '');
  if (!matCache.has(key)) matCache.set(key, new T.MeshStandardMaterial(Object.assign({ color: hex, roughness: 0.92, metalness: 0, flatShading: false }, opts || {})));
  return matCache.get(key);
}
const hexOf = (s) => parseInt(String(s).replace('#', ''), 16);

const geometryCache = new Map();
function mesh(geo, hex, shadow = true) {
  // Identical avatars share GPU geometry as well as materials.
  const key = geo.type + JSON.stringify(geo.parameters || {});
  if (geometryCache.has(key)) { geo.dispose(); geo = geometryCache.get(key); }
  else geometryCache.set(key, geo);
  const m = new T.Mesh(geo, mat(hex));
  m.castShadow = shadow;
  return m;
}

export function buildChicken({ color = 'vang', comb = 'don', tail: tailKind = 'quat', eyes = 'tron', bodyHex } = {}) {
  const hex = bodyHex ?? FEATHER[color] ?? FEATHER.vang;
  const g = new T.Group();
  const rig = new T.Group(); // xoay/nhún cả thân khi biểu cảm, KO
  g.add(rig);
  const parts = { rig };

  const body = mesh(new T.SphereGeometry(0.5, 24, 18), hex);
  body.scale.set(1, 0.92, 1.08); body.position.y = 0.55; rig.add(body);
  parts.body = body;

  // Ba kiểu đuôi (mục 5): quạt, nhọn, xoăn.
  const tail = new T.Group(); tail.position.set(0, 0.62, -0.45);
  if (tailKind === 'nhon') {
    const f = mesh(new T.ConeGeometry(0.1, 0.7, 5), hex); f.rotation.x = -2.5; f.position.y = 0.12; tail.add(f);
  } else if (tailKind === 'xoan') {
    for (const sx of [-1, 1]) { const c = mesh(new T.TorusGeometry(0.14, 0.05, 6, 12, Math.PI * 1.5), hex); c.rotation.y = Math.PI / 2; c.position.set(sx * 0.07, 0.12, -0.04); tail.add(c); }
  } else {
    for (let i = -1; i <= 1; i++) {
      const f = mesh(new T.SphereGeometry(0.12, 12, 10), hex);
      f.scale.set(0.65, 1.9, 0.7); f.rotation.x = -2.2; f.rotation.z = i * 0.4; f.position.x = i * 0.12; tail.add(f);
    }
  }
  rig.add(tail);

  const head = new T.Group(); head.position.set(0, 1.02, 0.08); rig.add(head); parts.head = head;
  head.add(mesh(new T.SphereGeometry(0.34, 24, 18), hex));
  for (const sx of [-1, 1]) head.add(eyeMesh(eyes, sx));
  const beak = mesh(new T.ConeGeometry(0.1, 0.22, 6), 0xe8922b, false);
  beak.rotation.x = Math.PI / 2; beak.position.set(0, -0.02, 0.37); head.add(beak);

  const combHex = 0xe0401f;
  const cg = new T.Group(); cg.position.set(0, 0.33, 0.05); head.add(cg);
  const blob = (x, s) => { const b = mesh(new T.SphereGeometry(0.1 * s, 8, 8), combHex, false); b.position.x = x; b.scale.y = 1.3; cg.add(b); };
  if (comb === 'kep') { blob(-0.09, 1); blob(0.09, 1); blob(0, 0.8); }
  else if (comb === 'ro') { for (let i = 0; i < 3; i++) { const c = mesh(new T.ConeGeometry(0.07, 0.2, 5), combHex, false); c.position.set((i - 1) * 0.12, 0.03, 0); cg.add(c); } }
  else { blob(0, 1.1); blob(-0.08, 0.7); blob(0.08, 0.7); }
  const wattle = mesh(new T.SphereGeometry(0.07, 8, 8), combHex, false); wattle.scale.y = 1.4; wattle.position.set(0, -0.22, 0.3); head.add(wattle);
  // Parts too small to read from the social camera; hidden for distant chickens to keep crowds cheap.
  const detail = [wattle];

  parts.wings = [-1, 1].map((sx) => {
    const w = mesh(new T.SphereGeometry(0.22, 10, 8), hex);
    w.scale.set(0.45, 0.9, 1); w.position.set(sx * 0.46, 0.55, 0); rig.add(w);
    return w;
  });
  parts.legs = [-1, 1].map((sx) => {
    const leg = new T.Group(); leg.position.set(sx * 0.17, 0.28, 0.02); rig.add(leg);
    const shin = mesh(new T.CylinderGeometry(0.035, 0.035, 0.28, 6), 0xe8922b, false); shin.position.y = -0.14; leg.add(shin);
    for (let toe = -1; toe <= 1; toe++) {
      const foot = mesh(new T.SphereGeometry(0.04, 8, 6), 0xe8922b, false);
      foot.scale.set(0.8, 0.5, 2.4); foot.position.set(toe * 0.05, -0.26, 0.09); foot.rotation.y = toe * 0.3; leg.add(foot);
      if (toe) detail.push(foot);
    }
    return leg;
  });

  parts.slots = {};
  for (const s of ['head', 'eye', 'neck']) { parts.slots[s] = new T.Group(); head.add(parts.slots[s]); }
  for (const s of ['body', 'back']) { parts.slots[s] = new T.Group(); rig.add(parts.slots[s]); }
  head.traverse((o) => { if (o.isMesh && o.geometry.parameters?.radius < 0.02) detail.push(o); }); // eye glints
  parts.detail = detail;
  g.userData.parts = parts;
  return g;
}

// Sáu kiểu mắt (mục 5): tròn, to, cười, mi cong, ngôi sao, ngái ngủ.
function eyeMesh(kind, sx) {
  const g = new T.Group();
  g.position.set(sx * 0.14, 0.06, 0.3);
  const black = 0x201510;
  if (kind === 'to') {
    g.add(mesh(new T.SphereGeometry(0.08, 10, 8), 0xffffff, false));
    const p = mesh(new T.SphereGeometry(0.05, 8, 8), black, false); p.position.z = 0.04; g.add(p);
  } else if (kind === 'cuoi') {
    const a = mesh(new T.TorusGeometry(0.05, 0.016, 6, 10, Math.PI), black, false); a.position.z = 0.02; g.add(a);
  } else if (kind === 'mi') {
    g.add(mesh(new T.SphereGeometry(0.055, 8, 8), black, false));
    const l = mesh(new T.BoxGeometry(0.1, 0.018, 0.02), black, false); l.position.set(sx * 0.02, 0.06, 0.02); l.rotation.z = sx * 0.4; g.add(l);
  } else if (kind === 'sao') {
    const st = mesh(new T.OctahedronGeometry(0.06), 0xf2b51a, false); st.scale.z = 0.4; g.add(st);
    const p = mesh(new T.SphereGeometry(0.025, 6, 6), black, false); p.position.z = 0.025; g.add(p);
  } else if (kind === 'ngai') {
    const e = mesh(new T.SphereGeometry(0.055, 8, 8), black, false); e.scale.y = 0.35; g.add(e);
  } else {
    g.add(mesh(new T.SphereGeometry(0.055, 10, 8), black, false));
    const glint = mesh(new T.SphereGeometry(0.017, 6, 6), 0xfff9df, false);
    glint.position.set(-0.013, 0.018, 0.045); g.add(glint);
  }
  return g;
}

// Trang bị theo slot; ô thân trống thì mặc áo khởi điểm của nhân vật.
export function dress(chk, equipped, char, cosmetics) {
  const p = chk.userData.parts;
  for (const s in p.slots) { const grp = p.slots[s]; while (grp.children.length) grp.remove(grp.children[0]); }
  const byId = (id) => cosmetics.find((c) => c.id === id);
  for (const slot of ['head', 'eye', 'neck', 'body', 'back']) {
    const c = equipped && equipped[slot] && byId(equipped[slot]);
    if (c) addCosmetic(p.slots[slot], c);
  }
  if (!(equipped && equipped.body) && char && SHIRT[char.shirt] != null) {
    addCosmetic(p.slots.body, { kind: 'shirt', color: SHIRT[char.shirt] });
  }
}

function addCosmetic(grp, c) {
  const col = typeof c.color === 'number' ? c.color : hexOf(c.color || '#888888');
  const add = (m, x, y, z) => { m.position.set(x, y, z); grp.add(m); return m; };
  switch (c.kind) {
    case 'straw':
      add(mesh(new T.CylinderGeometry(0.42, 0.42, 0.03, 16), 0xd9b65a), 0, 0.33, 0);
      add(mesh(new T.CylinderGeometry(0.2, 0.24, 0.22, 16), 0xcaa64a), 0, 0.45, 0);
      add(mesh(new T.CylinderGeometry(0.245, 0.245, 0.05, 16), 0x7b4f2d), 0, 0.37, 0);
      break;
    case 'cap':
      add(mesh(new T.SphereGeometry(0.33, 14, 10, 0, Math.PI * 2, 0, Math.PI / 2), col), 0, 0.3, 0);
      add(mesh(new T.BoxGeometry(0.34, 0.03, 0.24), 0xb33a2b), 0, 0.3, 0.3);
      break;
    case 'tophat':
      add(mesh(new T.CylinderGeometry(0.36, 0.36, 0.03, 16), col), 0, 0.33, 0);
      add(mesh(new T.CylinderGeometry(0.2, 0.2, 0.38, 16), col), 0, 0.53, 0);
      add(mesh(new T.CylinderGeometry(0.205, 0.205, 0.06, 16), 0xb33a2b), 0, 0.38, 0);
      break;
    case 'crown': {
      add(mesh(new T.CylinderGeometry(0.28, 0.28, 0.1, 12), col), 0, 0.38, 0);
      for (let i = 0; i < 6; i++) { const a = (i / 6) * Math.PI * 2; add(mesh(new T.ConeGeometry(0.05, 0.14, 5), 0xffe680), Math.cos(a) * 0.26, 0.5, Math.sin(a) * 0.26); }
      break;
    }
    case 'flower':
      for (let i = 0; i < 5; i++) { const a = (i / 5) * Math.PI * 2; add(mesh(new T.SphereGeometry(0.07, 8, 6), col), 0.22 + Math.cos(a) * 0.08, 0.25 + Math.sin(a) * 0.08, 0.12); }
      add(mesh(new T.SphereGeometry(0.06, 8, 6), 0xffe066), 0.22, 0.25, 0.14);
      break;
    case 'shades':
      add(mesh(new T.BoxGeometry(0.42, 0.11, 0.05), col), 0, 0.06, 0.32);
      break;
    case 'round':
      for (const sx of [-1, 1]) add(mesh(new T.TorusGeometry(0.08, 0.02, 8, 14), col), sx * 0.14, 0.06, 0.33);
      add(mesh(new T.BoxGeometry(0.1, 0.02, 0.02), col), 0, 0.07, 0.34);
      break;
    case 'heart':
      for (const sx of [-1, 1]) {
        const h = add(mesh(new T.SphereGeometry(0.075, 8, 6), col), sx * 0.14, 0.06, 0.33); h.scale.set(1.2, 1, 0.4);
      }
      break;
    case 'bow':
      for (const sx of [-1, 1]) { const w = add(mesh(new T.ConeGeometry(0.09, 0.16, 4), col), sx * 0.09, -0.3, 0.26); w.rotation.z = sx * Math.PI / 2; }
      add(mesh(new T.SphereGeometry(0.045, 6, 6), col), 0, -0.3, 0.28);
      break;
    case 'scarf':
      add(mesh(new T.TorusGeometry(0.27, 0.07, 8, 18), col), 0, -0.28, 0).rotation.x = Math.PI / 2;
      add(mesh(new T.BoxGeometry(0.1, 0.24, 0.05), col), 0.1, -0.42, 0.25);
      break;
    case 'shirt':
      add(mesh(new T.CylinderGeometry(0.52, 0.55, 0.4, 16), col), 0, 0.44, 0);
      break;
    case 'vest':
      add(mesh(new T.CylinderGeometry(0.53, 0.56, 0.44, 16), col), 0, 0.46, 0);
      add(mesh(new T.BoxGeometry(0.14, 0.34, 0.06), 0xf4f0e6), 0, 0.5, 0.5);
      add(mesh(new T.ConeGeometry(0.05, 0.14, 4), 0xb33a2b), 0, 0.55, 0.54).rotation.x = Math.PI;
      break;
    case 'dress':
      add(mesh(new T.CylinderGeometry(0.5, 0.55, 0.32, 16), col), 0, 0.5, 0);
      add(mesh(new T.ConeGeometry(0.74, 0.5, 16, 1, true), col), 0, 0.22, 0);
      break;
    case 'beanie':
      add(mesh(new T.SphereGeometry(0.34, 14, 10, 0, Math.PI * 2, 0, Math.PI / 2), col), 0, 0.22, 0);
      add(mesh(new T.CylinderGeometry(0.345, 0.345, 0.08, 14), 0xf4f0e6), 0, 0.24, 0);
      add(mesh(new T.SphereGeometry(0.08, 8, 6), 0xf4f0e6), 0, 0.58, 0);
      break;
    case 'overall':
      add(mesh(new T.CylinderGeometry(0.53, 0.56, 0.36, 16), col), 0, 0.4, 0);
      add(mesh(new T.BoxGeometry(0.32, 0.22, 0.06), col), 0, 0.64, 0.46);
      for (const sx of [-1, 1]) add(mesh(new T.BoxGeometry(0.06, 0.3, 0.06), col), sx * 0.16, 0.74, 0.4);
      break;
    case 'pack':
      add(mesh(new T.BoxGeometry(0.42, 0.42, 0.22), col), 0, 0.6, -0.55);
      add(mesh(new T.BoxGeometry(0.3, 0.14, 0.06), 0x5c3a1f), 0, 0.5, -0.67);
      break;
    case 'wings':
      for (const sx of [-1, 1]) { const w = add(mesh(new T.SphereGeometry(0.3, 10, 8), col), sx * 0.3, 0.85, -0.45); w.scale.set(0.9, 0.5, 0.15); w.rotation.z = sx * 0.5; }
      break;
  }
}

export function buildPig() {
  const g = new T.Group();
  const body = mesh(new T.SphereGeometry(0.55, 14, 12), 0xf0a9b0); body.scale.set(1.1, 0.9, 1); body.position.y = 0.6; g.add(body);
  const head = mesh(new T.SphereGeometry(0.4, 14, 12), 0xf3b4bb); head.position.set(0, 1.05, 0.1); g.add(head);
  const snout = mesh(new T.CylinderGeometry(0.14, 0.14, 0.12, 10), 0xe8969e); snout.rotation.x = Math.PI / 2; snout.position.set(0, 1.0, 0.48); g.add(snout);
  for (const sx of [-1, 1]) {
    const ear = mesh(new T.ConeGeometry(0.12, 0.2, 5), 0xe8969e); ear.position.set(sx * 0.2, 1.38, 0.05); g.add(ear);
    const eye = mesh(new T.SphereGeometry(0.04, 8, 8), 0x201510, false); eye.position.set(sx * 0.14, 1.1, 0.37); g.add(eye);
  }
  const overall = mesh(new T.CylinderGeometry(0.57, 0.62, 0.55, 16), 0x3f6b4e); overall.position.y = 0.48; g.add(overall);
  const hat = mesh(new T.CylinderGeometry(0.46, 0.46, 0.03, 16), 0xd9b65a); hat.position.set(0, 1.42, 0.05); g.add(hat);
  const crown = mesh(new T.CylinderGeometry(0.24, 0.27, 0.2, 16), 0xcaa64a); crown.position.set(0, 1.52, 0.05); g.add(crown);
  return g;
}

export function buildDog() {
  const g = new T.Group();
  const body = mesh(new T.SphereGeometry(0.45, 14, 12), 0x4a3b33); body.scale.set(1.2, 0.85, 1); body.position.y = 0.55; g.add(body);
  const head = mesh(new T.SphereGeometry(0.33, 14, 12), 0x5a4a40); head.position.set(0, 0.98, 0.2); g.add(head);
  const snout = mesh(new T.BoxGeometry(0.2, 0.16, 0.22), 0x4a3b33); snout.position.set(0, 0.9, 0.48); g.add(snout);
  const nose = mesh(new T.SphereGeometry(0.05, 6, 6), 0x111111, false); nose.position.set(0, 0.94, 0.6); g.add(nose);
  for (const sx of [-1, 1]) {
    const ear = mesh(new T.ConeGeometry(0.12, 0.26, 5), 0x3a2e28); ear.rotation.z = sx * 0.3; ear.position.set(sx * 0.22, 1.24, 0.12); g.add(ear);
    const eye = mesh(new T.SphereGeometry(0.04, 8, 8), 0xffffff, false); eye.position.set(sx * 0.12, 1.04, 0.47); g.add(eye);
  }
  const whistle = mesh(new T.CylinderGeometry(0.04, 0.04, 0.16, 6), 0xd0d0d0); whistle.rotation.z = Math.PI / 2; whistle.position.set(0.15, 0.78, 0.45); g.add(whistle);
  return g;
}

// Bò Mộc: thợ xây hiền lành, yếm xanh, mũ thợ (ảnh "Khu nhà bạn bè").
export function buildCow() {
  const g = new T.Group();
  const body = mesh(new T.SphereGeometry(0.55, 14, 12), 0xf4f1ea); body.scale.set(1.05, 0.95, 0.95); body.position.y = 0.62; g.add(body);
  const head = mesh(new T.SphereGeometry(0.42, 14, 12), 0xf4f1ea); head.position.set(0, 1.22, 0.06); g.add(head);
  for (const [x, y, z, r] of [[0.32, 0.75, 0.25, 0.16], [-0.3, 0.55, -0.2, 0.14], [0.2, 1.38, -0.15, 0.1]]) { const sp = mesh(new T.SphereGeometry(r, 8, 6), 0x2b2b2b, false); sp.position.set(x, y, z); sp.scale.z = 0.5; g.add(sp); }
  const muzzle = mesh(new T.SphereGeometry(0.22, 10, 8), 0xf0b6bf); muzzle.scale.set(1.2, 0.8, 0.8); muzzle.position.set(0, 1.12, 0.38); g.add(muzzle);
  for (const sx of [-1, 1]) {
    const eye = mesh(new T.SphereGeometry(0.045, 8, 8), 0x201510, false); eye.position.set(sx * 0.15, 1.3, 0.36); g.add(eye);
    const horn = mesh(new T.ConeGeometry(0.05, 0.18, 6), 0xe8dcc0); horn.position.set(sx * 0.26, 1.6, 0); horn.rotation.z = -sx * 0.5; g.add(horn);
    const ear = mesh(new T.SphereGeometry(0.1, 8, 6), 0xf4f1ea); ear.scale.set(1.5, 0.6, 0.6); ear.position.set(sx * 0.45, 1.36, 0); g.add(ear);
  }
  const overall = mesh(new T.CylinderGeometry(0.57, 0.6, 0.55, 16), 0x2f5f9e); overall.position.y = 0.48; g.add(overall);
  const cap = mesh(new T.SphereGeometry(0.3, 12, 8, 0, Math.PI * 2, 0, Math.PI / 2), 0x2f5f9e); cap.position.set(0, 1.5, 0.04); g.add(cap);
  const hammer = mesh(new T.BoxGeometry(0.06, 0.4, 0.06), 0x7b4f2d); hammer.position.set(0.6, 0.8, 0.2); hammer.rotation.z = 0.5; g.add(hammer);
  const head2 = mesh(new T.BoxGeometry(0.2, 0.08, 0.1), 0x8a8a8a); head2.position.set(0.7, 0.98, 0.2); head2.rotation.z = 0.5; g.add(head2);
  return g;
}

const ROOF = { do: 0xb8433a, xanh: 0x3e6fa8, luc: 0x4f8a45, nau: 0x8a5a30 };

// Chuồng gỗ mái màu theo mẫu của Bò Mộc; state "building" là khung giàn đang dựng.
export function buildCoop(roof, state) {
  const g = new T.Group();
  const wood = 0x8a5a30, dark = 0x5c3a1f;
  if (state === 'building') {
    for (const [x, z] of [[-1.2, -0.8], [1.2, -0.8], [-1.2, 0.8], [1.2, 0.8]]) { const p = mesh(new T.BoxGeometry(0.12, 1.8, 0.12), 0xc9a36a); p.position.set(x, 0.9, z); g.add(p); }
    for (const y of [0.9, 1.8]) for (const z of [-0.8, 0.8]) { const b = mesh(new T.BoxGeometry(2.5, 0.1, 0.1), 0xc9a36a); b.position.set(0, y, z); g.add(b); }
    for (let i = 0; i < 3; i++) { const plank = mesh(new T.BoxGeometry(1.1, 0.08, 0.25), 0xc9a36a); plank.position.set(1.8, 0.05 + i * 0.09, 1.2); plank.rotation.y = 0.2; g.add(plank); }
    return g;
  }
  const bodyM = mesh(new T.BoxGeometry(2.4, 1.4, 1.7), wood); bodyM.position.y = 0.85; bodyM.receiveShadow = true; g.add(bodyM);
  for (let i = 0; i < 4; i++) { const plank = mesh(new T.BoxGeometry(2.42, 0.04, 1.72), dark, false); plank.position.y = 0.4 + i * 0.32; g.add(plank); }
  const door = mesh(new T.BoxGeometry(0.7, 0.75, 0.05), 0x2e1f12, false); door.position.set(0, 0.6, 0.86); g.add(door);
  const ramp = mesh(new T.BoxGeometry(0.6, 0.06, 0.8), 0xa47447); ramp.position.set(0, 0.2, 1.2); ramp.rotation.x = 0.4; g.add(ramp);
  for (const sx of [-1, 1]) {
    const r = mesh(new T.BoxGeometry(1.45, 0.1, 2.1), ROOF[roof] || ROOF.do); r.position.set(sx * 0.6, 1.85, 0); r.rotation.z = -sx * 0.5; g.add(r);
  }
  return g;
}

// Đồ trang trí đặt trong chuồng riêng.
export function buildDecor(kind, ownerName) {
  const g = new T.Group();
  const add = (m, x, y, z) => { m.position.set(x, y, z); g.add(m); return m; };
  switch (kind) {
    case 'o_rom': {
      const n = add(mesh(new T.TorusGeometry(0.45, 0.17, 8, 16), 0xd9b65a), 0, 0.17, 0); n.rotation.x = Math.PI / 2;
      add(mesh(new T.CylinderGeometry(0.38, 0.38, 0.05, 14), 0xb8913f), 0, 0.04, 0);
      break;
    }
    case 'mang_thoc':
      add(mesh(new T.BoxGeometry(0.5, 0.3, 1.0), 0x6b4423), 0, 0.15, 0);
      add(mesh(new T.BoxGeometry(0.38, 0.06, 0.88), 0xe7a73a), 0, 0.3, 0);
      break;
    case 'luong_rau': case 'luong_ca_rot':
      add(mesh(new T.BoxGeometry(0.9, 0.12, 0.9), 0x5a3a20), 0, 0.06, 0);
      for (const [x, z] of [[-0.22, -0.22], [0.22, -0.22], [-0.22, 0.22], [0.22, 0.22]]) {
        if (kind === 'luong_rau') add(mesh(new T.IcosahedronGeometry(0.15), 0x6fbf4a), x, 0.22, z);
        else { add(mesh(new T.ConeGeometry(0.06, 0.2, 6), 0xe9833a), x, 0.16, z).rotation.x = Math.PI; add(mesh(new T.ConeGeometry(0.08, 0.2, 5), 0x4f9a3a), x, 0.32, z); }
      }
      break;
    case 'chau_hoa':
      add(mesh(new T.CylinderGeometry(0.2, 0.15, 0.28, 10), 0xb5643c), 0, 0.14, 0);
      for (let i = 0; i < 5; i++) { const a = (i / 5) * Math.PI * 2; add(mesh(new T.SphereGeometry(0.08, 8, 6), 0xe2577a), Math.cos(a) * 0.1, 0.4, Math.sin(a) * 0.1); }
      add(mesh(new T.SphereGeometry(0.06, 6, 6), 0xffe066), 0, 0.42, 0);
      break;
    case 'thung_go':
      add(mesh(new T.CylinderGeometry(0.32, 0.28, 0.65, 12), 0x7b4f2d), 0, 0.33, 0);
      add(mesh(new T.CylinderGeometry(0.28, 0.28, 0.03, 12), 0x8ec6d9), 0, 0.66, 0);
      break;
    case 'cay_xanh':
      add(mesh(new T.CylinderGeometry(0.1, 0.14, 0.7, 7), 0x6b4423), 0, 0.35, 0);
      add(mesh(new T.IcosahedronGeometry(0.55), 0x3f7d3a), 0, 1.0, 0);
      break;
    case 'den_long':
      add(mesh(new T.BoxGeometry(0.06, 1.2, 0.06), 0x5c3a1f), 0, 0.6, 0);
      add(mesh(new T.BoxGeometry(0.4, 0.05, 0.05), 0x5c3a1f), 0.15, 1.18, 0);
      add(new T.Mesh(new T.SphereGeometry(0.15, 10, 8), mat(0xe2452e, { emissive: 0x7a1a0a })), 0.3, 0.98, 0).scale.y = 1.2;
      break;
    case 'ban_tiec':
      add(mesh(new T.CylinderGeometry(0.45, 0.45, 0.06, 14), 0xc99a5b), 0, 0.5, 0);
      add(mesh(new T.CylinderGeometry(0.06, 0.1, 0.48, 6), 0x7b4f2d), 0, 0.24, 0);
      add(mesh(new T.CylinderGeometry(0.16, 0.16, 0.16, 12), 0xf4e1d2), 0, 0.61, 0);
      add(mesh(new T.CylinderGeometry(0.165, 0.165, 0.04, 12), 0xe2577a), 0, 0.7, 0);
      break;
    case 'bang_ten': {
      add(mesh(new T.BoxGeometry(0.08, 0.9, 0.08), 0x6b4423), 0, 0.45, 0);
      add(mesh(new T.BoxGeometry(0.9, 0.4, 0.06), 0x9c6b36), 0, 0.85, 0);
      if (ownerName) { const l = label(ownerName, { bg: 'rgba(92,58,31,.9)', scale: 0.35 }); l.position.set(0, 0.85, 0.06); g.add(l); }
      break;
    }
  }
  g.traverse((o) => { if (o.isMesh) o.castShadow = true; });
  return g;
}

export function buildElder(cosmetics) {
  const g = buildChicken({ color: 'kem' });
  g.scale.setScalar(1.25);
  dress(g, { head: 'mu_rom' }, null, cosmetics);
  return g;
}

// Nhãn nổi bằng canvas sprite (tên người chơi, NPC, cổng).
const labelCache = new Map();
export function label(text, { bg = 'rgba(61,43,26,.86)', fg = '#fff', scale = 0.9 } = {}) {
  const key = text + bg + fg;
  let tex = labelCache.get(key);
  let w;
  if (!tex) {
    const cv = document.createElement('canvas');
    const ctx = cv.getContext('2d');
    ctx.font = 'bold 40px system-ui, sans-serif';
    w = Math.ceil(ctx.measureText(text).width) + 44;
    cv.width = w; cv.height = 72;
    ctx.font = 'bold 40px system-ui, sans-serif';
    ctx.textBaseline = 'middle';
    ctx.fillStyle = bg;
    roundRect(ctx, 2, 8, w - 4, 56, 18); ctx.fill();
    ctx.fillStyle = fg; ctx.fillText(text, 22, 37);
    tex = new T.CanvasTexture(cv);
    tex.userData = { w };
    labelCache.set(key, tex);
  }
  w = tex.userData.w;
  const sp = new T.Sprite(new T.SpriteMaterial({ map: tex, depthTest: false, depthWrite: false, transparent: true }));
  sp.scale.set((w / 72) * scale, scale, 1);
  sp.renderOrder = 10;
  return sp;
}
function roundRect(ctx, x, y, w, h, r) {
  ctx.beginPath(); ctx.moveTo(x + r, y); ctx.arcTo(x + w, y, x + w, y + h, r); ctx.arcTo(x + w, y + h, x, y + h, r);
  ctx.arcTo(x, y + h, x, y, r); ctx.arcTo(x, y, x + w, y, r); ctx.closePath();
}

export function wormMesh(color) {
  const g = new T.Group();
  const col = hexOf(color);
  for (let i = 0; i < 6; i++) {
    const s = mesh(new T.SphereGeometry(0.07 - i * 0.004, 8, 6), col);
    s.position.set(Math.sin(i * 0.9) * 0.06, i * 0.09, 0);
    g.add(s);
  }
  return g;
}

// ---------- V1.1/V1.2: NPC mới, gà con đồng hành, phương tiện ----------

// Vịt Bông: chăm trứng ở trạm ấp (ảnh 6).
export function buildDuck() {
  const g = new T.Group();
  const body = mesh(new T.SphereGeometry(0.5, 14, 12), 0xf5f2e9); body.scale.set(1, 0.85, 1.2); body.position.y = 0.5; g.add(body);
  const head = mesh(new T.SphereGeometry(0.32, 14, 12), 0xf5f2e9); head.position.set(0, 1.0, 0.25); g.add(head);
  const bill = mesh(new T.BoxGeometry(0.3, 0.08, 0.3), 0xf0a23a); bill.position.set(0, 0.95, 0.58); g.add(bill);
  for (const sx of [-1, 1]) { const eye = mesh(new T.SphereGeometry(0.045, 8, 8), 0x201510, false); eye.position.set(sx * 0.13, 1.06, 0.5); g.add(eye); }
  const scarf = mesh(new T.TorusGeometry(0.26, 0.07, 8, 16), 0xe2577a); scarf.rotation.x = Math.PI / 2; scarf.position.set(0, 0.78, 0.18); g.add(scarf);
  const apron = mesh(new T.BoxGeometry(0.6, 0.45, 0.06), 0x8ec6d9); apron.position.set(0, 0.48, 0.6); g.add(apron);
  return g;
}

// Chim Chíp: người đưa thư và điều phối bến xe (ảnh 9).
export function buildBird() {
  const g = new T.Group();
  const post = mesh(new T.CylinderGeometry(0.08, 0.1, 1.0, 8), 0x6b4423); post.position.y = 0.5; g.add(post);
  const body = mesh(new T.SphereGeometry(0.32, 12, 10), 0x6fb3e0); body.scale.set(1, 0.9, 1.1); body.position.y = 1.3; g.add(body);
  const head = mesh(new T.SphereGeometry(0.22, 12, 10), 0x6fb3e0); head.position.set(0, 1.62, 0.12); g.add(head);
  const beak = mesh(new T.ConeGeometry(0.06, 0.16, 6), 0xf0a23a); beak.rotation.x = Math.PI / 2; beak.position.set(0, 1.6, 0.36); g.add(beak);
  for (const sx of [-1, 1]) { const eye = mesh(new T.SphereGeometry(0.035, 6, 6), 0x201510, false); eye.position.set(sx * 0.09, 1.67, 0.3); g.add(eye); }
  const cap = mesh(new T.CylinderGeometry(0.16, 0.18, 0.12, 12), 0x2f5f9e); cap.position.set(0, 1.82, 0.08); g.add(cap);
  const bag = mesh(new T.BoxGeometry(0.22, 0.18, 0.1), 0x9c6b36); bag.position.set(0.28, 1.22, 0.05); g.add(bag);
  return g;
}

// Dê Thợ Máy: bán và sửa xe (ảnh 10).
export function buildGoat() {
  const g = new T.Group();
  const body = mesh(new T.SphereGeometry(0.5, 14, 12), 0xe8e2d6); body.scale.set(1.05, 0.95, 1); body.position.y = 0.6; g.add(body);
  const head = mesh(new T.SphereGeometry(0.33, 12, 10), 0xe8e2d6); head.position.set(0, 1.15, 0.12); g.add(head);
  const muzzle = mesh(new T.BoxGeometry(0.24, 0.18, 0.22), 0xd4cbbb); muzzle.position.set(0, 1.07, 0.4); g.add(muzzle);
  const beard = mesh(new T.ConeGeometry(0.06, 0.2, 5), 0xbdb3a1); beard.rotation.x = Math.PI; beard.position.set(0, 0.9, 0.4); g.add(beard);
  for (const sx of [-1, 1]) {
    const horn = mesh(new T.ConeGeometry(0.05, 0.28, 6), 0x8a7a64); horn.position.set(sx * 0.16, 1.48, 0.02); horn.rotation.z = -sx * 0.35; horn.rotation.x = -0.3; g.add(horn);
    const eye = mesh(new T.SphereGeometry(0.04, 6, 6), 0x201510, false); eye.position.set(sx * 0.13, 1.2, 0.38); g.add(eye);
  }
  const overall = mesh(new T.CylinderGeometry(0.52, 0.56, 0.5, 14), 0xd64b3a); overall.position.y = 0.5; g.add(overall);
  const wrench = mesh(new T.BoxGeometry(0.05, 0.4, 0.05), 0x9a9a9a); wrench.position.set(0.58, 0.75, 0.2); wrench.rotation.z = 0.4; g.add(wrench);
  return g;
}

// Ếch Xanh: hướng dẫn ở trại Ven Suối (ảnh 11).
export function buildFrog() {
  const g = new T.Group();
  const body = mesh(new T.SphereGeometry(0.45, 14, 10), 0x5fae4a); body.scale.set(1.2, 0.75, 1); body.position.y = 0.38; g.add(body);
  for (const sx of [-1, 1]) {
    const eyeBall = mesh(new T.SphereGeometry(0.13, 10, 8), 0x5fae4a); eyeBall.position.set(sx * 0.2, 0.75, 0.2); g.add(eyeBall);
    const pupil = mesh(new T.SphereGeometry(0.06, 8, 6), 0x201510, false); pupil.position.set(sx * 0.2, 0.8, 0.3); g.add(pupil);
    const leg = mesh(new T.SphereGeometry(0.16, 8, 6), 0x4f9a3a); leg.scale.set(1, 0.5, 1.4); leg.position.set(sx * 0.45, 0.12, 0); g.add(leg);
  }
  const hat = mesh(new T.ConeGeometry(0.42, 0.25, 14), 0xd9b65a); hat.position.set(0, 0.95, 0.05); g.add(hat);
  return g;
}

// Gà con đồng hành (mục 12).
export function buildChick(color = '#f4cf3f') {
  const g = new T.Group();
  const hex = hexOf(color);
  const body = mesh(new T.SphereGeometry(0.22, 12, 10), hex); body.position.y = 0.22; g.add(body);
  const head = mesh(new T.SphereGeometry(0.15, 12, 10), hex); head.position.set(0, 0.46, 0.06); g.add(head);
  const beak = mesh(new T.ConeGeometry(0.04, 0.09, 5), 0xe8922b); beak.rotation.x = Math.PI / 2; beak.position.set(0, 0.45, 0.22); g.add(beak);
  for (const sx of [-1, 1]) { const eye = mesh(new T.SphereGeometry(0.025, 6, 6), 0x201510, false); eye.position.set(sx * 0.06, 0.49, 0.18); g.add(eye); }
  for (const sx of [-1, 1]) { const f = mesh(new T.BoxGeometry(0.08, 0.02, 0.1), 0xe8922b); f.position.set(sx * 0.07, 0.01, 0.03); g.add(f); }
  return g;
}

function wheel(r, w) {
  const m = mesh(new T.CylinderGeometry(r, r, w, 14), 0x2b2b2b);
  m.rotation.z = Math.PI / 2;
  return m;
}

// Xe cá nhân (mục 8.4). Gốc toạ độ ở vị trí người lái; trục z hướng về phía trước.
export function buildVehicle(kind) {
  const g = new T.Group();
  const add = (m, x, y, z) => { m.position.set(x, y, z); g.add(m); return m; };
  if (kind === 'xe_dap') {
    add(wheel(0.32, 0.06), 0, 0.32, 0.55);
    add(wheel(0.32, 0.06), 0, 0.32, -0.75);
    add(mesh(new T.BoxGeometry(0.06, 0.06, 1.3), 0x3f86c9), 0, 0.55, -0.1);
    add(mesh(new T.BoxGeometry(0.4, 0.05, 0.05), 0x2b2b2b), 0, 0.95, 0.5);
    add(mesh(new T.BoxGeometry(0.28, 0.06, 0.32), 0x7b4f2d), 0, 0.6, -0.72); // ghế sau chở bạn
  } else if (kind === 'xe_may') {
    add(wheel(0.3, 0.12), 0, 0.3, 0.7);
    add(wheel(0.3, 0.12), 0, 0.3, -0.8);
    add(mesh(new T.BoxGeometry(0.4, 0.35, 1.5), 0xd64b3a), 0, 0.55, -0.05);
    add(mesh(new T.BoxGeometry(0.36, 0.1, 1.1), 0x2b2b2b), 0, 0.77, -0.3);
    add(mesh(new T.BoxGeometry(0.55, 0.05, 0.05), 0x9a9a9a), 0, 1.0, 0.65);
    add(mesh(new T.SphereGeometry(0.09, 8, 6), 0xffe066), 0, 0.8, 0.85);
  } else if (kind === 'o_to') {
    const body = add(mesh(new T.BoxGeometry(1.6, 0.55, 2.5), 0xe7a73a), 0, 0.55, -0.2);
    body.receiveShadow = true;
    add(mesh(new T.BoxGeometry(1.4, 0.5, 1.4), 0xf2c26a), 0, 1.05, -0.35);
    add(mesh(new T.BoxGeometry(1.42, 0.38, 0.05), 0x9ed3e8), 0, 1.07, 0.36);
    for (const [x, z] of [[-0.75, 0.6], [0.75, 0.6], [-0.75, -1.0], [0.75, -1.0]]) add(wheel(0.28, 0.18), x, 0.28, z);
    for (const sx of [-1, 1]) add(mesh(new T.SphereGeometry(0.09, 8, 6), 0xffe066), sx * 0.55, 0.6, 1.06);
  }
  g.traverse((o) => { if (o.isMesh) o.castShadow = true; });
  return g;
}

// Xe buýt tuyến Chim Chíp (ảnh 9): tài xế NPC riêng, 8 ghế khách.
export function buildBus() {
  const g = new T.Group();
  const add = (m, x, y, z) => { m.position.set(x, y, z); g.add(m); return m; };
  add(mesh(new T.BoxGeometry(2.0, 1.4, 4.4), 0xe7c23a), 0, 1.05, 0);
  add(mesh(new T.BoxGeometry(2.02, 0.5, 3.6), 0x9ed3e8), 0, 1.35, -0.2);
  add(mesh(new T.BoxGeometry(2.04, 0.12, 4.42), 0x3f86c9), 0, 0.6, 0);
  add(mesh(new T.BoxGeometry(1.9, 0.1, 4.3), 0xd6aa2a), 0, 1.8, 0);
  for (const [x, z] of [[-1, 1.4], [1, 1.4], [-1, -1.4], [1, -1.4]]) add(wheel(0.36, 0.24), x, 0.36, z);
  const sign = label('🐦 Chim Chíp', { bg: 'rgba(63,134,201,.9)', scale: 0.55 });
  sign.position.set(0, 2.15, 0); g.add(sign);
  g.traverse((o) => { if (o.isMesh) o.castShadow = true; });
  return g;
}

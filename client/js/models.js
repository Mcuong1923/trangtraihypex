// Bộ nhân vật và phương tiện bo-2: hình khối mềm, vật liệu mờ, rig giữ tương thích gameplay.
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

// Shared bevelled forms keep small props/vehicle panels from looking like raw cubes.
function softBox(w, h, d, radius, color) {
  const r = Math.min(radius, w / 2, h / 2, d / 2), geo = new T.BoxGeometry(w, h, d, 3, 3, 3);
  const p = geo.attributes.position, n = geo.attributes.normal;
  for (let i = 0; i < p.count; i++) {
    const x = p.getX(i), y = p.getY(i), z = p.getZ(i);
    const cx = T.MathUtils.clamp(x, -w / 2 + r, w / 2 - r), cy = T.MathUtils.clamp(y, -h / 2 + r, h / 2 - r), cz = T.MathUtils.clamp(z, -d / 2 + r, d / 2 - r);
    const v = new T.Vector3(x - cx, y - cy, z - cz).normalize();
    p.setXYZ(i, cx + v.x * r, cy + v.y * r, cz + v.z * r); n.setXYZ(i, v.x, v.y, v.z);
  }
  geo.parameters.bevel = r;
  return mesh(geo, color);
}
function link(g, a, b, radius, color) {
  const start = new T.Vector3(...a), end = new T.Vector3(...b), diff = end.clone().sub(start);
  const m = mesh(new T.CylinderGeometry(radius, radius, diff.length(), 8), color);
  m.position.copy(start).add(end).multiplyScalar(.5); m.quaternion.setFromUnitVectors(new T.Vector3(0, 1, 0), diff.normalize()); g.add(m); return m;
}
function cloth(color, hem = .19, top = .73, skirt = false) {
  const points = [[.43, hem], [.53, hem + .08], [.56, .42], [.51, .59], [.38, top]].map(([r, y]) => new T.Vector2(skirt && y < .35 ? r * 1.2 : r, y));
  const m = mesh(new T.LatheGeometry(points, 24), color); m.scale.z = 1.05; return m;
}

export function buildChicken({ color = 'vang', comb = 'don', tail: tailKind = 'quat', eyes = 'tron', bodyHex } = {}) {
  const hex = bodyHex ?? FEATHER[color] ?? FEATHER.vang;
  const g = new T.Group();
  const rig = new T.Group(); // xoay/nhún cả thân khi biểu cảm, KO
  g.add(rig);
  const parts = { rig };

  const profile = [[0,.16],[.25,.18],[.43,.25],[.53,.4],[.55,.57],[.5,.76],[.4,.94],[.28,1.08],[0,1.1]].map(([r,y]) => new T.Vector2(r,y));
  const body = mesh(new T.LatheGeometry(profile, 28), hex);
  body.scale.z = 1.06; rig.add(body);
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

  const head = new T.Group(); head.position.set(0, 1.06, 0.14); rig.add(head); parts.head = head;
  head.add(mesh(new T.SphereGeometry(0.35, 24, 18), hex));
  for (const sx of [-1, 1]) head.add(eyeMesh(eyes, sx));
  const beak = mesh(new T.ConeGeometry(0.105, 0.19, 12), 0xe8922b, false);
  beak.rotation.x = Math.PI / 2; beak.position.set(0, -0.02, 0.37); head.add(beak);

  const combHex = 0xe0401f;
  const cg = new T.Group(); cg.position.set(0, 0.33, 0.05); head.add(cg);
  const blob = (x, s) => { const b = mesh(new T.SphereGeometry(0.1 * s, 12, 10), combHex, false); b.position.x = x; b.scale.y = 1.3; cg.add(b); };
  if (comb === 'kep') { blob(-0.09, 1); blob(0.09, 1); blob(0, 0.8); }
  else if (comb === 'ro') { for (let i = 0; i < 3; i++) { const c = mesh(new T.ConeGeometry(0.07, 0.2, 5), combHex, false); c.position.set((i - 1) * 0.12, 0.03, 0); cg.add(c); } }
  else { blob(0, 1.1); blob(-0.08, 0.7); blob(0.08, 0.7); }
  const wattle = mesh(new T.SphereGeometry(0.07, 8, 8), combHex, false); wattle.scale.y = 1.4; wattle.position.set(0, -0.22, 0.3); head.add(wattle);
  // Parts too small to read from the social camera; hidden for distant chickens to keep crowds cheap.
  const detail = [wattle];

  parts.wings = [-1, 1].map((sx) => {
    const w = mesh(new T.SphereGeometry(0.24, 16, 12), hex);
    w.scale.set(0.48, 1.05, 1.08); w.position.set(sx * 0.5, 0.58, .06); w.rotation.x = -.2; rig.add(w);
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
      grp.add(cloth(col));
      break;
    case 'vest':
      grp.add(cloth(col, .18, .78));
      add(mesh(new T.BoxGeometry(0.14, 0.34, 0.06), 0xf4f0e6), 0, 0.5, 0.5);
      add(mesh(new T.ConeGeometry(0.05, 0.14, 4), 0xb33a2b), 0, 0.55, 0.54).rotation.x = Math.PI;
      break;
    case 'dress':
      grp.add(cloth(col, .14, .78, true));
      break;
    case 'beanie':
      add(mesh(new T.SphereGeometry(0.34, 14, 10, 0, Math.PI * 2, 0, Math.PI / 2), col), 0, 0.22, 0);
      add(mesh(new T.CylinderGeometry(0.345, 0.345, 0.08, 14), 0xf4f0e6), 0, 0.24, 0);
      add(mesh(new T.SphereGeometry(0.08, 8, 6), 0xf4f0e6), 0, 0.58, 0);
      break;
    case 'overall':
      grp.add(cloth(col, .18, .68));
      add(mesh(new T.BoxGeometry(0.32, 0.22, 0.06), col), 0, 0.64, 0.46);
      for (const sx of [-1, 1]) add(mesh(new T.BoxGeometry(0.06, 0.3, 0.06), col), sx * 0.16, 0.74, 0.4);
      break;
    case 'pack':
      add(softBox(.48, .46, .23, .06, col), 0, .62, -.53);
      add(softBox(.46, .15, .055, .025, col), 0, .78, -.66);
      add(softBox(.07, .09, .025, .015, 0xd8b05b), 0, .68, -.69);
      break;
    case 'wings':
      for (const sx of [-1, 1]) { const w = add(mesh(new T.SphereGeometry(0.3, 10, 8), col), sx * 0.3, 0.85, -0.45); w.scale.set(0.9, 0.5, 0.15); w.rotation.z = sx * 0.5; }
      break;
  }
}

export function buildPig() {
  const g = new T.Group();
  const body = mesh(new T.SphereGeometry(0.55, 24, 16), 0xefa1a0); body.scale.set(1.1, 0.9, 1); body.position.y = 0.6; g.add(body);
  const head = mesh(new T.SphereGeometry(0.4, 24, 16), 0xf3b0a6); head.position.set(0, 1.05, 0.1); g.add(head);
  const snout = mesh(new T.CylinderGeometry(0.14, 0.14, 0.12, 10), 0xe8969e); snout.rotation.x = Math.PI / 2; snout.position.set(0, 1.0, 0.48); g.add(snout);
  for (const sx of [-1, 1]) {
    const ear = mesh(new T.ConeGeometry(0.12, 0.2, 5), 0xe8969e); ear.position.set(sx * 0.2, 1.38, 0.05); g.add(ear);
    const eye = mesh(new T.SphereGeometry(0.04, 8, 8), 0x201510, false); eye.position.set(sx * 0.14, 1.1, 0.37); g.add(eye);
  }
  const overall = cloth(0x546c3d, .19, .8); g.add(overall);
  const bib = softBox(.35,.27,.06,.035,0x546c3d); bib.position.set(0,.77,.46); g.add(bib);
  for (const sx of [-1,1]) {
    const arm = mesh(new T.SphereGeometry(.15,16,12),0xefa1a0); arm.scale.set(.8,1.5,.9); arm.position.set(sx*.56,.65,.12); arm.rotation.z=-sx*.3; g.add(arm);
    const foot = softBox(.18,.15,.27,.04,0x875b4b); foot.position.set(sx*.23,.09,.13); g.add(foot);
    const button=mesh(new T.SphereGeometry(.035,8,6),0xd9ad55,false); button.position.set(sx*.13,.83,.5);g.add(button);
    const nostril=mesh(new T.SphereGeometry(.025,8,6),0x9f5e54,false);nostril.position.set(sx*.055,1.015,.548);g.add(nostril);
  }
  return g;
}

export function buildDog() {
  const g = new T.Group(), fur = 0x302c28;
  const body = mesh(new T.SphereGeometry(.4,24,16),fur); body.scale.set(.95,1.15,1); body.position.set(0,.53,-.03);g.add(body);
  const head = mesh(new T.SphereGeometry(.33,24,16),fur);head.position.set(0,1.04,.18);g.add(head);
  for(const sx of [-1,1]){
    const ear=mesh(new T.ConeGeometry(.14,.34,12),fur);ear.position.set(sx*.23,1.32,.1);ear.rotation.z=-sx*.22;g.add(ear);
    const inner=mesh(new T.ConeGeometry(.08,.21,10),0x755653);inner.position.set(sx*.23,1.34,.17);inner.rotation.z=-sx*.22;g.add(inner);
    const cheek=mesh(new T.SphereGeometry(.125,16,12),0x84776c);cheek.position.set(sx*.075,.94,.45);g.add(cheek);
    const eye=mesh(new T.SphereGeometry(.045,12,10),0x0b1010,false);eye.position.set(sx*.125,1.11,.46);g.add(eye);
    const glint=mesh(new T.SphereGeometry(.012,6,6),0xfff8df,false);glint.position.set(sx*.125-.01,1.125,.5);g.add(glint);
    const leg=mesh(new T.SphereGeometry(.13,16,12),fur);leg.scale.set(1,1.8,1);leg.position.set(sx*.22,.23,.2);g.add(leg);
    const haunch=mesh(new T.SphereGeometry(.2,16,12),fur);haunch.position.set(sx*.3,.2,-.13);g.add(haunch);
  }
  const nose=mesh(new T.SphereGeometry(.07,12,10),0x181819);nose.scale.y=.7;nose.position.set(0,1.0,.55);g.add(nose);
  const tongue=mesh(new T.SphereGeometry(.055,12,10),0xc97a78);tongue.scale.set(.8,1.3,.6);tongue.position.set(0,.88,.52);g.add(tongue);
  const collar=mesh(new T.TorusGeometry(.265,.04,8,24),0x967331);collar.rotation.x=Math.PI/2;collar.position.set(0,.8,.07);g.add(collar);
  const medal=mesh(new T.CylinderGeometry(.085,.085,.025,16),0xe0b35c);medal.rotation.x=Math.PI/2;medal.position.set(0,.72,.36);g.add(medal);
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
  const overall = mesh(new T.CylinderGeometry(0.57, 0.6, 0.55, 24), 0x7a674d); overall.position.y = 0.48; g.add(overall);
  const cap = mesh(new T.SphereGeometry(0.35, 20, 12, 0, Math.PI * 2, 0, Math.PI / 2), 0xd8ad48); cap.position.set(0, 1.5, 0.04); g.add(cap);
  const brim=mesh(new T.CylinderGeometry(.43,.43,.05,24),0xd8ad48);brim.position.set(0,1.5,.08);g.add(brim);
  const hammer = mesh(new T.BoxGeometry(0.06, 0.4, 0.06), 0x7b4f2d); hammer.position.set(0.6, 0.8, 0.2); hammer.rotation.z = 0.5; g.add(hammer);
  const head2 = mesh(new T.BoxGeometry(0.2, 0.08, 0.1), 0x8a8a8a); head2.position.set(0.7, 0.98, 0.2); head2.rotation.z = 0.5; g.add(head2);
  return g;
}

const ROOF = { do: 0xb8433a, xanh: 0x3e6fa8, luc: 0x4f8a45, nau: 0x8a5a30 };

// Nhà theo thiết kế và vật liệu của Bò Mộc (PLAN_LOGIC_GAME 13.3): rơm có vách đan và mái rơm dày, gỗ có vách ván
// và mái màu, gạch có tường gạch đỏ mạch vữa và mái ngói màu. Thiết kế: chuồng nhỏ, nhà hai gian, nhà gạch hai
// tầng (cấp 15). state "building" là khung giàn đang dựng. Mặt tiền (cửa) quay về +z.
const HOUSE_SIZE = { chuong_nho: [2.4, 1.7, 1.4], nha_hai_gian: [4.2, 2.4, 1.6], nha_hai_tang: [3.6, 2.6, 3.1] };
const WALL = { rom: 0xd8b45a, go: 0x8a5a30, gach: 0xb5553c };
const TRIM = { rom: 0x9c7a35, go: 0x5c3a1f, gach: 0xe8dcc6 };
export function buildHouse(design, material, roof, state) {
  const g = new T.Group();
  const [W, D, H] = HOUSE_SIZE[design] || HOUSE_SIZE.chuong_nho, mt = WALL[material] ? material : 'go';
  if (state === 'building') {
    for (const sx of [-1, 1]) for (const sz of [-1, 1]) { const p = mesh(new T.BoxGeometry(0.12, H + 0.5, 0.12), 0xc9a36a); p.position.set(sx * W / 2, (H + 0.5) / 2, sz * D / 2); g.add(p); }
    for (const y of [0.9, H + 0.4]) for (const sz of [-1, 1]) { const b = mesh(new T.BoxGeometry(W + 0.1, 0.1, 0.1), 0xc9a36a); b.position.set(0, y, sz * D / 2); g.add(b); }
    for (let i = 0; i < 3; i++) { const plank = mesh(new T.BoxGeometry(1.1, 0.08, 0.25), 0xc9a36a); plank.position.set(W / 2 + 0.6, 0.05 + i * 0.09, D / 2 + 0.4); plank.rotation.y = 0.2; g.add(plank); }
    const pile = mesh(new T.BoxGeometry(0.8, 0.4, 0.6), WALL[mt]); pile.position.set(-W / 2 - 0.6, 0.2, D / 2 + 0.2); g.add(pile);
    return g;
  }
  const floors = design === 'nha_hai_tang' ? 2 : 1, fh = H / floors;
  for (let f = 0; f < floors; f++) {
    const body = mesh(new T.BoxGeometry(W - f * 0.2, fh, D - f * 0.2), WALL[mt]); body.position.y = 0.15 + fh / 2 + f * fh; body.receiveShadow = true; g.add(body);
    // Vân vách: bó rơm, ván gỗ hoặc mạch vữa gạch.
    const lines = mt === 'gach' ? 5 : 4;
    for (let i = 1; i < lines; i++) {
      const ln = mesh(new T.BoxGeometry(W - f * 0.2 + 0.02, mt === 'gach' ? 0.025 : 0.045, D - f * 0.2 + 0.02), TRIM[mt], false);
      ln.position.y = 0.15 + f * fh + (fh * i) / lines; g.add(ln);
    }
    // Cửa sổ hai bên mặt tiền.
    for (const sx of design === 'chuong_nho' ? [] : [-1, 1]) {
      const win = mesh(new T.BoxGeometry(0.55, 0.45, 0.05), 0x3a2a1a, false); win.position.set(sx * W * 0.3, 0.15 + f * fh + fh * 0.6, (D - f * 0.2) / 2 + 0.02); g.add(win);
      const sill = mesh(new T.BoxGeometry(0.7, 0.06, 0.12), TRIM[mt], false); sill.position.set(sx * W * 0.3, 0.15 + f * fh + fh * 0.33, (D - f * 0.2) / 2 + 0.05); g.add(sill);
    }
  }
  const base = mesh(new T.BoxGeometry(W + 0.3, 0.15, D + 0.3), mt === 'gach' ? 0x8b8378 : 0x6e5434); base.position.y = 0.075; g.add(base);
  // Cửa: nhà hai gian có hai cửa, nhà khác một cửa; ban công cho nhà hai tầng.
  for (const dx of design === 'nha_hai_gian' ? [-W * 0.18, W * 0.18] : [0]) {
    const door = mesh(new T.BoxGeometry(0.7, 0.95, 0.05), 0x2e1f12, false); door.position.set(dx, 0.62, D / 2 + 0.02); g.add(door);
  }
  if (floors === 2) {
    const deck = mesh(new T.BoxGeometry(W * 0.6, 0.08, 0.6), TRIM[mt]); deck.position.set(0, 0.15 + fh, D / 2 + 0.3); g.add(deck);
    for (let i = -2; i <= 2; i++) { const post = mesh(new T.BoxGeometry(0.05, 0.45, 0.05), 0x5c3a1f, false); post.position.set(i * W * 0.14, 0.4 + fh, D / 2 + 0.58); g.add(post); }
    const rail = mesh(new T.BoxGeometry(W * 0.6, 0.06, 0.06), 0x5c3a1f, false); rail.position.set(0, 0.62 + fh, D / 2 + 0.58); g.add(rail);
  }
  // Mái: rơm dày màu rơm (mép mái dải màu đã chọn), gỗ/gạch mái dốc theo màu đã chọn.
  const roofHex = mt === 'rom' ? 0xc9a04a : ROOF[roof] || ROOF.do, top = 0.15 + H;
  for (const sx of [-1, 1]) {
    const r = mesh(new T.BoxGeometry(W / 2 + 0.45, mt === 'rom' ? 0.22 : 0.1, D + 0.5), roofHex); r.position.set(sx * (W / 4 + 0.05), top + W * 0.14, 0); r.rotation.z = -sx * 0.55; g.add(r);
  }
  const ridge = mesh(new T.BoxGeometry(0.24, 0.16, D + 0.55), mt === 'rom' ? ROOF[roof] || ROOF.do : 0x4a3322); ridge.position.set(0, top + W * 0.28, 0); g.add(ridge);
  if (mt === 'gach') { const chim = mesh(new T.BoxGeometry(0.35, 0.7, 0.35), 0x9a4632); chim.position.set(W * 0.28, top + W * 0.28, -D * 0.2); g.add(chim); }
  return g;
}
export function buildCoop(roof, state) { return buildHouse('chuong_nho', 'go', roof, state); }

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
  dress(g, {}, {shirt:'none'}, cosmetics);
  const scarf=mesh(new T.TorusGeometry(.28,.07,8,24),0x9a4933);scarf.rotation.x=Math.PI/2;scarf.position.set(0,-.28,.02);g.userData.parts.head.add(scarf);
  return g;
}

// Ông Chủ Đầu Tư (PLAN_LOGIC_GAME 13.2): gà trống bụng phệ, vest, mũ phớt, cầm bản đồ cuộn.
export function buildInvestor(cosmetics) {
  const g = buildChicken({ color: 'trang', comb: 'kep' });
  g.scale.setScalar(1.12);
  dress(g, { body: 'vest', head: 'mu_phot' }, { shirt: 'none' }, cosmetics);
  const belly = mesh(new T.SphereGeometry(0.3, 16, 12), 0x2d3a52); belly.scale.set(1.1, 0.9, 0.7); belly.position.set(0, 0.5, 0.32); g.userData.parts.rig.add(belly);
  const scroll = mesh(new T.CylinderGeometry(0.06, 0.06, 0.5, 10), 0xe9dcc0); scroll.rotation.z = Math.PI / 2; scroll.position.set(0.5, 0.7, 0.25); g.userData.parts.rig.add(scroll);
  const tie = mesh(new T.ConeGeometry(0.06, 0.22, 4), 0xc8343a); tie.rotation.x = Math.PI; tie.position.set(0, 0.86, 0.5); g.userData.parts.rig.add(tie);
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
  const sp = new T.Sprite(new T.SpriteMaterial({ map: tex, depthTest: false, depthWrite: false, transparent: true, toneMapped: false }));
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
  // Mũ rơm của Vịt Bông (ảnh B02-06).
  const brim = mesh(new T.CylinderGeometry(0.42, 0.42, 0.035, 20), 0xdcb86e); brim.position.set(0, 1.27, 0.22); brim.rotation.x = -0.12; g.add(brim);
  const crown = mesh(new T.CylinderGeometry(0.2, 0.24, 0.2, 16), 0xdcb86e); crown.position.set(0, 1.38, 0.21); crown.rotation.x = -0.12; g.add(crown);
  const band = mesh(new T.CylinderGeometry(0.245, 0.245, 0.06, 16), 0x7a5a3a); band.position.set(0, 1.31, 0.215); band.rotation.x = -0.12; g.add(band);
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

function wheel(r, w, spokes = false) {
  const g = new T.Group();
  const tire = mesh(new T.TorusGeometry(r * .84, r * .16, 8, 28), 0x292a27);
  tire.rotation.y = Math.PI / 2; tire.scale.z = w / (r * .32); g.add(tire);
  const hub = mesh(new T.CylinderGeometry(r * .24, r * .24, w + .02, 16), 0x9a9a87);hub.rotation.z = Math.PI / 2;g.add(hub);
  if (spokes) {
    const rim = mesh(new T.TorusGeometry(r * .82, .018, 6, 28), 0xb0ae99);rim.rotation.y=Math.PI/2;g.add(rim);
    const sg = new T.CylinderGeometry(.008,.008,r*1.6,5), inst=new T.InstancedMesh(sg,mat(0xb8b9a5),10), dummy=new T.Object3D();
    for(let i=0;i<10;i++){dummy.rotation.x=i*Math.PI/10;dummy.updateMatrix();inst.setMatrixAt(i,dummy.matrix);}g.add(inst);
  } else {
    const rim=mesh(new T.CylinderGeometry(r*.58,r*.58,w*.93,20),0x6c6e66);rim.rotation.z=Math.PI/2;g.add(rim);
  }
  g.userData.wheel = true;
  return g;
}

// Compact static body pieces into one draw per material; wheel groups remain articulated.
function batchBody(g) {
  g.updateMatrixWorld(true);
  const buckets = new Map(), remove = [];
  g.traverse(o => {
    if(!o.isMesh || o.isInstancedMesh || o.parent !== g || o.material.transparent) return;
    const key=o.material.uuid;
    if(!buckets.has(key)) buckets.set(key,{material:o.material,positions:[],normals:[],uvs:[]});
    const b=buckets.get(key), geo=o.geometry.index?o.geometry.toNonIndexed():o.geometry.clone();
    geo.applyMatrix4(o.matrix); const ps=geo.attributes.position, ns=geo.attributes.normal, uv=geo.attributes.uv;
    for(let i=0;i<ps.count;i++){b.positions.push(ps.getX(i),ps.getY(i),ps.getZ(i));b.normals.push(ns.getX(i),ns.getY(i),ns.getZ(i));b.uvs.push(uv?uv.getX(i):0,uv?uv.getY(i):0);}
    geo.dispose();remove.push(o);
  });
  remove.forEach(o=>g.remove(o));
  for(const b of buckets.values()){
    const geo=new T.BufferGeometry();geo.setAttribute('position',new T.Float32BufferAttribute(b.positions,3));geo.setAttribute('normal',new T.Float32BufferAttribute(b.normals,3));geo.setAttribute('uv',new T.Float32BufferAttribute(b.uvs,2));geo.computeBoundingSphere();
    const m=new T.Mesh(geo,b.material);m.castShadow=true;m.receiveShadow=true;g.add(m);
  }
}

// Vehicles keep the original gameplay origin and +z forward direction.
export function buildVehicle(kind) {
  const g = new T.Group(), wheels=[];
  const add = (m,x,y,z) => {m.position.set(x,y,z);g.add(m);return m;};
  const addWheel=(r,w,x,z,spokes=false)=>{const m=add(wheel(r,w,spokes),x,r,z);wheels.push(m);return m;};
  if(kind==='xe_dap'){
    const blue=0x39637c, metal=0xaaa994;
    addWheel(.34,.065,0,.64,true);addWheel(.34,.065,0,-.76,true);
    for(const [a,b] of [[[0,.34,-.76],[0,.73,-.35]],[[0,.73,-.35],[0,.34,-.05]],[[0,.34,-.05],[0,.34,-.76]],[[0,.73,-.35],[0,.8,.42]],[[0,.8,.42],[0,.34,-.05]],[[0,.8,.42],[0,.34,.64]]])link(g,a,b,.025,blue);
    for(const sx of [-1,1])link(g,[sx*.045,.8,.42],[sx*.045,.34,.64],.022,blue);
    link(g,[0,.75,-.35],[0,.88,-.35],.023,metal);add(softBox(.25,.07,.28,.03,0x78533b),0,.88,-.34);
    link(g,[0,.78,.42],[0,1.02,.4],.022,metal);link(g,[-.25,1.02,.4],[.25,1.02,.4],.022,metal);
    for(const sx of [-1,1])link(g,[sx*.18,1.02,.4],[sx*.3,1.02,.4],.03,0x3b3832);
    add(softBox(.29,.08,.38,.025,0x9c6c46),0,.65,-.79);
    const back=add(softBox(.3,.3,.06,.03,blue),0,.85,-.98);back.rotation.x=-.16;
    for(const sx of [-1,1])link(g,[sx*.12,.62,-.81],[sx*.07,.36,-.76],.015,metal);
    const basket=add(softBox(.42,.28,.32,.035,0x8a643c),0,.88,.67);
    add(softBox(.35,.025,.25,.015,0x51402a),0,1.025,.67);
    for(let i=-2;i<=2;i++)link(g,[i*.075,.78,.838],[i*.075,1,.838],.008,0xd2ab68);
    const pedal=add(mesh(new T.CylinderGeometry(.06,.06,.22,12),metal),0,.35,-.05);pedal.rotation.z=Math.PI/2;
    for(const sx of [-1,1])add(softBox(.1,.035,.12,.01,0x34342f),sx*.14,.34,-.05);
  }else if(kind==='xe_may'){
    const red=0xb84d3b;
    addWheel(.3,.15,0,.7);addWheel(.3,.15,0,-.8);
    add(softBox(.52,.39,.67,.13,red),0,.51,-.62);
    add(softBox(.45,.1,.65,.04,0x777267),0,.34,.02);
    const apron=add(softBox(.51,.64,.14,.06,red),0,.71,.49);apron.rotation.x=-.15;
    add(softBox(.45,.12,1.08,.06,0x383a35),0,.78,-.33);
    add(softBox(.34,.12,.42,.06,red),0,.5,.72);
    link(g,[0,.78,.54],[0,1.03,.6],.055,red);link(g,[-.34,1.03,.6],[.34,1.03,.6],.035,0x6f716b);
    const lamp=add(mesh(new T.CylinderGeometry(.105,.105,.13,20),0xefddb0),0,1.04,.67);lamp.rotation.x=Math.PI/2;
    for(const sx of [-1,1]){
      link(g,[sx*.25,1.04,.6],[sx*.35,1.23,.55],.012,0x9e9e8e);
      add(softBox(.12,.085,.03,.025,0x9caaa6),sx*.35,1.24,.55);
      add(softBox(.12,.06,.3,.02,0xaaa68f),sx*.27,.3,-.62);
    }
    add(softBox(.15,.12,.025,.025,0xe58a63),0,.65,-.98);
  }else if(kind==='o_to'){
    const blue=0x56788a, glass=0x617773, trim=0x414a47;
    add(softBox(1.6,.55,2.5,.12,blue),0,.59,-.2);
    // Open cabin uses real pillars, roof and tinted windows, so riders can sit inside.
    add(softBox(1.42,.12,1.52,.07,blue),0,1.48,-.4);
    for(const sx of [-1,1]){
      for(const z of [-1.08,-.38,.32])link(g,[sx*.68,.83,z],[sx*.62,1.44,z-.04],.045,blue);
      add(softBox(.045,.38,1.32,.02,glass),sx*.685,1.14,-.4);
      add(softBox(.045,.07,.16,.015,0xc2bba0),sx*.81,.82,-.38);
      add(softBox(.18,.12,.1,.025,blue),sx*.83,1.12,.23);
      add(softBox(.08,.11,.16,.02,0xc06a4b),sx*.66,.62,-1.45);
      const light=add(mesh(new T.CylinderGeometry(.13,.13,.04,24),0xffd68a),sx*.53,.68,1.07);light.rotation.x=Math.PI/2;
    }
    const front=add(softBox(1.25,.43,.035,.015,glass),0,1.17,.3);front.rotation.x=-.14;
    add(softBox(1.24,.4,.035,.015,glass),0,1.18,-1.12);
    for(const [x,z]of[[-.75,.6],[.75,.6],[-.75,-1],[.75,-1]])addWheel(.29,.2,x,z);
    add(softBox(1.55,.13,.11,.025,0xaaa58f),0,.39,1.1);
    add(softBox(.53,.22,.03,.015,trim),0,.61,1.072);
    for(let i=0;i<4;i++)add(softBox(.48,.012,.01,.003,0x747c72),0,.54+i*.045,1.095);
    add(softBox(.3,.12,.015,.008,0xc1bda6),0,.4,1.165);
    for(const x of [-.36,.36])for(const z of [-.75,-.05]){add(softBox(.45,.13,.43,.04,0x5f5b4c),x,.76,z);add(softBox(.45,.37,.12,.04,0x5f5b4c),x,1,z-.17);}
    const steer=add(mesh(new T.TorusGeometry(.15,.018,6,20),0x3b3f39),-.36,1.02,.22);steer.rotation.x=-.55;
  }
  g.userData.wheels=wheels;
  batchBody(g);
  return g;
}

// Chim Chíp bus: olive body, cream stripe, visible cabin, eight passenger seats.
export function buildBus() {
  const g=new T.Group(),wheels=[],olive=0x697648,cream=0xc3ba83,trim=0x595b43;
  const add=(m,x,y,z)=>{m.position.set(x,y,z);g.add(m);return m;};
  add(softBox(2,.85,4.6,.12,olive),0,.83,0);
  add(softBox(2.06,.13,4.66,.06,cream),0,1.03,0);
  add(softBox(2.08,.22,4.7,.1,olive),0,2.25,0);
  add(softBox(1.88,.08,4.38,.025,0x625e46),0,1.23,0);
  const glassMat=mat(0x80928b,{transparent:true,opacity:.32,roughness:.25,metalness:.1,depthWrite:false});
  for(const sx of [-1,1]){
    for(let i=0;i<6;i++){
      const z=-1.94+i*.72;
      add(softBox(.075,1.02,.065,.018,trim),sx*.99,1.68,z-.36);
      // Leave the boarding doorway open on the right near the front.
      if(sx===1&&i===5)continue;
      const win=add(softBox(.025,.86,.63,.012,0x80928b),sx*1.005,1.69,z);win.material=glassMat;
      add(softBox(.08,.045,.67,.01,trim),sx*1.01,1.24,z);
    }
    for(const z of[-1.45,1.45]){const w=add(wheel(.4,.24),sx*1.01,.4,z);wheels.push(w);}
    add(softBox(.06,.14,4.32,.02,0x565b40),sx*1.025,.48,0);
    link(g,[sx*.96,1.89,2.02],[sx*1.22,1.89,2.18],.025,trim);
    add(softBox(.14,.28,.075,.025,trim),sx*1.23,1.79,2.2);
    const lamp=add(softBox(.33,.25,.06,.04,0xffd995),sx*.66,.79,2.32);lamp.material=mat(0xffd995,{emissive:0xd6983d,emissiveIntensity:.35});
    add(softBox(.11,.12,.025,.015,0xd68c4b),sx*.85,.66,2.33);
  }
  for(const x of [-.95,0,.95])add(softBox(.055,.95,.075,.01,trim),x,1.68,2.28);
  for(const sx of [-1,1]){
    const glass=add(softBox(.86,.84,.02,.01,0x80928b),sx*.475,1.68,2.31);glass.material=glassMat;
    const wiper=link(g,[sx*.5,1.28,2.335],[sx*.22,1.48,2.335],.014,0x313a32);
  }
  add(softBox(1.97,.16,.14,.035,0x77755d),0,.42,2.35);
  add(softBox(.8,.27,.035,.02,0x383f30),0,.77,2.325);
  for(let i=0;i<4;i++)add(softBox(.75,.02,.02,.005,0x85876d),0,.68+i*.06,2.35);
  add(softBox(.36,.14,.025,.01,0xb8b498),0,.43,2.43);
  add(softBox(.5,.08,.42,.025,0x46533d),-.4,2.39,-.9);
  add(softBox(.5,.08,.42,.025,0x46533d),.4,2.39,.9);
  for(const x of[-.5,.5])for(const z of[-1.65,-.93,-.21,.51]){
    add(softBox(.55,.15,.46,.035,0x736547),x,1.05,z);
    add(softBox(.55,.55,.12,.04,0x736547),x,1.34,z-.22);
  }
  // Boarding steps and threshold sit on the right, away from the driver's seat.
  add(softBox(.35,.12,.64,.025,0x8a8064),1.05,.45,1.75);
  add(softBox(.26,.12,.62,.025,0xa49a7a),1.15,.3,1.75);
  const driver=buildChicken({color:'kem'});driver.scale.setScalar(.62);driver.position.set(-.45,1.02,1.57);g.add(driver);
  addCosmetic(driver.userData.parts.slots.head,{kind:'cap',color:0x3a4b47});
  const steering=add(mesh(new T.TorusGeometry(.22,.025,8,24),trim),-.45,1.46,2.03);steering.rotation.x=-.45;
  const cv=document.createElement('canvas');cv.width=512;cv.height=100;const ctx=cv.getContext('2d');
  ctx.fillStyle='#51462f';ctx.fillRect(0,0,512,100);ctx.strokeStyle='#c1a772';ctx.lineWidth=5;ctx.strokeRect(5,5,502,90);ctx.font='600 50px system-ui';ctx.textAlign='center';ctx.fillStyle='#e4d3a5';ctx.fillText('CHIM CHÍP',256,69);
  const signMat=new T.MeshBasicMaterial({map:new T.CanvasTexture(cv)}),sign=new T.Mesh(new T.PlaneGeometry(1.4,.27),signMat);sign.position.set(0,2.12,2.365);g.add(sign);
  g.userData.wheels=wheels;
  batchBody(g);
  return g;
}

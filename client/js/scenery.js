// Art-directed 3D farm spaces. Geometry stays native and interactive at every angle.
// Coordinates match the authoritative zone data from server/world.go (items, obstacles, bounds).
import { mat, buildChicken, dress, buildCoop, buildBird, buildBus, buildVehicle } from './models.js';

const T = window.THREE;
const geo = new Map(), materials = new Map();
const C = { wood: 0x8e613b, edge: 0x5c3c25, light: 0xb88b54, cream: 0xf3dfaf, straw: 0xd5ac53, grass: 0x8fa262, sage: 0x56754b, leaf: 0x688949, dirt: 0xb89a63, path: 0xd1b480, blue: 0x7aa6ad, red: 0xaf5542 };
function random(seed) { let n = seed >>> 0; return () => { n = (1664525 * n + 1013904223) >>> 0; return n / 4294967296; }; }
function geometry(kind, ...args) { const k = kind + args.join(','); if (!geo.has(k)) geo.set(k, new T[kind](...args)); return geo.get(k); }
function material(color, opts = {}) { const k = color + JSON.stringify(opts); if (!materials.has(k)) materials.set(k, mat(color, opts)); return materials.get(k); }
function add(g, shape, color, x = 0, y = 0, z = 0, opts = {}) {
  const m = new T.Mesh(shape, typeof color === 'number' ? material(color) : color);
  m.position.set(x, y, z); m.castShadow = opts.shadow !== false; m.receiveShadow = true; g.add(m); return m;
}
// One shared unit cube scaled per use, so planks, posts and straws batch into a few instanced draws.
function box(g, w, h, d, color, x = 0, y = 0, z = 0) { const m = add(g, geometry('BoxGeometry', 1, 1, 1), color, x, y, z); m.scale.set(w, h, d); return m; }
function cylinder(g, r1, r2, h, color, x = 0, y = 0, z = 0, sides = 12) { return add(g, geometry('CylinderGeometry', r1, r2, h, sides), color, x, y, z); }
function sphere(g, r, color, x = 0, y = 0, z = 0) { return add(g, geometry('SphereGeometry', r, 14, 10), color, x, y, z); }
function beam(g, a, b, radius, color) {
  const av = new T.Vector3(...a), bv = new T.Vector3(...b), delta = bv.clone().sub(av);
  const m = box(g, radius, delta.length(), radius, color);
  m.position.copy(av).add(bv).multiplyScalar(0.5); m.quaternion.setFromUnitVectors(new T.Vector3(0, 1, 0), delta.normalize()); return m;
}

// ---------- Procedural textures (no external assets) ----------
const textures = {};
function canvasTexture(kind) {
  if (textures[kind]) return textures[kind];
  const cv = document.createElement('canvas'); cv.width = cv.height = 512;
  const ctx = cv.getContext('2d'), rng = random(kind.length * 977 + kind.charCodeAt(0));
  const base = { wood: '#c0a77e', earth: '#c9b591', grass: '#a9b878', weave: '#c99f55', straw: '#d6b261', grain: '#d9a948', stone: '#b9b3a2' }[kind] || '#ccc';
  ctx.fillStyle = base; ctx.fillRect(0, 0, 512, 512);
  if (kind === 'wood') {
    for (let i = 0; i < 90; i++) {
      const yy = rng() * 512; ctx.strokeStyle = `rgba(65,36,17,${0.04 + rng() * 0.15})`; ctx.lineWidth = rng() * 3 + 0.3;
      ctx.beginPath(); ctx.moveTo(0, yy); ctx.bezierCurveTo(140, yy + rng() * 8, 350, yy - rng() * 8, 512, yy + rng() * 3); ctx.stroke();
    }
    for (let i = 0; i < 6; i++) { ctx.strokeStyle = '#8e7959'; ctx.lineWidth = 1.4; ctx.beginPath(); ctx.ellipse(rng() * 512, rng() * 512, 8 + rng() * 16, 2 + rng() * 3, 0, 0, Math.PI * 2); ctx.stroke(); }
  } else if (kind === 'weave') {
    // Herringbone straw weave of a nest basket.
    for (let row = 0; row < 16; row++) for (let col = 0; col < 24; col++) {
      const x = col * 22 + (row % 2) * 11, y = row * 32, dir = row % 2 ? 1 : -1;
      const grad = ctx.createLinearGradient(x, y, x + 18, y + 26);
      grad.addColorStop(0, '#e7c879'); grad.addColorStop(.55, '#c39647'); grad.addColorStop(1, '#8c6428');
      ctx.fillStyle = grad; ctx.beginPath(); ctx.moveTo(x, y); ctx.lineTo(x + 16, y + dir * 6 + 6); ctx.lineTo(x + 16, y + dir * 6 + 28); ctx.lineTo(x, y + 22); ctx.closePath(); ctx.fill();
      ctx.strokeStyle = 'rgba(90,60,20,.35)'; ctx.lineWidth = 1; ctx.stroke();
    }
    for (let i = 0; i < 260; i++) { ctx.strokeStyle = `rgba(250,226,150,${rng() * .35})`; ctx.lineWidth = 1; const x = rng() * 512, y = rng() * 512; ctx.beginPath(); ctx.moveTo(x, y); ctx.lineTo(x + rng() * 24 - 12, y + rng() * 6); ctx.stroke(); }
  } else if (kind === 'straw' || kind === 'grain') {
    for (let i = 0; i < (kind === 'grain' ? 9000 : 2600); i++) {
      const v = rng(), x = rng() * 512, y = rng() * 512;
      ctx.strokeStyle = v > .5 ? `rgba(248,224,150,${.3 + rng() * .5})` : `rgba(120,82,28,${.2 + rng() * .4})`;
      ctx.lineWidth = kind === 'grain' ? 3 : 1 + rng() * 1.5;
      const a = rng() * Math.PI, l = kind === 'grain' ? 3 : 10 + rng() * 26;
      ctx.beginPath(); ctx.moveTo(x, y); ctx.lineTo(x + Math.cos(a) * l, y + Math.sin(a) * l); ctx.stroke();
    }
  } else if (kind === 'grass') {
    for (let i = 0; i < 9000; i++) { const v = rng(); ctx.fillStyle = v > .55 ? `rgba(206,214,150,${rng() * .35})` : `rgba(66,95,46,${rng() * .25})`; ctx.fillRect(rng() * 512, rng() * 512, 1 + rng() * 3, 2 + rng() * 6); }
  } else {
    for (let i = 0; i < 6500; i++) { const v = rng(); ctx.fillStyle = v > 0.5 ? `rgba(239,224,180,${rng() * 0.2})` : `rgba(86,72,41,${rng() * 0.15})`; ctx.fillRect(rng() * 512, rng() * 512, 1 + rng() * 8, 1 + rng() * 5); }
    for (let i = 0; i < 90; i++) { ctx.fillStyle = `rgba(110,92,64,${.2 + rng() * .25})`; ctx.beginPath(); ctx.ellipse(rng() * 512, rng() * 512, 2 + rng() * 4, 1.5 + rng() * 3, rng() * 3, 0, Math.PI * 2); ctx.fill(); }
  }
  const tex = new T.CanvasTexture(cv); tex.wrapS = tex.wrapT = T.RepeatWrapping; tex.anisotropy = 4; textures[kind] = tex; return tex;
}
function textured(kind, color = 0xffffff, repeat = [1, 1], opts = {}) {
  const k = 'tex:' + kind + color + repeat + JSON.stringify(opts);
  if (!materials.has(k)) {
    const map = canvasTexture(kind).clone(); map.needsUpdate = true; map.repeat.set(...repeat);
    materials.set(k, new T.MeshStandardMaterial(Object.assign({ color, map, roughness: .95 }, opts)));
  }
  return materials.get(k);
}
function timber(color = C.wood) { return textured('wood', color, [1, 1], { roughness: .94 }); }
function ground(g, w, d, color = C.dirt, kind = 'earth', y = 0) {
  const m = new T.Mesh(geometry('PlaneGeometry', w, d), textured(kind, color, [w / 6, d / 6], { roughness: 1 }));
  m.rotation.x = -Math.PI / 2; m.position.y = y; m.receiveShadow = true; g.add(m); return m;
}
let shadowTex;
function shadow(g, x, z, rx, rz = rx, y = 0.018) {
  if (!shadowTex) {
    const cv = document.createElement('canvas'); cv.width = cv.height = 64;
    const c = cv.getContext('2d'), grad = c.createRadialGradient(32, 32, 2, 32, 32, 31);
    grad.addColorStop(0, 'rgba(44,31,10,.3)'); grad.addColorStop(1, 'rgba(44,31,10,0)'); c.fillStyle = grad; c.fillRect(0, 0, 64, 64); shadowTex = new T.CanvasTexture(cv);
  }
  const key = 'groundShadow'; if (!materials.has(key)) materials.set(key, new T.MeshBasicMaterial({ map: shadowTex, transparent: true, depthWrite: false }));
  const m = add(g, geometry('PlaneGeometry', 2, 2), materials.get(key), x, y, z, { shadow: false }); m.rotation.x = -Math.PI / 2; m.scale.set(rx, rz, 1); return m;
}
// Fade-able architecture: the camera makes these translucent only while they hide the chicken.
export function occluder(info, group) {
  group.userData.dynamic = true;
  const clones = new Map();
  group.traverse(o => { if (o.isMesh && !Array.isArray(o.material)) { if (!clones.has(o.material)) clones.set(o.material, o.material.clone()); o.material = clones.get(o.material); } });
  info.occluders.push(group); return group;
}

// ---------- Props ----------
function nestShape() {
  if (!geo.has('nestLathe')) {
    const pts = [[.6, .03], [.88, .05], [1.0, .16], [1.04, .3], [.98, .42], [.86, .47], [.74, .4], [.68, .24]].map(([x, y]) => new T.Vector2(x, y));
    geo.set('nestLathe', new T.LatheGeometry(pts, 30));
  }
  return geo.get('nestLathe');
}
// Woven basket with a straw bed; the server owns occupancy and position.
function nest(g, x, z, size = 1, yy = 0) {
  shadow(g, x, z, 1.5 * size, 1.4 * size, yy + 0.02);
  const wall = add(g, nestShape(), textured('weave', 0xffffff, [7, 1], { side: T.DoubleSide }), x, yy, z);
  wall.scale.setScalar(size); wall.rotation.y = (x * 13 + z * 7) % 6.28;
  const bed = add(g, geometry('CircleGeometry', .76, 24), textured('straw', 0xe9d29b, [1, 1]), x, yy + .2 * size, z, { shadow: false });
  bed.rotation.x = -Math.PI / 2; bed.scale.setScalar(size);
  const rng = random(Math.round((x + 60) * 101 + (z + 60) * 53));
  for (let i = 0; i < 14; i++) {
    const a = rng() * Math.PI * 2, r = (0.84 + rng() * 0.2) * size;
    const s = box(g, .03 * size, .03 * size, (.3 + rng() * .3) * size, [0xe5c778, 0xc79c46, 0xdabb66][i % 3], x + Math.cos(a) * r, yy + (.38 + rng() * .1) * size, z + Math.sin(a) * r);
    s.rotation.set(rng() * .5 - .25, -a + Math.PI / 2 + rng() * .8, rng() * .6);
  }
}
function barrel(g, x, z, water = false, scale = 1) {
  const p = new T.Group(); p.position.set(x, 0, z); p.scale.setScalar(scale); g.add(p);
  shadow(p, 0, 0, .8);
  cylinder(p, 0.48, 0.43, 0.95, timber(C.wood), 0, 0.5, 0, 16);
  for (const y of [0.15, 0.75, 0.96]) cylinder(p, 0.49, 0.49, 0.07, 0x565846, 0, y, 0, 16);
  cylinder(p, 0.43, 0.43, 0.035, water ? 0x6fa1ae : 0xb78e4a, 0, 0.96, 0, 16);
}
function crate(g, x, z, s = 1, ry = 0) {
  const p = new T.Group(); p.position.set(x, 0, z); p.rotation.y = ry; p.scale.setScalar(s); g.add(p);
  box(p, 1, 1, 1, timber(0xa47a4c), 0, .5, 0);
  for (const y of [.08, .92]) for (const sz of [-1, 1]) box(p, 1.04, .12, .06, timber(C.edge), 0, y, sz * .5);
  beam(p, [-.48, .1, .52], [.48, .9, .52], .08, C.edge);
}
function sack(g, x, y, z, ry = 0) {
  const s = sphere(g, .5, 0xd8c8a0, x, y + .3, z); s.scale.set(1.15, .62, .82); s.rotation.y = ry;
  const tie = cylinder(g, .1, .14, .16, 0xbca677, x + Math.sin(ry) * .55, y + .42, z + Math.cos(ry) * .55, 8); tie.rotation.x = Math.PI / 2; tie.rotation.z = ry;
}
function hay(g, x, z, s = 1) {
  shadow(g, x, z, 1.8 * s);
  const m = add(g, geometry('ConeGeometry', 1.1, 2.8, 18), textured('straw', 0xe5c374, [3, 2]), x, 1.4 * s, z); m.scale.setScalar(s);
  const rng = random(Math.round((x + 100) * 11 + z * 9));
  for (let i = 0; i < 40; i++) {
    const a = rng() * Math.PI * 2, height = rng() * 2.3, rr = (1 - height / 2.8) * 1.12;
    beam(g, [x + Math.cos(a) * rr * s, height * s + 0.03, z + Math.sin(a) * rr * s], [x + Math.cos(a) * (rr + .05) * s, (height + .45) * s, z + Math.sin(a) * (rr + .05) * s], .02 * s, [0xddc77c, 0xb89c49, 0xe1c774][i % 3]);
  }
}
function trough(g, x, z, isWater = false, length = 7) {
  shadow(g, x, z, 1.2, length * .6);
  box(g, 1.15, .15, length, timber(C.edge), x, .4, z);
  for (const sx of [-1, 1]) box(g, .12, .5, length, timber(C.wood), x + sx * .56, .62, z);
  for (const sz of [-1, 1]) box(g, 1.15, .5, .12, timber(C.wood), x, .62, z + sz * length / 2);
  const fill = isWater ? material(0x6fa9b8, { roughness: .15, metalness: .1 }) : textured('grain', 0xffffff, [1, length / 2]);
  box(g, .96, .04, length - .2, fill, x, isWater ? .74 : .78, z);
  for (const sz of [-1, 1]) box(g, .85, .36, .22, timber(C.edge), x, .18, z + sz * length * .34);
}
function lantern(g, x, y, z) {
  const dark = 0x4a4230;
  cylinder(g, .22, .29, .18, dark, x, y + .35, z, 8);
  box(g, .4, .52, .4, material(0xffd68a, { emissive: 0xffc15b, emissiveIntensity: .45 }), x, y, z);
  for (const sx of [-1, 1]) for (const sz of [-1, 1]) box(g, .035, .6, .035, dark, x + sx * .21, y, z + sz * .21);
  box(g, .48, .07, .48, dark, x, y - .29, z);
}
function lampPost(g, x, z, h = 3.2) { cylinder(g, .09, .13, h, timber(C.edge), x, h / 2, z, 8); beam(g, [x, h - .1, z], [x + .6, h - .1, z], .1, C.edge); lantern(g, x + .6, h - .55, z); }
function flowerpot(g, x, z, s = 1, yy = 0) {
  cylinder(g, .35 * s, .23 * s, .45 * s, 0xb37953, x, yy + .23 * s, z, 12);
  const colors = [0xf0d071, 0xe6a58a, 0xc98077];
  for (let i = 0; i < 5; i++) {
    const a = i * 2.4, xx = x + Math.cos(a) * .22 * s, zz = z + Math.sin(a) * .22 * s;
    cylinder(g, .02 * s, .02 * s, .45 * s, 0x5b7845, xx, yy + .58 * s, zz, 5);
    const p = sphere(g, .13 * s, colors[i % 3], xx, yy + .81 * s, zz); p.scale.y = .55;
    sphere(g, .04 * s, 0xeed38c, xx, yy + .86 * s, zz);
  }
}
function flowers(g, cx, cz, r, n, seed) {
  const rng = random(seed);
  for (let i = 0; i < n; i++) {
    const a = rng() * Math.PI * 2, d = rng() * r, x = cx + Math.cos(a) * d, z = cz + Math.sin(a) * d;
    cylinder(g, .015, .015, .3, 0x5b7845, x, .15, z, 4);
    const p = sphere(g, .09, [0xf0d071, 0xf2f0e6, 0xe6a58a, 0xc9a0d8][i % 4], x, .32, z); p.scale.y = .6;
  }
}
function tree(g, x, z, scale = 1, tint = 0x66864b) {
  const p = new T.Group(); p.position.set(x, 0, z); p.scale.setScalar(scale); g.add(p);
  shadow(p, 0, 0, 3.2);
  cylinder(p, .22, .45, 2.7, timber(C.edge), 0, 1.35, 0, 10);
  beam(p, [0, 1.8, 0], [-.9, 3.2, .1], .25, C.wood); beam(p, [0, 2, 0], [.85, 3.6, -.35], .26, C.wood);
  for (const [xx, yy, zz, r] of [[0, 4.1, 0, 1.65], [-1.15, 3.3, .1, 1.4], [1.15, 3.6, -.1, 1.45], [.15, 3.5, 1.1, 1.35], [-.4, 3.7, -1.1, 1.2]]) {
    const leaf = add(p, geometry('IcosahedronGeometry', r, 1), tint, xx, yy, zz); leaf.scale.y = .9;
  }
  return p;
}
function bush(g, x, z, s = 1, tint = 0x6f8f4f) {
  shadow(g, x, z, 1.3 * s);
  for (const [dx, dz, r] of [[0, 0, .75], [.55, .2, .55], [-.5, .15, .6], [.1, -.45, .5]]) { const b = add(g, geometry('IcosahedronGeometry', r, 1), tint, x + dx * s, r * .75 * s, z + dz * s); b.scale.setScalar(s); }
}
function rock(g, x, z, s = 1) { const r = add(g, geometry('DodecahedronGeometry', .5, 0), 0x9d9886, x, .25 * s, z); r.scale.set(s * 1.2, s * .7, s); r.rotation.y = x; }
function grass(g, hx, hz, count, seed, excluded = () => false, cx = 0, cz = 0) {
  const rng = random(seed);
  for (let i = 0; i < count; i++) {
    const x = cx + (rng() * 2 - 1) * hx, z = cz + (rng() * 2 - 1) * hz;
    if (excluded(x, z)) continue;
    for (let k = 0; k < 3; k++) {
      const blade = add(g, geometry('ConeGeometry', .045, .38, 3), [0x71864b, 0x8b9b56, 0x5b7844][(i + k) % 3], x + (k - 1) * .08, .17, z + (k % 2) * .06, { shadow: false });
      blade.rotation.z = (k - 1) * .35; blade.scale.y = .6 + rng() * .9;
    }
  }
}
function fence(g, points, height = 1.4) {
  for (let i = 1; i < points.length; i++) {
    const a = new T.Vector3(points[i - 1][0], 0, points[i - 1][1]), b = new T.Vector3(points[i][0], 0, points[i][1]);
    const steps = Math.max(1, Math.ceil(a.distanceTo(b) / 3.2));
    for (let k = 0; k <= steps; k++) {
      const p = a.clone().lerp(b, k / steps); box(g, .2, height, .2, timber(C.edge), p.x, height / 2, p.z);
      if (k < steps) { const q = a.clone().lerp(b, (k + 1) / steps); for (const y of [height * .34, height * .76]) beam(g, [p.x, y, p.z], [q.x, y, q.z], .13, C.light); }
    }
  }
}
function path(g, points, width = 5, color = C.path) {
  const curve = new T.CatmullRomCurve3(points.map(p => new T.Vector3(p[0], .025, p[1]))), steps = Math.max(12, points.length * 12);
  const vertices = [], uvs = [], indices = [];
  for (let i = 0; i <= steps; i++) {
    const p = curve.getPoint(i / steps), t = curve.getTangent(i / steps), nx = -t.z, nz = t.x;
    for (const side of [-1, 1]) { vertices.push(p.x + nx * width / 2 * side, .03, p.z + nz * width / 2 * side); uvs.push(side === -1 ? 0 : width / 6, i / steps * curve.getLength() / 6); }
    if (i < steps) { const n = i * 2; indices.push(n, n + 2, n + 1, n + 1, n + 2, n + 3); }
  }
  const shape = new T.BufferGeometry(); shape.setAttribute('position', new T.Float32BufferAttribute(vertices, 3)); shape.setAttribute('uv', new T.Float32BufferAttribute(uvs, 2)); shape.setIndex(indices); shape.computeVertexNormals();
  const m = add(g, shape, textured('earth', color, [1, 1], { roughness: 1, side: T.DoubleSide, polygonOffset: true, polygonOffsetFactor: -1 }), 0, 0, 0, { shadow: false });
  for (let i = 0; i <= 10; i++) { // soft stone edging
    const p = curve.getPoint(i / 10), t = curve.getTangent(i / 10);
    for (const side of [-1, 1]) rock(g, p.x - t.z * (width / 2 + .3) * side, p.z + t.x * (width / 2 + .3) * side, .28);
  }
  return m;
}
function textCanvas(text, { bg = '#315b4b', fg = '#fff0c7', border = '#d8bb77', font = '600 58px Georgia, serif', w = 768, h = 192 } = {}) {
  const cv = document.createElement('canvas'); cv.width = w; cv.height = h;
  const c = cv.getContext('2d'); c.fillStyle = bg; c.fillRect(0, 0, w, h);
  c.strokeStyle = border; c.lineWidth = 6; c.strokeRect(12, 12, w - 24, h - 24);
  c.fillStyle = fg; c.textAlign = 'center'; c.textBaseline = 'middle'; c.font = font; c.fillText(text, w / 2, h / 2 + 2, w - 60);
  const t = new T.CanvasTexture(cv); t.anisotropy = 4; return t;
}
function plaque(g, text, x, y, z, width = 5, ry = 0, style) {
  const p = new T.Group(); p.position.set(x, y, z); p.rotation.y = ry; g.add(p);
  box(p, width + .16, width / 4 + .16, .12, timber(C.edge), 0, 0, -.08);
  add(p, geometry('PlaneGeometry', width, width / 4), new T.MeshStandardMaterial({ map: textCanvas(text, style), roughness: .9 }), 0, 0, 0, { shadow: false });
  return p;
}
// Wooden arch over a zone exit. Facing +ry towards the side people arrive from.
export function gateArch(title, ry = 0, wide = 4.2) {
  const g = new T.Group(); g.rotation.y = ry;
  for (const sx of [-1, 1]) {
    shadow(g, sx * wide / 2, 0, .8);
    box(g, .38, 3.6, .38, timber(C.edge), sx * wide / 2, 1.8, 0);
    box(g, .55, .3, .55, timber(C.light), sx * wide / 2, .15, 0);
    flowerpot(g, sx * (wide / 2 + .8), .4, .9);
  }
  box(g, wide + 1, .32, .45, timber(C.wood), 0, 3.55, 0);
  box(g, wide + .6, .22, .38, timber(C.light), 0, 3.15, 0);
  plaque(g, title, 0, 3.05, .26, Math.min(wide - .3, 3.8), 0, { font: '600 62px Georgia, serif' });
  plaque(g, title, 0, 3.05, -.26, Math.min(wide - .3, 3.8), Math.PI, { font: '600 62px Georgia, serif' });
  lantern(g, 0, 2.55, 0);
  return g;
}
// Trees and bushes just outside the walkable bounds so every outdoor zone has a natural edge.
function surround(g, hx, hz, seed, gaps = [], density = 1) {
  const rng = random(seed), tints = [0x6e8c4b, 0x7d9853, 0x66844f, 0x86a05a];
  const open = (x, z) => gaps.some(([gx, gz, r]) => Math.hypot(x - gx, z - gz) < (r || 6));
  const per = (len) => Math.ceil(len / 4.2 * density);
  const ring = (fn, len) => { for (let i = 0; i <= per(len); i++) fn(i / per(len)); };
  const put = (x, z) => { if (open(x, z)) return; if (rng() < .55) tree(g, x + (rng() - .5) * 2, z + (rng() - .5) * 2, .75 + rng() * .6, tints[Math.floor(rng() * 4)]); else bush(g, x, z, .8 + rng() * .6, tints[Math.floor(rng() * 4)]); };
  for (const d of [2.5, 7.5]) {
    ring(t => { put(-hx - d + 2 * (hx + d) * t, -hz - d); put(-hx - d + 2 * (hx + d) * t, hz + d); }, 2 * hx);
    ring(t => { put(-hx - d, -hz - d + 2 * (hz + d) * t); put(hx + d, -hz - d + 2 * (hz + d) * t); }, 2 * hz);
  }
}

// ---------- Residents (ambient NPCs, never counted as players) ----------
function resident(g, arr, cosmetics, name, x, z, action, color, shirt, yaw = 0, extra = {}) {
  const m = buildChicken({ color, eyes: action === 'sleep' ? 'ngai' : 'tron', comb: extra.comb || 'don' });
  dress(m, extra.equipped || {}, { shirt }, cosmetics || []);
  m.position.set(x, extra.y || 0, z); m.rotation.y = yaw; m.userData.dynamic = true; m.userData.npcName = name; g.add(m);
  const a = { model: m, name, action, x, z, baseY: extra.y || 0, yaw, phase: arr.length * 1.73, route: extra.route }; arr.push(a); return a;
}

// ---------- Community hall (reference: "Sảnh nông trại gà tối giản") ----------
const HALL = { hx: 16, hz: 21, wall: 2.6, front: 1.7, door: 4.4 };
function plankWall(g, x1, z1, x2, z2, h) {
  const len = Math.hypot(x2 - x1, z2 - z1), w = new T.Group();
  w.position.set((x1 + x2) / 2, 0, (z1 + z2) / 2); w.rotation.y = -Math.atan2(z2 - z1, x2 - x1); g.add(w);
  const rows = Math.round(h / .43);
  for (let i = 0; i < rows; i++) box(w, len, .39, .16, timber([0x8d6440, 0x96704a, 0x89603d][i % 3]), 0, .22 + i * .43, 0);
  box(w, len + .1, .16, .34, timber(C.edge), 0, rows * .43 + .06, 0);
  return w;
}
function hall(g, data, cosmetics, info) {
  const { hx, hz } = HALL;
  ground(g, hx * 2, hz * 2, 0xc4a379, 'earth', .01);
  // A worn straw runner marks the aisle from the door to the stage.
  box(g, 3.4, .02, 30, textured('straw', 0xcfb27a, [1, 8]), 0, .025, 1.5);
  // Walls: tall at the back and sides, low at the front like the reference fence.
  const walls = [[-hx, -hz, hx, -hz, HALL.wall], [-hx, -hz, -hx, hz, HALL.wall], [hx, -hz, hx, hz, HALL.wall], [-hx, hz, -HALL.door, hz, HALL.front], [HALL.door, hz, hx, hz, HALL.front]];
  for (const [x1, z1, x2, z2, h] of walls) {
    const side = new T.Group(); g.add(side); plankWall(side, x1, z1, x2, z2, h);
    const len = Math.hypot(x2 - x1, z2 - z1), n = Math.round(len / 4.2);
    for (let i = 0; i <= n; i++) {
      const x = x1 + (x2 - x1) * i / n, z = z1 + (z2 - z1) * i / n;
      box(side, .5, h + .7, .5, timber(C.edge), x, (h + .7) / 2, z);
      box(side, .62, .14, .62, timber(C.light), x, h + .77, z);
    }
    occluder(info, side);
  }
  // Doorway with the farm name, seen when walking in and from the social camera.
  const door = new T.Group(); g.add(door);
  for (const sx of [-1, 1]) { box(door, .62, 4.8, .62, timber(C.edge), sx * HALL.door, 2.4, hz); lantern(door, sx * (HALL.door - .7), 3.1, hz + .1); }
  box(door, HALL.door * 2 + 1.2, .4, .6, timber(C.wood), 0, 4.75, hz);
  plaque(door, 'CHUỒNG GÀ HYPEX', 0, 4.1, hz + .36, 5.4, 0);
  occluder(info, door);
  // Owner's stage: deck top is exactly y=1.2 to match the host transform from the server.
  const sz = -17;
  box(g, 11, 1.05, 4, timber(C.edge), 0, .525, sz);
  for (let i = 0; i < 8; i++) box(g, 11.2, .15, .48, timber(i % 2 ? 0xb88f60 : 0xa98152), 0, 1.125, sz - 1.75 + i * .5);
  for (let i = 0; i < 12; i++) box(g, .5, .9, .08, timber(i % 2 ? 0x7a5232 : 0x6c482b), -5.25 + i * .95, .5, sz + 2.02);
  for (const sx of [-1, 1]) for (let i = 0; i < 3; i++) box(g, 1.6, .3 * (3 - i), .55, timber(i % 2 ? 0xa98152 : C.wood), sx * 6.3, .15 * (3 - i), sz + .9 + i * .02 - i * .55 + 1.2);
  nest(g, 0, -17.3, 1.75, 1.2);
  for (const sx of [-1, 1]) { flowerpot(g, sx * 4.6, -18.3, 1.3, 1.2); cylinder(g, .09, .12, 2.6, timber(C.edge), sx * 5.2, 2.5, -15.4, 8); lantern(g, sx * 5.2, 3.6, -15.4); }
  // Bunting and sign on the back wall.
  plaque(g, 'HYPEX', 0, 4.2, -hz + .2, 6, 0, { bg: '#7c4f2c', fg: '#fff1cf', font: '700 92px Georgia, serif' });
  const flagColors = [0xd9a441, 0xd86f4b, 0x6f9c78, 0xe9d6a5];
  for (const sx of [-1, 1]) for (let i = 0; i < 7; i++) {
    const x = sx * (3.6 + i * 1.6), y = 3.7 - Math.sin(i / 6 * Math.PI) * .45;
    const f = add(g, geometry('ConeGeometry', .32, .55, 3), flagColors[i % 4], x, y, -hz + .3); f.rotation.x = Math.PI; f.scale.z = .2;
  }
  // 40 player nests in two blocks of four per row, with a broad central aisle.
  for (const it of data.items) if (it.kind === 'nest') nest(g, it.x, it.z);
  // Feed and water troughs along the side walls, hay and sacks in the corners.
  trough(g, -14.6, -2, false, 8); trough(g, 14.6, -2, true, 8);
  hay(g, -13.4, -18.4, 1); hay(g, -11.9, -19.4, .75); hay(g, 13.6, -18.8, .9);
  barrel(g, 12.2, -19.3); barrel(g, 14.6, -14.8, true);
  for (let i = 0; i < 5; i++) sack(g, -14.6 + (i % 3) * .2, i < 3 ? 0 : .55, 10.4 + (i % 3) * 1.05 - (i > 2 ? .5 : 0), .2 + i);
  crate(g, 14.4, 10.6, .9, .3); crate(g, 14.6, 12.2, .7, -.2); barrel(g, 13.2, 12.4, true, .8);
  for (const sx of [-1, 1]) { flowerpot(g, sx * 5.8, 20.2, 1.15); lampPost(g, sx * 15.2, 4, 3.4); }
  // Seven residents: owner on stage (from server data) + guide, sweeper, sleeper, two neighbours, mail bird.
  resident(g, info.ambient, cosmetics, 'Gà Đón Khách', 5.6, 16.2, 'greet', 'vang', 'xanh', -.5, { equipped: { head: 'mu_rom' } });
  const sweep = resident(g, info.ambient, cosmetics, 'Gà Quét Sân', -12.9, 6, 'sweep', 'kem', 'luc', 0, { route: [[-12.9, -12], [-12.9, 8]] });
  const broom = new T.Group(); sweep.model.add(broom); broom.position.set(.55, .15, .35); broom.rotation.z = -.3;
  cylinder(broom, .025, .025, 1.1, C.wood, 0, .5, 0, 6); const br = add(broom, geometry('ConeGeometry', .23, .35, 10), C.straw, 0, -.1, 0); br.rotation.x = Math.PI; sweep.broom = broom;
  // Residents rest in a few row nests and step aside into the aisle whenever a player comes for that nest.
  const at = id => data.items.find(i => i.id === id);
  const sitter = (id, name, action, color, shirt, yaw, extra = {}) => { const n = at(id); if (!n) return; const a = resident(g, info.ambient, cosmetics, name, n.x, n.z, action, color, shirt, yaw, { y: .05, ...extra }); a.nest = { x: n.x, z: n.z }; };
  sitter('nest_3_1', 'Gà Mê Ngủ', 'sleep', 'cam', 'none', Math.PI + .3);
  sitter('nest_1_5', 'Gà Hàng Xóm', 'chat', 'vang', 'do', Math.PI - .9, { route: [[5.9, -5.6], [13.2, -4.4]] });
  sitter('nest_1_6', 'Gà Hàng Xóm', 'chat', 'nau', 'tim', Math.PI + .9);
  sitter('nest_0_2', 'Gà Ham Đọc', 'chat', 'trang', 'xanh', Math.PI, { equipped: { eye: 'kinh_tron' } });
  const bird = buildBird(); bird.position.set(-7.1, .25, 16.4); bird.scale.setScalar(.8); bird.rotation.y = .4; bird.userData.dynamic = true; g.add(bird);
  info.ambient.push({ model: bird, name: 'Chim Đưa Thư', action: 'perch', x: -7.1, z: 16.4, baseY: .25, phase: 9 });
  // Countryside around the coop for views over the low walls.
  grass(g, hx - .6, hz - .6, 70, 41, (x, z) => Math.abs(x) < hx - 2.4 && z > -15 && z < 18);
  surround(g, hx + 6, hz + 6, 77, [[0, hz + 8, 9]], .8);
  info.handles = new Set(['nest', 'npc_guide', 'trough']);
  info.camera = { focus: [0, 1, 6], pitch: .8, radius: 31, yaw: 0 };
  info.npcCount = info.ambient.length + 1;
}

// ---------- Open countryside: the seamless map between the areas ----------
// Roads run from the plaza (hub) to each area's entry; trees, fields and lamps fill the land between.
function countryside(g, data, cosmetics, info) {
  const W = data.halfX, D = data.halfZ, hub = data.hub || { x: 0, z: 0 }, areas = data.areas || [];
  ground(g, W * 2 + 120, D * 2 + 120, 0xa3b26f, 'grass', -.02);
  const ringR = Math.min(10.5, ...areas.filter((a) => a.entry).map((a) => Math.hypot(a.entry.x - hub.x, a.entry.z - hub.z) * .6)), ring = (e) => { const l = Math.hypot(e.x - hub.x, e.z - hub.z) || 1; return { x: hub.x + (e.x - hub.x) / l * ringR, z: hub.z + (e.z - hub.z) / l * ringR }; };
  const roads = areas.filter((a) => a.entry).map((a) => {
    const r = ring(a.entry), into = { x: a.entry.x + (a.cx - a.entry.x) * .25, z: a.entry.z + (a.cz - a.entry.z) * .25 };
    path(g, [[r.x, r.z], [(r.x + a.entry.x) / 2, (r.z + a.entry.z) / 2], [a.entry.x, a.entry.z], [into.x, into.z]], a.id === 'station' ? 6 : 4.8);
    return { a, r };
  });
  const nearRoad = (x, z, m) => roads.some(({ a, r }) => segDist(x, z, r.x, r.z, a.entry.x, a.entry.z) < m);
  const inArea = (x, z, m) => areas.some((a) => Math.abs(x - a.cx) < a.halfX + m && Math.abs(z - a.cz) < a.halfZ + m);
  const free = (x, z, m = 6) => Math.hypot(x - hub.x, z - hub.z) > 18 && !nearRoad(x, z, m) && !inArea(x, z, m + 3);
  // Plaza with the old banyan (only where the server placed its trunk as an obstacle) and benches.
  const plaza = add(g, geometry('CircleGeometry', ringR, 48), textured('earth', 0xd1b688, [3, 3]), hub.x, .045, hub.z, { shadow: false }); plaza.rotation.x = -Math.PI / 2;
  const edge = add(g, geometry('TorusGeometry', ringR + .1, .2, 6, 64), 0xab9b78, hub.x, .05, hub.z); edge.rotation.x = Math.PI / 2;
  if ((data.obstacles || []).some((o) => Math.hypot(o.x - hub.x, o.z - hub.z) < 3)) {
    tree(g, hub.x, hub.z, 1.65, 0x71964b);
    const r2 = add(g, geometry('TorusGeometry', 2.5, .35, 8, 28), 0xb3ab95, hub.x, .25, hub.z); r2.rotation.x = Math.PI / 2;
    flowers(g, hub.x, hub.z, 2.2, 36, 17);
  }
  for (const a of [.3, 1.7, 3.1, 4.7]) {
    const bench = new T.Group(); bench.position.set(hub.x + Math.cos(a) * 6.5, 0, hub.z + Math.sin(a) * 6.5); bench.rotation.y = -a + Math.PI / 2; g.add(bench);
    box(bench, 2.2, .14, .65, timber(C.light), 0, .6, 0); for (const sx of [-1, 1]) box(bench, .18, .6, .6, timber(C.edge), sx * .8, .3, 0);
    box(bench, 2.2, .45, .12, timber(C.wood), 0, 1.05, -.3);
  }
  // Direction post: one arm per road, pointing down it.
  const post = new T.Group(); post.position.set(hub.x + 4.6, 0, hub.z + 7.4); g.add(post);
  cylinder(post, .12, .15, 4.4, timber(C.edge), 0, 2.2, 0, 8);
  roads.slice(0, 9).forEach(({ a }, i) => {
    const ang = Math.atan2(a.entry.x - post.position.x, a.entry.z - post.position.z), arm = new T.Group(); arm.position.y = 4 - i * .4; arm.rotation.y = ang - Math.PI / 2; post.add(arm);
    plaque(arm, a.title, 1.15, 0, 0, 2.3, 0, { font: '600 66px Georgia, serif', bg: '#6b4a2c' });
  });
  // Lamps along every road, on alternating sides.
  for (const { a, r } of roads) {
    const len = Math.hypot(a.entry.x - r.x, a.entry.z - r.z), nx = -(a.entry.z - r.z) / len, nz = (a.entry.x - r.x) / len;
    for (let d = 14, k = 0; d < len - 4; d += 20, k++) { const t = d / len, side = k % 2 ? 1 : -1; lampPost(g, r.x + (a.entry.x - r.x) * t + nx * 4 * side, r.z + (a.entry.z - r.z) * t + nz * 4 * side, 3.2); }
  }
  // Farmland, trees and a pond in the open land; nothing lands on a road or inside an area.
  const rng = random(391 + Math.round(W));
  let fields = 0;
  for (let i = 0; i < 400 && fields < 7; i++) {
    const x = hub.x + (rng() * 2 - 1) * (W - 14), z = hub.z + (rng() * 2 - 1) * (D - 14), fw = 9 + rng() * 6, fd = 7 + rng() * 4;
    if (![[-1, -1], [1, -1], [-1, 1], [1, 1], [0, 0]].every(([sx, sz]) => free(x + sx * fw / 2, z + sz * fd / 2, 5))) continue;
    fields++;
    box(g, fw, .1, fd, 0x7d5f3c, x, .05, z);
    const crop = [0x7aa05a, 0x93ad55, 0xc7b04f][fields % 3];
    for (let row = -fd / 2 + .8; row < fd / 2; row += 1.4) for (let c = -fw / 2 + .7; c < fw / 2; c += 1.1) bush(g, x + c, z + row, .32, crop);
    fence(g, [[x - fw / 2 - .6, z - fd / 2 - .6], [x + fw / 2 + .6, z - fd / 2 - .6], [x + fw / 2 + .6, z + fd / 2 + .6]], 1.1);
  }
  for (let i = 0, n = 0; i < 900 && n < Math.round(W * D / 110); i++) {
    const x = hub.x + (rng() * 2 - 1) * (W + 30), z = hub.z + (rng() * 2 - 1) * (D + 30);
    if (!free(x, z, 6)) continue;
    n++;
    if (n % 3) tree(g, x, z, .75 + rng() * .8, [0x6e8c4b, 0x7d9853, 0x66844f, 0x86a05a][n % 4]); else bush(g, x, z, 1 + rng() * .6);
  }
  for (let i = 0, n = 0; i < 300 && n < 4; i++) {
    const x = hub.x + (rng() * 2 - 1) * (W - 20), z = hub.z + (rng() * 2 - 1) * (D - 20);
    if (!free(x, z, 12)) continue;
    n++;
    const pond = add(g, geometry('CircleGeometry', 5.5, 36), material(0x6aafb6, { roughness: .2, metalness: .1 }), x, .04, z, { shadow: false }); pond.rotation.x = -Math.PI / 2; pond.scale.set(1.3, 1, 1);
    for (let k = 0; k < 10; k++) { const a = k * .63; rock(g, x + Math.cos(a) * 7.4, z + Math.sin(a) * 5.8, .6); }
    flowers(g, x + 6, z + 6, 2.5, 20, 40 + n);
  }
  // Rolling hills past the edge of the map hide where walking stops.
  for (let a = 0; a < Math.PI * 2; a += .21) {
    const hx = hub.x + Math.cos(a) * (W + 26), hz = hub.z + Math.sin(a) * (D + 26), s = 14 + (a * 37 % 9);
    const hill = add(g, geometry('SphereGeometry', 1, 16, 10), [0x7f9a59, 0x8aa462, 0x748f50][Math.floor(a * 10) % 3], hx, -s * .45, hz, { shadow: false }); hill.scale.set(s * 1.6, s, s * 1.6);
  }
  grass(g, W, D, Math.round(W * D / 18), 83, (x, z) => !free(x, z, 3.4), hub.x, hub.z);
  flowers(g, hub.x - 14, hub.z - 12, 4, 40, 3); flowers(g, hub.x + 16, hub.z + 12, 4, 40, 4);
  resident(g, info.ambient, cosmetics, 'Gà Dạo Làng', hub.x - 8, hub.z + 9, 'walk', 'kem', 'luc', .6, { route: [[hub.x - 8, hub.z + 9], [hub.x - 14, hub.z + 18]] });
  resident(g, info.ambient, cosmetics, 'Gà Chăm Cây', hub.x + 7, hub.z - 10, 'greet', 'vang', 'do', 2);
  resident(g, info.ambient, cosmetics, 'Gà Ngồi Ghế', hub.x + Math.cos(1.7) * 6.5, hub.z + Math.sin(1.7) * 6.5 + .1, 'chat', 'nau', 'xanh', Math.PI + .2, { y: .35 });
  info.camera = { pitch: .62, radius: 17 };
}
function segDist(x, z, ax, az, bx, bz) {
  const dx = bx - ax, dz = bz - az, t = Math.max(0, Math.min(1, ((x - ax) * dx + (z - az) * dz) / (dx * dx + dz * dz || 1)));
  return Math.hypot(x - ax - dx * t, z - az - dz * t);
}

function neighborhood(g, data, cosmetics, info) {
  ground(g, 48, 48, 0xa7ad73, 'grass', .004);
  path(g, [[0, 22], [0, 10], [0, -10], [0, -22]], 5.5);
  for (const sx of [-1, 1]) for (const z of [-12, 4]) {
    path(g, [[0, z + 3], [sx * 6, z + 3], [sx * 11, z + 3]], 3);
    const coop = buildCoop(sx < 0 ? 'do' : 'luc'); coop.position.set(sx * 13, 0, z); coop.rotation.y = -sx * Math.PI / 2; coop.scale.setScalar(2.8); g.add(coop);
    fence(g, [[sx * 6, z - 6], [sx * 19, z - 6], [sx * 19, z + 7]], 1.1);
    flowerpot(g, sx * 8, z + 5, 1.2); tree(g, sx * 18, z - 4, 1); flowers(g, sx * 15, z + 5, 2, 18, 30 + z);
  }
  for (const z of [-13, 6]) for (const sx of [-1, 1]) lampPost(g, sx * 3.6, z, 3.1);
  // Guild hall frontage behind its notice board.
  const hallG = new T.Group(); hallG.position.set(12, 0, -16); g.add(hallG);
  box(hallG, 7, 3, 4, timber(0x9b724b), 0, 1.5, 0); const roof = add(hallG, geometry('ConeGeometry', 5.4, 2.4, 4), 0xa04a3a, 0, 4.2, 0); roof.rotation.y = Math.PI / 4; roof.scale.z = .7;
  for (const sx of [-1, 1]) { box(hallG, .12, 5.4, .12, C.edge, sx * 2.6, 2.7, 2.4); const flag = box(hallG, .9, .6, .04, 0xd9a441, sx * 2.6 + sx * .48, 5, 2.4); flag.rotation.y = .1; }
  surround(g, 22, 22, 22, [[0, 24, 7]]);
  grass(g, 22, 22, 220, 22, (x, z) => Math.abs(x) < 3.5 || Math.abs(z - 7) < 2 || Math.abs(z + 9) < 2);
  resident(g, info.ambient, cosmetics, 'Gà Hàng Xóm', 6, 17, 'greet', 'vang', 'xanh', -.7);
  resident(g, info.ambient, cosmetics, 'Gà Tưới Hoa', -8.5, 9.2, 'sweep', 'kem', 'do', 1.2);
  info.camera = { pitch: .62, radius: 16 };
}
function park(g, data, cosmetics, info) {
  ground(g, 56, 52, 0xa2b06f, 'grass', .004);
  path(g, [[0, 24], [-9, 12], [-14, -3], [-8, -15], [5, -18], [17, -7], [16, 12], [0, 24]], 3.5);
  const pond = add(g, geometry('CircleGeometry', 10.2, 48), material(0x6aafb6, { roughness: .2, metalness: .12 }), 0, .04, -2, { shadow: false }); pond.rotation.x = -Math.PI / 2; pond.scale.set(1, .75, 1);
  const rim = add(g, geometry('TorusGeometry', 10.4, .34, 8, 48), 0xb9b08a, 0, .07, -2); rim.rotation.x = Math.PI / 2; rim.scale.y = .75;
  for (let i = 0; i < 12; i++) {
    const a = i * .63, x = Math.cos(a) * (2 + i % 3), z = -2 + Math.sin(a) * 4;
    const leaf = add(g, geometry('CircleGeometry', .35, 10), 0x6c9655, x, .07, z); leaf.rotation.x = -Math.PI / 2;
    if (i % 3 === 0) { const flower = sphere(g, .15, 0xe9b0a8, x, .14, z); flower.scale.y = .5; }
  }
  // Wooden jetty and festival tents.
  for (let i = 0; i < 6; i++) box(g, 1.6, .12, .4, timber(C.light), 8.2 + i * .0, .32, 4.4 - i * .45);
  for (const x of [-4, 0, 4]) {
    const tent = new T.Group(); tent.position.set(x, 0, -21); g.add(tent);
    for (const sx of [-1, 1]) cylinder(tent, .06, .06, 2.7, C.wood, sx * 1.3, 1.35, 0, 6);
    const canopy = add(tent, geometry('ConeGeometry', 2, 1.1, 4), x === 0 ? 0x72958c : 0xc18b69, 0, 3, 0); canopy.rotation.y = Math.PI / 4;
    box(tent, 2.6, .12, 1, timber(C.light), 0, 1, 0);
  }
  flowers(g, -16, 8, 3, 30, 8); flowers(g, 15, -14, 3, 30, 9);
  surround(g, 26, 24, 27, [[0, 26, 6]]);
  grass(g, 25, 23, 260, 27, (x, z) => Math.hypot(x, (z + 2) / .75) < 12);
  resident(g, info.ambient, cosmetics, 'Gà Ngắm Hồ', 11.8, 1.5, 'chat', 'kem', 'tim', -1.5);
  resident(g, info.ambient, cosmetics, 'Gà Hội Chợ', -6, -19, 'greet', 'vang', 'do', 0);
  info.camera = { pitch: .6, radius: 16 };
}

// ---------- Activity zones outside the coop (reference images 02–11) ----------
// Squircle patch of worked ground: reads as a clearing in the meadow rather than a square tile.
function zoneGround(g, data, color, kind = 'earth') {
  const hx = data.halfX + 2.5, hz = data.halfZ + 2.5, shape = new T.Shape();
  for (let i = 0; i <= 64; i++) {
    const a = i / 64 * Math.PI * 2, c = Math.cos(a), s = Math.sin(a);
    const x = Math.sign(c) * Math.pow(Math.abs(c), .5) * hx, y = Math.sign(s) * Math.pow(Math.abs(s), .5) * hz;
    i ? shape.lineTo(x, y) : shape.moveTo(x, y);
  }
  const m = add(g, new T.ShapeGeometry(shape), textured(kind, color, [1 / 6, 1 / 6], { roughness: 1 }), 0, .006, 0, { shadow: false });
  m.rotation.x = -Math.PI / 2;
}
function returnGap(data) { return data.entry ? [[data.entry.x, data.entry.z, 7]] : []; }

function garden(g, data, cosmetics, info) {
  zoneGround(g, data, 0xae9361);
  path(g, [[0, 9], [0, 5], [-3, 2], [-1, -2]], 2.6, 0xc9ae7d);
  // Raised beds and furrows between the dig mounds.
  for (const [x, z, w, d] of [[-7.6, -6.4, 4, 2], [5.6, -6.8, 5, 1.8], [8.6, 4.6, 2.4, 3.2]]) {
    box(g, w, .32, d, timber(C.edge), x, .16, z); box(g, w - .2, .06, d - .2, 0x6b4a2a, x, .33, z);
    for (let i = 0; i < Math.floor(w / .7); i++) bush(g, x - w / 2 + .5 + i * .7, z, .32, 0x7aa05a);
  }
  const wet = add(g, geometry('CircleGeometry', 3.4, 24), material(0x5e4329, { roughness: .6 }), 6.2, .02, -1.8, { shadow: false }); wet.rotation.x = -Math.PI / 2;
  cylinder(g, .85, 1, 1, timber(C.edge), -.5, .5, -6.3, 14); cylinder(g, .84, .84, .04, 0xcfa877, -.5, 1.01, -6.3, 14);
  // Tool rack, wheelbarrow, watering cans.
  box(g, .14, 1.8, .14, timber(C.edge), -9.4, .9, -3.6); box(g, .14, 1.8, .14, timber(C.edge), -9.4, .9, -1.2); box(g, .16, .16, 2.6, timber(C.light), -9.4, 1.6, -2.4);
  for (let i = 0; i < 3; i++) { box(g, .06, 1.3, .06, timber(C.light), -9.25, .95, -3.2 + i * .8); box(g, .3, .36, .05, 0x8c8c86, -9.25, .3, -3.2 + i * .8); }
  const wb = new T.Group(); wb.position.set(-8, 0, 3.4); wb.rotation.y = .6; g.add(wb);
  box(wb, 1, .45, 1.5, timber(C.wood), 0, .6, 0); box(wb, .85, .1, 1.3, 0x5f4328, 0, .8, 0); const wheel = add(wb, geometry('TorusGeometry', .26, .07, 8, 16), 0x3a3a34, 0, .28, .9); wheel.rotation.y = Math.PI / 2;
  barrel(g, -9.2, .2, true); barrel(g, 8.6, -7.4, true, .8);
  fence(g, [[-10.4, 9], [-10.4, -9], [10.4, -9], [10.4, 9]], 1);
  hay(g, -8.6, -7.6, .8);
  surround(g, data.halfX + 2, data.halfZ + 2, 51, returnGap(data));
  grass(g, data.halfX, data.halfZ, 120, 11, (x, z) => data.items.some(i => Math.hypot(i.x - x, i.z - z) < 1.6));
  resident(g, info.ambient, cosmetics, 'Gà Làm Vườn', 4, 6.4, 'sweep', 'nau', 'luc', 2.4);
  info.handles = new Set(['gate']);
  info.camera = { pitch: .66, radius: 15 };
}
function stall(g, x, z, w, roof, ry = 0) {
  const s = new T.Group(); s.position.set(x, 0, z); s.rotation.y = ry; g.add(s);
  shadow(s, 0, 0, w * .7, 1.6);
  box(s, w, .95, 1.2, timber(C.wood), 0, .48, 0); box(s, w + .2, .1, 1.4, timber(C.light), 0, 1, 0);
  for (const sx of [-1, 1]) for (const sz of [-.6, .6]) box(s, .14, 2.6, .14, timber(C.edge), sx * (w / 2), 1.3, sz);
  // Striped cloth awning.
  for (let i = 0; i < 6; i++) { const a = box(s, w / 6 + .02, .06, 2, i % 2 ? 0xf4ead2 : roof, -w / 2 + w / 12 + i * w / 6, 2.62, .2); a.rotation.x = .18; }
  for (let i = 0; i < 6; i++) { const f = add(s, geometry('ConeGeometry', w / 14, .3, 3), i % 2 ? roof : 0xf4ead2, -w / 2 + w / 12 + i * w / 6, 2.3, 1.2); f.rotation.x = Math.PI; f.scale.z = .3; }
  return s;
}
function market(g, data, cosmetics, info) {
  zoneGround(g, data, 0xc6a976);
  path(g, [[0, 10], [0, 2], [0, -2]], 4, 0xd8c39a);
  // Each server shop point gets its own stall facing the street.
  for (const it of data.items) {
    if (it.kind === 'shop_sell') { const s = stall(g, it.x, it.z - 1.6, 3.6, 0x8f9c6a); crate(s, -1, .2, .5); crate(s, -.3, 0.1, .45); for (let i = 0; i < 4; i++) { const w = add(s, geometry('TorusGeometry', .1, .035, 6, 10, Math.PI * 1.4), 0xd98a8a, .5 + i * .3, 1.12, 0); w.rotation.x = Math.PI / 2; } plaque(g, 'THU MUA GIUN', it.x, 3.3, it.z - .6, 2.8); }
    if (it.kind === 'shop_food') { const s = stall(g, it.x, it.z - 1.6, 3.8, 0xc9874b); for (const [bx, col] of [[-1.2, 0xe7a73a], [-.4, 0x8a6b3a], [.4, 0x6fa03a], [1.2, 0xc9a35a]]) { cylinder(s, .3, .22, .2, timber(C.light), bx, 1.15, 0, 12); cylinder(s, .27, .27, .04, col, bx, 1.26, 0, 12); } plaque(g, 'QUẦY THỨC ĂN', it.x, 3.3, it.z - .6, 2.8); }
    if (it.kind === 'shop_fashion') { const s = stall(g, it.x, it.z - 1.6, 3.6, 0x6f8fb0); box(s, 3, .05, .05, timber(C.edge), 0, 2.1, .5); for (const [bx, col] of [[-1, 0x3f86c9], [0, 0xd64b3a], [1, 0xe7c23a]]) box(s, .55, .6, .05, col, bx, 1.72, .5); cylinder(s, .3, .3, .04, 0xd9b65a, .8, 1.1, -.2, 14); plaque(g, 'THỜI TRANG', it.x, 3.3, it.z - .6, 2.8); }
  }
  for (const [x, z] of [[-10, -7], [-9.6, 4], [10, 4.6]]) { sack(g, x, 0, z, 1); sack(g, x + .7, 0, z + .3, 2); crate(g, x + .2, z - 1.2, .7, .3); }
  // Bunting across the street.
  for (let i = 0; i < 14; i++) { const f = add(g, geometry('ConeGeometry', .22, .4, 3), [0xd9a441, 0xd86f4b, 0x6f9c78][i % 3], -9 + i * 1.4, 3.6 - Math.sin(i / 13 * Math.PI) * .5, 1.4); f.rotation.x = Math.PI; f.scale.z = .2; }
  for (const sx of [-1, 1]) cylinder(g, .1, .12, 4, timber(C.edge), sx * 10, 2, 1.4, 8);
  lampPost(g, -4, 3, 3.2); lampPost(g, 4.4, 3, 3.2);
  surround(g, data.halfX + 2, data.halfZ + 2, 61, returnGap(data));
  grass(g, data.halfX, data.halfZ, 60, 12, (x, z) => Math.abs(x) < 4 || z < -3);
  resident(g, info.ambient, cosmetics, 'Gà Bán Rau', 0, -5.6, 'greet', 'vang', 'do', 0);
  resident(g, info.ambient, cosmetics, 'Gà Thợ May', 7, -5.6, 'chat', 'kem', 'tim', 0, { equipped: { eye: 'kinh_tron' } });
  resident(g, info.ambient, cosmetics, 'Gà Đi Chợ', -3.4, 3.2, 'walk', 'nau', 'xanh', 0, { route: [[-3.4, 3.2], [3.6, 3.2]] });
  info.handles = new Set(['shop_food', 'shop_fashion', 'gate']);
  info.camera = { pitch: .62, radius: 15 };
}
function arena(g, data, cosmetics, info) {
  zoneGround(g, data, 0xb98a52);
  const sand = add(g, geometry('CircleGeometry', 4.4, 40), textured('earth', 0xd8b77a, [2, 2]), 0, .03, 0, { shadow: false }); sand.rotation.x = -Math.PI / 2;
  const ring = add(g, geometry('TorusGeometry', 4.4, .18, 8, 48), 0xe8e0cc, 0, .08, 0); ring.rotation.x = Math.PI / 2;
  for (let i = 0; i < 10; i++) { const a = (i / 10) * Math.PI * 2; box(g, .2, .8, .2, timber(C.edge), Math.cos(a) * 4.9, .4, Math.sin(a) * 4.9); beam(g, [Math.cos(a) * 4.9, .62, Math.sin(a) * 4.9], [Math.cos(a + .628) * 4.9, .62, Math.sin(a + .628) * 4.9], .06, 0xd9c8a0); }
  // Tiered benches for spectators.
  for (const sx of [-1, 1]) for (let r = 0; r < 3; r++) { box(g, .8, .3 + r * .4, 7, timber(r % 2 ? C.light : C.wood), sx * (7.2 + r * .8), .15 + r * .2, 0); }
  for (let i = 0; i < 3; i++) { const f = add(g, geometry('ConeGeometry', .35, .6, 3), [0xd64b3a, 0xe7c23a, 0x3f86c9][i], -3 + i * 3, 3, -8.2); f.rotation.x = Math.PI; f.scale.z = .2; }
  for (const sx of [-1, 1]) cylinder(g, .1, .12, 4, timber(C.edge), sx * 5, 2, -8.4, 8);
  box(g, 10, .05, .05, timber(C.edge), 0, 3.25, -8.4);
  plaque(g, 'SÂN TỶ THÍ', 0, 4.4, -8.5, 4);
  surround(g, data.halfX + 2, data.halfZ + 2, 71, returnGap(data));
  resident(g, info.ambient, cosmetics, 'Gà Cổ Vũ', -7.2, 2, 'greet', 'vang', 'do', Math.PI / 2, { y: .45 });
  resident(g, info.ambient, cosmetics, 'Gà Cổ Vũ', 7.2, -1.4, 'chat', 'kem', 'xanh', -Math.PI / 2, { y: .45 });
  info.handles = new Set(['gate']);
  info.camera = { pitch: .7, radius: 15 };
}
function hatchery(g, data, cosmetics, info) {
  zoneGround(g, data, 0xc9a978);
  // Warm open-front house sheltering the incubator nests.
  const house = new T.Group(); house.position.set(0, 0, -5.8); g.add(house);
  box(house, 12, .2, 4, timber(C.light), 0, .1, 0);
  box(house, 12, 3.2, .3, timber(0x9b724b), 0, 1.6, -2); for (const sx of [-1, 1]) box(house, .3, 3.2, 4, timber(0x9b724b), sx * 6, 1.6, 0);
  for (const sx of [-1, 1]) { const r = box(house, 6.6, .18, 5, 0xc0703f, sx * 3.1, 3.7, .2); r.rotation.z = -sx * .28; }
  occluder(info, house);
  for (let i = 0; i < 4; i++) {
    const x = -4.2 + i * 2.8; nest(g, x, -6, .8, .2);
    for (let k = 0; k < 3; k++) { const e = sphere(g, .14, 0xf6ecd6, x + (k - 1) * .2, .5, -6 + (k % 2) * .14); e.scale.y = 1.3; }
    cylinder(g, .04, .04, 1.6, C.edge, x, 1.9, -6.6, 6); const lamp = add(g, geometry('ConeGeometry', .26, .3, 10, 1, true), material(0xe7a73a, { emissive: 0xc06010, emissiveIntensity: .4, side: T.DoubleSide }), x, 2.6, -6.2); lamp.rotation.x = Math.PI;
  }
  plaque(g, 'TRẠM ẤP TRỨNG', 0, 4.4, -3.6, 4);
  flowers(g, 5.6, 3.6, 2.2, 26, 61); barrel(g, 7.6, 3, true); hay(g, -8, 3.6, .7);
  surround(g, data.halfX + 2, data.halfZ + 2, 81, returnGap(data));
  info.handles = new Set(['gate']);
  info.camera = { pitch: .64, radius: 14 };
}
function busShelter(g, x, z, ry = 0) {
  const s = new T.Group(); s.position.set(x, 0, z); s.rotation.y = ry; g.add(s);
  for (const sx of [-1, 1]) box(s, .15, 2.6, .15, timber(C.edge), sx * 1.8, 1.3, -.6);
  box(s, 4.2, .14, 1.8, 0x5f8fb0, 0, 2.65, -.2); box(s, 3.4, .12, .5, timber(C.light), 0, .55, -.5); box(s, 3.6, 1.4, .08, timber(C.wood), 0, 1.5, -.9);
  plaque(s, 'BẾN CHIM CHÍP', 0, 3.1, -.2, 2.6);
  return s;
}
function road(g, x1, z1, x2, z2, w = 4) {
  path(g, [[x1, z1], [x2, z2]], w, 0x8b8578);
  const len = Math.hypot(x2 - x1, z2 - z1), n = Math.floor(len / 2.2);
  for (let i = 1; i < n; i++) { const t = i / n; const m = box(g, .16, .02, .9, 0xf4f0e6, x1 + (x2 - x1) * t, .05, z1 + (z2 - z1) * t); m.rotation.y = Math.atan2(x2 - x1, z2 - z1); }
}
function station(g, data, cosmetics, info, far = false) {
  zoneGround(g, data, far ? 0xa9a36a : 0xb59a6a);
  road(g, 0, data.halfZ + 3, 0, -data.halfZ - 3, 4);
  for (const it of data.items) {
    if (it.kind === 'bus_stop') { busShelter(g, it.x + 2, it.z - .6); const bus = buildBus(); bus.position.set(-.9, 0, it.z - 1.2); g.add(bus); g.userData.bus = bus; }
    if (it.kind === 'parking') for (let i = -1; i <= 1; i++) box(g, .1, .02, 2.4, 0xf4f0e6, it.x + i * 1.6, .05, it.z + 1.2);
    if (it.kind === 'road') { const sign = plaque(g, 'ĐƯỜNG LIÊN TRẠI →', it.x, 2, it.z - 1, 2.6, Math.PI / 2); sign.position.y = 2; cylinder(g, .08, .1, 2, C.edge, it.x, 1, it.z - 1, 6); }
  }
  // Rice fields on both sides.
  for (const [x, z, w, d] of [[-7.5, -5, 5, 4], [7.5, -5.5, 4, 3]]) {
    box(g, w, .06, d, 0x8fb04f, x, .03, z);
    const rng = random(Math.round(x * 10 + z));
    for (let i = 0; i < w * d * 2; i++) { const s = add(g, geometry('ConeGeometry', .05, .45, 3), 0xc8b64a, x + (rng() - .5) * w, .22, z + (rng() - .5) * d, { shadow: false }); s.rotation.z = (rng() - .5) * .3; }
  }
  if (far) {
    const stream = add(g, geometry('PlaneGeometry', 3, 60), material(0x5fa5c0, { roughness: .2 }), -data.halfX - 4, .04, 0, { shadow: false }); stream.rotation.x = -Math.PI / 2;
  }
  lampPost(g, -2.8, 4.6, 3.2); lampPost(g, 2.8, -6, 3.2);
  surround(g, data.halfX + 2, data.halfZ + 2, far ? 91 : 93, [...returnGap(data), [0, -data.halfZ - 2, 4], [0, data.halfZ + 2, 4]]);
  info.handles = new Set(['bus_stop', 'parking', 'road', 'gate']);
  info.camera = { pitch: .62, radius: 15 };
}
function garage(g, data, cosmetics, info) {
  zoneGround(g, data, 0xa8a196, 'stone');
  const shop = new T.Group(); shop.position.set(0, 0, -7); g.add(shop);
  box(shop, 9, 3, .3, timber(0x8a5a30), 0, 1.5, -.6); box(shop, 9.6, .2, 3, 0xd05a42, 0, 3.1, .6); for (const sx of [-1, 1]) box(shop, .2, 3, .2, timber(C.edge), sx * 4.6, 1.5, 1.9);
  plaque(shop, 'XƯỞNG XE DÊ THỢ', 0, 3.7, 2.1, 3.6);
  occluder(info, shop);
  for (const [k, x] of [['xe_dap', -5.5], ['xe_may', -2], ['o_to', 2.2]]) { const v = buildVehicle(k); v.position.set(x, 0, -5.2); v.rotation.y = .4; g.add(v); }
  for (let i = -1; i <= 2; i++) box(g, .1, .02, 2.6, 0xf4f0e6, 4 + i * 1.7, .03, -1.2);
  barrel(g, 7.5, -6.5); barrel(g, 8.1, -5.7); crate(g, -7.6, -1.2, .8, .4);
  surround(g, data.halfX + 2, data.halfZ + 2, 101, returnGap(data));
  info.handles = new Set(['parking', 'gate']);
  info.camera = { pitch: .64, radius: 14 };
}
function vensuoi(g, data, cosmetics, info) {
  zoneGround(g, data, 0x9c8a55, 'grass');
  const water = add(g, geometry('PlaneGeometry', 2.6, data.halfZ * 2 + 6), material(0x5fa5c0, { roughness: .18, metalness: .1 }), 9.4, .05, 0, { shadow: false }); water.rotation.x = -Math.PI / 2;
  for (const sx of [-1, 1]) for (let z = -data.halfZ - 3; z < data.halfZ + 3; z += 1.3) { rock(g, 9.4 + sx * 1.6, z, .5); if ((z * 7) % 3 < 1) cylinder(g, .03, .03, .9, 0x6f8f3a, 9.4 + sx * 2, .45, z + .4, 4); }
  const bridge = new T.Group(); bridge.position.set(9.4, 0, -5); g.add(bridge);
  for (let i = 0; i < 8; i++) box(bridge, .4, .12, 1.6, timber(i % 2 ? C.light : C.wood), -1.5 + i * .42, .38, 0);
  for (const sz of [-1, 1]) box(bridge, 3.6, .1, .1, timber(C.edge), 0, .9, sz * .76);
  const mud = add(g, geometry('CircleGeometry', 4.5, 24), material(0x5e4329, { roughness: .5 }), -3, .02, -3.2, { shadow: false }); mud.rotation.x = -Math.PI / 2;
  flowers(g, -6, 5, 3, 30, 71);
  surround(g, data.halfX + 2, data.halfZ + 2, 111, returnGap(data));
  grass(g, data.halfX, data.halfZ, 140, 13, (x, z) => x > 7.5 || Math.hypot(x + 3, z + 3.2) < 4.6);
  resident(g, info.ambient, cosmetics, 'Gà Câu Cá', 7.4, 2.4, 'chat', 'nau', 'xanh', Math.PI / 2);
  info.handles = new Set(['gate']);
  info.camera = { pitch: .62, radius: 15 };
}

// Batch decorative meshes, retaining animated and camera-faded objects separately.
function batchStatic(g, info, chunk = Infinity) {
  for (const occ of info?.occluders || []) if (occ.parent) batchStatic(occ);
  g.updateMatrixWorld(true);
  // r128 bounds an InstancedMesh by its base geometry only, so fade boxes are measured before instancing.
  if (g.userData.dynamic && !g.userData.box) { g.userData.box = new T.Box3().setFromObject(g); g.userData.fade = 1; }
  const buckets = new Map(), inv = new T.Matrix4().copy(g.matrixWorld).invert(), wp = new T.Vector3();
  g.traverse(o => {
    if (!o.isMesh || o.isInstancedMesh || Array.isArray(o.material) || o.material.transparent) return;
    for (let p = o.parent; p && p !== g; p = p.parent) if (p.userData.dynamic) return;
    // The open countryside is batched in chunks so the frustum and far plane skip distant patches.
    o.getWorldPosition(wp);
    const k = o.geometry.uuid + ':' + o.material.uuid + ':' + o.castShadow + (chunk < Infinity ? ':' + Math.floor(wp.x / chunk) + ',' + Math.floor(wp.z / chunk) : '');
    if (!buckets.has(k)) buckets.set(k, []); buckets.get(k).push(o);
  });
  const m = new T.Matrix4(), b = new T.Box3();
  for (const list of buckets.values()) {
    if (list.length < 3) continue;
    // Each batch gets its own geometry whose bounding sphere covers every instance, so culling works.
    const base = list[0].geometry; base.boundingBox || base.computeBoundingBox();
    const bounds = new T.Box3(), inst = new T.InstancedMesh(base.clone(), list[0].material, list.length);
    inst.castShadow = list[0].castShadow; inst.receiveShadow = true;
    list.forEach((o, i) => { m.multiplyMatrices(inv, o.matrixWorld); inst.setMatrixAt(i, m); bounds.union(b.copy(base.boundingBox).applyMatrix4(m)); o.parent.remove(o); });
    inst.geometry.boundingSphere = bounds.getBoundingSphere(new T.Sphere());
    inst.instanceMatrix.needsUpdate = true; g.add(inst);
  }
}

const BUILDERS = { lobby: hall, village: countryside, vs_field: countryside, neighborhood, park, garden, market, arena, hatchery, station, garage, vensuoi, vs_station: (g, d, c, i) => station(g, d, c, i, true) };
export function buildScenery(id, group, data, cosmetics) {
  const info = { handled: false, ambient: [], occluders: [], handles: new Set(), camera: null };
  const fn = BUILDERS[id];
  if (!fn) return info;
  info.handled = true; fn(group, data, cosmetics, info); batchStatic(group, info, data.outdoor ? 60 : Infinity); return info;
}

// Ambient life. Each resident keeps its base height so a stage or nest never sinks it into the floor.
export function updateScenery(sceneData, dt, time, actors = []) {
  if (!sceneData) return;
  const t = time == null ? performance.now() / 1000 : time;
  for (const a of sceneData.ambient || []) {
    const m = a.model, p = m.userData.parts, tt = t + a.phase;
    if (a.nest) {
      // A player near this nest has priority: stand in the aisle beside it, facing the nest.
      const busy = actors.some(q => Math.hypot(q.x - a.nest.x, q.z - a.nest.z) < 1.7);
      a.yield = Math.max(0, Math.min(1, (a.yield || 0) + (busy ? dt : -dt * .5) * 2.5));
      if (a.yield > 0) {
        const sx = a.nest.x, sz = a.nest.z + 2.15, k = a.yield;
        m.position.set(a.nest.x + (sx - a.nest.x) * k, a.baseY * (1 - k), a.nest.z + (sz - a.nest.z) * k);
        m.rotation.y = Math.PI; p.legs[0].rotation.x = p.legs[1].rotation.x = 0; p.rig.position.y = 0; p.head.rotation.x = .1;
        continue;
      }
    }
    if (sceneData.reducedMotion) continue;
    if (a.action === 'perch') { m.rotation.y = a.yaw + Math.sin(tt * .5) * .5; m.position.y = a.baseY + Math.abs(Math.sin(tt * 3)) * (Math.sin(tt * .4) > .9 ? .08 : 0); continue; }
    p.rig.position.y = Math.sin(tt * 1.8) * .012; p.rig.rotation.x = 0;
    p.head.rotation.y = Math.sin(tt * .3) * .25; p.head.rotation.x = 0;
    p.wings[0].rotation.z = p.wings[1].rotation.z = 0;
    let walking = false;
    if (a.route) {
      // Cycle: rest at home → walk out → pause → walk back. Sweepers keep walking.
      const [h, o] = a.route, period = a.action === 'sweep' ? 24 : 40, u = (tt % period) / period;
      const leg = a.action === 'sweep' ? [0, .5, .5, 1] : [.45, .6, .78, .93];
      let k = 0;
      if (u < leg[0]) k = 0; else if (u < leg[1]) { k = (u - leg[0]) / (leg[1] - leg[0]); walking = true; } else if (u < leg[2]) k = 1; else if (u < leg[3]) { k = 1 - (u - leg[2]) / (leg[3] - leg[2]); walking = true; } else k = 0;
      const x = h[0] + (o[0] - h[0]) * k, z = h[1] + (o[1] - h[1]) * k;
      if (walking) { const back = u >= leg[2]; m.rotation.y = Math.atan2((back ? h[0] - o[0] : o[0] - h[0]), (back ? h[1] - o[1] : o[1] - h[1])); }
      else m.rotation.y = k > .5 ? m.rotation.y : a.yaw;
      m.position.set(x, k > .02 ? 0 : a.baseY, z);
      if (!walking && k > .5 && a.action !== 'sweep') p.head.rotation.x = .5 + Math.abs(Math.sin(tt * 6)) * .3; // drinking at the trough
    }
    const seatedHome = !walking && (!a.route || m.position.distanceTo(new T.Vector3(a.x, m.position.y, a.z)) < .1) && a.baseY > 0;
    if (walking) { p.legs[0].rotation.x = Math.sin(tt * 9) * .5; p.legs[1].rotation.x = -Math.sin(tt * 9) * .5; p.rig.position.y = Math.abs(Math.sin(tt * 18)) * .04; }
    else if (seatedHome || a.action === 'sleep') { p.legs[0].rotation.x = p.legs[1].rotation.x = -1.2; p.rig.position.y = -.1; }
    else { p.legs[0].rotation.x = p.legs[1].rotation.x = 0; }
    if (a.action === 'sleep') { p.rig.position.y = -.2 + Math.sin(tt * 1.2) * .01; p.head.rotation.x = .55; p.head.rotation.y = 0; if (Math.sin(tt * .25) > .97) p.head.rotation.x = -.3; }
    else if (a.action === 'greet' && !walking) { p.wings[1].rotation.z = Math.sin(tt * .6) > .8 ? .85 + Math.sin(tt * 8) * .2 : .05; }
    else if (a.action === 'sweep') { p.head.rotation.x = .25; p.wings[1].rotation.z = .4 + Math.sin(tt * 3) * .2; if (a.broom) a.broom.rotation.x = Math.sin(tt * 3) * .35; }
    else if (a.action === 'chat' && !walking) { p.head.rotation.x = Math.sin(tt * 2) * .08; p.wings[0].rotation.z = Math.sin(tt * .7) > .6 ? -.5 : -.05; }
    else if (a.action === 'walk' && !a.route) { const dz = Math.sin(tt * .22) * 2; m.position.set(a.x, a.baseY, a.z + dz); }
  }
}

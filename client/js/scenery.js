// Art-directed 3D farm spaces. Geometry stays native and interactive at every angle.
// Coordinates match the authoritative zone data from server/world.go (items, obstacles, bounds).
import { mat, buildChicken, dress, buildCoop, buildBird, buildBus, buildVehicle } from './models.js';
import { farmMaterial, createThatchRoof, createStrawNest, dressGround } from './farm-art.js';

const T = window.THREE;
const geo = new Map(), materials = new Map();
const C = { wood: 0x7c5233, edge: 0x4d321e, light: 0xa77b4b, cream: 0xf3dfaf, straw: 0xd5ac53, grass: 0x6f8a45, sage: 0x56754b, leaf: 0x587a3c, dirt: 0x8a6a45, path: 0xa58058, blue: 0x5d8f9e, red: 0xaf5542 };
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
function timber(color = C.wood) { const key='bo2wood'+color;if(!materials.has(key)){const m=farmMaterial('wood').clone();m.color.setHex(color).lerp(new T.Color(0xffffff),.5);materials.set(key,m);}return materials.get(key); }
function groundMaterial(kind,color,w,d){const m=farmMaterial(kind==='grass'?'grass':'earth').clone();m.map=m.map.clone();m.map.repeat.set(w/12,d/12);m.map.needsUpdate=true;return m;}
function ground(g, w, d, color = C.dirt, kind = 'earth', y = 0) {
  const m = new T.Mesh(geometry('PlaneGeometry', w, d), groundMaterial(kind,color,w,d));
  m.rotation.x = -Math.PI / 2; m.position.y = y; m.receiveShadow = true; g.add(m); return m;
}
let shadowTex;
function shadowMaterial() {
  if (!shadowTex) {
    const cv = document.createElement('canvas'); cv.width = cv.height = 64;
    const c = cv.getContext('2d'), grad = c.createRadialGradient(32, 32, 2, 32, 32, 31);
    grad.addColorStop(0, 'rgba(44,31,10,.3)'); grad.addColorStop(1, 'rgba(44,31,10,0)'); c.fillStyle = grad; c.fillRect(0, 0, 64, 64); shadowTex = new T.CanvasTexture(cv);
  }
  const key = 'groundShadow'; if (!materials.has(key)) materials.set(key, new T.MeshBasicMaterial({ map: shadowTex, transparent: true, depthWrite: false }));
  return materials.get(key);
}
function shadow(g, x, z, rx, rz = rx, y = 0.018) {
  const m = add(g, geometry('PlaneGeometry', 2, 2), shadowMaterial(), x, y, z, { shadow: false }); m.rotation.x = -Math.PI / 2; m.scale.set(rx, rz, 1); return m;
}
// Fade-able architecture: the camera makes these translucent only while they hide the chicken.
export function occluder(info, group) {
  group.userData.dynamic = true;
  const clones = new Map();
  const own = (m) => { if (!clones.has(m)) clones.set(m, m.clone()); return clones.get(m); };
  group.traverse(o => { if (o.isMesh) o.material = Array.isArray(o.material) ? o.material.map(own) : own(o.material); });
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
  const n = createStrawNest(Math.round((x + 60) * 101 + (z + 60) * 53));
  n.position.set(x, yy, z); n.scale.setScalar(size); g.add(n);
  shadow(g, x, z, 1.2 * size, 1.2 * size, yy + .025);
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
let glowTex;
// Quầng sáng ấm quanh đèn lồng: mọi đèn của một khu gộp chung một lượt vẽ (Points), không thêm nguồn sáng.
function glowMaterial(){
 if(!glowTex){const cv=document.createElement('canvas');cv.width=cv.height=64;const c=cv.getContext('2d'),gr=c.createRadialGradient(32,32,0,32,32,32);gr.addColorStop(0,'rgba(255,214,128,.95)');gr.addColorStop(.25,'rgba(255,180,80,.45)');gr.addColorStop(1,'rgba(255,160,60,0)');c.fillStyle=gr;c.fillRect(0,0,64,64);glowTex=new T.CanvasTexture(cv);}
 if(!materials.has('glow'))materials.set('glow',new T.PointsMaterial({map:glowTex,size:1.5,transparent:true,depthWrite:false,blending:T.AdditiveBlending,toneMapped:false,fog:true}));
 return materials.get('glow');
}
function poolMaterial(){
 if(!materials.has('pool'))materials.set('pool',new T.MeshBasicMaterial({color:0xffc35b,transparent:true,opacity:.045,depthWrite:false}));
 return materials.get('pool');
}
const LANTERN_LIT={emissive:0xffa53a,emissiveIntensity:1.35};
// Đêm (PLAN_LOGIC_GAME 16): k = 0 trưa … 1 nửa đêm. Đèn lồng sáng rực, quầng to hơn, in vũng sáng ấm trên đất.
export function setNightGlow(k){
 material(0xffc76a,LANTERN_LIT).emissiveIntensity=1.35+1.8*k;
 const g=glowMaterial();g.size=1.5+1.6*k;g.opacity=.75+.25*k;
 poolMaterial().opacity=.045+.2*k;
}
function lantern(g,x,y,z){
 const p=new T.Group();p.position.set(x,y,z);g.add(p);const dark=0x2f2416;
 p.userData.glow=true;
 cylinder(p,.16,.25,.17,dark,0,.36,0,4);
 box(p,.32,.48,.3,material(0xffc76a,LANTERN_LIT),0,0,0);
 for(const sx of [-1,1])for(const sz of [-1,1])box(p,.025,.55,.025,dark,sx*.17,0,sz*.16);
 box(p,.4,.06,.36,dark,0,-.28,0);cylinder(p,.055,.055,.12,dark,0,.5,0,8);
 // A small warm pool on nearby ground, with no extra shadow-casting lights.
 const glow=add(p,geometry('CircleGeometry',1,24),poolMaterial(),0,-y+.027,0,{shadow:false});glow.rotation.x=-Math.PI/2;glow.scale.set(1.5,1.1,1);glow.userData.pool=true;
 return p;
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
// Cây và thông được ghi vào sink rồi dựng bằng InstancedMesh theo từng ô bản đồ (instanceTrees), nhờ vậy
// có thể làm mờ riêng từng cây đang che gà (PLAN_LOGIC_GAME 10) mà vẫn ít lượt vẽ.
let treeSink = null;
const TREE_LEAVES = [[0, 4.1, 0, 1.65], [-1.15, 3.3, .1, 1.4], [1.15, 3.6, -.1, 1.45], [.15, 3.5, 1.1, 1.35], [-.4, 3.7, -1.1, 1.2]];
const PINE_CONES = [[2.2, 1.9, 2.6], [3.6, 1.5, 2.3], [4.9, 1.05, 2], [6, .6, 1.5]];
function tree(g, x, z, scale = 1, tint = 0x55743c) {
  const at = g.localToWorld ? treeLocal(g, x, z) : { x, z };
  if (treeSink) { treeSink.push({ kind: 'tree', x: at.x, z: at.z, s: scale, tint }); return null; }
  const p = new T.Group(); p.position.set(x, 0, z); p.scale.setScalar(scale); g.add(p);
  cylinder(p, .22, .45, 2.7, timber(C.edge), 0, 1.35, 0, 10);
  for (const [xx, yy, zz, r] of TREE_LEAVES) { const leaf = add(p, geometry('IcosahedronGeometry', r, 1), tint, xx, yy, zz); leaf.scale.y = .9; }
  return p;
}
function bush(g, x, z, s = 1, tint = 0x5b7a3e) {
  shadow(g, x, z, 1.3 * s);
  for (const [dx, dz, r] of [[0, 0, .75], [.55, .2, .55], [-.5, .15, .6], [.1, -.45, .5]]) { const b = add(g, geometry('IcosahedronGeometry', r, 1), tint, x + dx * s, r * .75 * s, z + dz * s); b.scale.setScalar(s); }
}
function pine(g, x, z, scale = 1) {
  const at = treeLocal(g, x, z);
  if (treeSink) { treeSink.push({ kind: 'pine', x: at.x, z: at.z, s: scale, tint: 0x3d5a32 }); return null; }
  const p = new T.Group(); p.position.set(x, 0, z); p.scale.setScalar(scale); g.add(p);
  cylinder(p, .16, .26, 2, timber(C.edge), 0, 1, 0, 8);
  for (const [y, r, h] of PINE_CONES) add(p, geometry('ConeGeometry', r, h, 9), 0x3d5a32, 0, y, 0);
  return p;
}
// Toạ độ cây trong khung của khu (cây có thể được thêm qua nhóm con như lán, quầy).
let sinkRoot = null;
function treeLocal(g, x, z) {
  if (!sinkRoot || g === sinkRoot) return { x, z };
  g.updateMatrixWorld(true); sinkRoot.updateMatrixWorld(true);
  const v = g.localToWorld(new T.Vector3(x, 0, z)); sinkRoot.worldToLocal(v);
  return { x: v.x, z: v.z };
}

// instanceTrees dựng mọi cây đã ghi thành InstancedMesh theo ô (chunk mét), trả về sổ cây để làm mờ.
function instanceTrees(g, list, chunk) {
  const out = { trees: [], grid: new Map(), ghosts: [] };
  if (!list.length) return out;
  const unitIco = geometry('IcosahedronGeometry', 1, 1), unitCone = geometry('ConeGeometry', 1, 1, 9);
  const trunkT = geometry('CylinderGeometry', .22, .45, 2.7, 10), trunkP = geometry('CylinderGeometry', .16, .26, 2, 8);
  const blob = geometry('PlaneGeometry', 2, 2);
  const leafMat = new T.MeshStandardMaterial({ color: 0xffffff, roughness: .95 });
  const shadowMat = shadowMaterial();
  const chunks = new Map();
  const keyOf = (x, z) => chunk === Infinity ? '0' : Math.floor(x / chunk) + ',' + Math.floor(z / chunk);
  for (const t of list) { const k = keyOf(t.x, t.z); if (!chunks.has(k)) chunks.set(k, []); chunks.get(k).push(t); }
  const m4 = new T.Matrix4(), q = new T.Quaternion(), e = new T.Euler(), v = new T.Vector3(), sc = new T.Vector3(), col = new T.Color();
  for (const group of chunks.values()) {
    const trees = group.filter((t) => t.kind === 'tree'), pines = group.filter((t) => t.kind === 'pine');
    const parts = {
      trunkT: trees.length && new T.InstancedMesh(trunkT, timber(C.edge), trees.length),
      leaves: trees.length && new T.InstancedMesh(unitIco, leafMat, trees.length * TREE_LEAVES.length),
      trunkP: pines.length && new T.InstancedMesh(trunkP, timber(C.edge), pines.length),
      cones: pines.length && new T.InstancedMesh(unitCone, material(0x3d5a32), pines.length * PINE_CONES.length),
      shade: new T.InstancedMesh(blob, shadowMat, group.length),
    };
    let ti = 0, li = 0, pi = 0, ci = 0, si = 0;
    const box = new T.Box3();
    for (const t of group) {
      const rec = { x: t.x, z: t.z, s: t.s, kind: t.kind, inst: [], hidden: false };
      const put = (mesh, i, px, py, pz, sx, sy, sz, rx = 0) => {
        e.set(rx, 0, 0); q.setFromEuler(e); v.set(t.x + px * t.s, py * t.s, t.z + pz * t.s); sc.set(sx * t.s, sy * t.s, sz * t.s);
        m4.compose(v, q, sc); mesh.setMatrixAt(i, m4); rec.inst.push([mesh, i, m4.clone()]);
        box.expandByPoint(v.clone().add(new T.Vector3(3 * t.s, 4 * t.s, 3 * t.s))); box.expandByPoint(v.clone().sub(new T.Vector3(3 * t.s, 0, 3 * t.s)));
      };
      put(parts.shade, si++, 0, .018, 0, t.kind === 'pine' ? 2.2 : 3.2, t.kind === 'pine' ? 2.2 : 3.2, 1, -Math.PI / 2);
      if (t.kind === 'tree') {
        put(parts.trunkT, ti++, 0, 1.35, 0, 1, 1, 1);
        for (const [xx, yy, zz, r] of TREE_LEAVES) { parts.leaves.setColorAt(li, col.setHex(t.tint)); put(parts.leaves, li++, xx, yy, zz, r, r * .9, r); }
      } else {
        put(parts.trunkP, pi++, 0, 1, 0, 1, 1, 1);
        for (const [y, r, h] of PINE_CONES) put(parts.cones, ci++, 0, y, 0, r, h, r);
      }
      const idx = out.trees.length;
      out.trees.push(rec);
      const cell = Math.floor(t.x / 10) + ',' + Math.floor(t.z / 10);
      if (!out.grid.has(cell)) out.grid.set(cell, []);
      out.grid.get(cell).push(idx);
    }
    for (const mesh of Object.values(parts)) {
      if (!mesh) continue;
      mesh.instanceMatrix.needsUpdate = true;
      if (mesh.instanceColor) mesh.instanceColor.needsUpdate = true;
      mesh.geometry = mesh.geometry.clone();
      mesh.geometry.boundingSphere = box.getBoundingSphere(new T.Sphere());
      mesh.castShadow = mesh !== parts.shade; mesh.receiveShadow = true;
      if (mesh === parts.shade) mesh.renderOrder = 1;
      g.add(mesh);
    }
  }
  return out;
}

// Cây mẫu bán trong suốt thay chỗ cây đang che gà.
let ghostTpl = null;
function ghostTree(kind) {
  if (!ghostTpl) {
    ghostTpl = {};
    const gm = new T.MeshStandardMaterial({ color: 0x6f8f4f, transparent: true, opacity: .22, depthWrite: false, roughness: 1 });
    const tm = new T.MeshStandardMaterial({ color: 0x6b4a2c, transparent: true, opacity: .3, depthWrite: false, roughness: 1 });
    const t = new T.Group();
    const trunk = new T.Mesh(geometry('CylinderGeometry', .22, .45, 2.7, 10), tm); trunk.position.y = 1.35; t.add(trunk);
    for (const [xx, yy, zz, r] of TREE_LEAVES) { const l = new T.Mesh(geometry('IcosahedronGeometry', r, 1), gm); l.position.set(xx, yy, zz); l.scale.y = .9; t.add(l); }
    const p = new T.Group();
    const pt = new T.Mesh(geometry('CylinderGeometry', .16, .26, 2, 8), tm); pt.position.y = 1; p.add(pt);
    for (const [y, r, h] of PINE_CONES) { const c = new T.Mesh(geometry('ConeGeometry', r, h, 9), gm); c.position.y = y; p.add(c); }
    ghostTpl.tree = t; ghostTpl.pine = p;
  }
  return ghostTpl[kind].clone();
}

const _zero = new T.Matrix4().makeScale(0, 0, 0), _a = new T.Vector3(), _b = new T.Vector3();
// fadeTrees: cây nằm giữa camera và gà (theo mặt bằng, dưới tầm cao của tán) thì ẩn bản đặc và hiện bản mờ.
export function fadeTrees(info, group, cam, target) {
  const reg = info?.treeReg;
  if (!reg || !reg.trees.length) return;
  _a.copy(cam); group.worldToLocal(_a); _b.set(target.x, 1, target.z); group.worldToLocal(_b);
  const dx = _b.x - _a.x, dz = _b.z - _a.z, len2 = dx * dx + dz * dz || 1;
  const want = new Set();
  const x0 = Math.floor(Math.min(_a.x, _b.x) / 10) - 1, x1 = Math.floor(Math.max(_a.x, _b.x) / 10) + 1;
  const z0 = Math.floor(Math.min(_a.z, _b.z) / 10) - 1, z1 = Math.floor(Math.max(_a.z, _b.z) / 10) + 1;
  if ((x1 - x0) * (z1 - z0) < 400) {
    for (let cx = x0; cx <= x1; cx++) for (let cz = z0; cz <= z1; cz++) {
      for (const i of reg.grid.get(cx + ',' + cz) || []) {
        const t = reg.trees[i];
        const u = ((t.x - _a.x) * dx + (t.z - _a.z) * dz) / len2;
        if (u < .04 || u > .97) continue;
        const d = Math.hypot(_a.x + dx * u - t.x, _a.z + dz * u - t.z), r = (t.kind === 'pine' ? 1.9 : 2.1) * t.s + .35;
        const y = _a.y + (_b.y - _a.y) * u, top = (t.kind === 'pine' ? 6.8 : 5.6) * t.s;
        if (d < r && y < top) want.add(i);
      }
    }
  }
  for (const i of [...reg.faded.keys()]) if (!want.has(i)) {
    const t = reg.trees[i];
    for (const [mesh, k, m] of t.inst) { mesh.setMatrixAt(k, m); mesh.instanceMatrix.needsUpdate = true; }
    const gh = reg.faded.get(i); gh.parent?.remove(gh); reg.faded.delete(i);
  }
  for (const i of want) {
    if (reg.faded.has(i) || reg.faded.size >= 16) continue;
    const t = reg.trees[i];
    for (const [mesh, k] of t.inst) { mesh.setMatrixAt(k, _zero); mesh.instanceMatrix.needsUpdate = true; }
    const gh = ghostTree(t.kind); gh.position.set(t.x, 0, t.z); gh.scale.setScalar(t.s); group.add(gh);
    reg.faded.set(i, gh);
  }
}
function rock(g, x, z, s = 1) { const r = add(g, geometry('DodecahedronGeometry', .5, 0), 0x9d9886, x, .25 * s, z); r.scale.set(s * 1.2, s * .7, s); r.rotation.y = x; }
function grass(g, hx, hz, count, seed, excluded = () => false, cx = 0, cz = 0) {
  const rng = random(seed);
  for (let i = 0; i < count; i++) {
    const x = cx + (rng() * 2 - 1) * hx, z = cz + (rng() * 2 - 1) * hz;
    if (excluded(x, z)) continue;
    for (let k = 0; k < 3; k++) {
      const blade = add(g, geometry('ConeGeometry', .045, .38, 3), [0x5d7a3c, 0x78904a, 0x4a6634][(i + k) % 3], x + (k - 1) * .08, .17, z + (k % 2) * .06, { shadow: false });
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
  const g = new T.Group(); g.rotation.y = ry; g.userData.fadeHide = true;
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
  const rng = random(seed), tints = [0x52713a, 0x5f7f40, 0x48662f, 0x6a8746];
  const open = (x, z) => gaps.some(([gx, gz, r]) => Math.hypot(x - gx, z - gz) < (r || 6));
  const per = (len) => Math.ceil(len / 4.2 * density);
  const ring = (fn, len) => { for (let i = 0; i <= per(len); i++) fn(i / per(len)); };
  const put = (x, z) => { if (open(x, z)) return; const roll = rng(); if (roll < .22) pine(g, x + (rng() - .5) * 2, z + (rng() - .5) * 2, .8 + rng() * .6); else if (roll < .6) tree(g, x + (rng() - .5) * 2, z + (rng() - .5) * 2, .75 + rng() * .6, tints[Math.floor(rng() * 4)]); else bush(g, x, z, .8 + rng() * .6, tints[Math.floor(rng() * 4)]); };
  for (const d of [2.5, 7.5]) {
    ring(t => { put(-hx - d + 2 * (hx + d) * t, -hz - d); put(-hx - d + 2 * (hx + d) * t, hz + d); }, 2 * hx);
    ring(t => { put(-hx - d, -hz - d + 2 * (hz + d) * t); put(hx + d, -hz - d + 2 * (hz + d) * t); }, 2 * hz);
  }
}

// ---------- Bo-2 prop kit: counters, signs, baskets, racks and tools ----------
// Shared unit geometries + cached materials, so batchStatic folds repeats into a few instanced draws.
// Plank-fronted counter facing +z (customers), with an overhanging top and a dark kick board.
function counter(g, x, z, w, d = 1.2, h = 1, ry = 0) {
  const c = new T.Group(); c.position.set(x, 0, z); c.rotation.y = ry; g.add(c);
  shadow(c, 0, 0, w * .6, d);
  box(c, w, h - .12, d, timber(C.wood), 0, (h - .12) / 2, 0);
  const n = Math.max(3, Math.round(w / .42));
  for (let i = 0; i < n; i++) box(c, w / n - .03, h - .2, .05, timber([0x8a5d39, 0x7a5131, 0x93653e][i % 3]), -w / 2 + (i + .5) * w / n, (h - .12) / 2, d / 2 + .02);
  box(c, w + .25, .1, d + .2, timber(C.light), 0, h - .06, 0);
  box(c, w + .05, .14, d + .04, timber(C.edge), 0, .07, 0);
  return c;
}
// Hanging signboard on two chains (shop names in the reference shots).
function hangingSign(g, text, x, y, z, w = 2.4, ry = 0) {
  const p = plaque(g, text, x, y, z, w, ry, { bg: '#6a4426', fg: '#f6e3b4', border: '#3f2814', font: '700 74px "Baloo 2", Georgia, serif' });
  for (const sx of [-1, 1]) { const ch = cylinder(p, .018, .018, 1.1, 0x3b3125, sx * w * .38, w / 8 + .6, -.08, 5); ch.castShadow = false; }
  return p;
}
function tray(g, x, y, z, w, d, fill, ry = 0) {
  const t = new T.Group(); t.position.set(x, y, z); t.rotation.y = ry; g.add(t);
  box(t, w, .12, d, timber(0x9a6a3e), 0, .06, 0);
  for (const sz of [-1, 1]) box(t, w, .16, .05, timber(C.edge), 0, .1, sz * d / 2);
  for (const sx of [-1, 1]) box(t, .05, .16, d, timber(C.edge), sx * w / 2, .1, 0);
  if (fill != null) box(t, w - .1, .03, d - .1, typeof fill === 'number' ? material(fill) : fill, 0, .13, 0);
  return t;
}
function wormPile(g, x, y, z, w, d, n, seed = 1) {
  const r = random(seed);
  for (let i = 0; i < n; i++) {
    const m = add(g, geometry('TorusGeometry', .07, .026, 5, 10, Math.PI * 1.3), [0xe08a8f, 0xd27479, 0xeb9fa2][i % 3], x + (r() - .5) * w, y, z + (r() - .5) * d, { shadow: false });
    m.rotation.set(Math.PI / 2 + (r() - .5) * .5, 0, r() * 6.28);
  }
}
// Woven basket bowl with its contents (grain, seeds, greens) as a textured disc.
function basket(g, x, y, z, r = .36, fill = 0xe2b347, h = .28) {
  if (!geo.has('basketLathe')) geo.set('basketLathe', new T.LatheGeometry([[.6, 0], [.86, .08], [1, .62], [1.04, 1], [.9, 1]].map(([a, b]) => new T.Vector2(a, b)), 18));
  const b = add(g, geo.get('basketLathe'), textured('weave', 0xd8b06a, [3, 1]), x, y, z); b.scale.set(r, h, r);
  const top = add(g, geometry('CylinderGeometry', 1, 1, .04, 18), textured('grain', fill, [1, 1]), x, y + h * .92, z, { shadow: false }); top.scale.set(r * .92, 1, r * .92);
  return b;
}
function tub(g, x, z, r = .8, h = .75) {
  cylinder(g, r, r * .92, h, timber(C.wood), x, h / 2, z, 18);
  for (const y of [.12, h - .12]) cylinder(g, r + .02, r + .02, .06, 0x4d4a3e, x, y, z, 18);
  cylinder(g, r * .92, r * .92, .02, material(0x5f9db2, { roughness: .12, metalness: .1 }), x, h - .06, z, 18);
}
function stool(g, x, z, h = .55) { cylinder(g, .3, .3, .08, timber(C.light), x, h, z, 10); for (let i = 0; i < 3; i++) { const a = i * 2.1; beam(g, [x + Math.cos(a) * .2, h, z + Math.sin(a) * .2], [x + Math.cos(a) * .3, 0, z + Math.sin(a) * .3], .05, C.edge); } }
// Rounded haystack (dome, loose straws at the skirt) used in foreground corners.
function hayMound(g, x, z, s = 1) {
  if (!geo.has('hayMound')) geo.set('hayMound', new T.LatheGeometry([[1.25, 0], [1.3, .3], [1.18, .8], [.86, 1.25], [.42, 1.5], [0, 1.58]].map(([a, b]) => new T.Vector2(a, b)), 20));
  shadow(g, x, z, 1.9 * s);
  const m = add(g, geo.get('hayMound'), textured('straw', 0xe8c470, [3, 2]), x, 0, z); m.scale.setScalar(s);
  const rng = random(Math.round((x + 200) * 13 + z * 7));
  for (let i = 0; i < 26; i++) { const a = rng() * 6.28, rr = 1.25 * s; beam(g, [x + Math.cos(a) * rr, .05, z + Math.sin(a) * rr], [x + Math.cos(a + .2) * (rr + .35 * s), .02, z + Math.sin(a + .2) * (rr + .35 * s)], .025, [0xe1c774, 0xc9a24f][i % 2]); }
}
function shovel(g, x, z, ry = 0, lean = .25) {
  const p = new T.Group(); p.position.set(x, 0, z); p.rotation.set(0, ry, lean); g.add(p);
  cylinder(p, .03, .03, 1.3, timber(C.light), 0, 1.0, 0, 6);
  box(p, .26, .32, .03, material(0x8d918c, { metalness: .5, roughness: .45 }), 0, .2, 0);
  box(p, .22, .05, .05, timber(C.edge), 0, 1.66, 0);
}
function bucket(g, x, z, fill) {
  cylinder(g, .25, .2, .4, 0x7e8380, x, .2, z, 12);
  if (fill != null) cylinder(g, .23, .23, .02, fill, x, .37, z, 12);
}
function stump(g, x, z, s = 1) {
  cylinder(g, .5 * s, .58 * s, .6 * s, timber(0x6d4a2c), x, .3 * s, z, 12);
  cylinder(g, .49 * s, .49 * s, .02, 0xc9a46c, x, .61 * s, z, 12);
}
// A shirt (body + sleeves) or a dress hung from a rail, facing +z of the rack group.
function garment(g, x, y, kind, color) {
  if (kind === 'dress') { const d = add(g, geometry('ConeGeometry', .32, .8, 10), color, x, y - .45, 0); d.scale.z = .35; box(g, .26, .2, .1, color, x, y - .1, 0); }
  else {
    box(g, .5, .62, .1, color, x, y - .36, 0);
    for (const sx of [-1, 1]) { const sl = box(g, .2, .32, .1, color, x + sx * .32, y - .2, 0); sl.rotation.z = sx * .5; }
    box(g, .16, .05, .11, 0xf2ead2, x, y - .08, .01);
  }
  cylinder(g, .01, .01, .12, 0x3b3125, x, y + .02, 0, 4);
}
function clothesRack(g, x, z, len, ry, items) {
  const r = new T.Group(); r.position.set(x, 0, z); r.rotation.y = ry; g.add(r);
  for (const sx of [-1, 1]) box(r, .1, 2.2, .1, timber(C.edge), sx * len / 2, 1.1, 0);
  cylinder(r, .03, .03, len, 0x8e6b45, 0, 2.05, 0, 6).rotation.z = Math.PI / 2;
  items.forEach(([kind, color], i) => garment(r, -len / 2 + (i + .5) * len / items.length, 2.0, kind, color));
  return r;
}
function strawHat(g, x, y, z, s = 1, band = 0xb7473a) {
  const brim = cylinder(g, .34 * s, .34 * s, .03, textured('weave', 0xe2c07a, [2, 1]), x, y, z, 16);
  const crown = add(g, geometry('SphereGeometry', .18, 12, 8, 0, Math.PI * 2, 0, Math.PI / 2), textured('weave', 0xe2c07a, [2, 1]), x, y, z); crown.scale.setScalar(s);
  cylinder(g, .185 * s, .185 * s, .05 * s, band, x, y + .03 * s, z, 12);
  return brim;
}
function capHat(g, x, y, z, color) {
  const c = add(g, geometry('SphereGeometry', .17, 12, 8, 0, Math.PI * 2, 0, Math.PI / 2), color, x, y, z);
  const bill = box(g, .2, .02, .16, color, x, y + .01, z + .2); return c;
}
function chalkboard(g, x, z, ry = 0, text = '〰') {
  const p = new T.Group(); p.position.set(x, 0, z); p.rotation.y = ry; g.add(p);
  for (const sx of [-1, 1]) beam(p, [sx * .42, 0, .18], [sx * .38, 1.2, -.02], .06, C.edge);
  const b = box(p, .86, .72, .05, timber(C.edge), 0, .78, 0); b.rotation.x = -.15;
  const face = add(p, geometry('PlaneGeometry', .74, .6), new T.MeshStandardMaterial({ map: textCanvas(text, { bg: '#2f3a33', fg: '#e8a0a6', border: '#2f3a33', font: '700 120px Georgia, serif', w: 256, h: 210 }), roughness: 1 }), 0, .78, .03, { shadow: false }); face.rotation.x = -.15;
  return p;
}
function brassScale(g, x, y, z) {
  const p = new T.Group(); p.position.set(x, y, z); g.add(p);
  const brass = material(0xb58e38, { metalness: .6, roughness: .35 });
  cylinder(p, .12, .16, .06, brass, 0, .03, 0, 12); cylinder(p, .03, .03, .5, brass, 0, .3, 0, 6);
  box(p, .7, .03, .04, brass, 0, .55, 0);
  for (const sx of [-1, 1]) { cylinder(p, .17, .08, .06, brass, sx * .32, .38, 0, 14); cylinder(p, .005, .005, .17, brass, sx * .32, .47, 0, 3); }
}
function table(g, x, z, w = 1.6, d = .9, h = .8, cloth) {
  box(g, w, .08, d, timber(C.light), x, h, z);
  for (const sx of [-1, 1]) for (const sz of [-1, 1]) box(g, .08, h, .08, timber(C.edge), x + sx * (w / 2 - .1), h / 2, z + sz * (d / 2 - .1));
  if (cloth != null) { box(g, w + .04, .02, d * .7, cloth, x, h + .05, z); box(g, w * .6, .3, .02, cloth, x, h - .1, z + d / 2 + .01); }
}
function bannerFlag(g, x, y, z, color = 0xe9dcc0, text = '🌾', ry = 0) {
  const p = new T.Group(); p.position.set(x, y, z); p.rotation.y = ry; g.add(p);
  cylinder(p, .03, .03, 1.1, timber(C.edge), 0, .55, 0, 6).rotation.z = Math.PI / 2;
  add(p, geometry('PlaneGeometry', .9, 1.3), new T.MeshStandardMaterial({ map: textCanvas(text, { bg: '#e8dcc0', fg: '#b18440', border: '#cdb88f', font: '700 150px Georgia, serif', w: 256, h: 370 }), side: T.DoubleSide, roughness: 1 }), 0, -.1, 0, { shadow: false });
  return p;
}

// ---------- Residents (ambient NPCs, never counted as players) ----------
// A dressed chicken is ~25-40 small meshes. Residents never change clothes, so every plain-colour mesh is
// merged into one vertex-coloured mesh per animated joint (rig, head, wings, legs): ~6 draws per NPC.
const bakedMaterial = new T.MeshStandardMaterial({ vertexColors: true, roughness: .9 });
function bakeJoints(model) {
  const p = model.userData.parts; if (!p) return;
  const joints = [p.rig, p.head, ...(p.wings || []), ...(p.legs || [])].filter(Boolean), set = new Set(joints);
  model.updateMatrixWorld(true);
  const groups = new Map(joints.map(j => [j, []]));
  model.traverse(o => {
    if (!o.isMesh || Array.isArray(o.material) || o.material.map || o.material.transparent || !o.material.color) return;
    let j = o.parent; while (j && !set.has(j)) j = j.parent;
    if (j) groups.get(j).push(o);
  });
  const inv = new T.Matrix4(), rel = new T.Matrix4(), nm = new T.Matrix3(), v = new T.Vector3(), c = new T.Color();
  for (const [joint, list] of groups) {
    if (list.length < 2) continue;
    inv.copy(joint.matrixWorld).invert();
    const pos = [], nor = [], col = [];
    for (const o of list) {
      const geo = o.geometry.index ? o.geometry.toNonIndexed() : o.geometry, P = geo.attributes.position, N = geo.attributes.normal;
      rel.multiplyMatrices(inv, o.matrixWorld); nm.getNormalMatrix(rel); c.copy(o.material.color);
      for (let i = 0; i < P.count; i++) {
        v.fromBufferAttribute(P, i).applyMatrix4(rel); pos.push(v.x, v.y, v.z);
        v.fromBufferAttribute(N, i).applyMatrix3(nm).normalize(); nor.push(v.x, v.y, v.z); col.push(c.r, c.g, c.b);
      }
      if (geo !== o.geometry) geo.dispose();
      o.parent.remove(o);
    }
    const geo = new T.BufferGeometry();
    geo.setAttribute('position', new T.Float32BufferAttribute(pos, 3)); geo.setAttribute('normal', new T.Float32BufferAttribute(nor, 3)); geo.setAttribute('color', new T.Float32BufferAttribute(col, 3));
    geo.computeBoundingSphere();
    const baked = new T.Mesh(geo, bakedMaterial); baked.castShadow = true; baked.receiveShadow = true; joint.add(baked);
  }
}

function resident(g, arr, cosmetics, name, x, z, action, color, shirt, yaw = 0, extra = {}) {
  const m = buildChicken({ color, eyes: action === 'sleep' ? 'ngai' : 'tron', comb: extra.comb || 'don' });
  dress(m, extra.equipped || {}, { shirt }, cosmetics || []);
  bakeJoints(m);
  m.position.set(x, extra.y || 0, z); m.rotation.y = yaw; m.userData.dynamic = true; m.userData.npcName = name; g.add(m);
  const a = { model: m, name, action, x, z, baseY: extra.y || 0, yaw, phase: arr.length * 1.73, route: extra.route }; arr.push(a); return a;
}

// ---------- Community hall (reference: "Sảnh nông trại gà tối giản") ----------
const HALL = { hx: 16, hz: 21, wall: 7, front: 2.4, door: 4.4 };
// Vách lưới mắt cáo: cột gỗ + thanh ngang + giằng chéo, nhìn xuyên ra cây phía sau.
function latticeWall(g, x1, z1, x2, z2, h = 6.8) {
  const len = Math.hypot(x2 - x1, z2 - z1), w = new T.Group();
  w.position.set((x1+x2)/2,0,(z1+z2)/2);w.rotation.y=-Math.atan2(z2-z1,x2-x1);g.add(w);
  const n=Math.max(1,Math.round(len/5)),panels=[];
  for(const y of [.22,2.25,h-1.3,h-.15])box(w,len,.18,.23,timber(C.wood),0,y,0);
  for(let i=0;i<=n;i++){
    const x=-len/2+i*len/n;box(w,.38,h,.4,timber(C.edge),x,h/2,0);
    if(i<n){
      panels.push([x+len/n/2,1.2,len/n-.25,2.1]);
      // Thin upper wire panels leave open windows to the surrounding countryside.
      if(i%2===0&&h>4.6)panels.push([x+len/n/2,h-2.45,len/n-.25,2.1]);
      beam(w,[x+.1,h-1.4,0],[x+1.1,h-.2,0],.13,C.wood);
    }
    if(i%2===0 && h>3)lantern(w,x+.4,3.3,.35);
  }
  wirePanels(w,panels);
  return w;
}
// All chicken-wire panels of a wall as one mesh; UVs in metres so the shared texture keeps ~15 cm cells.
function wirePanels(w,panels){
  if(!panels.length)return;
  const pos=[],uv=[],idx=[];
  for(const [cx,cy,pw,ph] of panels){
    const b=pos.length/3;
    for(const [dx,dy] of [[-.5,-.5],[.5,-.5],[.5,.5],[-.5,.5]]){const x=cx+dx*pw,y=cy+dy*ph;pos.push(x,y,0);uv.push(x*.8,y*.8);}
    idx.push(b,b+1,b+2,b,b+2,b+3);
  }
  const geo=new T.BufferGeometry();geo.setAttribute('position',new T.Float32BufferAttribute(pos,3));geo.setAttribute('uv',new T.Float32BufferAttribute(uv,2));geo.setIndex(idx);geo.computeVertexNormals();geo.computeBoundingSphere();
  const m=new T.Mesh(geo,farmMaterial('mesh'));m.castShadow=false;w.add(m);
}
function strawRoof(g,hx,hz,eave=7,ridge=13){
  const r=createThatchRoof(hx,hz,eave,ridge);g.add(r);return r;
}
function pavilion(g,info,hx,hz,eave=5.2,ridge=8.8){
  for(const ends of [[-hx,-hz,hx,-hz],[-hx,-hz,-hx,hz],[hx,-hz,hx,hz]]){
    const wall=latticeWall(g,...ends,eave);occluder(info,wall);
  }
  const r=strawRoof(g,hx+.45,hz+.45,eave,ridge);
  // Each slope is tested separately: being under a roof must not fade the entire building.
  for(const piece of [...r.children])if(piece.userData.roof)occluder(info,piece);
  return r;
}
// Boardwalk of weathered planks laid lengthwise in staggered columns, with dark gaps and slight
// height/angle jitter like the reference walks (not square tiles).
function woodWalk(g,points,width=3.2){
 const rng=random(312+Math.round(width*10)),cols=Math.max(3,Math.round(width/.56)),cw=width/cols,tones=[0x9b7d5a,0x8a6c4b,0xa88a64,0x7d6246,0x947552];
 for(let j=1;j<points.length;j++){
  const [ax,az]=points[j-1],[bx,bz]=points[j],len=Math.hypot(bx-ax,bz-az),a=Math.atan2(bx-ax,bz-az),ux=(bx-ax)/len,uz=(bz-az)/len;
  box(g,width+.1,.05,len,timber(0x3d2a1a),ax+(bx-ax)/2,.04,az+(bz-az)/2).rotation.y=a;
  for(let c=0;c<cols;c++){
   const off=(c-(cols-1)/2)*cw;let t=-rng()*1.2;
   while(t<len){
    const l=1.3+rng()*1.3,s0=Math.max(0,t),s1=Math.min(len,t+l);t+=l;if(s1-s0<.3)continue;
    const mid=(s0+s1)/2,x=ax+ux*mid+Math.cos(a)*off,z=az+uz*mid-Math.sin(a)*off;
    const pl=box(g,cw-.06,.07,s1-s0-.05,timber(tones[Math.floor(rng()*tones.length)]),x,.1+rng()*.025,z);
    pl.rotation.set((rng()-.5)*.02,a+(rng()-.5)*.03,(rng()-.5)*.03);
   }
  }
 }
}

// Lối ván gỗ từ cửa tới bục: các tấm lệch nhẹ, có khe và vân.
function plankAisle(g,hz){woodWalk(g,[[0,hz-1],[0,-hz+6]],3.2);}
function hall(g, data, cosmetics, info) {
  const { hx, hz } = HALL;
  ground(g, hx * 2, hz * 2, 0xc4a379, 'earth', .01);
  // Lối ván gỗ chạy từ cửa vào tới bục (mục 4.2: trục lối ván chính giữa).
  plankAisle(g, hz);
  pavilion(g,info,hx,hz,7,13);
  // The front opening frames the hall; its centre is clear for the entrance camera.
  for(const sx of [-1,1]){
    const wall=latticeWall(g,sx*HALL.door,hz,sx*hx,hz,2.4);occluder(info,wall);
    box(g,.5,7,.5,timber(C.edge),sx*hx,3.5,hz);
  }
  dressGround(g,hx-.5,hz-.5,2026,(x,z)=>Math.abs(x)<1.85||z<-14||data.items.some(i=>i.kind==='nest'&&Math.hypot(x-i.x,z-i.z)<1.12));
  // Owner's stage: deck top is exactly y=1.2 to match the host transform from the server.
  const sz = -17;
  box(g, 11, 1.05, 4, timber(C.edge), 0, .525, sz);
  for (let i = 0; i < 8; i++) box(g, 11.2, .15, .48, timber(i % 2 ? 0xb88f60 : 0xa98152), 0, 1.125, sz - 1.75 + i * .5);
  for (let i = 0; i < 12; i++) box(g, .5, .9, .08, timber(i % 2 ? 0x7a5232 : 0x6c482b), -5.25 + i * .95, .5, sz + 2.02);
  for (const sx of [-1, 1]) for (let i = 0; i < 3; i++) box(g, 1.6, .3 * (3 - i), .55, timber(i % 2 ? 0xa98152 : C.wood), sx * 6.3, .15 * (3 - i), sz + .9 + i * .02 - i * .55 + 1.2);
  nest(g, 0, -17.3, 1.75, 1.2);
  for (const sx of [-1, 1]) { flowerpot(g, sx * 4.6, -18.3, 1.3, 1.2); cylinder(g, .09, .12, 2.6, timber(C.edge), sx * 5.2, 2.5, -15.4, 8); lantern(g, sx * 5.2, 3.6, -15.4); }
  hangingSign(g,'HYPEX FARM',0,4.15,-hz+5.7,4.4);
  for(const x of [-2.9,2.9]){cylinder(g,.025,.025,.9,0x5b4529,x,5,-hz+5.7,6);lantern(g,x,4.3,-hz+5.7);}
  // Lower thatch canopy over the owner's stage, on its own posts (B02-02 back of the hall).
  const backCanopy=strawRoof(g,8.6,2.6,5.4,7.2);backCanopy.position.z=-hz+3.1;
  for(const sx of [-1,1])for(const zz of [-hz+.8,-hz+5.4])box(g,.32,5.5,.32,timber(C.edge),sx*8.2,2.75,zz);
  for(const sx of [-1,1]){
    for(const z of [-12,-2,8]){box(g,.65,.12,3.2,timber(C.light),sx*15.2,.7,z);for(const dz of [-1.2,1.2])box(g,.5,.65,.2,timber(C.edge),sx*15.2,.32,z+dz);}
  }
  // 40 player nests in two blocks of four per row, with a broad central aisle.
  for (const it of data.items) if (it.kind === 'nest') nest(g, it.x, it.z);
  // Máng thóc phía trước trái, máng nước phía trước phải (mục 4.2), rơm và bao ở các góc.
  for(const sx of [-1,1]){const tg=new T.Group();tg.position.set(sx*11.8,0,12.8);tg.rotation.y=Math.PI/2;g.add(tg);trough(tg,0,0,sx>0,5.6);}
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
  // Góc đối chiếu B02-02: camera trong sảnh nhìn dọc lối ván về bục, thấy mặt dưới mái rơm.
  info.camera = { focus: [0, 1, -6.5], pitch: .25, radius: 24.5, yaw: 0, fov: 62, followPitch: .45, followRadius: 11 };
  info.npcCount = info.ambient.length + 1;
}

// ---------- Open countryside: the seamless map between the areas ----------
// Roads run from the plaza (hub) to each area's entry; trees, fields and lamps fill the land between.
function countryside(g, data, cosmetics, info) {
  const W = data.halfX, D = data.halfZ, hub = data.hub || { x: 0, z: 0 }, areas = data.areas || [];
  ground(g, W * 2 + 120, D * 2 + 120, C.grass, 'grass', -.02);
  const ringR = Math.min(10.5, ...areas.filter((a) => a.entry).map((a) => Math.hypot(a.entry.x - hub.x, a.entry.z - hub.z) * .6)), ring = (e) => { const l = Math.hypot(e.x - hub.x, e.z - hub.z) || 1; return { x: hub.x + (e.x - hub.x) / l * ringR, z: hub.z + (e.z - hub.z) / l * ringR }; };
  const roads = areas.filter((a) => a.entry).map((a) => {
    const r = ring(a.entry), into = { x: a.entry.x + (a.cx - a.entry.x) * .25, z: a.entry.z + (a.cz - a.entry.z) * .25 };
    path(g, [[r.x, r.z], [(r.x + a.entry.x) / 2, (r.z + a.entry.z) / 2], [a.entry.x, a.entry.z], [into.x, into.z]], a.id === 'station' ? 6 : 4.8);
    return { a, r };
  });
  const nearRoad = (x, z, m) => roads.some(({ a, r }) => segDist(x, z, r.x, r.z, a.entry.x, a.entry.z) < m);
  const inArea = (x, z, m) => areas.some((a) => Math.abs(x - a.cx) < a.halfX + m && Math.abs(z - a.cz) < a.halfZ + m);
  const belt = data.belt && areas.some((a) => a.entry && Math.hypot(a.entry.x - hub.x, a.entry.z - hub.z) > data.belt + 4) ? data.belt : 0;
  const onBelt = (x, z, m) => belt && Math.abs(Math.hypot(x - hub.x, z - hub.z) - belt) < 2.6 + m;
  const free = (x, z, m = 6) => Math.hypot(x - hub.x, z - hub.z) > 18 && !nearRoad(x, z, m) && !inArea(x, z, m + 3) && !onBelt(x, z, m * .6);
  // Đường vành đai (PLAN_LOGIC_GAME 7.1): xe buýt vòng làng chạy trên đây rồi rẽ vào bến ở cổng từng khu.
  if (belt) {
    const ringRoad = add(g, new T.RingGeometry(belt - 2.4, belt + 2.4, 160, 1), textured('earth', C.path, [10, 10], { roughness: 1, polygonOffset: true, polygonOffsetFactor: -1 }), hub.x, .032, hub.z, { shadow: false });
    ringRoad.rotation.x = -Math.PI / 2;
    for (let a = 0; a < Math.PI * 2; a += .21) for (const side of [-1, 1]) rock(g, hub.x + Math.cos(a) * (belt + side * 2.75), hub.z + Math.sin(a) * (belt + side * 2.75), .26);
  }
  for (const st of data.roads?.stops || []) busStopShelter(g, st, hub);
  xeomStand(g, hub);
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
    if (n % 5 === 1) pine(g, x, z, .8 + rng() * .7); else if (n % 3) tree(g, x, z, .75 + rng() * .8, [0x52713a, 0x5f7f40, 0x48662f, 0x6a8746][n % 4]); else bush(g, x, z, 1 + rng() * .6);
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
    const hx = hub.x + Math.cos(a) * (W + 48), hz = hub.z + Math.sin(a) * (D + 48), s = 11 + (a * 37 % 7);
    const hill = add(g, geometry('SphereGeometry', 1, 16, 10), [0x5f7a40, 0x6a8547, 0x56703a][Math.floor(a * 10) % 3], hx, -s * .45, hz, { shadow: false }); hill.scale.set(s * 1.6, s, s * 1.6);
  }
  grass(g, W, D, Math.round(W * D / 18), 83, (x, z) => !free(x, z, 3.4), hub.x, hub.z);
  flowers(g, hub.x - 14, hub.z - 12, 4, 40, 3); flowers(g, hub.x + 16, hub.z + 12, 4, 40, 4);
  resident(g, info.ambient, cosmetics, 'Gà Dạo Làng', hub.x - 8, hub.z + 9, 'walk', 'kem', 'luc', .6, { route: [[hub.x - 8, hub.z + 9], [hub.x - 14, hub.z + 18]] });
  resident(g, info.ambient, cosmetics, 'Gà Chăm Cây', hub.x + 7, hub.z - 10, 'greet', 'vang', 'do', 2);
  resident(g, info.ambient, cosmetics, 'Gà Ngồi Ghế', hub.x + Math.cos(1.7) * 6.5, hub.z + Math.sin(1.7) * 6.5 + .1, 'chat', 'nau', 'xanh', Math.PI + .2, { y: .35 });
  info.camera = { pitch: .62, radius: 17 };
}
// Mái chờ xe buýt cạnh đường, ngay trước cổng khu: ghế, mái rơm nhỏ, biển 🚌 tên bến.
function busStopShelter(g, st, hub) {
  const dx = st.pos.x - hub.x, dz = st.pos.z - hub.z, l = Math.hypot(dx, dz) || 1;
  const sx = dz / l, sz = -dx / l, s = new T.Group();
  s.position.set(st.pos.x + sx * 3.3, 0, st.pos.z + sz * 3.3); s.rotation.y = Math.atan2(-sx, -sz); g.add(s);
  for (const x of [-1.2, 1.2]) for (const z of [-.5, .5]) box(s, .14, 2.5, .14, timber(C.edge), x, 1.25, z);
  const roof = createThatchRoof(.9, 1.5, 2.45, 3.1); roof.rotation.y = Math.PI / 2; s.add(roof);
  box(s, 2.2, .1, .5, timber(C.light), 0, .55, -.15); for (const x of [-.9, .9]) box(s, .1, .55, .4, timber(C.edge), x, .27, -.15);
  const post = new T.Group(); post.position.set(1.7, 0, .6); s.add(post); box(post, .1, 2.3, .1, timber(C.edge), 0, 1.15, 0);
  plaque(post, '🚌 ' + st.title, 0, 2.15, .07, 1.6, 0, { bg: '#2f6f9e', fg: '#fff6e4', border: '#1d4a6b', font: '700 54px "Baloo 2", Georgia, serif' });
}
// Bến xe ôm cây đa cạnh quảng trường: ô che, ghế đẩu, biển.
function xeomStand(g, hub) {
  const x = hub.x - 6.5, z = hub.z + 9.5, s = new T.Group(); s.position.set(x - 1.6, 0, z + .8); g.add(s);
  cylinder(s, .05, .05, 2.4, 0x6b4a2c, 0, 1.2, 0, 6);
  const top = add(s, geometry('ConeGeometry', 1.5, .6, 10), 0xc8642c, 0, 2.5, 0); top.castShadow = true;
  stool(s, .6, .4); stool(s, -.5, .5);
  const sign = new T.Group(); sign.position.set(1.4, 0, -.2); s.add(sign); box(sign, .1, 1.7, .1, timber(C.edge), 0, .85, 0);
  plaque(sign, '🛵 BẾN XE ÔM', 0, 1.6, .07, 1.6, 0, { bg: '#c8642c', fg: '#fff6e4', border: '#7a3a14', font: '700 56px "Baloo 2", Georgia, serif' });
}
function segDist(x, z, ax, az, bx, bz) {
  const dx = bx - ax, dz = bz - az, t = Math.max(0, Math.min(1, ((x - ax) * dx + (z - az) * dz) / (dx * dx + dz * dz || 1)));
  return Math.hypot(x - ax - dx * t, z - az - dz * t);
}

// Thatched coop with plank walls, a mesh front and a door towards +z; ridge runs along x.
function coopShed(g, x, z, w, d, ry = 0, flag) {
  const c = new T.Group(); c.position.set(x, 0, z); c.rotation.y = ry; g.add(c);
  const h = 2.4;
  box(c, w, .16, d, timber(C.light), 0, .08, 0);
  for (let i = 0; i < Math.round(w / .5); i++) box(c, w / Math.round(w / .5) - .04, h, .1, timber([0x7a5131, 0x8a5d39, 0x6e4a2c][i % 3]), -w / 2 + (i + .5) * w / Math.round(w / .5), h / 2, -d / 2);
  for (const sx of [-1, 1]) box(c, .1, h, d, timber(C.wood), sx * w / 2, h / 2, 0);
  latticeWall(c, -w / 2, d / 2, -.7, d / 2, h); latticeWall(c, .7, d / 2, w / 2, d / 2, h);
  for (const sx of [-1, 1]) box(c, .2, h + .2, .2, timber(C.edge), sx * .7, (h + .2) / 2, d / 2);
  const roof = createThatchRoof(d / 2 + .7, w / 2 + .6, h + .1, h + 1.8); roof.rotation.y = Math.PI / 2; c.add(roof);
  hayMound(c, -w / 4, -d / 6, .55); nest(c, w / 4, -d / 6, .9);
  lantern(c, w / 2 - .3, 2, d / 2 + .3);
  if (flag != null) { box(c, .07, 1.6, .07, timber(C.edge), -w / 2 + .3, h + .6, d / 2); bannerFlag(c, -w / 2 + .75, h + 1.05, d / 2 + .02, 0xd8c7a0, flag); }
  return c;
}
// Construction frame: posts, beams and rafters without walls (Bò Mộc's job site).
function houseFrame(g, x, z, w, d, h = 2.6) {
  const f = new T.Group(); f.position.set(x, 0, z); g.add(f);
  box(f, w + .4, .2, d + .4, timber(C.light), 0, .1, 0);
  for (const sx of [-1, 1]) for (const sz of [-1, 1]) box(f, .22, h, .22, timber(C.wood), sx * w / 2, h / 2, sz * d / 2);
  for (const sz of [-1, 1]) box(f, w + .2, .18, .18, timber(C.wood), 0, h, sz * d / 2);
  for (const sx of [-1, 1]) box(f, .18, .18, d + .2, timber(C.wood), sx * w / 2, h, 0);
  for (let i = 0; i <= 3; i++) { const zz = -d / 2 + i * d / 3; beam(f, [-w / 2 - .2, h, zz], [0, h + 1.5, zz], .14, C.light); beam(f, [w / 2 + .2, h, zz], [0, h + 1.5, zz], .14, C.light); }
  box(f, .16, .16, d + .4, timber(C.light), 0, h + 1.5, 0);
  for (let i = 0; i < 5; i++) box(f, 2.4, .16, .3, timber(C.light), -w / 2 - 1.4, .1 + i * .17, d / 2 - .3 - (i % 2) * .35);
  return f;
}
function vegBed(g, x, z, w, d, seed = 1) {
  box(g, w, .3, d, timber(C.edge), x, .15, z); box(g, w - .2, .05, d - .2, material(0x4a3420), x, .3, z);
  const r = random(seed);
  for (let i = 0; i < Math.floor(w / .6); i++) for (let j = 0; j < Math.max(1, Math.floor(d / .7)); j++) {
    const px = x - w / 2 + .4 + i * .6, pz = z - d / 2 + .4 + j * .7, k = Math.floor(r() * 4);
    if (k === 0) { const c = sphere(g, .2, 0x7fa64e, px, .42, pz); c.scale.y = .7; }
    else if (k === 1) { add(g, geometry('ConeGeometry', .07, .3, 6), 0xe0802e, px, .38, pz).rotation.x = Math.PI; bush(g, px, pz, .16, 0x5f8f3a); }
    else if (k === 2) { bush(g, px, pz, .22, 0x5d8a3c); sphere(g, .07, 0xc8432f, px + .1, .45, pz); }
    else bush(g, px, pz, .2, 0x6f9a48);
  }
}
// B02-07 Khu nhà bạn bè: gate and big signboard in front, a dirt road with stepping planks, thatched
// coops with fenced yards and vegetable patches on both sides, Bò Mộc at a house frame on the right.
function neighborhood(g, data, cosmetics, info) {
  ground(g, 48, 48, C.grass, 'grass', .004);
  path(g, [[0, 24], [0, 10], [0, -10], [0, -22]], 5.4, C.path);
  for (let z = 19; z > -18; z -= 2.6) { const p = box(g, 1.2, .07, .55, timber(0x8a6c4b), (z * 7 % 3 - 1) * .25, .06, z); p.rotation.y = (z % 2) * .15; }
  for (const sx of [-1, 1]) for (const z of [-12, 4]) {
    coopShed(g, sx * 13, z, 7, 4.6, -sx * Math.PI / 2, ['🐓', '🐤', '🌾', '🥚'][(sx + 1) + (z > 0 ? 1 : 0)]);
    fence(g, [[sx * 4.2, z - 4], [sx * 10, z - 4]], 1.1); fence(g, [[sx * 4.2, z + 4.2], [sx * 10, z + 4.2]], 1.1); fence(g, [[sx * 4.2, z - 4], [sx * 4.2, z - 1]], 1.1);
    vegBed(g, sx * 7.4, z + 2.4, 3.6, 1.6, 40 + z + sx); trough(g, sx * 8.6, z - 2, false, 2);
    hay(g, sx * 9.4, z - 3.2, .5); lampPost(g, sx * 3.6, z + 4.6, 3);
    resident(g, info.ambient, cosmetics, 'Gà Hàng Xóm', sx * 6.2, z - .4, sx > 0 ? 'chat' : 'sweep', sx > 0 ? 'trang' : 'vang', sx > 0 ? 'xanh' : 'do', -sx * Math.PI / 2);
  }
  // Guild hall front behind its notice board; flags of the village guild.
  const hallG = new T.Group(); hallG.position.set(12, 0, -16.6); g.add(hallG);
  coopShed(hallG, 0, 0, 7.4, 4, 0, '🏰');
  // Bò Mộc's site (server obstacle 14.4,13).
  houseFrame(g, 14.4, 13, 3.8, 3.8);
  // Gate with the big neighbourhood sign (server home_sign at -4.4,16.6).
  for (const sx of [-1, 1]) { box(g, .5, 3.6, .5, timber(C.edge), sx * 3.2, 1.8, 20.6); lantern(g, sx * 3.2, 2.9, 21); }
  fence(g, [[-3.2, 20.6], [-14, 20.6]], 1.3); fence(g, [[3.2, 20.6], [14, 20.6]], 1.3);
  const bigSign = new T.Group(); bigSign.position.set(-3.4, 0, 16.4); bigSign.rotation.y = .3; g.add(bigSign);
  for (const sx of [-1, 1]) box(bigSign, .2, 1.8, .2, timber(C.edge), sx * 1.2, .9, 0);
  plaque(bigSign, 'KHU NHÀ BẠN BÈ', 0, 1.35, .12, 2.9, 0, { bg: '#6a4426', fg: '#f6e3b4', border: '#3f2814', font: '700 64px "Baloo 2", Georgia, serif' });
  // Village roofs in the distance.
  for (const [x, z] of [[-8, -30], [4, -32], [16, -29], [-18, -27]]) shed(g, x, z, 4.4, 3, 2.2);
  surround(g, 23, 23, 22, [[0, 25, 7]], .7);
  grass(g, 22, 22, 220, 22, (x, z) => Math.abs(x) < 3.5 || Math.abs(x) > 4 && Math.abs(x) < 16 && (Math.abs(z + 12) < 4.5 || Math.abs(z - 4) < 4.5));
  resident(g, info.ambient, cosmetics, 'Gà Đi Dạo', 2.2, 8.6, 'walk', 'trang', 'luc', 0, { route: [[2.2, 8.6], [1.4, -6]] });
  resident(g, info.ambient, cosmetics, 'Gà Chào Bạn', 1.6, 15.4, 'greet', 'trang', 'luc', 2.8);
  // Văn phòng đất nền của Ông Chủ Đầu Tư (server: chu_dau_tu 6.4,15.4; quầy 6.4,13.6).
  counter(g, 6.4, 13.6, 3, 1.1, 1.05);
  const kroof = createThatchRoof(1.4, 2.1, 2.7, 3.5); kroof.rotation.y = Math.PI / 2; kroof.position.set(6.4, 0, 13.1); g.add(kroof);
  for (const x of [4.6, 8.2]) for (const z of [12, 14.3]) box(g, .16, 2.7, .16, timber(C.edge), x, 1.35, z);
  hangingSign(g, 'VĂN PHÒNG ĐẤT NỀN', 6.4, 2.3, 14.45, 2.8);
  const board = new T.Group(); board.position.set(9.4, 0, 14.6); board.rotation.y = -.4; g.add(board);
  for (const x of [-.7, .7]) box(board, .1, 1.6, .1, timber(C.edge), x, .8, 0);
  plaque(board, 'LÔ ĐẸP · SỔ RƠM', 0, 1.35, .06, 1.7, 0, { bg: '#f3e4c4', fg: '#7a3a14', border: '#8a5a30', font: '700 52px "Baloo 2", Georgia, serif' });
  for (let i = 0; i < 3; i++) cylinder(g, .07, .07, .55, 0xe9dcc0, 5.6 + i * .5, 1.12, 13.5, 8).rotation.z = Math.PI / 2;
  info.offsets = { npc_investor: [0, -2.4, 0, 1] };
  info.handles = new Set(['home_sign']);
  info.camera = { focus: [.6, 1, 4], pitch: .2, radius: 21, yaw: -.1, fov: 56, followPitch: .52, followRadius: 13 };
}
// B02-08 Ao và vườn: pond on the left with lily pads and a fishing jetty, raised vegetable beds on the
// right, two thatched produce stalls at the back, fruit trees and plank paths between.
function park(g, data, cosmetics, info) {
  ground(g, 56, 52, C.grass, 'grass', .004);
  path(g, [[0, 25], [1.6, 14], [2.4, 6.4]], 3.2, C.path); path(g, [[2.4, 6.4], [11, 6.4], [13.4, -8], [6, -12], [-5, -12]], 3, C.path);
  const shore = new T.Shape();
  for (let i = 0; i <= 48; i++) { const a = i / 48 * Math.PI * 2, r = 1 + .06 * Math.sin(a * 5) + .04 * Math.cos(a * 3); const x = Math.cos(a) * 9.6 * r, y = Math.sin(a) * 6.4 * r; i ? shore.lineTo(x, y) : shore.moveTo(x, y); }
  const bank = add(g, new T.ShapeGeometry(shore), material(0x5b4a2e), 0, .02, -2, { shadow: false }); bank.rotation.x = -Math.PI / 2; bank.scale.set(1.08, 1.1, 1);
  const water = add(g, new T.ShapeGeometry(shore), material(0x3f7f86, { roughness: .08, metalness: .25 }), 0, .06, -2, { shadow: false }); water.rotation.x = -Math.PI / 2;
  for (let i = 0; i < 26; i++) { const a = i / 26 * Math.PI * 2; rock(g, Math.cos(a) * 10.3 + Math.sin(i * 3) * .4, -2 + Math.sin(a) * 7.1 + Math.cos(i) * .3, .5 + (i % 3) * .25); if (i % 2) for (let k = 0; k < 4; k++) { const rd = cylinder(g, .025, .025, 1.1 + k * .1, 0x6f8a3a, Math.cos(a + .05 * k) * 9.6, .55, -2 + Math.sin(a + .05 * k) * 6.6, 4); rd.rotation.z = (k - 1.5) * .12; } }
  const rr = random(88);
  for (let i = 0; i < 22; i++) { const x = (rr() * 2 - 1) * 7, z = -2 + (rr() * 2 - 1) * 4.6; const pad = add(g, geometry('CircleGeometry', .38, 12), 0x5f8f3e, x, .075, z, { shadow: false }); pad.rotation.x = -Math.PI / 2; pad.scale.setScalar(.7 + rr() * .6); if (i % 4 === 0) { const f = sphere(g, .14, 0xf0b6c4, x, .14, z); f.scale.y = .6; } }
  // Fishing jetty from the front shore, two anglers with rods.
  for (let i = 0; i < 9; i++) box(g, 3.2, .12, .5, timber(i % 2 ? 0xa88a64 : 0x8a6c4b), -1.2, .42, 7.6 - i * .55);
  for (const sx of [-1, 1]) for (const zz of [7.4, 5.2, 3.2]) cylinder(g, .1, .1, 1.1, timber(C.edge), -1.2 + sx * 1.5, .2, zz, 8);
  lantern(g, .3, 1.2, 3.1); lantern(g, -2.7, 1.2, 3.1);
  for (const [x, yaw] of [[-2, Math.PI + .3], [-.4, Math.PI - .2]]) {
    const a = resident(g, info.ambient, cosmetics, 'Gà Câu Cá', x, 3.6, 'chat', x < -1 ? 'trang' : 'vang', x < -1 ? 'xanh' : 'luc', yaw, { y: .48 });
    beam(a.model, [.3, .8, .2], [.9, 1.9, 2.2], .03, C.light);
  }
  // Vegetable beds on the right inside a low fence.
  for (const [z, sd] of [[-6, 1], [-2.4, 2], [1.2, 3]]) vegBed(g, 18.4, z, 7, 2.2, sd);
  fence(g, [[14, -8.6], [23, -8.6], [23, 3.4]], 1.1); fence(g, [[14, 3.4], [16, 3.4]], 1.1);
  for (const [x, z] of [[22.4, -12], [23.4, 8.6], [-21, 8]]) { tree(g, x, z, 1.05, 0x4f7a36); for (let i = 0; i < 9; i++) sphere(g, .16, 0xe98b2b, x + Math.cos(i) * 1.6, 3.6 + Math.sin(i * 2) * .7, z + Math.sin(i) * 1.4); }
  // Two thatched produce stalls at the back.
  for (const [x, fill] of [[-4.4, 0xe9a23b], [5.6, 0x8fb04c]]) {
    counter(g, x, -14, 4.4, 1.2, 1);
    const roof = createThatchRoof(1.6, 2.7, 2.9, 3.8); roof.rotation.y = Math.PI / 2; roof.position.set(x, 0, -14.4); g.add(roof);
    for (const sx of [-1, 1]) for (const zz of [-13.2, -15.6]) box(g, .18, 2.9, .18, timber(C.edge), x + sx * 2.1, 1.45, zz);
    for (let i = 0; i < 4; i++) basket(g, x - 1.5 + i, 1, -14, .36, i % 2 ? fill : 0xd7532f, .3);
    for (let i = 0; i < 3; i++) crate(g, x - 1.2 + i * 1.2, -12.4, .55, i * .3);
  }
  resident(g, info.ambient, cosmetics, 'Gà Bán Rau', -4.4, -15.2, 'greet', 'trang', 'do', 0);
  resident(g, info.ambient, cosmetics, 'Gà Bán Quả', 5.6, -15.2, 'chat', 'vang', 'luc', 0, { equipped: { head: 'mu_rom' } });
  resident(g, info.ambient, cosmetics, 'Gà Chăm Rau', 16.4, -.6, 'sweep', 'trang', 'luc', -1.4);
  resident(g, info.ambient, cosmetics, 'Gà Chăm Rau', 20.6, -4.2, 'chat', 'vang', 'nau', -2, { equipped: { head: 'mu_rom' } });
  for (const [x, z] of [[11.4, 3.6], [-11.6, 7.4], [12.4, -10.8]]) lampPost(g, x, z, 3);
  hayMound(g, 12.6, 9.4, .7); barrel(g, 14.4, 8.2, true, .8); basket(g, 15.2, 0, 4.4, .4, 0xd7532f, .45);
  flowers(g, -16, 8, 3, 30, 8); flowers(g, 15, -14, 3, 30, 9); flowers(g, -6, 9, 2, 16, 10);
  surround(g, 26, 24, 27, [[0, 26, 6]], .7);
  grass(g, 25, 23, 260, 27, (x, z) => Math.hypot(x / 10.6, (z + 2) / 7.4) < 1.1 || x > 13.6 && x < 23.2 && z > -8.8 && z < 3.6);
  info.camera = { focus: [2.4, 1, -1.6], pitch: .31, radius: 20, yaw: .12, fov: 54, followPitch: .52, followRadius: 14 };
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
  const m = add(g, new T.ShapeGeometry(shape), (()=>{const m=farmMaterial('earth').clone();m.map=m.map.clone();m.map.repeat.set(1/12,1/12);m.map.needsUpdate=true;return m;})(), 0, .006, 0, { shadow: false });
  m.rotation.x = -Math.PI / 2;
}
function returnGap(data) { return data.entry ? [[data.entry.x, data.entry.z, 7]] : []; }

// B02-03 Vườn giun: dark dug soil with mounds, Lợn Ủn's tool shed back right, a plank walk from the
// entrance to its porch, a thatched lean-to and mesh fence on the left, trough and stump in front.
function shed(g, x, z, w, d, h = 2.6, ry = 0) {
  const s = new T.Group(); s.position.set(x, 0, z); s.rotation.y = ry; g.add(s);
  box(s, w, .2, d + 1.2, timber(C.light), 0, .1, .6);
  for (let i = 0; i < Math.round(w / .5); i++) box(s, w / Math.round(w / .5) - .04, h, .1, timber([0x7a5131, 0x8a5d39, 0x6e4a2c][i % 3]), -w / 2 + (i + .5) * w / Math.round(w / .5), h / 2, -d / 2);
  for (const sx of [-1, 1]) box(s, .1, h, d, timber(C.wood), sx * w / 2, h / 2, 0);
  for (const sx of [-1, 1]) for (const zz of [-d / 2, d / 2 + 1]) box(s, .22, h + .3, .22, timber(C.edge), sx * (w / 2 + .05), (h + .3) / 2, zz);
  const roof = createThatchRoof(d / 2 + 1.1, w / 2 + .7, h + .2, h + 1.9); roof.rotation.y = Math.PI / 2; roof.position.z = .5; s.add(roof);
  return s;
}
function garden(g, data, cosmetics, info) {
  zoneGround(g, data);
  const soil = add(g, geometry('CircleGeometry', 1, 32), material(0x4a3420, { roughness: 1 }), -.5, .012, -1.2, { shadow: false }); soil.rotation.x = -Math.PI / 2; soil.scale.set(8.2, 6.2, 1);
  for (const ends of [[-10.4, -9, 10.4, -9], [-10.4, -9, -10.4, 9], [10.4, -9, 10.4, 9]]) latticeWall(g, ...ends, 2.6);
  // Tool shed (server obstacle 7.4,-7.4): plank walls, thatch, shovels, ladder, worm sign, barrels.
  shed(g, 7.4, -7.6, 5.4, 2.4, 2.6);
  for (const [x, ry] of [[4.3, .2], [4.7, -.15]]) shovel(g, x, -6.1, ry, .18);
  box(g, .08, 2.2, .08, timber(C.light), 9.6, 1.1, -6); box(g, .08, 2.2, .08, timber(C.light), 10.1, 1.1, -6); for (let i = 0; i < 5; i++) box(g, .58, .06, .06, timber(C.light), 9.85, .3 + i * .42, -6);
  plaque(g, '〰', 8.6, 1.9, -6.2, 1, 0, { bg: '#d9c39a', fg: '#a2614f', border: '#7b5636', font: '700 150px Georgia, serif' });
  barrel(g, 5.2, -4.9, false, .8); bucket(g, 9.7, -4.3, 0xd98a8a); crate(g, 10, -3.2, .7, .3);
  woodWalk(g, [[7.4, 10.6], [8.6, 3], [8.5, -4.6]], 2);
  // Lean-to on the left: thatch on posts over a feed trough.
  const lean = createThatchRoof(1.9, 5.2, 2.5, 3.3); lean.position.set(-8.6, 0, -2.4); g.add(lean);
  for (const zz of [-7.4, -2.4, 2.6]) box(g, .22, 2.6, .22, timber(C.edge), -6.8, 1.3, zz);
  trough(g, -9, -4.6, false, 3.6);
  lantern(g, -6.8, 2.1, -2.2); lantern(g, -6.8, 2.1, 2.4);
  hay(g, -9, 1.2, .75);
  // Front: grain trough, chopping stump with an axe, worm sign post, worm bucket.
  const tg = new T.Group(); tg.position.set(-6.6, 0, 7.4); tg.rotation.y = Math.PI / 2; g.add(tg); trough(tg, 0, 0, false, 2.8);
  stump(g, -9.2, 5.6); beam(g, [-9.1, .62, 5.6], [-8.6, 1.3, 5.9], .05, C.light); box(g, .32, .2, .04, material(0x8d918c, { metalness: .5, roughness: .45 }), -9.15, .7, 5.5);
  box(g, .12, 1.4, .12, timber(C.edge), -9.6, .7, 3.4); plaque(g, '〰', -9.6, 1.45, 3.48, 1, 0, { bg: '#7b5636', fg: '#e8c79a', border: '#4d321e', font: '700 150px Georgia, serif' });
  tub(g, 5.6, 6.2, .6, .55); wormPile(g, 5.6, .56, 6.2, .7, .7, 12, 9);
  for (const [x, z] of [[-3.4, 3.4], [1.2, 3.8], [-6.2, -.6], [3.4, -5]]) wormPile(g, x, .03, z, .5, .5, 2, Math.round(x * 17 + z));
  flowers(g, -4, 7.6, 1.2, 10, 31); flowers(g, 3.6, 7.8, 1, 8, 32); flowers(g, -8.4, -7.6, 1, 8, 33);
  // Distant hut and field beyond the back fence (reference background).
  const hut = new T.Group(); hut.position.set(-5, 0, -17); g.add(hut); shed(hut, 0, 0, 4, 3, 2.2);
  box(g, 14, .1, 6, 0x6d5434, 6, .04, -15.5); for (let x = 0; x < 13; x += 1.3) bush(g, x, -15.5, .35, 0x7e9a4c);
  surround(g, data.halfX + 4, data.halfZ + 5, 51, returnGap(data), .8);
  grass(g, data.halfX, data.halfZ, 120, 11, (x, z) => data.items.some(i => Math.hypot(i.x - x, i.z - z) < 1.6) || Math.abs(x - 8.4) < 1.4);
  // Diggers with shovels, as in the reference.
  for (const [x, z, col, sh, yaw] of [[-2.8, 1.8, 'trang', 'xanh', 2.4], [2.9, -.2, 'trang', 'nau', -2.2], [-1.4, -4, 'vang', 'luc', 2.8]]) {
    const a = resident(g, info.ambient, cosmetics, 'Gà Đào Giun', x, z, 'sweep', col, sh, yaw);
    const sv = new T.Group(); a.model.add(sv); sv.position.set(.5, .2, .4); sv.rotation.z = -.4; a.broom = sv; shovel(sv, 0, 0, 0, 0);
  }
  info.handles = new Set(['gate']);
  info.camera = { focus: [1, .9, -1.6], pitch: .36, radius: 15.5, yaw: -.12, fov: 55, followPitch: .5, followRadius: 12 };
}
function stall(g,x,z,w,roof,ry=0){
 const s=new T.Group();s.position.set(x,0,z);s.rotation.y=ry;g.add(s);
 box(s,w,.9,1.25,timber(C.wood),0,.45,0);box(s,w+.2,.12,1.4,timber(C.light),0,.95,0);
 for(const sx of [-1,1])for(const zz of [-1,.7])box(s,.16,3.1,.16,timber(C.edge),sx*w/2,1.55,zz);
 const canopy=createThatchRoof(w/2+.3,1.3,3.1,3.9);s.add(canopy);
 return s;
}
// B02-01 Chợ thóc: worm counter + Lợn Ủn on the left, food stall under its own thatch in the middle,
// clothes racks on the right; a plank walk from the entrance to the food stall.
function market(g, data, cosmetics, info) {
  zoneGround(g, data);
  pavilion(g, info, 11.4, 9, 5.6, 9.6);
  woodWalk(g, [[0, 10.6], [0, -3.6]], 3.4);
  // Worm counter (server obstacle -7,-5.8): pink worm trays, brass scale, chalkboard, sacks behind.
  counter(g, -7, -5.7, 3.9, 1.3, 1);
  for (const [x, z] of [[-8.3, -5.8], [-7, -5.5], [-5.8, -5.9]]) { tray(g, x, 1, z, 1.05, .7, 0x6b4a2e); wormPile(g, x, 1.17, z, .8, .45, 14, Math.round(x * 31)); }
  brassScale(g, -5.2, 1, -5.3);
  chalkboard(g, -9.4, -3.4, .5, '〰');
  for (let i = 0; i < 5; i++) sack(g, -9.8 + i * .8, i > 2 ? .55 : 0, -7.6 + (i % 2) * .3, i);
  crate(g, -10.2, -5.6, .8, .2); crate(g, -10.2, -4.6, .6, -.3);
  hangingSign(g, 'Lợn Ủn', -7.2, 3.35, -4.3, 2.4);
  // Food stall: its own thatch canopy inside the hall, bowls of grain/seeds/greens, a water tub, grain banner.
  counter(g, 0, -5.7, 4.1, 1.3, 1);
  const canopy = createThatchRoof(2.7, 1.9, 3.05, 3.9); canopy.position.set(0, 0, -6.1); g.add(canopy);
  for (const sx of [-1, 1]) for (const zz of [-4.4, -7.8]) box(g, .2, 3.1, .2, timber(C.edge), sx * 2.5, 1.55, zz);
  [[-1.45, 0xe7b23e], [-.5, 0x8a6b3a], [.45, 0x87a650], [1.4, 0xece3c4]].forEach(([x, c]) => basket(g, x, 1, -5.5, .4, c, .3));
  for (let i = 0; i < 6; i++) sack(g, -1.8 + (i % 3) * 1.2, i > 2 ? .55 : 0, -7.5, i * .7);
  tub(g, 3.1, -5.3, .85, .8); barrel(g, -3.1, -6.8, false, .8);
  bannerFlag(g, 1.9, 2.4, -7.6, 0xe9dcc0, '🌾');
  hangingSign(g, 'Thức ăn', 0, 2.55, -3.9, 2.2);
  // Fashion corner: counter of folded clothes and caps, a long rack of shirts/dresses along the right wall.
  counter(g, 7, -5.7, 3.9, 1.3, 1);
  for (const [x, c] of [[5.8, 0xd8574c], [6.7, 0x7aa3c9], [7.6, 0x6f8f4a]]) capHat(g, x, 1.05, -5.6, c);
  strawHat(g, 8.4, 1.06, -5.6, 1.1);
  for (let i = 0; i < 3; i++) box(g, .6, .1, .45, [0xe9d7b0, 0x9fb7cf, 0xd8a8a0][i], 6.3 + i * .05, 1.06 + i * .1, -6.1);
  clothesRack(g, 9.9, -2.6, 6.2, -Math.PI / 2, [['shirt', 0x5c86b8], ['shirt', 0xece6d8], ['shirt', 0x6f8f4a], ['dress', 0xe9b84a], ['shirt', 0xd8574c], ['dress', 0x8e6bb0], ['shirt', 0x5c86b8]]);
  table(g, 8.4, 1.2, 1.4, .9, .8, 0xdcd3c0);
  for (const [x, c] of [[7.95, 0x5a6f8a], [8.45, 0xb04a3a], [8.95, 0x6f8f4a]]) capHat(g, x, .86, 1.0, c);
  strawHat(g, 8.4, .87, 1.45, 1.05, 0x3f6f5a);
  hangingSign(g, 'Trang phục', 9, 3.3, -1.2, 2.6, -Math.PI / 2);
  // Foreground: hay, grain barrel, crates, a bench; flowers along the walk.
  hayMound(g, -9.8, 7.4, 1.05); barrel(g, -8.2, 8.2, false, .9);
  for (let i = 0; i < 3; i++) basket(g, -7.4 + i * .6, 0, 8.6, .26, [0xe7b23e, 0x87a650, 0xe7b23e][i], .3);
  box(g, 2.6, .12, .6, timber(C.light), -5.8, .55, 8.3); for (const sx of [-1, 1]) box(g, .14, .55, .5, timber(C.edge), -5.8 + sx * 1.1, .27, 8.3);
  table(g, 9.2, 7.3, 1.5, 1, .82, 0xd6cfc0); stool(g, 7.9, 7.8);
  crate(g, 10.2, 5.2, .7, .3); hay(g, 10, -7.6, .55);
  flowers(g, -3, 6.8, 1.2, 10, 21); flowers(g, 3.2, 7.4, 1.2, 10, 22); flowers(g, -2.6, -1, .8, 6, 23);
  for (const [x, z] of [[-10.6, 2], [10.6, -6.5], [-10.6, -7.5]]) lantern(g, x, 2.6, z);
  surround(g, data.halfX + 2, data.halfZ + 2, 61, returnGap(data));
  // Shoppers and stall keepers (ambient; never counted as players).
  resident(g, info.ambient, cosmetics, 'Gà Bán Thóc', 0, -7, 'greet', 'trang', 'luc', 0);
  resident(g, info.ambient, cosmetics, 'Gà Thợ May', 7.2, -7, 'chat', 'kem', 'tim', 0, { equipped: { eye: 'kinh_tron' } });
  resident(g, info.ambient, cosmetics, 'Gà Đi Chợ', -4.3, -2.5, 'chat', 'vang', 'xanh', -2.4, { equipped: { head: 'mu_rom' } });
  resident(g, info.ambient, cosmetics, 'Gà Đi Chợ', -6.2, -1.6, 'chat', 'nau', 'luc', -2.9);
  resident(g, info.ambient, cosmetics, 'Gà Đi Chợ', 1.4, -3.2, 'greet', 'trang', 'do', 2.9);
  resident(g, info.ambient, cosmetics, 'Gà Đi Chợ', 6.4, -1.3, 'chat', 'trang', 'xanh', 2.2);
  resident(g, info.ambient, cosmetics, 'Gà Đi Chợ', 5.2, -3.6, 'walk', 'vang', 'luc', 0, { route: [[5.2, -3.6], [2.6, 2.4]] });
  info.offsets = { shop_sell: [0, -2.35, 0, 1.25] };
  info.handles = new Set(['shop_food', 'shop_fashion', 'gate']);
  info.camera = { focus: [0, 1.1, -3.6], pitch: .25, radius: 10.4, yaw: 0, fov: 60, followPitch: .4, followRadius: 9 };
}
// B02-04 Sân tỷ thí: chalk ring on packed earth, Chó Mực behind it, roofed bleachers on the left,
// emblem banners and benches on the right, a low front fence with spectators in the foreground.
function arena(g, data, cosmetics, info) {
  zoneGround(g, data);
  const ring = add(g, geometry('TorusGeometry', 4.4, .06, 4, 96), material(0xe9e2cf, { roughness: 1 }), 0, .03, .2, { shadow: false }); ring.rotation.x = Math.PI / 2; ring.scale.set(1, 1, .02);
  const dust = add(g, geometry('CircleGeometry', 4.3, 40), material(0x8c6a45, { roughness: 1 }), 0, .015, .2, { shadow: false }); dust.rotation.x = -Math.PI / 2;
  for (const ends of [[-9.4, -9, 9.4, -9]]) latticeWall(g, ...ends, 2.6);
  // Left bleachers (server obstacle -8,0): three tiers under a thatch lean-to, seated spectators.
  for (let r = 0; r < 3; r++) { box(g, .9, .35 + r * .42, 7, timber(r % 2 ? C.light : C.wood), -6.9 - r * .9, (.35 + r * .42) / 2, 0); box(g, 1, .08, 7.1, timber(C.light), -6.9 - r * .9, .35 + r * .42 + .04, 0); }
  const lean = createThatchRoof(2.2, 4.4, 3.4, 4.6); lean.position.set(-8, 0, 0); g.add(lean);
  for (const zz of [-3.7, 0, 3.7]) { box(g, .24, 3.5, .24, timber(C.edge), -6.2, 1.75, zz); lantern(g, -6, 2.4, zz + .2); }
  // Right benches (server obstacle 8,0), hay-bale seats, banners with the chicken emblem.
  for (let r = 0; r < 2; r++) { box(g, .9, .4 + r * .45, 7, timber(r % 2 ? C.light : C.wood), 7.1 + r * .9, (.4 + r * .45) / 2, 0); box(g, 1, .08, 7.1, timber(C.light), 7.1 + r * .9, .44 + r * .45, 0); }
  for (const zz of [-2.4, 2.6]) box(g, 1.4, .7, .9, textured('straw', 0xb8955a, [2, 1]), 6.1, .35, zz);
  for (const x of [3.2, 6.4]) { box(g, .18, 4.6, .18, timber(C.edge), x - .7, 2.3, -8.6); box(g, .18, 4.6, .18, timber(C.edge), x + .7, 2.3, -8.6); box(g, 1.7, .12, .12, timber(C.edge), x, 4.4, -8.6); bannerFlag(g, x, 3.4, -8.5, 0xd8c7a0, '🐓'); }
  for (const [x, z] of [[7.6, -5.6], [-7.4, -5.8], [7.8, 5.8]]) lampPost(g, x, z, 3);
  barrel(g, 7.8, -4.2, true, .8); crate(g, -3.6, -7.6, .8, .2); hay(g, -7.6, -7.4, .7);
  // Front fence with an opening at the middle; spectators behind it frame the shot.
  fence(g, [[-9.4, 8.4], [-2.6, 8.4]], 1.3); fence(g, [[2.6, 8.4], [9.4, 8.4]], 1.3);
  for (const [x, z, col, sh] of [[-6.2, 8.9, 'vang', 'xanh'], [-4.6, 9.1, 'trang', 'xanh'], [-3.4, 8.9, 'nau', 'do'], [5.6, 9, 'vang', 'luc']]) resident(g, info.ambient, cosmetics, 'Khán giả', x, z, 'greet', col, sh, Math.PI);
  for (let i = 0; i < 6; i++) resident(g, info.ambient, cosmetics, 'Khán giả', -6.9 - (i % 3) * .9, -2.6 + Math.floor(i / 3) * 3.4 + (i % 3) * .4, i % 2 ? 'greet' : 'chat', ['trang', 'vang', 'nau'][i % 3], ['xanh', 'do', 'luc'][i % 3], Math.PI / 2, { y: .35 + (i % 3) * .42 });
  for (let i = 0; i < 4; i++) resident(g, info.ambient, cosmetics, 'Khán giả', 7.1 + (i % 2) * .9, -2.2 + Math.floor(i / 2) * 4 + (i % 2) * .5, 'chat', ['vang', 'trang'][i % 2], ['do', 'xanh'][i % 2], -Math.PI / 2, { y: .44 + (i % 2) * .45 });
  surround(g, data.halfX + 3, data.halfZ + 4, 71, [...returnGap(data), [0, data.halfZ + 3, 8]], .8);
  info.handles = new Set(['gate']);
  info.camera = { focus: [0, .8, -1], pitch: .32, radius: 15, yaw: 0, fov: 56, followPitch: .5, followRadius: 12 };
}
// B02-06 Trạm ấp trứng: plank walk to Vịt Bông's counter at the back (under its own thatch), big
// egg nests on both sides, sacks and baskets behind, lattice walls and lanterns all round.
function hatchery(g,data,cosmetics,info){
 zoneGround(g,data);pavilion(g,info,9.4,8,5.4,9);woodWalk(g,[[0,9.6],[0,-4.2]],2.6);
 counter(g,0,-5.7,5.6,1.2,1);
 const canopy=createThatchRoof(4,1.9,3,3.9);canopy.position.set(0,0,-6.2);g.add(canopy);
 for(const sx of [-1,1])for(const zz of [-4.5,-7.8])box(g,.2,3.1,.2,timber(C.edge),sx*3.7,1.55,zz);
 for(let i=0;i<8;i++)sack(g,-3.4+(i%4)*1.1+(i>3?.5:0)+(i%4>1?1.3:0)*(i>3?0:1),i>3?.55:0,-7.7,i*.6);
 basket(g,1.9,1,-5.6,.38,0xe7b23e,.3);box(g,.7,.12,.5,timber(C.light),-1.6,1.06,-5.6);lantern(g,1.2,1.32,-5.4);
 chalkboard(g,3.1,-4.4,-.2,'🥚');basket(g,4.6,0,-6.2,.5,0xe7b23e,.5);basket(g,5.6,0,-5.6,.4,0x87a650,.45);
 crate(g,-4.6,-6.6,.8,.2);barrel(g,-5.8,-5.6,false,.8);
 hangingSign(g,'Vịt Bông',0,2.55,-4.3,1.9);
 // Egg nests on both sides of the walk, chickens brooding in a few of them.
 const egg=(x,z,s=1)=>{nest(g,x,z,1.25*s);const e=sphere(g,.3*s,0xf2e6c8,x,.44*s,z);e.scale.y=1.3;};
 for(const [x,z] of [[-4.2,-2.2],[4.4,-2.6],[-5.4,1.6],[5.2,1.2],[-6,5],[6.2,4.8]])egg(x,z);
 for(const [x,z] of [[-7.4,-.6],[7.6,-1.4]]){const t=new T.Group();t.position.set(x,0,z);g.add(t);trough(t,0,0,false,2.6);}
 for(const [x,z] of [[-8.4,4],[8.2,-5.4],[-8.2,-6]])barrel(g,x,z,false,.75);
 resident(g,info.ambient,cosmetics,'Gà Ấp Trứng',6.2,4.8,'sleep','trang','do',Math.PI+.4,{y:.15});
 resident(g,info.ambient,cosmetics,'Gà Chăm Trứng',-4.2,.6,'chat','vang','luc',2.6);
 resident(g,info.ambient,cosmetics,'Gà Chăm Trứng',-2.2,-1,'greet','nau','luc',2.2);
 for(const [x,z] of [[1,3.4],[-.6,5.2],[1.4,1.4]]){const chick=buildChicken({color:'vang'});chick.scale.setScalar(.42);chick.position.set(x,0,z);chick.rotation.y=Math.PI+x;g.add(chick);}
 info.offsets={npc_duck:[0,-3.05,0,1.15]};
 info.handles=new Set(['gate']);info.camera={focus:[0,1,-2.4],pitch:.27,radius:11.4,yaw:0,fov:60,followPitch:.42,followRadius:9};
 surround(g,12,11,81,returnGap(data));
}
function busShelter(g,x,z,ry=0){
 const s=new T.Group();s.position.set(x,0,z);s.rotation.y=ry;g.add(s);
 const roof=createThatchRoof(2.3,4.5,3.7,5);roof.rotation.y=Math.PI/2;s.add(roof);
 latticeWall(s,-3.9,-1.4,3.9,-1.4,2.2);
 for(const xx of [-3.9,3.9])for(const zz of [-1.4,1.9])box(s,.26,3.8,.26,timber(C.edge),xx,1.9,zz);
 box(s,5.8,.14,.8,timber(C.light),-.9,.62,-.6);for(const xx of [-3.2,-.9,1.4])box(s,.16,.6,.6,timber(C.edge),xx,.3,-.6);
 for(const xx of [-2.9,-1,.9])nest(s,xx,-.6,.6,.69);
 hangingSign(s,'BẾN XE CHIM CHÍP',0,2.72,1.9,4.2);
 for(const xx of [-3.9,3.9])lantern(s,xx,2.5,1.85);
 return s;
}
function road(g, x1, z1, x2, z2, w = 4) {
  path(g, [[x1, z1], [x2, z2]], w, 0x8b8578);
  const len = Math.hypot(x2 - x1, z2 - z1), n = Math.floor(len / 2.2);
  for (let i = 1; i < n; i++) { const t = i / n; const m = box(g, .16, .02, .9, 0xf4f0e6, x1 + (x2 - x1) * t, .05, z1 + (z2 - z1) * t); m.rotation.y = Math.atan2(x2 - x1, z2 - z1); }
}
// B02-09 Bến xe Chim Chíp: thatched shelter with nest seats and the ticket counter on the left, the big
// olive bus on the right with a plank boarding ramp at its door, the dirt road running on behind it.
function station(g,data,cosmetics,info,far=false){
 zoneGround(g,data);
 path(g,[[6.6,data.halfZ+6],[6.6,2],[6.6,-data.halfZ-10]],5.6,0x9a7a52);
 busShelter(g,-6.4,-6.6);
 counter(g,-4.6,-4.4,2.6,1.1,1.05);box(g,.6,.08,.4,material(0xe9dcc0),-4.2,1.1,-4.3);lantern(g,-3.6,1.35,-4.3);
 const routeBoard=new T.Group();routeBoard.position.set(-2.4,0,-4.6);g.add(routeBoard);box(routeBoard,.1,2,.1,timber(C.edge),0,1,0);plaque(routeBoard,'🚌 Tuyến',0,1.75,.06,1.1,0,{bg:'#2f3a33',fg:'#e9dcc0',border:'#4d321e',font:'700 90px Georgia, serif'});
 const bus=buildBus();bus.position.set(6.6,0,-4.2);bus.rotation.y=-.12;bus.scale.setScalar(1.5);g.add(bus);g.userData.bus=bus;
 woodWalk(g,[[3.4,-1.2],[4.8,-3.4]],1.6);
 for(const [x,z] of [[-10,1],[-9.6,4.6]])hayMound(g,x,z,.9);
 fence(g,[[-11,7.6],[-3,7.6]],1.3);fence(g,[[10.6,-8],[10.6,6.5]],1.3);
 const sign=new T.Group();sign.position.set(3.4,0,6.4);g.add(sign);box(sign,.18,1.9,.18,timber(C.edge),0,.95,0);plaque(sign,'🚌',0,1.6,.1,.9,0,{bg:'#3a2c1e',fg:'#efe3c4',border:'#2a1f15',font:'700 120px Georgia, serif'});
 barrel(g,-8.6,6.2,false,.8);const tg=new T.Group();tg.position.set(-8.6,0,7.6);tg.rotation.y=Math.PI/2;g.add(tg);trough(tg,0,0,true,2.4);
 flowers(g,1.4,5.2,1.4,12,61);flowers(g,9.4,5,1.2,10,62);
 lampPost(g,-10.4,-1.6,3);lampPost(g,10,-2,3.2);
 for(let i=0;i<4;i++)resident(g,info.ambient,cosmetics,'Gà Chờ Xe',3.1+i*.55,-.4-i*1.15,'chat',['vang','trang','nau','vang'][i],['luc','tim','xanh','do'][i],-1.9+i*.1,{equipped:i%2?{head:'mu_rom'}:{}});
 resident(g,info.ambient,cosmetics,'Gà Đi Xe',-7.6,-4.6,'chat','vang','xanh',.3,{y:.69});
 resident(g,info.ambient,cosmetics,'Gà Đi Xe',-.6,1.8,'walk','trang','tim',2.6,{route:[[-.6,1.8],[2.2,-.4]]});
 surround(g,data.halfX+3,data.halfZ+3,far?91:93,[...returnGap(data),[6.6,-data.halfZ-3,7],[6.6,data.halfZ+3,7]],.8);
 info.offsets={npc_bird:[0,-1.4,0,1]};
 info.handles=new Set(['bus_stop','parking','road','gate']);info.camera={focus:[.4,1.4,-3],pitch:.2,radius:12.4,yaw:-.1,fov:58,followPitch:.48,followRadius:12};
}
// B02-11 Cửa hàng xe: thatched open workshop, the mechanic's bench and tool wall at the back, three
// display bays with painted outlines (bicycle, scooter, car) and little icon signs, tyres on the right.
function garage(g,data,cosmetics,info){
 zoneGround(g,data);pavilion(g,info,9.4,8,5.4,9);
 counter(g,0,-6.5,6,1.2,1.05);
 box(g,8.6,3,.2,timber(C.wood),0,1.5,-7.9);for(let i=0;i<16;i++)box(g,.5,2.9,.05,timber([0x7a5131,0x8a5d39,0x6e4a2c][i%3]),-4+i*.53,1.5,-7.78);
 for(let i=0;i<9;i++){box(g,.05,.5,.04,0x8d918c,-2.4+i*.55,2.15,-7.72);add(g,geometry('TorusGeometry',.08,.025,6,10),0x8d918c,-2.4+i*.55,2.45,-7.72);}
 const tire=add(g,geometry('TorusGeometry',.42,.15,10,22),0x2e302b,3.4,2.3,-7.7);
 hangingSign(g,'Cửa hàng xe',0,3.65,-5.2,3.6);
 for(const [x,z] of [[-3.6,-7.2],[3.8,-7.1]])crate(g,x,z,.7,.2);
 box(g,.9,.06,.6,material(0xe9dcc0),1.8,1.11,-6.4);lantern(g,-1.6,1.4,-6.3);
 for(const [k,x,icon] of [['xe_dap',-4.8,'🚲'],['xe_may',0,'🛵'],['o_to',4.9,'🚗']]){
  const v=buildVehicle(k);v.position.set(x,0,-1.6);v.rotation.y=k==='o_to'?-.18:.25;v.scale.setScalar(k==='o_to'?1.35:1.3);g.add(v);
  for(const sx of [-1,1])box(g,.1,.015,4,0xe8dfc4,x+sx*1.7,.035,-1.6);box(g,3.5,.015,.1,0xe8dfc4,x,.035,.4);
  const sg=new T.Group();sg.position.set(x,0,1.3);g.add(sg);for(const xx of [-.5,.5])box(sg,.1,.6,.1,timber(C.edge),xx,.3,0);plaque(sg,icon,0,.62,.06,1.2,0,{bg:'#7b5636',fg:'#efe3c4',border:'#4d321e',font:'700 120px Georgia, serif'});
 }
 for(let i=0;i<4;i++){const t=add(g,geometry('TorusGeometry',.42,.16,10,22),0x2e302b,8,.18+i*.3,-3.6);t.rotation.x=Math.PI/2;}
 for(let i=0;i<3;i++){const t=add(g,geometry('TorusGeometry',.42,.16,10,22),0x2e302b,8.3,.18+i*.3,-1.9);t.rotation.x=Math.PI/2;}
 hayMound(g,-8,-3.4,.85);const tg=new T.Group();tg.position.set(-7.8,0,1.6);tg.rotation.y=Math.PI/2;g.add(tg);trough(tg,0,0,true,2.4);
 chalkboard(g,8.4,1.4,-.5,'🔧');stool(g,7.6,-5.4);barrel(g,-8.4,-6.4,false,.8);
 fence(g,[[-9.4,8.2],[-2.2,8.2]],1.2);fence(g,[[2.2,8.2],[9.4,8.2]],1.2);
 for(const x of [-6.4,6.4])lampPost(g,x,7.2,2.6);
 resident(g,info.ambient,cosmetics,'Gà Mua Xe',6.8,.6,'chat','vang','xanh',-1.2);
 resident(g,info.ambient,cosmetics,'Gà Mua Xe',7.7,1,'greet','trang','nau',-1.6);
 resident(g,info.ambient,cosmetics,'Gà Thợ Phụ',1.3,-7.3,'greet','trang','do',0);
 woodWalk(g,[[0,10],[0,5.2]],2.6);
 surround(g,data.halfX+3,data.halfZ+3,101,returnGap(data),.8);
 info.offsets={npc_goat:[-1.4,-2.7,0,1.05]};
 info.handles=new Set(['parking','gate']);info.camera={focus:[0,1,-2.4],pitch:.3,radius:12.2,yaw:0,fov:60,followPitch:.42,followRadius:9};
}
// B02-10 Ven Suối: long thatched coop on the left, dig plots, a rocky stream along the right with a plank
// bridge at the crossing (server stream obstacles, gap at z = -5), white flowers and reeds on the banks.
function vensuoi(g, data, cosmetics, info) {
  zoneGround(g, data);
  ground(g, data.halfX * 2 + 40, data.halfZ * 2 + 40, C.grass, 'grass', -.01);
  const bed = add(g, geometry('PlaneGeometry', 3.6, data.halfZ * 2 + 30), material(0x4c3d27), 9.4, .02, 0, { shadow: false }); bed.rotation.x = -Math.PI / 2;
  const water = add(g, geometry('PlaneGeometry', 2.5, data.halfZ * 2 + 30), material(0x4f8f9a, { roughness: .08, metalness: .25 }), 9.4, .07, 0, { shadow: false }); water.rotation.x = -Math.PI / 2;
  const rr = random(113);
  for (let z = -data.halfZ - 14; z < data.halfZ + 14; z += .9) {
    for (const sx of [-1, 1]) { rock(g, 9.4 + sx * (1.35 + rr() * .5), z + rr() * .5, .4 + rr() * .45); if (rr() < .45) for (let k = 0; k < 3; k++) cylinder(g, .02, .02, .8 + rr() * .5, 0x6f8a3a, 9.4 + sx * (2.1 + rr() * .6), .45, z + k * .12, 4); }
    if (rr() < .35) { rock(g, 9.4 + (rr() - .5) * 1.4, z, .3); const foam = add(g, geometry('CircleGeometry', .25, 8), material(0xe9f0ee, { roughness: .4 }), 9.4 + (rr() - .5) * 1.6, .08, z, { shadow: false }); foam.rotation.x = -Math.PI / 2; foam.scale.set(1.4, .6, 1); }
  }
  // Plank bridge with rope rails and lanterns at the crossing.
  const bridge = new T.Group(); bridge.position.set(9.4, 0, -5); g.add(bridge);
  for (let i = 0; i < 9; i++) box(bridge, .42, .12, 1.8, timber(i % 2 ? 0xa88a64 : 0x8a6c4b), -1.9 + i * .47, .42, 0);
  for (const sx of [-1, 1]) for (const sz of [-1, 1]) { box(bridge, .14, 1.2, .14, timber(C.edge), sx * 2.1, .6, sz * .95); }
  for (const sz of [-1, 1]) box(bridge, 4.3, .06, .06, 0xb39463, 0, 1, sz * .95);
  lantern(g, 7.2, 1.5, -4); lantern(g, 11.6, 1.5, -6);
  // Long coop on the left edge (server obstacle -10.2,-2).
  coopShed(g, -10.2, -2, 10, 2.6, Math.PI / 2);
  for (const z of [-5.4, 1.4]) { const t = new T.Group(); t.position.set(-8.3, 0, z); g.add(t); trough(t, 0, 0, false, 2.4); }
  for (const [x, z] of [[-9.6, 6.6], [-6.4, -8.2]]) hayMound(g, x, z, .8);
  path(g, [[0, 10], [2, 2], [5.4, -4.4], [7.6, -5]], 2.4, C.path);
  for (const [x, z] of [[-3, 6.4], [4.6, 6], [-4.4, -7.4], [6.6, 2.6]]) flowers(g, x, z, 1.3, 12, Math.round(x * 13 + z));
  const sp = new T.Group(); sp.position.set(-1.6, 0, 8.2); g.add(sp); box(sp, .14, 2.2, .14, timber(C.edge), 0, 1.1, 0);
  plaque(sp, 'Trạm về Hypex ↓', .9, 1.9, .08, 2.2, 0, { bg: '#6a4426', fg: '#f6e3b4', border: '#3f2814', font: '700 60px "Baloo 2", Georgia, serif' });
  fence(g, [[-11.4, -9.2], [7, -9.2]], 1.1); fence(g, [[-11.4, 9.2], [-3.4, 9.2]], 1.1);
  surround(g, data.halfX + 6, data.halfZ + 4, 111, [...returnGap(data), [9.4, -data.halfZ - 4, 4], [9.4, data.halfZ + 4, 4]], .8);
  grass(g, data.halfX, data.halfZ, 140, 13, (x, z) => x > 7.2 || x < -8.4 || data.items.some(i => Math.hypot(i.x - x, i.z - z) < 1.5));
  for (const [x, z, c, sh] of [[-5.2, -1.6, 'vang', 'xanh'], [1.2, -3.6, 'trang', 'nau'], [-2.4, 3.8, 'vang', 'luc']]) {
    const a = resident(g, info.ambient, cosmetics, 'Gà Đào Giun', x, z, 'sweep', c, sh, 2.6);
    const sv = new T.Group(); a.model.add(sv); sv.position.set(.5, .2, .4); sv.rotation.z = -.4; a.broom = sv; shovel(sv, 0, 0, 0, 0);
  }
  resident(g, info.ambient, cosmetics, 'Gà Qua Cầu', 9.4, -5, 'walk', 'trang', 'do', Math.PI / 2, { y: .48 });
  info.handles = new Set(['gate']);
  info.camera = { focus: [1, .9, -1.6], pitch: .4, radius: 17, yaw: -.32, fov: 55, followPitch: .52, followRadius: 13 };
}

// B02-05 Nhà Hội: big thatched hall, central planning table with blueprints and a lamp, nests around it,
// a raised nest under the guild banner at the back, the shared task board, a cart of timber in front.
function guildhall(g,data,cosmetics,info){
 ground(g,data.halfX*2+30,data.halfZ*2+30,C.grass,'grass',-.01);zoneGround(g,data);
 pavilion(g,info,data.halfX,data.halfZ,5.6,9.6);woodWalk(g,[[0,10.6],[0,3.4]],3);woodWalk(g,[[0,-3.6],[0,-6.2]],2.4);
 table(g,0,-.6,4.4,2.6,.75);
 const plan=add(g,geometry('PlaneGeometry',2.6,1.7),new T.MeshStandardMaterial({map:textCanvas('HYPEX · BẢN VẼ',{bg:'#e4d3a8',fg:'#765b34',border:'#a68e63',font:'500 44px Georgia, serif'}),roughness:1}),-.3,.81,-.6,{shadow:false});plan.rotation.x=-Math.PI/2;plan.rotation.z=.08;
 lantern(g,1.5,1.05,-.9);for(let i=0;i<3;i++)cylinder(g,.05,.05,.7,0xe7dcc0,-1.6+i*.15,.84,.1,8).rotation.z=Math.PI/2;
 for(let i=0;i<8;i++){const a=i/8*Math.PI*2;nest(g,Math.sin(a)*3.6,-.6+Math.cos(a)*2.6,.9);}
 box(g,6,.4,2.8,timber(C.wood),0,.2,-6.8);nest(g,0,-6.8,1.4,.4);
 for(const sx of [-1,1])box(g,.16,3.6,.16,timber(C.edge),sx*1.2,1.8,-7.9);box(g,2.6,.12,.12,timber(C.edge),0,3.5,-7.9);bannerFlag(g,0,2.7,-7.86,0xe9dcc0,'🐓');
 const board=new T.Group();board.position.set(6.4,0,-6.4);board.rotation.y=-.4;g.add(board);for(const sx of [-1,1])box(board,.14,2.4,.14,timber(C.edge),sx*1.4,1.2,0);
 box(board,3,1.5,.1,timber(C.light),0,1.6,0);plaque(board,'Nhiệm vụ chung',0,2.55,.08,2.6,0,{bg:'#6a4426',fg:'#f6e3b4',border:'#3f2814',font:'700 64px "Baloo 2", Georgia, serif'});
 for(let i=0;i<4;i++){const n=box(board,.55,.5,.02,[0xf2e8cf,0xe8d9b0,0xf2e8cf,0xdfe8d4][i],-1+i*.65,1.55+(i%2)*.1,.07);n.rotation.z=(i-1.5)*.05;}
 // Timber cart, stone pile and sacks: materials for the guild projects.
 const cart=new T.Group();cart.position.set(-7.4,0,4.4);cart.rotation.y=.5;g.add(cart);box(cart,2.2,.18,1.3,timber(C.wood),0,.7,0);for(const sx of [-1,1])box(cart,2.2,.4,.08,timber(C.edge),0,.95,sx*.62);
 for(const sz of [-1,1]){const w=add(cart,geometry('TorusGeometry',.42,.07,8,18),timber(C.edge),-.2,.42,sz*.75);w.rotation.y=0;}beam(cart,[1.1,.7,0],[2.4,.3,0],.08,C.edge);
 for(let i=0;i<5;i++)box(cart,2,.18,.2,timber(C.light),0,.92+Math.floor(i/3)*.18,-.4+(i%3)*.4);
 for(let i=0;i<7;i++)rock(g,-8.6+(i%4)*.6,6.6+Math.floor(i/4)*.5,.6);
 for(let i=0;i<4;i++)sack(g,-9.6+(i%2)*.7,i>1?.55:0,1.6,i);hayMound(g,-8.6,-5.6,.9);
 for(let i=0;i<6;i++)box(g,2.4,.16,.3,timber(C.light),8.2,.1+Math.floor(i/3)*.17,2+(i%3)*.38);
 for(const [x,z] of [[8.4,-1.6],[-8.6,-1.4]]){const t=new T.Group();t.position.set(x,0,z);g.add(t);trough(t,0,0,false,2.4);}
 fence(g,[[-data.halfX,data.halfZ+.4],[-1.8,data.halfZ+.4]],1.2);fence(g,[[1.8,data.halfZ+.4],[data.halfX,data.halfZ+.4]],1.2);
 for(let i=0;i<5;i++){const a=i/8*Math.PI*2+.4;resident(g,info.ambient,cosmetics,'Gà Hội Viên',Math.sin(a)*3.6,-.6+Math.cos(a)*2.6,'chat',['kem','vang','trang','nau','vang'][i],['do','xanh','luc','tim','do'][i],a+Math.PI,{y:.06});}
 surround(g,data.halfX+4,data.halfZ+4,131,[[0,data.halfZ+4,6]],.7);
 info.handles=new Set();info.camera={focus:[0,1,-1.6],pitch:.33,radius:13.6,yaw:0,fov:60,followPitch:.42,followRadius:9};
}
// One Points draw per area for every lantern halo (positions in the area's frame).
function glows(g) {
  g.updateMatrixWorld(true);
  const inv = new T.Matrix4().copy(g.matrixWorld).invert(), v = new T.Vector3(), pts = [];
  g.traverse(o => { if (o.userData.glow) { v.setFromMatrixPosition(o.matrixWorld).applyMatrix4(inv); pts.push(v.x, v.y, v.z); } });
  if (!pts.length) return;
  const geo = new T.BufferGeometry(); geo.setAttribute('position', new T.Float32BufferAttribute(pts, 3)); geo.computeBoundingSphere();
  const halo = new T.Points(geo, glowMaterial()); halo.renderOrder = 3; g.add(halo);
}
// Batch decorative meshes, retaining animated and camera-faded objects separately. Static meshes that
// share a material (and shadow flag, and map chunk outdoors) become one draw: many copies of one shape
// are instanced, everything else is merged into a single pre-transformed geometry.
function batchStatic(g, info, chunk = Infinity) {
  for (const occ of info?.occluders || []) if (occ.parent) batchStatic(occ);
  g.updateMatrixWorld(true);
  // r128 bounds an InstancedMesh by its base geometry only, so fade boxes are measured before batching.
  if (g.userData.dynamic && !g.userData.box) { g.userData.box = new T.Box3().setFromObject(g); g.userData.fade = 1; }
  const buckets = new Map(), inv = new T.Matrix4().copy(g.matrixWorld).invert(), wp = new T.Vector3();
  g.traverse(o => {
    if (!o.isMesh || o.isInstancedMesh || Array.isArray(o.material) || o.material.transparent) return;
    for (let p = o.parent; p && p !== g; p = p.parent) if (p.userData.dynamic) return;
    // The open countryside is batched in chunks so the frustum and far plane skip distant patches.
    o.getWorldPosition(wp);
    const k = o.material.uuid + ':' + o.castShadow + (chunk < Infinity ? ':' + Math.floor(wp.x / chunk) + ',' + Math.floor(wp.z / chunk) : '');
    if (!buckets.has(k)) buckets.set(k, []); buckets.get(k).push(o);
  });
  const m = new T.Matrix4(), b = new T.Box3();
  for (const list of buckets.values()) {
    if (list.length < 2) continue;
    const base = list[0].geometry, same = list.every(o => o.geometry === base);
    if (same && list.length >= 8) {
      // Many copies of one shape: instancing keeps memory flat. Each batch gets its own bounding sphere.
      base.boundingBox || base.computeBoundingBox();
      const bounds = new T.Box3(), inst = new T.InstancedMesh(base.clone(), list[0].material, list.length);
      inst.castShadow = list[0].castShadow; inst.receiveShadow = true;
      list.forEach((o, i) => { m.multiplyMatrices(inv, o.matrixWorld); inst.setMatrixAt(i, m); bounds.union(b.copy(base.boundingBox).applyMatrix4(m)); o.parent.remove(o); });
      inst.geometry.boundingSphere = bounds.getBoundingSphere(new T.Sphere());
      inst.instanceMatrix.needsUpdate = true; g.add(inst);
    } else {
      const merged = mergeInto(list, inv);
      if (!merged) continue;
      const mesh = new T.Mesh(merged, list[0].material); mesh.castShadow = list[0].castShadow; mesh.receiveShadow = true; g.add(mesh);
      list.forEach(o => o.parent.remove(o));
    }
  }
}
// Pre-transformed union of several meshes (positions, normals, and UVs when every part has them).
function mergeInto(list, inv) {
  const withUv = list.every(o => o.geometry.attributes.uv);
  let count = 0;
  const parts = list.map(o => { const geo = o.geometry.index ? o.geometry.toNonIndexed() : o.geometry; count += geo.attributes.position.count; return [o, geo]; });
  if (count > 600000) return null;
  const pos = new Float32Array(count * 3), nor = new Float32Array(count * 3), uv = withUv ? new Float32Array(count * 2) : null;
  const rel = new T.Matrix4(), nm = new T.Matrix3(), v = new T.Vector3();
  let at = 0;
  for (const [o, geo] of parts) {
    rel.multiplyMatrices(inv, o.matrixWorld); nm.getNormalMatrix(rel);
    const P = geo.attributes.position, N = geo.attributes.normal, U = geo.attributes.uv;
    for (let i = 0; i < P.count; i++, at++) {
      v.fromBufferAttribute(P, i).applyMatrix4(rel); pos[at * 3] = v.x; pos[at * 3 + 1] = v.y; pos[at * 3 + 2] = v.z;
      if (N) { v.fromBufferAttribute(N, i).applyMatrix3(nm).normalize(); nor[at * 3] = v.x; nor[at * 3 + 1] = v.y; nor[at * 3 + 2] = v.z; }
      if (uv) { uv[at * 2] = U.getX(i); uv[at * 2 + 1] = U.getY(i); }
    }
    if (geo !== o.geometry) geo.dispose();
  }
  const out = new T.BufferGeometry();
  out.setAttribute('position', new T.BufferAttribute(pos, 3)); out.setAttribute('normal', new T.BufferAttribute(nor, 3));
  if (uv) out.setAttribute('uv', new T.BufferAttribute(uv, 2));
  out.computeBoundingSphere();
  return out;
}

const BUILDERS = { guildhall, lobby: hall, village: countryside, vs_field: countryside, neighborhood, park, garden, market, arena, hatchery, station, garage, vensuoi, vs_station: (g, d, c, i) => station(g, d, c, i, true) };
export function buildScenery(id, group, data, cosmetics) {
  const info = { handled: false, ambient: [], occluders: [], handles: new Set(), camera: null };
  const fn = BUILDERS[id];
  if (!fn) return info;
  info.handled = true; treeSink = []; sinkRoot = group;
  try { fn(group, data, cosmetics, info); } finally { info.treeList = treeSink; treeSink = null; sinkRoot = null; }
  glows(group); if(!data.outdoor && id!=='lobby'){dressGround(group,data.halfX-.6,data.halfZ-.6,id.length*97,(x,z)=>Math.abs(x)<1.5||data.items.some(i=>Math.hypot(x-i.x,z-i.z)<1.3)||(data.obstacles||[]).some(o=>Math.abs(x-o.x)<o.w/2+.3&&Math.abs(z-o.z)<o.d/2+.3));} batchStatic(group, info, data.outdoor ? 60 : Infinity);
  info.treeReg = instanceTrees(group, info.treeList, data.outdoor ? 60 : Infinity); info.treeReg.faded = new Map(); info.treeList = null;
  return info;
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

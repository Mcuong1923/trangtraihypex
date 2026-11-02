// Dựng cảnh 3D các khu từ dữ liệu zone của máy chủ. Vị trí tương tác lấy từ server;
// phần trang trí (ổ rơm, bao thóc, sạp…) là của client theo bộ ảnh minh hoạ.
import { mat, buildPig, buildDog, buildElder, buildCow, buildCoop, buildDecor, buildDuck, buildBird, buildGoat, buildFrog, buildBus, buildVehicle, buildInvestor, label } from './models.js';

import {CameraController} from './camera.js';
import {buildScenery, updateScenery, gateArch, occluder, fadeTrees, setNightGlow} from './scenery.js';
import {toLocal} from './navigation.js';

const T = window.THREE;

export class World {
  constructor(container) {
    this.renderer = new T.WebGLRenderer({ antialias: true });
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    this.renderer.shadowMap.enabled = true;
    this.renderer.shadowMap.type = T.PCFSoftShadowMap;
    // Output vẫn tuyến tính (màu khai báo theo hex sRGB như bảng màu ảnh minh hoạ); ACES nén vùng sáng
    // cho tông ấm, có chiều sâu như bộ ảnh 2 thay vì bạc trắng khi nhiều nguồn sáng cộng lại.
    this.renderer.toneMapping = T.ACESFilmicToneMapping;
    this.renderer.toneMappingExposure = .66;
    container.appendChild(this.renderer.domElement);

    this.scene = new T.Scene();
    this.scene.background = skyTexture();
    // Fog hides the far plane, so distant chunks of the big map are never drawn; its haze matches the horizon.
    this.scene.fog = new T.FogExp2(0xcbd8d4, .0115);
    this.camera = new T.PerspectiveCamera(45, 1, 0.1, 330);
 this.controls=new CameraController(this.camera);
 this.routeGroup=new T.Group();this.scene.add(this.routeGroup);
    // yaw 0 = camera đứng phía lối vào (+z) nhìn vào trong, như góc chụp của bộ ảnh minh hoạ.
    this.camYaw = 0; this.camTargetYaw = 0; this.camDist = 15;

    // Nắng chiều ấm + trời xanh dịu: mặt hướng lên ~1,4 lần màu gốc trước tone mapping, mặt khuất ~0,6.
    const sun = new T.DirectionalLight(0xffd9a6, 1.0);
    sun.position.set(-18, 38, 20); sun.castShadow = true;
    sun.shadow.mapSize.set(2048, 2048);
    // The shadow box travels with the chicken across the big map (see update()).
    Object.assign(sun.shadow.camera, { left: -45, right: 45, top: 45, bottom: -45, near: 1, far: 130 });
    this.hemi = new T.HemisphereLight(0xe2e6f0, 0x6b5333, 0.42); this.amb = new T.AmbientLight(0xffeed7, 0.12);
    this.scene.add(sun, sun.target, this.hemi, this.amb);
    // Dãy núi xa mờ sương ở chân trời, luôn đi theo người chơi nên không bao giờ tới gần được.
    this.horizon = mountains(); this.scene.add(this.horizon);

    sun.shadow.bias = -.00025; sun.shadow.normalBias = .035;
    this.sun = sun;
    this.reducedMotion = false;
    this.zones = {};   // id → { group, data (world coords), local (area coords), npcs: [] }
    this.current = null;
    this.active = [];  // zones drawn now: every area of the region on the seamless map, or one home
    this.resize();
    window.addEventListener('resize', () => this.resize());
  }

  // Mức đồ hoạ (mục 14, 18.1): giảm mật độ điểm ảnh và bóng đổ trên máy yếu.
  setQuality(q) {
    const dpr = window.devicePixelRatio || 1;
    this.renderer.setPixelRatio(q === 'low' ? 1 : q === 'mid' ? Math.min(dpr, 1.5) : Math.min(dpr, 2));
    this.renderer.shadowMap.enabled = q !== 'low';
    this.lowQuality = q === 'low';
    // Warm, slightly richer grade like the bo-2 renders; skipped on the low preset to save a compositing pass.
    this.renderer.domElement.style.filter = q === 'low' ? '' : 'saturate(1.14) contrast(1.05)';
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

  // Dựng các khu chưa có, mỗi khu đặt đúng chỗ trên bản đồ liền của trại.
  // Khu "home" khác nhau theo từng nhà nên dựng qua setHome().
  build(zones, cosmetics, world) {
    this.worldInfo = world || null;
    this.regionIds = world ? Object.keys(zones).filter((id) => id !== 'home' && id !== 'guildhall') : [];
    for (const id of this.regionIds) {
      if (this.zones[id]) continue;
      const data = zones[id], local = localData(data, zones, world);
      const group = new T.Group();
      group.position.set(data.cx || 0, 0, data.cz || 0); group.rotation.y = data.flip ? Math.PI : 0;
      group.visible = false;
      this.scene.add(group); group.updateMatrixWorld(true);
      const z = { group, data, local, npcs: [] };
      this.zones[id] = z;
      z.scenery = buildScenery(id, group, local, cosmetics);
      if (!z.scenery?.handled) { ground(group, data.halfX * 2 + 1.2, data.halfZ * 2 + 1.2, 0xc9a978); fence(group, data.halfX * 2 + 1.2, data.halfZ * 2 + 1.2); }
      local.items.forEach((it, i) => this.placeItem(z, it, cosmetics, data.items[i]));
      // A welcome arch where the village road reaches the area (the hall has its own doorway).
      if (local.entry && !data.outdoor && id !== 'lobby' && z.scenery.handled) {
        const arch = gateArch(data.title, 0, 5.2), e = local.entry;
        arch.position.set(e.x, 0, e.z - Math.sign(e.z) * 2.2); group.add(arch); occluder(z.scenery, arch);
      }
    }
  }

  placeItem(z, it, cosmetics, worldItem = it) {
    const g = z.group;
    if(it.kind==='nest'||it.kind==='npc_guide'||z.scenery?.handles?.has(it.kind)&&it.kind!=='gate')return;
    let obj;
    switch (it.kind) {
      case 'npc_elder': obj = buildElder(cosmetics); break;
      case 'npc_pig': case 'shop_sell': obj = buildPig(); break;
      case 'pk': obj = buildDog(); break;
      case 'npc_cow': obj = buildCow(); break;
      case 'npc_duck': obj = buildDuck(); break;
      case 'npc_bird': obj = buildBird(); break;
      case 'npc_goat': obj = buildGoat(); break;
      case 'npc_investor': obj = buildInvestor(cosmetics); break;
      case 'bed': obj = new T.Group(); break;
      case 'npc_frog': obj = buildFrog(); break;
      case 'bus_stop': obj = busStop(); break;
      case 'parking': obj = parkingSign(); break;
      case 'road': obj = roadSign(); break;
      case 'dig': obj = digMound(it.label); break;
      case 'gate': {
        // Lobby door is architecture; elsewhere each exit is an arch named after where it leads.
        if (z.data.id === 'lobby' || !z.scenery?.handled) { obj = z.scenery?.handled ? new T.Group() : signpost(); break; }
        const side = Math.abs(it.x) / z.data.halfX > Math.abs(it.z) / z.data.halfZ;
        const title = it.label.replace(/^(Ra|Về|Vào) /, '');
        obj = gateArch(title[0].toUpperCase() + title.slice(1), side ? Math.PI / 2 : 0, 4.4);
        occluder(z.scenery, obj); break;
      }
      case 'board': case 'home_board': obj = noticeBoard(); break;
      case 'shop_food': case 'shop_fashion': case 'trough': case 'home_sign': case 'home_exit': case 'guild_board': obj = signpost(); break;
      default: obj = new T.Group();
    }
    obj.position.set(it.x, it.y||0, it.z);
    // Stall keepers stand behind their counter; the interaction point stays where customers queue.
    const off = z.scenery?.offsets?.[it.kind]; if (off) { obj.position.set(it.x + off[0], off[2] || it.y || 0, it.z + off[1]); if (off[3]) obj.scale.setScalar(off[3]); }
    obj.userData.baseY = it.y || 0;
    if (it.kind === 'npc_elder' && it.y) {
      // Seated in the big nest on the stage, facing the hall.
      obj.scale.setScalar(1.6); obj.userData.baseY = it.y - .1; obj.position.y = obj.userData.baseY;
      obj.userData.parts?.legs?.forEach(l => l.rotation.x = -1.2);
    }
    if (it.kind.startsWith('npc') || it.kind === 'pk' || it.kind === 'shop_sell') { obj.rotation.y = it.kind === 'npc_cow' ? -0.6 : 0; if (it.kind !== 'npc_bird') z.npcs.push(obj); }
    if (it.kind === 'bed') obj.userData.bed = true;
    if (it.kind !== 'gate') {
      const lab = label(`${it.emo ? it.emo + ' ' : ''}${it.label}`, it.kind === 'dig' ? { bg: 'rgba(92,58,31,.75)', scale: 0.62 } : {});
      lab.position.set(0, it.kind === 'dig' ? 1.25 : (it.kind === 'npc_elder' ? 1.95 : 2.35), 0);
      lab.visible = false; lab.scale.multiplyScalar(.62 / obj.scale.x); lab.material.depthTest = true; obj.userData.label = lab; obj.add(lab);
    }
    obj.userData.item = worldItem;
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
    if (wasCurrent) { this.current = z; this.active = [z]; }
  }

  // Nhà Hội theo ảnh "Sân họp gà mộc mạc": bàn họp trải bản vẽ, vòng ổ rơm, lán vật liệu, bảng hội.
  setGuildHall(data, hall, cosmetics) {
    if (this.zones.guildhall) this.scene.remove(this.zones.guildhall.group);
    const group = new T.Group();
    group.visible = false;
    const z = { group, data, npcs: [] };
    this.zones.guildhall = z;
    this.scene.add(group);
    // Hall, planning table, nests, banner and task board from the bo-2 builder; projects add on top.
    z.scenery = buildScenery('guildhall', group, data, cosmetics);
    const p = hall.projects || {};
    if (p.nha_hoi_2) { box(group, 6.4, 0.14, 2.8, 0xb8433a, 0, 2.6, -6.6); box(group, 3.6, 0.3, 1.4, 0x8a5a30, 0, 0.35, 5.6); }
    if (p.co_hoi) {
      const col = parseInt(String(hall.color || '#d64b3a').slice(1), 16);
      for (const sx of [-1, 1]) { box(group, 0.1, 3.4, 0.1, 0x5c3a1f, sx * 4.2, 1.7, -3.6); const f = new T.Mesh(new T.BoxGeometry(1.2, 0.8, 0.04), mat(col)); f.position.set(sx * 4.2 + sx * 0.62, 3.0, -3.6); group.add(f); }
    }
    if (p.den_hoi) for (const [x, zz] of [[-5, -2], [5, -2], [-5, 3.5], [5, 3.5]]) {
      box(group, 0.08, 2.0, 0.08, 0x5c3a1f, x, 1.0, zz);
      const l = new T.Mesh(new T.SphereGeometry(0.24, 10, 8), mat(0xe2452e, { emissive: 0x8a1f0a })); l.position.set(x, 2.15, zz); group.add(l);
    }
    const sign = label(`${hall.emblem || '🏰'} ${hall.name}`, { bg: 'rgba(61,43,26,.86)', scale: 0.9 });
    sign.position.set(0, 3.2, -0.6); group.add(sign);
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
    // Bunting across the hall over the nest rows, lantern posts along the central aisle.
    for (const z of [-12.4, -3.4, 5.4]) {
      const rope = new T.Mesh(new T.BoxGeometry(32, 0.03, 0.03), mat(0x5c3a1f)); rope.position.set(0, 3.4, z); g.add(rope);
      for (let i = 0; i < 18; i++) {
        const f = new T.Mesh(new T.ConeGeometry(0.22, 0.4, 3), mat(colors[i % 4])); f.rotation.x = Math.PI; f.position.set(-15.3 + i * 1.8, 3.15, z); g.add(f);
      }
    }
    for (const [x, z] of [[-1.9, -12.4], [1.9, -12.4], [-1.9, -3.4], [1.9, -3.4], [-1.9, 5.4], [1.9, 5.4]]) {
      const post = new T.Mesh(new T.BoxGeometry(0.1, 2.2, 0.1), mat(0x5c3a1f)); post.position.set(x, 1.1, z); g.add(post);
      const l = new T.Mesh(new T.SphereGeometry(0.28, 10, 8), mat(0xe2452e, { emissive: 0x8a1f0a })); l.scale.y = 1.25; l.position.set(x, 2.35, z); g.add(l);
      const cap = new T.Mesh(new T.CylinderGeometry(0.12, 0.16, 0.08, 8), mat(0xe7c23a)); cap.position.set(x, 2.72, z); g.add(cap);
    }
    lobby.group.add(g);
    lobby.festival = g;
  }

  // Region rooms draw every area of the seamless map; homes and the guild hall draw only themselves.
  show(id, pos, ry) {
    const ids = this.regionIds.includes(id) ? this.regionIds : [id];
    for (const k in this.zones) this.zones[k].group.visible = ids.includes(k);
    this.active = ids.map((k) => this.zones[k]).filter(Boolean);
    this.current = this.zones[id];
    const obstacles = this.worldInfo && ids.length > 1 ? this.worldInfo : this.current.data;
    this.controls.enter(obstacles, this.profileFor(id), pos, ry);
    this.setRoute([]);
  }
  // Camera presets authored in area space (the hall's view from the door) mapped onto the map.
  profileFor(id) {
    const z = this.zones[id], p = z?.scenery?.camera;
    if (!p?.focus) return p ? { pitch: p.pitch, radius: p.radius } : null;
    const [lx, ly, lz] = p.focus, f = z.data.flip ? -1 : 1;
    return { ...p, focus: [(z.data.cx || 0) + lx * f, ly, (z.data.cz || 0) + lz * f], yaw: (p.yaw || 0) + (z.data.flip ? Math.PI : 0) };
  }
  // Walking into an area swaps the camera presets (the ◫ hall view only exists inside the hall).
  setArea(id) { const p = this.profileFor(id); this.controls.profile = p?.focus ? p : { ...(p || {}), pitch: p?.pitch ?? .56, radius: p?.radius ?? 14 }; }

  get camYaw(){return this.controls.yaw;}
  set camYaw(v){if(this.controls)this.controls.yaw=v;}
  get camTargetYaw(){return this.controls.targetYaw;}
  set camTargetYaw(v){if(this.controls)this.controls.targetYaw=v;}
  get camDist(){return this.controls.targetRadius;}
  set camDist(v){if(this.controls)this.controls.targetRadius=v;}
  setRoute(points){
    while(this.routeGroup.children.length){const o=this.routeGroup.children.pop();o.parent=null;}
    if(!points||points.length<1)return;
    this.routeMat ||= new T.MeshBasicMaterial({color:0xffde82,transparent:true,opacity:.8,depthWrite:false});
    this.routeGeo ||= new T.CircleGeometry(.17,10);
    for(let i=1;i<points.length;i++){const a=points[i-1],b=points[i],d=Math.hypot(a.x-b.x,a.z-b.z);for(let t=0;t<d;t+=1.1){const m=new T.Mesh(this.routeGeo,this.routeMat);m.rotation.x=-Math.PI/2;m.position.set(a.x+(b.x-a.x)*t/d,.045,a.z+(b.z-a.z)*t/d);this.routeGroup.add(m);}}
  }
  // Ground arrow orbiting the chicken, pointing at the next waypoint of the active route.
  setGuide(g) {
    if (!this.guide) {
      const shape = new T.Shape(); shape.moveTo(0, .55); shape.lineTo(.36, -.12); shape.lineTo(0, .04); shape.lineTo(-.36, -.12); shape.closePath();
      const m = new T.Mesh(new T.ShapeGeometry(shape), new T.MeshBasicMaterial({ color: 0xffc35a, transparent: true, opacity: .95, depthWrite: false }));
      m.rotation.x = -Math.PI / 2; m.renderOrder = 5;
      this.guide = new T.Group(); this.guide.add(m); this.scene.add(this.guide);
    }
    this.guide.visible = !!g;
    if (!g) return;
    const dx = g.to.x - g.from.x, dz = g.to.z - g.from.z, l = Math.hypot(dx, dz) || 1, pulse = 1 + Math.sin(performance.now() * .006) * .08;
    this.guide.position.set(g.from.x + dx / l * 1.35, .07, g.from.z + dz / l * 1.35);
    this.guide.rotation.y = Math.atan2(dx, dz) + Math.PI; this.guide.scale.setScalar(pulse);
  }
  markDestination(p){
    if(!this.marker){this.marker=new T.Mesh(new T.RingGeometry(.3,.4,32),new T.MeshBasicMaterial({color:0xffe9a2,transparent:true,opacity:.9,side:T.DoubleSide}));this.marker.rotation.x=-Math.PI/2;this.scene.add(this.marker);}
    this.marker.position.set(p.x,.06,p.z);this.marker.visible=true;this.markerUntil=performance.now()+1800;
  }
  // Architecture between the camera and the chicken (or the social focus) turns translucent.
  fadeOccluders(dt, target, z) {
    const list = z.scenery?.occluders;
    if (!list?.length) return;
    const cam = this.camera.position, ray = this._ray ||= new T.Ray(), hit = this._hit ||= new T.Vector3();
    const goals = [this.controls.focus, this._chest ||= new T.Vector3()];
    goals[1].set(target.x, 1, target.z);
    for (const o of list) {
      if (!o.userData.box) { o.updateMatrixWorld(true); o.userData.box = new T.Box3().setFromObject(o); o.userData.fade = 1; }
      let blocked = false;
      if (o.userData.roof) {
        // Mái che: chỉ mờ khi đoạn camera → gà cắt qua mặt mái; đứng dưới mái vẫn thấy trần thật.
        blocked = roofBlocks(o, cam, goals[1]);
      } else {
        for (const gp of goals) {
          const d = cam.distanceTo(gp); ray.origin.copy(cam); ray.direction.copy(gp).sub(cam).normalize();
          if (ray.intersectBox(o.userData.box, hit) && hit.distanceTo(cam) < d - .4) { blocked = true; break; }
        }
      }
      const goal = blocked ? (o.userData.fadeHide ? 0 : .16) : 1, prev = o.userData.fade;
      o.userData.fade += (goal - prev) * Math.min(1, dt * 7);
      if (Math.abs(o.userData.fade - prev) < .002 && Math.abs(o.userData.fade - goal) < .01) continue;
      const f = o.userData.fade;
      o.visible = f > .02;
      o.traverse(m => { if (!m.isMesh) return; m.userData.originalShadow ??= m.castShadow; m.castShadow = m.userData.originalShadow && f > .5; for (const q of Array.isArray(m.material) ? m.material : [m.material]) { q.userData.blend ??= q.transparent; q.transparent = q.userData.blend || f < .985; q.opacity = f * (q.userData.baseOpacity ??= q.opacity); q.depthWrite = !q.userData.blend && f >= .985; } });
    }
  }
  // ---------- Ngày/đêm và mưa (PLAN_LOGIC_GAME 16) ----------
  // Máy chủ báo pha của ngày game và thời tiết; client tự chạy đồng hồ giữa hai lần báo rồi nội suy ánh sáng.
  setClock(c) {
    if (!c) return;
    this.clock = { phase: c.phase, at: performance.now(), day: c.day || 2880, rain: !!c.rain };
  }
  phase() {
    const c = this.clock;
    return c ? (c.phase + (performance.now() - c.at) / 1000 / c.day) % 1 : 0.18;
  }
  updateSky(dt, target) {
    const s = Math.sin(this.phase() * Math.PI * 2);
    const light = smooth(-0.15, 0.3, s), dusk = Math.max(0, 1 - Math.abs(s) / 0.3) * 0.65;
    this.rainK = (this.rainK || 0) + ((this.clock?.rain ? 1 : 0) - (this.rainK || 0)) * Math.min(1, dt * 0.6);
    const rain = this.rainK;
    const key = Math.round(light * 200) + ':' + Math.round(dusk * 100) + ':' + Math.round(rain * 100);
    if (key !== this.skyKey) {
      this.skyKey = key;
      const stops = SKY.day.map((d, i) => {
        let c = mix(SKY.night[i], d, light);
        c = mix(c, SKY.dusk[i], dusk);
        return mix(c, mix(SKY.rainNight[i], SKY.rain[i], light), rain * 0.75);
      });
      paintSky(this.scene.background, stops);
      this.scene.fog.color.setRGB(stops[2][0], stops[2][1], stops[2][2]);
      this.scene.fog.density = .0115 + rain * .009;
      this.sun.intensity = (0.16 + 0.84 * light) * (1 - 0.62 * rain);
      this.sun.color.setHex(0x9fb4ff).lerp(new T.Color(dusk > 0.2 && light > 0.3 ? 0xffa86a : 0xffd9a6), light); // trăng xanh nhạt → nắng
      this.hemi.intensity = (0.24 + 0.18 * light) * (1 - 0.25 * rain);
      this.hemi.color.setHex(0x4d5d8f).lerp(new T.Color(0xe2e6f0), light);
      this.amb.intensity = 0.21 - 0.09 * light;
      this.amb.color.setHex(0x8494c8).lerp(new T.Color(0xffeed7), light);
      this.renderer.toneMappingExposure = .66 + (1 - light) * .2;
      for (const [m, day, night] of this.horizon.userData.mats) m.color.setHex(night).lerp(new T.Color(day), light);
      setNightGlow(1 - light);
    }
    this.updateRain(dt, target, rain);
  }
  // Mưa nhẹ: các vệt nước rơi quanh gà, đi theo người chơi; ẩn hẳn khi tạnh.
  updateRain(dt, target, k) {
    if (k < 0.02) { if (this.rain) this.rain.visible = false; return; }
    if (!this.rain) {
      const n = this.lowQuality ? 500 : 1400, pos = new Float32Array(n * 6);
      const geo = new T.BufferGeometry(); geo.setAttribute('position', new T.BufferAttribute(pos, 3));
      this.rain = new T.LineSegments(geo, new T.LineBasicMaterial({ color: 0xdde8f2, transparent: true, opacity: 0, depthWrite: false }));
      this.rain.frustumCulled = false; this.rain.userData.drops = Array.from({ length: n }, () => [Math.random() * 50 - 25, Math.random() * 22, Math.random() * 50 - 25]);
      this.scene.add(this.rain);
    }
    const r = this.rain, P = r.geometry.attributes.position.array, fall = (this.reducedMotion ? 9 : 17) * dt;
    r.visible = true; r.material.opacity = 0.75 * k; r.position.set(target.x, 0, target.z);
    r.userData.drops.forEach((d, i) => {
      d[1] -= fall; d[0] += fall * 0.12;
      if (d[1] < 0) { d[0] = Math.random() * 50 - 25; d[1] = 18 + Math.random() * 4; d[2] = Math.random() * 50 - 25; }
      P.set([d[0], d[1], d[2], d[0] - 0.1, d[1] + 0.85, d[2]], i * 6);
    });
    r.geometry.attributes.position.needsUpdate = true;
  }
  update(dt,target,moving=false,vehicle=null){
    this.controls.update(dt,target,moving,this.reducedMotion,vehicle);
    this.updateSky(dt, target);
    const t=performance.now()*.001;
    this.horizon.position.set(target.x, 0, target.z);
    this.sun.position.set(target.x - 18, 38, target.z + 20); this.sun.target.position.set(target.x, 0, target.z); this.sun.target.updateMatrixWorld();
    for (const z of this.active) {
      // Far areas are hidden behind the fog; only nearby ones animate.
      let near = true;
      if (this.active.length > 1 && !z.data.outdoor) {
        const d = Math.hypot(Math.max(0, Math.abs(target.x - z.data.cx) - z.data.halfX), Math.max(0, Math.abs(target.z - z.data.cz) - z.data.halfZ));
        z.group.visible = d < 90; near = d < 45;
        if (!z.group.visible) continue;
      }
      if (!near) continue;
      if (!this.reducedMotion) z.npcs.forEach((n, i) => { n.position.y = (n.userData.baseY || 0) + Math.sin(t * 1.5 + i) * .018; });
      if (z.scenery) { z.scenery.reducedMotion = this.reducedMotion; updateScenery(z.scenery, dt, t, (this.actors || [target]).map((q) => toLocal(z.data, q.x, q.z))); }
      for (const child of z.group.children) { const it = child.userData.item, lab = child.userData.label; if (!lab || !it) continue; lab.visible = Math.hypot(it.x - target.x, it.z - target.z) < 5; if (lab.visible) fitLabel(lab, this.camera); }
      this.fadeOccluders(dt, target, z);
      fadeTrees(z.scenery, z.group, this.camera.position, target);
    }
    if(this.marker&&performance.now()>this.markerUntil)this.marker.visible=false;
    this.renderer.render(this.scene,this.camera);
  }

}

// Area copy in its own frame: centre (0,0), entrance on +z. The outdoor area also gets the other
// areas' footprints and entries so it can lay the roads between them.
function localData(data, zones, world) {
  const L = (p) => ({ ...p, ...toLocal(data, p.x, p.z) });
  const out = { ...data, items: (data.items || []).map(L), spawns: (data.spawns || []).map(L), obstacles: (data.obstacles || []).map(L), entry: data.entry ? L(data.entry) : null };
  if (data.outdoor) out.areas = Object.values(zones).filter((a) => !a.outdoor && a.id !== 'home' && a.id !== 'guildhall').map((a) => ({ id: a.id, title: a.title, cx: a.cx - (data.cx || 0), cz: a.cz - (data.cz || 0), halfX: a.halfX, halfZ: a.halfZ, entry: a.entry && { x: a.entry.x - (data.cx || 0), z: a.entry.z - (data.cz || 0) } })), out.hub = world?.hub || { x: 0, z: 0 }, out.roads = world?.roads || null, out.belt = world?.belt || 0;
  return out;
}

// Name plates keep a readable on-screen size (≈14 px text) instead of growing as the camera nears.
const _lp = new T.Vector3();
export function fitLabel(sp, camera) {
  const aspect = sp.userData.aspect || (sp.scale.x / sp.scale.y);
  sp.userData.aspect = aspect;
  let parentScale = 1;
  for (let o = sp.parent; o; o = o.parent) parentScale *= o.scale.y;
  const d = sp.getWorldPosition(_lp).distanceTo(camera.position);
  const h = Math.min(1.1, Math.max(.2, .021 * d * 900 / window.innerHeight)) / parentScale;
  sp.scale.set(aspect * h, h, 1);
}

// A roof slope blocks the view when the camera→chicken segment passes from one side of its surface
// to the other inside the slope's footprint (camera above the thatch, or outside the eaves looking in).
const _inv = new T.Matrix4(), _p = new T.Vector3(), _a = new T.Vector3(), _b = new T.Vector3();
function roofBlocks(o, cam, target) {
  const { hx, hz, eave, ridge, side } = o.userData.roof;
  _inv.copy(o.matrixWorld).invert(); _a.copy(cam).applyMatrix4(_inv); _b.copy(target).applyMatrix4(_inv);
  const above = (q) => q.y - (ridge - (ridge - eave) * Math.abs(q.x) / hx);
  const below = above(_b) < 0;
  for (let i = 0; i <= 16; i++) {
    _p.lerpVectors(_a, _b, i / 16);
    if (_p.x * side < -.01 || Math.abs(_p.x) > hx || Math.abs(_p.z) > hz) continue;
    if ((above(_p) > 0) === below) return true;
  }
  return false;
}

// ---------- Trời và chân trời ----------
// Gradient trời xanh → mù sáng ở chân trời như bộ ảnh 2 (vẽ theo màn hình, không tốn hình học).
// Bốn điểm màu trời (đỉnh → chân trời) cho trưa, đêm, hoàng hôn và mưa; điểm thứ ba cũng là màu sương.
const hex = (h) => [(h >> 16 & 255) / 255, (h >> 8 & 255) / 255, (h & 255) / 255];
const SKY = {
  day: [0x6f9cc4, 0xa9c6d8, 0xcbd8d4, 0xd9dccd].map(hex),
  night: [0x0b1430, 0x1a2747, 0x2a3752, 0x343f58].map(hex),
  dusk: [0x4a5f93, 0xc78c74, 0xe8a979, 0xf0c591].map(hex),
  rain: [0x7a8794, 0x9aa5ad, 0xaab3b6, 0xb5bcbc].map(hex),
  rainNight: [0x141b2a, 0x222b3b, 0x2e3746, 0x363e4b].map(hex),
};
const mix = (a, b, t) => a.map((v, i) => v + (b[i] - v) * t);
const smooth = (a, b, x) => { const t = Math.min(1, Math.max(0, (x - a) / (b - a))); return t * t * (3 - 2 * t); };
function paintSky(tex, stops) {
  const cv = tex.image, c = cv.getContext('2d'), g = c.createLinearGradient(0, 0, 0, 256);
  const css = (v) => `rgb(${v.map((x) => Math.round(x * 255)).join(',')})`;
  [0, .42, .72, 1].forEach((at, i) => g.addColorStop(at, css(stops[i])));
  c.fillStyle = g; c.fillRect(0, 0, 4, 256);
  tex.needsUpdate = true;
}
function skyTexture() {
  const cv = document.createElement('canvas'); cv.width = 4; cv.height = 256;
  const tex = new T.CanvasTexture(cv);
  paintSky(tex, SKY.day);
  return tex;
}
function mountains() {
  const g = new T.Group(), rng = (() => { let n = 7; return () => (n = (n * 16807) % 2147483647) / 2147483647; })();
  // Hai lớp: dãy xa xanh xám, đồi gần xanh rêu; không nhận sương để vẫn hiện rõ ở chân trời.
  g.userData.mats = [];
  for (const [r, col, hMin, hMax, n, night] of [[285, 0x9db3bd, 30, 60, 26, 0x252f45], [245, 0x7f977c, 16, 32, 32, 0x1f2a26]]) {
    const m = new T.MeshStandardMaterial({ color: col, fog: false, flatShading: true, roughness: 1 });
    g.userData.mats.push([m, col, night]);
    for (let i = 0; i < n; i++) {
      const a = (i + rng() * .6) / n * Math.PI * 2, h = hMin + rng() * (hMax - hMin), w = h * (1.3 + rng() * .9);
      const peak = new T.Mesh(new T.ConeGeometry(w, h, 5 + Math.floor(rng() * 3), 1), m);
      peak.position.set(Math.cos(a) * r, h / 2 - 4, Math.sin(a) * r); peak.rotation.y = rng() * 6; peak.scale.z = .6 + rng() * .5;
      g.add(peak);
    }
  }
  return g;
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

function noticeBoard() {
  const g = new T.Group();
  for (const sx of [-1, 1]) box(g, 0.16, 2.2, 0.16, 0x5c3a1f, sx * 0.9, 1.1, 0);
  box(g, 2.1, 1.2, 0.1, 0x9c6b36, 0, 1.45, 0);
  box(g, 2.3, 0.12, 0.3, 0x6b4423, 0, 2.15, 0);
  [[-0.55, 1.6, 0xf4ecd4], [0.1, 1.55, 0xe8d9b0], [0.6, 1.3, 0xf4ecd4], [-0.3, 1.2, 0xdfe8d4]].forEach(([x, y, c]) => { const p = box(g, 0.48, 0.38, 0.02, c, x, y, 0.06); p.rotation.z = (x * 7) % 0.2 - 0.1; });
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
  const dirt = new T.Mesh(new T.SphereGeometry(0.9, 16, 10, 0, Math.PI * 2, 0, Math.PI / 2), mat(hex));
  dirt.scale.y = 0.42; dirt.castShadow = true; dirt.receiveShadow = true; g.add(dirt);
  const hole = new T.Mesh(new T.CircleGeometry(0.32, 14), mat(0x2e1f12));
  hole.rotation.x = -Math.PI / 2; hole.position.y = 0.385; g.add(hole);
  if (soil === 'Gốc cây') { const st = new T.Mesh(new T.CylinderGeometry(0.32, 0.4, 0.5, 10), mat(0x7a5531)); st.position.set(0.55, 0.25, -0.35); st.castShadow = true; g.add(st); }
  for (let i = 0; i < 3; i++) { const w = new T.Mesh(new T.TorusGeometry(0.09, 0.03, 6, 10, Math.PI), mat(0xd98a8a)); w.position.set(Math.cos(i * 2.1) * 0.7, 0.12, Math.sin(i * 2.1) * 0.7); w.rotation.y = i; g.add(w); }
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

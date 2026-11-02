// Bản đồ của trại: minimap cuộn theo gà, bản đồ lớn vẽ đúng địa hình thật và chỉ đường đi bộ theo đường làng.
// Bản đồ chỉ đặt chỉ đường; không bao giờ dịch chuyển nhân vật.
import {G,$,esc,openSheet,closeSheet,toast} from './ui.js';
import {findPath,walkable,areaAt} from './navigation.js';
const icons={lobby:'⌂',village:'✦',vs_field:'✦',garden:'〰',market:'▤',arena:'⚑',hatchery:'◉',neighborhood:'⌂',park:'♧',station:'▣',garage:'▰',vs_station:'▣',vensuoi:'≈',home:'⌂',guildhall:'⚑'};
const descriptions={lobby:'Ổ rơm ấm, những người bạn và Cụ Cục trên bục.',village:'Quảng trường cây đa, nơi mọi con đường gặp nhau.',vs_field:'Con đường đất nối bến xe với đồng ven suối.',garden:'Đất ẩm, giun quý và những điều bất ngờ.',market:'Thức ăn, thời trang và sạp thu mua giun.',arena:'Hẹn bạn thi đấu, cùng nhau ăn mừng.',hatchery:'Một quả trứng. Một người bạn mới.',neighborhood:'Về chuồng của mình, ghé nhà một người bạn, ghé Nhà Hội.',park:'Dạo quanh hồ, nghỉ chân dưới tán cây.',station:'Một chuyến xe, một vùng đất mới.',garage:'Chọn chiếc xe và rủ bạn đi cùng.',vensuoi:'Bãi bồi xanh và những loài giun đặc sản.',vs_station:'Cửa ngõ của trang trại Ven Suối.'};
const tints={lobby:'#c9a46d',garden:'#a98a5a',market:'#d3b47e',arena:'#c79a5f',hatchery:'#d6b78a',neighborhood:'#b9c98c',park:'#b4cf95',station:'#b8ab8f',garage:'#b4b0a6',vs_station:'#b8ab8f',vensuoi:'#9fbd86'};
const WALK=4.6;
let destination=null,lastDraw=0;

const areas=()=>Object.values(G.zones||{}).filter(z=>!z.outdoor&&z.id!=='home'&&z.id!=='guildhall');
const outdoorId=()=>G.worldMap?.outdoor;
const hub=()=>G.worldMap?.hub||{x:0,z:0};
// Point where a road meets the plaza; small hubs (Ven Suối) keep it halfway to the entry.
const ring=(e,r=9)=>{const h=hub(),l=Math.hypot(e.x-h.x,e.z-h.z)||1;r=Math.min(r,l*.5);return {x:h.x+(e.x-h.x)/l*r,z:h.z+(e.z-h.z)/l*r};};
function segDist(p,a,b){const dx=b.x-a.x,dz=b.z-a.z,t=Math.max(0,Math.min(1,((p.x-a.x)*dx+(p.z-a.z)*dz)/(dx*dx+dz*dz||1)));return {d:Math.hypot(p.x-a.x-dx*t,p.z-a.z-dz*t),x:a.x+dx*t,z:a.z+dz*t};}
// Nearest walkable spot to a point of interest (an NPC on the stage, a stall behind its counter…).
function reach(p){
 const w=G.worldMap||G.zones[G.zone];if(walkable(w,p.x,p.z))return p;
 for(let r=1;r<7;r+=.5)for(let a=0;a<Math.PI*2;a+=Math.PI/8){const q={x:p.x+Math.sin(a)*r,z:p.z+Math.cos(a)*r};if(walkable(w,q.x,q.z))return q;}
 return p;
}
// Where the route ends inside an area: its main NPC or stand, else where people arrive.
function poi(z){
 const it=(z.items||[]).find(i=>['npc_elder','dig','shop_food','npc_duck','bus_stop','home_sign','pk','npc_goat','npc_frog','board'].includes(i.kind));
 return reach(it?{x:it.x,z:it.z}:(z.spawns?.[0]||{x:z.cx,z:z.cz}));
}
// Waypoints along the village roads: out of the current area by its road, across the plaza, down the target road.
function stops(pos,targetId){
 const zones=G.zones,T=zones[targetId],here=G.zone,H=zones[here],out=[];
 if(!T)return out;
 if(here===targetId)return [poi(T)];
 const roads=areas().filter(a=>a.entry).map(a=>({a,r:ring(a.entry)}));
 const onRoad=(a)=>segDist(pos,ring(a.entry),a.entry).d<6;
 if(H&&!H.outdoor&&H.entry){out.push(H.entry);if(T.outdoor||T.entry)out.push(ring(H.entry));}
 else if(!(T.entry&&onRoad(T))&&roads.length){
  const near=roads.map(({a,r})=>({a,r,s:segDist(pos,r,a.entry)})).sort((x,y)=>x.s.d-y.s.d)[0];
  if(near.s.d>6)out.push({x:near.s.x,z:near.s.z});
  out.push(near.r);
 }
 if(T.outdoor){const h=hub();out.push({x:h.x,z:h.z+9});return out;}
 if(T.entry){if(!(onRoad(T)&&!(H&&!H.outdoor)))out.push(ring(T.entry));out.push(T.entry);}
 out.push(poi(T));
 return out;
}
function routeTo(targetId){
 const w=G.worldMap||G.zones[G.zone];let at={x:G.self.pos.x,z:G.self.pos.z};const pts=[];
 for(const st of stops(at,targetId)){const seg=findPath(w,at,st);pts.push(...(seg.length?seg:[st]));at=st;}
 return pts;
}
const lengthOf=(from,pts)=>{let d=0,a=from;for(const p of pts){d+=Math.hypot(p.x-a.x,p.z-a.z);a=p;}return d;};
export function etaSeconds(targetId){if(!G.self||!G.zones?.[targetId])return null;return Math.round(lengthOf(G.self.pos,routeTo(targetId))/WALK);}
const fmtEta=s=>s==null?'—':s<60?`≈ ${Math.max(5,Math.round(s/5)*5)} giây đi bộ`:`≈ ${Math.round(s/60)} phút đi bộ`;
const npcNames=z=>(z?.items||[]).filter(i=>i.kind.startsWith('npc')||i.kind==='pk'||i.kind==='shop_sell').map(i=>i.label.split(' · ')[0]).slice(0,3).join(', ');

export function setDestination(id,msg='Đã ghim điểm đến. Đi theo dấu chỉ đường.'){destination=id;G.routeTarget=id;closeSheet();toast(msg,'good');updateRoute(true);}
export function cancelRoute(){destination=null;G.routeTarget=null;G.routePoints=[];$('routeCard').hidden=true;$('routeArrow').hidden=true;G.world?.setRoute?.([]);G.world?.setGuide?.(null);}
export function currentDestination(){return destination;}
// Small guide pin over the destination while it is on screen; the ground arrow handles the rest.
const _v=new window.THREE.Vector3();
function pointArrow(point){
 const el=$('routeArrow');if(!point||G.modal){el.hidden=true;return;}
 _v.set(point.x,1.6,point.z).project(G.world.camera);
 const W=innerWidth,H=innerHeight,x=(_v.x*.5+.5)*W,y=(-_v.y*.5+.5)*H,m=56;
 const inside=_v.z<1&&x>m&&x<W-m&&y>m+60&&y<H-m-60;
 el.hidden=!inside;if(inside)el.style.transform=`translate(${x}px,${y}px)`;
}
export function updateRoute(force=false){
 const now=performance.now();if(!G.self||!G.zones)return;
 if(now-lastDraw>150||force){drawMini();lastDraw=now;}
 if(!destination)return;
 // Another farm: walk to this farm's bus station first; tickets are bought there, never from the map.
 let target=destination;
 if(!G.zones[target])target=G.region==='vensuoi'?'vs_station':'station';
 const T=G.zones[target];if(!T)return;
 if(force||!G.routeAt||now-G.routeAt>1000){
  G.routePoints=routeTo(target);G.routeAt=now;
  G.world?.setRoute?.(G.routePoints.length?[{x:G.self.pos.x,z:G.self.pos.z},...G.routePoints]:[]);
 }
 const pts=G.routePoints||[],end=pts[pts.length-1],left=end?lengthOf(G.self.pos,pts.filter(q=>Math.hypot(q.x-G.self.pos.x,q.z-G.self.pos.z)>1.6)):0;
 const arrived=G.zone===target&&left<4;
 $('routeCard').hidden=false;
 $('routeName').textContent=G.zones[destination]?.title||(destination==='vensuoi'?'Trang trại Ven Suối':'Trại Hypex');
 $('routeStep').textContent=arrived?(destination===target?'Bạn đã tới nơi':'Tới bến rồi, đặt chuyến ở Trạm xe buýt'):G.zone===target?'Đã vào khu, đi thêm vài bước':`Theo đường làng tới ${T.title}`;
 $('routeDistance').textContent=arrived?'':Math.ceil(left)+' m';
 pointArrow(arrived?null:end);
 const next=pts.find(q=>Math.hypot(q.x-G.self.pos.x,q.z-G.self.pos.z)>1.6);
 G.world.setGuide(!arrived&&next?{from:G.self.pos,to:next}:null);
}

// ---------- Bản đồ lớn: địa hình thật, đường làng, khu, vị trí của bạn ----------
function svgMap(selected){
 const w=G.worldMap,h=hub(),pad=6,x0=w.cx-w.halfX-pad,z0=w.cz-w.halfZ-pad,W=w.halfX*2+pad*2,D=w.halfZ*2+pad*2;
 const list=areas(),seed=(i)=>{const v=Math.sin(i*127.1+311.7)*43758.5453;return v-Math.floor(v);};
 const inArea=(x,z)=>list.some(a=>Math.abs(x-a.cx)<a.halfX+6&&Math.abs(z-a.cz)<a.halfZ+6);
 const trees=[];for(let i=0;i<220;i++){const x=x0+seed(i)*W,z=z0+seed(i+500)*D;if(inArea(x,z)||Math.hypot(x-h.x,z-h.z)<16)continue;trees.push(`<circle cx="${x.toFixed(1)}" cy="${z.toFixed(1)}" r="${(1.6+seed(i+900)*1.4).toFixed(1)}"/>`);}
 const rn=w.roads;
 // Đường làng do máy chủ dựng (vòng quảng trường, đường nhánh, vành đai) + đoạn ngắn từ cổng vào trong khu.
 const roads=(rn?.edges?rn.edges.map(([a,b])=>`<path d="M${rn.nodes[a].x} ${rn.nodes[a].z}L${rn.nodes[b].x} ${rn.nodes[b].z}"/>`).join(''):'')+list.filter(a=>a.entry).map(a=>{const r=ring(a.entry,10);return `<path d="M${rn?a.entry.x:r.x} ${rn?a.entry.z:r.z}L${a.entry.x} ${a.entry.z}L${a.entry.x+(a.cx-a.entry.x)*.3} ${a.entry.z+(a.cz-a.entry.z)*.3}"/>`;}).join('');
 const busStops=(rn?.stops||[]).map(st=>`<g class="map-stop"><circle cx="${st.pos.x}" cy="${st.pos.z}" r="2.4" fill="#2f6f9e" stroke="#fff" stroke-width=".7"/><text x="${st.pos.x}" y="${st.pos.z+1.1}" text-anchor="middle" font-size="2.6" fill="#fff">B</text></g>`).join('');
 const buses=[...(G.actors?.values()||[])].filter(a=>a.info.kind==='bus').map(a=>`<rect x="${(a.mesh.position.x-2.2).toFixed(1)}" y="${(a.mesh.position.z-1.4).toFixed(1)}" width="4.4" height="2.8" rx="1" fill="#6b7a3a" stroke="#fff" stroke-width=".6"/>`).join('');
 const water=list.flatMap(a=>(a.obstacles||[]).filter(o=>o.h<=.2).map(o=>`<rect x="${o.x-o.w/2}" y="${o.z-o.d/2}" width="${o.w}" height="${o.d}" rx="${Math.min(o.w,o.d)/2}"/>`)).join('');
 const walls=list.flatMap(a=>(a.obstacles||[]).filter(o=>o.h>=1.6).map(o=>`<rect x="${o.x-o.w/2}" y="${o.z-o.d/2}" width="${o.w}" height="${o.d}"/>`)).join('');
 const blocks=list.map(a=>`<g class="map-poi ${G.zone===a.id?'here':''} ${selected===a.id?'selected':''}" data-zone="${esc(a.id)}" tabindex="0" role="button" aria-label="${esc(a.title)}"><rect class="area" x="${a.cx-a.halfX}" y="${a.cz-a.halfZ}" width="${a.halfX*2}" height="${a.halfZ*2}" rx="4" fill="${tints[a.id]||'#cdb98d'}"/><circle class="poi-disc" cx="${a.cx}" cy="${a.cz}" r="5"/><text x="${a.cx}" y="${a.cz+1.9}" text-anchor="middle" font-size="5.4" fill="#34513f">${icons[a.id]||'⌂'}</text><rect x="${a.cx-15}" y="${a.cz+a.halfZ+1.5}" width="30" height="6.4" rx="3" fill="#fffbed" opacity=".95"/><text x="${a.cx}" y="${a.cz+a.halfZ+6}" text-anchor="middle" font-size="3.6" fill="#344e3e">${esc(a.title)}</text></g>`).join('');
 const me=G.self.pos,ry=G.self.mesh.rotation.y;
 const others=[...G.players.values()].map(p=>`<circle cx="${p.mesh.position.x.toFixed(1)}" cy="${p.mesh.position.z.toFixed(1)}" r="1.4"/>`).join('');
 const route=(G.routePoints||[]).length?`<polyline class="map-route" points="${[me,...G.routePoints].map(p=>`${p.x.toFixed(1)},${p.z.toFixed(1)}`).join(' ')}"/>`:'';
 const pr=Math.min(10,...list.filter(a=>a.entry).map(a=>Math.hypot(a.entry.x-h.x,a.entry.z-h.z)*.6));
 const plaza=outdoorId()?`<g class="map-poi ${G.zone===outdoorId()?'here':''} ${selected===outdoorId()?'selected':''}" data-zone="${esc(outdoorId())}" tabindex="0" role="button" aria-label="Quảng trường"><circle cx="${h.x}" cy="${h.z}" r="${pr.toFixed(1)}" fill="#e3cf9f" stroke="#c9b17c"/><circle cx="${h.x}" cy="${h.z}" r="3.6" fill="#7d9b5c"/><text x="${h.x}" y="${h.z-12}" text-anchor="middle" font-size="3.6" fill="#344e3e">Quảng trường</text></g>`:'';
 return `<svg class="village-map" viewBox="${x0} ${z0} ${W} ${D}" role="img" aria-label="Bản đồ ${esc(w.title||'trại')}"><rect x="${x0}" y="${z0}" width="${W}" height="${D}" fill="#f2e8cd"/><g fill="#9fbb80" opacity=".55">${trees}</g><g fill="#9fcfd0">${water}</g><g fill="none" stroke="#c7b07e" stroke-width="5.4" stroke-linecap="round" stroke-linejoin="round">${roads}</g><g fill="none" stroke="#efe2bd" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">${roads}</g>${plaza}${blocks}${busStops}${buses}<g fill="#6a4b2e">${walls}</g><g fill="#5f8fb0">${others}</g>${route}<g transform="translate(${me.x.toFixed(1)} ${me.z.toFixed(1)}) rotate(${(-ry*180/Math.PI+180).toFixed(0)})"><circle r="4.6" fill="#ed784733"/><path d="M0 -3.4L2.6 2.6L0 1.2L-2.6 2.6Z" fill="#ed7847" stroke="#fff" stroke-width=".8"/></g><g transform="translate(${me.x.toFixed(1)} ${me.z.toFixed(1)})"><rect x="-6" y="5" width="12" height="5" rx="2.5" fill="#ed7847"/><text y="8.7" text-anchor="middle" font-size="3.4" font-weight="700" fill="#fff">BẠN</text></g><text x="${x0+W-8}" y="${z0+9}" font-size="5" fill="#69856c" text-anchor="end">B ↑</text></svg>`;
}
export function openMap(layer='village'){
 if(!G.me)return;G.walkPath=[];
 const zones=G.zones||{};
 if(!G.worldMap)layer='regions'; // trong nhà riêng / Nhà Hội chỉ xem được các trang trại
 let selected=zones[destination]?destination:(zones[G.zone]&&G.zone!=='home'&&G.zone!=='guildhall'?G.zone:outdoorId());
 const render=()=>{
  const z=zones[selected];
  const eta=selected===G.zone?'Bạn đang ở đây':fmtEta(etaSeconds(selected));
  openSheet('map','Khám phá thế giới',`<div class="map-tabs">${G.worldMap?`<button id="mapVillage" class="${layer==='village'?'active':''}">Bản đồ ${esc(G.worldMap.title||'trại')}</button>`:''}<button id="mapRegions" class="${layer==='regions'?'active':''}">Các trang trại</button><span>ĐI BỘ ĐẾN MỌI NƠI · ĐƯỜNG LÀNG NỐI CÁC KHU</span></div>${layer==='village'?`<div class="map-layout">${svgMap(selected)}<aside class="map-detail"><span class="eyebrow">ĐIỂM ĐẾN CỦA BẠN</span><div class="map-icon">${icons[selected]||'⌂'}</div><h2>${esc(z?.title||'Khu vực')}</h2><p>${esc(descriptions[selected]||'Một góc nhỏ để gặp gỡ và khám phá.')}</p><div class="map-facts"><span>Gặp</span><b>${esc(npcNames(z)||'Bạn bè trong làng')}</b><span>Hành trình</span><b>${eta}</b><span>Đến bằng</span><b>Đi bộ${['station','garage','market','neighborhood','park',outdoorId()].includes(selected)?' hoặc xe riêng':''}</b></div>${G.mapPick?`<button class="btn block hot" id="mapPick" ${selected===G.zone||selected===outdoorId()?'disabled':''}>${esc(G.mapPick.label(selected))}</button><small>Chọn khu trên bản đồ rồi bấm để ${esc(G.mapPick.verb||'đi tới đó')}.</small>`:`<button class="btn block" id="mapPin" ${selected===G.zone?'disabled':''}>Đặt chỉ đường ↗</button>${G.zone===outdoorId()&&!G.myCarrier?'<button class="btn block sec" id="mapXeOm" style="margin-top:6px">🛵 Gọi xe ôm (1–2 thóc)</button>':''}<small>Bản đồ chỉ đường. Hãy tự đi bộ tới nơi, đi xe buýt ở bến 🅑 hoặc gọi xe ôm.</small>`}</aside></div>`:`<div class="region-grid"><button data-region="hypex" class="region-card"><span class="region-art home-art">⌂</span><small>NGÔI LÀNG CỦA BẠN</small><h2>Trại Hypex</h2><p>Chuồng cộng đồng · Chợ · Vườn giun</p><b>${G.region==='hypex'?'Bạn đang ở trại này':'Chỉ đường tới bến về Hypex ↗'}</b></button><button data-region="vensuoi" class="region-card"><span class="region-art stream-art">≈</span><small>VÙNG ĐẤT BÊN DÒNG NƯỚC</small><h2>Trang trại Ven Suối</h2><p>Cấp 3 · Giun đặc sản · Những chuyến đi</p><b>${G.region==='vensuoi'?'Bạn đang ở trại này':'Chỉ đường tới bến xe ↗'}</b></button></div><p class="map-note">Đi giữa các trang trại bằng xe buýt ở bến hoặc xe riêng trên đường liên trại. Chọn trên bản đồ chưa trừ thóc và không dịch chuyển nhân vật.</p>`}`,{close:()=>{document.body.classList.remove('map-open');if(G.mapPick){const p=G.mapPick;G.mapPick=null;p.cancel?.();}}});
  document.body.classList.add('map-open');
  if($('mapVillage'))$('mapVillage').onclick=()=>{layer='village';render();};$('mapRegions').onclick=()=>{layer='regions';render();};
  document.querySelectorAll('.map-poi').forEach(el=>{const choose=()=>{selected=el.dataset.zone;render();};el.onclick=choose;el.onkeydown=e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();choose();}};});
  $('mapPin')?.addEventListener('click',()=>setDestination(selected));
  $('mapPick')?.addEventListener('click',()=>{const pick=G.mapPick;G.mapPick=null;closeSheet();pick.fn(selected);});
  $('mapXeOm')?.addEventListener('click',async()=>{closeSheet();const r=await G.net.act('xeom_call');toast(r.ok?r.msg:r.err,r.ok?'good':'warn');});
  document.querySelectorAll('[data-region]').forEach(el=>el.onclick=()=>{if(el.dataset.region===G.region){toast('Bạn đang ở trang trại này.');return;}if(!G.worldMap){toast('Ra khỏi chuồng rồi mở bản đồ để chỉ đường tới bến.','warn');return;}setDestination(el.dataset.region,'Đi bộ tới bến xe để lên chuyến.');});
 };render();
}

// ---------- Minimap: 80 m quanh gà, hướng bắc lên trên ----------
function drawMini(){
 const c=$('minimapCanvas');if(!c||!G.self)return;const ctx=c.getContext('2d');
 const W=c.width,H=c.height,me=G.self.pos,span=G.worldMap?80:Math.max(G.zones[G.zone]?.halfX||10,10)*2.4,s=W/span;
 const pt=(x,z)=>[W/2+(x-me.x)*s,H/2+(z-me.z)*s];
 ctx.fillStyle='#f2e8cd';ctx.fillRect(0,0,W,H);
 const list=G.worldMap?areas():[G.zones[G.zone]].filter(Boolean);
 if(G.worldMap){
  const h=hub();ctx.lineCap='round';
  const rn=G.worldMap.roads;
  for(const [col,wd] of [['#c7b07e',5.4],['#efe2bd',2.2]]){ctx.strokeStyle=col;ctx.lineWidth=wd*s;
   if(rn?.edges)for(const [a,b] of rn.edges){ctx.beginPath();ctx.moveTo(...pt(rn.nodes[a].x,rn.nodes[a].z));ctx.lineTo(...pt(rn.nodes[b].x,rn.nodes[b].z));ctx.stroke();}
   for(const a of list)if(a.entry){const r=rn?a.entry:ring(a.entry,10);ctx.beginPath();ctx.moveTo(...pt(r.x,r.z));ctx.lineTo(...pt(a.entry.x,a.entry.z));ctx.lineTo(...pt(a.entry.x+(a.cx-a.entry.x)*.3,a.entry.z+(a.cz-a.entry.z)*.3));ctx.stroke();}}
  for(const st of rn?.stops||[]){ctx.fillStyle='#2f6f9e';ctx.beginPath();ctx.arc(...pt(st.pos.x,st.pos.z),3,0,Math.PI*2);ctx.fill();}
  ctx.fillStyle='#e3cf9f';ctx.beginPath();ctx.arc(...pt(h.x,h.z),10*s,0,Math.PI*2);ctx.fill();
 }
 for(const a of list){const cx=a.cx||0,cz=a.cz||0;ctx.fillStyle=tints[a.id]||'#d8c398';ctx.fillRect(...pt(cx-a.halfX,cz-a.halfZ),a.halfX*2*s,a.halfZ*2*s);}
 for(const o of (G.worldMap||G.zones[G.zone])?.obstacles||[]){ctx.fillStyle=o.h<=.2?'#9fcfd0':o.h>=1.6?'#6a4b2e':'#8d9c6d';ctx.fillRect(...pt(o.x-o.w/2,o.z-o.d/2),Math.max(1,o.w*s),Math.max(1,o.d*s));}
 if(G.routePoints?.length){ctx.strokeStyle='#e8a64f';ctx.lineWidth=3;ctx.setLineDash([4,4]);ctx.beginPath();ctx.moveTo(...pt(me.x,me.z));for(const q of G.routePoints)ctx.lineTo(...pt(q.x,q.z));ctx.stroke();ctx.setLineDash([]);const e=G.routePoints[G.routePoints.length-1];ctx.beginPath();ctx.arc(...pt(e.x,e.z),4,0,Math.PI*2);ctx.fillStyle='#e8a64f';ctx.fill();}
 ctx.fillStyle='#5f8fb0';for(const p of G.players.values()){ctx.beginPath();ctx.arc(...pt(p.mesh.position.x,p.mesh.position.z),2.4,0,Math.PI*2);ctx.fill();}
 for(const a of G.actors?.values()||[]){if(a.info.kind!=='bus'&&a.info.kind!=='xeom')continue;ctx.fillStyle=a.info.kind==='bus'?'#6b7a3a':'#c8642c';const [x,y]=pt(a.mesh.position.x,a.mesh.position.z);ctx.fillRect(x-(a.info.kind==='bus'?4:2),y-(a.info.kind==='bus'?3:2),a.info.kind==='bus'?8:4,a.info.kind==='bus'?6:4);}
 ctx.save();ctx.translate(W/2,H/2);ctx.rotate(-G.self.mesh.rotation.y);ctx.beginPath();ctx.moveTo(0,6);ctx.lineTo(-4,-5);ctx.lineTo(0,-2);ctx.lineTo(4,-5);ctx.closePath();ctx.fillStyle='#ed7847';ctx.strokeStyle='#fff';ctx.lineWidth=1.5;ctx.fill();ctx.stroke();ctx.restore();
 $('miniLocation').textContent=G.zones[G.zone]?.title||'';
 const lz=G.world?.zones?.lobby,npc=lz?(lz.scenery?.ambient?.length||0)+lz.npcs.length:0;
 $('localPopulation').textContent=`${G.players.size+1} người chơi trong kênh${G.zone==='lobby'&&npc?` · ${npc} NPC trong sảnh`:''}`;
}
// Chọn điểm đến trên bản đồ cho xe ôm (PLAN_LOGIC_GAME 8.2): label(id) là chữ trên nút, fn(id) khi chọn.
export function pickDestination(label,fn,cancel,verb){G.mapPick={label,fn,cancel,verb};openMap('village');}
export function initMap(){ $('btnMap').onclick=()=>openMap();$('miniMapButton').onclick=()=>openMap();$('cancelRoute').onclick=cancelRoute;G.navigate=setDestination;G.openMap=openMap;}
export {areaAt};

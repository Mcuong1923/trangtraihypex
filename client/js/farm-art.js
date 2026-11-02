// Shared art kit for the bo-2 references. Every asset is genuine geometry;
// canvas maps are repeatable materials, never a flattened scene/background.
const T = window.THREE;
const maps = new Map(), mats = new Map(), geometries = new Map();
const rngFor = seed => { let s = seed >>> 0; return () => ((s = Math.imul(s, 1664525) + 1013904223 >>> 0) / 4294967296); };
const clamp = (x,a,b) => Math.max(a,Math.min(b,x));
function canvasMap(kind) {
  if (maps.has(kind)) return maps.get(kind);
  const c = document.createElement('canvas'); c.width = c.height = kind === 'mesh' ? 256 : 1024;
  const ctx = c.getContext('2d'), n = c.width, r = rngFor(712 + kind.length * 317);
  if (kind === 'mesh') {
    // Sixteen complete diamond cells tile without seams. Alpha-test keeps it
    // depth-writing, unlike a transparent pane across the entire chicken coop.
    ctx.clearRect(0,0,n,n); ctx.lineWidth = 3.2; ctx.strokeStyle = '#a7a796';
    for (let x=-n;x<=n*2;x+=32) {
      ctx.beginPath();ctx.moveTo(x,0);ctx.lineTo(x+n,n);ctx.stroke();
      ctx.beginPath();ctx.moveTo(x,0);ctx.lineTo(x-n,n);ctx.stroke();
    }
    ctx.lineWidth=1;ctx.strokeStyle='#5d604f';
    for(let x=-n;x<=n*2;x+=32){ctx.beginPath();ctx.moveTo(x+1,0);ctx.lineTo(x+n+1,n);ctx.stroke();}
  } else if (kind === 'wood') {
    ctx.fillStyle='#a87b48';ctx.fillRect(0,0,n,n);
    for(let i=0;i<850;i++){
      const y=r()*n, wave=2+r()*8;
      ctx.strokeStyle=i%5===0?'rgba(238,198,128,.29)':'rgba(62,34,13,.15)';ctx.lineWidth=.6+r()*2.4;
      ctx.beginPath();ctx.moveTo(-10,y);ctx.bezierCurveTo(260,y+wave,700,y-wave,n+10,y);ctx.stroke();
    }
    for(let i=0;i<20;i++){
      const x=r()*n,y=r()*n;
      for(let k=0;k<5;k++){
        ctx.strokeStyle=`rgba(65,36,17,${.16-k*.019})`;ctx.lineWidth=1.1;
        ctx.beginPath();ctx.ellipse(x,y,8+k*8,1.5+k*1.4,0,0,Math.PI*2);ctx.stroke();
      }
    }
    for(let i=0;i<9000;i++){ctx.fillStyle=r()>.5?'#d1aa6b22':'#3e26101a';ctx.fillRect(r()*n,r()*n,.8+r()*4,.5+r()*1.2);}
  } else if(kind==='thatch'||kind==='straw') {
    ctx.fillStyle=kind==='thatch'?'#a67c32':'#c8a351';ctx.fillRect(0,0,n,n);
    for(let i=0;i<12500;i++){
      const x=r()*n,y=r()*n, len=kind==='thatch'?35+r()*180:15+r()*65;
      const a=kind==='thatch'?(r()-.5)*.15:r()*Math.PI*2;
      ctx.strokeStyle=['#e5bf69','#b48b36','#765a27','#d4aa50','#efcc7a'][i%5];
      ctx.globalAlpha=.35+r()*.6;ctx.lineWidth=.6+r()*2.6;
      ctx.beginPath();ctx.moveTo(x,y);ctx.lineTo(x+Math.cos(a)*len,y+Math.sin(a)*len);ctx.stroke();
    }
    ctx.globalAlpha=1;
    if(kind==='thatch')for(let x=0;x<n;x+=128){
      const grad=ctx.createLinearGradient(x,0,x+44,0);grad.addColorStop(0,'#38270f66');grad.addColorStop(1,'#38270f00');ctx.fillStyle=grad;ctx.fillRect(x,0,44,n);
    }
  } else {
    // Large overlapping tonal patches then soil granules: no checkerboard or
    // flat yellow ground, and no scale-dependent geometry noise.
    const grassy=kind==='grass';
    ctx.fillStyle=grassy?'#516a2f':'#7a5b39';ctx.fillRect(0,0,n,n);
    for(let i=0;i<200;i++){
      const x=r()*n,y=r()*n,rad=30+r()*150;
      const g=ctx.createRadialGradient(x,y,0,x,y,rad);
      if(grassy)g.addColorStop(0,i%3===0?'#7e9446aa':i%3===1?'#41582688':'#8a7a4855');
      else g.addColorStop(0,i%3===0?'#55602c66':i%3===1?'#a37d4e77':'#3a261466');
      g.addColorStop(1,grassy?'#5d753600':'#7a5b3900');
      ctx.fillStyle=g;ctx.fillRect(x-rad,y-rad,rad*2,rad*2);
    }
    for(let i=0;i<46000;i++){
      ctx.fillStyle=grassy?['#9fb2604f','#2f421d55','#6f8a3c49','#c2b37a22'][i%4]:['#a4855a4f','#2e211344','#5d6a2c40','#b8a0702e'][i%4];
      const x=r()*n,y=r()*n;ctx.beginPath();ctx.ellipse(x,y,.5+r()*2.6,.4+r()*1.3,r()*6.28,0,6.28);ctx.fill();
    }
    for(let i=0;i<1600;i++){
      ctx.strokeStyle=i%3===0?'#d6af58aa':'#a98a4666';ctx.lineWidth=.7;
      const x=r()*n,y=r()*n,a=r()*6.28,l=2+r()*7;
      ctx.beginPath();ctx.moveTo(x,y);ctx.lineTo(x+Math.cos(a)*l,y+Math.sin(a)*l);ctx.stroke();
    }
  }
  const tex=new T.CanvasTexture(c);tex.wrapS=tex.wrapT=T.RepeatWrapping;tex.anisotropy=8;
  maps.set(kind,tex);return tex;
}
export function farmMaterial(kind) {
  if(mats.has(kind))return mats.get(kind);
  const opts={map:canvasMap(kind),color:0xffffff,roughness:kind==='wood'?.88:1,metalness:0};
  // Thatch undersides only see bounce light; a warm self-glow keeps them golden-brown like the reference.
  if(kind==='thatch')Object.assign(opts,{color:0xb59a72});
  // Blended rather than alpha-tested: thin wire would break up into dashes in the distance.
  if(kind==='mesh')Object.assign(opts,{side:T.DoubleSide,transparent:true,opacity:.9,depthWrite:false,alphaTest:.01,roughness:.8});
  const material=new T.MeshStandardMaterial(opts);mats.set(kind,material);return material;
}
function solid(color,glow=0) {
  const key='solid'+color+':'+glow;if(!mats.has(key))mats.set(key,new T.MeshStandardMaterial({color,roughness:1,emissive:new T.Color(color).multiplyScalar(glow)}));return mats.get(key);
}
function unitPlane(){if(!geometries.has('plane'))geometries.set('plane',new T.PlaneGeometry(1,1));return geometries.get('plane');}
function unitBox(){if(!geometries.has('box'))geometries.set('box',new T.BoxGeometry(1,1,1));return geometries.get('box');}
function box(g,w,h,d,material,x,y,z){const m=new T.Mesh(unitBox(),material);m.scale.set(w,h,d);m.position.set(x,y,z);m.castShadow=true;m.receiveShadow=true;g.add(m);return m;}
function beamTransform(a,b,width,depth=width){
  const av=new T.Vector3(...a),bv=new T.Vector3(...b),delta=bv.clone().sub(av),q=new T.Quaternion().setFromUnitVectors(new T.Vector3(0,1,0),delta.clone().normalize());
  return new T.Matrix4().compose(av.add(bv).multiplyScalar(.5),q,new T.Vector3(width,delta.length(),depth));
}
// Small, self-contained merge utility avoids a runtime dependency on examples/.
function mergeTransformed(base,transforms) {
  const source=base.index?base.toNonIndexed():base;
  const pos=source.attributes.position, norm=source.attributes.normal, uv=source.attributes.uv;
  const p=[], nn=[], u=[],v=new T.Vector3(),nm=new T.Matrix3();
  for(const matrix of transforms){
    nm.getNormalMatrix(matrix);
    for(let i=0;i<pos.count;i++){
      v.fromBufferAttribute(pos,i).applyMatrix4(matrix);p.push(v.x,v.y,v.z);
      v.fromBufferAttribute(norm,i).applyMatrix3(nm).normalize();nn.push(v.x,v.y,v.z);
      if(uv)u.push(uv.getX(i),uv.getY(i));
    }
  }
  const geo=new T.BufferGeometry();geo.setAttribute('position',new T.Float32BufferAttribute(p,3));geo.setAttribute('normal',new T.Float32BufferAttribute(nn,3));if(uv)geo.setAttribute('uv',new T.Float32BufferAttribute(u,2));geo.computeBoundingSphere();
  if(source!==base)source.dispose();return geo;
}
function mergedBeams(group,transforms,material,shadow=true){
  if(!transforms.length)return;
  const mesh=new T.Mesh(mergeTransformed(unitBox(),transforms),material);mesh.castShadow=shadow;mesh.receiveShadow=true;group.add(mesh);return mesh;
}
let faces;
function thatchFaces(){
  if(faces)return faces;
  const map=canvasMap('thatch');
  const top=new T.MeshStandardMaterial({map,color:0xb59a72,roughness:1});
  const under=new T.MeshStandardMaterial({map,color:0xcdb08a,roughness:1,emissive:0x7a5520,emissiveIntensity:.7});
  faces=[top,under];return faces;
}
export function createThatchRoof(hx,hz,eave=7,ridge=13) {
  const roof=new T.Group();roof.name='bo2-thatch-roof';
  roof.userData.eave=eave;roof.userData.ridge=ridge;
  const slope=Math.atan2(ridge-eave,hx),length=Math.hypot(hx,ridge-eave),r=rngFor(Math.round(hx*71+hz*19));
  box(roof,.3,.32,hz*2+.5,farmMaterial('wood'),0,ridge-.23,0).castShadow=false;
  for(const side of [-1,1]){
    const wing=new T.Group();wing.name=side<0?'roof-left':'roof-right';roof.add(wing);
    wing.userData.roof={hx:hx+.3,hz:hz+.4,eave,ridge,side};
    // Top face stays a sunlit thatch; the underside glows warm golden-brown as seen from inside.
    const [top,under]=thatchFaces();
    const skin=new T.Mesh(unitBox(),top);skin.scale.set(length+.65,.21,hz*2+.8);skin.position.set(side*hx/2,(ridge+eave)/2,0);skin.receiveShadow=true;wing.add(skin);skin.rotation.z=-side*slope;
    const ceil=new T.Mesh(unitPlane(),under);ceil.scale.set(length+.65,hz*2+.8,1);// Lay the plane flat facing down (X first), then tilt it with the slope (Z): Euler order ZYX.
    ceil.rotation.set(Math.PI/2,0,-side*slope,'ZYX');ceil.position.set(side*hx/2,(ridge+eave)/2-.11,0);wing.add(ceil);
    // Mái lợp thưa ánh nắng như ảnh mẫu: sàn dưới mái vẫn sáng, không đổ một mảng bóng kín.
    skin.castShadow=false;
    // A second thin fibrous layer under the roof is visible from the hall.
    // The narrow golden stems follow the slope, not a horizontal plank pattern.
    const fibers=[[],[],[]],rafters=[];
    for(let z=-hz-.25;z<=hz+.25;z+=.12){
      for(let k=0;k<7;k++){
        const u=(k+r()*.35)/7, l=.13+r()*.12;
        const x=side*hx*u, y=ridge-(ridge-eave)*u-.145;
        fibers[k%3].push(beamTransform([x,y,z],[x+side*hx*l,y-(ridge-eave)*l-.02,z+(r()-.5)*.1],.018+r()*.015,.022));
      }
    }
    // Loose, uneven stems at the eave and the front/back cut edges.
    for(let z=-hz-.35;z<hz+.4;z+=.065){
      const ext=.18+r()*.47;
      fibers[Math.floor(r()*3)].push(beamTransform([side*(hx-.2),eave+.04,z],[side*(hx+ext),eave-.13-ext*.38,z+(r()-.5)*.14],.025,.023));
    }
    for(const zz of [-hz-.42,hz+.42])for(let x=.12;x<hx;x+=.12){
      const y=ridge-(ridge-eave)*x/hx;
      fibers[Math.floor(r()*3)].push(beamTransform([side*x,y-.07,zz],[side*(x+.08),y-.1-r()*.18,zz+Math.sign(zz)*(.15+r()*.25)],.024,.023));
    }
    fibers.forEach((arr,i)=>mergedBeams(wing,arr,solid([0xc49a46,0xd8b361,0x8f743d][i],.5),false));
    for(let z=-hz;z<=hz+.01;z+=6){rafters.push(beamTransform([0,ridge-.36,z],[side*(hx+.1),eave-.37,z],.22,.22));}
    for(let x=3;x<hx;x+=3.4){
      const yy=ridge-(ridge-eave)*x/hx-.43;
      rafters.push(beamTransform([side*x,yy,-hz],[side*x,yy,hz],.14,.16));
    }
    mergedBeams(wing,rafters,farmMaterial('wood'),false);
  }
  return roof;
}
const nestCache=new Map();
export function createStrawNest(seed=1) {
  const variant=Math.abs(seed|0)%5;
  if(!nestCache.has(variant)){
    const group=new T.Group(),r=rngFor(991+variant*101);
    // Low irregular rolled straw, much flatter than a woven basket wall.
    const torus=new T.TorusGeometry(.83,.19,9,64),p=torus.attributes.position;
    for(let i=0;i<p.count;i++){
      const x=p.getX(i),y=p.getY(i),z=p.getZ(i),a=Math.atan2(y,x);
      const wobble=1+.028*Math.sin(a*7+variant)+.014*Math.cos(a*13);
      p.setXYZ(i,x*wobble,y*wobble,z*(.80+.1*Math.sin(a*9)));
    }
    torus.computeVertexNormals();
    const rim=new T.Mesh(torus,farmMaterial('straw'));rim.rotation.x=-Math.PI/2;rim.position.y=.22;rim.castShadow=true;rim.receiveShadow=true;group.add(rim);
    if(!mats.has('nest-bed'))mats.set('nest-bed',new T.MeshStandardMaterial({map:canvasMap('straw'),color:0xa18b61,roughness:1}));
    const bed=new T.Mesh(new T.CircleGeometry(.76,32),mats.get('nest-bed'));bed.rotation.x=-Math.PI/2;bed.position.y=.085;bed.receiveShadow=true;group.add(bed);
    const fibers=[[],[],[]];
    for(let i=0;i<620;i++){
      const a=r()*Math.PI*2,rad=.69+r()*.34,edge=Math.max(0,1-Math.abs(rad-.83)/.25),height=.15+edge*.15+r()*.04;
      const length=.18+r()*.38, twist=a+(r()-.5)*1.3;
      const x=Math.cos(a)*rad,z=Math.sin(a)*rad;
      fibers[i%3].push(beamTransform([x,height,z],[x+Math.cos(twist)*length,height-.04-r()*.08,z+Math.sin(twist)*length],.012+r()*.009,.014));
    }
    fibers.forEach((arr,i)=>mergedBeams(group,arr,solid([0xe4bd65,0xc49c48,0xb18a3a][i]),false));
    nestCache.set(variant,group);
  }
  return nestCache.get(variant).clone(true);
}
function instanced(group,base,material,matrices,colors) {
  if(!matrices.length)return;
  const mesh=new T.InstancedMesh(base,material,matrices.length);
  matrices.forEach((m,i)=>{mesh.setMatrixAt(i,m);if(colors)mesh.setColorAt(i,new T.Color(colors[i]));});
  mesh.instanceMatrix.needsUpdate=true;if(mesh.instanceColor)mesh.instanceColor.needsUpdate=true;
  // r128 culls by the original geometry, not transformed instances.
  mesh.frustumCulled=false;mesh.receiveShadow=true;mesh.castShadow=false;group.add(mesh);return mesh;
}
let mossMap;
function mossMaterial(){
  if(mats.has('moss'))return mats.get('moss');
  const c=document.createElement('canvas');c.width=c.height=128;const ctx=c.getContext('2d'),r=rngFor(163);
  for(let i=0;i<1800;i++){
    const x=r()*128,y=r()*128,d=Math.hypot(x-64,y-64)/64;
    if(r()>Math.max(0,1-d*d))continue;
    ctx.fillStyle=['#52602a88','#73813d99','#394b2477'][i%3];ctx.beginPath();ctx.ellipse(x,y,1+r()*6,1+r()*3,r()*6.28,0,6.28);ctx.fill();
  }
  mossMap=new T.CanvasTexture(c);
  const m=new T.MeshStandardMaterial({map:mossMap,transparent:true,depthWrite:false,roughness:1,polygonOffset:true,polygonOffsetFactor:-1});mats.set('moss',m);return m;
}
export function dressGround(group,hx,hz,seed=1,exclude=()=>false) {
  const r=rngFor(seed),area=hx*hz*4;
  const straw=[],blades=[],stones=[],patches=[],strawColors=[],grassColors=[];
  const transform=(x,y,z,sx,sy,sz,rx=0,ry=0,rz=0)=>new T.Matrix4().compose(new T.Vector3(x,y,z),new T.Quaternion().setFromEuler(new T.Euler(rx,ry,rz)),new T.Vector3(sx,sy,sz));
  for(let i=0;i<Math.min(19000,area*9);i++){
    const x=(r()*2-1)*hx,z=(r()*2-1)*hz;if(exclude(x,z))continue;
    // Broad clustered distribution with sparse breathing space between patches.
    const patch=Math.sin(x*.6+seed)+Math.cos(z*.51)+Math.sin(x*.9+z*.5);
    if(patch<-.45&&r()<.82)continue;
    straw.push(transform(x,.026+r()*.012,z,.014+r()*.012,.008,.08+r()*.3,0,r()*6.28,0));
    strawColors.push([0xd6b263,0xc39c4e,0xe2c274,0xa0833d][i%4]);
  }
  const tuftCount=Math.min(800,area*.55);
  for(let i=0;i<tuftCount;i++){
    const x=(r()*2-1)*(hx-.15),z=(r()*2-1)*(hz-.15);if(exclude(x,z))continue;
    if(Math.abs(x)<hx*.75&&Math.abs(z)<hz*.75&&r()<.45)continue;
    for(let k=0;k<5;k++){
      const h=.12+r()*.34,xx=x+(r()-.5)*.2,zz=z+(r()-.5)*.2;
      blades.push(transform(xx,h*.5,zz,.035+r()*.045,h,.035,0,r()*6.28,(r()-.5)*.8));
      grassColors.push([0x627d3a,0x83934b,0x486332,0x77924a][k%4]);
    }
    if(i%4===0)stones.push(transform(x,.04+r()*.04,z,.08+r()*.15,.04+r()*.08,.06+r()*.12,0,r()*6.28,0));
  }
  for(let i=0;i<Math.min(100,area*.09);i++){
    const x=(r()*2-1)*hx,z=(r()*2-1)*hz;if(exclude(x,z))continue;
    patches.push(transform(x,.018,z,1+r()*3,1+r()*3,1,-Math.PI/2,0,r()*6.28));
  }
  instanced(group,unitBox(),solid(0xffffff),straw,strawColors);
  if(!geometries.has('blade'))geometries.set('blade',new T.ConeGeometry(.5,1,3));
  if(!geometries.has('stone'))geometries.set('stone',new T.IcosahedronGeometry(1,0));
  if(!geometries.has('patch'))geometries.set('patch',new T.PlaneGeometry(1,1));
  instanced(group,geometries.get('blade'),solid(0xffffff),blades,grassColors);
  instanced(group,geometries.get('stone'),solid(0x9a9276),stones);
  instanced(group,geometries.get('patch'),mossMaterial(),patches);
  return {straw:straw.length,grass:blades.length,stones:stones.length};
}

// Navigation uses the same bounds/obstacles shipped by the authoritative server.
export function walkable(zone,x,z) {
  if(!zone || Math.abs(x-(zone.cx||0))>zone.halfX-.35 || Math.abs(z-(zone.cz||0))>zone.halfZ-.35)return false;
  return !(zone.obstacles||[]).some(o=>Math.abs(x-o.x)<o.w/2+.35&&Math.abs(z-o.z)<o.d/2+.35);
}
export function moveSafe(zone,from,x,z) {
  if(walkable(zone,x,z))return {x,z};
  let out={x:from.x,z:from.z};
  if(walkable(zone,x,out.z))out.x=x;
  if(walkable(zone,out.x,z))out.z=z;
  return out;
}
export function clearLine(zone,a,b) {
  const n=Math.max(1,Math.ceil(Math.hypot(a.x-b.x,a.z-b.z)/.25));
  for(let i=1;i<=n;i++)if(!walkable(zone,a.x+(b.x-a.x)*i/n,a.z+(b.z-a.z)*i/n))return false;
  return true;
}
export function findPath(zone,from,to) {
  if(!walkable(zone,to.x,to.z))return [];
  if(clearLine(zone,from,to))return [to];
  const key=(x,z)=>x+','+z, start={x:Math.round(from.x),z:Math.round(from.z)},end={x:Math.round(to.x),z:Math.round(to.z)};
  const open=[{...start,g:0,f:0}],seen=new Map(),closed=new Set();seen.set(key(start.x,start.z),open[0]);
  let goal=null,steps=0;
  while(open.length&&steps++<16000){
    open.sort((a,b)=>b.f-a.f);const a=open.pop(),ak=key(a.x,a.z);if(closed.has(ak))continue;closed.add(ak);
    if(Math.hypot(a.x-to.x,a.z-to.z)<1.5&&clearLine(zone,a,to)){goal=a;break;}
    for(const [dx,dz]of [[1,0],[-1,0],[0,1],[0,-1],[1,1],[-1,1],[1,-1],[-1,-1]]){
      const x=a.x+dx,z=a.z+dz,k=key(x,z);if(closed.has(k)||!walkable(zone,x,z)||!clearLine(zone,a,{x,z}))continue;
      const g=a.g+Math.hypot(dx,dz),old=seen.get(k);if(old&&old.g<=g)continue;
      const n={x,z,g,f:g+Math.hypot(x-end.x,z-end.z),parent:a};seen.set(k,n);open.push(n);
    }
  }
  if(!goal)return [];
  const result=[to];while(goal){result.unshift({x:goal.x,z:goal.z});goal=goal.parent;}
  const smooth=[];let at=from;
  for(let i=0;i<result.length;){let j=i;while(j+1<result.length&&clearLine(zone,at,result[j+1]))j++;smooth.push(result[j]);at=result[j];i=j+1;}
  return smooth;
}
// Which area of the seamless map a point is in; anywhere else is the region's outdoor area.
export function areaAt(zones,world,x,z){
  for(const id in zones){const a=zones[id];if(a.outdoor||id==='home'||id==='guildhall')continue;if(Math.abs(x-a.cx)<=a.halfX&&Math.abs(z-a.cz)<=a.halfZ)return id;}
  return world?.outdoor;
}
// Areas are authored in local space (centre 0,0, entrance on +z) and placed with an optional 180° turn.
export function toLocal(a,x,z){const cx=a.cx||0,cz=a.cz||0;return a.flip?{x:cx-x,z:cz-z}:{x:x-cx,z:z-cz};}
export function zoneRoute(zones,from,to){
  const q=[[from]],seen=new Set([from]);while(q.length){const p=q.shift(),id=p[p.length-1];if(id===to)return p;
    for(const it of zones[id]?.items||[])if(it.kind==='gate'&&!seen.has(it.to)){seen.add(it.to);q.push([...p,it.to]);}}
  return [];
}

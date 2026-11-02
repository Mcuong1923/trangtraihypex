const T=window.THREE;
const damp=(a,b,k,dt)=>T.MathUtils.lerp(a,b,1-Math.exp(-k*dt));
const wrap=a=>Math.atan2(Math.sin(a),Math.cos(a));
const k0=reduced=>reduced?100:4;
// Social view frames the hall from the door; follow view trails the chicken. Pitch: 0 = level, π/2 = top-down.
// Under a roof the follow view sits lower (followPitch/followRadius) so it stays beneath the thatch.
const followPitch=p=>p?.followPitch??.56, followRadius=p=>p?.followRadius??14;
export class CameraController {
  constructor(camera){this.camera=camera;this.yaw=0;this.targetYaw=0;this.pitch=.72;this.targetPitch=.72;this.radius=30;this.targetRadius=30;this.focus=new T.Vector3(0,1,-1);this.mode='social';this.zone=null;this.profile=null;this.userAt=0;}
  enter(zone,profile,pos,ry){
    this.zone=zone;this.profile=profile||{};
    const social=!!this.profile.focus;
    if(social){this.mode='social';this.targetYaw=this.yaw=this.profile.yaw||0;this.targetPitch=this.pitch=this.profile.pitch;this.targetRadius=this.radius=this.profile.radius;this.focus.set(...this.profile.focus);}
    else{
      // Arrive behind the chicken, looking where it walks: no 180° swing through the doorway.
      this.mode='follow';const behind=ry==null?0:wrap(ry+Math.PI);
      this.targetYaw=this.yaw=behind;this.targetPitch=this.pitch=this.profile.pitch||followPitch(this.profile);this.targetRadius=this.radius=this.profile.radius||followRadius(this.profile);
      if(pos)this.focus.set(pos.x,1.1,pos.z);
    }
  }
  orbit(dx,dy){this.targetYaw-=dx*.006;this.targetPitch=T.MathUtils.clamp(this.targetPitch+dy*.004,.26,1.05);this.userAt=performance.now();}
  zoom(delta){this.targetRadius=T.MathUtils.clamp(this.targetRadius+delta,6,this.mode==='social'?42:26);this.userAt=performance.now();}
  reset(){const p=this.profile||{};this.targetYaw=this.mode==='social'?(p.yaw||0):this.targetYaw;this.targetPitch=this.mode==='social'?p.pitch:(p.focus?followPitch(p):(p.pitch||.56));this.targetRadius=this.mode==='social'?p.radius:(p.focus?followRadius(p):(p.radius||14));}
  toggleSocial(){
    if(this.mode==='social'){this.mode='follow';this.targetPitch=followPitch(this.profile);this.targetRadius=followRadius(this.profile);return;}
    if(this.profile?.focus){this.mode='social';this.targetYaw=this.profile.yaw||0;this.targetPitch=this.profile.pitch;this.targetRadius=this.profile.radius;}
  }
  // Cận cảnh khi đào giun: máy đặt thấp bên phải-trước con gà, nhìn vào mỏ và miệng hố (PLAN_LOGIC_GAME 4.1).
  closeup(c){
    if(c){if(this.mode!=='closeup')this.prevMode=this.mode;this.mode='closeup';this.close=c;return;}
    if(this.mode==='closeup'){this.mode=this.prevMode||'follow';this.close=null;}
  }
  update(dt,pos,moving,reduced,vehicle){
    if(this.mode==='closeup'&&this.close){
      const c=this.close,fx=Math.sin(c.ry),fz=Math.cos(c.ry),rx=Math.cos(c.ry),rz=-Math.sin(c.ry),k=1-Math.exp(-(reduced?100:5)*dt);
      this.focus.lerp(new T.Vector3(c.x+fx*.62,.48,c.z+fz*.62),k);
      this.camera.position.lerp(new T.Vector3(c.x+rx*1.25+fx*1.55,1.05,c.z+rz*1.25+fz*1.55),k);
      if(Math.abs(this.camera.fov-50)>.05){this.camera.fov=damp(this.camera.fov,50,k0(reduced),dt);this.camera.updateProjectionMatrix();}
      this.camera.lookAt(this.focus);return;
    }
    if(moving&&this.mode==='social'){this.mode='follow';this.targetRadius=Math.min(this.targetRadius,followRadius(this.profile)+1);this.targetPitch=followPitch(this.profile);}
    // Each reference framing may widen the lens (whole market or bus in one shot); following uses 45°.
    const fov=this.mode==='social'&&this.profile?.fov||45;
    if(Math.abs(this.camera.fov-fov)>.05){this.camera.fov=damp(this.camera.fov,fov,k0(reduced),dt);this.camera.updateProjectionMatrix();}
    const k=reduced?100:5;
    // Shortest-arc yaw interpolation keeps the camera from spinning the long way round.
    this.yaw+=wrap(this.targetYaw-this.yaw)*(1-Math.exp(-k*dt));
    this.pitch=damp(this.pitch,this.targetPitch,k,dt);this.radius=damp(this.radius,this.targetRadius,k,dt);
    const target=this.mode==='social'?new T.Vector3(...this.profile.focus):new T.Vector3(pos.x-Math.sin(this.yaw)*1.2,1.1+(vehicle?.8:0),pos.z-Math.cos(this.yaw)*1.2);
    this.focus.lerp(target,1-Math.exp(-(reduced?100:this.mode==='social'?2.5:6)*dt));
    let r=this.radius*(vehicle?1.2:1);
    const dir=new T.Vector3(Math.sin(this.yaw)*Math.cos(this.pitch),Math.sin(this.pitch),Math.cos(this.yaw)*Math.cos(this.pitch));
    // Pull in against solid obstacles (stage, trunks) instead of clipping through them.
    for(let d=.8;d<r;d+=.4){const p=this.focus.clone().addScaledVector(dir,d);if((this.zone?.obstacles||[]).some(o=>p.y<o.h+.25&&Math.abs(p.x-o.x)<o.w/2+.2&&Math.abs(p.z-o.z)<o.d/2+.2)){r=Math.max(.8,d-.5);break;}}
    this.camera.position.copy(this.focus).addScaledVector(dir,r);this.camera.lookAt(this.focus);
  }
}

const T=window.THREE;
const damp=(a,b,k,dt)=>T.MathUtils.lerp(a,b,1-Math.exp(-k*dt));
const wrap=a=>Math.atan2(Math.sin(a),Math.cos(a));
// Social view frames the hall from the door; follow view trails the chicken. Pitch: 0 = level, π/2 = top-down.
export class CameraController {
  constructor(camera){this.camera=camera;this.yaw=0;this.targetYaw=0;this.pitch=.72;this.targetPitch=.72;this.radius=30;this.targetRadius=30;this.focus=new T.Vector3(0,1,-1);this.mode='social';this.zone=null;this.profile=null;this.userAt=0;}
  enter(zone,profile,pos,ry){
    this.zone=zone;this.profile=profile||{};
    const social=!!this.profile.focus;
    if(social){this.mode='social';this.targetYaw=this.yaw=this.profile.yaw||0;this.targetPitch=this.pitch=this.profile.pitch;this.targetRadius=this.radius=this.profile.radius;this.focus.set(...this.profile.focus);}
    else{
      // Arrive behind the chicken, looking where it walks: no 180° swing through the doorway.
      this.mode='follow';const behind=ry==null?0:wrap(ry+Math.PI);
      this.targetYaw=this.yaw=behind;this.targetPitch=this.pitch=this.profile.pitch||.56;this.targetRadius=this.radius=this.profile.radius||14;
      if(pos)this.focus.set(pos.x,1.1,pos.z);
    }
  }
  orbit(dx,dy){this.targetYaw-=dx*.006;this.targetPitch=T.MathUtils.clamp(this.targetPitch+dy*.004,.26,1.05);this.userAt=performance.now();}
  zoom(delta){this.targetRadius=T.MathUtils.clamp(this.targetRadius+delta,6,this.mode==='social'?42:26);this.userAt=performance.now();}
  reset(){const p=this.profile||{};this.targetYaw=this.mode==='social'?(p.yaw||0):this.targetYaw;this.targetPitch=this.mode==='social'?p.pitch:(p.focus?.56:(p.pitch||.56));this.targetRadius=this.mode==='social'?p.radius:(p.focus?14:(p.radius||14));}
  toggleSocial(){
    if(this.mode==='social'){this.mode='follow';this.targetPitch=.56;this.targetRadius=14;return;}
    if(this.profile?.focus){this.mode='social';this.targetYaw=this.profile.yaw||0;this.targetPitch=this.profile.pitch;this.targetRadius=this.profile.radius;}
  }
  update(dt,pos,moving,reduced,vehicle){
    if(moving&&this.mode==='social'){this.mode='follow';this.targetRadius=Math.min(this.targetRadius,15);this.targetPitch=.56;}
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

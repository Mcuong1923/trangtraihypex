// One gesture owner per pointer: drag never turns into a tap, pinch never walks.
export function bindWorldInput(canvas,{camera,enabled,tap}){
 const pointers=new Map();let pinching=false,pinchDistance=0;
 canvas.addEventListener('pointerdown',e=>{if(!enabled())return;canvas.setPointerCapture(e.pointerId);pointers.set(e.pointerId,{x:e.clientX,y:e.clientY,sx:e.clientX,sy:e.clientY,drag:false});if(pointers.size===2){pinching=true;const p=[...pointers.values()];pinchDistance=Math.hypot(p[0].x-p[1].x,p[0].y-p[1].y);}});
 canvas.addEventListener('pointermove',e=>{const p=pointers.get(e.pointerId);if(!p)return;const dx=e.clientX-p.x,dy=e.clientY-p.y;p.x=e.clientX;p.y=e.clientY;if(Math.hypot(p.x-p.sx,p.y-p.sy)>6)p.drag=true;
  if(pointers.size===2){const a=[...pointers.values()],d=Math.hypot(a[0].x-a[1].x,a[0].y-a[1].y);camera.zoom((pinchDistance-d)*.045);pinchDistance=d;}else if(!pinching&&p.drag&&enabled())camera.orbit(dx,dy);
 });
 const end=(e,cancel=false)=>{const p=pointers.get(e.pointerId);if(p&&!cancel&&!pinching&&!p.drag&&enabled())tap(e.clientX,e.clientY);pointers.delete(e.pointerId);if(!pointers.size)pinching=false;};
 canvas.addEventListener('pointerup',e=>end(e));canvas.addEventListener('pointercancel',e=>end(e,true));canvas.addEventListener('lostpointercapture',e=>end(e,true));
 canvas.addEventListener('wheel',e=>{if(enabled()){e.preventDefault();camera.zoom(e.deltaY*.012);}},{passive:false});
 window.addEventListener('blur',()=>{pointers.clear();pinching=false;});
}

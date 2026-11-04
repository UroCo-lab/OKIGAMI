const fs=require('node:fs'),vm=require('node:vm'),assert=require('node:assert/strict'),path=require('node:path');
const html=fs.readFileSync(path.join(__dirname,'src/web/index.html'),'utf8');
const source=html.slice(html.indexOf('const clampUnit='),html.indexOf('function render()'));
const elements=new Map(),listeners=new Map(),classes=new Set(),capture=new Set();
const element=id=>{if(!elements.has(id))elements.set(id,{value:'50',disabled:false,hidden:false,textContent:'',addEventListener(type,fn){listeners.set(id+':'+type,fn)}});return elements.get(id)};
let renders=0,queued=null;
const sandbox={
 mode:'image',loadedImage:{width:2400,height:1200},imagePlacement:null,cropX:.5,cropY:.5,cropDrag:null,cropRenderTick:null,W:1072,H:1448,
 $:element,document:{querySelectorAll(){return[]}},render(){renders++},noteBrowserActivity(){},requestAnimationFrame(fn){queued=fn;return 1},cancelAnimationFrame(){queued=null},
 canvas:{classList:{contains:v=>classes.has(v),add:v=>classes.add(v),remove:v=>classes.delete(v),toggle(v,on){if(on)classes.add(v);else classes.delete(v)}},
 getBoundingClientRect:()=>({left:0,top:0,width:268,height:362}),setPointerCapture:id=>capture.add(id),hasPointerCapture:id=>capture.has(id),releasePointerCapture:id=>capture.delete(id),addEventListener:(name,fn)=>listeners.set('canvas:'+name,fn)},
};
element('image-fit').value='cover';vm.createContext(sandbox);vm.runInContext(source,sandbox);
function run(code){return vm.runInContext(code,sandbox)}
function pointer(type,x,y,id=1){listeners.get('canvas:'+type)({isPrimary:true,button:0,pointerId:id,clientX:x,clientY:y,preventDefault(){}})}
let cases=0;
for(const [iw,ih]of [[2400,1200],[1200,2400],[1072,1448],[1448,1072],[100,100],[6000,800]])for(const [bw,bh]of [[1072,1448],[1448,1072],[1072,1247]])for(const x of [0,.5,1])for(const y of [0,.5,1]){
 const r=sandbox.coverPlacement(iw,ih,bw,bh,x,y),eps=1e-7;
 assert.ok(r.x<=eps&&r.y<=eps&&r.x+r.dw>=bw-eps&&r.y+r.dh>=bh-eps,'crop exposed empty space');
 if(x===0)assert.ok(r.x===0);if(y===0)assert.ok(r.y===0);
 if(x===1)assert.ok(Math.abs(r.x+r.dw-bw)<eps);if(y===1)assert.ok(Math.abs(r.y+r.dh-bh)<eps);cases++;
}
run('imagePlacement=coverPlacement(2400,1200,W,H,cropX,cropY);updateCropControls()');
assert.equal(element('crop-x').disabled,false);assert.equal(element('crop-y').disabled,true);
pointer('pointerdown',100,100);pointer('pointermove',200,100);pointer('pointerup',200,100);
assert.ok(Math.abs(sandbox.cropX-(.5-400/1824))<1e-7,'CSS-scaled drag mismatch');assert.equal(renders,1,'release did not flush latest crop');
pointer('pointerdown',100,100);pointer('pointermove',-10000,100);pointer('pointerup',-10000,100);assert.equal(sandbox.cropX,1,'right crop edge not clamped');
element('crop-reset').onclick();assert.equal(sandbox.cropX,.5);assert.equal(sandbox.cropY,.5);
run('loadedImage={width:1200,height:2400};imagePlacement=coverPlacement(1200,2400,W,H,cropX,cropY);updateCropControls()');
assert.equal(element('crop-x').disabled,true);assert.equal(element('crop-y').disabled,false);
pointer('pointerdown',100,150);pointer('pointermove',100,-10000);pointer('pointercancel',100,-10000);assert.equal(sandbox.cropY,1);assert.equal(sandbox.cropDrag,null);assert.equal(capture.size,0);
run('loadedImage={width:1072,height:1448};imagePlacement=coverPlacement(1072,1448,W,H,.5,.5);updateCropControls()');
assert.equal(element('crop-x').disabled,true);assert.equal(element('crop-y').disabled,true);assert.equal(classes.has('crop-ready'),false);
run('loadedImage={width:2400,height:1200};imagePlacement=coverPlacement(2400,1200,W,H-160,.5,.5);imagePlacement.areaY=160;updateCropControls()');
pointer('pointerdown',100,10);assert.equal(sandbox.cropDrag,null,'caption area started image drag');
pointer('pointerdown',100,100);assert.ok(sandbox.cropDrag,'image below top caption did not start drag');pointer('pointerup',100,100);
run('imagePlacement.areaY=0;imagePlacement.areaH=H;imagePlacement.excludedBand={top:600,bottom:800}');
pointer('pointerdown',100,175);assert.equal(sandbox.cropDrag,null,'center text band started image drag');
pointer('pointerdown',100,50);assert.ok(sandbox.cropDrag,'upper photo area cannot be dragged');pointer('pointerup',100,50);
pointer('pointerdown',100,300);assert.ok(sandbox.cropDrag,'lower photo area cannot be dragged');pointer('pointerup',100,300);
console.log(`CROP_PASS: ${cases} coverage cases; drag scaling, bounds, cancel, reset, unavailable axes`);

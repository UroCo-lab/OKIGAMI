const fs=require('node:fs'),vm=require('node:vm'),assert=require('node:assert/strict'),path=require('node:path');
const html=fs.readFileSync(path.join(__dirname,'src/web/index.html'),'utf8');
const source=html.slice(html.indexOf('let checkingConnection='),html.indexOf("window.addEventListener('hashchange'"));
const elements=new Map(),el=id=>{if(!elements.has(id))elements.set(id,{value:'',textContent:''});return elements.get(id)};
let calls=0,activity=0,next={sleeping:false,preview:false,width:1072,height:1448,sleepAfterSeconds:5,status:{battery:45}};
const s={$:el,ready:false,busy:false,token:'test',previewMode:false,deviceStatus:{},document:{hidden:false,addEventListener(){}},
 api:async(p)=>{assert.equal(p,'/api/config');calls++;if(next instanceof Error)throw next;return{json:async()=>next}},loadConnectionQR:async()=>{},
 render(){},orient(){},controls(){el('send').disabled=!s.ready},status(text){el('status').textContent=text},noteBrowserActivity(){activity++},setTimeout(){}};
vm.createContext(s);vm.runInContext(source,s);
(async()=>{
 el('message').value='unsent draft';await s.syncConnection(true);assert.equal(s.ready,true);assert.equal(el('sleep-after').value,'5');
 next={...next,sleeping:true};await s.syncConnection();assert.equal(s.ready,false);assert.equal(el('send').disabled,true);
 next=new Error('network suspended');await s.syncConnection();assert.equal(s.ready,false);
 next={sleeping:false,preview:false,sleepAfterSeconds:5,status:{battery:44}};await s.syncConnection();assert.equal(s.ready,true);assert.equal(el('send').disabled,false);assert.equal(el('message').value,'unsent draft');
 assert.equal(activity,0,'read-only connection polling extended sleep');
 next=Object.assign(new Error('old token'),{status:401});await s.syncConnection();assert.equal(s.ready,false);const before=calls;await s.syncConnection();assert.equal(calls,before,'invalid token repeatedly retried');
 assert.equal(html.includes('id="reconnect"'),false);console.log('CONNECTION_PASS: sleep, network loss, automatic recovery, draft preserved, no activity from polling, old token requires QR');
})().catch(e=>{console.error(e);process.exitCode=1});

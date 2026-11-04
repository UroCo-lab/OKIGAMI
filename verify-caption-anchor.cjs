const fs=require('node:fs'),vm=require('node:vm'),assert=require('node:assert/strict'),path=require('node:path');
const html=fs.readFileSync(path.join(__dirname,'src/web/index.html'),'utf8'),source=html.slice(html.indexOf('function captionAnchor('),html.indexOf('const anchorButtons=')),s={};vm.createContext(s);vm.runInContext(source,s);
for(const [w,h]of [[1072,1448],[1448,1072]])for(let n=1;n<=9;n++){
 const a=s.captionAnchor(n,w,h,120,true,35,0),row=Math.floor((n-1)/3),col=(n-1)%3;
 assert.equal(a.row,row);assert.equal(a.col,col);assert.equal(a.align,['left','center','right'][col]);
 assert.equal(a.y,[35,(h-120)/2,h-155][row]);assert.equal(a.x,[80,w/2,w-80][col]);
 // The anchor does not shrink wrapping to a third or clip at cell boundaries.
 const tall=s.captionAnchor(n,w,h,700,true,35,0);assert.equal(tall.x,a.x);
 for(const bandH of [120,400]){const b=s.captionAnchor(n,w,h,120,false,35,bandH),zones=s.captionPhotoZones(h,b.bandY,bandH);assert.equal(zones.reduce((sum,z)=>sum+z.height,0),h-bandH);assert.equal(b.y,b.bandY+35);for(const z of zones)assert.ok(z.screenY+z.height<=b.bandY||z.screenY>=b.bandY+bandH,'photo overlaps band')}
}
console.log('ANCHOR_PASS: all 9 positions in both orientations; tall text spans cells; full photo retained outside top/center/bottom bands');

// Exercise the parsers in a packaged/installed runtime, never a real workspace.
const assert=require('node:assert/strict');
const fs=require('node:fs');
const os=require('node:os');
const path=require('node:path');
const net=require('node:net');
const crypto=require('node:crypto');
const {spawn}=require('node:child_process');

async function main(binary){
 const scratch=fs.mkdtempSync(path.join(os.tmpdir(),'learn-source-smoke-'));
 const workspace=path.join(scratch,'workspace');fs.mkdirSync(workspace);
 const listener=net.createServer();await new Promise(r=>listener.listen(0,'127.0.0.1',r));
 const port=listener.address().port;await new Promise(r=>listener.close(r));
 const absent=path.join(scratch,'no-cli');
 const child=spawn(path.resolve(binary),['--no-browser','--workspace',workspace,'--port',String(port)],{cwd:scratch,windowsHide:true,env:{...process.env,WORKSPACE:workspace,LLL_WORKSPACE_ROOT:workspace,CLAUDE_BIN:absent,CODEX_BIN:absent,CODEBUDDY_BIN:absent,WORKBUDDY_BIN:absent,HERMES_BIN:absent,TRAE_BIN:absent},stdio:['ignore','pipe','pipe']});
 let log='',exited=false;child.stdout.on('data',b=>log+=b);child.stderr.on('data',b=>log+=b);child.on('exit',()=>exited=true);
 const base=`http://127.0.0.1:${port}`;
 const pause=()=>new Promise(r=>setTimeout(r,100));
 try{
  let health;const deadline=Date.now()+30000;
  while(Date.now()<deadline&&!exited){try{const r=await fetch(base+'/api/health');if(r.ok){health=await r.json();break}}catch{}await pause()}
  assert(health?.ok,log);assert.equal(path.resolve(health.workspace),workspace);
  const created=await fetch(base+'/api/projects',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({slug:'parser-smoke',title:'Parser smoke',projectType:'system-learning'})});
  assert.equal(created.status,201,await created.text());
  const url=base+'/api/projects/parser-smoke/sources';
  const fixtures=path.resolve(__dirname,'../../backend-go/internal/modules/sources/internal/parser/testdata');
  async function upload(name,bytes){
   const form=new FormData();form.set('operationId',crypto.randomUUID());form.set('file',new Blob([bytes]),name);
   return fetch(url,{method:'POST',body:form});
  }
  async function parsed(name,bytes,status='ready',failure){
   const response=await upload(name,bytes);assert.equal(response.status,201,name);
   const {source}=await response.json();let current;
   const end=Date.now()+15000;
   while(Date.now()<end){current=(await (await fetch(url+'/'+source.sourceId)).json()).source;if(current.status===status)break;if(current.status==='failed'&&status==='ready')throw Error(name+': '+current.failureCode);await pause()}
   assert.equal(current.status,status,name);if(failure)assert.equal(current.failureCode,failure,name);
   if(status==='ready'){
    assert(current.extractedBytes>0&&current.parser,name);
    const text=await (await fetch(url+'/'+source.sourceId+'/revisions/'+source.currentRevisionId+'/content')).text();assert(text.trim().length>0,name);
   }
  }
  for(const name of ['embedded-cjk.pdf','lesson.docx','lesson.pptx','lesson.xlsx'])await parsed(name,fs.readFileSync(path.join(fixtures,name)));
  await parsed('notes.md',Buffer.from('# 笔记\n词法环境与闭包。'));
  await parsed('table.csv',Buffer.from('概念,说明\n闭包,捕获词法环境\n'));
  await parsed('textless.pdf',fs.readFileSync(path.join(fixtures,'textless.pdf')),'failed','no_extractable_text');
  await parsed('broken.pdf',Buffer.from('not PDF'),'failed','invalid_document');
  const exact=await upload('boundary.zip',Buffer.alloc(50*1024*1024));assert.equal(exact.status,201,'50MiB must be accepted');await exact.arrayBuffer();
  const large=await upload('too-large.md',Buffer.alloc(50*1024*1024+1));assert.equal(large.status,413,'50MiB+1 must be rejected');await large.arrayBuffer();
  const items=(await (await fetch(url)).json()).sources;assert.equal(items.length,9,'oversize file left a source');
  const taskResponse=await fetch(base+'/api/projects/parser-smoke/assistant-tasks');assert.equal(taskResponse.status,200);
  const tasks=(await taskResponse.json()).tasks;assert(tasks===null||Array.isArray(tasks),'invalid task response');assert.equal((tasks??[]).length,0,'upload created assistant task');
  const stopped=await fetch(base+'/api/system/shutdown',{method:'POST'});assert.equal(stopped.status,202);
  for(let i=0;i<50&&!exited;i++)await pause();assert(exited,'shutdown failed');
  console.log('Installed parsers: PDF/MD/PPTX/DOCX/XLSX/CSV, textless/malformed errors, exact 50MiB/oversize rejection, no assistant passed');
 }finally{
  if(!exited){child.kill();await new Promise(r=>{child.once('exit',r);setTimeout(r,5000)})}
  const relative=path.relative(os.tmpdir(),scratch);if(relative.startsWith('learn-source-smoke-')&&!relative.includes(path.sep))fs.rmSync(scratch,{recursive:true,force:true,maxRetries:10,retryDelay:100});
 }
}
main(process.argv[2]).catch(e=>{console.error(e.stack);process.exitCode=1});

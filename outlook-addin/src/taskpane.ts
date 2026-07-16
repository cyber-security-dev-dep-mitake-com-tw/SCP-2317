declare const Office: any;

type Verdict = { status: "clean"|"suspicious"|"malicious"; score:number };
const API = "https://localhost:8080/v1/email/scan";

async function scanCurrentItem(): Promise<Verdict> {
  const item = Office.context.mailbox.item;
  const subject = await new Promise<string>(resolve => item.subject.getAsync((r:any)=>resolve(r.value || "")));
  const body = await new Promise<string>(resolve => item.body.getAsync("text", (r:any)=>resolve((r.value || "").slice(0, 4096))));
  const attachments = (item.attachments || []).map((a:any)=>({name:a.name, content_type:a.contentType}));
  const response = await fetch(API,{method:"POST",headers:{"content-type":"application/json"},body:JSON.stringify({message_id:item.itemId||"compose",sender:"",subject,body_preview:body,attachments})});
  if (!response.ok) throw new Error(`scan failed: ${response.status}`);
  return response.json();
}

(globalThis as any).scanCurrentItem = scanCurrentItem;

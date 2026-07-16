type Verdict = {
  status: "clean" | "suspicious" | "malicious";
  score: number;
  findings?: Array<{ engine: string; severity: string; summary: string }>;
};

const API_BASE =
  (globalThis as { SCP2317_API_BASE?: string }).SCP2317_API_BASE ||
  "http://127.0.0.1:8080";

const URL_RE = /\bhttps?:\/\/[^\s<>"')]+/gi;

function extractUrls(text: string): string[] {
  const found = text.match(URL_RE) || [];
  return [...new Set(found.map((u) => u.replace(/[.,;:)]+$/, "")))];
}

async function scanCurrentItem(): Promise<Verdict> {
  const item = Office.context.mailbox.item;
  if (!item) {
    throw new Error("no mailbox item");
  }

  const readItem = item as Office.MessageRead;
  const subject =
    typeof readItem.subject === "string" ? readItem.subject : "";

  const body = await new Promise<string>((resolve, reject) => {
    readItem.body.getAsync(Office.CoercionType.Text, (result) => {
      if (result.status === Office.AsyncResultStatus.Succeeded) {
        resolve((result.value || "").slice(0, 4096));
      } else {
        reject(result.error);
      }
    });
  });

  const sender = readItem.from?.emailAddress || "";
  const urls = extractUrls(body);
  const attachments = (readItem.attachments || []).map((a) => ({
    name: a.name,
    content_type: a.contentType,
  }));

  const response = await fetch(`${API_BASE}/v1/email/scan`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      message_id: readItem.itemId || "read-item",
      sender,
      subject,
      body_preview: body,
      urls,
      attachments,
    }),
  });
  if (!response.ok) {
    throw new Error(`scan failed: ${response.status}`);
  }
  return response.json();
}

Office.onReady(() => {
  (globalThis as { scanCurrentItem?: typeof scanCurrentItem }).scanCurrentItem =
    scanCurrentItem;
});

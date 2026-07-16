export type Verdict = {
  status: "clean" | "suspicious" | "malicious";
  score: number;
  findings?: Array<{ engine: string; severity: string; summary: string }>;
};

export type SandboxSubmitResponse = {
  job_id: string;
};

export const API_BASE =
  (globalThis as { SCP2317_API_BASE?: string }).SCP2317_API_BASE ||
  "http://127.0.0.1:8080";

const URL_RE = /\bhttps?:\/\/[^\s<>"')]+/gi;

export function extractUrls(text: string): string[] {
  const found = text.match(URL_RE) || [];
  return [...new Set(found.map((u) => u.replace(/[.,;:)]+$/, "")))];
}

function getBodyPreview(item: Office.MessageRead): Promise<string> {
  return new Promise((resolve, reject) => {
    item.body.getAsync(Office.CoercionType.Text, (result) => {
      if (result.status === Office.AsyncResultStatus.Succeeded) {
        resolve((result.value || "").slice(0, 4096));
      } else {
        reject(result.error);
      }
    });
  });
}

export function getCurrentReadItem(): Office.MessageRead {
  const item = Office.context.mailbox.item;
  if (!item) {
    throw new Error("no mailbox item");
  }
  return item as Office.MessageRead;
}

async function buildScanRequest(item: Office.MessageRead) {
  const subject = typeof item.subject === "string" ? item.subject : "";
  const body = await getBodyPreview(item);
  const sender = item.from?.emailAddress || "";
  const urls = extractUrls(body);
  const attachments = (item.attachments || []).map((a) => ({
    name: a.name,
    content_type: a.contentType,
  }));
  return {
    message_id: item.itemId || "read-item",
    sender,
    subject,
    body_preview: body,
    urls,
    attachments,
  };
}

export async function scanCurrentItem(): Promise<Verdict> {
  const readItem = getCurrentReadItem();
  const payload = await buildScanRequest(readItem);
  const response = await fetch(`${API_BASE}/v1/email/scan`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!response.ok) {
    throw new Error(`scan failed: ${response.status}`);
  }
  return response.json();
}

/** Escalates the current message to the sandbox stub (see internal/sandbox).
 *  This never executes the sample; it only records a not_executed job. */
export async function reportSuspicious(
  messageId: string
): Promise<SandboxSubmitResponse> {
  const response = await fetch(`${API_BASE}/v1/sandbox/submit`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      hash: messageId,
      objectRef: `outlook:${messageId}`,
      policy: "user-reported",
    }),
  });
  if (!response.ok) {
    throw new Error(`report failed: ${response.status}`);
  }
  return response.json();
}

export async function checkApiHealth(): Promise<boolean> {
  try {
    const response = await fetch(`${API_BASE}/healthz`, { method: "GET" });
    return response.ok;
  } catch {
    return false;
  }
}

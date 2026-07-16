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

/** Reads the Authentication-Results header so the API can surface SPF/DKIM/DMARC
 *  outcomes. Resolves to "" when the host/API set does not expose headers. */
function getAuthResults(item: Office.MessageRead): Promise<string> {
  return new Promise((resolve) => {
    const anyItem = item as unknown as {
      getAllInternetHeadersAsync?: (
        cb: (r: Office.AsyncResult<string>) => void
      ) => void;
    };
    if (typeof anyItem.getAllInternetHeadersAsync !== "function") {
      resolve("");
      return;
    }
    anyItem.getAllInternetHeadersAsync((result) => {
      if (result.status !== Office.AsyncResultStatus.Succeeded) {
        resolve("");
        return;
      }
      const line = (result.value || "")
        .split(/\r?\n/)
        .find((l) => /^authentication-results:/i.test(l));
      resolve(line ? line.replace(/^authentication-results:\s*/i, "") : "");
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
  const senderDisplay = item.from?.displayName || "";
  const replyToList = (item as unknown as {
    replyTo?: Array<{ emailAddress?: string }>;
  }).replyTo;
  const replyTo =
    (replyToList && replyToList[0] && replyToList[0].emailAddress) || "";
  const authResults = await getAuthResults(item);
  const urls = extractUrls(body);
  const attachments = (item.attachments || []).map((a) => ({
    name: a.name,
    content_type: a.contentType,
  }));
  return {
    message_id: item.itemId || "read-item",
    sender,
    sender_display: senderDisplay,
    reply_to: replyTo,
    auth_results: authResults,
    direction: "inbound",
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

function getComposeValue(
  getter: { getAsync: (cb: (r: Office.AsyncResult<string>) => void) => void }
): Promise<string> {
  return new Promise((resolve) => {
    getter.getAsync((r) =>
      resolve(r.status === Office.AsyncResultStatus.Succeeded ? r.value || "" : "")
    );
  });
}

/** Scans the message being composed for outbound data loss (DLP). Used by the
 *  OnMessageSend handler to decide whether to block the send. */
export async function scanOutgoingItem(): Promise<Verdict> {
  const item = Office.context.mailbox.item as Office.MessageCompose;
  const subject = await getComposeValue(item.subject);
  const body = await new Promise<string>((resolve) => {
    item.body.getAsync(Office.CoercionType.Text, (r) =>
      resolve(
        r.status === Office.AsyncResultStatus.Succeeded
          ? (r.value || "").slice(0, 8192)
          : ""
      )
    );
  });
  const payload = {
    message_id: "compose-item",
    direction: "outbound",
    subject,
    body_preview: body,
    urls: extractUrls(body),
  };
  const response = await fetch(`${API_BASE}/v1/email/scan`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!response.ok) {
    throw new Error(`outbound scan failed: ${response.status}`);
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

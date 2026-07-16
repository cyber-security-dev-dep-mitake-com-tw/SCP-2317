import {
  API_BASE,
  checkApiHealth,
  getCurrentReadItem,
  reportSuspicious,
  scanCurrentItem,
  Verdict,
} from "./scpApi.js";

const NOTIFICATION_ICON = "Icon.16x16";

function notify(
  key: string,
  type: Office.MailboxEnums.ItemNotificationMessageType,
  message: string
) {
  const item = Office.context.mailbox.item;
  if (!item) {
    return;
  }
  const details: Office.NotificationMessageDetails = {
    type,
    message: message.slice(0, 150),
  };
  if (type === Office.MailboxEnums.ItemNotificationMessageType.InformationalMessage) {
    details.icon = NOTIFICATION_ICON;
    details.persistent = false;
  }
  item.notificationMessages.replaceAsync(key, details);
}

function verdictSeverity(
  verdict: Verdict
): Office.MailboxEnums.ItemNotificationMessageType {
  return verdict.status === "malicious"
    ? Office.MailboxEnums.ItemNotificationMessageType.ErrorMessage
    : Office.MailboxEnums.ItemNotificationMessageType.InformationalMessage;
}

function verdictText(verdict: Verdict): string {
  return `SCP-2317: ${verdict.status.toUpperCase()} (score ${verdict.score})`;
}

async function onQuickScan(event: Office.AddinCommands.Event) {
  try {
    const verdict = await scanCurrentItem();
    notify("scp2317-verdict", verdictSeverity(verdict), verdictText(verdict));
  } catch (err) {
    notify(
      "scp2317-verdict",
      Office.MailboxEnums.ItemNotificationMessageType.ErrorMessage,
      `SCP-2317 scan failed: ${String(err)}`
    );
  } finally {
    event.completed();
  }
}

async function onReportSuspicious(event: Office.AddinCommands.Event) {
  try {
    const item = getCurrentReadItem();
    const verdict = await scanCurrentItem();
    const report = await reportSuspicious(item.itemId || "read-item");
    notify(
      "scp2317-report",
      Office.MailboxEnums.ItemNotificationMessageType.InformationalMessage,
      `Reported (job ${report.job_id}, not_executed). ${verdictText(verdict)}`
    );
  } catch (err) {
    notify(
      "scp2317-report",
      Office.MailboxEnums.ItemNotificationMessageType.ErrorMessage,
      `SCP-2317 report failed: ${String(err)}`
    );
  } finally {
    event.completed();
  }
}

async function onCheckStatus(event: Office.AddinCommands.Event) {
  const healthy = await checkApiHealth();
  notify(
    "scp2317-status",
    healthy
      ? Office.MailboxEnums.ItemNotificationMessageType.InformationalMessage
      : Office.MailboxEnums.ItemNotificationMessageType.ErrorMessage,
    healthy
      ? `SCP-2317 API is reachable at ${API_BASE}.`
      : `SCP-2317 API is unreachable at ${API_BASE}.`
  );
  event.completed();
}

function onShowAbout(event: Office.AddinCommands.Event) {
  notify(
    "scp2317-about",
    Office.MailboxEnums.ItemNotificationMessageType.InformationalMessage,
    "SCP-2317 Email Scanner 0.2.0 - heuristic MVP. Attachments are never executed."
  );
  event.completed();
}

Office.onReady(() => {
  Office.actions.associate("onQuickScan", onQuickScan);
  Office.actions.associate("onReportSuspicious", onReportSuspicious);
  Office.actions.associate("onCheckStatus", onCheckStatus);
  Office.actions.associate("onShowAbout", onShowAbout);
});

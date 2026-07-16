//! Portable logic shared by the COM add-in: request shape, URL extraction,
//! and the HTTP call to the SCP-2317 Go API. Nothing here touches COM, so it
//! builds and unit-tests on any host OS, matching the contract in
//! `api/openapi.yaml` and `outlook-addin/src/scpApi.ts`.

use serde::{Deserialize, Serialize};
use std::collections::HashSet;

pub const DEFAULT_API_BASE: &str = "http://127.0.0.1:8080";

#[derive(Debug, Clone, Serialize)]
pub struct Attachment {
    pub name: String,
    pub content_type: String,
}

#[derive(Debug, Clone, Serialize)]
pub struct EmailScanRequest {
    pub message_id: String,
    pub sender: String,
    pub subject: String,
    pub body_preview: String,
    pub urls: Vec<String>,
    pub attachments: Vec<Attachment>,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct Finding {
    #[serde(default)]
    pub engine: String,
    #[serde(default)]
    pub severity: String,
    #[serde(default)]
    pub summary: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct Verdict {
    pub status: String,
    pub score: i64,
    #[serde(default)]
    pub findings: Vec<Finding>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct SandboxSubmitResponse {
    pub job_id: String,
}

#[derive(Debug)]
pub enum ScanError {
    Http(String),
    Decode(String),
}

impl std::fmt::Display for ScanError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ScanError::Http(m) => write!(f, "request failed: {m}"),
            ScanError::Decode(m) => write!(f, "response decode failed: {m}"),
        }
    }
}

/// Mirrors `outlook-addin/src/scpApi.ts#extractUrls`: a plain `http(s)://`
/// scanner (not a full RFC 3986 parser) is enough for the heuristic MVP.
pub fn extract_urls(text: &str) -> Vec<String> {
    let mut out = Vec::new();
    let mut seen = HashSet::new();
    for scheme in ["http://", "https://"] {
        let mut search_from = 0usize;
        while let Some(rel) = text[search_from..].find(scheme) {
            let begin = search_from + rel;
            let rest = &text[begin..];
            let end = rest
                .find(|c: char| c.is_whitespace() || matches!(c, '<' | '>' | '"' | '\''))
                .unwrap_or(rest.len());
            let mut url = &rest[..end];
            while let Some(last) = url.chars().last() {
                if ".,;:)".contains(last) {
                    url = &url[..url.len() - last.len_utf8()];
                } else {
                    break;
                }
            }
            if !url.is_empty() && seen.insert(url.to_string()) {
                out.push(url.to_string());
            }
            search_from = begin + scheme.len();
        }
    }
    out
}

pub fn build_request(
    message_id: &str,
    sender: &str,
    subject: &str,
    body_preview: &str,
    attachments: Vec<Attachment>,
) -> EmailScanRequest {
    EmailScanRequest {
        message_id: message_id.to_string(),
        sender: sender.to_string(),
        subject: subject.to_string(),
        urls: extract_urls(body_preview),
        body_preview: body_preview.to_string(),
        attachments,
    }
}

pub fn verdict_text(v: &Verdict) -> String {
    format!("SCP-2317: {} (score {})", v.status.to_uppercase(), v.score)
}

pub fn call_scan_api(base_url: &str, req: &EmailScanRequest) -> Result<Verdict, ScanError> {
    let url = format!("{}/v1/email/scan", base_url.trim_end_matches('/'));
    let resp = ureq::post(&url)
        .set("content-type", "application/json")
        .send_string(&serde_json::to_string(req).map_err(|e| ScanError::Decode(e.to_string()))?)
        .map_err(|e| ScanError::Http(e.to_string()))?;
    resp.into_json::<Verdict>()
        .map_err(|e| ScanError::Decode(e.to_string()))
}

/// Escalates a message to the sandbox stub (see `internal/sandbox`). This
/// never executes the sample; it only records a `not_executed` job.
pub fn report_suspicious(
    base_url: &str,
    message_id: &str,
) -> Result<SandboxSubmitResponse, ScanError> {
    let url = format!("{}/v1/sandbox/submit", base_url.trim_end_matches('/'));
    let body = serde_json::json!({
        "hash": message_id,
        "objectRef": format!("outlook:{message_id}"),
        "policy": "user-reported",
    });
    let resp = ureq::post(&url)
        .set("content-type", "application/json")
        .send_string(&body.to_string())
        .map_err(|e| ScanError::Http(e.to_string()))?;
    resp.into_json::<SandboxSubmitResponse>()
        .map_err(|e| ScanError::Decode(e.to_string()))
}

pub fn check_api_health(base_url: &str) -> bool {
    let url = format!("{}/healthz", base_url.trim_end_matches('/'));
    ureq::get(&url)
        .call()
        .map(|r| r.status() == 200)
        .unwrap_or(false)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::{Read, Write};
    use std::net::TcpListener;

    #[test]
    fn extract_urls_dedupes_and_trims_trailing_punctuation() {
        let text = "See https://example.com/a, and https://example.com/a. Also http://x.test/b)";
        let urls = extract_urls(text);
        assert_eq!(urls, vec!["https://example.com/a", "http://x.test/b"]);
    }

    #[test]
    fn extract_urls_returns_empty_for_plain_text() {
        assert!(extract_urls("no links here").is_empty());
    }

    #[test]
    fn build_request_populates_urls_from_body() {
        let req = build_request(
            "id-1",
            "sender@example.com",
            "subject",
            "click http://evil.test/x now",
            vec![],
        );
        assert_eq!(req.urls, vec!["http://evil.test/x"]);
        assert_eq!(req.message_id, "id-1");
    }

    #[test]
    fn verdict_text_formats_status_and_score() {
        let v = Verdict {
            status: "suspicious".into(),
            score: 42,
            findings: vec![],
        };
        assert_eq!(verdict_text(&v), "SCP-2317: SUSPICIOUS (score 42)");
    }

    /// Minimal single-shot HTTP/1.1 server used only to exercise
    /// `call_scan_api` without adding a mocking dependency.
    fn spawn_fake_server(response_json: &'static str) -> String {
        let listener = TcpListener::bind("127.0.0.1:0").expect("bind loopback");
        let addr = listener.local_addr().expect("local addr");
        std::thread::spawn(move || {
            if let Ok((mut stream, _)) = listener.accept() {
                let mut buf = [0u8; 4096];
                let _ = stream.read(&mut buf);
                let body = response_json.as_bytes();
                let resp = format!(
                    "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
                    body.len()
                );
                let _ = stream.write_all(resp.as_bytes());
                let _ = stream.write_all(body);
            }
        });
        format!("http://{addr}")
    }

    #[test]
    fn call_scan_api_parses_verdict_from_response() {
        let base = spawn_fake_server(
            r#"{"status":"malicious","score":90,"findings":[{"engine":"heuristic","severity":"high","summary":"macro"}]}"#,
        );
        let req = build_request("id-2", "a@b.test", "subj", "body", vec![]);
        let verdict = call_scan_api(&base, &req).expect("call succeeds");
        assert_eq!(verdict.status, "malicious");
        assert_eq!(verdict.score, 90);
        assert_eq!(verdict.findings.len(), 1);
    }
}

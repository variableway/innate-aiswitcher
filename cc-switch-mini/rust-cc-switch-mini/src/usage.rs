use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UsageEntry {
    pub timestamp: i64,
    pub provider: String,
    pub session_id: String,
    pub model: Option<String>,
    pub input_tokens: Option<u64>,
    pub output_tokens: Option<u64>,
    pub success: bool,
}

#[derive(Debug, Default)]
pub struct UsageLog {
    entries: Vec<UsageEntry>,
    /// 磁盘持久化路径，每条记录追加写入
    persist_path: Option<String>,
}

impl UsageLog {
    pub fn with_persist(path: String) -> Self {
        let mut log = Self { entries: vec![], persist_path: Some(path) };
        log.load_from_disk();
        log
    }

    pub fn record(
        &mut self,
        provider: &str,
        session_id: &str,
        model: Option<&str>,
        input_tokens: Option<u64>,
        output_tokens: Option<u64>,
        success: bool,
    ) {
        let entry = UsageEntry {
            timestamp: chrono::Utc::now().timestamp(),
            provider: provider.to_string(),
            session_id: session_id.to_string(),
            model: model.map(|s| s.to_string()),
            input_tokens,
            output_tokens,
            success,
        };
        self.entries.push(entry.clone());
        self.append_to_disk(&entry);
    }

    pub fn summary(&self) -> UsageSummary {
        let mut s = UsageSummary::default();
        for e in &self.entries {
            s.requests += 1;
            if e.success { s.successes += 1; } else { s.failures += 1; }
            s.input_tokens += e.input_tokens.unwrap_or(0);
            s.output_tokens += e.output_tokens.unwrap_or(0);
        }
        s
    }

    pub fn per_session(&self) -> Vec<SessionUsage> {
        let mut map: HashMap<String, SessionUsage> = HashMap::new();
        for e in &self.entries {
            let su = map.entry(e.session_id.clone()).or_insert_with(|| SessionUsage {
                session_id: e.session_id.clone(),
                provider: e.provider.clone(),
                ..Default::default()
            });
            su.requests += 1;
            su.input_tokens += e.input_tokens.unwrap_or(0);
            su.output_tokens += e.output_tokens.unwrap_or(0);
        }
        let mut result: Vec<_> = map.into_values().collect();
        result.sort_by(|a, b| b.requests.cmp(&a.requests));
        result
    }

    pub fn per_provider(&self) -> Vec<ProviderUsage> {
        let mut map: HashMap<String, ProviderUsage> = HashMap::new();
        for e in &self.entries {
            let pu = map.entry(e.provider.clone()).or_insert_with(|| ProviderUsage {
                provider: e.provider.clone(),
                ..Default::default()
            });
            pu.requests += 1;
            pu.input_tokens += e.input_tokens.unwrap_or(0);
            pu.output_tokens += e.output_tokens.unwrap_or(0);
        }
        let mut result: Vec<_> = map.into_values().collect();
        result.sort_by(|a, b| b.requests.cmp(&a.requests));
        result
    }

    fn append_to_disk(&self, entry: &UsageEntry) {
        if let Some(ref path) = self.persist_path {
            if let Ok(json) = serde_json::to_string(entry) {
                let _ = std::fs::OpenOptions::new()
                    .create(true).append(true)
                    .open(path)
                    .and_then(|mut f| {
                        use std::io::Write;
                        writeln!(f, "{json}")
                    });
            }
        }
    }

    fn load_from_disk(&mut self) {
        if let Some(ref path) = self.persist_path {
            if let Ok(content) = std::fs::read_to_string(path) {
                for line in content.lines() {
                    if let Ok(entry) = serde_json::from_str::<UsageEntry>(line) {
                        self.entries.push(entry);
                    }
                }
            }
        }
    }
}

#[derive(Debug, Default, Serialize)]
pub struct UsageSummary {
    pub requests: u64,
    pub successes: u64,
    pub failures: u64,
    pub input_tokens: u64,
    pub output_tokens: u64,
}

#[derive(Debug, Default, Serialize)]
pub struct SessionUsage {
    pub session_id: String,
    pub provider: String,
    pub requests: u64,
    pub input_tokens: u64,
    pub output_tokens: u64,
}

#[derive(Debug, Default, Serialize)]
pub struct ProviderUsage {
    pub provider: String,
    pub requests: u64,
    pub input_tokens: u64,
    pub output_tokens: u64,
}

/// 从请求中提取 Session ID
pub fn extract_session_id(
    headers: &axum::http::HeaderMap,
    body: &serde_json::Value,
    path: &str,
) -> String {
    // Claude Code: x-claude-code-session-id header 或 metadata.user_id
    for name in &["x-claude-code-session-id", "claude-code-session-id"] {
        if let Some(v) = headers.get(*name).and_then(|h| h.to_str().ok()) {
            if !v.is_empty() {
                return v.to_string();
            }
        }
    }

    // Claude: metadata.user_id (格式: user_xxx_session_yyy)
    if path.contains("/v1/messages") {
        if let Some(uid) = body["metadata"]["user_id"].as_str() {
            if let Some(session_part) = uid.rsplit("_session_").next() {
                return session_part.to_string();
            }
        }
        if let Some(sid) = body["metadata"]["session_id"].as_str() {
            return sid.to_string();
        }
    }

    // Codex: session_id header 或 previous_response_id
    for name in &["session_id", "x-session-id"] {
        if let Some(v) = headers.get(*name).and_then(|h| h.to_str().ok()) {
            if !v.is_empty() && v.len() > 5 {
                return v.to_string();
            }
        }
    }

    // fallback: 用路径 + IP 做个简单区分
    format!("anon-{}", uuid::Uuid::new_v4().to_string().split('-').next().unwrap_or("x"))
}

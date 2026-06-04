//! 最简 Anthropic ↔ OpenAI 格式转换
//!
//! 透传时不做任何事，仅在 provider 标记 needs_transform=true 时启用。

use serde_json::{json, Value};

/// 检测是否需要转换：provider 的 base_url 包含 openrouter/groq/openai 等
pub fn needs_transform(base_url: &str) -> bool {
    let lower = base_url.to_lowercase();
    lower.contains("openrouter")
        || lower.contains("groq.com")
        || lower.contains("api.openai.com")
}

/// Anthropic Messages API 请求 → OpenAI Chat Completions 请求
pub fn anthropic_to_openai(body: &Value) -> Value {
    let model = body["model"].as_str().unwrap_or("gpt-4o");
    let max_tokens = body["max_tokens"].as_u64().unwrap_or(4096);
    let system = body["system"].as_str().unwrap_or("");

    // 转换 messages
    let mut messages: Vec<Value> = vec![];
    if let Some(anthropic_msgs) = body["messages"].as_array() {
        for msg in anthropic_msgs {
            let role = msg["role"].as_str().unwrap_or("user");
            let content = extract_text_content(msg);

            // Anthropic tool_use → OpenAI tool_calls
            if role == "assistant" {
                let mut m = json!({"role": role, "content": content});
                // 提取 tool_calls
                if let Some(tool_uses) = extract_tool_uses(msg) {
                    m["tool_calls"] = tool_uses;
                }
                messages.push(m);
            } else {
                let mut m = json!({"role": role, "content": content});
                // 提取 tool results
                if let Some(tool_results) = extract_tool_results(msg) {
                    m["tool_call_id"] = json!(tool_results.id);
                    m["content"] = tool_results.content;
                    m["role"] = json!("tool");
                }
                messages.push(m);
            }
        }
    }

    // system message 前置
    if !system.is_empty() {
        messages.insert(0, json!({"role": "system", "content": system}));
    }

    let mut openai_body = json!({
        "model": model,
        "messages": messages,
        "max_tokens": max_tokens,
    });

    if body["stream"].as_bool() == Some(true) {
        openai_body["stream"] = json!(true);
    }

    openai_body
}

/// OpenAI Chat Completion 响应 → Anthropic Message 响应
pub fn openai_to_anthropic(body: &Value) -> Value {
    let id = body["id"].as_str().unwrap_or("msg_001");

    let choice = &body["choices"][0];
    let message = &choice["message"];
    let content_text = message["content"].as_str().unwrap_or("");

    let mut anthropic = json!({
        "id": id,
        "type": "message",
        "role": "assistant",
        "model": body["model"],
        "content": [{"type": "text", "text": content_text}],
        "stop_reason": choice["finish_reason"].as_str().unwrap_or("end_turn"),
    });

    // 带上 usage
    if let Some(usage) = body.get("usage") {
        let input_tokens = usage["prompt_tokens"].as_u64().unwrap_or(0);
        let output_tokens = usage["completion_tokens"].as_u64().unwrap_or(0);
        anthropic["usage"] = json!({
            "input_tokens": input_tokens,
            "output_tokens": output_tokens,
        });
    }

    anthropic
}

/// OpenAI SSE chunk → Anthropic SSE event（预留流式转换）
#[allow(dead_code)]
pub fn openai_sse_to_anthropic_sse(line: &str) -> Option<String> {
    let data = line.strip_prefix("data: ")?;
    if data == "[DONE]" {
        return Some("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n".to_string());
    }
    let chunk: Value = serde_json::from_str(data).ok()?;
    let choice = &chunk["choices"][0];
    let delta = &choice["delta"];
    let content = delta["content"].as_str().unwrap_or("");

    if content.is_empty() {
        return None;
    }

    let event = json!({
        "type": "content_block_delta",
        "index": 0,
        "delta": {
            "type": "text_delta",
            "text": content
        }
    });

    Some(format!(
        "event: content_block_delta\ndata: {}\n\n",
        serde_json::to_string(&event).unwrap()
    ))
}

// ── helpers ──

fn extract_text_content(msg: &Value) -> Value {
    if let Some(text) = msg["content"].as_str() {
        return json!(text);
    }
    if let Some(blocks) = msg["content"].as_array() {
        let texts: Vec<String> = blocks
            .iter()
            .filter_map(|b| b["text"].as_str())
            .map(|s| s.to_string())
            .collect();
        return json!(texts.join("\n"));
    }
    json!("")
}

struct ToolCallInfo {
    id: String,
    content: Value,
}

fn extract_tool_uses(msg: &Value) -> Option<Value> {
    let blocks = msg["content"].as_array()?;
    let mut tool_calls: Vec<Value> = vec![];
    for (i, block) in blocks.iter().enumerate() {
        if block["type"].as_str() == Some("tool_use") {
            tool_calls.push(json!({
                "id": block["id"],
                "type": "function",
                "function": {
                    "name": block["name"],
                    "arguments": serde_json::to_string(&block["input"]).unwrap_or_default(),
                }
            }));
            // Only return first tool call for simplicity
            if i == 0 {
                return Some(json!([tool_calls.remove(0)]));
            }
        }
    }
    if tool_calls.is_empty() { None } else { Some(json!(tool_calls)) }
}

fn extract_tool_results(msg: &Value) -> Option<ToolCallInfo> {
    let blocks = msg["content"].as_array()?;
    for block in blocks {
        if block["type"].as_str() == Some("tool_result") {
            return Some(ToolCallInfo {
                id: block["tool_use_id"].as_str().unwrap_or("").to_string(),
                content: block["content"].clone(),
            });
        }
    }
    None
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_basic_anthropic_to_openai() {
        let input = json!({
            "model": "claude-sonnet-4-20250514",
            "max_tokens": 1024,
            "system": "You are helpful",
            "messages": [
                {"role": "user", "content": "Hello"}
            ]
        });
        let output = anthropic_to_openai(&input);
        assert_eq!(output["messages"][0]["role"], "system");
        assert_eq!(output["messages"][1]["role"], "user");
        assert_eq!(output["model"], "claude-sonnet-4-20250514");
    }

    #[test]
    fn test_basic_openai_to_anthropic() {
        let input = json!({
            "id": "chatcmpl-123",
            "model": "gpt-4o",
            "choices": [{
                "index": 0,
                "message": {"role": "assistant", "content": "Hi there!"},
                "finish_reason": "stop"
            }],
            "usage": {"prompt_tokens": 10, "completion_tokens": 5}
        });
        let output = openai_to_anthropic(&input);
        assert_eq!(output["type"], "message");
        assert_eq!(output["content"][0]["text"], "Hi there!");
        assert_eq!(output["usage"]["input_tokens"], 10);
    }
}

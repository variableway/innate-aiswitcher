use crate::circuit_breaker::get_or_create_breaker;
use crate::state::ProxyState;
use crate::transform;
use crate::usage;
use axum::body::Bytes;
use axum::extract::State;
use axum::http::{header, HeaderMap, HeaderValue, Method, StatusCode};
use axum::response::{IntoResponse, Response};
use axum::routing::{any, get};
use axum::{Json, Router};
use http_body_util::Full;
use hyper::body::Incoming;
use serde_json::Value;

pub fn build_router(state: ProxyState) -> Router {
    Router::new()
        .route("/", get(|| async { "cc-switch-mini proxy\n/routes: /health /usage /usage/sessions\n" }))
        .route("/health", get(health))
        .route("/usage", get(get_usage))
        .route("/usage/sessions", get(get_session_usage))
        .route("/v1/{*path}", any(proxy_handler))
        .route("/v1beta/{*path}", any(proxy_handler))
        .with_state(state)
}

async fn health() -> &'static str {
    "ok"
}

async fn get_usage(State(state): State<ProxyState>) -> impl IntoResponse {
    let log = state.usage_log.read().await;
    Json(serde_json::json!({
        "summary": log.summary(),
        "per_provider": log.per_provider(),
    }))
}

async fn get_session_usage(State(state): State<ProxyState>) -> impl IntoResponse {
    let log = state.usage_log.read().await;
    Json(serde_json::json!({
        "sessions": log.per_session(),
    }))
}

async fn proxy_handler(
    State(state): State<ProxyState>,
    method: Method,
    uri: axum::http::Uri,
    headers: HeaderMap,
    body: Bytes,
) -> Result<Response, (StatusCode, String)> {
    let path = uri.path();
    let query = uri.query().map(|q| format!("?{q}")).unwrap_or_default();
    let total = state.config.providers.len();

    let original_body: Value = serde_json::from_slice(&body)
        .map_err(|e| (StatusCode::BAD_REQUEST, format!("Invalid JSON: {e}")))?;

    let session_id = usage::extract_session_id(&headers, &original_body, path);
    let model = original_body["model"].as_str().map(|s| s.to_string());
    let mut attempt: usize = 0;

    loop {
        let provider = &state.config.providers[attempt];
        let key = format!("{}:{}", provider.base_url, provider.name);

        let breaker = get_or_create_breaker(&state.breakers, &key).await;
        if !breaker.is_available().await {
            tracing::warn!("[{}/{}] {} is OPEN (session={})", attempt + 1, total, provider.name, &session_id[..8.min(session_id.len())]);
            attempt += 1;
            if attempt >= total {
                return Err((StatusCode::SERVICE_UNAVAILABLE, r#"{"error":"all providers unavailable"}"#.to_string()));
            }
            continue;
        }

        let converting = transform::needs_transform(&provider.base_url);
        let (req_body, target_path) = if converting {
            let openai_body = transform::anthropic_to_openai(&original_body);
            (serde_json::to_vec(&openai_body).unwrap_or_default(), "/v1/chat/completions")
        } else {
            (body.to_vec(), path)
        };

        let upstream_url = format!("{}{target_path}{query}", provider.base_url);
        tracing::info!("[{}/{}] {} {} → {} (session={}){}",
            attempt + 1, total, method, path, provider.name,
            &session_id[..8.min(session_id.len())],
            if converting { " [conv]" } else { "" });

        match build_upstream_request(&method, &upstream_url, &headers, &provider.api_key, &req_body).await {
            Ok(req) => match state.http_client.request(req).await {
                Ok(resp) => {
                    breaker.record_success().await;
                    let status = resp.status();
                    state.usage_log.write().await.record(&provider.name, &session_id, model.as_deref(), None, None, true);
                    tracing::info!("[{}/{}] {} returned {}", attempt + 1, total, provider.name, status);

                    if converting {
                        return Ok(convert_and_respond(resp).await);
                    }
                    return Ok(convert_response(resp));
                }
                Err(e) => {
                    breaker.record_failure().await;
                    state.usage_log.write().await.record(&provider.name, &session_id, model.as_deref(), None, None, false);
                    tracing::error!("[{}/{}] {} FAILED: {e}", attempt + 1, total, provider.name);
                }
            },
            Err(e) => tracing::error!("[{}/{}] Build request error: {e}", attempt + 1, total),
        }

        attempt += 1;
        if attempt >= total {
            return Err((StatusCode::BAD_GATEWAY, format!(r#"{{"error":"all {total} providers failed"}}"#)));
        }
    }
}

async fn build_upstream_request(
    method: &Method, url: &str, original_headers: &HeaderMap, api_key: &str, body: &[u8],
) -> Result<hyper::Request<Full<bytes::Bytes>>, String> {
    let uri: hyper::Uri = url.parse().map_err(|e| format!("bad url: {e}"))?;
    let mut req_builder = hyper::Request::builder().method(method).uri(uri);
    for (k, v) in original_headers.iter() {
        let low = k.as_str().to_lowercase();
        if low == "host" || low == "authorization" || low.starts_with("x-api") || low == "content-length" {
            continue;
        }
        req_builder = req_builder.header(k, v);
    }
    req_builder = req_builder.header(header::AUTHORIZATION, HeaderValue::from_str(&format!("Bearer {api_key}")).map_err(|e| format!("bad header: {e}"))?);
    req_builder.body(Full::new(bytes::Bytes::from(body.to_vec()))).map_err(|e| format!("build request: {e}"))
}

async fn convert_and_respond(resp: hyper::Response<Incoming>) -> Response {
    use http_body_util::BodyExt;
    let status = resp.status();
    if let Ok(full_body) = resp.collect().await {
        if let Ok(openai_resp) = serde_json::from_slice::<Value>(&full_body.to_bytes()) {
            let anthropic_resp = transform::openai_to_anthropic(&openai_resp);
            let converted = serde_json::to_vec(&anthropic_resp).unwrap_or_default();
            return Response::builder().status(200).header("content-type", "application/json")
                .body(axum::body::Body::from(converted)).unwrap();
        }
    }
    Response::builder().status(status).body(axum::body::Body::empty()).unwrap()
}

fn convert_response(resp: hyper::Response<Incoming>) -> Response {
    let status = resp.status();
    let mut response = Response::builder().status(status);
    let h = response.headers_mut().unwrap();
    for (k, v) in resp.headers().iter() {
        if k.as_str().to_lowercase() == "transfer-encoding" { continue; }
        h.insert(k, v.clone());
    }
    response.body(axum::body::Body::new(resp.into_body())).unwrap()
}

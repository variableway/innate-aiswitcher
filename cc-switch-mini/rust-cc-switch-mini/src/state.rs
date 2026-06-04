use crate::circuit_breaker::BreakerMap;
use crate::config::Config;
use crate::usage::UsageLog;
use bytes::Bytes;
use http_body_util::Full;
use hyper_util::client::legacy::connect::HttpConnector;
use hyper_util::client::legacy::Client;
use hyper_util::rt::TokioExecutor;
use std::sync::Arc;
use tokio::sync::RwLock;

pub type ProxyState = Arc<AppState>;

pub struct AppState {
    pub config: Config,
    pub breakers: BreakerMap,
    pub http_client: Client<HttpConnector, Full<Bytes>>,
    pub usage_log: Arc<RwLock<UsageLog>>,
}

pub async fn build_http_client() -> Client<HttpConnector, Full<Bytes>> {
    let mut connector = HttpConnector::new();
    connector.set_nodelay(true);
    connector.set_keepalive(Some(std::time::Duration::from_secs(30)));
    Client::builder(TokioExecutor::new())
        .pool_idle_timeout(std::time::Duration::from_secs(90))
        .pool_max_idle_per_host(5)
        .build(connector)
}

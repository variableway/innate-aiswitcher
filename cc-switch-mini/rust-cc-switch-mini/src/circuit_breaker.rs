use std::fmt;
use std::sync::atomic::{AtomicU32, Ordering};
use std::sync::Arc;
use std::time::Instant;
use tokio::sync::RwLock;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) enum CircuitState {
    Closed,
    Open,
    HalfOpen,
}

impl fmt::Display for CircuitState {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            CircuitState::Closed => write!(f, "closed"),
            CircuitState::Open => write!(f, "open"),
            CircuitState::HalfOpen => write!(f, "half_open"),
        }
    }
}

pub(crate) struct CircuitBreaker {
    state: RwLock<CircuitState>,
    consecutive_failures: AtomicU32,
    consecutive_successes: AtomicU32,
    last_opened_at: RwLock<Option<Instant>>,
    failure_threshold: u32,
    success_threshold: u32,
    timeout_secs: u64,
}

impl CircuitBreaker {
    pub(crate) fn new(failure_threshold: u32, success_threshold: u32, timeout_secs: u64) -> Self {
        Self {
            state: RwLock::new(CircuitState::Closed),
            consecutive_failures: AtomicU32::new(0),
            consecutive_successes: AtomicU32::new(0),
            last_opened_at: RwLock::new(None),
            failure_threshold,
            success_threshold,
            timeout_secs,
        }
    }

    pub(crate) async fn is_available(&self) -> bool {
        let state = *self.state.read().await;
        match state {
            CircuitState::Closed | CircuitState::HalfOpen => true,
            CircuitState::Open => {
                if let Some(opened_at) = *self.last_opened_at.read().await {
                    if opened_at.elapsed().as_secs() >= self.timeout_secs {
                        *self.state.write().await = CircuitState::HalfOpen;
                        tracing::info!("Circuit breaker: Open → HalfOpen (timeout)");
                        return true;
                    }
                }
                false
            }
        }
    }

    pub(crate) async fn record_success(&self) {
        self.consecutive_failures.store(0, Ordering::SeqCst);
        let state = *self.state.read().await;
        if state == CircuitState::HalfOpen {
            let count = self.consecutive_successes.fetch_add(1, Ordering::SeqCst) + 1;
            if count >= self.success_threshold {
                *self.state.write().await = CircuitState::Closed;
                self.consecutive_successes.store(0, Ordering::SeqCst);
                tracing::info!("Circuit breaker: HalfOpen → Closed (recovered)");
            }
        }
    }

    pub(crate) async fn record_failure(&self) {
        let state = *self.state.read().await;
        match state {
            CircuitState::Closed => {
                let count = self.consecutive_failures.fetch_add(1, Ordering::SeqCst) + 1;
                if count >= self.failure_threshold {
                    *self.state.write().await = CircuitState::Open;
                    *self.last_opened_at.write().await = Some(Instant::now());
                    tracing::info!(
                        "Circuit breaker: Closed → Open ({} consecutive failures)",
                        count
                    );
                }
            }
            CircuitState::HalfOpen => {
                *self.state.write().await = CircuitState::Open;
                *self.last_opened_at.write().await = Some(Instant::now());
                self.consecutive_successes.store(0, Ordering::SeqCst);
                tracing::info!("Circuit breaker: HalfOpen → Open (probe failed)");
            }
            CircuitState::Open => {}
        }
    }
}

pub(crate) type BreakerMap = Arc<RwLock<std::collections::HashMap<String, Arc<CircuitBreaker>>>>;

pub(crate) async fn get_or_create_breaker(
    breakers: &BreakerMap,
    key: &str,
) -> Arc<CircuitBreaker> {
    {
        let map = breakers.read().await;
        if let Some(b) = map.get(key) {
            return b.clone();
        }
    }
    let mut map = breakers.write().await;
    if let Some(b) = map.get(key) {
        return b.clone();
    }
    let b = Arc::new(CircuitBreaker::new(4, 2, 60));
    map.insert(key.to_string(), b.clone());
    b
}

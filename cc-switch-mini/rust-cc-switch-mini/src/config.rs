use anyhow::Result;
use serde::{Deserialize, Serialize};
use std::path::Path;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderConfig {
    pub name: String,
    pub base_url: String,
    pub api_key: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub listen_addr: String,
    pub listen_port: u16,
    pub providers: Vec<ProviderConfig>,
}

impl Config {
    pub fn from_file(path: &str) -> Result<Self> {
        let content = std::fs::read_to_string(path)?;
        let config: Config = serde_json::from_str(&content)?;
        Ok(config)
    }

    pub fn default_path() -> String {
        let home = std::env::var("HOME")
            .or_else(|_| std::env::var("USERPROFILE"))
            .unwrap_or_else(|_| ".".to_string());
        format!("{home}/.cc-switch-mini.json")
    }
}

pub fn write_default_config(path: &str) -> Result<()> {
    let config = Config {
        listen_addr: "127.0.0.1".to_string(),
        listen_port: 15721,
        providers: vec![
            ProviderConfig {
                name: "MiniMax".to_string(),
                base_url: "https://api.minimax.chat".to_string(),
                api_key: "sk-your-minimax-key".to_string(),
            },
            ProviderConfig {
                name: "OpenAI".to_string(),
                base_url: "https://api.openai.com".to_string(),
                api_key: "sk-your-openai-key".to_string(),
            },
        ],
    };
    if Path::new(path).exists() {
        tracing::info!("Config file already exists at {path}, skipping write");
        return Ok(());
    }
    let content = serde_json::to_string_pretty(&config)?;
    std::fs::write(path, content)?;
    tracing::info!("Default config written to {path}");
    Ok(())
}

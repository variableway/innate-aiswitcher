mod circuit_breaker;
mod config;
mod proxy;
mod state;
mod templates;
mod terminal;
mod tui;
mod transform;
mod usage;

use anyhow::Result;
use clap::{Parser, Subcommand};
use std::net::SocketAddr;
use std::sync::Arc;
use state::AppState;
use tracing_subscriber::EnvFilter;

#[derive(Parser, Debug)]
#[command(name = "cc-switch-mini")]
#[command(about = "Minimal LLM provider proxy with failover, TUI select, templates, and per-project .cc-switch-env")]
struct Args {
    #[command(subcommand)]
    command: Option<Commands>,

    #[arg(short, long, default_value_t = String::from("127.0.0.1"))]
    addr: String,

    #[arg(short, long, default_value_t = 15721)]
    port: u16,

    #[arg(short, long, default_value = "")]
    config: String,
}

#[derive(Subcommand, Debug)]
enum Commands {
    Proxy,
    Launch {
        #[arg(short, long)]
        provider: String,
        #[arg(short, long, default_value = "claude")]
        app: String,
        #[arg(short, long, default_value = "ghostty")]
        terminal: String,
        #[arg(long)]
        cwd: Option<String>,
        #[arg(trailing_var_arg = true, allow_hyphen_values = true)]
        args: Vec<String>,
    },
    Select,
    Templates,
}

fn find_provider(config: &config::Config, name: &str) -> Option<config::ProviderConfig> {
    config.providers.iter().find(|p| p.name.to_lowercase() == name.to_lowercase()).cloned()
}

fn add_provider_to_config(cfg: &mut config::Config, name: &str, base_url: &str, api_key: &str) -> Result<()> {
    let existing = find_provider(cfg, name);
    if existing.is_some() {
        println!("Provider '{}' 已存在，更新 API Key...", name);
        cfg.providers.retain(|x| x.name.to_lowercase() != name.to_lowercase());
    }
    cfg.providers.push(config::ProviderConfig { name: name.to_string(), base_url: base_url.to_string(), api_key: api_key.to_string() });
    let config_path = config::Config::default_path();
    std::fs::write(&config_path, serde_json::to_string_pretty(cfg)?)?;
    println!("已保存到 {}", config_path);
    Ok(())
}

fn do_launch(provider: &config::ProviderConfig, app: &str, terminal: &str, cwd: Option<&str>, extra_args: &[String]) -> Result<()> {
    let agent = terminal::find_agent(app).ok_or_else(|| anyhow::anyhow!("Unknown agent: {app}"))?;
    println!();
    println!("  Agent:    {} ({})", agent.display, agent.name);
    println!("  Provider: {}", provider.name);
    println!("  URL:      {}", provider.base_url);
    println!("  Terminal: {terminal}");
    println!();

    terminal::launch_agent(agent, &provider.api_key, &provider.base_url, terminal, cwd, &std::collections::HashMap::new(), extra_args)?;
    println!("Done! {} started with provider '{}'", agent.display, provider.name);
    Ok(())
}

/// 如果当前目录或其父目录有 .cc-switch-env，自动用里面的配置启动
fn try_auto_env(cfg: &config::Config) -> Result<bool> {
    let cwd = std::env::current_dir()?.to_string_lossy().to_string();
    let Some((env_path, penv)) = terminal::find_project_env(&cwd) else {
        return Ok(false);
    };

    println!("📁 发现项目配置: {}", env_path.display());
    println!("   APP={}  PROVIDER={}  TERMINAL={}",
        penv.app.as_deref().unwrap_or("claude"),
        penv.provider.as_deref().unwrap_or("(default)"),
        penv.terminal.as_deref().unwrap_or("ghostty"));

    let app = penv.app.as_deref().unwrap_or("claude");
    let terminal_name = penv.terminal.as_deref().unwrap_or("ghostty");
    let provider_name = penv.provider.as_deref().unwrap_or("");

    let provider = if provider_name.is_empty() {
        cfg.providers.first().cloned()
            .ok_or_else(|| anyhow::anyhow!("No providers configured"))?
    } else {
        find_provider(cfg, provider_name)
            .ok_or_else(|| anyhow::anyhow!("Provider '{}' not found", provider_name))?
    };

    do_launch(&provider, app, terminal_name, Some(&cwd), &[])?;
    Ok(true)
}

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("cc_switch_mini=info")))
        .init();

    let args = Args::parse();
    let config_path = if args.config.is_empty() { config::Config::default_path() } else { args.config };
    config::write_default_config(&config_path)?;
    let mut cfg = config::Config::from_file(&config_path)?;

    let command = args.command.unwrap_or(Commands::Select);

    match command {
        Commands::Templates => {
            if let Some(t) = tui::select_template()? {
                let key = tui::prompt_api_key(&t.name)?;
                add_provider_to_config(&mut cfg, &t.name, &t.base_url, &key)?;
                println!("✅ 已添加 Provider: {}", t.name);
                let launch_now: String = dialoguer::Input::with_theme(&dialoguer::theme::ColorfulTheme::default())
                    .with_prompt("是否立即启动? (y/n)").default("y".into()).interact_text()?;
                if launch_now.to_lowercase() == "y" {
                    let p = find_provider(&cfg, &t.name).unwrap();
                    do_launch(&p, "claude", "ghostty", None, &[])?;
                }
            }
        }

        Commands::Select => {
            // 1. 先检查项目级 .cc-switch-env
            if try_auto_env(&cfg)? {
                return Ok(());
            }

            // 2. No project env → but no providers? Add one first
            if cfg.providers.is_empty() {
                println!("还没有配置 Provider，先添加一个：\n");
                if let Some(t) = tui::select_template()? {
                    let key = tui::prompt_api_key(&t.name)?;
                    add_provider_to_config(&mut cfg, &t.name, &t.base_url, &key)?;
                } else {
                    return Ok(());
                }
            }

            match tui::select_provider(&cfg)? {
                Some((provider, app, terminal, extra_args_str)) => {
                    let args_vec: Vec<String> = if extra_args_str.is_empty() { vec![] }
                        else { extra_args_str.split_whitespace().map(|s| s.to_string()).collect() };
                    do_launch(&provider, &app, &terminal, None, &args_vec)?;
                }
                None => {
                    println!("\n添加新 Provider...\n");
                    if let Some(t) = tui::select_template()? {
                        let key = tui::prompt_api_key(&t.name)?;
                        add_provider_to_config(&mut cfg, &t.name, &t.base_url, &key)?;
                    }
                }
            }
        }

        Commands::Launch { provider, app, terminal, cwd, args: extra_args } => {
            let p = find_provider(&cfg, &provider)
                .ok_or_else(|| anyhow::anyhow!("Provider '{provider}' not found"))?;
            do_launch(&p, &app, &terminal, cwd.as_deref(), &extra_args)?;
        }

        Commands::Proxy => {
            cfg.listen_addr = args.addr;
            cfg.listen_port = args.port;
            let persist = format!("{}/usage.jsonl", std::env::var("HOME").unwrap_or_else(|_| ".".into()));
            let state = Arc::new(AppState {
                config: cfg.clone(),
                breakers: Default::default(),
                http_client: state::build_http_client().await,
                usage_log: Arc::new(tokio::sync::RwLock::new(usage::UsageLog::with_persist(persist))),
            });
            let addr: SocketAddr = format!("{}:{}", cfg.listen_addr, cfg.listen_port).parse()?;
            tracing::info!("cc-switch-mini proxy on {addr}");
            tracing::info!("  Usage log: ~/usage.jsonl");
            tracing::info!("  Routes: GET /usage  GET /usage/sessions  GET /health");
            for (i, p) in cfg.providers.iter().enumerate() {
                tracing::info!("  Provider {}: {} -> {}", i + 1, p.name, p.base_url);
            }
            let listener = tokio::net::TcpListener::bind(addr).await?;
            axum::serve(listener, proxy::build_router(state)).await?;
        }
    }

    Ok(())
}

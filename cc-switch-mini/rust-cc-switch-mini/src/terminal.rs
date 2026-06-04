//! 7 种 AI Agent 的终端启动 + 项目级 .cc-switch-env 自动检测
//!
//! 项目目录下放 .cc-switch-env 定义默认 agent + provider:
//!   APP=claude
//!   PROVIDER=MiniMax
//!   TERMINAL=ghostty
//!
//! 或直接用环境变量启动任意 agent

use anyhow::Result;
use serde::Deserialize;
use std::collections::HashMap;
use std::path::PathBuf;
use std::process::Command;

// ── Agent 定义 ──────────────────────────────────────────────────

pub struct Agent {
    pub name: &'static str,
    pub binary: &'static str,
    pub display: &'static str,
    pub env_vars: &'static [(&'static str, &'static str)], // (env_key, "api_key"|"base_url")
}

pub const AGENTS: &[Agent] = &[
    Agent { name: "claude", binary: "claude", display: "Claude Code",
        env_vars: &[("ANTHROPIC_API_KEY", "api_key"), ("ANTHROPIC_BASE_URL", "base_url")] },
    Agent { name: "codex", binary: "codex", display: "Codex CLI",
        env_vars: &[("OPENAI_API_KEY", "api_key"), ("OPENAI_BASE_URL", "base_url")] },
    Agent { name: "gemini", binary: "gemini", display: "Gemini CLI",
        env_vars: &[("GEMINI_API_KEY", "api_key"), ("GOOGLE_GEMINI_BASE_URL", "base_url")] },
    Agent { name: "opencode", binary: "opencode", display: "OpenCode",
        env_vars: &[("OPENAI_API_KEY", "api_key"), ("OPENAI_BASE_URL", "base_url")] },
    Agent { name: "openclaw", binary: "openclaw", display: "OpenClaw",
        env_vars: &[("ANTHROPIC_API_KEY", "api_key"), ("ANTHROPIC_BASE_URL", "base_url"), ("OPENAI_API_KEY", "api_key")] },
    Agent { name: "hermes", binary: "hermes", display: "Hermes Agent",
        env_vars: &[("ANTHROPIC_API_KEY", "api_key"), ("ANTHROPIC_BASE_URL", "base_url")] },
    Agent { name: "claude-desktop", binary: "claude", display: "Claude Desktop",
        env_vars: &[("ANTHROPIC_API_KEY", "api_key"), ("ANTHROPIC_BASE_URL", "base_url")] },
];

pub fn find_agent(name: &str) -> Option<&'static Agent> {
    AGENTS.iter().find(|a| a.name == name)
}

// ── .cc-switch-env 项目级配置 ──────────────────────────────────

#[derive(Debug, Deserialize, Default)]
pub struct ProjectEnv {
    pub app: Option<String>,
    pub provider: Option<String>,
    pub terminal: Option<String>,
    #[serde(flatten)]
    pub extra: HashMap<String, String>,
}

pub fn find_project_env(start_dir: &str) -> Option<(PathBuf, ProjectEnv)> {
    let mut dir: PathBuf = start_dir.into();
    loop {
        let env_path = dir.join(".cc-switch-env");
        if env_path.exists() {
            match load_project_env(&env_path) {
                Ok(env) => return Some((env_path, env)),
                Err(e) => {
                    eprintln!("Warning: failed to parse {:?}: {e}", env_path);
                    return None;
                }
            }
        }
        if !dir.pop() {
            break;
        }
    }
    None
}

fn load_project_env(path: &PathBuf) -> Result<ProjectEnv> {
    let content = std::fs::read_to_string(path)?;
    let mut env = ProjectEnv::default();
    for line in content.lines() {
        let line = line.trim();
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        if let Some((k, v)) = line.split_once('=') {
            let k = k.trim().to_lowercase();
            let v = v.trim().to_string();
            if v.is_empty() {
                continue;
            }
            match k.as_str() {
                "app" => env.app = Some(v),
                "provider" => env.provider = Some(v),
                "terminal" => env.terminal = Some(v),
                _ => { env.extra.insert(k, v); }
            }
        }
    }
    Ok(env)
}

// ── 启动函数 ────────────────────────────────────────────────────

pub fn launch_agent(
    agent: &Agent,
    api_key: &str,
    base_url: &str,
    terminal: &str,
    cwd: Option<&str>,
    extra_env: &HashMap<String, String>,
    args: &[String],
) -> Result<()> {
    let mut env_parts: Vec<String> = vec![];

    for (env_key, _source) in agent.env_vars {
        let val = if *env_key == "ANTHROPIC_API_KEY" || *env_key == "OPENAI_API_KEY" || *env_key == "GEMINI_API_KEY" {
            api_key
        } else {
            base_url
        };
        env_parts.push(format!("{env_key}=\"{val}\""));
    }

    for (k, v) in extra_env {
        env_parts.push(format!("{k}=\"{v}\""));
    }

    let env_str = env_parts.join(" ");
    let bin = agent.binary;
    let arg_str = if args.is_empty() { String::new() } else { format!(" {}", args.join(" ")) };
    let cmd = format!("{env_str} {bin}{arg_str}");

    launch(terminal, &cmd, cwd)
}

fn launch(terminal: &str, command: &str, cwd: Option<&str>) -> Result<()> {
    if command.trim().is_empty() {
        anyhow::bail!("Command is empty");
    }
    match terminal {
        "terminal" => launch_macos_terminal(command, cwd),
        "iterm" => launch_iterm(command, cwd),
        "ghostty" => launch_ghostty(command, cwd),
        "kitty" => launch_kitty(command, cwd),
        _ => anyhow::bail!("Unsupported terminal: {terminal}"),
    }
}

fn launch_macos_terminal(command: &str, cwd: Option<&str>) -> Result<()> {
    let full = build_shell_command(command, cwd);
    let escaped = full.replace('\\', "\\\\").replace('"', "\\\"");
    let script = format!(r#"tell application "Terminal" to activate
tell application "Terminal" to do script "{escaped}""#);
    let status = Command::new("osascript").args(["-e", &script]).status()?;
    anyhow::ensure!(status.success(), "Terminal launch failed");
    Ok(())
}

fn launch_iterm(command: &str, cwd: Option<&str>) -> Result<()> {
    let full = build_shell_command(command, cwd);
    let escaped = full.replace('\\', "\\\\").replace('"', "\\\"");
    let script = format!(r#"tell application "iTerm" to activate
tell application "iTerm"
    create window with default profile
    tell current session of current window
        write text "{escaped}"
    end tell
end tell"#);
    let status = Command::new("osascript").args(["-e", &script]).status()?;
    anyhow::ensure!(status.success(), "iTerm launch failed");
    Ok(())
}

fn launch_ghostty(command: &str, cwd: Option<&str>) -> Result<()> {
    let shell = std::env::var("SHELL").unwrap_or_else(|_| "/bin/zsh".to_string());
    let mut args = vec!["-na".to_string(), "Ghostty".to_string(), "--args".to_string(), "--quit-after-last-window-closed=true".to_string()];
    if let Some(dir) = cwd {
        args.push(format!("--working-directory={dir}"));
    }
    args.extend_from_slice(&["-e".to_string(), shell, "-l".to_string(), "-c".to_string(), command.to_string()]);
    let status = Command::new("open").args(&args).status()?;
    anyhow::ensure!(status.success(), "Ghostty launch failed");
    Ok(())
}

fn launch_kitty(command: &str, cwd: Option<&str>) -> Result<()> {
    let full = build_shell_command(command, cwd);
    let shell = std::env::var("SHELL").unwrap_or_else(|_| "/bin/zsh".to_string());
    let status = Command::new("open").args(["-na", "kitty", "--args", "-e", &shell, "-l", "-c", &full]).status()?;
    anyhow::ensure!(status.success(), "Kitty launch failed");
    Ok(())
}

fn build_shell_command(command: &str, cwd: Option<&str>) -> String {
    match cwd {
        Some(dir) if !dir.trim().is_empty() => format!("cd \"{dir}\" && {command}"),
        _ => command.to_string(),
    }
}

use crate::config::{Config, ProviderConfig};
use crate::templates::ProviderTemplate;
use anyhow::Result;
use dialoguer::{theme::ColorfulTheme, Select};

pub fn select_provider(config: &Config) -> Result<Option<(ProviderConfig, String, String, String)>> {
    let agent_display: Vec<String> = crate::terminal::AGENTS.iter().map(|a| format!("{} ({})", a.display, a.name)).collect();
    let apps: Vec<&str> = crate::terminal::AGENTS.iter().map(|a| a.name).collect();
    let terminals = &["ghostty", "kitty", "iterm", "terminal"];

    println!();
    console::Term::stderr().clear_screen().ok();

    // ---- Step 1: Pick app ----
    let app_idx = Select::with_theme(&ColorfulTheme::default())
        .with_prompt("选择 AI 工具")
        .default(0)
        .items(&agent_display)
        .interact()?;
    let app = apps[app_idx].to_string();

    // ---- Step 2: Pick provider ----
    let names: Vec<String> = config
        .providers
        .iter()
        .map(|p| format!("{}  →  {}", p.name, p.base_url))
        .collect();
    let add_new = format!("➕ 新增 Provider... (共 {} 个预设)", crate::templates::all_templates().len());

    let mut items: Vec<String> = names.clone();
    items.push("[──────]".to_string());
    items.push(add_new);

    let p_idx = Select::with_theme(&ColorfulTheme::default())
        .with_prompt(format!("选择 Provider ({})", app))
        .default(0)
        .items(&items)
        .interact()?;

    let provider_config = if p_idx < names.len() {
        config.providers[p_idx].clone()
    } else {
        return Ok(None); // caller should handle "add new"
    };

    // ---- Step 3: Pick terminal ----
    let t_idx = Select::with_theme(&ColorfulTheme::default())
        .with_prompt("选择终端")
        .default(0)
        .items(terminals)
        .interact()?;
    let terminal = terminals[t_idx].to_string();

    // ---- Step 4: Extra args (optional) ----
    let args: String = dialoguer::Input::with_theme(&ColorfulTheme::default())
        .with_prompt("额外参数 (可选，如 --resume abc123)")
        .allow_empty(true)
        .interact_text()?;

    Ok(Some((provider_config, app, terminal, args)))
}

pub fn select_template() -> Result<Option<ProviderTemplate>> {
    let templates = crate::templates::all_templates();
    let items: Vec<String> = templates
        .iter()
        .map(|t| format!("{}  —  {}", t.name, t.description))
        .collect();

    let idx = Select::with_theme(&ColorfulTheme::default())
        .with_prompt("选择 Provider 模板")
        .default(0)
        .items(&items)
        .interact()?;

    Ok(Some(templates[idx].clone()))
}

pub fn prompt_api_key(name: &str) -> Result<String> {
    let key: String = dialoguer::Password::with_theme(&ColorfulTheme::default())
        .with_prompt(format!("输入 {} 的 API Key (输入不可见)", name))
        .interact()?;
    Ok(key)
}

use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderTemplate {
    pub name: String,
    pub base_url: String,
    pub description: String,
}

pub fn all_templates() -> Vec<ProviderTemplate> {
    vec![
        ProviderTemplate {
            name: "Anthropic 官方".into(),
            base_url: "https://api.anthropic.com".into(),
            description: "Claude 官方 API，需要海外支付".into(),
        },
        ProviderTemplate {
            name: "OpenAI 官方".into(),
            base_url: "https://api.openai.com".into(),
            description: "GPT 官方 API，需要海外支付".into(),
        },
        ProviderTemplate {
            name: "MiniMax".into(),
            base_url: "https://api.minimax.chat".into(),
            description: "国内可用，Claude/Codex/Gemini 全系中转".into(),
        },
        ProviderTemplate {
            name: "硅基流动 SiliconFlow".into(),
            base_url: "https://api.siliconflow.cn".into(),
            description: "国内可用，支持 Claude/Gemini/DeepSeek/Qwen".into(),
        },
        ProviderTemplate {
            name: "火山方舟 VolcEngine".into(),
            base_url: "https://ark.cn-beijing.volces.com/api/v3".into(),
            description: "字节跳动，支持豆包/DeepSeek/GLM".into(),
        },
        ProviderTemplate {
            name: "胜算云".into(),
            base_url: "https://api.shengsuanyun.com".into(),
            description: "Claude/GPT/Gemini 全系中转，企业级 SLA".into(),
        },
        ProviderTemplate {
            name: "CrazyRouter".into(),
            base_url: "https://api.crazyrouter.com".into(),
            description: "聚合 300+ 模型，低至 55% 官方定价".into(),
        },
        ProviderTemplate {
            name: "PackyCode".into(),
            base_url: "https://api.packyapi.com".into(),
            description: "Claude Code/Codex/Gemini 中转，稳定高效".into(),
        },
        ProviderTemplate {
            name: "AICodeMirror".into(),
            base_url: "https://api.aicodemirror.com".into(),
            description: "Claude Code 低至 3.8 折，企业级高并发".into(),
        },
        ProviderTemplate {
            name: "AWS Bedrock".into(),
            base_url: "https://bedrock-runtime.{region}.amazonaws.com".into(),
            description: "Amazon Bedrock，企业级合规部署".into(),
        },
        ProviderTemplate {
            name: "Groq".into(),
            base_url: "https://api.groq.com/openai/v1".into(),
            description: "超低延迟推理，适合开源模型".into(),
        },
        ProviderTemplate {
            name: "DeepSeek 官方".into(),
            base_url: "https://api.deepseek.com".into(),
            description: "DeepSeek 官方 API，极低成本".into(),
        },
        ProviderTemplate {
            name: "ClaudeAPI".into(),
            base_url: "https://api.claudeapi.com".into(),
            description: "Anthropic 官方 Key + AWS Bedrock 双通道".into(),
        },
        ProviderTemplate {
            name: "DMXAPI".into(),
            base_url: "https://api.dmxapi.cn".into(),
            description: "全系模型 6.8 折，GPT/Claude/Gemini".into(),
        },
        ProviderTemplate {
            name: "PatewayAI".into(),
            base_url: "https://api.pateway.ai".into(),
            description: "100% 官方直供，Token 级账单透明".into(),
        },
        ProviderTemplate {
            name: "AGoCode".into(),
            base_url: "https://api.aigocode.com".into(),
            description: "国内直连，极速响应，零封号风险".into(),
        },
        ProviderTemplate {
            name: "LemonData".into(),
            base_url: "https://api.lemondata.cc".into(),
            description: "300+ 模型，30%-70% 官方定价".into(),
        },
    ]
}

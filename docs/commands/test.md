---
title: "aisw test"
description: "Test provider connectivity and list available models."
---

# aisw test

用 Provider 的 API key 发起最小请求，验证连通性或列出可用模型。默认模型来自 Provider 的 `default_model`（不会按协议猜测），缺失时需 `--model`。

```bash
aisw test <subcommand>
```

## aisw test provider SLUG

向 Provider 发一次最小模型请求，报告状态码与结果。

```bash
aisw test provider glm
```

覆盖模型：

```bash
aisw test provider glm --model glm-5.3
```

| Flag | 说明 |
|------|------|
| `--model` | 本次测试使用的模型（留空用 Provider 的 `default_model`） |

成功返回 0 退出码；失败（非 2xx 或请求出错）返回非 0 并打印状态码。

## aisw test models SLUG

调用 Provider 的 models 端点列出可用模型。

```bash
aisw test models glm
```

## 说明

- 厂商 Provider（带 `variants`）会先解析出默认端点再测试；`requestFor`（`internal/httpcheck/check.go`）是唯一按 `api_protocol` 翻译请求形状的地方；新增协议只需在那里加分支。
- 测试使用的 endpoint override 来自 Provider 的 `endpoints`（JSON map），如 `chat_completions`、`responses`、`messages`、`models`。

## 相关

- [aisw provider](/commands/provider) - Provider 的 `default_model` / `endpoints` 来源
- [REST API](/API) - `GET /api/aisw/providers/{slug}/models`、`POST /api/aisw/providers/{slug}/test`

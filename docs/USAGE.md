# AISwitcher 使用指南

> 在线文档：[variableway.github.io/innate-aiswitcher](https://variableway.github.io/innate-aiswitcher/)

## 前置要求

- **Go 1.26+**
- **[Task](https://taskfile.dev)** (可选，推荐用于快速构建/启动)

```powershell
# 安装 Task (Windows PowerShell)
winget install go-task.task

# 或通过 Scoop
scoop install task
```

---

## 快速开始

### 方式一：使用 Task (推荐)

```bash
# 1. 构建二进制
task build

# 2. 启动 Web UI + API 服务
task serve
```

打开浏览器访问 **http://127.0.0.1:8090/** 进入配置页面。

### 方式二：直接使用 Go

```bash
# 构建
go build -o bin/aisw.exe ./cmd/aisw

# 启动 Web 服务
go run ./cmd/aisw serve --http 127.0.0.1:8090

# 或运行构建好的二进制
.\bin\aisw.exe serve --http 127.0.0.1:8090
```

启动后会看到：

```
aisw: data dir ~/.innate-aiswitcher/pb_data
aisw: bootstrapping database...
aisw: server listening on http://127.0.0.1:8090
aisw: web UI    http://127.0.0.1:8090/
aisw: REST API  http://127.0.0.1:8090/api/aisw/
```

`serve` 常用参数：

| 参数 | 说明 |
|------|------|
| `--http ADDR` | 监听地址（默认 `127.0.0.1:8090`） |
| `--quiet` | 关闭 HTTP 访问日志 |
| `--origins` | CORS 允许来源（默认 `*`） |
| `--admin-ui` | 启用 PocketBase 管理后台 `/_` |
| `--show-admin-banner` | 显示 PocketBase 启动 banner |

### 方式三：开发模式（构建 + 启动一步完成）

```bash
task dev
```

---

## Task 命令速查

| 命令 | 说明 |
|------|------|
| `task build` | 构建二进制到 `bin/aisw.exe` |
| `task serve` | 启动 API 服务 + Web UI |
| `task serve:verbose` | 启动服务并显示 PocketBase 完整日志 |
| `task dev` | 构建并启动服务（一步完成） |
| `task run` | 启动交互式 TUI |
| `task test` | 运行单元测试 |
| `task verify` | 格式化 + 检查 + 测试 + 构建 |
| `task clean` | 清理构建产物 |
| `task docs:dev` | 启动 docmd 文档站本地预览 |
| `task docs:build` | 构建静态文档到 `site/` |

---

## Web UI 功能

打开 `http://127.0.0.1:8090/` 后，可以在浏览器中：

### Providers 管理
- 查看、添加、编辑、删除 LLM Provider 配置
- **从预设导入**：内置 DeepSeek、Kimi、MiniMax、OpenAI、Anthropic、小米 MiMo、火山方舟等 7 个 Provider 预设，一键导入
- 点击 **Test** 按钮测试 Provider 连通性

### Profiles 管理
- 创建 Agent + Provider 组合
- 设置默认 Profile
- 覆盖模型选择、CLI 参数、权限设置

REST API 完整参考见 [REST API](/API)。

---

## CLI 常用命令

### Provider 管理

```bash
# 列出所有 Provider
aisw provider list

# 添加 Provider
aisw provider add deepseek \
  --name "DeepSeek" \
  --base-url https://api.deepseek.com/v1 \
  --api-key sk-xxx \
  --protocol openai_chat \
  --model deepseek-v4-flash

# 查看内置预设
aisw provider presets

# 删除 Provider
aisw provider delete deepseek
```

### Profile 管理

```bash
# 列出所有 Profile
aisw profile list

# 创建 Profile（将 deepseek provider 绑定到 claude agent）
aisw profile add claude-deepseek \
  --name "Claude + DeepSeek" \
  --agent claude \
  --provider deepseek \
  --default
```

### 配置导入/导出

```bash
# 导出当前配置到 TOML 文件
aisw config export --path ~/.innate-aiswitcher/config.toml

# 导出（包含 API Key）
aisw config export --path config.toml --include-secrets

# 从文件导入配置
aisw config import --path config.toml
```

### 测试与启动

```bash
# 测试 Provider 连接
aisw test provider deepseek

# 列出 Provider 可用模型
aisw test models deepseek

# 用指定 Provider 启动 Agent
aisw start claude deepseek
```

### 更多帮助

```bash
aisw --help
aisw provider --help
aisw serve --help
```

---

## 数据存储

所有配置数据存储在 **本地 SQLite 数据库** 中：

```
~/.innate-aiswitcher/pb_data/
```

Web UI 和 CLI 操作的是 **同一个数据库**，两者可以交替使用。

### 数据备份

```bash
# 导出全量配置（含 API Key）
aisw config export --include-secrets --path backup.toml

# 导入时自动备份
aisw config import --path backup.toml --backup
```

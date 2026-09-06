---
title: "aisw config"
description: "Import/export the shared config mirror (TOML or JSON)."
---

# aisw config

导入/导出共享配置镜像（`[[providers]]` + `[[profiles]]`），支持 TOML 与 JSON。用于备份、迁移、以及在空库上首次播种。

```bash
aisw config <subcommand>
```

## aisw config export

把 SQLite 中的 providers/profiles 导出到文件。

```bash
aisw config export --path ~/.innate-aiswitcher/config.toml
aisw config export --path config.toml --include-secrets
aisw config export --path config.json --format json
```

| Flag | 说明 |
|------|------|
| `--path` | 输出路径（默认 `~/.innate-aiswitcher/config.toml`） |
| `--include-secrets` | 包含 API key（默认脱敏） |
| `--format` | `toml`(默认) / `json` |

## aisw config dump

导出全量配置（含 secrets）到文件，默认写到 init-config 路径，便于在新机器/空库上首次播种。

```bash
aisw config dump
aisw config dump --path ~/init.toml --format json
```

| Flag | 说明 |
|------|------|
| `--path` | 输出路径（默认 `~/.innate-aiswitcher/init-config.toml`） |
| `--format` | `toml`(默认) / `json` |

## aisw config import

备份当前配置后，从 TOML/JSON 文件导入 providers/profiles。导入在 SQLite transaction 内执行，失败回滚。

```bash
aisw config import --path config.toml
aisw config import --path config.toml --backup-path ~/.innate-aiswitcher/backups/before-import.toml
aisw config import --path config.toml --no-backup
aisw config import --path config.json --format json
```

| Flag | 说明 |
|------|------|
| `--path` | 输入路径（默认 `~/.innate-aiswitcher/config.toml`） |
| `--backup` | 导入前导出含 secrets 的备份（默认开启） |
| `--no-backup` | 跳过备份 |
| `--backup-path` | 自定义备份路径 |
| `--format` | `toml` / `json` / 留空自动按内容检测 |

## aisw config template

写出内置的配置模板（`config.example.toml`）。该模板含示例 providers/profiles，且 profile 只放 per-agent 覆盖（不复述 provider 的默认模型，slug 不与 provider 冲突）。

```bash
aisw config template --path ~/.innate-aiswitcher/config.toml
```

| Flag | 说明 |
|------|------|
| `--path` | 输出路径（默认 `~/.innate-aiswitcher/config.toml`） |

## 原子写入与首次播种

- 所有 secret-bearing 配置写入（export/dump/template/init-config）都走 `internal/safefile`：同目录临时文件 + `chmod 0o600` + `Sync` + `Rename` + 目录 `Sync`。
- 首次 bootstrap 空 DB 时，若 `~/.innate-aiswitcher/init-config.toml` 存在且 providers/profiles 为空，会自动导入（`maybeInitConfig`）。可用 `--init-config` 指定别的路径。

## 相关

- [aisw provider](/commands/provider) / [aisw profile](/commands/profile) - 配置镜像的内容来源
- [aisw serve](/commands/serve) - `serve` 启动时同样走 lazy bootstrap + init-config 播种

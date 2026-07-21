# Agent Flight Recorder v0.1.0

首个 Windows x64 公开预览版，交付本地单文件 CLI 与共根 Codex / Claude Code 薄插件。

## 包含内容

- `afr-windows-amd64.exe`
- `SHA256SUMS`
- 本发布说明及仓库中的快速开始、隐私与威胁模型

## 主要能力

- 启动并记录非交互式 Agent 子进程，托管 Windows 进程树。
- 采集 Git / 非 Git 工作区 before、after、session delta 与受限 patch。
- 写盘前脱敏、确定性风险规则、事件哈希链、evidence / derived manifest。
- 生成 Markdown、JSON 和离线单文件 HTML 报告。
- `list`、`show`、`verify`、可预览且需确认的 `clean`。
- Codex marketplace plugin 与 Claude Code `--plugin-dir` 共用一个 `afr` skill。

## 外部前置与边界

Git 和目标 Agent CLI 需单独安装。AFR 不提供数字签名、远端见证、操作系统文件监控或网络监控；详见[隐私与威胁模型](./SECURITY.md)。Claude Code 的真实 skill 调用需要用户先完成 Claude 登录。

## 校验

```powershell
$expected = (Get-Content .\SHA256SUMS).Split(' ')[0]
$actual = (Get-FileHash -Algorithm SHA256 .\afr-windows-amd64.exe).Hash.ToLowerInvariant()
if ($actual -ne $expected) { throw "AFR checksum mismatch" }
.\afr-windows-amd64.exe version
```

# AFR 5 分钟快速开始（Windows）

AFR 只记录通过它启动的**新任务**，不能追溯当前或已经运行中的 Agent 会话。Git 与 Codex / Claude Code 是外部前置，不包含在 `afr.exe` 中。

## 1. 安装并校验

从 GitHub Release 下载同一版本的 `afr-windows-amd64.exe` 和 `SHA256SUMS`，放到一个已加入 `PATH` 的目录并将程序重命名为 `afr.exe`。在下载目录执行：

```powershell
$expected = (Get-Content .\SHA256SUMS).Split(' ')[0]
$actual = (Get-FileHash -Algorithm SHA256 .\afr-windows-amd64.exe).Hash.ToLowerInvariant()
if ($actual -ne $expected) { throw "AFR checksum mismatch" }
.\afr-windows-amd64.exe version
```

从源码构建时，需要 Go 1.26 和 Git：

```powershell
git clone https://github.com/liao96312/agent-flight-recorder.git
Set-Location agent-flight-recorder
.\scripts\build.ps1 -Version 0.1.0
.\dist\afr-windows-amd64.exe version
```

## 2. 记录一次新任务

在目标仓库运行非交互 Codex；独立的 `--` 是 AFR 参数与子进程 argv 的唯一边界：

```powershell
afr run --workspace (Get-Location).Path -- codex exec --cd (Get-Location).Path "只读取仓库并概括目录结构，不要修改文件"
```

Claude Code 使用打印模式：

```powershell
afr run --workspace (Get-Location).Path -- claude --print "只读取仓库并概括目录结构，不要修改文件"
```

任务结束时，AFR 会把固定六行摘要写入 stderr；child stdout 保持原样。示例中的 capability 与计数会按实际会话变化：

```text
Session: 20260721T120000.000000000Z-example
Evidence: observed=process_output,workspace_snapshot; not_observable=network_monitor,os_file_monitor; truncated=false
Changed: added=0 modified=0 deleted=0 renamed=0 pre_existing=0
Risks: total=0 highest=none
Exit: child=0 state=complete
Report: C:\Users\you\.afr\sessions\20260721T120000.000000000Z-example\report.html
```

即使子进程失败，AFR 仍保留可检查的失败会话。

## 3. 查看报告并校验

```powershell
afr list
afr show latest
afr show --open latest
afr verify latest
```

报告是离线单文件 HTML。`verify` 成功只证明本地证据链与 manifest 当前一致，不代表数字签名或远端见证。

## 4. 可选：排除扫描路径

在工作区根目录创建 `.afrignore`，可排除生成目录或大文件：

```text
# 相对根目录的路径前缀
vendor/

# Go path.Match glob；不会递归匹配子目录
*.log
generated/*.tmp
```

Windows 分隔符 `\` 会统一为 `/`。非法模式会在 child 启动前失败。它不是完整 `.gitignore`，不支持 `!` 取反、递归 `**` 或父目录配置发现。

## 5. 预览并清理

清理默认只预览，不会后台自动执行：

```powershell
afr clean --older-than 30d
```

核对列出的精确 session 后，才显式删除：

```powershell
afr clean --older-than 30d --yes
```

删除 session 目录不可由 AFR 恢复。需要分享报告时，先阅读[隐私与威胁模型](./SECURITY.md)。

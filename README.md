# Agent Flight Recorder

AFR 是一个本地优先的编码智能体飞行记录器。Windows v0.1 CLI 已形成记录、脱敏、风险分析、离线报告与完整性校验闭环，并提供共根的 Codex / Claude Code 薄插件。

```powershell
go test ./...
go build ./cmd/afr
go build ./cmd/fake-agent
./afr.exe version
./afr.exe run -- cmd.exe /d /c "echo hello"
./afr.exe list --json
./afr.exe show latest
./afr.exe verify latest --json
./afr.exe clean --older-than 30d
# Review the preview, then rerun with --yes to delete exactly those sessions.
```

会话默认写入 `~/.afr/sessions/<session-id>`。POSIX 请求目录 `0700`、文件 `0600`；Windows 依赖当前用户私有目录的继承 ACL，Go 的 POSIX mode 参数本身不构成额外 ACL 保证。

唯一非标准库依赖是官方 `golang.org/x/sys/windows`，用于 Job Object；Unix 使用标准库 `syscall` 进程组。

安全边界：当前 session、events、workspace JSON 和 patch 共用强制写盘前脱敏入口，并由 `manifest.json` 分组记录 evidence/derived 的大小和 SHA-256。`afr verify` 流式校验事件链、最终 hash 和所有制品，并定位首个异常。stdout/stderr 以有界逻辑记录处理；secret 会替换为会话级 HMAC 占位符，二进制、未知编码、超长记录和超限 Diff 只保存元数据。当前会对可见的破坏性命令、敏感信息、大规模 workspace delta、越界路径、提权请求和网络目标生成 `risk_found`，但不声称具备 OS 文件或网络监控能力。

- [5 分钟快速开始](./docs/QUICKSTART.md)
- [隐私与威胁模型](./docs/SECURITY.md)
- [v0.1.0 发布说明](./docs/RELEASE_NOTES_v0.1.0.md)
- [实现计划与 TODO](./docs/TODO.md)

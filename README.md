# Agent Flight Recorder

当前实现已完成 Windows M0 记录闭环，并进入 M1：`afr run` 现会采集 Git/非 Git 工作区 before/after、区分 pre-existing 与本次 session delta，并对文件和超限输出使用会话级 HMAC 指纹。`afr list` 会只读识别硬杀和 torn-tail 会话。

```powershell
go test ./...
go build ./cmd/afr
go build ./cmd/fake-agent
./afr.exe version
./afr.exe run -- cmd.exe /d /c "echo hello"
./afr.exe list --json
```

会话默认写入 `~/.afr/sessions/<session-id>`。POSIX 请求目录 `0700`、文件 `0600`；Windows 依赖当前用户私有目录的继承 ACL，Go 的 POSIX mode 参数本身不构成额外 ACL 保证。

唯一非标准库依赖是官方 `golang.org/x/sys/windows`，用于 Job Object；Unix 使用标准库 `syscall` 进程组。

安全边界：当前 session、events、workspace JSON 和 patch 共用强制写盘前脱敏入口。stdout/stderr 以有界逻辑记录处理；secret 会替换为会话级 HMAC 占位符，二进制、未知编码、超长记录和超限 Diff 只保存元数据。当前会对可见的破坏性命令、敏感信息和大规模 workspace delta 生成 `risk_found`；报告、验证和插件仍按 [TODO](./docs/TODO.md) 继续实现。

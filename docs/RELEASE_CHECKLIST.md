# v0.1 发布检查表

发布负责人从干净 clone 执行下列检查，记录 commit、机器、时间、命令和结果。任一必选项失败都不发布；不能自动化的宿主登录与干净 VM 项必须保留人工证据。

## 一键本地检查

Windows PowerShell、Go 1.26、Git、Codex CLI 与 Claude Code 均可用时：

```powershell
.\scripts\release-check.ps1 -Full
```

脚本执行完整 Go 测试、定向安全矩阵、100 次强制终止、100 路并发输出、固定大仓库基准、Windows 单文件构建、SHA-256 复算、双宿主 manifest validator、Codex marketplace 发现和 `git diff --check`。

## 发布门槛映射

| 门槛 | 可重复检查 | 通过标准 |
|---|---|---|
| JSON/JSONL 与 v1 兼容 | `go test ./...`；`TestVerifySessionReadsV1`、`TestVerifyRejectsFutureVersionsWithoutModifyingSession` | v1 只读通过；未来版本拒绝且源目录逐字节不变 |
| 100 次强制终止 | `release-check.ps1 -Full` 中 `TestReleaseHundredForcedTerminationsHaveValidEvidence` | 100 个 session 均可完整 `VerifySession` |
| 100 路 stdout/stderr | `TestReleaseHundredConcurrentOutputWriters` | 无死锁，事件数不少于 10,006，哈希链完整 |
| 全目录 secret scan | `TestRunLeavesNoPlaintextSecretInSession` 及 redactor / patch tests | argv、输出、错误、路径、Diff、报告均无测试明文 |
| 篡改定位 | `TestVerifyLocatesFirstEventAnomaly`、`TestVerifyLocatesArtifactAnomaly` | 删除、插入、改写、重排、torn tail、替换制品均非零并定位 |
| clean 边界 | clean tests 与 Windows junction tests | forged、junction、活跃/变化 session 被拒绝；非 TTY 默认不删除 |
| HTML 离线安全 | `TestHTMLReportEscapesInjectionAndBoundsContent`，人工浏览器 Network 面板 | 注入不执行、CSP 哈希存在、无外部请求 |
| HTML 事件时间线 | `M3-22` 定向测试与人工浏览器筛选 | 按 seq 最多显示 1,000 行（前后各 500），每行摘要最多 512 UTF-8 bytes；超出时 omission 数量/区间明确；截断与能力缺口可见 |
| `run` 结束摘要 | `M3-23` CLI / E2E 测试 | stderr 固定输出 Session、capability vector、五类 change 计数、risk 数量/最高级别、child/state、绝对 HTML 路径；child stdout 不变 |
| 性能基准 | `scripts/benchmark.ps1` | before / after 各小于 10 秒且不 partial；记录环境与分段耗时 |
| Windows 干净 VM | GitHub Actions `windows`，另在干净 Windows VM 下载 artifact | 仅放 exe 可运行 `version`、`list --json`；缺 Git/Agent 时错误清楚 |
| Windows 发布物 | `scripts/build.ps1` 后复算 `SHA256SUMS` | exe、SHA-256、版本说明齐全且一致 |
| Codex 插件 | plugin/skill validator；marketplace add/list/install；`afr run -- codex exec ...` | validator、发现和真实记录成功 |
| Claude Code 插件 | `claude plugin validate --strict`；`--plugin-dir`；`/afr:afr` | 共根加载且 skill 调用同一 `afr`；需已登录 Claude |
| 插件生命周期 | 两宿主 install/update/disable/remove 前后复算 session manifest | `~/.afr/sessions` 不变；缺失/不兼容 CLI 提示明确 |
| 文档 | README、QUICKSTART、SECURITY、RELEASE_NOTES | 安装到 clean 全流程可复制；边界与删除方式明确 |
| License 与发布真值 | `REL-01`、`REL-04`、`REL-05` | License 已明确；最终 main、tag、Release、二进制版本、CI、release notes 与 SHA 指向同一 commit |
| 公开附件回下载 | `REL-06` | 从 GitHub Release 下载后校验一致，独立目录可运行并可按公开文档发现/安装插件 |

## 2026-07-21 候选版记录

- 分支：`agent/redaction-pipeline`
- Windows 基准：Windows 10 IoT Enterprise LTSC 10.0.19044、i5-12400、31.77 GiB RAM、Git 2.54.0；最终候选检查 before 1.040 s、after 1.008 s、delta 3.101 ms、patch 1.635 s。
- 发布压力与安全矩阵：100 次强制终止、100 路并发输出（10,000 条记录）、secret、篡改、clean、HTML 和 v1 兼容测试均通过。
- Codex：0.138.0，marketplace 发现/安装成功；兼容模型 `gpt-5.4` 的 AFR 包装冒烟返回 `AFR_CODEX_SMOKE_OK`，session verify 为 63 events、6 evidence、3 derived。
- Claude Code：2.1.185，strict validator、`--plugin-dir`、marketplace install/disable/enable/update/uninstall 和组件 inventory 均通过；真实 `/afr:afr` 模型调用因本机未登录，发布前需由已登录账户复验。
- 插件生命周期哨兵：session `20260721T021226.609628600Z-6ba15d328a82d12c6a5a70a6` 的 manifest SHA-256 在两宿主操作前后均为 `fde9db65d601835a408be26897ce1c7b30998c2addc893216d6bae26c748e96e`。
- Windows GitHub Actions：构建、空 profile 的 `version/list`、SHA-256 复算和 artifact 上传已通过；候选实现 run：<https://github.com/liao96312/agent-flight-recorder/actions/runs/29795780084>。
- 2026-07-21 文档—实现复核新增阻塞项：`M3-22` HTML 有界事件时间线、`M3-23` 六项运行摘要、License、独立干净 VM、已登录 Claude 实调用、正式 tag / Release 与回下载验证。以上未完成前状态保持 candidate。

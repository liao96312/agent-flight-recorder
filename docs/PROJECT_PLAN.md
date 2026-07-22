# Agent Flight Recorder 项目规划

> 状态：v0.1 候选实现已合并；正式公开发布尚未完成
> 日期：2026-07-21
> 依据：[AI智能体黑匣子技术方案.docx](../AI智能体黑匣子技术方案.docx) 及截至 2026-07-21 的用户补充要求
> 目标：先闭环发布真值和人工验收，再进入公开试用与数据驱动迭代

## 1. 需求记录

本节是后续需求的唯一记录入口。新增需求先追加到这里，再决定是否进入当前里程碑，避免口头要求散落。

| ID | 来源 | 要求 | 当前决策 | 状态 |
|---|---|---|---|---|
| REQ-001 | 原技术方案 | 建设跨 Agent 的本地审计、取证与复核工具 | 保留，产品名暂用 Agent Flight Recorder（AFR） | 已采纳 |
| REQ-002 | 原技术方案 | Windows 优先，Go 单文件 CLI，本地离线运行 | 保留，Windows x64 是 v0.1 唯一发布门槛 | 已采纳 |
| REQ-003 | 原技术方案 | 采集进程输出、工作区变化、风险事件并生成离线报告 | 保留，但每项能力必须声明真实可见范围 | 已采纳 |
| REQ-004 | 原技术方案 | 写盘前脱敏、事件哈希链、制品校验 | 保留，并先冻结威胁模型、格式版本和派生物边界 | 已采纳 |
| REQ-005 | 用户补充 | 最终产品形态优先 CLI，或 Codex / Claude Code 插件 | 采用“一个 CLI 内核 + 一个双宿主薄插件包” | 已采纳 |
| REQ-006 | 用户补充 | 仔细阅读技术文档并记录后续提出的内容 | 本文件维护需求与决策；源 DOCX 不直接修改 | 已采纳 |
| REQ-007 | 用户补充 | 给出详细规划、TODO List 和 PlantUML | 分别落在本文件、[TODO.md](./TODO.md)、[architecture.puml](./architecture.puml) | 已完成规划 |
| REQ-008 | 用户补充 | 建立 GitHub 仓库并先公开上传雏形 | 已建立公开仓库 `liao96312/agent-flight-recorder`，默认分支为 `main` | 已完成 |
| REQ-009 | 用户补充 | 按 TODO 持续推进直至完成 | v0.1 候选实现已完成；未完成项继续由本文件的数据门和 TODO 验收约束 | 持续执行 |
| REQ-010 | 用户补充 | GitHub 项目主页与账号内其他项目保持同类风格 | 已完成中英文 README、徽章、架构图、快速开始与安全边界整理 | 已完成 |
| REQ-011 | 用户补充 | 再次对照技术文档，更新下一步 TODO 和 plan | 重新核对源方案、当前代码、发布记录与架构图，冻结下述发布收口和公开试用路线 | 已完成规划 |

### 1.1 默认假设

- 团队规模按 1 名全职开发者估算；多人并行时仍先冻结数据和 CLI 契约。
- v0.1 只支持非交互式 Agent 命令，例如 `codex exec` 或 Claude Code 的打印模式。
- Git、目标 Agent 是外部前置，不打包进 `afr.exe`。
- CLI 是证据链唯一实现；插件不得复制存储、脱敏、哈希、规则或报告代码。
- 所有默认值都可在真实测试后调整，但不提前建设数据库、守护进程、云后台或模型流程。

### 1.2 v0.1 候选版后的决策现状

| 决策 | 已确认状态 | 下一检查点 |
|---|---|---|
| 开源许可证 | 产品负责人于 2026-07-21 选择 MIT | `REL-01` 已完成；根 License、README、插件 manifest 与 release notes 一致 |
| 产品正式名称 | 已固定为 Agent Flight Recorder / `afr` | 仅在品牌冲突时重开 |
| 首个验收 Agent | `codex exec` 已完成真实包装与 verify 冒烟 | 每次正式发布复验 |
| Claude Code 验收 | strict validator、加载和生命周期已通过；已登录账户的 `/afr:afr` 实调用仍缺 | `REL-02` |
| 会话保留 | 不后台删除；文档可建议 30 天，实际删除必须显式运行 `clean` 并给出条件 | 20 次真实会话后复盘 |
| 插件 hooks | 明确不进入 v0.1；只有 20 次会话评审证明 wrapper 的语义缺口反复阻碍复核才启动 | `DEC-01` |

## 2. 审读结论

原方案的方向正确：本地优先、Agent 外部旁路、追加写 JSONL、Git CLI、离线报告、确定性规则和无模型 MVP 构成了最小闭环。需要先修正以下契约，否则代码即使“能跑”也会产生过度承诺。

### 2.1 开工前必须冻结的修正

| ID | 原方案缺口 | 影响 | 本规划的处理 |
|---|---|---|---|
| GAP-001 | L1 通用包装器无法可靠观测任意工作区外写入、真实网络上传、UAC/sudo 或隐藏工具调用 | 会把不可见风险误报成已覆盖 | 把 L1/L2/L3 改成能力矩阵；不可见项显示 `not_observable` |
| GAP-002 | “不可静默篡改”容易被理解成可信签名 | 本机高权限攻击者可重算整条链 | v0.1 只承诺一致性与篡改迹象检测；签名/远端见证延后 |
| GAP-003 | `canonical_event_without_hash` 未定义 | 跨版本或跨实现无法稳定校验 | 哈希精确的已脱敏事件 body 字节，并提供格式版本与黄金向量 |
| GAP-004 | manifest 声称覆盖所有制品，同时 `afr report` 又可重生成 | 派生报告变化会破坏证据校验 | manifest 分 `evidence` 与 `derived`；证据冻结，报告可重建 |
| GAP-005 | `verify.json`、`replay.plan.json` 在各章节的强制性不一致 | 验收清单无法确定 | `verify` 默认只输出终端/指定外部 JSON；自动回放计划退出 v0.1 |
| GAP-006 | “异常退出”未区分子进程退出、记录器硬杀和断电 | 硬杀时不可能写 `session_interrupted` | 有序中断正常收尾；硬杀会话标为 incomplete，并从最后完整事件生成只读报告 |
| GAP-007 | 直接保存 secret 的 hash prefix | 低熵 secret 可被字典枚举 | 占位符使用仅存活于进程内的会话级 HMAC key；只保留关联前缀 |
| GAP-008 | stdout、stderr、Diff 的分块脱敏边界未定义 | secret 可能跨 chunk 泄漏 | 按有上限的逻辑记录处理；超长/不可解析记录 fail-closed，只保存大小与流哈希 |
| GAP-009 | Windows 只杀直接子进程可能遗留孙进程 | AFR 已结束但 Agent 后台仍改文件 | M0 做 Job Object 技术验证；必要时只引入 `x/sys/windows` |
| GAP-010 | Git 最终 Diff 不能自动区分任务前已有脏改动 | 会把用户旧改动归因给 Agent | 保存 before/after 状态并计算会话 delta；报告单列 pre-existing changes |
| GAP-011 | 单文件 HTML 内联脚本与严格 CSP 存在冲突 | XSS 或策略形同虚设 | 所有内容转义，CSP 使用脚本/样式哈希，禁止外部资源 |
| GAP-012 | 文档元数据日期与内容日期不一致 | 文档自身时间不能当审计依据 | AFR 区分文件系统时间、内容声明和容器元数据的来源与可信度 |

### 2.2 术语修正：从“证据等级”改为“证据能力”

L3 可以在没有 L2 时与 L1 组合，因此它不是严格等级。报告改用以下能力向量：

| 能力 | v0.1 wrapper | 插件 hook 模式 | 说明 |
|---|---:|---:|---|
| `process` | 是 | 否/部分 | argv、stdout/stderr、退出码、运行时长 |
| `workspace_git` | 是 | 可选 | Git before/after 与会话 delta |
| `workspace_scan` | 非 Git 时 | 可选 | 工作区内文件元数据和哈希 |
| `native_tool_events` | 否 | 是 | 仅宿主明确提供的工具/权限事件 |
| `local_policy` | 是 | 是 | 只对已经看见的证据运行规则 |
| `os_file_monitor` | 否 | 否 | v0.1 不做系统级文件审计 |
| `network_monitor` | 否 | 否 | v0.1 不做系统级网络审计 |

报告必须同时列出 `observed`、`not_observable` 和 `truncated`，不能用风险为零暗示全面安全。

### 2.3 2026-07-21 源方案—实现—发布真值复核

本轮重新逐段核对源 DOCX、`main` 分支代码、发布检查表和 GitHub 实际状态。结论是：证据内核和 Windows 候选制品已经形成，但“候选实现完成”不能写成“正式发布完成”。下表是下一阶段的基线，TODO 不得绕过这些差异。

| 原方案或既有契约 | 当前实现 / 外部事实 | 处理 |
|---|---|---|
| 公开可安装的首版 | 仓库已公开并采用 MIT，但没有 tag 或 GitHub Release | `REL-02` 至 `REL-06` 完成前只称 v0.1 候选版 |
| 运行结束输出 Session、Evidence、Changed、Risks、Exit、Report | 已固定输出六行 stderr 摘要，child stdout 不受污染 | `M3-23 [P0]` 已完成并通过三平台 CI |
| HTML 提供可筛选的完整时间线 | 已提供有界、已脱敏的 seq 时间线，最多 1,000 行 | `M3-22 [P0]` 已完成；10 万事件 fixture 通过 |
| 一条命令定位并打开报告 | `show --open` 已通过独立 argv 调用平台原生打开命令 | `M3-09 [P1]` 已完成，不阻塞 v0.1.0 发布 |
| 非 Git 扫描支持有限 `.afrignore` | 根配置在 child 启动前加载一次，支持路径前缀和 Go `path.Match` | `M1-11 [P1]` 已完成，明确不冒充完整 gitignore |
| workspace `.afr.json` 可添加 RE2 脱敏规则 | 严格 schema 在 session/child 启动前解析并编译 | `M2-07 [P1]` 已完成，无效规则 fail-closed |
| Unix 终止整个进程组并提供 Linux/macOS 版本 | 共享孙进程测试及 Ubuntu/macOS 的 go test/build/run/verify 已通过 | `M0-15` 与 `R0-05` 已完成；非 Windows 仍标 beta |
| `afr report` 可重建派生报告 | 当前只在 `run` 收尾自动生成报告 | `M3-10` 降为 P2；20 次评审证明需要重建时再做 |
| replay、原生 hooks、Dify / 团队分析 | 均未实现 | 保持暂缓；只能由 `DEC-01` 的真实数据启动，不因原方案列出就提前建设 |

本轮删除未进入实现和 TODO 的 `--summary-json` 过期承诺；`show --open` 不属于 v0.1 发布门槛，后续已作为独立 P1 交付。原方案的“证据等级”继续按能力矩阵表达；`verify.json`、自动 replay、OS 级文件/网络观测仍不伪装为已交付。

## 3. 推荐产品形态

### 3.1 结论

交付一个产品、三种入口：

1. **`afr` CLI：唯一证据内核。** 独立运行、可测试、可在 CI 中使用。
2. **Codex 插件：薄 skill。** 组装 `afr` 命令、解释能力范围、打开报告；v0.2 再评估 hooks。
3. **Claude Code 插件：薄 skill。** 与 Codex 共用说明和 CLI；v0.2 可通过官方 hooks 增加原生事件。

首版插件不假装能追溯已经开始的当前会话。它只能启动一次新的被记录任务，或查看/验证已有会话。

### 3.2 为什么不是“只做插件”

- 记录器必须独立于 Agent，插件生命周期不能成为证据可靠性的前提。
- 进程托管、Git 基线、脱敏、哈希链和异常收尾适合由一个本地二进制统一完成。
- Codex 与 Claude Code 的 hook 事件和结束语义不同；直接在插件中实现会产生两条不一致证据链。
- CLI 可以先用假 Agent 做确定性测试，插件只做适配验收。

### 3.3 v0.1 明确不建设

- 数据库、全文索引、Web 服务、后台常驻进程。
- 交互式 TUI/PTY 托管。
- 云同步、多租户、组织权限、计费、团队仪表盘。
- 大模型定级、Dify 工作流或远程分析。
- 自动执行回放，尤其是破坏性、提权或网络步骤。
- OS 级文件/网络监控、内核驱动、全系统沙箱。
- 数字签名、硬件密钥、远端时间戳见证。
- MCP 服务；薄 skill 直接调用 CLI 已覆盖首版需求。

## 4. v0.1 候选实现范围

### 4.1 必须交付

- `afr run -- <command> [args...]`
- `afr list`
- `afr show <session|latest>`
- `afr verify <session|latest>`
- `afr clean --older-than <duration>`，默认预览，删除需确认或 `--yes`
- Windows x64 单文件 `afr.exe`
- 非交互子进程托管、终端透传、退出码策略和有序取消
- Git before/after、pre-existing changes、会话 delta、未跟踪文件摘要
- 非 Git 工作区退化扫描，且不跟随符号链接/重解析点
- 写盘前脱敏、确定性本地风险规则、事件链和制品校验
- `session.json`、`events.jsonl`、`manifest.json`
- `agent-flight.md`、`agent-risk.json`、`report.html`
- incomplete 会话的只读复核
- 一个共享薄插件包；Codex 真实包装冒烟和 Claude Code 本地 validator / loader 已完成，Claude 已登录实调用进入正式发布门槛

### 4.2 候选版后续增强，不阻塞 v0.1.0 发布

- `afr show --open` 跨平台打开浏览器。
- 有限 `.afrignore` 与 workspace `.afr.json` 自定义 RE2 脱敏规则。
- Linux/macOS 同源编译和最小冒烟。
- 插件内提供“查看最近报告”和“验证最近会话”两个额外 skill。
- `afr report` 只有真实评审需要重建派生物时才立项，优先级为 P2。

### 4.3 推迟

- `afr replay` 与 `replay.plan.json`：没有真实复核数据前不建设。
- Codex/Claude Code 当前交互会话的自动 hook 采集。
- 原生 Agent JSONL 适配器。
- Dify、团队聚合、搜索索引、签名见证。

## 5. 系统架构

### 5.1 主链路

```text
用户 / 薄插件
  -> afr CLI
  -> 会话与进程托管
  -> stdout/stderr + Agent 可选原生事件 + 工作区 before/after
  -> 归一化
  -> 写盘前脱敏
  -> 基于可见证据的本地规则
  -> 序号与哈希链
  -> 会话目录
  -> Markdown / JSON / 单文件 HTML
  -> verify
```

### 5.2 最小代码边界

目标不是一次性创建所有文件；按职责增长，只有文件超过一个清晰职责时才拆分。

```text
cmd/afr/main.go          CLI 入口和子命令分派
internal/run.go          子进程、信号、输出泵、生命周期
internal/session.go      会话状态、目录、原子写、manifest
internal/event.go        事件格式、脱敏、序号、哈希链
internal/workspace.go    Git 与非 Git before/after
internal/policy.go       确定性规则与能力检查
internal/report.go       Markdown / JSON / HTML 派生物
web/report.html.tmpl     go:embed 的离线模板
plugins/afr/             双宿主薄插件包
```

不为这些模块建立单实现接口或工厂。Agent 原生适配确实出现第二种实现时，再抽象适配边界。

### 5.3 会话状态机

```text
starting -> running -> finalizing -> completed
                    \-> failed
running -> interrupted -> finalizing
starting/running/finalizing --硬杀或断电--> incomplete
```

- `completed` 表示记录器成功完成收尾，不等于子进程退出码为 0。
- 子进程非零退出可以对应完整且可验证的 `completed` 会话。
- `incomplete` 由后续只读命令推断，不伪造一个从未落盘的中断事件。

### 5.4 正常写入顺序

1. 校验工作区和命令，不写任何会话文件。
2. 用 UTC 时间和 `crypto/rand` 创建不可预测、不可冲突的 session ID。
3. 创建权限受限的会话目录和初始 `session.json`。
4. 打开唯一 `events.jsonl` writer，写 `session_started`。
5. 采集工作区基线，写基线事件和制品。
6. 启动子进程，持续 drain stdout/stderr；用户仍实时可见。
7. 每条可持久化内容先脱敏，再规则判断，再追加事件。
8. 子进程结束后采集工作区最终状态和会话 delta。
9. 写 `process_exited`、风险汇总和 `session_finished`。
10. 刷新并关闭事件文件，原子写最终 `session.json`。
11. 生成派生报告到临时文件，再逐个原子替换。
12. 最后原子写 `manifest.json`，CLI 返回约定退出码。

### 5.5 硬杀恢复边界

- `events.jsonl` 只认可最后一个完整换行前的事件。
- 尾部半行不删除、不续写；计算其字节数与会话级 HMAC 指纹，并在报告中标记 `torn_tail`。
- incomplete 会话不再追加业务事件，只从有效前缀生成派生报告。
- 不增加常驻恢复服务；`list/show/verify` 发现 incomplete 即可。

## 6. 数据契约

### 6.1 事件外壳

避免自创通用 canonical JSON。记录器先一次性编码已经脱敏的 `body`，再对这段精确字节计算哈希：

```json
{
  "format_version": 1,
  "prev_hash": "<hex>",
  "body": {
    "seq": 42,
    "ts": "2026-07-20T02:31:18.426Z",
    "elapsed_ns": 123456789,
    "type": "risk_found",
    "source": "policy",
    "payload": {}
  },
  "hash": "<hex>"
}
```

```text
hash = SHA256("AFR-EVENT-v1\n" || prev_hash_bytes || "\n" || exact_body_bytes)
```

规则：

- genesis `prev_hash` 固定为 32 个零字节。
- 文件编码固定 UTF-8，无 BOM，每条事件以 LF 结束。
- `seq` 从 1 开始严格递增。
- `elapsed_ns` 来自进程内单调时钟差，`ts` 仅用于人类时间。
- 原生适配器只提交未入链的候选事件；最终序号和哈希始终由 CLI 生成。
- 仓库保存黄金输入/输出向量，任何格式变更必须提升 `format_version`。

### 6.2 最小事件类型

| 类型 | 关键字段 |
|---|---|
| `session_started` | session_id、workspace、argv_redacted、capabilities |
| `workspace_baseline` | git/non-git、HEAD、pre-existing summary、artifact refs |
| `process_started` | pid、executable、cwd |
| `process_output` | stream、chunk_seq、text 或 omitted metadata |
| `output_truncated` | stream、limit、total_bytes、stream_fingerprint |
| `native_event` | adapter、host_event、correlation_id、redacted payload |
| `risk_found` | rule_id、severity、evidence_seq、explanation |
| `workspace_final` | final status、delta summary、artifact refs |
| `process_exited` | exit_code、signal/termination、duration |
| `session_interrupted` | received signal、forward/kill result |
| `session_finished` | final status、final capabilities、risk summary |

### 6.3 manifest 边界

`manifest.json` 自身不哈希自身，包含两组条目：

- `evidence`：`events.jsonl`、最终 `session.json`、before/after 快照、workspace patch；结束后冻结。
- `derived`：Markdown、风险 JSON、HTML；可以从已验证 evidence 重建。

`afr report` 若实现，必须先验证 evidence，只更新 derived 文件与 derived 哈希，不得修改 events 或 evidence 条目。

`verify.json` 默认不写入会话目录，避免自引用。需要机器输出时使用 stdout 或用户指定的会话目录外路径。

### 6.4 会话目录

```text
~/.afr/sessions/<session-id>/
  session.json
  events.jsonl
  manifest.json
  snapshots/
    git-before.json
    git-after.json
  diffs/
    workspace.patch
  agent-flight.md
  agent-risk.json
  report.html
```

不建立全局数据库。`afr list` 扫描一级 session 目录并只读取小型 `session.json`；出现真实性能瓶颈后再增加索引。

## 7. CLI 契约

### 7.1 命令面

```text
afr run [--workspace PATH] -- COMMAND [ARG...]
afr list [--limit N] [--json]
afr show [--json|--open] [SESSION|latest]
afr verify [--json] [SESSION|latest]
afr clean (--older-than DURATION | --max-bytes SIZE) [--yes]
afr version
```

`show --open` 已作为 P1 可用性命令交付。`report` 和 `replay` 不进入首个必需命令面：自动报告已经覆盖日常路径；有证据证明需要重建时再加入 `report`。

冻结的 v0.1 解析契约如下；`[]` 表示可选，命令名和 flag 区分大小写：

| 命令 | 默认值与边界 | 稳定用法错误示例 |
| --- | --- | --- |
| `run [--workspace PATH] -- COMMAND [ARG...]` | workspace 为当前目录；第一个独立 `--` 是 AFR 与 child argv 的唯一边界，之后的空格、引号、Unicode 和前导短横线逐项原样传递 | 缺少 `--`、空 command、未知 AFR flag |
| `list [--limit N] [--json]` | limit=20，按 session ID 倒序；N 必须大于 0 | 多余位置参数、非法 N |
| `show [--json\|--open] [SESSION\|latest]` | selector 默认为 `latest`；非 JSON 模式读取 Markdown 摘要；`--open` 输出绝对 HTML 路径后调用平台原生命令 | selector 歧义、额外参数、同时传入 `--json` 与 `--open` |
| `verify [--json] [SESSION\|latest]` | selector 默认为 `latest`；只读，不修复证据；`--json` 也兼容放在 selector 后 | 多个 selector、未知 flag |
| `clean (--older-than DURATION \| --max-bytes SIZE) [--yes]` | 恰好一个选择条件；默认只预览；非 TTY 删除必须显式 `--yes` | 无条件、双条件、非法 duration/size |
| `version` | 无参数，写 stdout | 任意额外参数 |

黄金 argv 示例：`afr run --workspace "C:\work space" -- agent.exe "space arg" "\"quoted\"" 中文 --leading` 必须产生五项 child argv：`agent.exe`、`space arg`、`"quoted"`、`中文`、`--leading`（可执行文件加四个参数），不得重组为 shell 字符串。所有用法错误以 `AFR_USAGE:` 开头、返回 64，且不得创建 session。

### 7.2 流与退出码

- child stdout 原样实时写到 AFR stdout；child stderr 原样实时写到 AFR stderr。
- AFR 的会话摘要写 stderr，避免污染 child 的机器可读 stdout；当前候选版已固定输出 Session/Evidence/Changed/Risks/Exit/Report 六行摘要。
- v0.1 不提供 `--summary-json`；如果真实脚本消费需求出现，先冻结 schema 和输出路径再新增。
- `run` 在记录器完整收尾时透传 child exit code。
- 记录器在启动 child 前失败时返回保留的 AFR 错误码，并且打印稳定错误类别。
- 记录器在 child 已运行后自身收尾失败时返回 AFR 错误码，同时保留 child exit code 到 `session.json`。
- `verify`：0 表示通过，2 表示证据不一致，其他值表示读取/用法失败。

冻结数值与脚本判别规则：

| 场景 | 退出码 | stdout | stderr 前缀/内容 |
| --- | ---: | --- | --- |
| `run` 完整收尾 | child 原退出码（0–255） | child stdout 原样 | child stderr 原样，并在末尾写 Session/Evidence/Changed/Risks/Exit/Report 六行摘要 |
| child 启动前或 AFR 收尾失败 | 70 | 已产生的 child stdout（若有） | `AFR_RUNTIME:`；已分配 session 时追加其目录 |
| CLI 用法错误 | 64 | 空 | `AFR_USAGE:` |
| `verify` 通过 | 0 | 文本或 `--json` 结果 | 空 |
| `verify` 发现不一致 | 2 | `--json` 时为结构化结果，否则空 | 非 JSON 模式以 `AFR_VERIFY:` 开头 |
| `list/show/verify` 读取失败 | 70 | 空 | `AFR_RUNTIME:` |

child 自身返回 2、64 或 70 时仍属于“完整收尾”，脚本通过 stderr 中是否出现 `AFR_RUNTIME:` / `AFR_USAGE:` 区分 AFR 故障；child 退出码同时持久化到 `session.json.child_exit_code`。AFR 不吞掉 child stderr，也不向 child stdout 注入摘要。

最终数值在 Phase 0 用 Windows 与 CI 行为测试冻结，不能只写文档不测试。

### 7.3 取消与进程树

- 第一次 Ctrl+C 转发给 child/process group，并进入宽限期。
- 宽限期后仍存活则终止整个托管进程树。
- 第二次 Ctrl+C 可以立即强制终止，但仍尽力 flush，不承诺完整报告。
- Windows 使用 Job Object 技术验证；Unix 使用 process group。
- v0.1 不托管交互式 TUI，stdin 行为保持明确：默认关闭或只在显式标志下继承。

## 8. 工作区证据

### 8.1 Git 模式

- 先定位仓库根并记录实际 Git 版本。
- 使用机器可解析、NUL 分隔的 status 输出，不解析本地化人类文本。
- 分别记录 tracked、staged、unstaged、untracked、rename、submodule 状态。
- before 已脏内容标记为 `pre_existing`；after-before 的变化才是 `session_delta`。
- 禁用 pager、交互凭据提示和外部 diff driver。
- binary、LFS、submodule 和超大 Diff 只做明确摘要与哈希，不伪装成完整 patch。
- Git 命令失败时降级并显式降低 workspace 能力，不默默输出空 Diff。

### 8.2 非 Git 模式

- 只扫描工作区真实路径之内，不跟随 symlink/junction/reparse point。
- 记录相对路径、类型、大小、mtime 和会话级 HMAC 内容指纹；默认不输出可被字典枚举的裸内容哈希。
- 固定跳过 `.git`；工作区根 `.afrignore` 支持注释、空行、路径前缀和 Go `path.Match` glob，Windows 分隔符先统一为 `/`。非法模式在 child 启动前失败；不支持 negation、递归 `**` 或父目录发现，不声称兼容完整 `.gitignore`。
- 配置时间、文件数和读取字节预算；超限后输出 `partial` 与遗漏原因。
- 权限拒绝、循环链接、路径消失都生成可见事件。

### 8.3 越界能力限制

L1 的 before/after 只能证明工作区内变化。只有命令参数或原生事件明确暴露真实目标路径时，规则才可报告越界；否则报告显示 OS 级越界监控未启用。

## 9. 脱敏与风险规则

### 9.1 统一写盘门

任何文本进入以下文件前必须经过同一脱敏函数：

- argv 与错误消息
- stdout/stderr
- 原生事件 payload
- 路径、文件名和环境变量名称
- Git Diff 与非 Git 摘要
- session、risk、Markdown、HTML 等所有派生物

派生报告只消费已脱敏结构，不重新读取未脱敏原始输出。

### 9.2 fail-closed 规则

- 二进制或未知编码：只保存字节数、会话级 HMAC 指纹、MIME/判定原因。
- 单条逻辑记录超过安全上限：不保存正文，只保存 omitted marker。
- 多行私钥等跨行模式：在有界记录缓冲中完整检测。
- 内置 detector 始终启用；工作区根 `.afr.json` 可增加具名 `regex` 规则。自定义正则使用 Go RE2，加载时编译；未知字段、重名、内置名冲突、空匹配和无效正则都在 session/child 启动前使配置失败。
- redaction 占位符使用会话内 HMAC 关联前缀，HMAC key 不写盘。
- 未持久化的原始流或 Diff 只保存会话级 HMAC 指纹，不保存可被低熵字典枚举的裸 SHA-256；manifest 的 SHA-256 只校验已经脱敏并实际存储的制品。

### 9.3 v0.1 规则来源

| 规则 | 可用证据 | 不能声称的能力 |
|---|---|---|
| 破坏性命令 | 可见 argv、shell/native tool 输入 | 看不到的子进程内部命令 |
| 工作区越界 | 可见真实目标路径、native event | 任意 OS 写入审计 |
| 权限提升 | 可见 `sudo`/UAC/权限请求事件 | 系统完整权限轨迹 |
| 敏感信息 | 经统一管线的可见文本 | 未读取文件中的秘密 |
| 异常网络 | 可见 URL/域名/上传命令 | 实际网络流量监控 |
| 大规模变化 | workspace delta | 工作区外变化 |

风险结果必须包含 `rule_version`、`evidence_seq[]`、事实、解释、严重级别和建议；事实与判断分开。

v0.1 的 `workspace.bulk_change` 规则版本为 1，本次 session delta 达到 100 个文件时触发；pre-existing 文件不计入阈值。

越界路径、提权请求和网络目标规则只读取 AFR 直接收到的 argv；即使命中规则，`os_file_monitor` 与 `network_monitor` 仍标记为 `not_observable`，不把请求语法等同于实际系统行为。

## 10. 离线报告

### 10.1 三个视图

- `agent-flight.md`：短摘要，适合评审和工单。
- `agent-risk.json`：稳定、版本化的机器输出。
- `report.html`：提供事件计数、输出预览、风险、文件筛选和按 seq 排序的有界事件时间线。

### 10.2 安全和容量

- 使用 `html/template` 和上下文转义，不拼接 HTML。
- CSP 禁止网络、对象、frame；内联 CSS/JS 使用构建时哈希。
- 不加载外部字体、图片、脚本或分析服务。
- HTML 不内嵌全部 200 MB 输出；只嵌入受限预览与统计，详细事件从同一文件中的受限数据块按需展示。
- 事件时间线最多 1,000 行、每行摘要最多 512 UTF-8 bytes；超过时保留前 500 / 后 500，并显示 omitted 数量与 seq 区间，不能静默制造“完整”假象。
- 注入测试覆盖 HTML、属性、脚本、CSS、JSON 和 Unicode 边界。

## 11. 插件规划

### 11.1 v0.1：共享薄插件

首选一个发行目录放两份 manifest 和一份共享 skill：

```text
plugins/afr/
  .codex-plugin/plugin.json
  .claude-plugin/plugin.json
  skills/afr/SKILL.md
```

skill 只做四件事：

1. 检查 `afr` 是否在 PATH，并报告安装缺失。
2. 根据宿主构造非交互命令，但保留 argv 数组边界。
3. 明确说明将启动新的被记录任务，不追溯当前对话。
4. 运行 `show/verify` 并给出报告路径。

两种宿主都要单独跑本地加载和调用冒烟；若任一宿主不能容忍同一根目录中的另一个 manifest，再拆成两个发布包，不提前维护两套业务代码。

候选版实测状态：Codex marketplace 发现、安装、真实 `afr run -- codex exec ...` 和 session verify 已通过；Claude Code strict validator、`--plugin-dir` 与插件生命周期已通过，但本机未登录，因此 `/afr:afr` 调用同一 `afr` 的端到端证据归入 `REL-02`，未通过前不能写成双宿主正式发布完成。

### 11.2 v0.2：hook 原生事件

只有真实用户证明 wrapper 模式不够时再增加：

```text
plugins/afr/
  hooks/hooks.json              Codex 默认 hook 文件
  hooks/hooks.claude.json       Claude Code 特有结束事件（如需要）
```

约束：

- hook 只把宿主事件送入 `afr hook`，不自行写 JSONL。
- Codex 没有等价的可靠 SessionEnd 时，不把 turn `Stop` 伪装成 session completed；只标记 idle/finalizable。
- 多个 hooks 可能并发启动；`afr hook` 必须用会话锁串行分配 seq/hash。
- 最小并发方案是会话内 O_EXCL 锁文件、短超时和受控陈旧锁恢复；吞吐成为瓶颈后才引入单写者进程。
- 父 wrapper 启动宿主时可设置经校验的 `AFR_SESSION_ID`，hook 事件并入同一会话，避免 L1/L2 双会话。
- 插件 hook 需要用户信任；报告必须记录 hook 是否安装、启用、受信和实际触发。

Codex 当前文档支持插件根默认 `hooks/hooks.json`；本地 plugin validator 对 manifest 中显式 `hooks` 字段存在版本差异，因此首选默认文件，不在 Codex manifest 中写 override，直到目标版本验证通过。

### 11.3 规范依据

- Codex 插件：<https://learn.chatgpt.com/docs/build-plugins>
- Codex hooks：<https://learn.chatgpt.com/docs/hooks>
- Claude Code 插件：<https://code.claude.com/docs/en/plugins>
- Claude Code hooks：<https://code.claude.com/docs/en/hooks>

插件规范会变化，发行 CI 必须校验目标宿主版本，不能只验证 JSON 可解析。

## 12. 测试策略

只留下能抓住真实失败的测试，不为每个一行函数建套件。

### 12.1 固定测试工具

- 一个仓库内 `fake-agent`，可配置 stdout/stderr、退出码、长行、二进制、延迟、孙进程和文件改动。
- 一个 Git fixture 生成脚本，覆盖 clean/dirty/staged/untracked/rename/binary/submodule。
- 一组公开的假 secret，不使用真实凭据。
- 哈希链黄金向量和恶意篡改样本。

### 12.2 最小测试层

1. **单元测试**：事件哈希、脱敏、路径边界、ignore、状态 delta。
2. **进程集成测试**：fake-agent、取消、进程树、stdout/stderr 并发、硬杀。
3. **端到端测试**：真实 Git 仓库 + `codex exec`，验证全目录制品。
4. **安全测试**：全目录 secret 扫描、HTML 注入、clean 越界、manifest 篡改。
5. **插件冒烟**：Codex 与 Claude Code 各自加载、调用 skill、找到 CLI、打开报告。
6. **基准测试**：10 万事件、大仓库基线、大输出截断；先固定硬件和 fixture 再谈指标。

固定性能 fixture：Windows x64；10,000 个 1 KiB 文本文件（100 目录 × 100 文件，共 10,240,000 bytes），Git clean baseline 后修改 100 个文件；工作区 before/after 各自必须在 10 秒扫描预算内完整完成，再分别记录 delta 与 patch 耗时。事件 fixture 为 100,000 个 1 KiB payload，flush 策略固定为 64 KiB 或 1 秒、关键事件立即 sync。运行 `scripts/benchmark.ps1` 会同时记录 Windows 版本、CPU/逻辑核、物理内存、当前磁盘与余量、Git 版本和分阶段耗时；日常 `go test ./...` 不运行重 fixture。

2026-07-21 基线：Windows 10 IoT Enterprise LTSC 10.0.19044、i5-12400（12 logical CPUs）、31.77 GiB RAM、D: 余量 396.33 GiB、Git 2.54.0；before 1.054 s、after 1.043 s、delta 3.003 ms、patch 1.656 s，完整 fixture 通过且未触发 partial/truncated。

### 12.3 发布门槛

- 所有 JSON/JSONL 均通过版本化 schema 或等价 Go 解码检查。
- 100 次强制终止子进程后，没有不可读完整行，所有有效前缀可校验。
- 100 路并发 stdout/stderr 压测无死锁、无管道阻塞、seq 唯一连续。
- 测试 secret 出现在 argv、输出、事件、Diff、路径、报告时，全会话目录无明文或可逆值。
- 删除/改写/重排事件和替换 evidence 制品时，`verify` 返回非零并定位首个异常。
- `clean` 在伪造目录、symlink/junction、活跃会话和非 TTY 下不越界。
- HTML 在浏览器中无外部请求，恶意内容不执行。
- Windows 干净 VM 只放 `afr.exe` 后可运行；外部 Git/Agent 前置有明确错误提示。
- 已登录 Claude Code 账户实际调用 `/afr:afr`，确认使用同一 `afr`、生成 session 且 `verify` 通过；validator 不能替代该人工门槛。
- 发布前明确 License，最终 `main` commit 的测试、CI、版本号和 SHA-256 一致；创建不可变 `v0.1.0` tag 与公开 GitHub Release。
- 从公开 Release 重新下载附件复算校验和，并在独立目录完成 `version`、`list --json` 与插件安装文档冒烟。

## 13. 里程碑

估算按 1 名全职开发者；历史阶段与下一阶段分开标记，完成代码不自动等于完成公开发布。

| 阶段 | 当前状态 | 后续投入 | 范围 | 退出门槛 |
|---|---|---:|---|---|
| Phase 0 契约冻结 | 已完成 | 0 | 威胁模型、能力矩阵、事件/manifest、CLI、退出码、fixture | 关键 JSON 示例和黄金哈希向量评审通过 |
| M0 记录闭环 | 已完成并通过三平台 CI | 0 | run、会话、输出泵、退出/取消、incomplete、进程树 | Windows、Ubuntu、macOS 路径均通过 |
| M1 工作区证据 | 已完成 | 0 | Git before/after/delta、非 Git 扫描、容量预算、有限 `.afrignore` | P0 与 `M1-11` 验收均通过 |
| M2 安全证据 | 已完成 | 0 | 统一脱敏、内置与自定义 RE2 规则、哈希链、manifest、verify | P0 与 `M2-07` 验收均通过 |
| M3 候选版 | 候选代码、报告时间线和运行摘要已完成 | 0 | 报告、CLI、Windows 包、薄插件 | `M3-22`、`M3-23` 与既有安全门槛均通过 |
| R0 正式发布 | 未完成 | P0 约 1 天 + 两项人工/外部门槛 | License、Claude 实调用、干净 VM、tag、Release、回下载 | `REL-01` 至 `REL-06` 全部有证据 |
| P1 可用性 | 已完成 | 0 | `show --open`、ignore、自定义 RE2 | 三项均独立测试、提交和 CI，不捆绑大版本 |
| OBS 公开试用 | 未开始 | 事件驱动，直到 20 次 | 真实 Codex/Claude 会话、本地评审表 | `R0-09` 形成 20 条可追溯记录 |
| DEC 数据门 | 未开始 | 半天 | 对证据缺口、性能、复核价值做决策 | `DEC-01` 明确“只做一个方向”或“保持现状” |
| M4 原生 hooks | 条件性暂缓 | 触发后约 1–2 周 | `afr hook`、并发锁、宿主事件映射 | 只有 `DEC-01` 证明 wrapper 语义缺口反复阻碍复核才启动 |
| M5 团队增强 | 条件性暂缓 | 数据驱动 | 签名见证、索引、Dify/团队分析 | 有真实使用数据、明确责任人和合规边界 |

### 13.1 两周停止门槛

M1 结束时必须能稳定回答：

1. 子进程异常退出后，已落盘证据是否可读？
2. pre-existing 与本次会话改动是否能可靠区分？
3. 输出和 Diff 是否在写盘前完成脱敏？

任一答案为否，就暂停插件、HTML 美化和平台化，只修核心证据链。

### 13.2 下一步执行计划

| 顺序 | TODO | 可并行性 | 完成定义 |
|---:|---|---|---|
| 1 | ✅ `M3-22` HTML 有界事件时间线；`M3-23` 完整运行摘要 | 已完成 | 10 万事件 fixture、固定六行摘要、child stdout 不污染和三平台 CI 均通过 |
| 2 | ✅ `REL-01` MIT License | 已完成 | 根 License、README、插件 manifest 与发布说明一致 |
| 3 | ⏸ `REL-02` 已登录 Claude `/afr:afr`；`REL-03` 独立干净 Windows VM | 产品负责人于 2026-07-22 暂时搁置 | 保持未完成，不用 validator、CI 或开发机替代人工证据 |
| 4 | ✅ `M3-09`、`M1-11`、`M2-07` | 发布门槛暂停期间已逐项完成；每项单独提交 | 浏览器打开、ignore 和配置失败验收及三平台 CI 分别通过 |
| 5 | `REL-04` 最终候选检查 | 依赖顺序 3 恢复并完成 | 从最终 `main` commit 运行完整 release-check，CI、版本、SHA-256 和 release notes 指向同一 commit |
| 6 | `REL-05` tag + GitHub Release；`REL-06` 回下载 / 公共安装复验 | 严格串行 | 公开附件可下载、校验一致、快速开始能从零复现 |
| 7 | `R0-09` 20 次真实会话 | `REL-06` 后立即开始；不建设遥测 | 每次记录 version/commit、host、session ID、发现、误报、缺口、复核耗时、磁盘/启动开销和继续使用意愿，可区分期间的补丁版 |
| 8 | `DEC-01` 数据门 | 依赖完整 20 次记录 | 形成 ADR：只启动一个证据最强的方向，或明确保持 wrapper；不得一次启动 hooks、Dify、索引和签名 |

宿主登录等待期间不跳过门槛，也不得在人工验收未通过时提前打正式 tag。

## 14. 发布与运维

- 使用 Go reproducible build 信息，`afr version` 输出版本、commit、构建时间、format version。
- Windows x64 候选 exe 与校验和已经生成；项目采用 MIT License，只有 `REL-02` 至 `REL-06` 通过后才能称为 v0.1.0 正式公开发布。代码签名作为企业增强，不伪装已有签名。
- 配置优先级固定为 CLI flags > workspace `.afr.json` > defaults；当前 `.afr.json` 仅提供严格的工作区自定义 RE2 脱敏规则，没有用户层或组织层配置。
- 会话目录默认位于用户主目录 `.afr/sessions`，继承/设置仅当前用户访问权限。
- 没有后台自动清理；用户显式运行 `clean`。
- schema 或哈希格式升级必须保留旧版本只读 verify，不能静默重写旧证据。
- GitHub Release 的附件、tag、release notes、CI 和 `afr version` 必须能追溯到同一 commit；发布后回下载校验是门槛，不是可选清理项。

## 15. 成功指标

首版上线后只收集本地、用户主动提供的结果，回答三个问题：

1. AFR 是否发现了最终 Diff 看不到的真实问题？
2. 故障复盘时间是否明显下降？
3. 启动、磁盘和认知开销是否让开发者愿意持续使用？

20 次真实会话使用本地评审表，不建设遥测服务。每条至少记录：AFR 版本与 commit、宿主/版本、session ID、任务类型、运行时长、会话目录大小、真实发现、误报、缺失证据、人工复核耗时、是否愿意再次使用。敏感内容只保留在本地，不为统计上传原始事件。

完成 20 条后执行 `DEC-01`。一个方向必须有多个可定位 session 的重复证据和明确的用户价值才能启动；单次轶事或“以后可能需要”不够。满足以下条件才启动对应扩展：

- 会话扫描实际成为瓶颈，再加索引。
- wrapper 看不到的工具/权限事件持续阻碍复核，再做 hooks/原生适配。
- 多人确实需要跨会话汇总，再评估 Dify/团队服务。
- 高保证客户明确要求对抗本机管理员，再做签名与远端见证。

如果没有一个方向满足门槛，正确决策是保持 CLI + 薄插件、修复具体缺陷并继续观察，而不是为了版本号扩张范围。

## 16. 风险台账

| 风险 | 触发信号 | 当前缓解 | 升级路径 |
|---|---|---|---|
| L1 可见性不足 | 复核频繁需要内部工具调用 | 能力矩阵、not_observable | M4 hooks/原生适配 |
| Windows 进程树失控 | 孙进程在 AFR 退出后继续运行 | M0 Job Object spike | 引入 `x/sys/windows` |
| 脱敏漏检 | 测试矩阵或真实会话发现明文 | 单一写盘门、fail-closed | 扩充 bounded detectors |
| 大仓库启动慢 | 基线超过用户可接受阈值 | Git 优先、预算、partial | 增量索引，仅在测量后 |
| 报告过大 | 浏览器加载/筛选卡顿 | 预览上限、统计化展示 | 分离查看器，非 v0.1 |
| 本机攻击者重算证据 | 高保证审计要求出现 | 明示威胁模型 | 系统密钥签名/远端见证 |
| 插件 API 漂移 | 宿主更新后 validation 失败 | 双宿主 CI 冒烟 | 版本范围与分包 |
| hook 并发破链 | 多事件同时到达 | v0.2 锁文件 | 单写者 ingest 进程 |

## 17. 变更规则

- 新需求先新增 `REQ-*`，不得直接塞进 TODO。
- 架构修正新增 `GAP-*` 或决策记录，并注明影响里程碑。
- TODO 完成只勾选任务；若验收标准改变，先改本规划。
- PlantUML 必须与本文件的状态、组件和插件边界一致。
- 原技术 DOCX 保留为来源，不在无明确请求时回写。

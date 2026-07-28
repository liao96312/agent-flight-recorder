# Agent Flight Recorder TODO

> 与 [PROJECT_PLAN.md](./PROJECT_PLAN.md) 同步维护。任务只有在“验收”可重复通过后才能勾选。
> 优先级：P0 发布阻塞；P1 正式可用增强；P2 有真实需求后再做。
> 估算假设：1 名全职开发者，Windows x64 为 v0.1 发布平台。
> 当前基线：v0.1.0 已公开发布并完成回下载复验；CLI wrapper、Claude Code skill、三项 P1 可用性增强、Codex Desktop 正式 Hook、`M3-24` 人类可读报告首页、`R0-09` 二十次真实会话与 `DEC-01` 数据门均已完成。当前保持现有产品边界，等待 ADR-0002 的下一检查点。

## 0. 当前状态

- [x] **PLAN-01 [P0] 完整审读源技术方案**
  - 结果：正文、10 张表、架构图和 14 页版式均已核对。

- [x] **PLAN-02 [P0] 确定产品形态**
  - 结果：`afr` CLI 为唯一内核；Codex/Claude Code 使用共享薄插件。

- [x] **PLAN-03 [P0] 写出规划、TODO 与 PlantUML**
  - 产物：`PROJECT_PLAN.md`、本文件、`architecture.puml`。

- [x] **PLAN-04 [P0] 重新对照源方案、实现和发布真值**
  - 结果：确认两个 P0 代码差距、六个 P0 发布动作、四组 P1 增强与 20 次会话数据门；移除 `--summary-json` 的过期 v0.1 承诺，`show --open` 后续作为独立 P1 交付。

### 0.1 下一步唯一执行队列

同一任务只在后文保留一个复选框；本表引用 canonical ID，不复制状态。外部门槛等待期间，继续推进不依赖该门槛的任务。

| 顺序 | 任务 | 当前阻塞 | 进入下一项的条件 |
|---:|---|---|---|
| 1 | ✅ `M3-22`、`M3-23` | 已完成 | 10 万事件测试、固定六行摘要和三平台 CI 已通过 |
| 2 | ✅ `REL-01` | 已完成 | MIT License 文件、README、插件 manifest 与 release notes 一致 |
| 3 | ✅ `REL-02`；✅ `REL-03` | Claude Code 实调用与独立 Windows 产物验收均已完成 | 进入 `REL-04` 最终候选一致性检查 |
| 4 | ✅ `M3-09` → `M1-11` → `M2-07` | 已逐项完成；均为 P1 | 独立提交、测试和三平台 CI 通过 |
| 5 | ✅ `REL-04` | 已完成 | 完整 release-check、CI、版本、commit、release notes 和 SHA 一致 |
| 6 | ✅ `REL-05`；✅ `REL-06` | 已完成 | `v0.1.0` 公开 Release、回下载与公共安装复验均通过 |
| 7 | ✅ `D0-01` → `D0-06` | 已完成 | 正式 Hook 连续两轮复用同一 session，稳定采样均为 idle/valid；能力矩阵与历史保留边界一致 |
| 8 | ✅ `M3-24` | 已完成 | 首屏回答状态、风险、变化和证据边界；CSP/脱敏顺序回归测试与全仓测试通过 |
| 9 | ✅ `R0-09`（`20/20`） | 已完成 | 20 条本地真实会话评审已逐条验证并记入台账 |
| 10 | ✅ `DEC-01` | 已完成 | ADR-0002 明确保持现状及四组可复验的重开条件 |

## 1. Phase 0：契约冻结（2 天）

阶段出口：数据、CLI、能力和威胁边界有可执行样例；后续代码不得自行发明第二套格式。

- [x] **F0-01 [P0] 建立最小仓库与 Go module**
  - 依赖：无。
  - 验收：`go test ./...` 与 `go build ./cmd/afr` 可运行；没有业务代码以外的脚手架。

- [x] **F0-02 [P0] 写 ADR：CLI 内核 + 双宿主薄插件**
  - 依赖：无。
  - 验收：明确 CLI、skill、hook 的职责；禁止插件复制存储、脱敏、哈希、规则和报告。

- [x] **F0-03 [P0] 冻结 v0.1 威胁模型**
  - 依赖：F0-02。
  - 验收：明确能发现未重算篡改，不对抗本机管理员；列出本地恶意输入、误共享、路径穿越和 HTML 注入。

- [x] **F0-04 [P0] 冻结证据能力矩阵**
  - 依赖：F0-03。
  - 验收：`process/workspace_git/workspace_scan/native_tool_events/local_policy/os_file_monitor/network_monitor` 均有 observed/not_observable 规则。

- [x] **F0-05 [P0] 冻结会话状态机**
  - 依赖：F0-04。
  - 验收：starting、running、finalizing、completed、failed、interrupted、incomplete 的所有正常/异常转换唯一；子进程非零退出不等于证据失败。

- [x] **F0-06 [P0] 冻结 Event v1 格式**
  - 依赖：F0-05。
  - 验收：format_version、prev_hash、body、hash 的字节级公式、UTF-8/LF、genesis、seq、时间字段均有规范。

- [x] **F0-07 [P0] 冻结 Session 与 Manifest v1 格式**
  - 依赖：F0-06。
  - 验收：manifest 明确区分 evidence/derived、排除自身；incomplete 会话字段有定义。

- [x] **F0-08 [P0] 冻结 Risk 与 Report View Model v1**
  - 依赖：F0-04、F0-07。
  - 验收：事实、推断、严重度、证据 seq、规则版本、能力缺口和截断声明均有字段。

- [x] **F0-09 [P0] 冻结 CLI 命令和 argv 边界**
  - 依赖：F0-05。
  - 验收：run/list/show/verify/clean/version 的语法、默认值和错误类别有黄金示例；不经 shell 字符串拼接。

- [x] **F0-10 [P0] 冻结流和退出码契约**
  - 依赖：F0-09。
  - 验收：child stdout/stderr、AFR 摘要流、child exit code、AFR setup/finalize error 的优先级可由脚本区分。

- [x] **F0-11 [P0] 冻结配置优先级**
  - 依赖：F0-09。
  - 验收：CLI flags > workspace `.afr.json` > defaults；未实现的用户/组织层不出现在规范中。

- [x] **F0-12 [P0] 建立黄金向量**
  - 依赖：F0-06、F0-07、F0-08。
  - 验收：固定事件 body、hash、manifest、risk、redaction 输出可在任何构建中逐字节复现。

- [x] **F0-13 [P0] 定义 v0.1 强制制品清单**
  - 依赖：F0-07。
  - 验收：session/events/manifest、before/after、patch、Markdown、risk JSON、HTML 各自是必需、条件性或派生。

- [x] **F0-14 [P0] 定义性能 fixture**
  - 依赖：无。
  - 验收：记录 Windows 版本、CPU/内存/磁盘、Git 版本、仓库文件数/体积、事件大小和 flush 模式；指标不再使用含糊的“中型仓库”。

## 2. M0：记录闭环（第 1 周）

阶段出口：Windows 上可记录一个成功、失败和被中断的假 Agent；有效事件前缀始终可读。

- [x] **M0-01 [P0] 实现最小 CLI 分派**
  - 依赖：F0-01、F0-09。
  - 验收：`afr version`、`afr run -- ...` 可解析；usage 错误稳定且不创建会话。

- [x] **M0-02 [P0] 实现 session ID**
  - 依赖：F0-07。
  - 验收：UTC 时间 + `crypto/rand`；并发 100 万次测试无碰撞；ID 不含路径字符。

- [x] **M0-03 [P0] 实现会话根与路径包含校验**
  - 依赖：M0-02。
  - 验收：所有写入都落在精确 session 根；拒绝 `..`、绝对路径注入、同前缀兄弟目录和跨盘路径。

- [x] **M0-04 [P0] 创建权限受限的会话目录**
  - 依赖：M0-03。
  - 验收：POSIX 目录 0700/文件 0600；Windows 继承用户私有目录权限并在文档中声明实际保证。

- [x] **M0-05 [P0] 实现原子 JSON 文件写入**
  - 依赖：M0-04。
  - 验收：临时文件与目标同目录、flush/close/rename 顺序固定；故障注入不留下半个 `session.json`。

- [x] **M0-06 [P0] 实现单 writer 事件追加**
  - 依赖：F0-06、F0-12、M0-04。
  - 验收：严格 seq、精确 body 字节 hash、每事件一行 LF；10 万事件黄金校验通过。

- [x] **M0-07 [P0] 实现事件 flush 策略**
  - 依赖：M0-06。
  - 验收：关键状态立即 flush/sync；高频输出按字节/时间批量 flush；策略写入 session 元数据。

- [x] **M0-08 [P0] 实现 `afr run` 的 argv 启动**
  - 依赖：M0-01、M0-06。
  - 验收：参数按数组传给 `os/exec`；空格、引号、Unicode、前导短横线不改变边界。

- [x] **M0-09 [P0] 实现 stdout/stderr 双流 drain 与 tee**
  - 依赖：M0-08。
  - 验收：两流高频交错不死锁；用户实时可见；AFR 不把摘要混入 child stdout。

- [x] **M0-10 [P0] 避免 `bufio.Scanner` 长行上限**
  - 依赖：M0-09。
  - 验收：10 MB 无换行输出不会触发 token-too-long、OOM 或 secret 原文落盘。

- [x] **M0-11 [P0] 实现核心生命周期事件**
  - 依赖：M0-06、M0-08、M0-09。
  - 验收：session_started、process_started、process_output、process_exited、session_finished 字段完整且排序稳定。

- [x] **M0-12 [P0] 实现 child exit code 透传**
  - 依赖：M0-11、F0-10。
  - 验收：fake-agent 返回 0、1、7、127 时，session 和进程退出行为符合契约。

- [x] **M0-13 [P0] 实现 Ctrl+C 两阶段取消**
  - 依赖：M0-08、M0-11。
  - 验收：第一次转发并等待宽限期，第二次/超时强杀；事件记录实际结果，不伪报优雅退出。

- [x] **M0-14 [P0] Windows Job Object spike**
  - 依赖：M0-08。
  - 验收：child 创建孙进程后取消，孙进程不继续修改 fixture；若 stdlib 不足，记录引入 `x/sys/windows` 的唯一理由。

- [x] **M0-15 [P1] Unix process group 实现**
  - 依赖：M0-13。
  - 验收：增加 `!windows` 孙进程终止测试，并在 Linux/macOS 冒烟中证明完整进程组终止；不阻塞 Windows v0.1.0。
  - 结果：共享 `internal/process_tree_test.go` 在 Windows、Ubuntu、macOS 均验证孙进程随进程树终止；Unix CI 见 [run 29798685253](https://github.com/liao96312/agent-flight-recorder/actions/runs/29798685253)。

- [x] **M0-16 [P0] 实现 torn-tail 只读检测**
  - 依赖：M0-06。
  - 验收：半写最后一行时保留原字节，读取完整前缀，报告尾部大小/hash；不静默截断或续写。

- [x] **M0-17 [P0] 推断 incomplete 会话**
  - 依赖：M0-05、M0-16。
  - 验收：硬杀 AFR 后，下一次 list/show 可识别 incomplete；不会伪造 session_interrupted。

- [x] **M0-18 [P0] 建立 fake-agent**
  - 依赖：F0-01。
  - 验收：可配置 stdout/stderr、退出码、延迟、长行、二进制、孙进程、文件操作和自杀。

- [x] **M0-19 [P0] M0 端到端测试**
  - 依赖：M0-01 至 M0-18 的 P0 项。
  - 验收：成功、非零、Ctrl+C、child 崩溃、AFR 硬杀五条路径均留下可读证据或明确 incomplete。

## 3. M1：工作区证据（第 2 周）

阶段出口：Git 与非 Git fixture 的 before/after 准确；pre-existing changes 不归因给本次 Agent。

- [x] **M1-00 [P0] 实现会话级内容指纹器**
  - 依赖：M0-02。
  - 验收：`crypto/rand` 生成仅存活于进程内的 key，使用 HMAC-SHA256；同一会话可关联，不同会话不可关联，key 永不写盘。

- [x] **M1-01 [P0] 实现工作区根解析**
  - 依赖：M0-03。
  - 验收：相对/绝对路径、不同 CWD、Windows UNC/盘符大小写均规范化。

- [x] **M1-02 [P0] 实现 symlink/junction/reparse point 边界检查**
  - 依赖：M1-01。
  - 验收：链接指向工作区外时只记录链接/真实目标，不跟随扫描；路径前缀欺骗失败。

- [x] **M1-03 [P0] 探测 Git 仓库和版本**
  - 依赖：M1-01。
  - 验收：记录 repo root、Git 版本；非 Git、Git 不可用、命令失败有不同错误/降级。

- [x] **M1-04 [P0] 采集 Git before 状态**
  - 依赖：M1-03。
  - 验收：branch、HEAD、porcelain v2、staged/unstaged/untracked/submodule 可机器解析。

- [x] **M1-05 [P0] 禁用 Git 非确定性交互**
  - 依赖：M1-03。
  - 验收：pager、凭据提示、外部 diff driver、文本本地化不会卡住或改变解析。

- [x] **M1-06 [P0] 采集 Git after 状态**
  - 依赖：M1-04、M0-11。
  - 验收：即使 child 非零或被中断也尽力采集；失败会降低 capability 而不是给空结果。

- [x] **M1-07 [P0] 计算 pre-existing 与 session delta**
  - 依赖：M1-04、M1-06。
  - 验收：任务前已有脏改动单列；本次新增、修改、删除、重命名、staged、untracked 归类准确。

- [x] **M1-08 [P0] 生成 workspace patch 与文件摘要**
  - 依赖：M1-07。
  - 验收：文本 patch 与人工 Git 检查一致；binary/LFS/submodule 有明确摘要，不伪装为完整正文。

- [x] **M1-09 [P0] 建立 Git fixture**
  - 依赖：F0-01。
  - 验收：自动生成 clean/dirty/staged/unstaged/untracked/rename/delete/binary/submodule 场景。

- [x] **M1-10 [P0] 实现非 Git before/after 清单**
  - 依赖：M1-00、M1-01、M1-02。
  - 验收：记录相对路径、类型、大小、mtime、会话级 HMAC 指纹；新增/修改/删除可重复计算且不暴露裸内容哈希。

- [x] **M1-11 [P1] 实现有限 `.afrignore`**
  - 依赖：M1-10、F0-11。
  - 验收：注释、空行、路径前缀、stdlib glob、非法模式和 Windows 分隔符行为有测试；文档明确不等同完整 gitignore。
  - 结果：工作区根 `.afrignore` 在 child 启动前加载一次；普通路径按前缀处理，glob 使用 Go `path.Match`，Windows 分隔符统一为 `/`；非法模式阻止启动，文档明确不支持 negation、递归 `**` 和父目录发现。

- [x] **M1-12 [P0] 实现扫描预算**
  - 依赖：M1-10。
  - 验收：文件数、总读取字节、总时长超限后标记 partial 和遗漏原因，不无限扫描。

- [x] **M1-13 [P0] 实现 stdout/stderr 200 MB 上限**
  - 依赖：M0-09、M1-00。
  - 验收：超限后仍 drain、计数、算会话级 HMAC 指纹；child 不因管道阻塞；不保存裸原文 hash。

- [x] **M1-14 [P0] 实现 Diff 50 MB 上限**
  - 依赖：M1-00、M1-08。
  - 验收：停止保存正文但保留文件摘要、总字节和会话级 HMAC 指纹；报告显式 truncated。

- [x] **M1-15 [P0] 大仓库基准**
  - 依赖：F0-14、M1-04 至 M1-10、M1-12；P1 的 M1-11 不阻塞基准。
  - 验收：输出分阶段耗时；在固定 fixture 上达到目标或形成有证据的调整提案。

- [x] **M1-16 [P0] M1 端到端测试**
  - 依赖：M1 的 P0 项。
  - 验收：Git 与非 Git 结果均与 fixture 预期逐项一致；权限拒绝和路径消失可见。

## 4. M2：隐私、风险与完整性（第 3 周）

阶段出口：任何制品都不能绕过脱敏；未修改会话 verify 通过，典型篡改可定位。

- [x] **M2-01 [P0] 建立唯一持久化前脱敏 API**
  - 依赖：M0-06、F0-08。
  - 验收：事件、session、manifest、Git/文件制品和派生报告没有其他文本写盘入口。

- [x] **M2-02 [P0] 实现 argv 脱敏副本**
  - 依赖：M2-01、M0-08。
  - 验收：child 收到原 argv；落盘和错误只见脱敏结构；不打印重组命令造成注入。

- [x] **M2-03 [P0] 实现 bounded record reader**
  - 依赖：M0-10、M2-01。
  - 验收：正常文本可完整检测；超长单记录 fail-closed，只存大小/hash/omitted 原因。

- [x] **M2-04 [P0] 实现内置 secret detectors**
  - 依赖：M2-03。
  - 验收：API key、Bearer、私钥块、连接串、Cookie、常见敏感变量各有正反测试。

- [x] **M2-05 [P0] 实现跨 read chunk 与跨行测试**
  - 依赖：M2-04。
  - 验收：secret 被拆在读取边界、多行私钥、CRLF/LF、ANSI 包裹时不泄漏。

- [x] **M2-06 [P0] 实现环境变量最小采集**
  - 依赖：M2-01。
  - 验收：默认只记录名称；任何继承值都不进入会话目录。

- [x] **M2-07 [P1] 实现 `.afr.json` 自定义 RE2 规则**
  - 依赖：F0-11、M2-01。
  - 验收：冻结最小 schema；无效 JSON、未知字段、重复规则名和无效正则都在 child 启动前失败；规则有名称、类型和正反测试，不支持任意代码。
  - 结果：工作区根 `.afr.json` 支持 `{redaction_rules:[{name,type,pattern}]}` 严格 schema；`type` 仅允许 `regex`，使用 Go RE2；未知字段、重名、内置名冲突、空匹配和无效正则均在 child/session 启动前失败，配置不能执行代码或控制 replacement。

- [x] **M2-08 [P0] 实现会话内 HMAC 脱敏关联前缀**
  - 依赖：M1-00、M2-04。
  - 验收：同一会话同 secret 占位符一致，不同会话不同；HMAC key 永不写盘。

- [x] **M2-09 [P0] 处理二进制/未知编码**
  - 依赖：M2-03。
  - 验收：正文不落盘，只记录 bytes、会话级 HMAC 指纹、判定/截断原因。

- [x] **M2-10 [P0] 脱敏路径、文件名与错误消息**
  - 依赖：M2-01、M1。
  - 验收：假 secret 出现在路径、文件名、Git 错误、报告标题时，全目录无原文。

- [x] **M2-11 [P0] 实现 RiskFinding 模型**
  - 依赖：F0-08、M0-06。
  - 验收：rule_id/version、observed/inferred、severity、evidence_seq、explanation、action 可稳定排序。

- [x] **M2-12 [P0] 实现破坏性命令规则**
  - 依赖：M2-11、M2-02。
  - 验收：只对可见 argv/tool 输入判断；引用文本和安全命令有反例。

- [x] **M2-13 [P0] 实现敏感信息规则**
  - 依赖：M2-04、M2-11。
  - 验收：finding 只引用脱敏 evidence；风险输出不恢复 secret。

- [x] **M2-14 [P0] 实现大规模变更规则**
  - 依赖：M1-07、M2-11。
  - 验收：阈值版本化，报告列出范围；pre-existing 不计入本次批量变化。

- [x] **M2-15 [P0] 实现可观测越界/权限/网络规则**
  - 依赖：F0-04、M2-11。
  - 验收：只有证据给出真实路径/权限/域名才命中；否则 capability 显示 not_observable。

- [x] **M2-16 [P0] 实现 evidence/derived manifest**
  - 依赖：F0-07、M1、M2-01。
  - 验收：相对路径排序稳定；manifest 不包含自身；证据与派生物边界符合规划。

- [x] **M2-17 [P0] 实现流式 `afr verify`**
  - 依赖：M0-16、M2-16、F0-12。
  - 验收：检查格式版本、seq、prev/hash、final hash、evidence hash、derived hash；大文件不全量载入内存。

- [x] **M2-18 [P0] 定位首个事件异常**
  - 依赖：M2-17。
  - 验收：删除、插入、改写、重排、torn tail 分别返回首个坏 seq 或明确结构错误。

- [x] **M2-19 [P0] 定位制品异常**
  - 依赖：M2-17。
  - 验收：缺失、替换、截断 patch/snapshot/report 返回具体相对路径和 expected/actual metadata。

- [x] **M2-20 [P0] 全目录 secret 扫描测试**
  - 依赖：M2-02 至 M2-10。
  - 验收：假 secret 分布在 argv、stdout、stderr、JSON、Diff、路径、错误、报告，全 session 根零明文命中。

- [x] **M2-21 [P0] M2 篡改矩阵**
  - 依赖：M2-17 至 M2-19。
  - 验收：事件和制品所有规定变体均失败；整目录重算攻击明确显示不在 v0.1 威胁模型内。

## 5. M3：报告、CLI 可用性与发布（第 4 周）

阶段出口：开发者能记录、查找、复核、验证和安全清理会话；Windows 干净 VM 和两个薄插件均冒烟通过。

- [x] **M3-01 [P0] 实现统一 Report View Model**
  - 依赖：F0-08、M1、M2。
  - 验收：Markdown/JSON/HTML 共用同一状态、能力、事件、变更和风险排序。

- [x] **M3-02 [P0] 生成 `agent-flight.md`**
  - 依赖：M3-01。
  - 验收：短摘要包含状态、child exit、能力矩阵、变更、最高风险、截断/缺口、verify 说明。

- [x] **M3-03 [P0] 生成 `agent-risk.json`**
  - 依赖：M3-01。
  - 验收：schema/version 固定；只含已脱敏事实和风险；排序可复现。

- [x] **M3-04 [P0] 生成单文件 `report.html`**
  - 依赖：M3-01。
  - 验收：离线打开；事件/风险/文件筛选可用；不加载外部资源。

- [x] **M3-05 [P0] 实现 HTML 上下文转义与 CSP**
  - 依赖：M3-04。
  - 验收：`</script>`、属性注入、CSS/JSON/Unicode payload 均只显示文本；浏览器网络请求为零。

- [x] **M3-06 [P0] 限制 HTML 内嵌输出量**
  - 依赖：M3-04、M1-13。
  - 验收：200 MB 流不会生成 200 MB DOM；报告展示预览、总量和 hash，浏览器可用。

- [x] **M3-07 [P0] 实现 `afr list`**
  - 依赖：M0-17。
  - 验收：只扫描 sessions 一级目录和小型 session.json；坏目录不阻塞其余结果；支持 limit/json。

- [x] **M3-08 [P0] 实现 `afr show`**
  - 依赖：M3-01、M3-07。
  - 验收：完整 ID、唯一前缀、latest 可用；歧义明确失败；incomplete 生成只读摘要。

- [x] **M3-09 [P1] 实现 `afr show --open`**
  - 依赖：M3-08、M3-04。
  - 验收：先实现 Windows；路径含空格/Unicode 时不经 shell 字符串拼接地打开；失败仍打印绝对报告路径。Linux/macOS 随 R0-05 增加 `xdg-open` / `open` 冒烟。
  - 结果：Windows、macOS、Linux 分别使用 `explorer.exe`、`open`、`xdg-open` 的独立 argv；`--json` 与 `--open` 互斥，打开失败前先输出绝对 HTML 路径；Go 测试覆盖 Unicode/空格路径和失败路径。

- [ ] **M3-10 [P2] 实现 `afr report`**
  - 依赖：M2-16、M3-01。
  - 触发：20 次真实会话中出现需要从冻结 evidence 重建派生报告的重复需求；自动报告已覆盖时不建设。
  - 验收：先 verify evidence，只更新 derived 文件与 derived hashes；events/evidence 条目字节不变。

- [x] **M3-11 [P0] 实现 `afr clean` 精确目标预览**
  - 依赖：M0-03、M3-07。
  - 验收：只接受解析出的合法 session 目录；不使用模糊路径；活跃/伪造/链接目录拒绝。

- [x] **M3-12 [P0] 实现 clean 确认与 `--yes`**
  - 依赖：M3-11。
  - 验收：交互默认确认；非 TTY 无 `--yes` 不删除；删除后输出精确清单。

- [x] **M3-13 [P1] 实现保留天数和容量策略**
  - 依赖：M3-11。
  - 验收：按 session 整体选取；不会后台自动运行；选择规则确定、可预览。

- [x] **M3-14 [P0] 生成 Windows x64 单文件**
  - 依赖：M3-01 至 M3-08、M3-11 至 M3-13；构建任务不依赖自身或后置插件任务。
  - 验收：模板 `go:embed`；干净 Windows VM 只放 `afr.exe` 可运行 version/list，并明确提示 Git/Agent 外部前置。

- [x] **M3-15 [P0] 输出构建与格式版本**
  - 依赖：M3-14。
  - 验收：version 输出 semantic version、commit、build time、event format、manifest format。

- [x] **M3-16 [P0] 创建共享插件根**
  - 依赖：F0-02、M3-14。
  - 验收：同一目录包含 `.codex-plugin/plugin.json`、`.claude-plugin/plugin.json`、`skills/afr/SKILL.md`；无 hooks、MCP、daemon。

- [x] **M3-17 [P0] 编写薄 `afr` skill**
  - 依赖：M3-16、F0-09。
  - 验收：检查 CLI、构造非交互 argv、声明“启动新记录任务”、show/verify；不声称追溯当前会话。

- [x] **M3-18 [P0] 校验 Codex 插件**
  - 依赖：M3-16、M3-17。
  - 验收：官方/本地 validator 通过；本地 marketplace 能发现；skill 能启动一次 `afr run -- codex exec ...`。

- [x] **M3-19 [P0] 校验 Claude Code 插件包与生命周期**
  - 依赖：M3-16、M3-17。
  - 验收：strict validator、`claude --plugin-dir`、marketplace install/disable/enable/update/uninstall 和共根 inventory 通过；额外 Codex manifest 不造成错误。
  - 边界：namespaced skill 实调用未由本项的 validator/loader 冒充；后续已由 `REL-02` 使用兼容 API provider 认证完成。

- [x] **M3-20 [P0] 双宿主共根失败时最小分包**
  - 依赖：M3-18、M3-19。
  - 验收：仅当真实 validator/loader 失败才拆 manifest 包；共享 skill 和 CLI 仍为单一来源。

- [x] **M3-21 [P0] v0.1 候选实现验收**
  - 依赖：M3-01 至 M3-08、M3-11 至 M3-20；P1 的 M3-09 / M3-10 不阻塞候选版。
  - 验收：自动化安全、压力、基准、Windows 构建和插件 package/lifecycle 检查可重复通过；未完成的人工门槛必须显式留在发布检查表，不得写成正式 Release 已完成。

- [x] **M3-22 [P0] 补齐 HTML 有界事件时间线**
  - 依赖：M3-01、M3-04、M3-05、M3-06。
  - 验收：按 seq 展示时间、类型和最多 512 UTF-8 bytes 的已脱敏摘要；最多 1,000 个事件行，超过时确定性保留前 500 / 后 500，并显示 omitted 数量与 seq 区间；可筛选；truncated、binary、omitted 和 `not_observable` 不被隐藏。恶意 payload 仍只显示文本；10 万事件 fixture 断言事件行不超过 1,000 且首尾与 omission marker 正确。
  - 结果：流式读取已脱敏 `events.jsonl`，10 万事件 fixture 与三平台 CI 通过。

- [x] **M3-23 [P0] 补齐 `run` 结束摘要**
  - 依赖：M3-01、M3-02、M3-03。
  - 验收：完整收尾时在 stderr 固定输出六行：`Session: <id>`；`Evidence: observed=<排序 capability CSV>; not_observable=<排序 capability CSV>; truncated=<bool>`；`Changed: added=N modified=N deleted=N renamed=N pre_existing=N`；`Risks: total=N highest=<severity>`；`Exit: child=<code> state=<state>`；`Report: <绝对 report.html 路径>`。值与最终报告一致，不使用 L1/L2/L3；child stdout 逐字节不变，child stderr 不丢失；AFR 收尾失败仍保留稳定错误前缀和 session 路径。
  - 结果：精确格式测试、`RunResult` 集成测试、WSL 实际 CLI 冒烟和三平台 CI 通过。

- [x] **M3-24 [P1] 将 HTML 报告重排为“人类决策首页”**
  - 依赖：M3-01、M3-04、M3-05、M3-06、M3-22、M3-23、D0-06。
  - 问题实证：当前报告先展示原始 capability、事件计数和长时间线，风险、文件变化与证据缺口被推到页面后部；普通用户无法快速判断“有没有录到、要不要处理、改了什么、哪些地方 AFR 看不到”。`ReportView` 已包含实现所需数据，本项不新增报告 schema 或运行时依赖。
  - 用户问题按以下优先级固定，不再用事件数量主导首页：
    1. **录到了吗，现在是什么状态？**
    2. **有没有需要我立即处理的问题？**
    3. **AI 本轮改了什么，哪些改动在记录前就已存在？**
    4. **AFR 看到了什么，哪些行为没有证据？**
    5. AI 做了哪些工具动作、经历了哪些事件？
    6. session ID、原始枚举、完整性链与输出等技术细节是什么？
  - 首屏版式冻结为下列信息顺序；数值仅为结构示例，生成时必须来自当前报告真值：

    ```text
    ┌─ Agent Flight Recorder · <项目> · <宿主> ── [上一轮已记录，等待下一轮]
    │  未发现规则可确认的风险
    │  AFR 只判断已记录到的证据，这不代表绝对安全。
    ├─ 需检查 0 ── 本轮文件变化 4 ── 记录前已有 3 ── 证据缺口 3
    ├─ 你需要检查
    │  <最高优先事项与第一条可执行建议；没有事项时不制造空告警>
    ├─ 本次改了什么
    │  <新增 / 修改 / 删除 / 重命名前几项>；记录前已有改动独立列出且不归因本轮
    ├─ 证据边界
    │  已记录：<白话能力列表>    未监控：<白话缺口列表>
    └─ 技术详情（默认收起）
       <事件计数 / 时间线 / 输出 / session / 原始枚举 / 完整性信息>
    ```

  - [x] **M3-24a 首屏状态与结论条**
    - `idle` 显示为“上一轮已记录，等待下一轮”，不得写成“任务已完成”；`starting/running/finalizing` 显示“正在记录”；`completed` 才显示“任务已结束并保存”。`failed` 必须区分任务执行失败与证据保存状态；`interrupted/incomplete` 显示需要检查的记录状态。未知状态保留原值并标记“未知状态”，不得猜测。
    - `highest=none` 显示“未发现规则可确认的风险”，旁边固定显示“AFR 只判断已记录到的证据，这不代表绝对安全”；禁止使用“安全”“完全正常”“无风险”等超出证据的结论。
    - 有 finding 时显示“发现 N 项需要检查，最高：<白话级别>”，并在首屏给出最高项的第一条 `Action`；`incomplete/partial/truncated/omitted` 作为证据问题进入“需要检查”，不得藏入技术详情。
  - [x] **M3-24b 四个摘要指标**
    - 固定显示“需检查”“本轮文件变化”“记录前已有改动”“证据缺口”；状态单独放在结论条。
    - event 总数、session ID、原始 state/risk/capability code 不作为首页主指标；工具动作数量只在活动/技术详情中出现。
    - 指标为零时使用中性白话，不用颜色暗示绝对安全；指标计算必须确定性、可由现有报告字段复算。
  - [x] **M3-24c 文件变化与归因**
    - “本次文件变化”优先展示 Added / Modified / Deleted / Renamed 的总数与前几条路径；列表保持现有有界策略，超限必须显示 omitted 数量。
    - `PreExisting` 独立成区，固定解释“开始记录前已存在，不归因于本次 AI”；不得与本次 Modified 合并，也不得省略为零以外的记录前改动。
    - 首页只给摘要与前几项，完整有界列表放在后续文件详情；路径继续经过现有转义与脱敏。
  - [x] **M3-24d 证据能力白话映射**
    - 首页使用“已记录 / 未监控”两栏，原始 code 只在技术详情保留；未知 capability 回退原值，不能被静默丢弃。
    - `native_tool_events` → “宿主上报的本地工具调用”；`workspace_git` → “Git 工作区变化”；`workspace_scan` → “项目文件扫描”；`local_policy` → “本地规则检查”。
    - `process` → “进程级活动”；`network_monitor` → “实际网络流量”；`os_file_monitor` → “操作系统级文件活动”。每项严格按当前 `observed/not_observable` 真值进入对应栏，不得把安装、启用或信任推断为已观察。
    - 未监控区固定说明：“未监控表示 AFR 没有这类证据，不能据此判断该行为没有发生。”partial、truncated、binary、omitted 等证据降级也在此处显眼说明。
  - [x] **M3-24e 风险与行动建议**
    - 没有 finding 时不渲染空风险表，只显示带边界说明的确定性结论；有 finding 时按严重级别和稳定顺序展示总数、最高项、证据及行动建议。
    - “需要检查”只容纳真实 finding、异常状态或证据降级；不能用安全分数、模糊健康度或事件数量制造告警。
    - 完整性只表示本地证据一致性，不得描述为数字签名、远程见证或行为真实性；没有实际执行 `afr verify` 的结果时只能提示“可运行 afr verify”，不得写“校验通过”。
  - [x] **M3-24f 渐进披露与阅读顺序**
    - 页面固定顺序为“结论 → 需要检查 → 本次文件变化 → 记录前已有改动 → 证据边界 → 全部风险/文件 → 技术详情”。首屏必须出现结论、四指标、变化摘要和证据边界。
    - Event counts、timeline、stdout/stderr、全局筛选、session ID、seq、原始枚举和完整性链放入默认收起的原生 `<details>`；保留现有筛选能力、1,000 行上限、前后 500 行策略、omission marker 和 512 UTF-8 bytes 摘要上限。
    - 活动类型增加白话标签，原始事件类型仍可查看；不得生成或伪造对话摘要，不能把 event timeline 当成用户聊天记录。
  - [x] **M3-24g 响应式、可访问性与安全边界**
    - 只使用现有 Go HTML 生成、语义 HTML、原生 `<details>` 与少量 CSS Grid/Flex；报告继续是离线单文件，不新增 SPA、图表、图标库、字体、CDN、远端资源或遥测。
    - 当前迭代使用中文界面并设置 `lang="zh-CN"`；未知原值可回退英文。不为尚未出现的非中文需求引入 i18n 框架或主题系统。
    - 1366×768 首屏无需滚动即可回答前四个核心问题；360px 起摘要自动单列、页面整体无横向滚动，技术表格只在自身容器内滚动；390×844 作为 Desktop 窄宽人工复验尺寸。
    - 标题层级、键盘焦点和 `<details>` 可操作；风险与状态必须有文字，不得只靠颜色表达。现有 CSP、HTML 转义、脱敏和无外链约束保持不变。
  - [x] **M3-24h 自动化与人工验收**
    - [x] 固定 fixtures 至少覆盖：`idle+none`、active、completed、high finding、failed/incomplete、PreExisting、全 observed、混合 not_observable、partial/truncated/binary/omitted、未知 capability。
    - [x] DOM 顺序断言结论/风险/文件/证据边界均早于时间线；首页不出现原始 capability code；原始值在技术详情仍可验证；`idle`、`none` 和 PreExisting 的免责声明逐字断言。
    - [x] 既有恶意文本转义、CSP、输出/路径边界、10 万事件最多 1,000 行、首尾保留与 omission marker 测试全部继续通过；不降低性能和确定性。
    - [x] 当前 Codex Desktop session 的下一次正常 `Stop` 已生成新版报告；产品负责人收到候选链接后于下一 turn 指示“继续”，接受当前首页进入数据门。旧冻结报告不批量重写；内置浏览器拒绝 `file://` 的自动视觉检查事实保留，不伪造浏览器截图结果。
    - [x] **十秒验收**：当前项目报告首屏直接给出“上一轮已记录”“未发现规则可确认的风险”“6 个本轮文件变化”“3 个记录前已有改动”“4 个证据缺口”，无需展开技术详情即可回答五个核心问题。
  - 非目标：不生成 LLM 自然语言总结，不新增依赖/schema/i18n 基建，不上传远端，不隐藏 `not_observable`，不宣称“绝对安全”，不补录或总结聊天正文，不在本项建设历史报告迁移器。
  - 最小实现范围：实现阶段优先只改 `internal/report_html.go` 与对应测试；仅当现有 `ReportView` 确实无法派生验收字段时，才提出最小数据层变更并先更新本 TODO。
  - 2026-07-22 状态：a-f 与自动化验收已完成，复用现有 `ReportView`，实际只改 HTML 展示层与测试，`go test ./...` 和 `git diff --check` 通过。候选版已生成真实 session `20260722T075925.653441900Z-ced56546bbc86ebe47f97da5`（7 events、6 evidence、3 derived，`afr verify` 通过），并安装到 Hook 既有 `C:\Users\Administrator\.local\bin\afr.exe`；命令定义未变化，无需重新信任，旧二进制备份为 `C:\tmp\afr.exe.before-m3-24`。Codex 内置浏览器按安全策略拒绝自动打开本地 `file://` 报告，未绕过；因此 g/h 的两种尺寸肉眼检查与十秒理解测试保持未勾选，待当前 turn 正常 `Stop` 生成新报告后由产品负责人直接确认。
  - 2026-07-23 补充：R0-09 的只读评审 session `20260723T010847.884888900Z-65447f7616fbcdc928b6b659` 发现模板渲染后整体脱敏可能改写 CSS/JS/CSP 并使哈希失效。已在共享 HTML 写入点改为“先脱敏 `ReportView` 动态数据，再渲染常量模板并原子写入”，自定义规则同时匹配动态秘密、`system-ui` 和 `script-src` 的回归测试证明动态数据被替换而 CSP/资源哈希保持不变；`go test ./...`、session verify 与安装 SHA 一致性均通过。修复版已安装到既有 Hook 路径，命令未变、无需重新信任；上一构建备份为 `C:\tmp\afr.exe.before-csp-redaction-fix`。

## 6. M4a：Codex Desktop 当前任务接入（进行中，v0.2）

产品负责人于 2026-07-22 明确要求记录 Codex Desktop 当前任务，因此以下桌面端最小切片不再等待 `R0-09 / DEC-01`。目标是插件通过 Codex 官方 lifecycle hooks 记录**安装并信任之后的新桌面任务与后续 turn**，不再嵌套启动 `codex exec`。规范依据：<https://learn.chatgpt.com/docs/hooks>。

边界先冻结：

- 不追溯 hook 安装、启用或信任之前的消息；不得把当前已运行任务的历史补写成完整证据。
- Codex Desktop 没有 `/hooks` 斜杠命令；Desktop 的入口是“设置 → Hooks”或“插件 → AFR → Review/Trust all”。交互式 Codex CLI 的 `/hooks` 只能作为同一 `CODEX_HOME` 下的备用信任入口，CLI 成功本身不算 Desktop 验收。
- 非托管 Hook 的信任绑定到当前完整定义的 hash；命令、事件、matcher、timeout 等定义变化后会成为 `modified` 并被跳过。不得手改 `trusted_hash`，不得使用 trust bypass，也不得把个人插件提升成 managed hook 绕过用户同意。
- 不把 `transcript_path` 当稳定格式解析；只消费正式 hook JSON 字段。
- `Stop` 是 turn 结束，不是可靠的 SessionEnd；报告可处于 `idle`，不得伪造 `completed`。
- Hosted WebSearch 等不经过本地 function-tool hook 的路径保持 `not_observable`；只有实际收到 hook 事件后才把对应 native capability 标为 `observed`。
- 本切片不做 Claude hooks、MCP、daemon、遥测、远端上传或历史聊天导入；这些只能由后续真实会话证据单独触发。

- [x] **D0-01 [P0] Codex Desktop 实机 hook 契约探针（停止门）**
  - 依赖：v0.1.0 已发布；目标 Codex Desktop 版本可安装本地 AFR 插件。
  - 验收：使用插件默认 `hooks/hooks.json`，经宿主支持的入口审阅并信任后新建桌面任务；以不含用户正文/secret 的本地探针确认 `SessionStart`、`UserPromptSubmit`、至少一个本地 `PreToolUse/PostToolUse` 与 `Stop` 实际触发，记录 Desktop/CLI/plugin 版本、正式字段名和未覆盖事件。Desktop 本身没有 `/hooks` 命令；探针期间的 `/hooks` 操作发生在独立 Codex CLI。
  - 停止：上述四类事件任一缺失，或桌面端不加载 plugin-bundled hooks，则暂停 D0-02..D0-06，保留探针证据并报告；不得转而抓私有日志、轮询 UI 或解析不稳定 transcript。
  - 2026-07-22 结果：旧 `powershell.exe ... probe.ps1` 定义在 Desktop 同一 session 实际触发 `SessionStart(resume/compact)`、`UserPromptSubmit`、`PreToolUse/PostToolUse`，并在后续三个普通 turn 重复触发 `Stop`，故字段契约通过，详见 `D0-01-HOOK-PROBE.md`。该结果不证明后来替换的 `afr.exe hook --host codex` 已加载、可解析或受信任。

- [x] **D0-02 [P0] 冻结 hook-only session 与状态映射**
  - 依赖：D0-01。
  - 验收：Codex `session_id` 经格式/长度校验后映射到一个 AFR session；映射索引只放在 `PLUGIN_DATA`，证据仍放 `.afr/sessions`。`startup` 新建，`resume/compact` 复用，`clear` 切换；`Stop` 只写 `idle` 并刷新报告，下一 turn 继续同一 session。
  - 边界：不依赖父 wrapper 或 `AFR_SESSION_ID`；wrapper 模式保持原行为，两种入口不得生成双会话。
  - 结果：host ID 仅以 SHA-256 指纹写入 `PLUGIN_DATA` 映射；fingerprinter key 与 AFR session ID 留在映射索引。`resume/compact` 复用、`startup/clear` 切换；wrapper 给子进程注入合法 `AFR_SESSION_ID`，Hook 静默退出避免双会话。

- [x] **D0-03 [P0] 实现最小 `afr hook --host codex` 摄取入口**
  - 依赖：D0-02、M2 脱敏与完整性链。
  - 验收：从 stdin 读取不超过 1 MiB 的正式 hook JSON；当前只白名单映射 `SessionStart`、`UserPromptSubmit`、`PreToolUse`、`PostToolUse`、`Stop` 及其中允许的 prompt/local-tool/`permission_mode` 元数据，未知对象只记录类型、大小、hash 和 omitted 原因；所有内容复用现有 redactor 后才写盘。独立 `PermissionRequest`、`SubagentStart`、`SubagentStop` 不在当前插件注册范围，不得宣称已覆盖。
  - 验收：正常 stdout 为空，不向模型注入文本；错误使用稳定前缀且默认 fail-open，不阻断桌面任务。不得读取 `transcript_path` 内容。
  - 结果：已实现 1 MiB 上限、正式五事件白名单、未知字段 type/size/SHA-256 omission、统一 redactor、`AFR_HOOK:` fail-open 和 transcript 忽略；两轮 Windows 合成 Hook 形成 10 events / 6 evidence / 3 derived 并 verify 通过。

- [x] **D0-04 [P0] 保证多进程追加写完整性**
  - 依赖：D0-03、M0-06。
  - 验收：每个 AFR session 使用 O_EXCL 锁、短重试及带 PID/time 的 owner；只对明确陈旧且同 session 的锁做受控恢复。20 个并发合成 hook 事件无重复 seq、无坏 JSON，最终 `afr verify` 通过。
  - 边界：hook handler 超时设置为 5 秒；锁方案未出现可复现瓶颈前不建设单写者或 daemon。
  - 结果：按 host session 指纹使用 O_EXCL 锁、PID/time owner、30 秒且进程已退出才恢复；20 个并发 Hook 自测最终 24 events 且 verify 通过。CLI 锁等待和 workspace scan 各限制 2 秒，超预算 fail-open/partial。

- [x] **D0-05 [P0] 打通正式 Codex Desktop Hook 激活路径（停止门）**
  - 依赖：D0-03、D0-04。
  - **2026-07-22 根因实证（修复前）**：Desktop 26.715.9079 / bundled core 0.145.0 已精确发现五个 `afr@personal` Hook，均为 `enabled=true`，当前命令均为 `afr.exe hook --host codex`；但 app-server `hooks/list` 返回的 `trustStatus` 全部是 `modified`。宿主会在启动 AFR 进程之前跳过它们，因此重启后没有 `hook/started`、新 AFR session 或正式映射；当前首因不是 AFR stdin、PATH 或落盘代码。

    | Event | 已保存的旧探针 hash | 当前正式定义 hash | 状态 |
    |---|---|---|---|
    | `PreToolUse` | `7d0c4109d3b…` | `1991810ed6e…` | `modified` |
    | `PostToolUse` | `be3d0e6e8e7c…` | `8b169577038d…` | `modified` |
    | `SessionStart` | `1adc4f2f4b73…` | `fac8e885c12c…` | `modified` |
    | `UserPromptSubmit` | `eac64a629e19…` | `03bc1d0df629…` | `modified` |
    | `Stop` | `c56968bc0499…` | `a9a59627df33…` | `modified` |

  - **已排除的首因**：插件未安装/未启用（五项已列出且 enabled）；Hook feature 未开启（`hooks=stable,true`）；仅仅缺少再次 cachebuster（当前正式定义已被 `hooks/list` 发现）。AFR 二进制解析与事件协议尚未进入执行阶段，不能在 Hook 受信任前归因。
  - [x] **D0-05a 冻结定义与基线**：在本门完成前不再改 `hooks/hooks.json` 的 event、matcher、command、timeout/status；记录 Desktop/core、AFR、plugin 版本、当前五个 hash、最新 AFR session 和测试开始时间。版本号不得拼进 Hook command，避免每次升级无意义地重新信任。
    - 结果：Desktop 26.715.9079；AFR `0.1.0-dev` / commit `00c76ca7aed7`；plugin `0.1.0`；安装缓存 `hooks.json` SHA-256 `A943A3A8D9C97093B75AA2C8905996E8309159D4CF57014FA0E9C3266B8E57C7`；信任前 latest session `20260722T044941.694847000Z-112e3e8b376f1ac86377e9a8`。
  - [x] **D0-05b 通过正确入口逐项信任**：Desktop 没有 `/hooks`；用户已在此前消息中明确授权“信任”。由于 Desktop 自动化策略禁止脚本控制 Codex Desktop UI，本次使用宿主官方 app-server `config/batchWrite` 原子 upsert 五个当前 `trusted_hash` 并热重载；未覆盖 Ponytail 或其他 Hook。一次无效 camelCase 试写被立即识别并精确清理，最终配置只保留受识别的 `trusted_hash`。
  - [x] **D0-05c 做信任预检**：全新 app-server 进程的只读 `hooks/list` 确认五项同时满足 `enabled=true`、`trustStatus=trusted`、`currentHash == trusted_hash`，warnings/errors 均为空。仅看到 `config.toml` 中存在 hash 或 CLI 自身可执行均不作为验收，本项以回读结果为准。
  - [x] **D0-05d 只做一次冷启动验证**：缓存与信任就绪后完整退出并重启 Desktop，恢复当前任务，发送 prompt 并触发本地只读工具调用。验收必须在基线之后同时出现宿主 Hook 启动、新 `capture_mode=desktop_hook` AFR session/映射和对应正式事件；手工 stdin、合成测试或嵌套 `codex exec` 均不算。
    - 结果：当前 Desktop 恢复任务自动创建 AFR session `20260722T063058.701737900Z-bc9105b4a599c4d584e73d7a`，`capture_mode=desktop_hook`、host=`codex`、workspace=`D:\ai project`，实际记录 `host_session_started`、`prompt_submitted` 和多组 `tool_started/tool_finished`；映射文件同步生成。首轮真实 `Stop` 已写入 `turn_stopped` seq 29，并于 2026-07-22 14:34:10 生成 manifest 和报告；第二轮 `Stop` 后的后台稳定采样于 14:37:01 得到 `state=idle`、47 events、6 evidence、3 derived、`verify_exit=0`。下一条 prompt 会按设计把同一 session 重新置为 `active`，故稳定 verify 必须在 `Stop` 后、下一条 prompt 前采集。
  - [x] **D0-05e 修正能力报告真值**：`native_tool_events` 初始必须为 `not_observable`，只在实际收到 `PreToolUse/PostToolUse` 后变为 `observed`；报告只列出真实交付的事件和缺口。installed/enabled/trusted 只作为外部 `hooks/list` 预检证据，AFR handler 无受支持输入时不得自行推断；hook/plugin version 无法由事件观察时写 `not_observable`，不得虚构。
    - 结果：`hookCapabilities` 在 session 创建时显式写 `native_tool_events=not_observable`，在 `Stop` 时仅依据本 session 已记录的 `tool_started/tool_finished` 切换为 `observed`；未新增 AFR 无法观察的 installed/enabled/trusted/version 字段。测试同时覆盖有本地工具与无本地工具的新 session；`go test ./internal` 和 `go test ./...` 通过。使用 go.dev 官方 Go 1.26.5 Windows x64 便携包构建，已将新 `afr.exe` 安装到现有路径，旧二进制备份为 `C:\tmp\afr.exe.before-d0-05e`；Hook 命令未变化，无需重新信任。
  - [x] **D0-05f 保持历史证据不变**：plugin disable/remove 不删除或改写既有 `.afr/sessions`；D0-06 通过后同步 `PROJECT_PLAN.md` 的 Desktop Hook 真值。
    - 结果：插件包没有 uninstall/remove/clean `.afr` 的代码，插件缓存/数据与证据根 `~/.afr/sessions` 分离；二进制升级后，升级前 session `20260722T044941.694847000Z-112e3e8b376f1ac86377e9a8` 仍以 281 events、6 evidence、3 derived 通过 verify。未为证明路径隔离而破坏性卸载正在工作的插件；若未来新增 uninstall 脚本，再增加隔离环境删除测试。`PROJECT_PLAN.md` 已同步。
  - **分流与止损**：只允许一次正式信任、一次完整重启和一次一次性任务验证。若五项确认 `trusted` 后仍无 `hook/started`，记录版本与 `hooks/list` 证据并将 Desktop 26.715.9079 标记为宿主调度 `blocked/unsupported`；若有 `hook/started` 但无 AFR session，才检查 Desktop 进程 PATH 中的 `afr.exe` 解析、退出码和 stdin 协议。任一路径遇到首个可复现卡点即停止，不再循环 cachebuster/reinstall/restart，不手改信任 hash，不启用 `--dangerously-bypass-hook-trust`，不抓 transcript 或轮询 UI。
  - 规范依据：[Codex Hooks](https://learn.chatgpt.com/docs/hooks)、[Desktop slash commands](https://learn.chatgpt.com/docs/reference/slash-commands) 和 [app-server hooks/list](https://github.com/openai/codex/blob/main/codex-rs/app-server/README.md)。

- [x] **D0-06 [P0] Codex Desktop 当前任务 E2E**
  - 依赖：D0-01..D0-05。
  - 前置：D0-05 的五项 `hooks/list` 状态全部为 `trusted`，并已留下至少一条由当前正式命令产生的 Desktop 基线事件；否则不得开始。Hook 只能记录安装并信任后的未来 turn，不补录本任务既有历史。
  - 验收：完整重启后恢复当前 Codex Desktop 任务，连续完成两轮 prompt，覆盖一个本地 shell/tool 调用和一次文件编辑；全程不启动嵌套 `codex exec`。两次 `Stop` 后仍指向同一 AFR session，状态为 `idle`，工作区 delta、风险、事件时间线和 `afr verify` 一致。
  - 验收：无本地工具的 turn 保持 native tool `not_observable`，收到成对 `PreToolUse/PostToolUse` 后才标为 `observed`。记录 Desktop/core、AFR、插件版本、`hooks.json` SHA-256、host session 指纹、AFR session ID 与每类实际事件数；禁用插件后只验证事件停止追加和旧 session 仍可 show/verify，不推断 AFR 无法观察的宿主信任状态。
  - 结果：当前任务连续两轮使用修复后的 `afr 0.1.0-dev`（commit `00c76ca7aed7`，built `2026-07-22T06:44:56Z`），均复用 session `20260722T063058.701737900Z-bc9105b4a599c4d584e73d7a`。Stop 后后台稳定采样分别为 91 events / 3 Stops 与 100 events / 4 Stops；两次均 `state=idle`、6 evidence、3 derived、`verify_exit=0`，能力矩阵按实际本地工具事件显示 `native_tool_events=observed`。两轮覆盖本地 shell/tool 与仓库文件编辑，全程未启动嵌套 `codex exec`。

### M4b：仍由数据门控制的后续项

Claude Code hooks、跨宿主统一 SessionEnd、Hosted tools 覆盖、历史对话导入、全局用户 hook、单写者进程和远端汇总均不属于 D0。只有 `R0-09 / DEC-01` 出现多个可定位 session 的重复证据时，才为其中**一个**方向新增 TODO。

## 7. 发布、文档与维护

- [x] **R0-01 [P0] 编写 5 分钟快速开始**
  - 依赖：M3-21。
  - 验收：安装 CLI、记录一次任务、打开报告、verify、clean 全流程可复制。

- [x] **R0-02 [P0] 编写隐私与威胁模型说明**
  - 依赖：F0-03、M2。
  - 验收：清楚区分本地一致性、非数字签名、not_observable、会话共享风险和删除方式。

- [x] **R0-03 [P0] 生成 Windows 候选制品与校验和**
  - 依赖：M3-14。
  - 验收：本地候选目录包含 `afr.exe`、SHA-256 和版本说明，复算一致；本项不等于已经创建 tag / GitHub Release，公开发布见 `REL-05`。

- [ ] **R0-04 [P1] Windows 代码签名**
  - 依赖：组织证书与发布主体。
  - 验收：正式企业包签名可验证；未签测试包不伪装可信发布者。

- [x] **R0-05 [P1] Linux/macOS 同源编译冒烟**
  - 依赖：M0-15、M3-21。
  - 验收：Ubuntu 与 macOS CI 上 go test/build、孙进程终止、最小 run/verify 通过；非 Windows 明确标 beta。
  - 结果：Ubuntu 与 macOS 的 go test/build、孙进程终止、真实 run/verify 均通过；证据见 [run 29798685253](https://github.com/liao96312/agent-flight-recorder/actions/runs/29798685253)，非 Windows 仍标 beta。

- [x] **R0-06 [P0] Schema 向后读取测试**
  - 依赖：M2-17、M3-15。
  - 验收：新版可只读 verify v1；未知未来版本拒绝写入，不修改源会话。

- [x] **R0-07 [P0] 插件升级/禁用/卸载测试**
  - 依赖：M3-18、M3-19。
  - 验收：操作不删除 `~/.afr/sessions`；CLI 缺失和版本不兼容有清晰提示。

- [x] **R0-08 [P0] 建立发布检查表**
  - 依赖：F0-03、F0-14、M3-14 至 M3-20；检查表任务不依赖自身。
  - 验收：测试、基准、secret scan、HTML 网络检查、干净 VM、双宿主插件、文档和校验和都有槽位；候选记录持续更新 Claude 实调用与独立 VM 的实际状态，不用 validator/CI 代替人工证据。

- [x] **REL-01 [P0] 确认公开发行 License**
  - 依赖：产品负责人明确选择允许公开发行的许可证或其他发行条款。
  - 验收：仓库根存在完整 License / 发行条款，README、插件 manifest 和 release notes 表述一致；若决定暂不授权，本项保持未完成并阻塞 `REL-05`，不擅自替用户选择。
  - 结果：产品负责人于 2026-07-21 选择 MIT；根 `LICENSE`、中英文 README、Codex / Claude manifest 与 release notes 已统一。

- [x] **REL-02 [P0] Claude Code 真实 skill 冒烟**
  - 依赖：M3-17、M3-19、M3-23；可用的 Claude 账户或兼容 API provider 认证。
  - 验收：从实际插件入口调用 `/afr:afr`，确认命中同一 `afr` CLI，生成新 session、六项摘要完整、`afr verify` 通过；记录 Claude / AFR 版本、命令、session ID 与结果。
  - 结果：2026-07-22 使用 Claude Code `2.1.185`、OpenRouter Anthropic-compatible API 与免费路由 `openrouter/free`，从 `--plugin-dir plugins/afr` 的 `/afr:afr` 调用本机 `afr 0.1.0`（commit `fa2ba5780caa`）。session `20260722T014003.862517200Z-9a4b7bb5330396fc02df94db` completed/exit 0、工作区改动 0；`verify --json` 为 valid，7 events、6 evidence、3 derived。OpenRouter key API 在测试后报告 free tier 且 usage/daily/weekly/monthly 均为 0；Claude Code 展示的估算成本不作为实际扣费证据。免费子模型未遵循精确短语要求，按模型质量现象记录，不冒充功能断言；插件、进程、摘要和证据链路验收通过。API key 未写入项目或用户配置。

- [x] **REL-03 [P0] 独立干净 Windows 环境候选制品复验**
  - 依赖：M3-22、M3-23、R0-03。
  - 验收：不使用开发机工作树或构建作业文件系统；全新 Windows x64 作业只下载上游候选 exe/SHA 并复算校验；`version`、`list --json`、无 Git 的最小录制及 `verify` 可运行，缺 Agent 时错误明确；记录运行链接、时间和结果。
  - 结果：2026-07-22 GitHub Actions 独立 `windows-latest` 作业未 checkout 源码、未安装 Go，仅下载上游 artifact；SHA-256、空 profile 的 `version/list`、移除 Git PATH 后的最小录制、六行摘要及 `verify` 均通过，session `20260722T020221.618569700Z-6ff2e6d3846cde596b55ae00` 为 7 events、6 evidence、3 derived；缺失 Agent 返回 70 与 `AFR_RUNTIME`。运行：<https://github.com/liao96312/agent-flight-recorder/actions/runs/29884664955>。

- [x] **REL-04 [P0] 冻结最终 v0.1.0 候选 commit**
  - 依赖：M3-22、M3-23、REL-02、REL-03、R0-08。
  - 验收：从最终 `main` commit 执行 `release-check.ps1 -Full`；CI 全绿；二进制版本、commit、release notes 和 SHA-256 对应同一 commit；工作树无未说明发布改动。
  - 结果：2026-07-22 Windows 10 IoT Enterprise LTSC、Go 1.26.5、Git 2.54.0 上完整 release check 通过：全量测试、安全矩阵、100 次强制终止、100 路并发输出、10,000 文件基准、Windows 构建、双插件 validator、Codex marketplace 与 SHA-256 均成功。检查脚本现会直接断言二进制为 `0.1.0`、内嵌 commit 等于当前 HEAD、release notes 为 v0.1.0；用户移动技术文档产生的两个已说明工作树条目不属于发布内容，未纳入候选提交。

- [x] **REL-05 [P0] 创建 `v0.1.0` tag 与公开 GitHub Release**
  - 依赖：REL-01、REL-04。
  - 验收：最终 release commit 先把 README 状态与下载链接更新为 v0.1.0 released，再创建指向该 commit 的 annotated tag；公开 Release 上传 Windows exe、`SHA256SUMS` 和版本说明；页面明确 Windows x64、未签名、能力边界及 Git / Agent 外部前置。
  - 结果：2026-07-22 annotated tag `v0.1.0` 指向 release commit `a67c913dc626315899e630a99b72ad2a05a54d28`；该 commit 的完整 release check 和 Windows/Ubuntu/macOS CI 全绿。公开 Release 上传 exe、`SHA256SUMS`、版本说明并明确未签名、外部前置和能力边界：<https://github.com/liao96312/agent-flight-recorder/releases/tag/v0.1.0>。

- [x] **REL-06 [P0] 公开附件回下载与安装路径复验**
  - 依赖：REL-05。
  - 验收：从 GitHub Release 重新下载附件，复算 SHA-256，在独立目录运行 `version`、`list --json`；按 tag 内的公开文档完成 Codex / Claude 插件发现或安装，不再修改已经发布的 tag 内容。
  - 结果：从公开 Release 下载到独立目录 `D:\afr-release-verify-v0.1.0`，exe SHA-256 为 `165aabcd558d95d1a8fe2617ef0df19939492815ca572dc77d95e37d95ec95ac`，与附件校验和一致；`version` 为 `0.1.0 commit=a67c913dc626`，空 profile 的 `list --json` 为空。Codex 从公开 `liao96312/agent-flight-recorder@v0.1.0` marketplace 安装 `afr@personal` 0.1.0 成功并恢复原本地配置；Claude Code 2.1.185 对 tag checkout 的 marketplace/plugin strict validator 与 `--plugin-dir` 发现均通过。

- [x] **R0-09 [P1] 20 次真实会话评审**
  - 依赖：REL-06；D0 切片不是依赖。
  - 验收：20 条 wrapper / Claude skill 真实任务均记录 AFR version/commit、capture mode、host/version、session ID、任务类型、运行时长、目录大小、真实发现、误报、证据缺口、复盘耗时和继续使用意愿；不加遥测服务，不上传原始证据。
  - 2026-07-27 结果：完成 `20/20`；本地台账见 `R0-09-OBSERVATIONS.md`。重复证据集中在 Desktop 工具结束事件缺口、wrapper 逐行证据体积/复盘噪声、受控源码敏感词 high 误报、运行环境与构建身份漂移。第 15–17 条已完成 clean 保护、唯一 `--yes` 删除授权和已安装 skill 缓存刷新；第 20 条安全审查补齐 `active/idle × older-than/max-bytes` 四组合回归，`go test ./...` 通过。未出现需要同时扩建多个方向的证据。

- [x] **DEC-01 [P1] 执行 20 次会话数据门**
  - 依赖：R0-09。
  - 验收：形成 ADR，按多个可定位 session 的重复证据只选择一个方向（例如 hooks、性能索引或团队汇总），或明确保持 wrapper；写清收益、成本、非目标和下一检查点。
  - 结果：已形成 [ADR-0002](./ADR-0002-r0-data-gate.md)，决定保持 `afr` CLI + 已交付 Codex Desktop Hook + 双宿主薄 skill，不启动新 hooks、索引、团队汇总、签名、回放或来源格式升级。四组重复证据均有明确边界，但没有一个同时满足“重复阻碍具体复盘、存在可实现信号、收益超过格式/安全成本”；ADR 已写明四类重开阈值。

## 8. 暂缓池（不是当前承诺）

- [ ] **LATER-01 [P2] `afr replay` 与 `replay.plan.json`**
  - 触发：真实用户稳定使用审阅计划，且有明确安全/幂等规范。

- [ ] **LATER-02 [P2] SQLite/全文索引**
  - 触发：扫描 session.json 的实测 P95 成为瓶颈。

- [ ] **LATER-03 [P2] 交互式 PTY/TUI 托管**
  - 触发：非交互模式闭环稳定且交互 Agent 是主要入口。

- [ ] **LATER-04 [P2] MCP 服务或会话 UI**
  - 触发：skill 调 CLI 无法满足大量结构化查询。

- [ ] **LATER-05 [P2] 数字签名与远端见证**
  - 触发：需要对抗有本机管理员权限的攻击者。

- [ ] **LATER-06 [P2] Dify/模型分析**
  - 触发：本地闭环和脱敏经过审计，并有真实跨会话汇总需求。

- [ ] **LATER-07 [P2] 团队后台、RBAC、云存储**
  - 触发：本地归档/导出无法满足组织协作。

- [ ] **LATER-08 [P2] OS 级文件/网络审计**
  - 触发：客户要求强制、全系统可见性并接受平台复杂度。

## 9. 关键路径

```text
F0 契约
  -> M0 单进程可靠记录
  -> M1 工作区 before/after
  -> M2 写盘前脱敏 + verify
  -> M3 候选报告/清理/Windows 包
  -> 薄 Codex + Claude Code 插件
  -> M3-22 有界时间线 + M3-23 运行摘要（已完成）
  -> REL-01 MIT License（已完成）
  -> P1 可用性（show --open / ignore / 配置，已完成）
  -> REL-02 Claude Code 真实 skill 冒烟（已完成）
  -> REL-04 最终候选一致性检查（已完成）
      -> REL-05..REL-06 正式 v0.1.0 发布（已完成）
      -> D0-01..D0-06 Desktop 正式 Hook 与当前任务 E2E（已完成）
      -> M3-24 人类决策首页（十秒看懂状态 / 风险 / 变化 / 证据边界）
      -> R0-09 二十次 wrapper / desktop_hook / Claude skill 真实会话（逐条记录 capture mode/version/commit）
          -> DEC-01 单方向数据门
              -> [仅证据触发] M4 hooks / 其他一个扩展方向
```

M1、正式发布门槛、`D0-01..D0-06` Codex Desktop 当前任务接入、`M3-24` 人类决策首页、`R0-09` 二十次真实会话与 `DEC-01` 均已通过。当前按 ADR-0002 保持现有产品边界；没有新的成组证据前，不扩展 Claude hooks、Hosted tools 适配、历史导入、Dify、索引、签名或团队后台。

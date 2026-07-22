# Agent Flight Recorder TODO

> 与 [PROJECT_PLAN.md](./PROJECT_PLAN.md) 同步维护。任务只有在“验收”可重复通过后才能勾选。
> 优先级：P0 发布阻塞；P1 正式可用增强；P2 有真实需求后再做。
> 估算假设：1 名全职开发者，Windows x64 为 v0.1 发布平台。
> 当前基线：v0.1 核心候选、HTML 有界事件时间线、六行运行摘要、Unix CI 证据、MIT License、Claude Code 真实 skill 冒烟和三项 P1 可用性增强已完成；正式 tag / Release 与已暂停的干净 VM 门槛尚未完成。

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
| 7 | `R0-09` | 当前下一项 | 完成 20 条本地真实会话评审；每条记录 version/commit，允许区分期间发布的补丁版 |
| 8 | `DEC-01` | `R0-09` | 只选择一个有重复证据的方向，或明确保持现状 |

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

## 6. M4：原生 hook 适配（按真实需求，v0.2）

触发条件：`R0-09` 完成且 `DEC-01` 以多个可定位 session 证明 wrapper 持续因缺少工具/权限语义而阻碍复核。未触发则整阶段跳过；不得以单次体验直接启动。

- [ ] **H0-01 [P2] 冻结 plugin-only 状态机**
  - 依赖：`DEC-01` 明确选择 hooks / 原生事件方向。
  - 验收：active/idle/completed/incomplete 与 wrapper 状态兼容；Codex 无 SessionEnd 时不伪造 completed。

- [ ] **H0-02 [P2] 实现 `afr hook --host`**
  - 依赖：H0-01、M2。
  - 验收：stdin 读取正式 hook JSON，stdout 为空，默认 fail-open；所有持久化先脱敏。

- [ ] **H0-03 [P2] 定义宿主事件映射**
  - 依赖：H0-02。
  - 验收：prompt、tool before/after/failure、permission、subagent、stop/end 均映射到统一 event 或明确忽略。

- [ ] **H0-04 [P2] 未知字段白名单化**
  - 依赖：H0-02。
  - 验收：未知对象不原样落盘，只记录 host 类型、大小、hash 和 omitted 原因。

- [ ] **H0-05 [P2] wrapper/plugin 会话关联**
  - 依赖：H0-02。
  - 验收：父进程设置并校验 `AFR_SESSION_ID`；hook 并入同一会话，不生成重复记录。

- [ ] **H0-06 [P2] 实现跨进程 session 锁**
  - 依赖：H0-02、M0-06。
  - 验收：O_EXCL 锁、短重试、PID/time owner、受控陈旧恢复；无模糊强删锁文件。

- [ ] **H0-07 [P2] 100 并发 hook 压测**
  - 依赖：H0-06。
  - 验收：无重复 seq、无坏 JSON、verify 通过；超时生成 evidence gap。

- [ ] **H0-08 [P2] Codex hooks 配置**
  - 依赖：H0-03。
  - 验收：使用默认 `hooks/hooks.json`，不依赖 manifest override；安装后经 `/hooks` 信任才宣称启用。

- [ ] **H0-09 [P2] Codex Stop/idle 策略**
  - 依赖：H0-01、H0-08。
  - 验收：turn Stop 刷新报告并标记 idle；下一 prompt 可继续；不当作 SessionEnd。

- [ ] **H0-10 [P2] Claude Code hooks 配置**
  - 依赖：H0-03。
  - 验收：使用官方 exec form/path placeholder；SessionEnd 只做快速状态收尾。

- [ ] **H0-11 [P2] Hook coverage 报告**
  - 依赖：H0-08、H0-10。
  - 验收：显示安装、启用、信任、实际事件、宿主不覆盖的工具路径；零事件不等于零风险。

- [ ] **H0-12 [P2] 双宿主 E2E**
  - 依赖：H0-08 至 H0-11。
  - 验收：Codex/Claude 各完成 prompt、工具、权限、subagent、stop/end 场景；规范事件一致，native metadata 可不同。

- [ ] **H0-13 [P2] 评估单写者进程**
  - 依赖：H0-07 的真实瓶颈。
  - 验收：只有锁文件吞吐/可靠性达不到目标才立项；否则不建设 daemon。

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

- [ ] **R0-09 [P1] 20 次真实会话评审**
  - 依赖：REL-06。
  - 验收：20 条均记录 AFR version/commit、host/version、session ID、任务类型、运行时长、目录大小、真实发现、误报、证据缺口、复盘耗时和继续使用意愿；不加遥测服务，不上传原始证据。

- [ ] **DEC-01 [P1] 执行 20 次会话数据门**
  - 依赖：R0-09。
  - 验收：形成 ADR，按多个可定位 session 的重复证据只选择一个方向（例如 hooks、性能索引或团队汇总），或明确保持 wrapper；写清收益、成本、非目标和下一检查点。

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
      -> [下一项] R0-09 二十次真实会话（逐条记录 version/commit）
          -> DEC-01 单方向数据门
              -> [仅证据触发] M4 hooks / 其他一个扩展方向
```

M1 与正式发布门槛已经通过。现在的停止门槛是 `DEC-01`：20 次会话没有重复证据前不启动 hooks、Dify、索引、签名或团队后台。

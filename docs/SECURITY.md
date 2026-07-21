# 隐私与威胁模型

## AFR 承诺什么

AFR v0.1 是本地包装器，不上传会话，也不运行后台服务。它记录自己直接看到的子进程 argv、stdout/stderr、退出状态和工作区 before/after，写盘前统一脱敏，并用事件哈希链与 `manifest.json` 的 SHA-256 检查删除、改写、重排和制品替换。

校验结果表示“这些本地文件相对于同一会话的记录彼此一致”。它不是数字签名、时间戳服务或远端见证。本机高权限攻击者可以同时改写证据并重算哈希，因此 AFR 不提供对抗本机管理员的不可否认性。

## 看不到什么

报告的 capability matrix 是覆盖范围的权威来源。v0.1 wrapper 中：

- `process` 与可见的工作区变化为 `observed`；超过预算的内容会标为 `truncated` 或只存元数据。
- `native_tool_events`、`os_file_monitor`、`network_monitor` 为 `not_observable`。
- 零风险不等于全面安全；子 Agent 内部未输出的工具调用、工作区外写入和真实网络流量可能不可见。

## 脱敏边界

已识别的 secret 会被会话级 HMAC 占位符替换；HMAC key 只存活于记录进程。二进制、未知编码、超长逻辑记录和超限 Diff 采用 fail-closed，只保存大小、类型或流指纹。规则是有限的，不能保证识别所有业务敏感数据。

argv、文件路径、正常输出、Diff、派生 Markdown/JSON/HTML 都可能包含未被规则识别的项目名、用户名、源码或业务内容。分享整个 `~/.afr/sessions/<session-id>` 前应按组织政策人工复核；不要把会话目录自动同步到公共网盘或提交进仓库。

## 本地存储与删除

会话默认位于 `~/.afr/sessions`。Windows 依赖用户私有目录继承 ACL；POSIX 请求目录 `0700`、文件 `0600`。共享账户、宽松 ACL、备份软件和云同步都会扩大可访问范围。

AFR 不自动清理。先运行 `afr clean --older-than 30d` 或 `afr clean --max-bytes <size>` 预览，再在明确确认后添加 `--yes`。也可在 AFR 停止运行时删除单个完整 session 目录；删除前确认无需审计留存，并考虑备份、回收站或同步服务中的副本。

## 报告安全

`report.html` 使用上下文转义、内容限制和禁止外部资源的 CSP，但仍应作为敏感本地文件处理。不要关闭浏览器安全策略来打开报告。发现疑似脱敏泄漏时，停止分享、保留最小复现，并在公开 issue 中只放已人工清理的样本。

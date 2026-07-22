# D0-01 Codex Desktop Hook 契约探针记录

日期：2026-07-22  
结论：CLI 契约与持久信任已通过；当前桌面任务不热加载，等待新建 Desktop 任务完成最终探针。D0-02..D0-06 继续暂停。

## 环境

| 项目 | 值 |
|---|---|
| Codex Desktop | Windows 包 `26.715.9079.0` |
| Codex CLI | `0.138.0` |
| 可用 Codex CLI | npm 安装的 `0.145.0` |
| AFR public release | `v0.1.0` |
| 本地探针插件 | `afr@personal`，安装时缓存版本 `0.1.0+codex.20260722031148` |
| Hook 来源 | 插件默认 `hooks/hooks.json`，未在 manifest 声明 override |

## 已通过

- `probe.ps1` 合成输入自测通过；输出只包含事件元数据、字段类型、长度和 SHA-256 会话指纹。
- 自测确认不落盘 prompt、原始 session/turn ID、`transcript_path` 或工具输入正文。
- 插件结构通过 `plugin-creator` validator，Hook JSON 可解析。
- `codex plugin add afr@personal` 成功，插件状态为 installed、enabled。
- npm Codex `0.145.0` 不带 `--dangerously-bypass-hook-trust` 完成一次 shell turn，五类 Hook 均触发并写入同一 session 指纹；证明 AFR Hook 已持久信任。

## 未通过

- PATH 优先命中了 OpenClaw 自带 Codex `0.138.0`；该版本未发现 AFR 默认 plugin hooks。显式使用 npm Codex `0.145.0` 后问题消失，无需下载新依赖。
- 默认模型 `gpt-5.6-sol` 要求更新版 CLI，首次冒烟在工具调用前被宿主拒绝；该次结果不计入事件覆盖。
- 用户已在 Codex `0.145.0` 的 `/hooks` 界面确认 AFR Hook 为信任状态。当前 Codex Desktop 任务随后提交的新消息仍没有生成探针文件，证明当前任务不热加载新 Hook；尚未创建新的 Desktop 任务。

## 恢复步骤

1. 新建一个 Codex Desktop 任务，提交一条不含敏感内容的提示，并执行一个本地 shell/tool。
2. 仅当 `SessionStart`、`UserPromptSubmit`、`PreToolUse`、`PostToolUse`、`Stop` 五类 AFR 探针文件齐全时勾选 D0-01。

未满足恢复条件前，不改用私有日志、UI 轮询、transcript 解析或 manifest override 绕过停止门。

# D0-01 Codex Desktop Hook 契约探针记录

日期：2026-07-22  
结论：CLI 契约与持久信任已通过；Desktop 完整重启后触发四类 Hook，但连续两个 turn 转换均未观察到 `Stop`。D0-01 停止门未通过，D0-02..D0-06 不实施。

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
- Desktop 完整重启后，`gpt-5.6-sol` 当前任务实际触发 `SessionStart(source=resume)`、`UserPromptSubmit`、`PreToolUse` 和 `PostToolUse`。
- 随后的 `SessionStart(source=compact)` 沿用相同 session 指纹，证明 Desktop 的 resume/compact 事件可以稳定关联同一宿主 session。

## 未通过

- PATH 优先命中了 OpenClaw 自带 Codex `0.138.0`；该版本未发现 AFR 默认 plugin hooks。显式使用 npm Codex `0.145.0` 后问题消失，无需下载新依赖。
- 默认模型 `gpt-5.6-sol` 要求更新版 CLI，首次冒烟在工具调用前被宿主拒绝；该次结果不计入事件覆盖。
- 用户已在 Codex `0.145.0` 的 `/hooks` 界面确认 AFR Hook 为信任状态。当前 Codex Desktop 任务随后提交的新消息仍没有生成探针文件，证明当前任务不热加载新 Hook；尚未创建新的 Desktop 任务。
- 用户随后在 Desktop 新建任务并完成一次本地工具调用，但探针文件仍停留在 CLI 会话产生的 10 个，最新时间为 11:34:22。Desktop 主进程启动于 08:59，早于 AFR Hook 安装；同一应用进程内的新任务不会重新加载该插件 Hook。
- Desktop 重启后的两个完整 turn 转换均未生成 `Stop` 探针；其他四类事件持续生成，因此不归因于插件未加载或依赖缺失。

## 停止决定

1. 将缺失 `Stop` 记录为当前 Codex Desktop 包的宿主能力缺口。
2. 不实施依赖完整五事件契约的 D0-02..D0-06。
3. 项目继续使用已发布的 CLI wrapper 和 Claude Code skill 进入真实会话验证；不把 Desktop 四事件探针包装成不完整产品。

除非后续 Desktop 版本明确可重复触发 `Stop`，否则不重开 D0；不改用私有日志、UI 轮询、transcript 解析或 manifest override 绕过停止门。

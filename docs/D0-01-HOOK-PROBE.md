# D0-01 Codex Desktop Hook 契约探针记录

日期：2026-07-22  
结论：停止门未通过，暂停 D0-02..D0-06。

## 环境

| 项目 | 值 |
|---|---|
| Codex Desktop | Windows 包 `26.715.9079.0` |
| Codex CLI | `0.138.0` |
| AFR public release | `v0.1.0` |
| 本地探针插件 | `afr@personal`，安装时缓存版本 `0.1.0+codex.20260722031148` |
| Hook 来源 | 插件默认 `hooks/hooks.json`，未在 manifest 声明 override |

## 已通过

- `probe.ps1` 合成输入自测通过；输出只包含事件元数据、字段类型、长度和 SHA-256 会话指纹。
- 自测确认不落盘 prompt、原始 session/turn ID、`transcript_path` 或工具输入正文。
- 插件结构通过 `plugin-creator` validator，Hook JSON 可解析。
- `codex plugin add afr@personal` 成功，插件状态为 installed、enabled。

## 未通过

- 使用兼容模型 `gpt-5.5` 的 CLI 只读冒烟完成了一次本地 shell 调用，但没有生成 AFR 探针文件，也没有观察到可归属于 AFR 的 `PreToolUse`、`PostToolUse` 或 `Stop`。
- CLI 输出中的 `SessionStart` 与 `UserPromptSubmit` 不能作为 AFR 证据：本机已安装的 Ponytail 也注册了这两类 Hook，且 AFR 没有对应落盘文件。
- 默认模型 `gpt-5.6-sol` 要求更新版 CLI，首次冒烟在工具调用前被宿主拒绝；该次结果不计入事件覆盖。
- 当前 Codex Desktop 任务没有热加载新插件 Hook。Windows UI 自动化能读取当前任务，但截图接口返回 `0x80004002`，可访问性树未暴露可编辑 composer，因此未代替用户执行 `/hooks`，也未创建新任务。

## 恢复步骤

1. 在 Codex Desktop 手动执行 `/hooks`，确认 AFR 的五类 Hook 可见并信任当前定义。
2. 新建一个 Codex Desktop 任务，提交一条不含敏感内容的提示，并执行一个本地 shell/tool。
3. 仅当 `SessionStart`、`UserPromptSubmit`、`PreToolUse`、`PostToolUse`、`Stop` 五类 AFR 探针文件齐全时勾选 D0-01。

未满足恢复条件前，不改用私有日志、UI 轮询、transcript 解析或 manifest override 绕过停止门。

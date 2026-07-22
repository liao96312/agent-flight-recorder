# R0-09 真实会话评审台账

进度：**2/20**

仅记录 20 条 wrapper / Claude skill 真实任务的本地评审结果。本台账不上传原始证据；本次建表不计为 observation。

| # | AFR version/commit | capture mode | host/version | session ID | 任务类型 | 运行时长 | 目录大小 | 真实发现 | 误报 | 证据缺口 | 复盘耗时 | 继续使用意愿 |
|---:|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 0.1.0 / fa2ba5780caa | wrapper | Codex CLI 0.145.0 | 20260722T040209.889783700Z-8f93d616123f71d200a66530 | PR #13 只读审查 | 124 秒（外层超时） | 825,188 B | Windows 外层超时后 AFR/Codex 子进程组残留；已按精确 PID 停止 | 无 | session incomplete、缺 manifest，PR 审查无最终结论；native tool/network/OS file 不可观测 | 约 2 分钟 | 有条件：需避免用短外层超时运行长任务，并验证父进程终止时的子进程清理 |
| 2 | 0.1.0 / fa2ba5780caa | wrapper | Codex CLI 0.145.0 | 20260722T044941.694847000Z-112e3e8b376f1ac86377e9a8 | README 快速开始与本机 CLI 一致性检查 | 56 秒 | 261,179 B | PATH 指向旧构建 fa2ba5780caa，缺少已由发布构建 a67c913dc626 交付的 `show --open`；公开文档本身正确 | 初始现象像文档错误，复核发布产物后排除 | 仅检查快速开始和本机 CLI；native tool/network/OS file 不可观测 | 约 2 分钟 | 是；后续显式使用发布产物，避免同 semver 构建漂移 |

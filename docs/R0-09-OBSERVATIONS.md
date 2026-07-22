# R0-09 真实会话评审台账

进度：**1/20**

仅记录 20 条 wrapper / Claude skill 真实任务的本地评审结果。本台账不上传原始证据；本次建表不计为 observation。

| # | AFR version/commit | capture mode | host/version | session ID | 任务类型 | 运行时长 | 目录大小 | 真实发现 | 误报 | 证据缺口 | 复盘耗时 | 继续使用意愿 |
|---:|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 0.1.0 / fa2ba5780caa | wrapper | Codex CLI 0.145.0 | 20260722T040209.889783700Z-8f93d616123f71d200a66530 | PR #13 只读审查 | 124 秒（外层超时） | 825,188 B | Windows 外层超时后 AFR/Codex 子进程组残留；已按精确 PID 停止 | 无 | session incomplete、缺 manifest，PR 审查无最终结论；native tool/network/OS file 不可观测 | 约 2 分钟 | 有条件：需避免用短外层超时运行长任务，并验证父进程终止时的子进程清理 |

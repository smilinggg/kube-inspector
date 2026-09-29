# v0.1 验证记录

验证日期：2026-09-23。

验证环境：Linux，Go 1.27.1。项目只使用标准库；Go 工具链来自官方发布列表，并核对了发布列表中的 SHA-256。

| 项目 | 结果 |
| --- | --- |
| gofmt | 所有 Go 文件已格式化 |
| `go test ./...` | CLI 和规则包测试通过 |
| `go vet ./...` | 通过 |
| `go build -o kube-inspector ./cmd/kube-inspector` | 在独立临时目录中正常构建 |
| 默认样例 JSON 报告 | 5 条发现，退出码 2 |
| `--restart-threshold 8` | 4 条发现，退出码 2 |
| `--namespace batch` | 输出空 JSON 数组，退出码 0 |
| 非法阈值 0 | 退出码 1 |
| `--help` | 正常输出帮助，退出码 0 |
| 表格输出 | 5 条发现及资源身份、规则编号、解释完整 |

单元测试覆盖 Running/Ready 不一致、条件缺失、Unknown Ready、完成的 Job、未纳入规则的 Failed、正常 ContainerCreating、镜像拉取失败、CrashLoopBackOff、重启阈值边界、init 容器、删除中对象、空列表、错误 JSON、错误 kind、缺失字段、文件不存在、输出失败等情况。

当前验证针对离线 PodList 输入。client-go、真实集群连接、Informer、Prometheus 与 Controller 属于后续学习任务，尚未实现或进行集成验证。当前规则不构成完整的生产巡检覆盖。

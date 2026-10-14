**kube-inspector · Week 03 学习任务**

目标：用 client-go 从实验集群读取 Pod，复用现有检查与报告，并正确处理 API 失败。建议投入 6～8 小时，分 5 次完成；初次配置实验集群的时间另计。

本次依据 main 分支提交 [513916a](https://github.com/smilinggg/kube-inspector/commit/513916a57191c1539f1df1f696698aad180ea56f)，核对日期为 2026-10-14。已阅读入口、模型、规则、测试、Week 02 作业和学习路线。本次是静态审查，当前环境没有 Go 工具链，未执行项目测试。本文提供需求、验收标准与最小提示，由你完成实现。

**先确认目前的基础。** 你已经把规则参数放进 Options，增加了创建时间、Pending 年龄检查和非零退出码检查，继续保留了普通容器与 init 容器的处理。这些内容足以开始学习 client-go。

| 当前代码 | 本周处理 |
| --- | --- |
| Options 在 model 包；校验写在入口 | 可以保留，不要求专门移动包或增加 Validate 方法 |
| Pending 时间检查已加入 | 保留；让评估时间可由测试传入，修正依赖执行日期的用例 |
| CONTAINER_TERMINATED 检查 exitCode != 0 | 已实现；仅覆盖当前过滤后留下的 Running/Pending Pod |
| Failed Pod 被入口条件跳过 | POD_FAILED 尚未实现，可作为后续独立规则 |
| map 汇总、额外目录、长学习记录 | 本周不要求补齐 |
| 缺失或无效创建时间时跳过时间规则 | 与上周约定不同；暂作为明确的覆盖范围限制记录，不要求本周实现复杂的部分结果报告 |

没有发现只能表示已执行规则没有命中。当前代码跳过 Failed Pod，也可能跳过无法计算年龄的 Pod，不能将退出码 0 解释为所有检查均已通过。

代码依据：[命令行入口](https://github.com/smilinggg/kube-inspector/blob/513916a57191c1539f1df1f696698aad180ea56f/cmd/kube-inspector/main.go)、[规则](https://github.com/smilinggg/kube-inspector/blob/513916a57191c1539f1df1f696698aad180ea56f/internal/check/pod.go)、[规则测试](https://github.com/smilinggg/kube-inspector/blob/513916a57191c1539f1df1f696698aad180ea56f/internal/check/pod_test.go)。

**第 1 次：修正参数取值，再做第一次 Pod List（约 90 分钟）。**

先处理一个最关键的问题：入口在 flags.Parse(args) 之前执行了下面的赋值。

~~~go
options.RestartThreshold = *flags.Int("restart-threshold", 5, "reported restartCount threshold, inclusive")
~~~

flags.Int 返回 *int，即指向参数值的指针。表达式前面的 * 是解引用，读取“此刻”的值；给 Options 的 int 字段赋值会复制这个数值。此时还没解析命令行，读到的是默认值 5。后续 Parse 修改参数变量，不会自动修改之前复制出来的 Options 字段。PendingThreshold 存在相同问题。

最小提示：先保留参数指针，成功 Parse 后再取值，组装 Options，随后校验。

验收：

- 现有样例中 api-crash 的 restartCount 为 7，worker-restarted 为 8。传入 --restart-threshold=8 后，前者不再产生 CONTAINER_RESTARTS，后者仍应产生。
- --pending-threshold 的改变能够影响 Pending 时间规则。
- 0、负数、0s、负时间分别被拒绝；abc 由 Duration 参数解析报错。
- 测试非法阈值时传入一个确实存在的输入文件，并断言 stderr 是参数错误。当前某些用例只检查退出码 1，遗漏 --file 后可能因默认文件不存在而通过，掩盖参数校验问题。

~~~bash
go run ./cmd/kube-inspector --file=testdata/pods.json --restart-threshold=8 --output=json
~~~

接着记录实际 go version、实验集群 Kubernetes 版本，再选择 client-go 版本。仓库 go.mod 目前写的是 go 1.23.0，这不是你机器已安装版本的证明。检查所选依赖的 Go 要求；k8s.io/client-go、k8s.io/api、k8s.io/apimachinery 统一固定到同一个 v0.N.P 版本，提交 go.mod/go.sum。与集群相同 minor 是便于学习的起点，跨版本兼容以官方说明为准。

阅读官方 out-of-cluster 示例，独立完成：从一个明确指定的 kubeconfig 创建客户端，列出 default 命名空间的 Pod 名称和 phase。用相同 kubeconfig、相同命名空间的 kubectl 输出对照；动态状态以观察时刻为准。

学习点：指针取值、值复制、Go Modules、clientcmd、Clientset、List。

最小提示：沿着“配置 → 客户端 → CoreV1 → Pods(namespace) → List”寻找需要的类型和方法。

**第 2 次：让集群数据进入现有规则（约 90 分钟）。**

增加一个小的集群采集函数。可放在 internal/collect/kubernetes.go；目录名不作为验收要求。

本周建议保留 model.Pod，用一个转换函数把 corev1.Pod 转成现有模型，继续调用 CheckPods。采集函数负责取得对象，转换函数负责字段映射，规则函数继续负责判断。

| 现有规则需要的信息 | 转换时应保留的内容 |
| --- | --- |
| 对象身份与时间 | namespace、name、creationTimestamp、deletionTimestamp |
| Pod 状态 | phase、conditions 的 type/status |
| 两类容器 | containerStatuses、initContainerStatuses |
| 容器证据 | name、restartCount、waiting.reason、terminated.exitCode/reason/message |

注意：Kubernetes 官方类型的时间字段不是普通字符串。区分零值和缺失值；不要把不存在的删除时间转换成一个非 nil 的字符串指针，否则现有过滤逻辑会把 Pod 当作正在删除而跳过。

验收：

- 一个固定 Pod 对象经转换后，身份、状态、时间和容器字段不丢失。
- 同一份固定对象分别经文件解析、官方类型转换进入规则，在相同 Options 与 now 下得到相同 findings。
- 普通容器与 init 容器各选一个有代表性的样例；一个 Pod 可以有多条发现。
- API 返回错误时向上返回 error，不能将它替换为“成功取得空列表”。

学习点：外部包类型、结构体转换、slice、指针与 nil、函数职责。

最小提示：按 CheckPods 实际读取的字段整理转换清单，先验证转换，再调规则。

**第 3 次：接入命令行双输入（约 60 分钟）。**

| 参数 | 本周约定 |
| --- | --- |
| --source | file 或 cluster，默认 file；拒绝其他值 |
| --file | 保留原有文件输入；仅在 file 模式读取 |
| --kubeconfig | cluster 模式必须明确提供文件路径；使用其中的 current-context |
| --namespace | 保留现有含义；cluster 模式传给 Pod List，空字符串表示查询所有命名空间 |
| --timeout | API 采集时间预算，默认 10s，必须大于 0 |
| --output | 保留 table/json |
| 两个规则阈值 | 两种输入模式共用已经修正的参数处理 |

本周把 --context 覆盖选项、默认 kubeconfig 查找与多文件合并列为选做。先用明确路径和 current-context 完成一个可解释的版本。

期望运行方式：

~~~bash
go run ./cmd/kube-inspector --source=file --file=testdata/pods.json --output=json
go run ./cmd/kube-inspector --source=cluster --kubeconfig=/path/to/lab.config --namespace=default --timeout=10s --output=json
~~~

验收：

- 不传 --source 时，原有文件模式仍能运行。
- cluster 模式不读取默认 PodList 文件。
- 同一套报告函数处理两种数据源。
- JSON 模式下 stdout 只包含报告，诊断信息进入 stderr。
- 首先支持一次 Pod List、检查、输出、退出；Watch、Node、Deployment 在后续周次开展。

学习点：入口编排、错误分支、复用已有函数。

**第 4 次：处理超时和 API 失败（约 90 分钟）。**

先区分两个同名概念：

| 名称 | 作用 |
| --- | --- |
| kubeconfig 中的 context | 选择集群、用户和默认命名空间的一组配置 |
| Go 的 context.Context | 传播取消信号、截止时间等请求信息 |

由入口建立带超时的 Go context，传给采集函数，最终传给 List。创建超时后安排 cancel 释放相关资源。采集函数遇到错误，使用包含动作与目标的信息包装错误，并保留原始错误供上层判断；可以复用你在 readPods 中已经用过的 %w。

本周 --timeout 的契约是 API 采集预算，不承诺限制整个进程及任意外部认证插件的运行时间。先完成单次 List；如果自行设置 Limit 开始分页，就必须处理全部 continue token，并共享一次采集的时间预算，不能把第一页当成全量结果。

| 场景 | 程序行为 |
| --- | --- |
| 成功取得对象，当前规则无发现 | 退出码 0，JSON 为 [] |
| 成功取得对象，有发现 | 退出码 2，输出发现 |
| 成功查询一个存在但没有 Pod 的命名空间 | 退出码 0，空结果 |
| kubeconfig 错误、认证失败、Forbidden、连接失败、超时 | 退出码 1，stderr 说明失败，stdout 不输出成功报告 |

做三组实验：

1. 正常访问实验命名空间，与 kubectl 对照。
2. 使用缺少该命名空间 Pod list 权限的身份，确认 Forbidden 进入失败路径。仅执行 List 时，学习版本所需的资源权限就是 pods 的 list；使用 --namespace="" 查询全命名空间还需相应的集群范围权限。
3. 用可控制延迟的测试 HTTP 服务让响应晚于截止时间，确认请求取消并及时返回错误。无效地址常常立即连接失败，不能仅凭这个实验证明超时生效。

学习点：context.Context、context.WithTimeout、defer、错误包装与逐层返回。暂不要求并发、Informer 或 WorkQueue。

最小提示：检查每一层有没有收到同一个 ctx；仅创建 ctx 而 List 仍使用另一个 Background，不能验证取消传播。

**第 5 次：补直接保护本周行为的测试，提交演示（约 90 分钟）。**

先修复现有时间测试：internal/check/pod_test.go 把 recentCreation 固定为 2026-09-29，但规则用 time.Since 读取执行时刻。该日期现在已不是“刚创建”，因此原来只想检查普通 Pending 或 init 镜像错误的用例会多出 Pending 超时发现。这是静态推断，需你运行测试确认。

让检查接受外部评估时间 now，入口一次扫描取一次时间，测试传入固定时间。不要用“把样例日期改成今天”或真实等待解决。

| 必须覆盖的行为 | 最小验收 |
| --- | --- |
| Pending 阈值边界 | 同一个固定 now，年龄 4m59s 不命中，5m 与 5m1s 命中 |
| 自定义 CLI 阈值 | 确认实际改变规则结果，非法值因参数校验失败 |
| 官方对象转换 | 身份、时间、waiting/terminated、init 容器字段保留 |
| 输入源选择 | file 默认兼容；cluster 失败不回退为成功空结果 |
| 错误与超时 | 可控错误、可控延迟分别进入执行失败路径 |
| 原有输出 | findings 和空数组报告、0/2/1 语义保持正确 |

顺便让 Pending 消息准确表达“当前是 Pending，年龄从创建时间计算”。现有 “phase is Pending for ...” 容易让读者以为已经测量了连续 Pending 时长，单次快照无法证明这一点。

规则单元测试用固定数据；超时可以使用 net/http/httptest 的测试服务。client-go fake 可帮助模拟 API 返回值，但不能证明真实认证、RBAC 或网络超时正确，不要求这周搭建完整 mock 框架。

提交内容：

- 你写的代码、必要测试、go.mod/go.sum。
- go test ./... 和 go vet ./... 的实际输出。
- 正常查询、Forbidden、受控超时的简短记录，附命令与结果。
- README 补上集群运行命令、必要权限、退出码和当前规则范围；不用写长文。
- 用自己的话解释一次“参数指针何时取值”和“两个 context 的区别”。

检查退出码时使用现有 run 测试入口或构建后的二进制；不要把 go run 工具自身的退出状态直接当作目标程序退出码。

**本周过关以行为为准。** 集群与文件输入能复用规则；参数确实生效；时间测试不随日期漂移；API 获取失败不会输出成功的空报告。可以保留现有 Options 位置和简单函数设计。POD_FAILED、新资源检查、统计汇总以及更完整的输入异常报告可独立安排。

今天先完成“修正参数取值＋第一次 Pod List”。把相关 diff、测试输出和卡住的位置发来，继续按“一次指出一个关键问题 → 解释相关 Go 语法 → 最小提示 → 你修改后审查”的方式推进。

参考材料按任务阅读：

- [Go flag：返回指针与 Parse](https://pkg.go.dev/flag)
- [client-go 集群外客户端示例](https://github.com/kubernetes/client-go/tree/master/examples/out-of-cluster-client-configuration)
- [client-go 版本与兼容说明](https://github.com/kubernetes/client-go#compatibility-matrix)
- [Go context：WithTimeout 与取消](https://pkg.go.dev/context)
- [Kubernetes RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
- [Go httptest](https://pkg.go.dev/net/http/httptest)


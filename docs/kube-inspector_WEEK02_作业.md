# kube-inspector · Week 2 作业

主题：规则配置、Pending 时间检查、稳定的边界测试。

建议投入：6～8 小时，分 5 次完成。接着你完成的 Week 1 版本开发。交付物是你自己写的代码、测试和学习记录；本作业只给需求、验收标准和小提示。

本周继续使用本地 PodList JSON，配置通过命令行传入。核心成果：增加可配置的 POD_PENDING_TOO_LONG 规则，保留第一周已有规则的行为。

**1. 先保存第一周的基线（约 20 分钟）**

- 运行现有测试，记录结果并提交一次 Git commit。
- 用自己的话解释：输入里的 phase=Failed 与输出里的 Rule=POD_FAILED 各表示什么。
- 找出 POD_FAILED 判断与原有过滤条件的先后顺序，说明哪些 Pod 会被跳过。
- 这一步以你本地已完成的代码为基准；后续检查代码时再审查实际实现。

**2. 作业 A：把规则参数集中起来（约 60 分钟）**

设计一个 Options 结构体，在 check 包中统一携带以下规则参数：

| 字段 | Go 类型 | 命令行参数 | CLI 默认值 | 有效范围 |
| --- | --- | --- | --- | --- |
| RestartThreshold | int | --restart-threshold | 5 | 大于 0 |
| PendingThreshold | time.Duration | --pending-threshold | 5m | 大于 0 |

实现一个配置校验方法 Options.Validate() error。自己选择接收者，说明这个方法是否需要修改 Options。

要求：

- 现有重启规则使用 Options 中的阈值，保留 restartCount 大于等于阈值的行为。
- 新参数支持 30s、5m、1h 这样的时间长度。
- 0s 和负时间应在配置校验中被拒绝；abc 应作为参数解析错误处理。
- 原有重启阈值 0 和负数仍应被拒绝。
- 默认值由命令行入口设置；显式传入非法值时返回错误，不能自动替换成默认值。
- 配置错误打印到 stderr，程序退出码为 1。
- 配置有效时，原有功能的报告行为保持一致。

Go 学习点：结构体、方法、接收者、导出名称、error、指针取值。

最小提示：在现有 FlagSet 上找与 Int 类似的 Duration 方法；观察它的返回类型。

**3. 作业 B：读入 Pod 的创建时间（约 45 分钟）**

在现有 Metadata 类型中增加 creationTimestamp 对应的字段。本周建议先保存成 string，再在时间检查中解析，用来练习 time.Parse 与错误处理。

下面是可用于 JSON 解析测试的完整最小输入：

~~~json
{
  "apiVersion": "v1",
  "kind": "PodList",
  "items": [
    {
      "metadata": {
        "namespace": "default",
        "name": "pending-demo",
        "creationTimestamp": "2026-09-29T00:54:00Z"
      },
      "status": {
        "phase": "Pending"
      }
    }
  ]
}
~~~

先只验证：解析后的 Go 字段确实有值。检查字段是否导出、JSON tag 是否对应正确。

Kubernetes 的 creationTimestamp 表示对象的创建时间。当前是 Pending、创建已满多久，是这次巡检采用的近似口径；它不等于对“连续处于 Pending 多久”的精确测量。单次快照没有完整的阶段变化历史。

Go 学习点：JSON tag、字符串、time.Time、time.Parse、time.RFC3339。

最小提示：从 JSON 到结构体的映射先单独验证，再写超时判断。

**4. 作业 C：实现 Pending 时间规则（约 90 分钟）**

本周统一使用以下规则契约：

| 项目 | 要求 |
| --- | --- |
| 规则名 | POD_PENDING_TOO_LONG |
| 级别 | WARN |
| 检查对象 | 当前 phase=Pending 且 deletionTimestamp 为空的 Pod |
| 时间口径 | 传入的评估时间减去 creationTimestamp |
| 命中条件 | 对象年龄大于等于 PendingThreshold |
| 结果数量 | 每个命中的 Pod 产生 1 条该规则发现 |
| 结果身份 | 保留 namespace、Pod 名称；Container 留空 |
| 结果证据 | Message 写出当前 phase、对象年龄、阈值，并说明年龄从创建时间计算 |

固定评估时间为 2026-09-29T01:00:00Z、阈值为 5m 时，上面的 pending-demo 年龄为 6 分钟，应命中这条规则。

设计要求：

- 检查函数接受外部传入的 now，类型为 time.Time。
- 命令行入口一次扫描只获取一次 time.Now()；测试传入固定时间。
- 规则函数只做判断与返回结果，保留输入数据，不读取文件、不打印报告。
- 一个 Pod 可以同时命中 Pending 时间规则与镜像拉取失败规则；每个规则各自保留证据。
- Running、Succeeded、Failed Pod 不执行这条 Pending 规则。Failed Pod 仍按你第一周实现的 POD_FAILED 规则处理。
- 正在删除的 Pod 跳过这条规则及其时间校验。

错误处理也是这次作业的一部分。对于需要执行 Pending 检查的 Pod：

| 时间数据 | 本周约定 |
| --- | --- |
| creationTimestamp 缺失或空字符串 | 返回检查错误 |
| 格式无效 | 返回解析错误 |
| 创建时间晚于 now | 返回时间数据错误 |
| 数据有效且年龄未达阈值 | 正常返回，不产生本规则发现 |

检查错误必须能传回命令行入口，包含 namespace/Pod 名称与失败原因，退出码为 1。本周采用“本次扫描失败”的处理方式：stderr 输出错误，stdout 不输出成功报告，不把“无法判断”转换成“没有问题”。无错误时沿用原有报告与退出码约定：没有发现为 0，有发现为 2。

这是本次练习规定的错误策略。真实集群中的时钟偏差容忍度与部分结果报告，留到相应功能扩展时再设计。

Go 学习点：time.Time 与 time.Duration 的区别、时间相减、比较运算、返回多个值、错误向上返回。

最小提示：把“是否需要检查”“时间是否有效”“是否达到阈值”分别想清楚，再安排判断顺序。

**5. 作业 D：用表驱动测试证明规则正确（约 90 分钟）**

统一测试基准：

- now：2026-09-29T01:00:00Z。
- PendingThreshold：5 分钟。
- 测试不依赖真实等待，也不依赖执行当天的日期。
- 将下面每一行做成独立用例；没有时间相关错误时，明确检查目标规则是否存在。

| 编号 | 场景 | POD_PENDING_TOO_LONG 的预期 |
| --- | --- | --- |
| 1 | Pending，创建 4 分 59 秒 | 不命中 |
| 2 | Pending，创建恰好 5 分钟 | 命中 1 条 |
| 3 | Pending，创建 5 分 1 秒 | 命中 1 条 |
| 4 | Running，创建 30 分钟 | 不命中；其他规则独立判断 |
| 5 | Succeeded，创建 30 分钟 | 不命中 |
| 6 | Failed，创建 30 分钟 | 不命中；保留 POD_FAILED 的预期 |
| 7 | Pending，创建 10 分钟，正在删除 | 跳过 |
| 8 | Pending，创建时间缺失或为空 | 返回错误，不能视为通过 |
| 9 | Pending，创建时间为 not-a-time | 返回错误 |
| 10 | Pending，创建时间晚于 now | 返回错误 |
| 11 | 将边界创建时间写成 00:55:00Z 或 08:55:00+08:00，同一天 | 两种写法结果相同，都命中 |

再补充这些组合或回归检查：

- Pending 超时且 ImagePullBackOff：应同时有两条相应规则发现。
- 非 Pending Pod 没有 creationTimestamp：不能因此让本条时间规则报错。
- 空 Pod 列表与合法配置：仍返回空发现列表，JSON 报告保持 []。
- 作业 A 的合法与非法配置都有对应测试。
- 接入新增的错误返回后，命令行入口测试覆盖退出码 1、2、0 的各自含义。

注意两处旧测试适配：

1. 调整函数签名后，要同步更新调用处和测试。
2. 原来的 Pending 用例没有 creationTimestamp，需要补成固定 now 附近的有效时间，防止引入与原用例无关的时间错误或超时发现。保留原测试想验证的行为。

Go 学习点：匿名结构体切片、range、t.Run、error 断言、边界值、稳定输入。

最小提示：多条规则可以同时命中；验证特定规则时，不要把“没有这条规则”误写成“整个 findings 必须为空”。

**6. 作业 E：接好命令行并整理说明（约 60 分钟）**

预期支持的运行方式：

~~~bash
go run ./cmd/kube-inspector --file testdata/pending.json --restart-threshold 5 --pending-threshold 5m --output json
~~~

其中 pending.json 是你自己保存的 PodList 测试文件。命令行使用实际评估时间，因此固定样例的年龄会随运行日期变化；精确的 4 分 59 秒和 5 分钟边界由固定 now 的单元测试验收。

继续保持这三个包的职责：

| 位置 | 职责 |
| --- | --- |
| model 包 | JSON 数据字段 |
| check 包 | Options、校验与规则判断 |
| main 包 | 参数、文件输入、调用检查、报告与退出码 |

不要求扩张目录数量。你需要能解释：为什么配置校验放在这里、为什么 JSON 类型由规则包引用、为什么导入关系不能形成循环。

README 补充四点：

- --pending-threshold 的默认值、格式和有效范围。
- 新规则的时间口径与大于等于边界。
- 时间缺失或错误时，工具怎样返回失败。
- 离线 JSON 表示快照状态；旧文件不能证明集群当前仍然 Pending。

运行一次代码格式化、go test ./... 和 go vet ./...，检查输出并修复实际问题。退出码验收应调用现有 run 测试入口或编译后的可执行文件；go run 本身的退出状态不要直接当成目标程序退出码。

**7. 选做：用 map 汇总规则命中次数（约 30 分钟）**

完成必做项后，写一个独立的小函数，将 []Finding 汇总成 map[string]int。

举例：POD_FAILED 出现 2 次，POD_PENDING_TOO_LONG 出现 3 次，对应两个规则计数为 2 和 3。

这个统计计数的是“发现记录”，同一个 Pod 命中两条规则会计入两次。给空输入、重复规则、不同规则各补一个用例即可。

学习点：map 的初始化、读写、零值。先通过小函数与测试展示结果。

**8. 过关标准与提交方式**

完成时交付：

- 修改后的相关 Go 代码或代码 diff。
- 新增和调整的测试，以及 go test ./...、go vet ./... 的结果。
- 一组 JSON 测试输入。
- README 的新增说明与一条学习记录。
- 至少一个清楚说明改动目的的 Git commit。

验收清单：

- [ ] 新阈值可通过命令行调整，非法值不会静默生效。
- [ ] 原有规则的回归测试通过。
- [ ] Pending 时间规则的正常、异常、精确边界和字段缺失均有测试。
- [ ] 同样的输入和固定 now，重复执行得到相同结果。
- [ ] 不能计算时间时，返回执行失败而非“零问题”。
- [ ] 你能说明方法接收者、*time.Duration、time.Time 与 time.Duration 的作用。
- [ ] 你能说明 creationTimestamp 的使用口径和单次快照的局限。

继续采用现有辅导方式：先自己尝试；卡住时提供当前相关代码、失败测试和预期结果。我每次先指出一个关键问题，解释相关 Go 语法，再给一个最小提示；你改完后再做代码审查。

今天只做作业 A。完成后先提交 Options、Validate 方法和对应测试，避免多个新问题一起出现。

**只查当前需要的官方文档**

- Go flag，重点看 FlagSet.Duration：https://pkg.go.dev/flag#FlagSet.Duration
- Go time，重点看 Parse、RFC3339、Time.Sub：https://pkg.go.dev/time
- Kubernetes ObjectMeta，重点看 creationTimestamp：https://kubernetes.io/docs/reference/kubernetes-api/definitions/object-meta-v1-meta/
- Kubernetes Pod 生命周期：https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/

制定依据：现有《Go_CNCF_学习实践路线》的 Week 2 目标——增加规则配置、整理包、补充 Pending 时间规则，以及覆盖正常、异常、阈值边界和字段缺失。


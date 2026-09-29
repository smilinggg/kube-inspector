# Go × Kubernetes 学习实践路线

适用对象：有 Kubernetes 运维经验，断续接触过 Go，希望读懂云原生项目源码并逐步参与开发的工程师。

主线项目：**kube-inspector——Kubernetes 巡检工具**。

建议节奏：每周约 6～8 小时；前 8 周完成核心项目，随后用 2～4 周练习 Controller。按验收结果推进，周数可以顺延。

## 1. 这次要形成的能力

你已经接触过 Go 指针、接口、HTTP、并发，也看过 TinyKV/TinySQL 和 Operator 相关内容。这次把这些知识固定在一个持续演进的项目里，每周都使用旧知识完成新功能。

三个目标：

1. 能把熟悉的运维检查写成 Go 程序，有输入、判断、报告、错误处理和测试。
2. 能解释自己的程序怎样访问 APIServer，怎样通过 List/Watch、缓存、队列持续工作。
3. 能在一个上游项目中定位一个具体行为，复现问题、补测试并准备小范围 PR。

第一阶段的衡量标准是“独立修改一个功能并验证”。允许查 API 文档。看过教程、运行过别人写的代码，只作为起点记录。

## 2. 为什么选择巡检工具

你熟悉 Pod 状态、Node、Deployment、故障判断和巡检报告，能够先明确“程序应该判断什么”，再集中练习“用 Go 如何表达”。项目还能用于整理自己工作中的规则。

本项目围绕 Kubernetes 和 Prometheus 两个 CNCF 生态项目展开；它是自己的学习项目，并不表示它已经加入 CNCF。client-go、sample-controller、Kubebuilder 是相关开发组件与学习材料，也不在这里被当作独立 CNCF 项目列举。

| 项目阶段 | 做出来的东西 | 能理解的云原生机制 |
| --- | --- | --- |
| 本地 CLI | 读取 PodList 快照并检查 | API JSON 与 Go 类型的对应关系 |
| 集群 CLI | 直接查询 Pod/Node/Deployment | kubeconfig、认证、RBAC、REST API、分页和错误 |
| 常驻模式 | 缓存对象并响应变化 | List/Watch、Informer、Lister、WorkQueue |
| 指标模式 | /metrics 暴露巡检结果和工具自身状态 | Prometheus 客户端、指标类型、标签基数 |
| Controller 扩展 | 用 Inspection CR 声明巡检配置并保存摘要状态 | CRD、Reconcile、Status、幂等、失败重试 |

前八周交付的范围是单集群巡检、命令行/JSON 报告、常驻模式和少量指标。多租户平台、多集群管理、Web 界面、数据库、AI 自动诊断、自愈操作留到后续独立规划。

## 3. 项目功能与数据边界

### 3.1 起步版已提供

下载的 v0.1 只依赖标准库，读取 `kubectl get pods -A -o json` 对应的 PodList 文件。

| 检查项 | 当前实现 | 需要讲清的判断 |
| --- | --- | --- |
| Running 但未就绪 | 查看 Ready condition | Running 不保证 Ready；条件缺失也不能当成 True |
| 容器等待异常 | CrashLoopBackOff、ImagePullBackOff、ErrImagePull | 等待原因来自容器状态，不是 Pod phase |
| 重启累计值较高 | restartCount ≥ 可配置阈值 | 单次快照不能说明“最近五分钟重启几次” |

普通容器与 init 容器均纳入后两条规则。当前仅检查未删除的 Running/Pending Pod；Succeeded、Failed、Unknown 和正在删除的 Pod 不在这三条规则范围内。第一周会练习添加 Failed 检查。

### 3.2 八周内逐步加入

- Pod：Failed、长时间 Pending；先写清持续时间的近似口径。
- Node：Ready 非 True、MemoryPressure、DiskPressure；区分压力状态与用量数据。
- Deployment：期望与可用副本、observedGeneration、滚动更新宽限时间；避免部署尚在正常推进时立刻判失败。
- 输出：表格与结构化 JSON；每条发现保留对象身份、规则编号、级别、证据。
- 工具自身：API 错误、超时、最近一次成功扫描时间、扫描耗时；获取失败应显示 UNKNOWN/执行失败，不能显示“没有风险”。

CPU/内存实际使用率需要 metrics.k8s.io 或 Prometheus 等额外数据源。Node/Pod 的普通资源对象主要给出状态、配置和容量信息；请求值、限制值、容量和实际用量要分清。

### 3.3 先约定退出码

| 退出码 | 含义 |
| --- | --- |
| 0 | 本次执行成功，当前选定规则没有发现问题 |
| 2 | 本次执行成功，当前选定规则发现问题 |
| 1 | 执行失败或结果不完整，例如输入错误、认证失败、API 超时 |

后续支持部分结果时，报告中必须包含 `partial=true` 和失败原因，并仍返回执行失败。退出码 0 不能理解为“集群各方面都健康”。

## 4. 八周核心路线

每周固定交付：一个能演示的功能、一组对应测试、一条学习记录和一个 Git commit。

| 周次 | 本周功能 | Go 重点 | Kubernetes / 云原生重点 | 过关标准 |
| --- | --- | --- | --- | --- |
| 第 1 周 | 运行样例；加 Failed 检查；改阈值 | struct、slice、for、函数、指针、JSON tag、error | phase、conditions、containerStatuses | 不看参考实现，45 分钟内增加一个小规则并补测试 |
| 第 2 周 | 增加规则配置；整理包；补 Pending 时间规则 | 方法、接收者、包、map、time、表驱动测试 | 状态证据、阈值、边界和误报 | 测试覆盖正常、异常、阈值边界、字段缺失；能解释每个导出符号 |
| 第 3 周 | 用 client-go 取代文件输入；支持 kubeconfig/context/namespace | Go Modules、context、错误包装 | Clientset、RBAC、List、超时、Forbidden | 实验集群报告可与 kubectl 对照；错误凭据和超时不会报告为健康 |
| 第 4 周 | 加 Node、Deployment 检查；统一结果模型 | 小接口、组合、依赖注入、fake、单元测试 | Conditions、Generation、滚动更新、容量与用量 | 新增规则不需要修改所有已有规则；每条规则可独立测试 |
| 第 5 周 | 分资源类型并发采集；限制并发；可取消 | goroutine、channel、WaitGroup/errgroup、互斥、context | API 请求限速、分页、部分失败 | 并发有上限；Ctrl+C 能退出；模拟一类查询失败时保留证据并报不完整；race 检查通过 |
| 第 6 周 | 实现 watch 常驻模式 | 回调、队列、后台 worker、共享状态 | Informer、缓存同步、Lister、WorkQueue、重试 | 更新/删除 Pod 后结果同步变化；事件回调只入队，worker 处理；断线重连后恢复一致 |
| 第 7 周 | 暴露 /metrics 与进程健康接口 | net/http、客户端库、锁或快照、日志 | Prometheus 指标、Gauge/Counter、标签基数 | Prometheus 能抓取；扫描失败和结果过期可观察；指标不使用无界错误文本作为标签 |
| 第 8 周 | 容器化，在 kind 部署；整理 README 和发布标签 | build、配置、信号退出、工程协作 | ServiceAccount、只读 RBAC、Deployment、Service | 从新目录按 README 可以构建部署；演示正常/故障/恢复；形成 v0.1.0 核心版 |

一周没有通过验收就顺延。若每周只能投入三小时，按两周完成一行安排。

## 5. 每一周的具体实现提示

### 第 1～2 周：建立可以反复使用的最小程序

从随包的 `docs/WEEK01.md` 开始。只补当前功能所需的 Go 知识；读不懂 `*int` 时就围绕 flag 参数做小实验，读不懂 slice 时就围绕 findings 的 append 做实验。

先用直接函数调用完成一条规则。当第二、第三条规则确实需要一致的执行方式，再提取小接口。例如第四周可以设计：

```go
type Check interface {
    Name() string
    Run(ctx context.Context, snapshot Snapshot) ([]Finding, error)
}
```

这里的 Snapshot、Finding 都需要由自己定义。让 PodCheck 和 NodeCheck 各实现一次，再解释：为什么不需要 `implements`，值接收者和指针接收者会影响哪些赋值。采集与规则分开，可以让规则输入成为固定数据，避免每条规则重复请求 APIServer。

### 第 3～4 周：真正接触 Kubernetes 编程

先阅读官方 client-go 的 out-of-cluster 示例，完成一个命名空间的 Pod List。熟悉之后再加节点和工作负载。

三个依赖 `k8s.io/client-go`、`k8s.io/api`、`k8s.io/apimachinery` 固定到同一发布系列及合适补丁版本，提交 go.mod/go.sum。`client-go v0.N.P` 对应 Kubernetes `v1.N.P`；相同 minor 是便于学习的选择，实际跨版本兼容需查看官方矩阵和所用 API。不要把“必须与集群完全同版”当作通用规则。

Go 版本需满足所选依赖的 go.mod 要求。先建立一个干净实验集群并记录版本，后续阅读源码也固定 tag/commit，减少漂移。

本阶段做三个故障实验：

1. 使用没有 Pod List 权限的账号，确认报告是执行失败，能解释 RBAC 错误。
2. 配置无效 APIServer 地址和短超时，确认程序按时退出。
3. 同一个 Pod 同时 Running、Ready=False，确认规则仍报告未就绪。

实际试验在 kind 或专用实验集群进行。首轮 API 权限只给目标资源的 get/list，常驻 Watch 阶段再增加 watch。

### 第 5 周：让并发解决一个真实等待问题

把 Pod、Node、Deployment 采集分成少量并发任务；不要为了练 goroutine 给每个 Pod 都发一次独立 API 请求。以批量 List、必要分页、客户端 QPS/Burst 和有限并发控制开销。

必须明确：谁创建 context、谁调用 cancel、谁关闭 channel、谁等待 worker 退出。通过测试服务或可替换客户端注入“一个请求一直阻塞”的场景，验证取消能传播。使用 `go test -race ./...` 检查当前测试实际执行到的并发路径；通过 race 不代表未执行路径绝无竞争。

### 第 6 周：理解 Controller 的核心运行方式

先把 Pod 对象接入 shared informer，等待缓存同步，再通过 lister 查询；事件回调提取 namespace/name 放到队列，worker 读取最新缓存状态重新计算。

学习时要能回答：

- Watch 中断后谁负责恢复？为什么启动时要等待缓存同步？
- 缓存为什么可能滞后？为什么不应直接修改 lister 返回的共享对象？
- 同一个 key 重复进入队列时，为什么业务逻辑需要幂等？
- 对象已经删除、Lister 返回 NotFound 时，如何移除旧的巡检结果？
- “Pending 超过五分钟”即使没有新事件也会变化，如何通过周期任务或延迟入队触发重算？

将 List + Watch 作为概念起点，实际初始化细节以固定版本的 client-go 为准。阅读时核对自己使用的版本，记录实测调用链。

### 第 7～8 周：变成可部署、可观察的小工具

建议先做四类指标：

```text
kube_inspector_findings{rule="pod_ready",severity="warning"} 3
kube_inspector_scan_success 1
kube_inspector_last_success_timestamp_seconds 1790157600
kube_inspector_scan_duration_seconds 0.18
```

这些是指标设计示意，不是起步版已提供的接口。其中当前发现数是 Gauge，累计扫描次数才适合 Counter；最后成功时间是 Unix 秒值。错误全文、动态报告文本放日志或 JSON，避免成为无界标签值。

扫描失败时保留“上一次成功结果”的含义并展示过期时间，不能把失败写成 0 个问题。HTTP 指标处理从内存快照读取；采集在后台执行，使 scrape 不与一次长时间 API 扫描绑在一起。资源删除后清理相应缓存和旧指标。

部署验证至少包含：新启动时缓存尚未同步、扫描出错、Pod 恢复、Pod 删除、程序收到终止信号。用截图、日志或小视频保留正常→故障→恢复的证据。

## 6. Controller 扩展：再投入 2～4 周

前八周项目能独立运行之后，用 Kubebuilder 搭建一个 namespaced `Inspection` API。一个 Inspection 对象管理本命名空间的巡检配置，先减少跨命名空间映射的复杂度。

```yaml
apiVersion: ops.example.com/v1alpha1
kind: Inspection
metadata:
  name: application-check
  namespace: learning
spec:
  intervalSeconds: 60
  restartThreshold: 5
status:
  observedGeneration: 1
  findingCount: 2
  lastSuccessfulTime: "2026-09-23T10:00:00Z"
```

这是预期 API 形状，不能直接用于当前 v0.1 工程。实际 CR YAML 由用户填写 spec；status 由 Controller 更新。

先完成“读取 CR → 采集/检查 → 更新 status → RequeueAfter 定期再运行”。再练习 Pod 变化映射到同命名空间 Inspection 的请求。更新 status 时避免无意义的重复写入和事件循环。

验收四个实验：修改阈值后结果变化；Controller 重启后继续执行；重复 Reconcile 不产生重复副作用；API 写冲突和临时错误能重试。更新 status 需要相应 CR/status 写权限，业务 Pod/Node 等仍按读操作设计。

这个阶段再集中理解 API 类型、Scheme、DeepCopy、CRD schema、Reconciler、Status、Generation，以及 controller-gen 生成文件的作用。关联你此前对 KubeKey CR 设计的疑问：当巡检配置需要通过 Kubernetes API 声明、校验、监听并记录状态时，CRD 的价值才具体可见。生成工具能减少重复代码，业务检查与调谐逻辑仍需自己实现。

## 7. 用同一个项目解决“老是忘记”

### 每次 60 分钟的结构

| 时间 | 操作 |
| --- | --- |
| 前 10 分钟 | 不看上次实现，解释或重写一个旧函数 |
| 接着 35 分钟 | 完成本次最小改动，卡住只查当前问题 |
| 接着 10 分钟 | 写测试并运行，解释结果为何符合预期 |
| 最后 5 分钟 | 提交代码，写下一次的第一条命令和一个小任务 |

采用 1、3、7、14 天的复用提醒作为学习安排，不承诺“这样就永远记住”。每次换一种输入或增加一个边界条件，让概念在项目里再次出现。

| 概念 | 第一次使用 | 后续复用 |
| --- | --- | --- |
| 指针与 nil | flag 参数、可选 Waiting 字段 | 指针接收者、可选配置、API 对象更新 |
| slice/map | Pod 列表、发现列表 | namespace 汇总、缓存索引、测试用例 |
| 接口 | io.Writer 的输出替换 | Check 接口、采集客户端替换、fake |
| error | 文件读取和 JSON 解析 | API 错误、部分结果、队列重试 |
| context | API 超时 | 并发取消、worker 退出、Controller |
| 测试 | 规则表驱动测试 | fake 客户端、并发错误、集成实验 |

如果中断一周：先读上次的“下一步”，跑测试，修一个小问题。只有某个知识点确实阻止当前任务时才回补对应内容，不需要从变量和 for 循环重新开始一整套课程。

AI 的使用约定：先独立尝试 20 分钟，再请求提示；先让 AI 解释报错和 review，再参考完整实现。每次合入 AI 建议前，自己讲出输入、输出、关键分支和失败路径。

## 8. 源码阅读与开源参与

把“读完整个项目”改为“回答一个具体问题”。一次阅读只跟一条功能链，记录仓库、固定 tag/commit、入口函数、关键类型、对应测试和结论。

| 顺序 | 官方材料 | 这次要回答的问题 |
| --- | --- | --- |
| 1 | client-go 的 out-of-cluster / in-cluster 示例 | kubeconfig 和 ServiceAccount 如何变成可调用的客户端？ |
| 2 | sample-controller 的 main.go、controller.go 及测试 | 事件如何入队？worker 如何取 key、查缓存、执行、重试？ |
| 3 | kube-state-metrics 的 Pod 指标实现与测试 | 一个 Pod API 字段如何变成指标？删除如何反映到结果？ |
| 4 | Kubebuilder Book 的 Controller 示例 | Reconcile 如何处理重复调用、错误和 Status 更新？ |

每读到一个机制就在自己的项目里做最小实验。后续回看 KubeKey 的 API、Controller、生成文件，会更容易区分框架结构和业务逻辑。

第八周以后选一个相关上游项目持续观察。kube-state-metrics 与本项目的问题域接近，可作为候选；是否适合第一个 PR，要看当时 issue、维护者意见和贡献指南。

首个贡献的行动顺序：阅读 CONTRIBUTING → 运行相关测试 → 复现一个小问题或明确文档缺口 → 先讨论预期行为 → 最小修改及测试 → 提交 PR → 响应 review。一个补边界测试、改错误说明或纠正文档示例的 PR 就是有效起点。能否合并由维护者决定。

注意实际贡献入口：`kubernetes/sample-controller` 和 `kubernetes/client-go` 是 Kubernetes staging 发布镜像；针对其中代码的贡献按照各自 CONTRIBUTING 指向 `kubernetes/kubernetes` 中的对应 staging 路径。不能默认直接向镜像仓库提 PR。

## 9. 完成检查表

- [ ] 我能运行起步版并解释 5 条发现。
- [ ] 我独立增加过至少两条规则和相应测试。
- [ ] 我能说明指针接收者、接口隐式实现、nil 和 slice 的实际用途。
- [ ] 我能用 client-go 访问实验集群，并区分权限、连接、超时等错误。
- [ ] 我能让并发任务有上限，并在取消后退出。
- [ ] 我能解释缓存同步、事件入队、worker 重算和删除清理。
- [ ] 我能让 Prometheus 抓取指标，识别采集失败与结果过期。
- [ ] 我能从空目录按 README 构建部署并演示故障恢复。
- [ ] 我能在一个上游项目中定位一个行为及其测试。
- [ ] 我准备过一个范围明确的上游改进提案或 PR。

## 10. 官方参考

核对日期：2026-09-23。版本、目录与贡献要求以实际使用的 tag/commit 及当时文档为准。

1. Go 安装与学习入口：https://go.dev/learn/ ；Go Tour：https://go.dev/tour/
2. client-go 与版本兼容：https://github.com/kubernetes/client-go
3. 集群外客户端示例：https://github.com/kubernetes/client-go/tree/master/examples/out-of-cluster-client-configuration
4. 集群内客户端示例：https://github.com/kubernetes/client-go/tree/master/examples/in-cluster-client-configuration
5. Kubernetes Pod 生命周期：https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/
6. kind Quick Start：https://kind.sigs.k8s.io/docs/user/quick-start/
7. sample-controller：https://github.com/kubernetes/sample-controller
8. kube-state-metrics：https://github.com/kubernetes/kube-state-metrics
9. kube-state-metrics 贡献说明：https://github.com/kubernetes/kube-state-metrics/blob/main/CONTRIBUTING.md
10. Prometheus Go 客户端：https://github.com/prometheus/client_golang
11. Prometheus 指标命名与标签：https://prometheus.io/docs/practices/naming/
12. Kubebuilder Controller：https://book.kubebuilder.io/cronjob-tutorial/controller-overview.html
13. Kubernetes 贡献入门：https://www.kubernetes.dev/docs/guide/

随包 `kube-inspector/README.md`、`docs/WEEK01.md` 和 `notes/learning-log-template.md` 提供运行步骤、首周任务和复习记录模板。

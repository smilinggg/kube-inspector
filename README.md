# kube-inspector：面向 K8s 运维工程师的 Go 学习项目

这是一个自主练习项目，围绕 Kubernetes 和 Prometheus 生态演进；不代表已经加入 CNCF。

当前交付 v0.1：读取 Kubernetes PodList JSON → 执行三类检查 → 输出表格/JSON。它仅依赖 Go 标准库。client-go、Informer、Prometheus 和 Controller 是接下来逐步实现的学习任务，具体见 [学习路线](docs/Go_CNCF_学习实践路线.md)。

## 运行

安装当前受支持的稳定版 Go：https://go.dev/doc/install 。`go.mod` 中 `go 1.22.0` 是本例的最低语言版本声明，不是建议安装旧版本。

进入解压后的 `kube-inspector` 目录：

```bash
go version
go test ./...
go run ./cmd/kube-inspector --file testdata/pods.json
go run ./cmd/kube-inspector --file testdata/pods.json --output json
```

样例共 5 个 Pod，默认阈值 5，产生 5 条发现：`api-crash` 3 条、`image-pull` 1 条、`worker-restarted` 1 条。正常 Pod 和完成的 Job 不触发这三个检查。

发现问题时程序返回 **2**，因此 `go run` 会额外打印 `exit status 2`；这是预期行为。需要准确获取程序退出码时，先编译再执行：

```bash
# Linux / WSL / macOS
go build -o kube-inspector ./cmd/kube-inspector
./kube-inspector --file testdata/pods.json --output json > report.json
echo $?
```

Windows PowerShell 使用：

```powershell
go build -o kube-inspector.exe ./cmd/kube-inspector
.\kube-inspector.exe --file testdata/pods.json --output json
$LASTEXITCODE
```

stdout 只包含报告，stderr 包含本次输入 Pod 数、筛选后 Pod 数、发现数及错误。退出码：0=完成且本版规则无发现，2=完成且有发现，1=执行失败。**0 不代表集群所有方面都健康。**

## 读取自己实验集群的快照

```bash
kubectl get pods -A -o json > pods-local.json
go run ./cmd/kube-inspector --file pods-local.json --namespace default
```

上述 shell 重定向按 Linux/WSL/macOS 编写。Windows PowerShell 5.1 的 `>` 可能生成 UTF-16；可用下面的方式写为无 BOM UTF-8：

```powershell
$podJson = kubectl get pods -A -o json | Out-String
[System.IO.File]::WriteAllText((Join-Path $PWD 'pods-local.json'), $podJson, [System.Text.UTF8Encoding]::new($false))
```

这个版本的 Go 程序读取文件；与集群通信的是上面的 kubectl。第三周再将数据来源换成 client-go。

## 已实现的规则与边界

| 规则 | 条件 | 本版含义 |
| --- | --- | --- |
| POD_NOT_READY | Running 且 Ready 条件不为 True | 当前快照未就绪；还没有持续时间去抖 |
| CONTAINER_WAITING | 普通或 init 容器等待原因是 CrashLoopBackOff / ImagePullBackOff / ErrImagePull | 明确等待异常；不是所有 Waiting 都异常 |
| CONTAINER_RESTARTS | 普通或 init 容器上报的 restartCount ≥ 阈值 | 累计值提醒，不能直接解释为“最近频繁重启” |

仅对 Running/Pending 且没有 deletionTimestamp 的 Pod 运行这三个检查。Succeeded、Failed、Unknown、正在删除的 Pod 均不属于当前规则覆盖范围；Failed 和长时间 Pending 是第一周的扩展作业。

在真实 API 对象里，CrashLoopBackOff 是容器 Waiting 的 reason，不是 Pod phase。累计重启计数可能发生重置；实现近期重启告警需要保存多次采样及时间、按 Pod UID 和容器身份关联，并处理重置。

## 从哪里读代码

| 文件 | 作用 | 对应 Go 知识 |
| --- | --- | --- |
| `cmd/kube-inspector/main.go` | 参数、读文件、调用规则、输出、退出码 | flag、指针、error、io.Writer、包引用 |
| `internal/model/pod.go` | JSON 对象到 Go 类型 | struct、tag、slice、指针与 nil |
| `internal/check/pod.go` | 三条规则 | 函数、for/range、switch、append |
| `internal/check/pod_test.go` | 规则边界测试 | testing、表驱动测试、子测试 |
| `cmd/kube-inspector/main_test.go` | 输入错误、筛选、JSON 和退出码测试 | bytes.Buffer、临时文件、错误路径 |
| `testdata/pods.json` | 可重复运行的输入 | K8s API 数据结构 |

先跟踪 `main → run → readPods → CheckPods → writeFindings`。第一天只需解释 `CheckPods` 里的规则分支。

## 第一周交付

照着 [第一周任务](docs/WEEK01.md) 完成：改阈值、加入 Failed 检查、补边界测试、脱离参考重写一个规则。使用 [学习记录模板](notes/learning-log-template.md) 保存下一次可继续执行的一条命令。

```bash
gofmt -w cmd internal
go test ./...
go vet ./...
```

样例工程以教学为目标。验证范围及结果见 [VALIDATION.md](VALIDATION.md)。

// Package check 实现纯规则函数：输入数据，返回发现；不读取文件或访问集群。
package check

import (
	"fmt"

	"example.com/kube-inspector/internal/model"
)

type Finding struct {
	Rule      string `json:"rule"`
	Severity  string `json:"severity"`
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Container string `json:"container,omitempty"`
	Message   string `json:"message"`
}

// CheckPods 执行三个基础检查。restartThreshold 必须大于 0。
// 返回空切片而不是 nil，保证 JSON 输出为 []。
func CheckPods(pods []model.Pod, options model.Options) []Finding {
	findings := make([]Finding, 0)
	for _, pod := range pods {
		// 这些规则检查活动 Pod。已完成、已失败、正在删除的 Pod
		// 需要各自的规则，当前版本不检查它们。

		if pod.Metadata.DeletionTimestamp != nil ||
			(pod.Status.Phase != "Running" && pod.Status.Phase != "Pending") {
			continue
		}

		if pod.Status.Phase == "Running" && !isReady(pod) {
			findings = append(findings, newFinding(pod, "POD_NOT_READY", "WARN", "",
				"phase is Running but Ready condition is not True"))
		}

		// 分别检查普通容器和 init 容器；仅读取，不修改输入对象。
		for _, statuses := range [][]model.ContainerStatus{
			pod.Status.InitContainerStatuses,
			pod.Status.ContainerStatuses,
		} {
			for _, container := range statuses {
				if waiting := container.State.Waiting; waiting != nil {
					switch waiting.Reason {
					case "CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull":
						findings = append(findings, newFinding(pod, "CONTAINER_WAITING", "ERROR",
							container.Name, "waiting reason: "+waiting.Reason))
					}
				}
				if terminated := container.State.Terminated; terminated != nil {
					if terminated.ExitCode != 0 {
						findings = append(findings, newFinding(pod, "CONTAINER_TERMINATED", "ERROR", container.Name, fmt.Sprintf("exitCode=%d,reason=%s", terminated.ExitCode, terminated.Reason)))
					}
				}
				if int(container.RestartCount) >= options.RestartThreshold {
					findings = append(findings, newFinding(pod, "CONTAINER_RESTARTS", "WARN",
						container.Name, fmt.Sprintf("reported restartCount=%d, threshold=%d (not a recent rate)",
							container.RestartCount, options.RestartThreshold)))
				}
			}
		}
	}
	return findings
}

func isReady(pod model.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == "Ready" {
			return condition.Status == "True"
		}
	}
	return false
}

func newFinding(pod model.Pod, rule, severity, container, message string) Finding {
	return Finding{
		Rule: rule, Severity: severity,
		Namespace: pod.Metadata.Namespace, Pod: pod.Metadata.Name,
		Container: container, Message: message,
	}
}

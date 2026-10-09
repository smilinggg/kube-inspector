// Package model 定义第一周需要读取的 Kubernetes JSON 字段。
// 第三周练习：使用 k8s.io/api/core/v1 中的官方类型替换这些简化类型。
package model

type PodList struct {
	Kind  string `json:"kind"`
	Items []Pod  `json:"items"`
}

type Pod struct {
	Metadata Metadata  `json:"metadata"`
	Status   PodStatus `json:"status"`
}

type Metadata struct {
	Name              string  `json:"name"`
	Namespace         string  `json:"namespace"`
	DeletionTimestamp *string `json:"deletionTimestamp,omitempty"`
	CreationTimestamp *string `json:"creationTimestamp,omitempty"`
}

type PodStatus struct {
	Phase                 string            `json:"phase"`
	Conditions            []Condition       `json:"conditions"`
	ContainerStatuses     []ContainerStatus `json:"containerStatuses"`
	InitContainerStatuses []ContainerStatus `json:"initContainerStatuses"`
}

type Condition struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

type ContainerStatus struct {
	Name         string         `json:"name"`
	RestartCount int32          `json:"restartCount"`
	State        ContainerState `json:"state"`
}

type ContainerState struct {
	Waiting    *Waiting    `json:"waiting,omitempty"`
	Terminated *Terminated `json:"terminated,omitempty"`
}

type Waiting struct {
	Reason string `json:"reason"`
}

type Terminated struct {
	ExitCode int32  `json:"exitCode,omitempty"`
	Reason   string `json:"reason"`
	Message  string `json:"message"`
}

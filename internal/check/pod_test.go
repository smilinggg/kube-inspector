package check

import (
	"reflect"
	"testing"
	"time"

	"example.com/kube-inspector/internal/model"
)

// 每一行测试都来自一个真实判断边界，后续规则也按此方式补案例。
func TestCheckPods(t *testing.T) {
	recentCreation := "2026-09-29T01:00:00Z"
	tests := []struct {
		name              string
		phase             string
		ready             string
		reason            string
		restarts          int32
		init              bool
		deleting          bool
		want              []string
		creationTimestamp *string
	}{
		{name: "healthy", phase: "Running", ready: "True"},
		{name: "running but not ready", phase: "Running", ready: "False", want: []string{"POD_NOT_READY"}},
		{name: "missing ready condition", phase: "Running", want: []string{"POD_NOT_READY"}},
		{name: "unknown ready condition", phase: "Running", ready: "Unknown", want: []string{"POD_NOT_READY"}},
		{name: "completed job is excluded", phase: "Succeeded", ready: "False", restarts: 9},
		{name: "failed pod is outside v0 scope", phase: "Failed", restarts: 9},
		{name: "ordinary pending is not an image error", phase: "Pending", reason: "ContainerCreating", creationTimestamp: &recentCreation},
		{name: "pending without timestamp still checks containers", phase: "Pending", reason: "ImagePullBackOff", want: []string{"CONTAINER_WAITING"}},
		{name: "image backoff", phase: "Pending", reason: "ImagePullBackOff", want: []string{"CONTAINER_WAITING"}},
		{name: "image pull error", phase: "Pending", reason: "ErrImagePull", want: []string{"CONTAINER_WAITING"}},
		{name: "crash loop has three findings", phase: "Running", ready: "False", reason: "CrashLoopBackOff", restarts: 7,
			want: []string{"POD_NOT_READY", "CONTAINER_WAITING", "CONTAINER_RESTARTS"}},
		{name: "threshold minus one", phase: "Running", ready: "True", restarts: 4},
		{name: "exact threshold", phase: "Running", ready: "True", restarts: 5, want: []string{"CONTAINER_RESTARTS"}},
		{name: "init container image backoff", phase: "Pending", init: true, reason: "ImagePullBackOff", want: []string{"CONTAINER_WAITING"}, creationTimestamp: &recentCreation},
		{name: "init container restart threshold", phase: "Pending", init: true, restarts: 5, want: []string{"CONTAINER_RESTARTS"}},
		{name: "terminating pod has separate semantics", phase: "Running", deleting: true, ready: "False", restarts: 9},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := model.Options{RestartThreshold: 5, PendingThreshold: 5 * time.Minute}
			pod := model.Pod{Metadata: model.Metadata{Name: "demo", Namespace: "lab"}}
			pod.Metadata.CreationTimestamp = test.creationTimestamp
			pod.Status.Phase = test.phase
			if test.ready != "" {
				pod.Status.Conditions = []model.Condition{{Type: "Ready", Status: test.ready}}
			}
			container := model.ContainerStatus{Name: "app", RestartCount: test.restarts}
			if test.reason != "" {
				container.State.Waiting = &model.Waiting{Reason: test.reason}
			}
			if test.init {
				pod.Status.InitContainerStatuses = []model.ContainerStatus{container}
			} else {
				pod.Status.ContainerStatuses = []model.ContainerStatus{container}
			}
			if test.deleting {
				stamp := "2026-09-23T00:00:00Z"
				pod.Metadata.DeletionTimestamp = &stamp
			}
			findings := CheckPods([]model.Pod{pod}, options)
			var rules []string
			for _, finding := range findings {
				rules = append(rules, finding.Rule)
				if finding.Namespace != "lab" || finding.Pod != "demo" {
					t.Fatalf("resource identity lost: %+v", finding)
				}
			}
			if !reflect.DeepEqual(rules, test.want) {
				t.Fatalf("rules=%v, want %v", rules, test.want)
			}
		})
	}
}

func TestEmptyInputReturnsEmptySlice(t *testing.T) {
	if got := CheckPods(nil, model.Options{RestartThreshold: 5, PendingThreshold: 5 * time.Minute}); got == nil || len(got) != 0 {
		t.Fatalf("expected a non-nil empty slice, got %#v", got)
	}
}

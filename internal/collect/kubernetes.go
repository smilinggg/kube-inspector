package collect

import corev1 "k8s.io/api/core/v1"
import "example.com/kube-inspector/internal/model"

func toModelPod(p corev1.Pod) model.Pod {
	var result model.Pod

	result.Metadata.Name = p.Name

	result.Metadata.Namespace = p.Namespace
	result.Status.Phase = string(p.Status.Phase)

	return result
}

func toModelPods(pods []corev1.Pod) []model.Pod {
	result := make([]model.Pod, 0, len(pods))

	for _, pod := range pods {
		converted := toModelPod(pod)
		result = append(result, converted)
	}

	return result
}

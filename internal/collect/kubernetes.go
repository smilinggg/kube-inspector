package collect

import (
	"context"
	"fmt"

	"example.com/kube-inspector/internal/model"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

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

func ReadPods(
	ctx context.Context,
	kubeconfigPath string,
	namespace string,
) ([]model.Pod, error) {
	// 1. 根据 kubeconfig 生成客户端配置
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}

	// 2. 创建访问 Kubernetes 的客户端
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	// 3. 查询 Pod
	podList, err := clientset.CoreV1().Pods(namespace).List(
		ctx,
		metav1.ListOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	// 4. 查询成功，调用你已经写好的转换函数
	return toModelPods(podList.Items), nil
}

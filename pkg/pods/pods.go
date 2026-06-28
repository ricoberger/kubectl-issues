package pods

import (
	"context"
	"fmt"
	"time"

	"github.com/ricoberger/kubectl-issues/pkg/cmd/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Pod describes a single unhealthy Pod. The fields are already formatted so
// they can be printed in a table directly.
type Pod struct {
	Context   string
	Namespace string
	Name      string
	Ready     string
	Status    string
	Restarts  string
	Age       string
}

// ListUnhealthy returns all unhealthy Pods in the given namespace. If the
// namespace is empty, Pods from all namespaces are returned. The contextName is
// only used to populate the Context field of the returned Pods.
func ListUnhealthy(ctx context.Context, client kubernetes.Interface, namespace, contextName string) ([]Pod, error) {
	list, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []Pod

	for _, pod := range list.Items {
		var shouldReady int64
		var isReady int64
		var restarts int32
		var restartAge string

		for _, containerStatus := range pod.Status.ContainerStatuses {
			shouldReady++
			if containerStatus.Ready {
				isReady++
			}

			if containerStatus.RestartCount > 0 {
				restarts = restarts + containerStatus.RestartCount

				if containerStatus.LastTerminationState.Terminated != nil && containerStatus.LastTerminationState.Terminated.ExitCode != 0 && containerStatus.LastTerminationState.Terminated.FinishedAt.After(time.Now().Add(-24*time.Hour)) {
					restartAge = utils.GetAge(containerStatus.LastTerminationState.Terminated.FinishedAt)
				}
			}
		}

		status := GetStatus(pod)

		if pod.Status.Phase != corev1.PodSucceeded || status != "Completed" {
			if !IsHealthy(pod) || shouldReady != isReady || restartAge != "" {
				restartsCell := fmt.Sprintf("%d", restarts)
				if restartAge != "" {
					restartsCell = fmt.Sprintf("%d (%s ago)", restarts, restartAge)
				}

				result = append(result, Pod{
					Context:   contextName,
					Namespace: pod.Namespace,
					Name:      pod.Name,
					Ready:     fmt.Sprintf("%d/%d", isReady, shouldReady),
					Status:    status,
					Restarts:  restartsCell,
					Age:       utils.GetAge(pod.CreationTimestamp),
				})
			}
		}
	}

	return result, nil
}

// GetStatus returns the human readable status of a Pod, similar to the STATUS
// column of "kubectl get pods".
func GetStatus(pod corev1.Pod) string {
	switch pod.Status.Phase {
	case corev1.PodSucceeded:
		for _, status := range pod.Status.ContainerStatuses {
			if status.State.Terminated != nil {
				return string(status.State.Terminated.Reason)
			}
		}
	case corev1.PodFailed:
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodInitialized && condition.Status == corev1.ConditionFalse {
				return "Init:Error"
			}
			for _, status := range pod.Status.ContainerStatuses {
				if status.State.Terminated != nil {
					return string(status.State.Terminated.Reason)
				}
			}
		}
	case corev1.PodRunning:
		for _, status := range pod.Status.ContainerStatuses {
			if status.State.Waiting != nil {
				return string(status.State.Waiting.Reason)
			}
		}
	case corev1.PodPending:
		for _, status := range pod.Status.ContainerStatuses {
			if status.State.Waiting != nil {
				return string(status.State.Waiting.Reason)
			}
		}
	default:
		if pod.DeletionTimestamp != nil && !pod.DeletionTimestamp.IsZero() {
			return "Terminating"
		}
	}

	return string(pod.Status.Phase)
}

// IsHealthy returns true if the given Pod is considered healthy.
func IsHealthy(pod corev1.Pod) bool {
	switch pod.Status.Phase {
	case corev1.PodSucceeded:
		for _, status := range pod.Status.ContainerStatuses {
			if status.State.Terminated != nil && status.State.Terminated.ExitCode != 0 {
				return false
			}
		}
	case corev1.PodPending:
		if isWaitingContainers(pod) {
			return false
		}
	case corev1.PodRunning:
		for _, condition := range pod.Status.Conditions {
			if condition.Status == corev1.ConditionFalse {
				return false
			}
		}

		if isWaitingContainers(pod) {
			return false
		}

	default:
		return false
	}

	return true
}

func isWaitingContainers(pod corev1.Pod) bool {
	for _, st := range pod.Status.ContainerStatuses {
		if st.State.Waiting != nil {
			return true
		}
	}
	return false
}

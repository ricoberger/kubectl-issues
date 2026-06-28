package pods

import (
	"context"
	"fmt"
	"strconv"
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
		s := describePod(pod)

		// Skip Pods that completed successfully.
		if pod.Status.Phase == corev1.PodSucceeded && s.status == "Completed" {
			continue
		}

		// Only surface Pods that have an actual issue: not Ready (which folds
		// in container readiness, sidecars and readiness gates), or a container
		// that crashed recently (so flapping Pods that are momentarily Ready are
		// still reported).
		if hasPodReadyCondition(pod.Status.Conditions) && !hasRecentCrash(pod) {
			continue
		}

		result = append(result, Pod{
			Context:   contextName,
			Namespace: pod.Namespace,
			Name:      pod.Name,
			Ready:     s.ready(),
			Status:    s.status,
			Restarts:  s.restartsCell(),
			Age:       utils.GetAge(pod.CreationTimestamp),
		})
	}

	return result, nil
}

// hasRecentCrash reports whether any container of the Pod (init or regular)
// terminated with a non-zero exit code within the last 24 hours.
func hasRecentCrash(pod corev1.Pod) bool {
	cutoff := time.Now().Add(-24 * time.Hour)

	check := func(statuses []corev1.ContainerStatus) bool {
		for _, cs := range statuses {
			term := cs.LastTerminationState.Terminated
			if cs.RestartCount > 0 && term != nil && term.ExitCode != 0 && term.FinishedAt.After(cutoff) {
				return true
			}
		}
		return false
	}

	return check(pod.Status.InitContainerStatuses) || check(pod.Status.ContainerStatuses)
}

// nodeUnreachablePodReason is the reason set on a Pod's status when the node it
// runs on becomes unreachable. It mirrors NodeUnreachablePodReason from
// k8s.io/kubernetes/pkg/util/node, which is not importable here.
const nodeUnreachablePodReason = "NodeLost"

// summary holds the READY, STATUS and RESTARTS values for a Pod as printed by
// "kubectl get pods".
type summary struct {
	readyContainers int
	totalContainers int
	status          string
	restarts        int
	lastRestartDate metav1.Time
}

// ready returns the formatted READY column, e.g. "1/2".
func (s summary) ready() string {
	return fmt.Sprintf("%d/%d", s.readyContainers, s.totalContainers)
}

// restartsCell returns the formatted RESTARTS column, e.g. "3" or
// "3 (5m ago)".
func (s summary) restartsCell() string {
	if s.restarts != 0 && !s.lastRestartDate.IsZero() {
		return fmt.Sprintf("%d (%s ago)", s.restarts, utils.GetAge(s.lastRestartDate))
	}
	return strconv.Itoa(s.restarts)
}

// GetStatus returns the human readable status of a Pod, matching the STATUS
// column of "kubectl get pods".
func GetStatus(pod corev1.Pod) string {
	return describePod(pod).status
}

// describePod computes the READY, STATUS and RESTARTS values for a Pod. The
// logic is ported from the upstream printPod function in
// k8s.io/kubernetes/pkg/printers/internalversion so that the result stays
// consistent with kubectl, including init/sidecar containers, scheduling gates,
// terminating/unknown pods, restart counts and signal/exit-code reasons.
func describePod(pod corev1.Pod) summary {
	restarts := 0
	restartableInitContainerRestarts := 0
	totalContainers := len(pod.Spec.Containers)
	readyContainers := 0
	lastRestartDate := metav1.Time{}
	lastRestartableInitContainerRestartDate := metav1.Time{}

	podPhase := pod.Status.Phase
	reason := string(podPhase)
	if pod.Status.Reason != "" {
		reason = pod.Status.Reason
	}

	// If the Pod carries {type:PodScheduled, reason:SchedulingGated}, set the
	// reason to "SchedulingGated".
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodScheduled && condition.Reason == corev1.PodReasonSchedulingGated {
			reason = corev1.PodReasonSchedulingGated
		}
	}

	initContainers := make(map[string]*corev1.Container)
	for i := range pod.Spec.InitContainers {
		initContainers[pod.Spec.InitContainers[i].Name] = &pod.Spec.InitContainers[i]
		if isRestartableInitContainer(&pod.Spec.InitContainers[i]) {
			totalContainers++
		}
	}

	initializing := false
	for i := range pod.Status.InitContainerStatuses {
		container := pod.Status.InitContainerStatuses[i]
		restarts += int(container.RestartCount)
		if container.LastTerminationState.Terminated != nil {
			terminatedDate := container.LastTerminationState.Terminated.FinishedAt
			if lastRestartDate.Before(&terminatedDate) {
				lastRestartDate = terminatedDate
			}
		}
		if isRestartableInitContainer(initContainers[container.Name]) {
			restartableInitContainerRestarts += int(container.RestartCount)
			if container.LastTerminationState.Terminated != nil {
				terminatedDate := container.LastTerminationState.Terminated.FinishedAt
				if lastRestartableInitContainerRestartDate.Before(&terminatedDate) {
					lastRestartableInitContainerRestartDate = terminatedDate
				}
			}
		}
		switch {
		case container.State.Terminated != nil && container.State.Terminated.ExitCode == 0:
			continue
		case isRestartableInitContainer(initContainers[container.Name]) &&
			container.Started != nil && *container.Started:
			if container.Ready {
				readyContainers++
			}
			continue
		case container.State.Terminated != nil:
			// initialization has failed
			if len(container.State.Terminated.Reason) == 0 {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Init:Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("Init:ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else {
				reason = "Init:" + container.State.Terminated.Reason
			}
			initializing = true
		case container.State.Waiting != nil && len(container.State.Waiting.Reason) > 0 && container.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + container.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
			initializing = true
		}
		break
	}

	if !initializing || isPodInitializedConditionTrue(&pod.Status) {
		restarts = restartableInitContainerRestarts
		lastRestartDate = lastRestartableInitContainerRestartDate
		hasRunning := false
		errorReason := ""
		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			container := pod.Status.ContainerStatuses[i]

			restarts += int(container.RestartCount)
			if container.LastTerminationState.Terminated != nil {
				terminatedDate := container.LastTerminationState.Terminated.FinishedAt
				if lastRestartDate.Before(&terminatedDate) {
					lastRestartDate = terminatedDate
				}
			}
			switch {
			case container.State.Waiting != nil && container.State.Waiting.Reason != "":
				reason = container.State.Waiting.Reason
			case container.State.Terminated != nil:
				if len(container.State.Terminated.Reason) > 0 {
					reason = container.State.Terminated.Reason
				} else if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode:%d", container.State.Terminated.ExitCode)
				}
				if container.State.Terminated.ExitCode != 0 {
					errorReason = reason
				}
			case container.Ready && container.State.Running != nil:
				hasRunning = true
				readyContainers++
			}
		}

		// Change the pod status back to "Running" if there is at least one
		// container still reporting as "Running".
		if reason == "Completed" {
			if hasRunning && hasPodReadyCondition(pod.Status.Conditions) {
				reason = "Running"
			} else if errorReason != "" {
				reason = errorReason
			} else if hasRunning {
				reason = "NotReady"
			}
		}
	}

	if pod.DeletionTimestamp != nil && pod.Status.Reason == nodeUnreachablePodReason {
		reason = "Unknown"
	} else if pod.DeletionTimestamp != nil && !isPodPhaseTerminal(podPhase) {
		reason = "Terminating"
	}

	return summary{
		readyContainers: readyContainers,
		totalContainers: totalContainers,
		status:          reason,
		restarts:        restarts,
		lastRestartDate: lastRestartDate,
	}
}

// isRestartableInitContainer reports whether the given init container is a
// sidecar (restartPolicy: Always).
func isRestartableInitContainer(initContainer *corev1.Container) bool {
	if initContainer == nil || initContainer.RestartPolicy == nil {
		return false
	}
	return *initContainer.RestartPolicy == corev1.ContainerRestartPolicyAlways
}

// isPodInitializedConditionTrue reports whether the Pod has the Initialized
// condition set to True.
func isPodInitializedConditionTrue(status *corev1.PodStatus) bool {
	for _, condition := range status.Conditions {
		if condition.Type != corev1.PodInitialized {
			continue
		}
		return condition.Status == corev1.ConditionTrue
	}
	return false
}

// hasPodReadyCondition reports whether the Pod has the Ready condition set to
// True.
func hasPodReadyCondition(conditions []corev1.PodCondition) bool {
	for _, condition := range conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// isPodPhaseTerminal reports whether the given phase is a terminal phase
// (Succeeded or Failed).
func isPodPhaseTerminal(phase corev1.PodPhase) bool {
	return phase == corev1.PodFailed || phase == corev1.PodSucceeded
}

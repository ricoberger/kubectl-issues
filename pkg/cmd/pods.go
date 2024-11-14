package cmd

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/ricoberger/kubectl-issues/pkg/cmd/utils"
	"github.com/ricoberger/kubectl-issues/pkg/writer"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type PodsOptions struct {
	IssuesOptions
}

func newPodsOptions(options IssuesOptions) *PodsOptions {
	return &PodsOptions{
		IssuesOptions: options,
	}
}

func newPodsCommand(factory cmdutil.Factory, options IssuesOptions) *cobra.Command {
	o := newPodsOptions(options)

	cmd := &cobra.Command{
		Use:          "pods",
		Aliases:      []string{"pod", "po"},
		Short:        "List issues with Pods",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(factory, c); err != nil {
				return err
			}

			ctx := context.Background()
			noHeader := c.Flag("no-headers").Changed
			if err := o.Run(ctx, noHeader); err != nil {
				fmt.Fprintln(options.Streams.ErrOut, err.Error())
				return nil
			}
			return nil
		},
	}

	o.ResourceBuilderFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *PodsOptions) Run(ctx context.Context, noHeader bool) error {
	client, err := o.GetClient()
	if err != nil {
		return err
	}

	pods, err := client.CoreV1().Pods(o.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	var matrix [][]string

	for _, pod := range pods.Items {
		var shouldReady int64
		var isReady int64
		var restarts int32
		var hasRecentRestarts bool

		for _, containerStatus := range pod.Status.ContainerStatuses {
			shouldReady++
			if containerStatus.Ready {
				isReady++
			}

			if containerStatus.RestartCount > 0 {
				restarts = restarts + containerStatus.RestartCount

				if containerStatus.LastTerminationState.Terminated != nil && containerStatus.LastTerminationState.Terminated.ExitCode != 0 && containerStatus.LastTerminationState.Terminated.FinishedAt.After(time.Now().Add(-24*time.Hour)) {
					hasRecentRestarts = true
				}
			}
		}

		status := getPodStatus(pod)

		if !(pod.Status.Phase == corev1.PodSucceeded && status == "Completed") {
			if !isPodHealthy(pod) || shouldReady != isReady || hasRecentRestarts {
				row := []string{pod.Namespace, pod.Name, fmt.Sprintf("%d/%d", isReady, shouldReady), status, fmt.Sprintf("%d", restarts), utils.GetAge(pod.CreationTimestamp)}
				matrix = append(matrix, row)
			}
		}
	}

	headers := []string{"NAMESPACE", "NAME", "READY", "STATUS", "RESTARTS", "AGE"}

	buf := bytes.NewBuffer(nil)
	writer.WriteResults(buf, headers, matrix, noHeader)
	fmt.Printf("%s", buf.String())

	return nil
}

func getPodStatus(pod corev1.Pod) string {
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

func isPodWaitingContainers(pod corev1.Pod) bool {
	for _, st := range pod.Status.ContainerStatuses {
		if st.State.Waiting != nil {
			return true
		}
	}
	return false
}

func isPodHealthy(pod corev1.Pod) bool {
	switch pod.Status.Phase {
	case corev1.PodSucceeded:
		for _, status := range pod.Status.ContainerStatuses {
			if status.State.Terminated != nil && status.State.Terminated.ExitCode != 0 {
				return false
			}
		}
	case corev1.PodPending:
		if isPodWaitingContainers(pod) {
			return false
		}
	case corev1.PodRunning:
		for _, condition := range pod.Status.Conditions {
			if condition.Status == corev1.ConditionFalse {
				return false
			}
		}

		if isPodWaitingContainers(pod) {
			return false
		}

	default:
		return false
	}

	return true
}

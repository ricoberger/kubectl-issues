package pods

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func boolPtr(b bool) *bool { return &b }

func restartPolicyPtr(p corev1.ContainerRestartPolicy) *corev1.ContainerRestartPolicy {
	return &p
}

func TestGetStatus(t *testing.T) {
	now := metav1.NewTime(time.Now())

	tests := []struct {
		name string
		pod  corev1.Pod
		want string
	}{
		{
			name: "running and ready",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
					},
				},
			},
			want: "Running",
		},
		{
			name: "crash loop back off",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}},
					},
				},
			},
			want: "CrashLoopBackOff",
		},
		{
			name: "completed job",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodSucceeded,
					ContainerStatuses: []corev1.ContainerStatus{
						{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Completed", ExitCode: 0}}},
					},
				},
			},
			want: "Completed",
		},
		{
			name: "evicted pod uses status reason",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase:  corev1.PodFailed,
					Reason: "Evicted",
				},
			},
			want: "Evicted",
		},
		{
			name: "terminating running pod",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &now},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
					},
				},
			},
			want: "Terminating",
		},
		{
			name: "node lost shows unknown",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &now},
				Status: corev1.PodStatus{
					Phase:  corev1.PodRunning,
					Reason: "NodeLost",
				},
			},
			want: "Unknown",
		},
		{
			name: "init container crash loop",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "init"}},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					InitContainerStatuses: []corev1.ContainerStatus{
						{Name: "init", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}},
					},
				},
			},
			want: "Init:CrashLoopBackOff",
		},
		{
			name: "init container progress",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "init1"}, {Name: "init2"}},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					InitContainerStatuses: []corev1.ContainerStatus{
						{Name: "init1", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "PodInitializing"}}},
					},
				},
			},
			want: "Init:0/2",
		},
		{
			name: "terminated without reason uses exit code",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 137}}},
					},
				},
			},
			want: "ExitCode:137",
		},
		{
			name: "terminated without reason uses signal",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Signal: 9}}},
					},
				},
			},
			want: "Signal:9",
		},
		{
			name: "scheduling gated",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					Conditions: []corev1.PodCondition{
						{Type: corev1.PodScheduled, Reason: corev1.PodReasonSchedulingGated},
					},
				},
			},
			want: "SchedulingGated",
		},
		{
			name: "completed reason but a container still running and ready",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{Type: corev1.PodReady, Status: corev1.ConditionTrue},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Completed", ExitCode: 0}}},
						{Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
					},
				},
			},
			want: "Running",
		},
		{
			name: "sidecar init container started does not block main containers",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{Name: "sidecar", RestartPolicy: restartPolicyPtr(corev1.ContainerRestartPolicyAlways)},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					InitContainerStatuses: []corev1.ContainerStatus{
						{Name: "sidecar", Started: boolPtr(true), Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"}}},
					},
				},
			},
			want: "ImagePullBackOff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetStatus(tt.pod); got != tt.want {
				t.Errorf("GetStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDescribePodReadyAndRestarts(t *testing.T) {
	old := metav1.NewTime(time.Now().Add(-5 * time.Minute))
	older := metav1.NewTime(time.Now().Add(-2 * time.Hour))

	tests := []struct {
		name         string
		pod          corev1.Pod
		wantReady    string
		wantRestarts string
	}{
		{
			name: "sidecar counts towards total and ready",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app"}},
					InitContainers: []corev1.Container{
						{Name: "sidecar", RestartPolicy: restartPolicyPtr(corev1.ContainerRestartPolicyAlways)},
						{Name: "setup"},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					InitContainerStatuses: []corev1.ContainerStatus{
						{Name: "setup", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 0}}},
						{Name: "sidecar", Started: boolPtr(true), Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{Name: "app", Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
					},
				},
			},
			wantReady:    "2/2",
			wantRestarts: "0",
		},
		{
			name: "restarts include init and main containers",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers:     []corev1.Container{{Name: "app"}},
					InitContainers: []corev1.Container{{Name: "sidecar", RestartPolicy: restartPolicyPtr(corev1.ContainerRestartPolicyAlways)}},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					InitContainerStatuses: []corev1.ContainerStatus{
						{
							Name:         "sidecar",
							Started:      boolPtr(true),
							Ready:        true,
							RestartCount: 2,
							State:        corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
							LastTerminationState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, FinishedAt: older},
							},
						},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:         "app",
							Ready:        true,
							RestartCount: 3,
							State:        corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
							LastTerminationState: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, FinishedAt: old},
							},
						},
					},
				},
			},
			wantReady:    "2/2",
			wantRestarts: "5 (5m ago)",
		},
		{
			name: "no restarts shows plain count",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app"}}},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{Name: "app", Ready: false, State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}},
					},
				},
			},
			wantReady:    "0/1",
			wantRestarts: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := describePod(tt.pod)
			if got := s.ready(); got != tt.wantReady {
				t.Errorf("ready() = %q, want %q", got, tt.wantReady)
			}
			if got := s.restartsCell(); got != tt.wantRestarts {
				t.Errorf("restartsCell() = %q, want %q", got, tt.wantRestarts)
			}
		})
	}
}

package translator

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/vk-azure-container-apps-sandboxes/pkg/types"
)

const (
	containerStateRunning    = "Running"
	containerStateTerminated = "Terminated"
)

func SandboxToPodStatus(sandbox *types.Sandbox) *corev1.PodStatus {
	containerStatuses := containerStatuses(sandbox)
	phase := podPhase(sandbox.State, containerStatuses)

	ready := len(containerStatuses) > 0
	for _, containerStatus := range containerStatuses {
		ready = ready && containerStatus.Ready
	}

	status := &corev1.PodStatus{
		Phase:             phase,
		ContainerStatuses: containerStatuses,
		Conditions: []corev1.PodCondition{
			condition(corev1.PodScheduled, true, sandbox.CreatedAt),
			condition(corev1.PodInitialized, true, sandbox.CreatedAt),
			condition(corev1.ContainersReady, ready, sandbox.CreatedAt),
			condition(corev1.PodReady, ready && phase == corev1.PodRunning, sandbox.CreatedAt),
		},
	}
	if !sandbox.CreatedAt.IsZero() {
		startTime := metav1.NewTime(sandbox.CreatedAt)
		status.StartTime = &startTime
	}
	return status
}

func podPhase(state string, containerStatuses []corev1.ContainerStatus) corev1.PodPhase {
	switch state {
	case types.SandboxStateCreating:
		return corev1.PodPending
	case types.SandboxStateRunning, types.SandboxStateIdle, types.SandboxStateStopping:
		return corev1.PodRunning
	case types.SandboxStateStopped:
		for _, containerStatus := range containerStatuses {
			if terminated := containerStatus.State.Terminated; terminated == nil || terminated.ExitCode != 0 {
				return corev1.PodFailed
			}
		}
		return corev1.PodSucceeded
	case types.SandboxStateStopFailed:
		return corev1.PodFailed
	default:
		return corev1.PodUnknown
	}
}

func containerStatuses(sandbox *types.Sandbox) []corev1.ContainerStatus {
	reported := map[string]types.ContainerStatus{}
	for _, containerStatus := range sandbox.ContainerStatuses {
		reported[containerStatus.Name] = containerStatus
	}

	var names []string
	if sandbox.SourcesRef != nil && sandbox.SourcesRef.Pod != nil {
		for _, container := range sandbox.SourcesRef.Pod.Containers {
			names = append(names, container.Name)
		}
	} else {
		for _, containerStatus := range sandbox.ContainerStatuses {
			names = append(names, containerStatus.Name)
		}
	}

	statuses := make([]corev1.ContainerStatus, 0, len(names))
	for _, name := range names {
		containerStatus, ok := reported[name]
		if !ok {
			statuses = append(statuses, corev1.ContainerStatus{
				Name:  name,
				State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ContainerCreating"}},
			})
			continue
		}
		statuses = append(statuses, containerStatusToK8s(containerStatus))
	}
	return statuses
}

func containerStatusToK8s(containerStatus types.ContainerStatus) corev1.ContainerStatus {
	started := containerStatus.Started
	status := corev1.ContainerStatus{
		Name:         containerStatus.Name,
		Ready:        containerStatus.Ready,
		Started:      &started,
		RestartCount: int32(containerStatus.RestartCount),
	}

	switch containerStatus.State {
	case containerStateRunning:
		status.State.Running = &corev1.ContainerStateRunning{StartedAt: metaTime(containerStatus.LastStartedAt)}
	case containerStateTerminated:
		terminated := &corev1.ContainerStateTerminated{
			Reason:     containerStatus.Reason,
			Message:    containerStatus.Message,
			StartedAt:  metaTime(containerStatus.LastStartedAt),
			FinishedAt: metaTime(containerStatus.LastFinishedAt),
		}
		if containerStatus.LastExitCode != nil {
			terminated.ExitCode = int32(*containerStatus.LastExitCode)
		}
		status.State.Terminated = terminated
	default:
		reason := containerStatus.Reason
		if reason == "" {
			reason = containerStatus.State
		}
		status.State.Waiting = &corev1.ContainerStateWaiting{Reason: reason, Message: containerStatus.Message}
	}
	return status
}

func condition(conditionType corev1.PodConditionType, ok bool, transitionTime time.Time) corev1.PodCondition {
	status := corev1.ConditionFalse
	if ok {
		status = corev1.ConditionTrue
	}
	return corev1.PodCondition{Type: conditionType, Status: status, LastTransitionTime: metav1.NewTime(transitionTime)}
}

func metaTime(t *time.Time) metav1.Time {
	if t == nil {
		return metav1.Time{}
	}
	return metav1.NewTime(*t)
}

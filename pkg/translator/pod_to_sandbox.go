package translator

import (
	"fmt"

	"github.com/Azure/vk-azure-container-apps-sandboxes/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func ValidatePod(pod *corev1.Pod) error {
	if len(pod.Spec.InitContainers) > 0 {
		return fmt.Errorf("init containers are not supported")
	}
	for _, container := range pod.Spec.Containers {
		for _, envVar := range container.Env {
			if envVar.ValueFrom != nil {
				return fmt.Errorf("container %q: env var %q: valueFrom is not supported", container.Name, envVar.Name)
			}
		}
	}
	return nil
}

func PodToSandbox(pod *corev1.Pod, diskImageIDs map[string]string) *types.Sandbox {
	var containers []types.ContainerSpec
	for _, container := range pod.Spec.Containers {
		env := map[string]string{}
		for _, envVar := range container.Env {
			env[envVar.Name] = envVar.Value
		}
		containers = append(containers, types.ContainerSpec{
			Name:      container.Name,
			DiskImage: &types.SandboxDiskImage{ID: diskImageIDs[container.Image]},
			Command:   container.Command,
			Args:      container.Args,
			Env:       env,
		})
	}

	return &types.Sandbox{
		Labels: map[string]string{
			types.ManagedByLabelKey: types.ManagedByLabelValue,
			types.NamespaceLabelKey: pod.Namespace,
			types.PodNameLabelKey:   pod.Name,
			types.PodUIDLabelKey:    string(pod.UID),
		},
		SourcesRef: &types.SandboxSource{Pod: &types.SandboxPod{
			Containers:    containers,
			RestartPolicy: string(pod.Spec.RestartPolicy),
		}},
		Resources: sandboxResources(pod),
	}
}

const (
	defaultCPU    = "1000m"
	defaultMemory = "2048Mi"
)

func sandboxResources(pod *corev1.Pod) *types.SandboxResources {
	cpu := resource.NewMilliQuantity(0, resource.DecimalSI)
	memory := resource.NewQuantity(0, resource.BinarySI)
	for _, container := range pod.Spec.Containers {
		cpu.Add(containerQuantity(container, corev1.ResourceCPU))
		memory.Add(containerQuantity(container, corev1.ResourceMemory))
	}

	resources := &types.SandboxResources{CPU: defaultCPU, Memory: defaultMemory}
	if !cpu.IsZero() {
		resources.CPU = cpu.String()
	}
	if !memory.IsZero() {
		resources.Memory = memory.String()
	}
	return resources
}

func containerQuantity(container corev1.Container, name corev1.ResourceName) resource.Quantity {
	if quantity, ok := container.Resources.Limits[name]; ok {
		return quantity
	}
	return container.Resources.Requests[name]
}

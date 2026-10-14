package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
	"github.com/virtual-kubelet/virtual-kubelet/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	corev1listers "k8s.io/client-go/listers/core/v1"

	"github.com/Azure/vk-azure-container-apps-sandboxes/pkg/clients"
	"github.com/Azure/vk-azure-container-apps-sandboxes/pkg/translator"
	"github.com/Azure/vk-azure-container-apps-sandboxes/pkg/types"
)

type AcaSandboxesProvider struct {
	sandboxesClient  *clients.AcaSandboxesAPIClient
	diskImagesClient *clients.AcaDiskImagesAPIClient
	podLister        corev1listers.PodLister
}

func NewAcaSandboxesProvider(sandboxesClient *clients.AcaSandboxesAPIClient, diskImagesClient *clients.AcaDiskImagesAPIClient, podLister corev1listers.PodLister) *AcaSandboxesProvider {
	return &AcaSandboxesProvider{sandboxesClient: sandboxesClient, diskImagesClient: diskImagesClient, podLister: podLister}
}

func (p *AcaSandboxesProvider) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	if err := translator.ValidatePod(pod); err != nil {
		return err
	}

	diskImageIDs := map[string]string{}
	for _, container := range pod.Spec.Containers {
		if _, ok := diskImageIDs[container.Image]; ok {
			continue
		}
		log.G(ctx).WithField("image", container.Image).Info("importing disk image")
		// For now, treating all pod creates essentially as ImagePullPolicy: Always
		diskImage, err := p.diskImagesClient.CreateAndWait(ctx, &types.DiskImage{
			Labels: map[string]string{types.ManagedByLabelKey: types.ManagedByLabelValue},
			Source: &types.DiskImageSource{Kind: types.DiskImageSourceKindRegistry, ImageURL: container.Image},
		})
		if err != nil {
			return fmt.Errorf("importing image %q: %w", container.Image, err)
		}
		log.G(ctx).WithField("image", container.Image).WithField("diskImageID", diskImage.ID).Info("disk image ready")
		diskImageIDs[container.Image] = diskImage.ID
	}

	log.G(ctx).Info("creating sandbox")
	sandbox, err := p.sandboxesClient.Create(ctx, translator.PodToSandbox(pod, diskImageIDs))
	if err != nil {
		return fmt.Errorf("creating sandbox for pod %s/%s: %w", pod.Namespace, pod.Name, err)
	}
	log.G(ctx).WithField("sandboxID", sandbox.ID).WithField("state", sandbox.State).Info("created sandbox")
	return nil
}

// Since sandboxes are considered immutable, updating a pod involves deleting its existing sandbox and creating a new one.
func (p *AcaSandboxesProvider) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
	if err := p.DeletePod(ctx, pod); err != nil {
		return err
	}
	return p.CreatePod(ctx, pod)
}

func (p *AcaSandboxesProvider) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	sandboxes, err := p.sandboxesClient.List(ctx)
	if err != nil {
		return fmt.Errorf("listing sandboxes to delete pod %s/%s: %w", pod.Namespace, pod.Name, err)
	}

	deleted := false
	for _, sandbox := range sandboxes {
		if sandbox.Labels[types.NamespaceLabelKey] != pod.Namespace ||
			sandbox.Labels[types.PodNameLabelKey] != pod.Name ||
			sandbox.Labels[types.PodUIDLabelKey] != string(pod.UID) {
			continue
		}
		log.G(ctx).WithField("sandboxID", sandbox.ID).Info("deleting sandbox")
		err := p.sandboxesClient.Delete(ctx, sandbox.ID)
		var responseErr *clients.ResponseError
		if err != nil && !(errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusNotFound) {
			return fmt.Errorf("deleting sandbox %s for pod %s/%s: %w", sandbox.ID, pod.Namespace, pod.Name, err)
		}
		deleted = true
	}
	if !deleted {
		log.G(ctx).Info("no sandbox to delete")
	}
	return nil
}

func (p *AcaSandboxesProvider) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	sandbox, err := p.findSandbox(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	return p.podFor(*sandbox), nil
}

func (p *AcaSandboxesProvider) GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error) {
	sandbox, err := p.findSandbox(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	sandbox, err = p.sandboxesClient.Get(ctx, sandbox.ID)
	if err != nil {
		return nil, fmt.Errorf("getting sandbox status for pod %s/%s: %w", namespace, name, err)
	}
	return translator.SandboxToPodStatus(sandbox), nil
}

// findSandbox returns the sandbox for the pod, or a not-found error if there is none.
func (p *AcaSandboxesProvider) findSandbox(ctx context.Context, namespace, name string) (*types.Sandbox, error) {
	sandboxes, err := p.sandboxesClient.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing sandboxes to find pod %s/%s: %w", namespace, name, err)
	}

	for i := range sandboxes {
		if sandboxes[i].Labels[types.NamespaceLabelKey] == namespace && sandboxes[i].Labels[types.PodNameLabelKey] == name {
			return &sandboxes[i], nil
		}
	}
	return nil, errdefs.NotFoundf("pod %s/%s not found", namespace, name)
}

func (p *AcaSandboxesProvider) GetPods(ctx context.Context) ([]*corev1.Pod, error) {
	sandboxes, err := p.sandboxesClient.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing sandboxes: %w", err)
	}

	var pods []*corev1.Pod
	for _, sandbox := range sandboxes {
		if sandbox.Labels[types.NamespaceLabelKey] == "" || sandbox.Labels[types.PodNameLabelKey] == "" {
			continue
		}
		pods = append(pods, p.podFor(sandbox))
	}
	return pods, nil
}

func (p *AcaSandboxesProvider) podFor(sandbox types.Sandbox) *corev1.Pod {
	namespace, name := sandbox.Labels[types.NamespaceLabelKey], sandbox.Labels[types.PodNameLabelKey]
	if pod, err := p.podLister.Pods(namespace).Get(name); err == nil {
		return pod.DeepCopy()
	}
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Namespace: namespace,
		Name:      name,
		UID:       k8stypes.UID(sandbox.Labels[types.PodUIDLabelKey]),
	}}
}

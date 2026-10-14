package provider

import (
	"context"
	"errors"
	"io"

	dto "github.com/prometheus/client_model/go"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	"github.com/virtual-kubelet/virtual-kubelet/node/nodeutil"
	statsv1alpha1 "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

var _ nodeutil.Provider = (*AcaSandboxesProvider)(nil)

var errNotImplemented = errors.New("not implemented")

func (p *AcaSandboxesProvider) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (io.ReadCloser, error) {
	return nil, errNotImplemented
}

func (p *AcaSandboxesProvider) RunInContainer(ctx context.Context, namespace, podName, containerName string, cmd []string, attach api.AttachIO) error {
	return errNotImplemented
}

func (p *AcaSandboxesProvider) AttachToContainer(ctx context.Context, namespace, podName, containerName string, attach api.AttachIO) error {
	return errNotImplemented
}

func (p *AcaSandboxesProvider) GetStatsSummary(ctx context.Context) (*statsv1alpha1.Summary, error) {
	return nil, errNotImplemented
}

func (p *AcaSandboxesProvider) GetMetricsResource(ctx context.Context) ([]*dto.MetricFamily, error) {
	return nil, errNotImplemented
}

func (p *AcaSandboxesProvider) PortForward(ctx context.Context, namespace, pod string, port int32, stream io.ReadWriteCloser) error {
	return errNotImplemented
}

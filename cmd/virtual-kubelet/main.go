package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/virtual-kubelet/virtual-kubelet/node"
	"github.com/virtual-kubelet/virtual-kubelet/node/nodeutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/Azure/vk-azure-container-apps-sandboxes/pkg/clients"
	"github.com/Azure/vk-azure-container-apps-sandboxes/pkg/provider"
)

func main() {
	if err := run(); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	nodeName := os.Getenv("NODE_NAME")
	for _, name := range []string{"SUBSCRIPTION_ID", "RESOURCE_GROUP", "SANDBOX_GROUP", "REGION"} {
		if os.Getenv(name) == "" {
			return fmt.Errorf("%s environment variable is required", name)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Default Azure Credential will get the managed identity credential tied to the service account.
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return fmt.Errorf("creating Azure credential: %w", err)
	}
	adc, err := clients.NewADCClient(credential, os.Getenv("REGION"))
	if err != nil {
		return err
	}
	options := clients.AcaSandboxesAPIClientOptions{
		SubscriptionID: os.Getenv("SUBSCRIPTION_ID"),
		ResourceGroup:  os.Getenv("RESOURCE_GROUP"),
		SandboxGroup:   os.Getenv("SANDBOX_GROUP"),
	}
	sandboxesClient, err := clients.NewAcaSandboxesAPIClient(adc, options)
	if err != nil {
		return err
	}
	diskImagesClient, err := clients.NewAcaDiskImagesAPIClient(adc, options)
	if err != nil {
		return err
	}

	virtualNode, err := nodeutil.NewNode(nodeName,
		func(config nodeutil.ProviderConfig) (nodeutil.Provider, node.NodeProvider, error) {
			return provider.NewAcaSandboxesProvider(sandboxesClient, diskImagesClient, config.Pods), nil, nil
		},
		configureNode,
	)
	if err != nil {
		return fmt.Errorf("creating virtual node: %w", err)
	}

	return virtualNode.Run(ctx)
}

func configureNode(config *nodeutil.NodeConfig) error {
	config.SkipDownwardAPIResolution = true

	config.NodeSpec.Labels["kubernetes.io/os"] = "linux"
	config.NodeSpec.Labels["kubernetes.io/arch"] = "amd64"
	config.NodeSpec.Spec.Taints = []corev1.Taint{{
		Key:    "virtual-kubelet.io/provider",
		Value:  "aca-sandboxes",
		Effect: corev1.TaintEffectNoSchedule,
	}}
	config.NodeSpec.Status.NodeInfo.OperatingSystem = "linux"
	config.NodeSpec.Status.NodeInfo.Architecture = "amd64"

	capacity := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("1000"),
		corev1.ResourceMemory: resource.MustParse("4Ti"),
		corev1.ResourcePods:   resource.MustParse("1000"),
	}
	config.NodeSpec.Status.Capacity = capacity
	config.NodeSpec.Status.Allocatable = capacity
	return nil
}

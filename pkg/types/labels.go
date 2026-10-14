package types

// Every sandbox and disk image the provider creates carries the managed-by label, so the provider can list the
// resources it owns. Sandboxes also carry their pod's identity.
const (
	ManagedByLabelKey   = "managed-by"
	ManagedByLabelValue = "virtual-kubelet-aca-sandboxes"

	NamespaceLabelKey = "kubernetes-namespace"
	PodNameLabelKey   = "kubernetes-pod-name"
	PodUIDLabelKey    = "kubernetes-pod-uid"
)

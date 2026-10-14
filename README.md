# Azure Container Apps Sandboxes Virtual Kubelet Provider

A [Virtual Kubelet](https://virtual-kubelet.io) provider that runs pods as
[Azure Container Apps Sandboxes](https://sandboxes.azure.com/docs/sandboxes). Azure Container App Sandboxes are isolated, on-demand microVMs - built for AI agents, code execution, and dev environments. Each sandbox is a hardware-isolated microVM with its own Linux kernel.

>Thank you for exploring the preview version of the Virtual Kubelet implementation for Azure Container Apps Sandboxes. Please share your feedback by opening an issue in the repository or emailing [ACASupport@microsoft.com](mailto:ACASupport@microsoft.com).

## Prerequisites

Tools:

- [Go](https://go.dev/dl/) 1.26 or newer
- [Azure CLI](https://learn.microsoft.com/cli/azure/install-azure-cli), logged in with `az login`
- [Helm](https://helm.sh/docs/intro/install/) 3 and `kubectl`
- Bash, to run the deploy script (Git Bash or WSL on Windows)

Azure resources:

- A **sandbox group** (`Microsoft.App/SandboxGroups`) in a resource group and region. Sandboxes, and the disk images
  they use, are scoped to it.
- An **AKS cluster** to install the virtual node on.
- An **Azure Container Registry** to hold the provider's own image. The AKS cluster must be able to pull from it via Kubelet identity or by enabling anonymous pull.
- An **Azure identity** for the provider, with the **Container Apps SandboxGroup Data Owner** role on the sandbox
  group (or its resource group or subscription). Only Microsoft Entra identities work. The provider uses it through
  [workload identity](https://learn.microsoft.com/azure/aks/workload-identity-overview), so the AKS cluster needs the
  OIDC issuer and workload identity enabled. See [Azure identity](#azure-identity).

### Azure identity

The provider authenticates to Azure with workload identity in AKS. Before you
run the script, set up the identity whose client ID you will pass:

1. Use an existing user-assigned managed identity, or create one.
2. Give it the **Container Apps SandboxGroup Data Owner** role on the sandbox group.
3. Add a federated credential to it for the provider's service account. The subject is `system:serviceaccount:virtual-kubelet:virtual-kubelet`. The issuer is the AKS cluster's OIDC issuer URL which can be obtained by running a command like ` az aks show --resource-group <rg> --name <cluster-name> --query "oidcIssuerProfile.issuerUrl" --output tsv`

## Deploy to AKS

`deploy/deploy.sh` builds the image in ACR, pushes it, and installs the Helm chart:

```bash
bash deploy/deploy.sh \
  --sandbox-group-id /subscriptions/<id>/resourceGroups/<rg>/providers/Microsoft.App/sandboxGroups/<name> \
  --aks-resource-id /subscriptions/<id>/resourceGroups/<rg>/providers/Microsoft.ContainerService/managedClusters/<name> \
  --acr-name <registry-name> \
  --client-id <identity-client-id>
```

Optional flags are `--region` (defaults to the AKS cluster's region), `--image-tag` (defaults to a timestamp). Run it with `--help` for the full list.

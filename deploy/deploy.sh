#!/usr/bin/env bash
#
# Builds and pushes the virtual kubelet image and installs the helm chart.

set -euo pipefail

native_path() {
  local tool="$1" path="$2"
  if command -v wslpath > /dev/null && [[ "$(command -v "$tool")" == /mnt/* ]]; then
    wslpath -w "$path"
  else
    echo "$path"
  fi
}

usage() {
  cat <<'EOF'
Usage: deploy.sh --sandbox-group-id <id> --aks-resource-id <id> --acr-name <name> --client-id <id> [options]

Required:
  --sandbox-group-id  Resource ID of the sandbox group
  --aks-resource-id   Resource ID of the AKS cluster to install on
  --acr-name          Name of the Azure Container Registry to push the image to
  --client-id         Client ID of the Azure identity the provider uses through workload identity

Options:
  --region            Region of the sandbox group (default: the AKS cluster's region)
  --image-tag         Tag for the image (default: a timestamp, so every run deploys a new image)
  -h, --help          Show this help
EOF
}

sandbox_group_id=""
aks_resource_id=""
acr_name=""
client_id=""
region=""
image_tag="$(date +%Y%m%d%H%M%S)"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --sandbox-group-id) sandbox_group_id="${2:?--sandbox-group-id needs a value}"; shift 2 ;;
    --aks-resource-id) aks_resource_id="${2:?--aks-resource-id needs a value}"; shift 2 ;;
    --acr-name) acr_name="${2:?--acr-name needs a value}"; shift 2 ;;
    --client-id) client_id="${2:?--client-id needs a value}"; shift 2 ;;
    --region) region="${2:?--region needs a value}"; shift 2 ;;
    --image-tag) image_tag="${2:?--image-tag needs a value}"; shift 2 ;;
    -h | --help) usage; exit 0 ;;
    *) echo "error: unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

for required in sandbox_group_id aks_resource_id acr_name client_id; do
  if [[ -z "${!required}" ]]; then
    echo "error: --${required//_/-} is required" >&2
    usage >&2
    exit 2
  fi
done

shopt -s nocasematch
sandbox_group_pattern='^/subscriptions/([^/]+)/resourceGroups/([^/]+)/providers/Microsoft\.App/sandboxGroups/([^/]+)$'
if [[ ! "$sandbox_group_id" =~ $sandbox_group_pattern ]]; then
  echo "error: '$sandbox_group_id' is not a sandbox group resource ID." >&2
  echo "Expected /subscriptions/<id>/resourceGroups/<name>/providers/Microsoft.App/sandboxGroups/<name>." >&2
  exit 2
fi
subscription_id="${BASH_REMATCH[1]}"
resource_group="${BASH_REMATCH[2]}"
sandbox_group="${BASH_REMATCH[3]}"

aks_pattern='^/subscriptions/([^/]+)/resourceGroups/([^/]+)/providers/Microsoft\.ContainerService/managedClusters/([^/]+)$'
if [[ ! "$aks_resource_id" =~ $aks_pattern ]]; then
  echo "error: '$aks_resource_id' is not an AKS cluster resource ID." >&2
  echo "Expected /subscriptions/<id>/resourceGroups/<name>/providers/Microsoft.ContainerService/managedClusters/<name>." >&2
  exit 2
fi
shopt -u nocasematch
aks_subscription_id="${BASH_REMATCH[1]}"
aks_resource_group="${BASH_REMATCH[2]}"
aks_name="${BASH_REMATCH[3]}"

for tool in az helm; do
  if ! command -v "$tool" > /dev/null; then
    echo "error: '$tool' was not found on the PATH." >&2
    exit 1
  fi
done

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
chart_path="$repo_root/charts"
dockerfile="cmd/virtual-kubelet/Dockerfile"
image_name="virtual-kubelet"

if [[ -z "$region" ]]; then
  echo "Reading the region from AKS cluster $aks_name"
  region="$(az aks show --subscription "$aks_subscription_id" --resource-group "$aks_resource_group" --name "$aks_name" --query location --output tsv | tr -d '\r')"
fi

login_server="$(az acr show --name "$acr_name" --query loginServer --output tsv | tr -d '\r')"

echo "Building $login_server/$image_name:$image_tag in ACR"
az acr build --registry "$acr_name" --image "$image_name:$image_tag" --file "$dockerfile" "$(native_path az "$repo_root")"

kubeconfig="$(mktemp)"
trap 'rm -f "$kubeconfig"' EXIT

echo "Getting credentials for AKS cluster $aks_name"
az aks get-credentials --subscription "$aks_subscription_id" --resource-group "$aks_resource_group" --name "$aks_name" --file "$(native_path az "$kubeconfig")" --overwrite-existing

echo "Installing virtual-kubelet into namespace virtual-kubelet"
helm upgrade --install virtual-kubelet "$(native_path helm "$chart_path")" \
  --kubeconfig "$(native_path helm "$kubeconfig")" \
  --namespace virtual-kubelet --create-namespace \
  --set-string "image.repository=$login_server" \
  --set-string "image.tag=$image_tag" \
  --set-string "subscriptionId=$subscription_id" \
  --set-string "resourceGroup=$resource_group" \
  --set-string "sandboxGroup=$sandbox_group" \
  --set-string "region=$region" \
  --set-string "workloadIdentity.clientId=$client_id" \
  --wait --timeout 5m

  echo "Helm Upgrade finished"
#!/usr/bin/env bash
# Builds and pushes the virtual kubelet image.

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: image-deploy.sh --acr-name <name> [--image-tag <tag>]

Required:
  --acr-name   Name of the Azure Container Registry to push the image to

Options:
  --image-tag  Tag for the image (default: a timestamp)
  -h, --help   Show this help
EOF
}

acr_name=""
image_tag="$(date +%Y%m%d%H%M%S)"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --acr-name | --image-tag)
      if [[ $# -lt 2 || -z "$2" || "$2" == --* ]]; then
        echo "error: $1 needs a value" >&2
        exit 2
      fi
      case "$1" in
        --acr-name) acr_name="$2" ;;
        --image-tag) image_tag="$2" ;;
      esac
      shift 2
      ;;
    -h | --help) usage; exit 0 ;;
    *) echo "error: unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

if [[ -z "$acr_name" ]]; then
  echo "error: --acr-name is required" >&2
  usage >&2
  exit 2
fi

if ! command -v az > /dev/null; then
  echo "error: 'az' was not found on the PATH." >&2
  exit 1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
build_context="$repo_root"
if command -v wslpath > /dev/null && [[ "$(command -v az)" == /mnt/* ]]; then
  build_context="$(wslpath -w "$repo_root")"
fi

login_server="$(az acr show --name "$acr_name" --query loginServer --output tsv | tr -d '\r')"
if [[ -z "$login_server" ]]; then
  echo "error: no login server returned for registry '$acr_name'." >&2
  exit 1
fi

image_name="virtual-kubelet"
image_ref="$login_server/$image_name:$image_tag"
echo "Building $image_ref in ACR" >&2
az acr build --registry "$acr_name" --image "$image_name:$image_tag" \
  --file "cmd/virtual-kubelet/Dockerfile" "$build_context" >&2

printf '%s\n' "$image_ref"

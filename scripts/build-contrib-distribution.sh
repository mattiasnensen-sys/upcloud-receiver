#!/usr/bin/env bash
set -euo pipefail

OTELCOL_VERSION="${1:-0.146.1}"
CONFIG_PATH="${2:-./_build/otelcol-contrib-upcloud-builder-config.yaml}"

MANIFEST_URL="https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector-releases/v${OTELCOL_VERSION}/distributions/otelcol-contrib/manifest.yaml"
DIST_MODULE="github.com/upcloud-community/opentelemetry-upcloud-receiver/collector-contrib"
DIST_NAME="otelcol-contrib-upcloud"
DIST_DESCRIPTION="OpenTelemetry Collector Contrib with UpCloud receiver"
DIST_OUTPUT_PATH="./_build"

tmp_manifest="$(mktemp)"
trap 'rm -f "${tmp_manifest}"' EXIT

mkdir -p "$(dirname "${CONFIG_PATH}")"
curl -fsSL "${MANIFEST_URL}" > "${tmp_manifest}"

awk \
  -v dist_module="${DIST_MODULE}" \
  -v dist_name="${DIST_NAME}" \
  -v dist_description="${DIST_DESCRIPTION}" \
  -v dist_output_path="${DIST_OUTPUT_PATH}" '
BEGIN {
  in_dist = 0
  added_receiver = 0
}
{
  if ($0 == "dist:") {
    in_dist = 1
    print
    next
  }
  if (in_dist == 1 && $0 ~ /^[a-z][a-z_]*:$/) {
    in_dist = 0
  }
  if (in_dist == 1) {
    if ($0 ~ /^  module: /) {
      print "  module: " dist_module
      next
    }
    if ($0 ~ /^  name: /) {
      print "  name: " dist_name
      next
    }
    if ($0 ~ /^  description: /) {
      print "  description: " dist_description
      next
    }
    if ($0 ~ /^  output_path: /) {
      print "  output_path: " dist_output_path
      next
    }
  }
  print
  if ($0 == "receivers:" && added_receiver == 0) {
    print "  - gomod: github.com/upcloud-community/opentelemetry-upcloud-receiver v0.0.0"
    print "    import: github.com/upcloud-community/opentelemetry-upcloud-receiver/receiver/upcloudreceiver"
    print "    name: upcloudreceiver"
    print "    path: ./"
    added_receiver = 1
  }
}
END {
  if (added_receiver == 0) {
    print "receivers block not found in upstream manifest" > "/dev/stderr"
    exit 1
  }
}
' "${tmp_manifest}" > "${CONFIG_PATH}"

go install "go.opentelemetry.io/collector/cmd/builder@v${OTELCOL_VERSION}"
"$(go env GOPATH)/bin/builder" --skip-strict-versioning --config "${CONFIG_PATH}"

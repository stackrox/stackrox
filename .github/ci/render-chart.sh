#!/usr/bin/env bash
# Render a StackRox helm chart from the in-repo .htpl templates.
# Replaces roxctl helm output — no external tools needed.
#
# Usage: ./render-chart.sh <central|sensor> <output-dir> <main-tag> <scanner-tag>
set -euo pipefail

CHART_TYPE="$1"   # "central" or "sensor"
OUTPUT_DIR="$2"
MAIN_TAG="$3"
SCANNER_TAG="${4:-}"
REGISTRY="${5:-quay.io/stackrox-io}"

case "$CHART_TYPE" in
  central) TEMPLATE_DIR="image/templates/helm/stackrox-central" ;;
  sensor)  TEMPLATE_DIR="image/templates/helm/stackrox-secured-cluster" ;;
  *) echo "Usage: $0 <central|sensor> <output-dir> <main-tag> [scanner-tag] [registry]"; exit 1 ;;
esac

# Copy the template directory
cp -r "$TEMPLATE_DIR" "$OUTPUT_DIR"

# Process .htpl files — expand template variables and rename
find "$OUTPUT_DIR" -name "*.htpl" | while read -r htpl; do
  out="${htpl%.htpl}"
  sed \
    -e "s|\[< required \"\" \.ImageTag >\]|${MAIN_TAG}|g" \
    -e "s|\[< required \"\" \.CentralDBImageTag >\]|${MAIN_TAG}|g" \
    -e "s|\[< required \"\" \.ScannerImageTag >\]|${SCANNER_TAG}|g" \
    -e "s|\[< required \"\" \.ScannerV4ImageTag >\]|${MAIN_TAG}|g" \
    -e "s|\[< required \"\" \.MainRegistry >\]|${REGISTRY}|g" \
    -e "s|\[< required \"\" \.ImageRemote >\]|main|g" \
    -e "s|\[< required \"\" \.CentralDBImageRemote >\]|central-db|g" \
    -e "s|\[< required \"\" \.ScannerImageRemote >\]|scanner|g" \
    -e "s|\[< required \"\" \.ScannerDBImageRemote >\]|scanner-db|g" \
    -e "s|\[< required \"\" \.ScannerV4ImageRemote >\]|scanner-v4|g" \
    -e "s|\[< required \"\" \.ScannerV4DBImageRemote >\]|scanner-v4-db|g" \
    -e "s|\[< required \"\" \.Versions\.ChartVersion >\]|400.0.0|g" \
    -e "s|\[< required \"\" \.Versions\.MainVersion >\]|${MAIN_TAG}|g" \
    -e "s|\[< required \"\" \.ChartRepo\.URL >\]|https://charts.stackrox.io|g" \
    -e "s|\[< required \"\" \.ChartRepo\.IconURL >\]|https://raw.githubusercontent.com/stackrox/stackrox/master/image/templates/helm/shared/assets/StackRox_icon.png|g" \
    -e "s|\[< \.ImagePullSecrets\.AllowNone >\]|true|g" \
    -e "s|\[< \.EnablePodSecurityPolicies >\]|false|g" \
    -e "s|\[< \.TelemetryEnabled >\]|false|g" \
    -e "s|\[< \.TelemetryEndpoint >\]||g" \
    -e "s|\[< \.TelemetryKey >\]||g" \
    -e "s|\[< \.RenderMode >\]||g" \
    -e '/\[< if \.KubectlOutput >\]/,/\[< end >\]/d' \
    -e '/\[< if not \.Operator >\]/d' \
    -e '/\[< end >\]/d' \
    -e '/\[< if eq \.RenderMode/d' \
    -e '/\[< if ne \.RenderMode/d' \
    -e '/\[< else if eq \.RenderMode/d' \
    "$htpl" > "$out"
  rm "$htpl"
done

echo "Chart rendered: $OUTPUT_DIR (tag=$MAIN_TAG, scanner=$SCANNER_TAG)"

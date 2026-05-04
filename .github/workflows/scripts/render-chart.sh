#!/usr/bin/env bash
# Render a StackRox helm chart from the in-repo .htpl templates.
# Replaces roxctl helm output — no external tools needed.
#
# Usage: ./render-chart.sh <central|sensor> <output-dir> <main-tag> <scanner-tag> [registry]
set -euo pipefail

CHART_TYPE="$1"
OUTPUT_DIR="$2"
MAIN_TAG="$3"
SCANNER_TAG="${4:-}"
REGISTRY="${5:-quay.io/stackrox-io}"

case "$CHART_TYPE" in
  central) TEMPLATE_DIR="image/templates/helm/stackrox-central" ;;
  sensor)  TEMPLATE_DIR="image/templates/helm/stackrox-secured-cluster" ;;
  *) echo "Usage: $0 <central|sensor> <output-dir> <main-tag> [scanner-tag] [registry]"; exit 1 ;;
esac

SHARED_DIR="image/templates/helm/shared"

# Copy chart + merge shared templates
cp -r "$TEMPLATE_DIR" "$OUTPUT_DIR"
for subdir in templates internal assets config-templates; do
  if [ -d "$SHARED_DIR/$subdir" ]; then
    mkdir -p "$OUTPUT_DIR/$subdir"
    cp -rn "$SHARED_DIR/$subdir"/* "$OUTPUT_DIR/$subdir/" 2>/dev/null || true
  fi
done

# Process .htpl files — substitute values and resolve conditionals
find "$OUTPUT_DIR" -name "*.htpl" | while read -r htpl; do
  out="${htpl%.htpl}"

  # Substitute all template variables. Use env vars + perl for clean escaping.
  MAIN_TAG="$MAIN_TAG" SCANNER_TAG="$SCANNER_TAG" REGISTRY="$REGISTRY" \
  perl -pe '
    s/\[<-?\s*required\s+""\s+\.ImageTag\s*-?>\]/$ENV{MAIN_TAG}/g;
    s/\[<-?\s*required\s+""\s+\.CentralDBImageTag\s*-?>\]/$ENV{MAIN_TAG}/g;
    s/\[<-?\s*required\s+""\s+\.ScannerImageTag\s*-?>\]/$ENV{SCANNER_TAG}/g;
    s/\[<-?\s*required\s+""\s+\.ScannerV4ImageTag\s*-?>\]/$ENV{MAIN_TAG}/g;
    s/\[<-?\s*required\s+""\s+\.MainRegistry\s*-?>\]/$ENV{REGISTRY}/g;
    s/\[<-?\s*required\s+""\s+\.ImageRemote\s*-?>\]/main/g;
    s/\[<-?\s*required\s+""\s+\.CentralDBImageRemote\s*-?>\]/central-db/g;
    s/\[<-?\s*required\s+""\s+\.ScannerImageRemote\s*-?>\]/scanner/g;
    s/\[<-?\s*required\s+""\s+\.ScannerDBImageRemote\s*-?>\]/scanner-db/g;
    s/\[<-?\s*required\s+""\s+\.ScannerV4ImageRemote\s*-?>\]/scanner-v4/g;
    s/\[<-?\s*required\s+""\s+\.ScannerV4DBImageRemote\s*-?>\]/scanner-v4-db/g;
    s/\[<-?\s*required\s+""\s+[^>]*Versions\.ChartVersion[^>]*>\]/400.0.0/g;
    s/\[<-?\s*required\s+""\s+[^>]*Versions\.MainVersion[^>]*>\]/$ENV{MAIN_TAG}/g;
    s/\[<-?\s*required\s+""\s+\.ChartRepo\.URL\s*-?>\]/https:\/\/charts.stackrox.io/g;
    s/\[<-?\s*required\s+""\s+\.ChartRepo\.IconURL\s*-?>\]/https:\/\/raw.githubusercontent.com\/stackrox\/stackrox\/master\/image\/templates\/helm\/shared\/assets\/StackRox_icon.png/g;
    s/\[<-?\s*\.ImagePullSecrets\.AllowNone\s*-?>\]/true/g;
    s/\[<-?\s*\.EnablePodSecurityPolicies\s*-?>\]/false/g;
    s/\[<-?\s*\.TelemetryEnabled\s*-?>\]/false/g;
    s/\[<-?\s*\.TelemetryEndpoint\s*-?>\]//g;
    s/\[<-?\s*\.TelemetryKey\s*-?>\]//g;
    s/\[<-?\s*\.RenderMode\s*-?>\]//g;
  ' "$htpl" > "$out"

  # Resolve conditionals for our CI context:
  # - AutoSensePodSecurityPolicies: false → remove the if block content
  # - KubectlOutput: false → keep the "not KubectlOutput" branch
  # - Operator: false → keep the "not Operator" branch
  # - RenderMode != "scannerOnly" → keep content
  # Strategy: remove the [<...>] directive lines, keep the content between
  # them. For false conditions, remove the if/end block. For true conditions,
  # keep the content.
  perl -i -ne '
    # Skip lines that are ONLY htpl directives (if/else/end/range)
    next if /^\s*\[<-?\s*(if|else|end|range)\b/;
    # Remove inline [< >] markers but keep the rest of the line
    s/\[<-?\s*[^>]*?\s*-?>\]//g;
    # Skip lines that became empty name/value pairs from range removal
    next if /^\s*-\s*name:\s*$/;
    next if /^\s*value:\s*$/;
    print;
  ' "$out"

  rm "$htpl"
done

# Simplify NOTES.txt — original calls srox.init which has strict validation
echo "StackRox CI deployment" > "$OUTPUT_DIR/templates/NOTES.txt"

echo "Chart rendered: $OUTPUT_DIR (tag=$MAIN_TAG, scanner=$SCANNER_TAG)"

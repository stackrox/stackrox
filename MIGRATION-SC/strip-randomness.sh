#!/bin/bash
set -euo pipefail

# Strips randomized values from roxctl sensor generate output directories.
# Randomized content includes: TLS certificates and private keys (PEM files
# and base64-encoded in Secret YAMLs), and the cluster name in sensor.yaml
# and NOTES.txt.
#
# After processing, two runs of the same command with the same options
# should produce byte-for-byte identical output.

dir="$1"
if [ -z "$dir" ] || [ ! -d "$dir" ]; then
    echo "Usage: $0 <output-directory>" >&2
    exit 1
fi

# Remove helm subdirectory if present.
rm -rf "$dir/helm"

# Replace PEM files with deterministic content.
for f in "$dir"/*.pem; do
    [ -f "$f" ] || continue
    echo "REDACTED" > "$f"
done

# Replace cluster name in sensor.yaml and NOTES.txt with a placeholder.
for f in "$dir/sensor.yaml" "$dir/NOTES.txt"; do
    [ -f "$f" ] || continue
    perl -i -pe '
        if ($prev_is_cluster_name) {
            s/^(\s+)\S.*$/${1}REDACTED_CLUSTER_NAME/;
            $prev_is_cluster_name = 0;
        }
        $prev_is_cluster_name = 1 if /^\s+cluster-name:\s*\|/;
    ' "$f"
    # NOTES.txt has "Name:  <whitespace> <cluster-name>" format
    perl -i -pe 's/^(\s+Name:\s+)\S+.*$/${1}REDACTED_CLUSTER_NAME/' "$f"
done

# Process secret YAML files: replace base64-encoded cert/key data.
for f in "$dir"/*-secret.yaml; do
    [ -f "$f" ] || continue
    perl -i -0777 -pe '
        sub indent_of {
            my ($block) = @_;
            my @lines = split(/\n/, $block);
            return 4 unless @lines && $lines[0] =~ /^(\s*)\S/;
            return length($1);
        }

        s/(-----BEGIN CERTIFICATE-----\n)(.*?)(\n[ \t]*-----END CERTIFICATE-----)/
            $1 . (" " x indent_of($2)) . "REDACTED_CERTIFICATE\n" . $3/ges;

        s/(-----BEGIN EC PRIVATE KEY-----\n)(.*?)(\n[ \t]*-----END EC PRIVATE KEY-----)/
            $1 . (" " x indent_of($2)) . "REDACTED_EC_PRIVATE_KEY\n" . $3/ges;

        s/(-----BEGIN RSA PRIVATE KEY-----\n)(.*?)(\n[ \t]*-----END RSA PRIVATE KEY-----)/
            $1 . (" " x indent_of($2)) . "REDACTED_RSA_PRIVATE_KEY\n" . $3/ges;
    ' "$f"
done

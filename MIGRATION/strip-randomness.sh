#!/bin/bash
set -euo pipefail

# Strips randomized values from roxctl central generate output directories.
# Randomized content includes: TLS certificates and private keys, admin
# password (bcrypt hash and plaintext), DB passwords, JWT signing keys,
# and the random suffix in the generated-values secret name.
#
# After processing, two runs of the same command with the same options
# should produce byte-for-byte identical output.

dir="$1"
if [ -z "$dir" ] || [ ! -d "$dir" ]; then
    echo "Usage: $0 <output-directory>" >&2
    exit 1
fi

# Remove the helm subdirectory — we are only interested in the kubectl output.
rm -rf "$dir/helm"

# Replace password file with deterministic content.
if [ -f "$dir/password" ]; then
    echo "REDACTED_PASSWORD" > "$dir/password"
fi

# Process all YAML files.
find "$dir" -type f \( -name '*.yaml' -o -name '*.yml' \) -print0 | while IFS= read -r -d '' file; do
    # Multi-line replacements: PEM certificate and key blocks.
    # Replaces the base64 body between BEGIN/END markers with a single
    # placeholder line, preserving the original indentation.
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
    ' "$file"

    # Single-line, context-aware replacements.
    perl -i -pe '
        # Replace bcrypt password hashes (admin htpasswd).
        s/(admin:)\$2a\$\d+\$[A-Za-z0-9.\/]+/$1REDACTED_BCRYPT/;

        # Replace stackrox-generated-XXXXXX random suffix.
        s/stackrox-generated-[a-z0-9]{4,}/stackrox-generated-REDACTED/g;

        # State machine for password values. Two patterns:
        #
        # Pattern A — YAML mapping with a "value:" child:
        #   password:
        #     value: <random>
        #
        # Pattern B — YAML block scalar:
        #   password: |
        #     <random>

        if ($expect_password_value_key) {
            s/^(\s+value:\s+)\S+[ \t]*$/${1}REDACTED_PASSWORD/;
            $expect_password_value_key = 0;
        }

        if ($expect_password_bare_value) {
            s/^(\s+)\S+[ \t]*$/${1}REDACTED_PASSWORD/;
            $expect_password_bare_value = 0;
        }

        # "password:" with no inline value — next line has "value: <random>"
        $expect_password_value_key = 1 if /^\s+password:\s*$/;

        # "password: |" — next line is the bare password string
        $expect_password_bare_value = 1 if /^\s+password:\s*\|/;
    ' "$file"
done

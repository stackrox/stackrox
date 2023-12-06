#! /bin/bash

source "scripts/ci/lib.sh"

GOVULNCHECK_BIN="$(make  which-govulncheck --silent)"
find bin/linux_amd64 -type f | while read -r file;
do
    echo "Analyzing binary $file"
    $GOVULNCHECK_BIN -mode=binary --json "$file" > vulns.json
    go run govulncheck/main.go vulns.json > filtered_vulns.json
    cat filtered_vulns.json
    jq '.data[] | select(.id) | [.id, .summary, .details] | @tsv' -r filtered_vulns.json | sort -u | while IFS=$'\t' read -r -a vulns
    do
      id="${vulns[0]}"
      summary="${vulns[1]}"
      details="${vulns[2]}"
      echo "$id" "$summary"
      save_junit_failure "$id" "$summary" "$details"
    done
done

if [[ $(jq '.data[] | length' -r filtered_vulns.json) != 0 ]]; then
    jq '.data[] | length' -r filtered_vulns.json
    echo "Found vulnerabilities. If they are false positives, add them to govulncheck-allowlist.json" >&2
    cat filtered_vulns.json
    exit 1
fi

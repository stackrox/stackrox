# Image CVE List REST — local proof

This VM cannot run full ACS (no Docker/Podman, no Kubernetes, no Central).
Proof on this branch is programmatic: generated proto stubs compile, and
`go test ./central/cve/image/service` asserts List/Count query filtering plus
field parity vs GraphQL `imageCVEs` / `getImageCVEList`.

## RPCs

| RPC | HTTP | Request | Response |
| --- | --- | --- | --- |
| `ImageCVEService.ListImageCVEs` | `GET /v1/imagecves` | `query`, `pagination.*`, optional `requestStatuses` | `{ "imageCves": [ ImageCVE ] }` |
| `ImageCVEService.CountImageCVEs` | `GET /v1/imagecvescount` | `RawQuery` (`query`) | `{ "count": N }` |

Wrapped GraphQL: `ImageCVEs` / `ImageCVECount` in
`central/graphql/resolvers/image_cve_core.go` (`ImageCVEView.Get` / `.Count`),
plus `ImageCVECore.distroTuples` and `exceptionCount`.

## Deploy Central locally

From a machine with a cluster (see `deploy/AGENTS.md`):

```bash
# optional: skip UI for this Central-only API
export SKIP_UI_BUILD=1
./deploy/deploy-local.sh

# or: roxie deploy
```

Wait until Central is Ready, then port-forward if needed:

```bash
kubectl -n stackrox port-forward svc/central 8443:443
```

Default local admin password is in the deploy output / `deploy/k8s/central-deploy`.

## curl

```bash
# Replace PASSWORD. Query language is the same RawQuery string GraphQL uses.
export ROX_URL="https://localhost:8443"
export ROX_PASSWORD='<admin password>'

# List — Workload CVE overview fields (CVE, severity counts, CVSS, EPSS, distro, exceptionCount)
curl -sk -u "admin:${ROX_PASSWORD}" \
  --get "${ROX_URL}/v1/imagecves" \
  --data-urlencode 'query=CVE:CVE-2021-44228' \
  --data-urlencode 'pagination.limit=20' \
  --data-urlencode 'pagination.offset=0' \
  --data-urlencode 'pagination.sortOption.field=CVE' \
  --data-urlencode 'requestStatuses=PENDING'

# Count
curl -sk -u "admin:${ROX_PASSWORD}" \
  --get "${ROX_URL}/v1/imagecvescount" \
  --data-urlencode 'query=CVE:CVE-2021-44228'
```

Example list item (JSON names from proto `json_names_for_fields`):

```json
{
  "imageCves": [
    {
      "cve": "CVE-2021-44228",
      "topCvss": 9.8,
      "topNvdCvss": 10,
      "affectedImageCount": 4,
      "affectedImageCountBySeverity": {
        "critical": 4,
        "important": 0,
        "moderate": 0,
        "low": 0,
        "unknown": 0
      },
      "firstDiscoveredInSystem": "2021-12-01T00:00:00Z",
      "publishedOn": "2021-11-26T00:00:00Z",
      "distroTuples": [
        {
          "summary": "Log4Shell",
          "operatingSystem": "debian:11",
          "cvss": 9.8,
          "scoreVersion": "V3",
          "nvdCvss": 10,
          "nvdScoreVersion": "V3",
          "epssProbability": 0.97,
          "knownRansomwareCampaignUse": "Known",
          "severity": "CRITICAL_VULNERABILITY_SEVERITY"
        }
      ],
      "exceptionCount": 1
    }
  ]
}
```

`requestStatuses` matches GraphQL `exceptionCount(requestStatus)` /
`statusesForExceptionCount` (`PENDING` on the Observed tab,
`APPROVED_PENDING_UPDATE` on exception tabs).

## Compare to GraphQL

```bash
curl -sk -u "admin:${ROX_PASSWORD}" \
  -H 'Content-Type: application/json' \
  -d '{"query":"query { imageCVEs(query: \"CVE:CVE-2021-44228\") { cve topCVSS topNvdCVSS affectedImageCount exceptionCount(requestStatus: [\"PENDING\"]) distroTuples { summary operatingSystem cvss epss: cveBaseInfo { epss { epssProbability } } } } }"}' \
  "${ROX_URL}/api/graphql"
```

REST `imageCves[]` should match GraphQL `imageCVEs[]` for those fields.

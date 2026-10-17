# Skill: Am I Affected?

Detect whether a given ACS-monitored environment is affected by a specific vulnerability.

Answer questions like:
- "Is CVE-2021-44228 affecting me?"
- "Is GHSA-5wvp-7f3h-6wmm present in any deployments in namespace 'payments' on cluster 'prod'?"
- "Show me all deployments affected by RHSA-2024:1234 in cluster 'staging'"

---

## Inputs

| Input | Required | Description |
|-------|----------|-------------|
| Vulnerability ID | Yes | A recognized vulnerability identifier (see format list below) |
| ACS Central endpoint | Yes | URL of the ACS Central instance |
| API token | Yes | ROX API token for authentication |
| Cluster(s) | No | Narrow scope to specific cluster(s) |
| Namespace(s) | No | Narrow scope to specific namespace(s) |

---

## Step 1: Input Validation

This step can be run by a separate agent. Validate all inputs before proceeding.

### 1.1 Vulnerability ID Format

The vulnerability ID must match one of the following recognized formats:

| Prefix | Format | Example | Source |
|--------|--------|---------|--------|
| CVE | `CVE-YYYY-NNNNN+` | CVE-2021-44228 | MITRE / NVD |
| GHSA | `GHSA-XXXX-XXXX-XXXX` (chars: `2-9cfghjmpqrvwx`) | GHSA-5wvp-7f3h-6wmm | GitHub Security Advisories |
| RHSA | `RHSA-YYYY:NNNNN+` | RHSA-2024:1234 | Red Hat Security Advisory |
| RHBA | `RHBA-YYYY:NNNNN+` | RHBA-2024:5678 | Red Hat Bugfix Advisory |
| RHEA | `RHEA-YYYY:NNNNN+` | RHEA-2024:9012 | Red Hat Enhancement Advisory |
| ALAS | `ALAS[N]*-YYYY-NNNNN+` | ALAS2-2024-001 | Amazon Linux Security Advisory |
| DSA | `DSA-YYYY-NNNNN+` | DSA-2024-1234 | Debian Security Advisory |

**Validation regex patterns** (derived from StackRox scanner code in `pkg/scannerv4/mappers/mappers.go`):

```
CVE:    CVE-\d{4}-\d+
GHSA:   GHSA(-[2-9cfghjmpqrvwx]{4}){3}
RHSA:   (RHSA|RHBA|RHEA)-\d{4}:\d+
ALAS:   ALAS\d*-\d{4}-\d+
DSA:    DSA-\d{4}-\d+
Generic fallback: [A-Z]+-\d{4}[-:]\d+
```

If the ID does not match any known format, warn the user but allow proceeding (it may be a format not listed here).

**Internet check (optional):** If web access is available, verify the vulnerability ID exists by querying an appropriate source:
- CVE: `https://cveawg.mitre.org/api/cve/<ID>`
- GHSA: `https://github.com/advisories/<ID>`
- RHSA/RHBA/RHEA: `https://access.redhat.com/errata/<ID>`
- OSV-indexed IDs: `https://api.osv.dev/v1/vulns/<ID>`

### 1.2 API Token Validation

ROX API tokens are **JWTs (JSON Web Tokens)** signed with RS256. The token consists of three Base64url-encoded parts separated by dots: `header.payload.signature`.

**Validation steps:**

1. Check the token has three dot-separated parts (basic JWT structure).
2. Decode the header (first part, Base64url) and verify it contains `"alg"` (expected: `RS256`).
3. Decode the payload (second part, Base64url) and verify it contains expected ROX claims:
   - `iss` (issuer) should be present
   - `aud` (audience) should be present
   - `jti` (JWT ID) should be a UUID
   - `exp` (expiration) if present, must not be in the past
   - Optional ROX-specific claims: `name`, `roles` or `role`

**Quick structural check (command line):**

```bash
# Split token and decode header
echo "$ROX_API_TOKEN" | cut -d. -f1 | base64 -d 2>/dev/null | jq .

# Split token and decode payload (check claims)
echo "$ROX_API_TOKEN" | cut -d. -f2 | base64 -d 2>/dev/null | jq .
```

If the token is clearly not a JWT (no dots, not Base64-decodable), reject immediately with a clear error.

### 1.3 Endpoint URL Sanitization

1. **Trim whitespace** from the URL.
2. **Trim trailing slash(es)** — e.g. `https://central.example.com/` becomes `https://central.example.com`.
3. **Ensure scheme is present** — if no `://` is found, prepend `https://`. Always prefer `https://`.
4. **Strip path components** if the user provided a full URL with a path (e.g. `https://central.example.com/v1/ping` should become `https://central.example.com`).

### 1.4 Connectivity Check

Verify the ACS Central instance is reachable using an **unauthenticated** endpoint:

```bash
# Preferred: lightweight ping
curl -sk "${ROX_ENDPOINT}/v1/ping"
# Expected response: {"status":"ok"}
```

The `/v1/ping` endpoint is unauthenticated, lightweight, and returns `{"status":"ok"}` if Central is reachable.

**Alternative:** `/v1/config/public` (also unauthenticated, returns login banners and telemetry config).

Then verify the **token works** with an authenticated endpoint:

```bash
# Verify token authentication
curl -sk -H "Authorization: Bearer ${ROX_API_TOKEN}" "${ROX_ENDPOINT}/v1/metadata"
# Expected: JSON with "version", "buildFlavor", "releaseBuild" fields
```

If `/v1/metadata` returns version information, the token is valid and authenticated.

---

## Step 2: Query ACS Central for Affected Resources

After validation, query three areas in parallel. Each area can be handled by a separate agent.

### 2.1 Check Deployments (via VulnMgmt Export)

**Best endpoint:** `/v1/export/vuln-mgmt/workloads` (streaming)

This is the **recommended** endpoint because it returns deployments together with their full image objects, including all vulnerability data. This avoids needing to make separate calls to fetch image details.

**Why not ListDeployments or ExportDeployments?**
- `ListDeployments` (`/v1/deployments`) returns only metadata (name, namespace, cluster) — no vulnerability data.
- `ExportDeployments` (`/v1/export/deployments`) returns full deployment objects but images contain only references, not vulnerability details.
- `VulnMgmtExportWorkloads` returns deployments + full images with embedded CVE data in a single streaming call.

**Build the query:**

The query uses the ACS search syntax: `Field:value+Field:value`. The `CVE` search field can be used to filter directly by vulnerability ID.

```bash
# Basic: find all workloads affected by a specific CVE
VULN_ID="CVE-2021-44228"
QUERY="CVE:${VULN_ID}"

# With cluster scope
QUERY="CVE:${VULN_ID}+Cluster:prod-cluster"

# With cluster and namespace scope
QUERY="CVE:${VULN_ID}+Cluster:prod-cluster+Namespace:payments"

# URL-encode the query for HTTP
ENCODED_QUERY=$(python3 -c "import urllib.parse; print(urllib.parse.quote('${QUERY}', safe=''))")

curl -sk \
  -H "Authorization: Bearer ${ROX_API_TOKEN}" \
  "${ROX_ENDPOINT}/v1/export/vuln-mgmt/workloads?query=${ENCODED_QUERY}"
```

**Response format** (newline-delimited JSON, one object per line):

```json
{"result":{"deployment":{"id":"...","name":"my-app","namespace":"default","clusterId":"...","clusterName":"prod",...},"images":[{"id":"...","name":{"fullName":"registry.example.com/my-app:v1.2"},"scan":{"components":[{"name":"log4j-core","version":"2.14.1","vulns":[{"cve":"CVE-2021-44228","cvss":10.0,"severity":"CRITICAL_VULNERABILITY_SEVERITY","summary":"...","fixedBy":"2.17.1","link":"https://nvd.nist.gov/vuln/detail/CVE-2021-44228"}]}]}}]}}
```

**Parsing the response:**

For each streamed result:
1. Extract `result.deployment.name`, `result.deployment.namespace`, `result.deployment.clusterName`
2. For each image in `result.images`:
   - Walk `image.scan.components[]`
   - For each component, walk `component.vulns[]` (or `component.vulnerabilities[]`)
   - Match where `vuln.cve` equals the target vulnerability ID
   - Record: component name, component version, fixed-by version, CVSS score, severity

```bash
# Parse with jq — extract affected components from the streaming response
curl -sk \
  -H "Authorization: Bearer ${ROX_API_TOKEN}" \
  "${ROX_ENDPOINT}/v1/export/vuln-mgmt/workloads?query=${ENCODED_QUERY}" \
  | jq -c '
    .result |
    {
      deployment: .deployment.name,
      namespace: .deployment.namespace,
      cluster: .deployment.clusterName,
      affected_components: [
        .images[].scan.components[] |
        select(.vulns[]?.cve == "'"${VULN_ID}"'") |
        {
          component: .name,
          version: .version,
          vulns: [.vulns[] | select(.cve == "'"${VULN_ID}"'") | {
            cve: .cve,
            cvss: .cvss,
            severity: .severity,
            fixedBy: .fixedBy,
            link: .link
          }]
        }
      ]
    } | select(.affected_components | length > 0)
  '
```

### 2.2 Check Nodes

**Endpoint:** `/v1/export/nodes` (streaming)

Nodes contain vulnerability data directly in their scan results. Each node scan includes components with embedded vulnerabilities.

**Build the query:**

```bash
VULN_ID="CVE-2021-44228"
QUERY="CVE:${VULN_ID}"

# With cluster scope
QUERY="CVE:${VULN_ID}+Cluster:prod-cluster"

ENCODED_QUERY=$(python3 -c "import urllib.parse; print(urllib.parse.quote('${QUERY}', safe=''))")

curl -sk \
  -H "Authorization: Bearer ${ROX_API_TOKEN}" \
  "${ROX_ENDPOINT}/v1/export/nodes?query=${ENCODED_QUERY}"
```

**Response format** (newline-delimited JSON):

```json
{"result":{"node":{"id":"...","name":"worker-1","clusterId":"...","clusterName":"prod","scan":{"components":[{"name":"openssl","version":"1.1.1k","vulns":[{"cve":"CVE-2021-44228","cvss":10.0,"severity":"CRITICAL_VULNERABILITY_SEVERITY","fixedBy":"..."}],"vulnerabilities":[...]}]}}}}
```

**Parsing the response:**

```bash
curl -sk \
  -H "Authorization: Bearer ${ROX_API_TOKEN}" \
  "${ROX_ENDPOINT}/v1/export/nodes?query=${ENCODED_QUERY}" \
  | jq -c '
    .result.node |
    {
      node: .name,
      cluster: .clusterName,
      osImage: .osImage,
      kernelVersion: .kernelVersion,
      affected_components: [
        .scan.components[] |
        select(.vulns[]?.cve == "'"${VULN_ID}"'" or .vulnerabilities[]?.cve == "'"${VULN_ID}"'") |
        {
          component: .name,
          version: .version,
          vulns: [(.vulns // [])[] | select(.cve == "'"${VULN_ID}"'") | {
            cve: .cve,
            cvss: .cvss,
            severity: .severity,
            fixedBy: .fixedBy
          }]
        }
      ]
    } | select(.affected_components | length > 0)
  '
```

### 2.3 Check Cluster-Level (Platform) Vulnerabilities

**Endpoint:** `/v1/clusters` does **not** contain vulnerability data — it returns cluster metadata only (name, health status, sensor version, etc.).

Platform-level vulnerabilities (Kubernetes, Istio, OpenShift CVEs) are tracked separately. To query them, use the **search API** with the `CLUSTER_VULNERABILITIES` category.

```bash
VULN_ID="CVE-2021-44228"

curl -sk \
  -H "Authorization: Bearer ${ROX_API_TOKEN}" \
  "${ROX_ENDPOINT}/v1/search?query=CVE:${VULN_ID}&categories=CLUSTER_VULNERABILITIES"
```

**Response:**

```json
{
  "results": [
    {
      "id": "...",
      "name": "CVE-2021-44228",
      "category": "CLUSTER_VULNERABILITIES",
      "fieldToMatches": {...}
    }
  ]
}
```

If results are returned, the vulnerability affects the cluster platform itself (Kubernetes, OpenShift, Istio components). Extract the cluster name from the result's `fieldToMatches` or make a follow-up query to get details.

**Alternative approach** — query cluster CVEs directly:

```bash
# List cluster-level CVEs matching the ID
curl -sk \
  -H "Authorization: Bearer ${ROX_API_TOKEN}" \
  "${ROX_ENDPOINT}/v2/cluster-cves?query=CVE:${VULN_ID}"
```

---

## Step 3: Analyze and Report Results

After all three parallel queries complete, consolidate the findings.

### Result Classification

Count total findings across all three areas (deployments/images, nodes, cluster platform).

### Reporting Rules

**If no matches found:**
- State clearly: "The vulnerability `<ID>` was not found in any monitored deployments, nodes, or cluster platforms."
- Note caveats: the vulnerability might exist in unscanned images, recently deployed workloads not yet scanned, or components outside ACS monitoring scope.

**If 1-5 findings:**
- List every finding with full details.

**If more than 5 findings:**
- State the total count: "Found `<N>` resources affected by `<ID>`."
- Pick the most impactful findings to illustrate (prefer higher CVSS, CRITICAL severity, production clusters, or workloads with more live pods).
- Show up to 5 representative findings.
- Summarize the rest (e.g., "Additionally, 12 other deployments in namespace 'backend' on cluster 'prod' are affected").

### Finding Detail Format

For each finding, present:

**Deployment findings:**
| Field | Value |
|-------|-------|
| Cluster | `<cluster name>` |
| Namespace | `<namespace>` |
| Deployment | `<deployment name>` |
| Image | `<full image name>` |
| Component | `<component name>` @ `<version>` |
| CVE | `<vuln ID>` |
| CVSS | `<score>` |
| Severity | `<severity>` |
| Fixed By | `<fixed version>` (or "No fix available") |
| Link | `<reference URL>` |

**Node findings:**
| Field | Value |
|-------|-------|
| Cluster | `<cluster name>` |
| Node | `<node name>` |
| OS | `<os image>` |
| Kernel | `<kernel version>` |
| Component | `<component name>` @ `<version>` |
| CVE | `<vuln ID>` |
| CVSS | `<score>` |
| Severity | `<severity>` |
| Fixed By | `<fixed version>` (or "No fix available") |

**Cluster platform findings:**
| Field | Value |
|-------|-------|
| Cluster | `<cluster name>` |
| CVE Type | `<K8S_CVE / ISTIO_CVE / OPENSHIFT_CVE>` |
| CVE | `<vuln ID>` |
| CVSS | `<score>` |
| Severity | `<severity>` |
| Fixed By | `<fixed version>` (or "No fix available") |

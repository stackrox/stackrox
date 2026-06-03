---
name: amplitude-telemetry-report
description: Generate a concise ACS telemetry report from Amplitude data. Use when the user asks for a telemetry report, usage summary, feature adoption metrics, weekly/monthly analytics report, or wants to understand how ACS features are being used. Also use when asked to check telemetry trends, compare periods, or identify changes in product usage patterns.
---

# ACS Telemetry Report Generator

Generate a concise telemetry report by querying Amplitude via MCP tools. The report summarizes fleet health, feature usage trends, and significant changes — designed for product managers and engineers.

## Parameters

When invoked, check the user's message for these parameters. Use defaults if not specified.

| Parameter | Default | Options |
|-----------|---------|---------|
| Period | `monthly` | `weekly` (7 days), `monthly` (30 days) |
| Project ID | `418843` | Any valid Amplitude project ID |

## Workflow

Follow these steps in order. Run queries in parallel where possible to minimize latency.

### Step 1: Setup

1. Call `get_context` to verify project access.
2. Compute date ranges using **only completed intervals** — never include the current partial day/week/month. This ensures current and previous periods are the same length and directly comparable.
   - **Weekly:** current period = the 7 days ending yesterday (e.g., if today is Wednesday Jun 4, use May 28–Jun 3). Previous period = the 7 days before that (May 21–May 27).
   - **Monthly:** current period = the 30 days ending yesterday. Previous period = the 30 days before that.
   - Both `start` and `end` are inclusive dates at midnight UTC. Use ISO 8601 format (e.g., `2026-05-28T00:00:00Z`).
   - **Never use today's date as the end of a period** — today is incomplete and would undercount.
3. Call `get_chart_definition_params` with `chartType: "eventsSegmentation"` to confirm the query schema.

### Step 2: Query Current Period

Run these `query_dataset` calls. Use `type: "eventsSegmentation"` and `app: "<projectId>"` for all. Set the time range using `start` and `end` parameters with the current period dates.

Queries are grouped by feature domain. All groups within a step can run in parallel.

---

#### Fleet Health

**Query 1 — Active Fleet (uniques):**

```json
{
  "type": "eventsSegmentation",
  "app": "418843",
  "params": {
    "start": "<current_start>",
    "end": "<current_end>",
    "interval": 0,
    "metric": "uniques",
    "events": [
      {"event_type": "Updated Central Identity", "filters": [], "group_by": []},
      {"event_type": "Updated Secured Cluster Identity", "filters": [], "group_by": []}
    ],
    "segments": [{"conditions": []}]
  }
}
```

Read the `overallSeries` values — do NOT sum timeSeries (uniques are non-additive).

**Query 2 — Version Distribution (uniques, grouped):**

Identifies which ACS release versions are active in the fleet by grouping `Updated Central Identity` by the `Central version` user property.

```json
{
  "type": "eventsSegmentation",
  "app": "418843",
  "params": {
    "start": "<current_start>",
    "end": "<current_end>",
    "interval": 0,
    "metric": "uniques",
    "events": [
      {"event_type": "Updated Central Identity", "filters": [], "group_by": []}
    ],
    "segments": [{"conditions": []}],
    "groupBy": [{"type": "user", "value": "gp:Central version", "group_type": "User"}]
  }
}
```

Use `groupByLimit: 30` to capture the long tail of versions. The user property requires the `gp:` prefix (i.e., `gp:Central version`).

**Query 3 — Page Views by Feature Area (totals, grouped):**

```json
{
  "type": "eventsSegmentation",
  "app": "418843",
  "params": {
    "start": "<current_start>",
    "end": "<current_end>",
    "interval": 0,
    "metric": "totals",
    "events": [
      {
        "event_type": "Page Viewed",
        "filters": [],
        "group_by": [{"type": "event", "value": "path"}]
      }
    ],
    "segments": [{"conditions": []}]
  }
}
```

Use `groupByLimit: 20` to get the top 20 paths.

---

#### Vulnerability Management

**Query 4 — Vulnerability Management (totals, 9 events):**

All CVE scanning, vulnerability reporting, and SBOM events in a single query:

- `Workload CVE Filter Applied`
- `Platform CVE Filter Applied`
- `Platform CVE Entity Context View`
- `Node CVE Filter Applied`
- `Node CVE Entity Context View`
- `Vulnerability Report Created`
- `Vulnerability Report Download Generated`
- `View Based Report Generated`
- `Image SBOM Generated`

---

#### Cluster Onboarding

**Query 5 — Cluster Onboarding (totals, 7 events):**

The full cluster onboarding funnel — from init bundle creation through cluster registration:

- `Create Init Bundle Clicked`
- `Download Init Bundle`
- `Create Cluster Registration Secret Clicked`
- `Secure a Cluster Link Clicked`
- `CRS Secure a Cluster Link Clicked`
- `Secured Cluster Registered`
- `Secured Cluster Initialized`

---

#### Compliance & Collections

**Query 6 — Compliance & Collections (totals, 3 events):**

- `Compliance Report Download Generation Triggered`
- `Compliance Schedules Wizard Save Clicked`
- `Collection Created`

---

#### OCP Console Plugin

Tracks adoption and usage of the ACS dynamic console plugin for OpenShift. All adoption queries use the "Secured Clusters Prod" cohort segment (`gkuaq1d`) to filter to production clusters, matching the [ACS console plugin dashboard](https://app.amplitude.com/analytics/redhat/dashboard/8o0l3lmx).

**Query 7a — Plugin Adoption (uniques):**

Two events in one query — clusters with plugin installed vs. clusters actively using it:

```json
{
  "type": "eventsSegmentation",
  "app": "418843",
  "params": {
    "start": "<current_start>",
    "end": "<current_end>",
    "interval": 0,
    "metric": "uniques",
    "events": [
      {"event_type": "Internal Token Issued", "filters": [], "group_by": []},
      {
        "event_type": "Internal Token Issued",
        "filters": [{"subprop_op": "greater", "subprop_key": "Total Cluster Scopes", "subprop_type": "event", "subprop_value": ["0"]}],
        "group_by": []
      }
    ],
    "segments": [{"conditions": [{"op": "is", "prop": "userdata_cohort", "type": "property", "values": ["gkuaq1d"], "prop_type": "user", "group_type": "User"}]}]
  }
}
```

- Event 0 = clusters with plugin installed (any token issuance).
- Event 1 = clusters actively using the plugin (tokens with at least one cluster scope).

Read the `overallSeries` values — uniques are non-additive.

**Query 7b — Token Volume (totals):**

```json
{
  "type": "eventsSegmentation",
  "app": "418843",
  "params": {
    "start": "<current_start>",
    "end": "<current_end>",
    "interval": 0,
    "metric": "totals",
    "events": [
      {
        "event_type": "Internal Token Issued",
        "filters": [{"subprop_op": "greater", "subprop_key": "Total Cluster Scopes", "subprop_type": "event", "subprop_value": ["0"]}],
        "group_by": []
      },
      {
        "event_type": "Internal Token Issued",
        "filters": [{"subprop_op": "greater", "subprop_key": "Cluster Scopes With Full Access", "subprop_type": "event", "subprop_value": ["0"]}],
        "group_by": []
      }
    ],
    "segments": [{"conditions": [{"op": "is", "prop": "userdata_cohort", "type": "property", "values": ["gkuaq1d"], "prop_type": "user", "group_type": "User"}]}]
  }
}
```

- Event 0 = total internal tokens with scoped access.
- Event 1 = tokens with full cluster access ("All projects" selected).

**Query 7c — Security Tab Page Views (totals):**

Page views from the console plugin's security tab, split by page type:

```json
{
  "type": "eventsSegmentation",
  "app": "418843",
  "params": {
    "start": "<current_start>",
    "end": "<current_end>",
    "interval": 0,
    "metric": "totals",
    "events": [
      {
        "event_type": "Page Viewed",
        "filters": [
          {"subprop_op": "is", "subprop_key": "acsApplicationSource", "subprop_type": "event", "subprop_value": ["console-plugin"]},
          {"subprop_op": "contains", "subprop_key": "path", "subprop_type": "event", "subprop_value": ["/acs/security/vulnerabilities"]}
        ],
        "group_by": []
      },
      {
        "event_type": "Page Viewed",
        "filters": [
          {"subprop_op": "is", "subprop_key": "acsApplicationSource", "subprop_type": "event", "subprop_value": ["console-plugin"]},
          {"subprop_op": "contains", "subprop_key": "path", "subprop_type": "event", "subprop_value": ["/k8s/cluster"]}
        ],
        "group_by": []
      },
      {
        "event_type": "Page Viewed",
        "filters": [
          {"subprop_op": "is", "subprop_key": "acsApplicationSource", "subprop_type": "event", "subprop_value": ["console-plugin"]},
          {"subprop_op": "contains", "subprop_key": "path", "subprop_type": "event", "subprop_value": ["/k8s/ns"]}
        ],
        "group_by": []
      }
    ],
    "segments": [{"conditions": []}]
  }
}
```

- Event 0 = Vulnerabilities page views.
- Event 1 = Security tab in cluster details.
- Event 2 = Security tab in namespace/workload details.

---

#### Platform & Access

**Query 8 — Platform & Access (totals, 3 events):**

General platform activity, access management, and stability:

- `API Call`
- `Invite Users Submitted`
- `Page Crash`

---

#### Auth Providers

**Query 13 — Auth Providers:**

Shows how production Centrals are configured for external authentication. Run composition queries using `MOST_RECENT` on static Central user properties, filtered to the "Centrals Prod" cohort (`59vr7tn`). All queries use the same template as Image Integrations — only the `property.value` changes.

Run these composition queries in parallel:

| Query | User Property | Purpose |
|-------|---------------|---------|
| 13a | `gp:Auth Providers` | Primary auth provider type (string: `openshift`, `oidc`, `saml`, `userpki`, `(none)`) |
| 13b | `gp:Total Declarative Auth Providers` | Declarative (config-as-code) auth provider count |
| 13c | `gp:Total Imperative Auth Providers` | Imperative (UI/API) auth provider count |
| 13d | `gp:Total Groups of oidc` | OIDC role-mapping groups per Central |
| 13e | `gp:Total Groups of saml` | SAML role-mapping groups per Central |
| 13f | `gp:Total Groups of openshift` | OpenShift role-mapping groups per Central |
| 13g | `gp:Total Groups of userpki` | User PKI role-mapping groups per Central |
| 13h | `gp:Total Groups of iap` | IAP role-mapping groups per Central |
| 13i | `gp:Total Groups of auth0` | Auth0 role-mapping groups per Central |

Template (same as Image Integrations):

```json
{
  "type": "composition",
  "app": "418843",
  "params": {
    "range": "Last 30 Days",
    "metric": "MOST_RECENT",
    "interval": 1,
    "property": {"type": "user", "value": "<user_property>", "group_type": "User"},
    "groupBy": [],
    "segments": [{"conditions": [{"op": "is", "prop": "userdata_cohort", "type": "property", "values": ["59vr7tn"], "prop_type": "user", "group_type": "User"}]}],
    "countGroup": "User"
  }
}
```

Use `groupByLimit: 20` for all queries.

From the results, compute:

**Auth Provider Types (Query 13a):** The `Auth Providers` property contains the primary configured auth provider type as a string. Report the distribution directly — each row is a provider type and the count of Centrals using it. Centrals with `(none)` either have no external auth provider or are on older versions that don't report this property.

**Configuration Method (Queries 13b, 13c):** For each, count Centrals with value > 0 (exclude `0` and `(none)`). This shows how many Centrals use declarative vs. imperative auth provider management.

**Role-Mapping Groups (Queries 13d–13i):** For each provider type, count Centrals where groups > 0 (exclude `(none)` and `0`). This shows which auth provider types have active role-mapping configurations. Omit provider types with 0 adoption (e.g., Auth0, IAP) from the report table.

This query does not need a previous-period comparison since it uses a rolling 30-day window.

---

#### Signature Integration

**Query 9 — Signature Integration:**

Query the 5 charts from the [Signature integration charts](https://app.amplitude.com/analytics/redhat/dashboard/hcgn3s0n) dashboard. These are composition charts using `MOST_RECENT` on user properties, filtered to the "Centrals Prod" cohort (`59vr7tn`).

Use `query_charts` with chart IDs `["y9th4d37", "dimd52om", "wgh1zyck", "n87yt4k6", "8o6ko77i"]`:
- **Chart `y9th4d37`** ("Total signature integrations") — user property `gp:Total Signature Integrations`. Distribution of how many signature integrations each Central has configured.
- **Chart `dimd52om`** ("Total number of public keys") — user property `gp:Total Signature Integration Cosign Public Keys`. Sum of cosign public keys across all integrations per Central.
- **Chart `wgh1zyck`** ("Total number of certificates") — user property `gp:Total Signature Integration Certificates`. Sum of certificates across all integrations per Central.
- **Chart `n87yt4k6`** ("Transparency log integrations") — user property `gp:Total Signature Integration With Transparency Log Validation`. Integrations with transparency log validation enabled.
- **Chart `8o6ko77i`** ("Keyless verification integrations") — user property `gp:Total Signature Integration Certificates`, additionally filtered to Centrals where `gp:Total Signature Integration With Certificate Transparency Log Validation` is not "0" or "(none)". Heuristic for keyless (Sigstore/Fulcio) verification.

From the results, compute for each chart:
1. **Total eligible Centrals** — sum of all rows (the denominator).
2. **Feature not configured / (none)** — count of Centrals with value `(none)` or `0`, and percentage.
3. **Centrals with feature enabled** — count of Centrals with a numeric value > 0.

For the primary "Total signature integrations" chart, additionally report:
4. **Total integrations** — sum of (value × count) for all numeric values > 0.

This query does not need a previous-period comparison since the charts already use a rolling 30-day window. Include the results in the report's Low / No Usage section when adoption is below 5% of eligible Centrals.

---

#### Image Integrations

**Query 10 — Image Integrations:**

Query the distribution of configured image registry integrations across the production fleet. Run one `query_dataset` composition query per registry type, all in parallel. Each query uses the same structure — only the `property.value` changes.

Reference dashboard: [Image Integrations](https://app.amplitude.com/analytics/redhat/dashboard/a9fiq9jy).

Template for each registry query:

```json
{
  "type": "composition",
  "app": "418843",
  "params": {
    "range": "Last 30 Days",
    "metric": "MOST_RECENT",
    "interval": 1,
    "property": {"type": "user", "value": "<user_property>", "group_type": "User"},
    "groupBy": [],
    "segments": [{"conditions": [{"op": "is", "prop": "userdata_cohort", "type": "property", "values": ["59vr7tn"], "prop_type": "user", "group_type": "User"}]}],
    "countGroup": "User"
  }
}
```

Run one query per registry, substituting `<user_property>`:

| Registry | User Property |
|----------|---------------|
| Docker | `gp:Total Docker Image Integrations` |
| Artifactory | `gp:Total Artifactory Image Integrations` |
| Nexus | `gp:Total Nexus Image Integrations` |
| IBM | `gp:Total Ibm Image Integrations` |
| ACR (Azure) | `gp:Total Azure Image Integrations` |
| ECR (AWS) | `gp:Total Ecr Image Integrations` |
| GCR (Google) | `gp:Total Google Image Integrations` |
| Artifact Registry (Google) | `gp:Total Artifactregistry Image Integrations` |
| Red Hat | `gp:Total Rhel Image Integrations` |
| Quay | `gp:Total Quay Image Integrations` |

Run all 10 queries in parallel.

From each result, compute:
1. **Total eligible Centrals** — sum of all rows (should be consistent across registries since the cohort is the same).
2. **Centrals with ≥1 integration** — count of Centrals with a numeric value > 0 (exclude `(none)` and `0`).

**Summarization rules for the report:**
- Rank registries by adoption (Centrals with ≥1 integration) descending.
- Show the **top 5 registries** in the report table.
- If any remaining registries have adoption > 0, list them in a single "Also configured" line below the table (e.g., "Also configured: IBM (N), Nexus (N)").
- Omit registries with 0 adoption entirely.

This query does not need a previous-period comparison since it uses a rolling 30-day window.

---

#### Virtual Machine Usage

**Query 11 — Virtual Machine Usage:**

Query the two charts from the [ACS Virtual Machine Usage](https://app.amplitude.com/analytics/redhat/dashboard/2779g7bb) dashboard. These are composition charts using `MOST_RECENT` on user properties, filtered to production Centrals on ACS ≥ 4.9.0.

Use `query_charts` with chart IDs `["3crci9ik", "n2akv40h"]`:
- **Chart `3crci9ik`** ("ACS Virtual Machine Usage") — user property `gp:Total Virtual Machines`. Shows how many VMs each Central has observed.
- **Chart `n2akv40h`** ("ACS Active Virtual Machines") — user property `gp:Total Virtual Machines With Active Agents (Last 24h)`. Shows how many VMs have active agents.

From the results, compute:
1. **Total eligible Centrals** — sum of all rows (the denominator).
2. **Feature disabled** — count of Centrals with value `(none)` and percentage.
3. **Centrals with VMs** — count of Centrals with a numeric value > 0, and total VM count (sum of value × count).
4. **Active VM agents** — same breakdown from the active agents chart.

This query does not need a previous-period comparison since the charts already use a rolling 30-day window. Include the results in the report's Low / No Usage section when adoption is below 5% of eligible Centrals.

---

#### Feature Cross-Correlation (monthly only)

**Query 12 — Feature Cross-Correlation:**

This query identifies which features are used together by the same users. Skip this for weekly reports (the data is more meaningful over 30 days).

**Step 12a — Baselines:** Query `Page Viewed` with `metric: "uniques"` and behavioral segments for each key feature. Use one segment with no conditions (total UI users) plus one segment per feature, each with a single behavioral condition (`type: "event"`, `event_type: "<feature event>"`, `time_type: "rolling"`, `time_value: 30`, `op: ">="`, `value: 1`).

Key features to correlate:
- `Workload CVE Filter Applied` (Workload CVE Scanning)
- `Vulnerability Report Created` (Vulnerability Reporting)
- `Collection Created` (Collections)
- `Compliance Schedules Wizard Save Clicked` (Compliance Scheduling)
- `Platform CVE Filter Applied` (VM Vulnerability Management)
- `Image SBOM Generated` (SBOM Export)

**Step 12b — Pairwise overlaps:** For each pair of features, query `Page Viewed` with `metric: "uniques"` and a segment containing TWO behavioral conditions (both features). Batch pairs into 2–3 queries using multiple segments per query.

**Step 12c — Compute affinity:** For each pair, compute `overlap / min(feature_A_users, feature_B_users)` — this shows what percentage of the smaller feature's users also use the larger feature. Rank pairs by this affinity score descending. Only report pairs where both features have ≥ 5 users and the overlap is ≥ 2 users.

### Step 3: Query Previous Period

Run the same queries 1–8 with `start` and `end` set to the previous period dates. Queries 9–13 do not need previous-period runs (composition charts use rolling windows; cross-correlation is computed from the current period only).

### Step 4: Compute Deltas and Identify New Releases

**Metric deltas:** For each metric:
1. Calculate `change = ((current - previous) / previous) * 100`.
2. Round to one decimal place.
3. If previous is 0 and current > 0, label as `New`.
4. If both are 0, label as `—`.
5. Flag any metric with `|change| > 25%` as a notable change.

**Version analysis:** Compare the version lists from Query 2 between the current and previous periods:
1. **New releases:** Versions present in the current period but absent from the previous period. These represent releases that shipped during (or just before) the current period.
2. **Top versions:** Show the top 10 versions by Central count for the current period, with their previous-period count and change.
3. **Adoption velocity:** For new releases, note how many Centrals adopted the version within the period — rapid adoption of a new minor/patch version indicates a healthy upgrade cycle.
4. **Version format:** ACS versions follow `major.minor.patch` (e.g., `4.10.2`). Group observations by minor version line (e.g., "4.10.x") when summarizing trends.

### Step 5: Format the Report

Use this template. Replace placeholders with computed values.

```markdown
# ACS Telemetry Report

**Period:** YYYY-MM-DD to YYYY-MM-DD (7 | 30 complete days)
**Previous period:** YYYY-MM-DD to YYYY-MM-DD
**Generated:** YYYY-MM-DDTHH:MM:SSZ
**Project:** ACS Instances (418843)
**Cadence:** Weekly | Monthly

---

## Executive Summary

Write 2–4 sentences that capture the most important takeaways from the data. Focus on:
1. Fleet growth direction (growing, stable, or shrinking).
2. Any new ACS releases that shipped and their adoption momentum.
3. The single most significant usage change (positive or negative) and its magnitude.
4. Any operational concern (e.g., page crash increase, sharp drop in a core workflow).

Keep the tone neutral and factual — no recommendations or speculation.

## Fleet Overview

| Metric | Current | Previous | Change |
|--------|---------|----------|--------|
| Active Centrals | N | N | +X.X% |
| Active Secured Clusters | N | N | +X.X% |

## Release Versions

### New Releases This Period

List any ACS versions that appeared in the current period but were absent from the previous period. These represent new releases that shipped during or just before this period:

- **X.Y.Z** — N Centrals adopted (New)

If no new versions appeared, write: "No new releases detected."

### Top Versions by Fleet Share

| Version | Centrals | Previous | Change |
|---------|----------|----------|--------|
| X.Y.Z | N | N | +X.X% |
| ... (top 10) | | | |

## Top Feature Pages

| Feature | Views | Change |
|---------|-------|--------|
| Vulnerability Management | N | +X% |
| Violations | N | +X% |
| ... (top 10) | | |

## Vulnerability Management

| Metric | Count | Change |
|--------|-------|--------|
| Workload CVE Filter Applied | N | +X% |
| Platform CVE Filter Applied | N | +X% |
| Platform CVE Detail Views | N | +X% |
| Node CVE Filter Applied | N | +X% |
| Node CVE Detail Views | N | +X% |
| Vulnerability Report Created | N | +X% |
| Vulnerability Report Downloaded | N | +X% |
| View-Based Report Generated | N | +X% |
| Image SBOM Generated | N | +X% |

## Cluster Onboarding

| Metric | Count | Change |
|--------|-------|--------|
| Init Bundle Created | N | +X% |
| Init Bundle Downloaded | N | +X% |
| CRS Created | N | +X% |
| Secure Cluster Link Clicked | N | +X% |
| CRS Secure Cluster Link Clicked | N | +X% |
| Secured Clusters Registered | N | +X% |
| Secured Clusters Initialized | N | +X% |

## Compliance & Collections

| Metric | Count | Change |
|--------|-------|--------|
| Compliance Report Downloads | N | +X% |
| Compliance Wizard Completed | N | +X% |
| Collection Created | N | +X% |

## OCP Console Plugin

### Plugin Adoption (production clusters)

| Metric | Current | Previous | Change |
|--------|---------|----------|--------|
| Clusters with Plugin Installed | N | N | +X.X% |
| Clusters Actively Using Plugin | N | N | +X.X% |

### Token Activity

| Metric | Count | Change |
|--------|-------|--------|
| Internal Tokens Issued | N | +X% |
| Tokens with Full Cluster Access | N | +X% |

### Security Tab Page Views

| Page | Views | Change |
|------|-------|--------|
| Vulnerabilities | N | +X% |
| Cluster Details | N | +X% |
| Namespace/Workload Details | N | +X% |

## Platform & Access

| Metric | Count | Change |
|--------|-------|--------|
| API Calls (tracked) | N | +X% |
| User Invitations | N | +X% |
| Page Crashes | N | +X% |

## Auth Providers

Shows how production Centrals are configured for external authentication. Data is from static Central user properties using rolling 30-day windows — no period-over-period comparison. Write 2–3 sentences summarizing the dominant auth provider type, the split between declarative and imperative configuration, and any notable gaps (e.g., zero adoption of certain provider types).

### Auth Provider Types

| Provider Type | Centrals | % of Fleet |
|--------------|----------|------------|
| OpenShift | N | X% |
| OIDC | N | X% |
| SAML | N | X% |
| User PKI | N | X% |
| None configured | N | X% |

### Configuration Method

| Method | Centrals | % of Fleet |
|--------|----------|------------|
| Imperative (UI/API) | N | X% |
| Declarative (config-as-code) | N | X% |

### Role-Mapping Groups by Provider Type

Centrals with at least one role-mapping group configured per auth provider type:

| Provider Type | Centrals with Groups | % of Fleet |
|--------------|---------------------|------------|
| OpenShift | N | X% |
| OIDC | N | X% |
| SAML | N | X% |
| User PKI | N | X% |

Omit provider types with 0 adoption (e.g., Auth0, IAP).

## Signature Integration

Shows how many production Centrals have configured cosign signature verification integrations. Data is from composition charts using rolling 30-day windows — no period-over-period comparison.

| Metric | Centrals | % of Fleet |
|--------|----------|------------|
| Eligible Centrals (total) | N | — |
| With signature integrations | N | X% |
| With public keys configured | N | X% |
| With certificates configured | N | X% |
| With transparency log validation | N | X% |
| With keyless verification | N | X% |

## Image Integrations

Top container registries configured across the production fleet (rolling 30-day window, no period-over-period comparison). Only the top 5 by adoption are shown.

| Registry | Centrals | % of Fleet |
|----------|----------|------------|
| Registry A | N | X% |
| Registry B | N | X% |
| ... (top 5) | | |

Also configured: Registry F (N), Registry G (N) — if any additional registries have adoption > 0.

## Notable Changes

List every metric where |change| > 25%, sorted by absolute change descending:

- ⬆ **Metric Name** up X% (N → N)
- ⬇ **Metric Name** down X% (N → N)

If no metrics exceed the 25% threshold, write: "No significant changes detected."

## Feature Cross-Correlation (monthly only)

Shows which features are commonly used together by the same users. Pairs are ranked by affinity — the percentage of the smaller feature's users who also use the larger feature.

| Feature Pair | Overlap | Affinity | Interpretation |
|-------------|---------|----------|----------------|
| Feature A ↔ Feature B | N users | X% of Feature B users | Brief note |

After the table, write 2–3 sentences summarizing the key feature clusters and any isolated features.

## Low / No Usage

List feature action metrics (from the Metric-to-Feature Mapping table) where the current period count is very low relative to the fleet size. Use these thresholds:
- **No usage:** count = 0
- **Very low usage:** count ≤ 5 (weekly) or ≤ 20 (monthly)

Exclude infrastructure metrics that are inherently high-volume or not tied to a specific feature (API Calls, Page Viewed, Page Crashes). Focus on feature adoption indicators only.

For each low-usage feature, show the count, the feature name, and the previous period count for context:

- **Feature Name** (Event Label) — N events (previous: N)

If no features fall below the threshold, write: "All tracked features saw meaningful usage this period."
```

### Step 6: Write Output

1. Create the output directory if it doesn't exist:
   ```
   mkdir -p docs/telemetry-reports
   ```
2. Write the full report to `docs/telemetry-reports/YYYY-MM-DD-telemetry-report.md` for weekly reports, or `docs/telemetry-reports/YYYY-MM-DD-monthly-telemetry-report.md` for monthly reports (use the current date).
3. Print a **terminal summary** containing only:
   - The report header (period, generated timestamp)
   - Executive Summary
   - Fleet Overview table
   - Notable Changes section
   - Path to the full report file

## Error Handling

- If a `query_dataset` call fails, note the error in the corresponding report section and continue with remaining queries.
- If a query returns no data for an event, show `0` for the count and `—` for the change.
- If the previous period has no data at all (e.g., telemetry was just enabled), skip the change column and note "No baseline data available."

## Grouping Page Paths into Feature Areas

When displaying page views, collapse specific entity paths (paths containing UUIDs or IDs) into their parent feature. Map the top-level path segments to feature names:

| Path prefix | Feature Name |
|-------------|-------------|
| `/main/dashboard` | Dashboard |
| `/main/vulnerability-management` | Vulnerability Management |
| `/main/vulnerabilities` | Vulnerability Management |
| `/main/violations` | Violations |
| `/main/network-graph` | Network Graph |
| `/main/compliance` | Compliance |
| `/main/clusters` | Clusters |
| `/main/collections` | Collections |
| `/main/risk` | Risk |
| `/main/policy-management` | Policies |
| `/main/integrations` | Integrations |
| `/main/access-control` | Access Control |
| `/main/system-health` | System Health |
| `/main/systemconfig` | System Configuration |
| `/main/configmanagement` | Configuration Management |

Aggregate view counts by feature name, not individual paths.

## Metric-to-Feature Mapping

Many telemetry events are indicators for well-known ACS product features. When displaying metrics in the report, use the feature-aware label (the "Report Label" column) so readers can immediately associate the number with the feature they know.

| Event Name | Feature | Report Label |
|------------|---------|-------------|
| `Internal Token Issued` (unfiltered, uniques) | OCP Console Plugin | Clusters with Plugin Installed |
| `Internal Token Issued` (Total Cluster Scopes > 0, uniques) | OCP Console Plugin | Clusters Actively Using Plugin |
| `Internal Token Issued` (Total Cluster Scopes > 0, totals) | OCP Console Plugin | Internal Tokens Issued |
| `Internal Token Issued` (Cluster Scopes With Full Access > 0, totals) | OCP Console Plugin | Tokens with Full Cluster Access |
| `Page Viewed` (acsApplicationSource = console-plugin, vulnerabilities path) | OCP Console Plugin | Security Tab: Vulnerabilities |
| `Page Viewed` (acsApplicationSource = console-plugin, cluster path) | OCP Console Plugin | Security Tab: Cluster Details |
| `Page Viewed` (acsApplicationSource = console-plugin, namespace path) | OCP Console Plugin | Security Tab: Namespace/Workload Details |
| `gp:Total Signature Integrations` (composition chart `y9th4d37`) | Signature Integration | Centrals with Signature Integrations |
| `gp:Total Signature Integration Cosign Public Keys` (composition chart `dimd52om`) | Signature Integration | Centrals with Public Keys |
| `gp:Total Signature Integration Certificates` (composition chart `wgh1zyck`) | Signature Integration | Centrals with Certificates |
| `gp:Total Signature Integration With Transparency Log Validation` (composition chart `n87yt4k6`) | Signature Integration | Centrals with Transparency Log |
| `gp:Total Signature Integration Certificates` + keyless filter (composition chart `8o6ko77i`) | Signature Integration | Centrals with Keyless Verification |
| `gp:Auth Providers` (composition, Centrals Prod) | Auth Providers | Auth Provider Types |
| `gp:Total Declarative Auth Providers` (composition, Centrals Prod) | Auth Providers | Declarative Auth Providers |
| `gp:Total Imperative Auth Providers` (composition, Centrals Prod) | Auth Providers | Imperative Auth Providers |
| `gp:Total Groups of oidc` (composition, Centrals Prod) | Auth Providers | OIDC Role-Mapping Groups |
| `gp:Total Groups of saml` (composition, Centrals Prod) | Auth Providers | SAML Role-Mapping Groups |
| `gp:Total Groups of openshift` (composition, Centrals Prod) | Auth Providers | OpenShift Role-Mapping Groups |
| `gp:Total Groups of userpki` (composition, Centrals Prod) | Auth Providers | User PKI Role-Mapping Groups |
| `Workload CVE Filter Applied` | Workload CVE Scanning | Workload CVE Filter Applied |
| `Platform CVE Filter Applied` | VM Vulnerability Management | Platform CVE Filter Applied |
| `Platform CVE Entity Context View` | VM Vulnerability Management | Platform CVE Detail Views |
| `Node CVE Filter Applied` | Node CVE Scanning | Node CVE Filter Applied |
| `Node CVE Entity Context View` | Node CVE Scanning | Node CVE Detail Views |
| `Vulnerability Report Created` | Vulnerability Reporting | Vulnerability Report Created |
| `Vulnerability Report Download Generated` | Vulnerability Reporting | Vulnerability Report Downloaded |
| `View Based Report Generated` | Vulnerability Reporting | View-Based Report Generated |
| `Image SBOM Generated` | SBOM Export | Image SBOM Generated |
| `Collection Created` | Collections | Collection Created |
| `Compliance Report Download Generation Triggered` | Compliance Reporting | Compliance Report Downloads |
| `Compliance Schedules Wizard Save Clicked` | Compliance Scheduling | Compliance Wizard Completed |
| `Create Init Bundle Clicked` | Init Bundles (cluster onboarding) | Init Bundle Created |
| `Download Init Bundle` | Init Bundles (cluster onboarding) | Init Bundle Downloaded |
| `Create Cluster Registration Secret Clicked` | Cluster Registration Secrets | CRS Created |
| `CRS Secure a Cluster Link Clicked` | Cluster Registration Secrets | CRS Secure Cluster Link Clicked |
| `Secure a Cluster Link Clicked` | Cluster Onboarding | Secure Cluster Link Clicked |
| `Secured Cluster Registered` | Cluster Onboarding | Secured Clusters Registered |
| `Secured Cluster Initialized` | Cluster Onboarding | Secured Clusters Initialized |
| `Invite Users Submitted` | User Management | User Invitations |
| `Page Crash` | Platform Stability | Page Crashes |
| `API Call` | — (general) | API Calls (tracked) |

In the **Executive Summary** and **Notable Changes** sections, always include the feature name in parentheses when referencing a metric that maps to a specific feature. For example: "Internal Tokens Issued (Console Plugin) rose 69.6%."

## Notes

- **Uniques are non-additive.** Always read `overallSeries` for unique counts. Do NOT sum `timeSeries` values — that double-counts users who appear in multiple intervals.
- **Event names are exact.** Use the event names listed above verbatim. Do not guess or modify them.
- **Exclude deleted/test events.** Only query events that are active and queryable (the events listed in this skill are pre-filtered).
- **Keep the report factual.** State numbers and changes. Do not speculate about causes or recommend actions — the audience will draw their own conclusions.

# 0008 - NVD Enricher

- **Author(s):** J. Victor Martins <jvdm@sdf.org>
- **Created:** [2024-01-22 Mon]

## Status

Status: Accepted.
Updates: [#0005 - Adapt Scanner V4 NVD CVSS data updating pipeline to integrate with NVD CVE API](0005-scannerv4-read-cvss-data.md)

## Context

The [ClairCore](https://github.com/quay/claircore) CVSS scores enricher does not offer all the necessary functionality to fulfill Stackrox Scanner's need for NVD data:

1.  It only pulls CVSS scores into the enrichment table.  Stackrox Scanner must pull additional data in NVD CVE objects to fill in incomplete information in specific security sources (e.g., Alpine).  Additionally, Stackrox Scanner wants to open the door to offer users the option to pull NVD in addition to the vulnerability information provided by the image provider's security source.
2.  At the time of this writing, ClairCore enricher does not support NVD API v2 and is still relying on the deprecated NVD v2 JSON documents.

Moreover, after the merge of ClairCore's [PR #1166](https://github.com/quay/claircore/pull/1166), enrichment can bundle their data into offline JSON blobs, which can be consumed with vulnerability data.  Previously, this was not possible, and [ADR #0005](0005-scannerv4-read-cvss-data.md) created a pipeline for that data.  This pipeline is now unnecessary.

## Decision

Stackrox Scanner will have its own CVSS enricher that:

1.  Pulls information from NVD API v2.
2.  Pulls additional CVE information from NVD, leaving the door open to add or remove information as needed.
3.  Replaces the NVD CVSS score pipeline by using the enricher in the vulnerability bundle exporter to consume the enrichments and vulnerability data later.

## Consequences

1.  Stackrox Scanner can decommission the NVD CVSS pipeline (workflows and Central handlers). 
2.  Stackrox Scanner can programmatically add or remove additional NVD information to the security sources.  The enricher allows for filtering NVD data not used to reduce bundle sizes and increase them as needed.
3.  Vulnerability bundles are currently versioned, so changes to NVD information captured by the enricher don't have to be backward compatible.

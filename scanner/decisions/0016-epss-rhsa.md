# 0016 - EPSS for RHSAs

- **Author(s):** Ross Tannenbaum
- **Created:** [2025-02-10 Mon]

## Status

Status: Accepted

## Context

The original decision for handling Red Hat advisories (without loss of generality will just be referred as RHSAs) is as follows:

> [...] Scanner V4 currently displays RHSA as the top-level entity rather than the related CVE(s) when the CVE(s) are fixed.
Meanwhile, all EPSS data are CVE-centric. In Scanner V4, the EPSS score for an RHSA will be the highest EPSS score among all CVEs linked to that RHSA of a given image, as multiple CVEs can be associated with a single RHSA.
This approach matches our current scheme for assigning an RHSA a CVSS score

See the original statement with full context in [#0014](0014-epss-score.md).

We believed the work associated with [#0015](0015-csaf-enricher.md) would aid in this endeavor, as the CSAF enricher would be
able to determine all CVEs related to each RHSA; however, this proved to be insufficient.

At a high-level: there is no clean API for consumers of Vulnerability Report enrichments to simply look at the mapping of CVE
to its associated EPSS score. So, there is no simple way for us to go from the list of each RHSA's related CVEs, see each CVE's score,
then choose the highest.

## Decision

We decided to accept that consumers may experience seeing a level of EPSS score flapping. The hope is this will not 
matter anymore in 4.8 when, hopefully, we can successfully stop showing RHSAs as the top-level vulnerability 
name and replace them with CVEs.

## Consequences

* EPSS scores may still experience a level of flapping which both https://github.com/stackrox/stackrox/pull/13559 
  and https://github.com/stackrox/stackrox/pull/13523 aimed to fix for RHSA severity, CVSS score, description, etc.
* The UI has been updated to omit EPSS columns for images and deployments for the 4.7 release
  * https://github.com/stackrox/stackrox/pull/14201

# 0013 - Manual CVSS Source

- **Author(s):** Ross Tannenbaum
- **Created:** [2024-10-03 Thurs]

## Status

Status: Accepted

Updates: [#0010](0010-nvd-cvss-scores.md)

## Context

It was decided to support the following sources of CVSS scores:

* Red Hat
* OSV
* NVD

However, this neglects the vulnerabilities we manually curate. The scores we create do not just come out of nowhere,
and the source does not necessarily need to be UNKNOWN.

## Decision

It is never truly defined where the vulnerability data may originate. Sometimes, it is lifted directly from NVD.
Other times, it is from another source which scores the vulnerability before NVD has the chance.

The [`claircore.Vulnerability`](https://github.com/quay/claircore/blob/main/vulnerability.go) struct does not
give us room to add our own metadata, aside from the updater name. We do not want to change the updater name,
as keeping the name consistent simplifies debugging (easy to search for vulnerabilities we manually added).

Instead, we opt to consider the `Links` field. The manually added vulnerabilities must set a link,
and we may use this link as the source of the CVSS score.

For example, one entry is copied below:

```yaml
# Vuln: CVE-2022-22963/GHSA-6v73-fgf6-w5j7
# Reason: The vuln table has an entry for GHSA-6v73-fgf6-w5j7, but Scanner V4
# may have trouble determining the groupID when pom.properties is missing.
# Source: https://osv-vulnerabilities.storage.googleapis.com/Maven/GHSA-6v73-fgf6-w5j7.json
- Name: CVE-2022-22963
  Description: Spring Cloud Function Code Injection with a specially crafted SpEL as a routing expression
  Issued: '2022-04-03T00:00:59Z'
  Links: https://nvd.nist.gov/vuln/detail/CVE-2022-22963
  Severity: CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H
  NormalizedSeverity: Critical
  Package:
    Name: spring-cloud-function-context
    Kind: Binary
    RepositoryHint: Maven
  FixedInVersion: introduced=0&fixed=3.1.7
  Repo:
    Name: maven
    URI: https://repo1.maven.apache.org/maven2
```

The `Links` field is a single NVD link. We will recognize this link and identify the CVSS source as NVD.

## Consequences

* We would need to ensure future entries to the manual entries are consistent and follow a set of rules.
  * There is now a [README.md](../updater/manual/README.md) file to ensure we maintain consistency.

# 0014 - EPSS Source

- **Author(s):** Yi Li
- **Created:** [2025-01-08 Wednesday]

## Status

Status: Accepted

## Context

The Exploit Prediction Scoring System (EPSS) is a data-driven approach to estimating the likelihood (probability) that a software vulnerability will be exploited in the wild.
Many users now consult EPSS scores to better prioritize vulnerability remediation efforts and Scanner V4 will integrate EPSS scores into its vulnerability reports.

The primary focus is on CVEs, along with RHSAs, RHEAs, and RHBAs. This document does not cover other types of advisories, such as Ubuntu's USNs.

The current API looks like the following:
```
message VulnerabilityReport {
  message Vulnerability {
    enum Severity {
      SEVERITY_UNSPECIFIED = 0;
      ...
      SEVERITY_CRITICAL = 4;
    }
    message CVSS {
      enum Source {
        SOURCE_UNKNOWN = 0;
        SOURCE_RED_HAT = 1;
        SOURCE_OSV = 2;
        SOURCE_NVD = 3;
      }
      message V2 {
        float base_score = 1;
        string vector = 2;
      }
      message V3 {
        float base_score = 1;
        string vector = 2;
      }
      V2 v2 = 1;
      V3 v3 = 2;
      Source source = 3;
      string url = 4;
    }
    string id = 1;
    string name = 2;
    string description = 3;
    google.protobuf.Timestamp issued = 4;
    string link = 5 [deprecated = true]; // link is duplicated with CVSS URL field, the exact deprecation date is undecided
    ...
    repeated CVSS cvss_metrics = 13;
  }
  ...
  repeated Note notes = 5;
}
```

## Decision

All EPSS data integrated by Scanner V4 are fetched from https://epss.cyentia.com/epss_scores-YYYY-MM-DD.csv.gz, originating from the [First Organization](https://www.first.org/epss/api). 

The [`claircore.EPSS`](https://github.com/quay/claircore/blob/main/enricher/epss/epss.go) is used for data fetching, parsing and enriching, as a component of Scanner V4.

All EPSS data is CVE-centric, aligning with Scanner V4's recent adaptation to VEX data for vulnerability matching for RHEL-based images.

All EPSS data integrated to Scanner V4 corresponds to the day prior to the current date, as this approach reduces the likelihood of failure compared to fetching the current date's data, which may not always be ready.

The Api will be extended to:

```
message VulnerabilityReport {
  message Vulnerability {
    enum Severity {
      SEVERITY_UNSPECIFIED = 0;
      ...
      SEVERITY_CRITICAL = 4;
    }
    message CVSS {
      enum Source {
        SOURCE_UNKNOWN = 0;
        ...
      }
      message V2 {
        float base_score = 1;
        string vector = 2;
      }
      message V3 {
        float base_score = 1;
        string vector = 2;
      }
      V2 v2 = 1;
      V3 v3 = 2;
      Source source = 3;
      string url = 4;
    }
    string id = 1;
    string name = 2;
    string description = 3;
    google.protobuf.Timestamp issued = 4;
    string link = 5 [deprecated = true]; // link is duplicated with CVSS URL field, the exact deprecation date is undecided
    ...
    repeated CVSS cvss_metrics = 13;
     message EPSS { <-- new proto
      string date = 1;
      string model_version = 2;
      float probability = 3;
      float percentile = 4;
    }
    EPSS epss = 14; <-- new field
  }
  ...
  repeated Note notes = 5;
}
```
### Handling RHSA/RHBA/RHEA

Without loss of generality, RHSA/RHBA/RHEA will just be referred to as the more well-known RHSA variant of the three.
Scanner V4 currently displays RHSA as the top-level entity rather than the related CVE(s) when the CVE(s) are fixed. 

And the vulnerability data from the VEX file is CVE-centric, CVE identifiers needs to be swapped with RHSAs in Scanner V4, if applicable.
The RHSA details displayed by Scanner V4 are based on the CVE with the highest CVSS score associated with that RHSA. 

As the result, the EPSS score shown in Scanner V4 for that RHSA will be the score associated with the same CVE, if applicable.

## Consequences

* We would need to ensure the [ First organization's data source](https://epss.cyentia.com/epss_scores-YYYY-MM-DD.csv.gz) remains consistently accessible. 
Additionally, we should be prepared to change the data updating process if any API key requirements or request throttling are introduced in the future.

* Currently the RHSA details displayed are associated with the CVE that has the highest CVSS score, as a single RHSA can relate to multiple CVEs.
As mentioned above, the EPSS score shown in Scanner V4 corresponds to the CVE with the highest CVSS score linked to that RHSA.
This may change when Scanner V4 introduces CVEs as top-level entities, allowing each CVE to have its own EPSS score, if applicable.

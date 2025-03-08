# 0014 - EPSS Score

- **Author(s):** Yi Li
- **Created:** [2025-01-08 Wednesday]

## Status

Status: Updated by [#0016](0016-epss-rhsa.md)

## Context

[EPSS](https://www.first.org/epss/) predicts the likelihood of a vulnerability being exploited within 30 days, assigning a CVE a probability from 0% to 100%. A higher probability means a greater risk. 
EPSS also provides a percentile ranking to indicate how the vulnerability compares to others in terms of threat level.
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

The [`claircore.EPSS` enricher](https://github.com/quay/claircore/blob/main/enricher/epss/epss.go) is used for data fetching, parsing and enriching, as a component of Scanner V4.

All workflows in GitHub Actions, including those running cron jobs, operate in UTC. Similarly, the EPSS data example aligns with UTC, as shown in the timestamp: `score_date:2025-01-10T00:00:00+0000`. 
All EPSS data integrated into Scanner V4 corresponds to the previous day in UTC. This approach minimizes the risk of failures caused by fetching data for the current UTC date, which may not yet be available.
To include the EPSS details in Scanner V4, the protobuf message will be extended to:

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
    message EPSS { <-- new proto
      string date = 1;
      string model_version = 2;
      float probability = 3;
      float percentile = 4;
    }
    string id = 1;
    string name = 2;
    string description = 3;
    google.protobuf.Timestamp issued = 4;
    string link = 5 [deprecated = true]; // link is duplicated with CVSS URL field, the exact deprecation date is undecided
    ...
    repeated CVSS cvss_metrics = 13;
    EPSS epss_metrics = 14; <-- new field
  }
  ...
  repeated Note notes = 5;
}
```
### Handling RHSA/RHBA/RHEA

Without loss of generality, RHSA/RHBA/RHEA will just be referred to as the more well-known RHSA variant of the three.
Scanner V4 currently displays RHSA as the top-level entity rather than the related CVE(s) when the CVE(s) are fixed. 
Meanwhile, all EPSS data are CVE-centric. In Scanner V4, the EPSS score for an RHSA will be the highest EPSS score among all CVEs linked to that RHSA of a given image, as multiple CVEs can be associated with a single RHSA.
This approach matches our current scheme for assigning an RHSA a CVSS score

## Consequences

* We would need to ensure the [First organization's data source](https://epss.cyentia.com/epss_scores-YYYY-MM-DD.csv.gz) remains consistently accessible. 
Additionally, we should be prepared to change the data updating process if any API key requirements or request throttling are introduced in the future.

* As noted earlier, the EPSS score displayed in Scanner V4 for an RHSA corresponds to the highest EPSS score among the CVEs linked to that RHSA in a specific image. This approach does not provide the highest EPSS score for a given RHSA, and we know Central can overwrite the EPSS score for an RHSA across different images.
An improvement would be to use the CSAF enricher to retrieve the CVE list for an RHSA, allowing us to calculate the highest EPSS score among all associated CVEs. However, this ADR does not address that improvement at this time.
When Scanner V4 introduces CVEs as top-level entities, we can have each CVE to have its own EPSS score, if applicable. So the RHSA EPSS score calculation will not longer needed.
# 0008 - NVD CVSS Scores

- **Author(s):** Ross Tannenbaum
- **Created:** [2024-04-11 Thurs]

## Status

Status: Accepted.

## Context

Users who require FedRAMP compliance [must be able see NVD's CVSS scores associated with each CVE](https://www.fedramp.gov/assets/resources/documents/CSP_Vulnerability_Scanning_Requirements.pdf).

Currently, Scanner V4 provides a single CVSS score and severity, which is preferably from the vendor.
In reality, the CVSS scores that are shown are from NVD except in the following cases:

* RHEL-base images
  * Red Hat-provided CVSS scores/severities are used.
* OSV-provided data
  * If OSV.dev has a CVSS score, then that score is used.
    * The score is converted into a severity based on the CVSS version's specification.
  * Note: it is likely, if not always true, the score displayed by OSV is the same as NVD.

The current API looks like the following:

```
message VulnerabilityReport {
  message Vulnerability {
    enum Severity {
      SEVERITY_UNSPECIFIED = 0;
      SEVERITY_LOW = 1;
      SEVERITY_MODERATE = 2;
      SEVERITY_IMPORTANT = 3;
      SEVERITY_CRITICAL = 4;
    }
    message CVSS {
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
    }
    ...
    string severity = 6;
    ...
    CVSS cvss = 12;
  }
  ...
}
```

## Decision

The VulnerabilityReport API will be extended to support more than one CVSS message. This will be done as follows:

```
message VulnerabilityReport {
  message Vulnerability {
    enum Severity {
      SEVERITY_UNSPECIFIED = 0;
      SEVERITY_LOW = 1;
      SEVERITY_MODERATE = 2;
      SEVERITY_IMPORTANT = 3;
      SEVERITY_CRITICAL = 4;
    }
    message CVSS {
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
    }
    ...
    string severity = 6;
    ...
    CVSS cvss = 12;
    CVSS nvd_cvss = 13; <-- New field.
  }
  ...
}
```

Currently, there is only a need to have CVSS scores from the vendor (if available) and NVD (if available), no more.
Making this explicit, then, makes it easy and clear to users to find each scores and choose between the two.

Another option is to provide a more generic solution. One way to do this would be to update `CVSS cvss = 12` to 
`repeated CVSS cvss = 12` (note: the [proto3 spec](https://protobuf.dev/programming-guides/proto3/#updating) indicates this is compatible).
Doing it this way gives us a more generic way of supporting multiple CVSS scores vs adding a new `nvd_cvss` field,
and it may certainly be the way to go in the future should we want to support a variety of score sources; however,
there is no need for a generic solution at this time, and adding the new field make it very easy to choose between the two options.

### Handling RHSA/RHBA/RHEA

Without loss of generality, RHSA/RHBA/RHEA will just be referred to as the more well-known RHSA variant of the three.

Scanner V4 currently shows RHSA as the top-level entity, rather than the related CVE(s), when the CVE(s) is/are fixed.
When this is done, Scanner V4 gives the RHSA the highest CVSS score from the associated CVE(s). We acknowledge this is not
ideal, and there are plans to resolve this in the future. For now, we will need to support NVD scores in a compatible manner.

The [`claircore.Vulnerability.Severity`](https://github.com/quay/claircore/blob/v1.5.25/vulnerability.go#L24) is currently set to the following:

`severity=<severity>&cvss3_score=<score3>&cvss3_vector=<vector3>cvss2_score=<score2>&cvss2_vector=<vector2>`

This URL encoding will be extended to include `cve=<CVE ID>`. 
By doing this, Scanner V4 has the ability to relate the RHSA back to the CVE which has the highest score and search NVD for that CVE's score.

## Consequences

NVD only tracks CVEs, while Scanner V4 shows a variety of advisories (CVE, RHSA/RHBA/RHEA, ALAS, etc).

The current plan is only to support CVEs as well as RHSA/RHBA/RHEA.

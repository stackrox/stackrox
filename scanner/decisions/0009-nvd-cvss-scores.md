# 0009 - NVD CVSS Scores

- **Author(s):** Ross Tannenbaum and Yi Li
- **Created:** [2024-04-11 Thurs]

## Status

Status: Accepted.

## Context

Users who require FedRAMP compliance [must be able see NVD's CVSS scores associated with each CVE](https://www.fedramp.gov/assets/resources/documents/CSP_Vulnerability_Scanning_Requirements.pdf).
Note: NVD tracks CVEs, so all of its data is CVE-based.

Currently, Scanner V4 provides a single CVSS score and severity, which is preferably from the vendor.
In reality, the CVSS scores that are shown are from NVD except in the following cases:

* RHEL-base images
  * Red Hat-provided CVSS scores/severities are used.
* OSV-provided data
  * If OSV.dev has a CVSS score, then that score is used.
    * The severity is derived from the score based on the CVSS version's specification.
  * Note: From personal experience, it is very common for the score displayed by OSV is the same as NVD.

Currently, there is only a need to have CVSS scores from the vendor (if available) and NVD (if available), no more.
However, it is possible this requirement changes in the future, and we may want to support more than just two
sources for vulnerability data.

The focus is on CVEs and RHSAs, RHEAs, and RHBAs. This document will not consider other types of advisories such as
Ubuntu's USNs.

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

The VulnerabilityReport API will be extended as follows:

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
      Source source = 3; <-- New field
      string url = 4;    <-- New field
    }
    ...
    string severity = 6;
    ...
    CVSS cvss = 12;
    repeated CVSS cvss_metrics = 13; <-- New field.
  }
  ...
}
```

There will be a new type plus two new fields added:

* `CVSS.Source`
  * This specifies the supported CVSS metrics data sources.
* `CVSS.source`
  * This specifies the source of the particular CVSS metrics.
* `CVSS.url`
  * This specifies the CVSS score source URL, aiding both API and UI users in tracking the origin of this metric.
* `cvss_metrics`
  * This is a list of each unique CVSS metric based on the source.

The original `cvss` field will remain and will continue to represent the Scanner's preferred CVSS score.
This is currently the score from the vulnerability's original data source, if available, otherwise NVD.

**JSON example**:


```sh
"9216828": {
	"id": "9216828",
	"name": "CVE-2023-48231",
	"description": "DOCUMENTATION: A heap use-after-free flaw was found in the vim package. ... MITIGATION: Mitigation for this issue is either not available or ..., applicability to widespread installation base or stability.",
	"issued": {
		"seconds": -62135596800
	},
	"link": "https://access.redhat.com/security/cve/CVE-2023-48231",
	"severity": "Low",
	"normalized_severity": 1,
	"cvss": {
		"source": "SOURCE_RED_HAT",
		"v3": {
			"base_score": 4.3,
			"vector": "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L"
			},
		"url":"https://access.redhat.com/security/cve/CVE-2023-48231"
		},
	"cvss_metrics": [{
		"source": "SOURCE_RED_HAT",
		"v3": {
			"base_score": 4.3,
			"vector": "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L"
			},
		"url":"https://access.redhat.com/security/cve/CVE-2023-48231"
		},{
		"source": "SOURCE_NVD",
		"v3": {
			"base_score": 4.3,
			"vector": "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L"
			},
		"url":"https://nvd.nist.gov/vuln/detail/CVE-2023-48231"
		}]
}
```
### Handling RHSA/RHBA/RHEA

Without loss of generality, RHSA/RHBA/RHEA will just be referred to as the more well-known RHSA variant of the three.

Scanner V4 currently shows RHSA as the top-level entity, rather than the related CVE(s), when the CVE(s) is/are fixed.
When this is done, Scanner V4 gives the RHSA the highest CVSS score from the associated CVE(s). We acknowledge this is not
ideal, and there are plans to resolve this in the future. For now, we will need to support NVD scores in a compatible manner.

Encoding the RHSA/RHEA/RHBA's related CVE allows Scanner V4 to relate the advisory back to the CVE which has the highest score and search NVD for that CVE's score.
Other type of advisories like ALAS and USN will not have a score from NVD.

The [`claircore.Vulnerability.Severity`](https://github.com/quay/claircore/blob/v1.5.25/vulnerability.go#L24) is currently set to the following:

`severity=<severity>&cvss3_score=<score3>&cvss3_vector=<vector3>cvss2_score=<score2>&cvss2_vector=<vector2>`

This URL encoding will be extended to include `cve=<CVE ID>`.

## Consequences

* Creating an `enum` for `Source` instead of just using a `string` ensures consistency and limits mistakes which may be made
* `repeated CVSS` field `cvss_metrics` will always include CVSS metrics from all data sources, including the Scanner's preferred CVSS metric.
  * This approach simplifies data querying and filtering, as `cvss_metrics` will be the sole field used for filtering data or making policies.
* OSV.dev sometimes does not related non-CVEs (like GHSAs) back to CVEs. When this happens, we cannot determine the CVSS score from NVD.
* protobufs do not support enums as key types, so we cannot do something like `map<Source, CVSS> cvss_metrics = 13`.
  * We could just use a `string`, but then we run into the same potential pitfalls mentioned previously.
* Keeping `cvss` as-is allows for easy access to Scanner's preferred CVSS score.
* Extending the `severity` field for Red Hat advisories means we further diverge away from ClairCore's Red Hat updater.
  * There are two major efforts related to this, which are currently being worked on by the Clair team:
    * Adoption of CSAF/VEX files instead of OVAL
    * Return CVE-centric reports rather than the current advisory-centric report.
  * We will have to re-evaluate our custom updater as these are being developed and completed.

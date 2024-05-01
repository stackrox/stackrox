# 0009 - Improve Scanner V4 manually added vulnerabilities updating process

- **Author(s):** Yi Li <yli3@redhat.com>
- **Created:** [2024-04-30]

## Context

Currently, Scanner v4 includes a feature for manually updating vulnerabilities to add and update urgent vulnerability data. 
This allows clients to access the most recent vulnerability information before it becomes officially available in any data source. 
However, this manual update process has a major flaw: the vulnerability data is integrated with the codebase, which means clients cannot access the updated data unless they install a patch release that includes these vulnerabilities. 
Therefore, it is essential to improve this process to ensure that manually inserted vulnerabilities are not dependent on any ACS release cycle. This improvement would enable all versions starting from ACS 4.4 to retrieve manually updated vulnerabilities without any issues.

## Decision

Introduce a JSON file at /scanner/updater/manual/vulns.json (The format of the json will be listed below), and update the corresponding manual vulnerability updater located in scanner/updater/manual/manual.go. 
This updater will be utilized within the existing vulnerability updating GitHub Actions workflow. 
The new manual vulnerability updater will retrieve and parse the manually inserted data from https://github.com/stackrox/stackrox/blob/master/scanner/updater/manual/vulns.json. 
The parsed vulnerabilities will then be added into a vulns.zst file, which is generated during the updater process in the GitHub Action. 
When this vulns.zst file is imported by Scanner v4, it will also include the manually inserted vulnerability data as part of the ZST bundle.

### JSON Format
```javascript
{
    "vulnerabilities": [
        {
            "Name": "CVE-2022-22963",
            "Description":        "Spring Cloud Function Code Injection with a specially crafted SpEL as a routing expression",
            "Issued":             "2022-04-03T00:00:59Z",
            "Links": "https://nvd.nist.gov/vuln/detail/CVE-2022-29885",
            "Severity": "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
            "NormalizedSeverity": "Critical",
            "Package": {
                "Name": "spring-cloud-function-context",
                "RepositoryHint": "Maven",
            },
            "IntroducedInVersion": "0",
            "FixedInVersion": "3.1.7",
            "Repo": {
                "Name": "maven",
                "URI":  "https://repo1.maven.apache.org/maven2",
            }
        },
        ...,
        {
            "Name": "CVE-2023-28708",
            "Description":        "When using the RemoteIpFilter with requests received from a reverse proxy via HTTP that include the X-Forwarded-Proto header set to https, session cookies created by Apache Tomcat 11.0.0-M1 to 11.0.0.-M2, 10.1.0-M1 to 10.1.5, 9.0.0-M1 to 9.0.71 and 8.5.0 to 8.5.85 did not include the secure attribute. This could result in the user agent transmitting the session cookie over an insecure channel.",
            "Issued":             "2023-03-22T11:15:10Z",
            "Links":              "https://nvd.nist.gov/vuln/detail/CVE-2022-29885",
            "Severity": "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:N/A:N",
            "NormalizedSeverity": "Medium",
            "Package": {
                "Name": "org.apache.tomcat-embed-core:tomcat-embed-core",
                "RepositoryHint": "Maven",
            },
            "IntroducedInVersion": "8.5.0",
            "FixedInVersion": "8.5.86",
            "Repo": {
                "Name": "maven",
                "URI":  "https://repo1.maven.apache.org/maven2",
            }
        }
    ]
}

```

## Consequences

* Following this modification, clients will no longer need to update the ACS patch release to access manually added vulnerabilities.
* Adding new vulnerabilities to the manual data source now only requires updates to a single branch, eliminating the need for changes across multiple patch release branches. 
* The vulnerability JSON data format must maintain backward compatibility with all supported ACS versions and patch releases, provided that versioned bundles continue to be utilized.
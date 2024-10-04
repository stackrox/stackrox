# Scanner V4 "manual" vulnerabilities

This directory contains [vulns.yaml](vulns.yaml) which lists vulnerabilities which Scanner V4 currently misses.
This may be for various reasons, such as:

* The vulnerability is brand new, and not all Linux vendors track it yet
* Scanner V4 is unable to match an affected package with the associated vulnerability in its normal capacity
* The typical sources of data only have partial data (for example: the vulnerability has a severity but no CVSS score)

## Adding a vulnerability

Be sure to do the following:

* Leave a comment directly above the vulnerability following the following format,
  so readers understand what this is and why it's here:
  * Vuln: (name or names)
  * Reason: (why are you adding this?)
  * Source: (what are the sources of this data?)
* Fill out each field in the `Vulnerability` struct defined in [manual.go](manual.go).
  * See current entries for examples
* It is **required** to set the link to the source of the CVSS score unless a convincing argument may be made, otherwise.
  * It is very likely the main source of the data is the same as the source of the CVSS score, anyway.
  * Note: OSV may be the main source of the data, but many times the data is from or at least matches NVD's data.
    In this case, NVD is the preferred link.
  * In the case NVD does not have a native score, but they show a different CNA's score, **do not use NVD's link**.
    * Otherwise, we may incorrectly attribute the score to NVD, which is not technically true, so it should be avoided.

## Testing

The easiest way to test is to do the following:

1. Create a pull request with the `pr-update-scanner-vulns` label
2. Install StackRox somewhere and be sure to change the `vulnerabilities_url` in the matcher-confg.yaml file (or update it in the related `ConfigMap`)
3. Scan an image which should be affected by the added vulnerability.

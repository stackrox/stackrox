# 0015 - CSAF Enricher

- **Author(s):** Ross Tannenbaum
- **Created:** [2025-01-24 Fri]

## Status

Status: Accepted

## Context

Prior to 4.6.0, Scanner V4 fetched vulnerability data affecting Red Hat products from
[Red Hat's OVAL](https://security.access.redhat.com/data/oval/v2/) feeds. This data tended to take an
advisory-centric approach. That is, when a CVE was fixed for a particular product, the CVE entry related to that product
would be replaced with the associated advisory (RHSA/RHEA/RHBA). This made it difficult for us to provide 
our users with a CVE-centric vulnerability model, which we'd prefer, as CVEs are rather ubiquitous.

We switched to [Red Hat's VEX](https://security.access.redhat.com/data/csaf/v2/vex/) data in 4.6.0, as it is recommended
and is CVE-centric, as desired. Using VEX data, Scanner V4 is able to provide much greater accuracy when scanning Red Hat products.

However, the StackRox product, as a whole, was not ready for this sudden CVE-centric change for Red Hat data for various reasons
which are outside the scope of this document. So, for the 4.6.0 release, we opted to continue using VEX data
but simply change the name and link for fixed vulnerabilities affecting Red Hat products to the associated advisory,
essentially making it look like there was no change in data sources. This change was done in [this PR](https://github.com/stackrox/stackrox/pull/13052).

When testing StackRox 4.6.0, we found RHSA data would be inconsistent. After further investigation, it became clear
the culprit was the quick (and perhaps under-tested) patch done in the PR referenced above. That PR
solely swapped the vulnerability's name and link, but kept everything else the same. This is problematic. One reason
is stated below:

* Advisories may resolve several CVEs for a particular product. So, if a package in an image is affected by more than
  one of these CVEs which are resolved by the same advisory, then Scanner V4 would output the same advisory multiple times
  with different descriptions, CVSS scores, severities.

[A change](https://github.com/stackrox/stackrox/pull/13559) to alleviate this concern has already been merged and ported to the 4.6.1 release.
This change definitely improves the situation (the score can usually only increase and only decrease for certain edge cases),
but it's not perfect, as the score can still change. One example of how this may happen follows:

* A package in an image is affected by some vulnerability, CVE-2025-1234, which is resolved by RHSA-2025:1234. This CVE
  has a severity of `Important`, so StackRox Central will store RHSA-2025:1234 with severity `Important`.
* The same package in a different image is affected by a different vulnerability, CVE-2025-12345, which is resolved by
  RHSA-2025:1234, as well. This CVE has a severity of `Moderate`, so StackRox Central will now overwrite the severity
  it previously stored for RHSA-2025:1234 with the new one, `Moderate`.

## Decision

This document outlines how we will address the Red Hat advisory inconsistencies by introducing a CSAF enricher.

Red Hat offers [CSAF data](https://security.access.redhat.com/data/csaf/v2/advisories/), which is advisory-centric (ie one file per advisory).
Scanner V4 will add another enricher, `stackrox.rhel-csaf`, which will enrich Vulnerability Reports with Red Hat's CSAF data.

The enricher will fetch Red Hat advisories and extract data we have determined has been inconsistent in current StackRox 4.6 releases:

* Description
  * The current implementation takes the description from the CVE, so if a package is affected by two different CVEs
    associated with the same advisory, then there is a clear inconsistency, as it is unclear which description may be shown.
    The CSAF enricher will use the advisory's "title" as the description.
* Published Date
  * The current implementation takes the published date from the CVE, so if a package is affected by two different CVEs
    associated with the same advisory, then there is a clear inconsistency, as it is unclear which published date may be shown.
    The CSAF enricher will use the advisory's initial release date as the published date.
* Severity
  * The current implementation takes the severity from the CVE, so if a package is affected by two different CVEs
    rated with different severities, then there is a clear inconsistency, as it is unclear which severity may be shown.
    The CSAF enricher will use the advisory's aggregate severity field as the severity.
  * Note: it is very possible a CVE has two different severity ratings, depending on the product.
    For example: https://access.redhat.com/security/cve/CVE-2023-3899 is rated Important, in general,
    but Moderate for subscription-manager in RHEL 7. For this case, the OVAL v2 entry in 
    https://security.access.redhat.com/data/oval/v2/RHEL7/rhel-7-including-unpatched.oval.xml.bz2
    for the associated RHSA, RHSA-2023:4701, actually has the expected, correct rating of Moderate.
    The CSAF entry in https://security.access.redhat.com/data/csaf/v2/advisories/2023/rhsa-2023_4701.json
    also lists the Moderate severity rating under CVE-2023-3899's "threats" entry.
* CVSS vectors and scores
  * Red Hat advisories are not given a CVSS score, so we calculate it as done prior to 4.6.0:
    * Pick the highest CVSS scores (v3 and v2) from the associated CVEs.
    * Note: it is very possible a CVE has two different CVSS scores, depending on the product.
      For example: https://access.redhat.com/security/cve/CVE-2023-3899 is scored 7.8, in general,
      but 6.1 for subscription-manager in RHEL 7. For this case, the OVAL v2 entry in 
      https://security.access.redhat.com/data/oval/v2/RHEL7/rhel-7-including-unpatched.oval.xml.bz2
      for the associated RHSA, RHSA-2023:4701, actually has the general CVSS score (7.8) instead of the true score (6.1).
      Meanwhile, the CSAF entry in https://security.access.redhat.com/data/csaf/v2/advisories/2023/rhsa-2023_4701.json
      lists the true, correct score of 6.1. So, the output we get here will differ from the previous OVAL v2-based output, 
      but it will be more accurate.
    * Note: we will not run into a case where an advisory has two different scores for the same CVE.
      CVEs are given an overall score which may be overridden for specific product. Since advisories
      are per-product, a single advisory cannot be associated with scores related to other products.

If we discover other fields have shown inconsistencies, we will include them, as-needed.

The relevant fields will be replaced with the enricher's data for any vulnerability renamed to a Red Hat advisory.

Adding this data to Scanner V4 is backwards compatible, so we will not need to bump the vulnerability version up (currently at v2).

## Consequences

* Though the change is backwards compatible, it will add this data to Scanner V4 versions relying on vulnerability v2 data
  even if they do not utilize this data. This added data is expected to be rather negligible, though.
* Should we decide to remove this enricher in the future, we will have to implement a migration.
  * Because this enricher is not native to Claircore, we may not want to do the traditional method of creating
    a `.sql` file and adding it to the slice of matcher migrations, as to not interfere with native Claircore migrations.
    One option would be to just run the deletion query upon Scanner V4 Matcher startup after the usual migrations
    but before anything else.

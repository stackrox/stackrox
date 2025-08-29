# 0017 - VEX Enricher

- **Author(s):** Ross Tannenbaum
- **Created:** [2025-07-03 Tue]

## Status

Status: Accepted

## Context

In 4.8.0, we introduced [the ability for Scanner V4 users to ignore non-Red Hat data when scanning layers found in Red Hat images](https://github.com/stackrox/stackrox/pull/14219).
A big reason to enable this feature is to eliminate the noise of false-positives.

[Red Hat's VEX data](https://redhatproductsecurity.github.io/security-data-guidelines/csaf-vex/) includes a product fix status called `known_not_affected`.
This is set for Red Hat products which Red Hat knows, after internal investigation, are unaffected by the vulnerability in question.
This also includes [third-party vulnerabilities](https://access.redhat.com/articles/5554431) which Red Hat started to track as of February 3, 2025.
This case is indicated by listing the `product_status` of `known_not_affected` with solely `red_hat_products` (ex: [CVE-2024-55918](https://security.access.redhat.com/data/csaf/v2/vex/2024/cve-2024-55918.json)).
For other cases, where Red Hat products might be affected, the Red Hat products known to be unaffected by the vulnerability
are explicitly listed (ex: [CVE-2021-42392](https://security.access.redhat.com/data/csaf/v2/vex/2021/cve-2021-42392.json)).
As of StackRox 4.8.0, Scanner V4/Claircore do not track the `known_not_affected` status. So, there are many times
when Red Hat claims one of their images is unaffected by a vulnerability, but StackRox says otherwise, as we
fetch data from multiple data sources. Typically, we find our language matchers will find some vulnerability from [OSV.dev](https://osv.dev/list) in this case.

With this feature enabled, when we scan Red Hat images built via the older, legacy build system, we stop using non-Red Hat data sources for packages found in Red Hat layers.
This minimizes false-positives as we will now stop matching vulnerabilities against OSV data (for example). In a way, this mimics the 
effect of us actually reading and considering the `known_not_affected` status from Red Hat's VEX files.

This feature is currently disabled by default, however, as there are caveats to enabling it:

1. It only works for images built with Red Hat's older, legacy build system. A large majority of the images in Red Hat's container image catalog
   are built with this system; however, newer images have been and will continue to be built with a newer build system. The [version of Claircore](https://github.com/quay/claircore/tree/v1.5.38) used in 4.8.0
   can only be used to identify layers in the older Red Hat images, so this solution does not work for these newer images.
   Handling this case is outside the scope of this document.
2. Enabling this feature introduces potential false-negatives. There are known gaps in Red Hat's VEX data,
   so only using Red Hat data may mean StackRox may suffer from false-negatives. For example, [not all Middleware is included](https://access.redhat.com/security/middleware_security_scanning_problem).
   Similarly, The VEX data only tracks Red Hat products which are supported at a single point in time.
   So, Red Hat's VEX data does not track older, unsupported products nor does it track newer products. It also does not track
   versions of pre-existing products after the point in time in which Red Hat first tracks the vulnerability.

A better solution would ideally:

* Ignore/hide non-Red Hat data if there is VEX data that explicitly states the image in question is not vulnerable to the vulnerability (i.e. read the `known_not_affected` status)
* Show non-Red Hat data (i.e., OSV) when there is no VEX data relating the image to the vulnerability

This will help us minimize false-negatives while keeping our false-positive mitigation introduced with the previously mentioned feature.

## Decision

[Claircore](https://github.com/quay/claircore/blob/v1.5.38/rhel/vex/updater.go) already utilizes Red Hat’s VEX data for vulnerabilities; however, we cannot really modify the 
current usage to support our use case. Instead, we will need to create a VEX Enricher. Creating yet-another-enricher is not ideal, 
and may not align with any plans to update the Matcher's database schema; however, it is all we can do at this very moment.

Our main goal for this ADR is to minimize false-positives and false-negatives, when it comes to Red Hat products, as much as possible.
This may be done by now considering the `known_not_affected` product status.

The enricher will parse each Red Hat VEX file for all products listed in each `known_not_affected`.
The enricher will track the listed products, including the cases where it solely lists `red_hat_products` (i.e. when the CVE is known to not affect any Red Hat product).
Scanner V4 will use this data filter out vulnerabilities in the VulnerabilityReport which Red Hat has explicitly stated do not affect the image in question.

TODO: we can already find Red Hat layers. I'm thinking we just need to recognize which packages identify the image (i.e. which packages originated from the RHCC package scanner).
Once we have that, we can check the `known_not_affected` list for each RHCC package (or the presence of `red_hat_products`) for each package found in a Red Hat layer.
If found, then filter that vuln out. Idea: may not need to consider RPMs, but rather solely language-specific stuff. Unsure how terrible this solution will be for performance.

## Consequences

* It is possible Red Hat's VEX data does not track a particular vulnerability as it affects an image until some later time (i.e. Red Hat's data is behind).
  * In this scenario, it is possible we show non-Red Hat data until the VEX data does, eventually, account for it.
  * The vulnerability data may somewhat suddenly change without much notice nor warning.
  * We accept this, but acknowledge it may cause confusion.
* Red Hat's VEX data does not track every single version of every single product, so it's possible there is no VEX data for a particular image.
  * Without explicit data, we cannot know for sure if Red Hat knows if a product is truly affected by a vulnerability or not.
  * After discussions, it was decided this is truly a data problem, and not ours. That is, if there is no data from Red Hat 
    about some product or version of a product, then the vulnerability scanner is not at fault for any false-positive or false-negative finding.
    Any issues seen due to lack of data is to be brought over to Red Hat's Product Security team.

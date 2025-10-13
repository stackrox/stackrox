# 0007 - Red Hat CVSS scores

- **Author(s):** Ross Tannenbaum
- **Created:** [2023-12-15 Fri]

## Context

[ClairCore](https://github.com/quay/claircore) provides us with CVSS scores from NVD in the form of a Vulnerability Report enrichment.
These scores may be inaccurate for Red Hat products, as Red Hat analyzes each CVE and, potentially,
reassigns the CVSS score. For example: [CVE-2021-26291](https://access.redhat.com/security/cve/CVE-2021-26291).

There is currently work being done in ClairCore to [adopt CSAF and VEX files for Red Hat vulnerability data](https://www.redhat.com/en/blog/vulnerability-exploitability-exchange-vex-beta-files-now-available),
which will deprecate the [OVAL v2 feeds](https://access.redhat.com/security/data/oval/v2/). Though there are plans to
expose Red Hat's CVSS scores once the adoption is complete, this work is not expected to be completed until after
the StackRox 4.4 release is cut. So, we need a temporary solution.

## Decision

We want to minimize any divergences we have with upstream ClairCore, which means we do **not** want to fork
the ClairCore repository. That being said, forking the repository is the simplest way to proceed here.
Instead of a true, git fork, however, we opt to copy over the necessary files into the `scanner/` directory.
This simplifies things, as we will not need to manage an entire forked repository. For the remainder of this document,
we will still refer to this process as a fork.

We know there is already an effort to replace the current [Red Hat vulnerability updater](https://github.com/quay/claircore/blob/v1.5.20/rhel/updaterset.go)
with an implementation which supports CSAF and VEX files, and we know Red Hat Product Security is definitely moving away from
the OVAL v2 feeds in favor of CSAF and VEX. Therefore, we are assured this is a **temporary** fork of the ClairCore repository.

This fork provides us with the simplest path forward. There are no plans to create a Red Hat CVSS score enricher in
upstream ClairCore at this time, and creating our own out-of-tree enricher will be time-consuming and tricky, as
CVSS scores are potentially also product-dependent instead of just CVE-specific.

The simplest path forward is to utilize a field of the [VulnerabilityReport](https://github.com/quay/claircore/blob/v1.5.20/vulnerabilityreport.go)
which Scanner v4/StackRox does not plan on utilizing at this time: `Severity` (not to be mistaken for `NormalizedSeverity`).
This idea was conceived a few months back, but was [rejected from upstream ClairCore](https://github.com/quay/claircore/pull/919),
as the team did not want to change the semantics of this field which may be used by consumers. For Scanner v4/StackRox,
we know we are not using this field, so we do not mind overwriting it. As can be seen from the original pull request,
implementing this is rather straightforward and quick to do.

There is already work towards this effort [here](https://github.com/stackrox/stackrox/pull/9112).

Again, we expect this to be temporary, as we know there is work in-progress to adopt CSAF and VEX files in favor of OVAL v2.

Once ClairCore implements CSAF and VEX support, our fork will be deleted in favor of using the new upstream updater.

## Consequences

There is always a risk this fork becomes more permanent than expected/desired. There is a very strong drive to ensure
differences between upstream Clair/ClairCore and Scanner v4 are minimal, so should this happen, corrective action will
need to be taken.

The main annoyance is the requirement to maintain a fork of the upstream ClairCore repository (even though it's a subset).
Any changes/fixes to ClairCore's current pre-VEX version will need to be ported to our repository.

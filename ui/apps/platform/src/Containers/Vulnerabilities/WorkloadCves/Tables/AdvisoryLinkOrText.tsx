import React, { ReactNode } from 'react';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';

// https://access.redhat.com/security/updates/advisory
// Red Hat publishes three types of errata:
// Red Hat Security Advisory (RHSA)
// Red Hat Bug Advisory (RHBA)
// Red Hat Enhancement Advisory (RHEA)

// https://access.redhat.com/articles/explaining_redhat_errata
// All advisories are given a year and a sequential number, which starts at 0001 and ends at the number of advisories shipped for that year.

export function isRedHatAdvisory(advisory: string) {
    return /^RH[SBE]A-\d\d\d\d:\d+$/.test(advisory);
}

export type AdvisoryLinkOrTextProps = {
    advisory: string | undefined;
};

function AdvisoryLinkOrText({ advisory }: AdvisoryLinkOrTextProps): ReactNode {
    if (typeof advisory === 'string') {
        if (isRedHatAdvisory(advisory)) {
            return (
                <ExternalLink>
                    <a
                        href={`https://access.redhat.com/errata/${advisory}`}
                        target="_blank"
                        rel="noopener noreferrer"
                    >
                        {advisory}
                    </a>
                </ExternalLink>
            );
        }

        // Unexpected, because other advisories like GHSA are not separated from CVE.
        if (advisory.length !== 0) {
            return advisory;
        }
    }

    return '-';
}

export default AdvisoryLinkOrText;

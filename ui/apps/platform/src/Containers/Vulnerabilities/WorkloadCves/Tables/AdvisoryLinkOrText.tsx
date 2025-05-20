import React, { ReactNode } from 'react';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import { Advisory } from 'types/cve.proto';

// https://access.redhat.com/security/updates/advisory
// Red Hat publishes three types of errata:
// Red Hat Security Advisory (RHSA)
// Red Hat Bug Advisory (RHBA)
// Red Hat Enhancement Advisory (RHEA)

// https://access.redhat.com/articles/explaining_redhat_errata
// All advisories are given a year and a sequential number, which starts at 0001 and ends at the number of advisories shipped for that year.

export type AdvisoryLinkOrTextProps = {
    advisory: Advisory | null | undefined;
};

function AdvisoryLinkOrText({ advisory }: AdvisoryLinkOrTextProps): ReactNode {
    if (advisory) {
        const { name: advisoryId, link: advisoryLink } = advisory;
        return (
            <ExternalLink>
                <a href={advisoryLink} target="_blank" rel="noopener noreferrer">
                    {advisoryId}
                </a>
            </ExternalLink>
        );
    }

    return '-';
}

export default AdvisoryLinkOrText;

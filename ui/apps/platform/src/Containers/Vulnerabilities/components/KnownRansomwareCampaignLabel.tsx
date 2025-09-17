import React from 'react';

import { Label, Tooltip } from '@patternfly/react-core';

const tooltip =
    'This CVE has been exploited in a known ransomware campaign. CVEs with this label should be addressed with high priority due to the risks posed by them. The existence of this label does not mean we have taken steps to determine if the CVE has been exploited in your environment.';

export type KnownRansomwareCampaignLabelProps = {
    isCompact: boolean; // true for table and false for vulnerability page
};

function KnownRansomwareCampaignLabel({ isCompact }: KnownRansomwareCampaignLabelProps) {
    return (
        <Tooltip content={tooltip} position="top-start" isContentLeftAligned>
            <Label color="red" isCompact={isCompact}>
                Known ransomware campaign
            </Label>
        </Tooltip>
    );
}

export default KnownRansomwareCampaignLabel;

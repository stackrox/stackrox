import React, { ReactNode } from 'react';

import { severityLabels } from 'messages/common';
import { VulnerabilitySeverity } from 'types/cve.proto';
import { PolicySeverity } from 'types/policy.proto';

import SeverityIcons from '../SeverityIcons';

import IconText from './IconText';

// TODO import PolicySeverityIcons instead
const policyToVulnerabilitySeverity: Record<PolicySeverity, VulnerabilitySeverity> = {
    LOW_SEVERITY: 'LOW_VULNERABILITY_SEVERITY',
    MEDIUM_SEVERITY: 'MODERATE_VULNERABILITY_SEVERITY',
    HIGH_SEVERITY: 'IMPORTANT_VULNERABILITY_SEVERITY',
    CRITICAL_SEVERITY: 'CRITICAL_VULNERABILITY_SEVERITY',
};

export type PolicySeverityIconTextProps = {
    policySeverity: PolicySeverity;
    isTextOnly?: boolean;
};

function PolicySeverityIconText({
    policySeverity,
    isTextOnly,
}: PolicySeverityIconTextProps): ReactNode {
    const Icon = SeverityIcons[policyToVulnerabilitySeverity[policySeverity]];
    const text = severityLabels[policySeverity];

    return <IconText Icon={<Icon />} text={text} isTextOnly={isTextOnly} />;
}

export default PolicySeverityIconText;

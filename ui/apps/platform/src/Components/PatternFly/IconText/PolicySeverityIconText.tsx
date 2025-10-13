import React, { ReactElement } from 'react';

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
    severity: PolicySeverity;
    isTextOnly?: boolean;
};

function PolicySeverityIconText({
    severity,
    isTextOnly,
}: PolicySeverityIconTextProps): ReactElement {
    const Icon = SeverityIcons[policyToVulnerabilitySeverity[severity]];
    const text = severityLabels[severity];

    return <IconText icon={<Icon />} text={text} isTextOnly={isTextOnly} />;
}

export default PolicySeverityIconText;

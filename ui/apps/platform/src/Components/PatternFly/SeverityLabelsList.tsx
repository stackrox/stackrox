import React, { ReactElement } from 'react';
import { Label } from '@patternfly/react-core';

import { vulnerabilitySeverityLabels } from 'constants/reportConstants';
import { VulnerabilitySeverity } from 'types/cve.proto';

export type SeverityLabelsListProps = {
    severities: VulnerabilitySeverity[];
};

const vulnerabilitySeverityLabelColors = {
    CRITICAL_VULNERABILITY_SEVERITY: 'red',
    IMPORTANT_VULNERABILITY_SEVERITY: 'orange',
    MODERATE_VULNERABILITY_SEVERITY: 'gold',
    LOW_VULNERABILITY_SEVERITY: 'grey',
} as const;

function SeverityLabelsList({ severities }: SeverityLabelsListProps): ReactElement {
    if (severities?.length > 0) {
        const severityLabels = severities.map((fixValue) => (
            <Label
                className="pf-u-mr-sm"
                color={vulnerabilitySeverityLabelColors[fixValue]}
                isCompact
            >
                {vulnerabilitySeverityLabels[fixValue]}
            </Label>
        ));

        return <>{severityLabels}</>;
    }

    return (
        <span>
            <em>No severities specified</em>
        </span>
    );
}

export default SeverityLabelsList;

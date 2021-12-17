import React, { ReactElement } from 'react';
import { Label } from '@patternfly/react-core';

import { vulnerabilitySeverityLabels } from 'constants/reportConstants';
import { VulnerabilitySeverity } from 'types/cve.proto';

export type SeverityLabelsListProps = {
    severities: VulnerabilitySeverity[];
};

function SeverityLabelsList({ severities }: SeverityLabelsListProps): ReactElement {
    if (severities?.length > 0) {
        const severityLabels = severities.map((fixValue) => (
            <Label className="pf-u-mr-sm" color="red" isCompact>
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

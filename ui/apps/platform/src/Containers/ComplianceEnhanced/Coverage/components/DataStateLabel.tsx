import { Label } from '@patternfly/react-core';
import { ExclamationTriangleIcon } from '@patternfly/react-icons';

import type { ComplianceDataState } from 'services/ComplianceCommon';

function DataStateLabel({ dataState }: { dataState?: ComplianceDataState }) {
    if (dataState === 'COMPLIANCE_DATA_STATE_OUTDATED') {
        return (
            <Label color="orange" icon={<ExclamationTriangleIcon />}>
                Outdated
            </Label>
        );
    }
    if (dataState === 'COMPLIANCE_DATA_STATE_CURRENT') {
        return <Label color="green">Current</Label>;
    }
    if (dataState === 'COMPLIANCE_DATA_STATE_UNKNOWN') {
        // Cannot evaluate freshness (no schedule, one-time scan, unresolved config, or
        // missing timestamp). Render an explicit label so the cell is not ambiguously blank.
        return <Label color="grey">Unknown</Label>;
    }
    return null;
}

export default DataStateLabel;

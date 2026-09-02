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
    return null;
}

export default DataStateLabel;

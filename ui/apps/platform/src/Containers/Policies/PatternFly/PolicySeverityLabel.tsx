import React, { ReactElement } from 'react';
import { Label } from '@patternfly/react-core';

import { severityColorMapPF } from 'constants/severityColors';
import { severityLabels } from 'messages/common';
import { PolicySeverity } from 'types/policy.proto';

type PolicySeverityLabelProps = {
    severity: PolicySeverity;
};

function PolicySeverityLabel({ severity }: PolicySeverityLabelProps): ReactElement {
    const severityLabel = severityLabels[severity];
    return <Label color={severityColorMapPF[severityLabel]}>{severityLabel}</Label>;
}

export default PolicySeverityLabel;

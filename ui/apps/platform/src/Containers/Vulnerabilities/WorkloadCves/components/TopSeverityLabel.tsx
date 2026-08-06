import { Label } from '@patternfly/react-core';

import SeverityIcons from 'Components/PatternFly/SeverityIcons';
import { vulnerabilitySeverityLabels } from 'messages/common';
import type { VulnerabilitySeverity } from 'types/cve.proto';

type TopSeverityLabelProps = {
    severity: VulnerabilitySeverity | undefined;
};

function TopSeverityLabel({ severity }: TopSeverityLabelProps) {
    const resolvedSeverity = severity ?? 'UNKNOWN_VULNERABILITY_SEVERITY';
    const Icon = SeverityIcons[resolvedSeverity];
    const label = vulnerabilitySeverityLabels[resolvedSeverity];

    return (
        <Label icon={<Icon />} variant="outline">
            {label}
        </Label>
    );
}

export default TopSeverityLabel;

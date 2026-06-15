import { Label } from '@patternfly/react-core';

import SeverityIcons from 'Components/PatternFly/SeverityIcons';

import type { VulnerabilitySeverityLabel } from '../types';
import { isVulnerabilitySeverityLabel } from '../types';
import { severityLabelToSeverity } from '../utils/searchUtils';

export function severityLabelFromCvss(cvss: number): VulnerabilitySeverityLabel {
    if (cvss >= 9.0) {
        return 'Critical';
    }
    if (cvss >= 7.0) {
        return 'Important';
    }
    if (cvss >= 4.0) {
        return 'Moderate';
    }
    if (cvss > 0) {
        return 'Low';
    }
    return 'Unknown';
}

type TopSeverityLabelProps =
    | { severity: string; cvss?: never }
    | { severity?: never; cvss: number };

function TopSeverityLabel(props: TopSeverityLabelProps) {
    let label: VulnerabilitySeverityLabel;
    if (props.severity !== undefined) {
        label = isVulnerabilitySeverityLabel(props.severity) ? props.severity : 'Unknown';
    } else {
        label = severityLabelFromCvss(props.cvss);
    }

    const severityKey = severityLabelToSeverity(label);
    const SeverityIcon = SeverityIcons[severityKey];

    return (
        <Label variant="outline" icon={<SeverityIcon />}>
            {label}
        </Label>
    );
}

export default TopSeverityLabel;

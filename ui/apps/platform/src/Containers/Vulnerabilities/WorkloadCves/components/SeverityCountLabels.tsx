import React from 'react';
import { Flex, Label } from '@patternfly/react-core';

import SeverityIcons from 'Components/PatternFly/SeverityIcons';
import { vulnSeverityTextColors } from 'constants/visuals/colors';

const fadedTextColor = 'var(--pf-global--Color--200)';

type SeverityCountLabelsProps = {
    critical: number;
    important: number;
    moderate: number;
    low: number;
};

function SeverityCountLabels({ critical, important, moderate, low }: SeverityCountLabelsProps) {
    const CriticalIcon = SeverityIcons.CRITICAL_VULNERABILITY_SEVERITY;
    const ImportantIcon = SeverityIcons.IMPORTANT_VULNERABILITY_SEVERITY;
    const ModerateIcon = SeverityIcons.MODERATE_VULNERABILITY_SEVERITY;
    const LowIcon = SeverityIcons.LOW_VULNERABILITY_SEVERITY;

    return (
        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
            <Label
                variant="outline"
                className="pf-u-font-weight-bold"
                icon={<CriticalIcon color={critical ? undefined : fadedTextColor} />}
            >
                <span
                    style={{
                        color: critical
                            ? vulnSeverityTextColors.CRITICAL_VULNERABILITY_SEVERITY
                            : fadedTextColor,
                    }}
                >
                    {critical}
                </span>
            </Label>
            <Label
                variant="outline"
                className="pf-u-font-weight-bold"
                icon={<ImportantIcon color={important ? undefined : fadedTextColor} />}
            >
                <span
                    style={{
                        color: important
                            ? vulnSeverityTextColors.IMPORTANT_VULNERABILITY_SEVERITY
                            : fadedTextColor,
                    }}
                >
                    {important}
                </span>
            </Label>
            <Label
                variant="outline"
                className="pf-u-font-weight-bold"
                icon={<ModerateIcon color={moderate ? undefined : fadedTextColor} />}
            >
                <span
                    style={{
                        color: moderate
                            ? vulnSeverityTextColors.MODERATE_VULNERABILITY_SEVERITY
                            : fadedTextColor,
                    }}
                >
                    {moderate}
                </span>
            </Label>
            <Label
                variant="outline"
                className="pf-u-font-weight-bold"
                icon={<LowIcon color={low ? undefined : fadedTextColor} />}
            >
                <span
                    style={{
                        color: low
                            ? vulnSeverityTextColors.LOW_VULNERABILITY_SEVERITY
                            : fadedTextColor,
                    }}
                >
                    {low}
                </span>
            </Label>
        </Flex>
    );
}

export default SeverityCountLabels;

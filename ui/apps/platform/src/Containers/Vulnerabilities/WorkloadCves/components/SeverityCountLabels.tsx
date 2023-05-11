import React from 'react';
import { Flex, Label, Tooltip, pluralize } from '@patternfly/react-core';

import SeverityIcons from 'Components/PatternFly/SeverityIcons';
import { vulnSeverityTextColors } from 'constants/visuals/colors';

const fadedTextColor = 'var(--pf-global--Color--200)';

type SeverityCountLabelsProps = {
    critical: number;
    important: number;
    moderate: number;
    low: number;
    entity?: string;
};

function getTooltipContent(severityCount: number, severity: string, entity?: string) {
    if (entity) {
        return `${pluralize(severityCount, `${severity} CVE`)} across this ${entity}`;
    }
    return `${pluralize(severityCount, 'image')} with ${severity} CVEs`;
}

function SeverityCountLabels({
    critical,
    important,
    moderate,
    low,
    entity,
}: SeverityCountLabelsProps) {
    const CriticalIcon = SeverityIcons.CRITICAL_VULNERABILITY_SEVERITY;
    const ImportantIcon = SeverityIcons.IMPORTANT_VULNERABILITY_SEVERITY;
    const ModerateIcon = SeverityIcons.MODERATE_VULNERABILITY_SEVERITY;
    const LowIcon = SeverityIcons.LOW_VULNERABILITY_SEVERITY;

    return (
        <Flex spaceItems={{ default: 'spaceItemsSm' }} flexWrap={{ default: 'nowrap' }}>
            <Tooltip content={getTooltipContent(critical, 'critical', entity)}>
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
            </Tooltip>
            <Tooltip content={getTooltipContent(important, 'important', entity)}>
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
            </Tooltip>
            <Tooltip content={getTooltipContent(moderate, 'moderate', entity)}>
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
            </Tooltip>
            <Tooltip content={getTooltipContent(low, 'low', entity)}>
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
            </Tooltip>
        </Flex>
    );
}

export default SeverityCountLabels;

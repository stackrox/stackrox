import React from 'react';
import { Flex, Label, Tooltip, pluralize, capitalize } from '@patternfly/react-core';
import { EllipsisHIcon } from '@patternfly/react-icons';

import SeverityIcons from 'Components/PatternFly/SeverityIcons';
import { vulnSeverityTextColors } from 'constants/visuals/colors';
import { VulnerabilitySeverityLabel } from '../types';

const fadedTextColor = 'var(--pf-global--Color--200)';

type SeverityCountLabelsProps = {
    criticalCount: number;
    importantCount: number;
    moderateCount: number;
    lowCount: number;
    entity?: string;
    filteredSeverities?: VulnerabilitySeverityLabel[];
};

function getTooltipContent(severity: string, severityCount?: number, entity?: string) {
    if (!severityCount && severityCount !== 0) {
        return `${capitalize(severity)} severity is hidden by the applied filter`;
    }
    if (entity) {
        return `${pluralize(severityCount, `${severity} severity CVE`)} across this ${entity}`;
    }
    return `${pluralize(severityCount, 'image')} with ${severity} severity`;
}

function SeverityCountLabels({
    criticalCount,
    importantCount,
    moderateCount,
    lowCount,
    entity,
    filteredSeverities,
}: SeverityCountLabelsProps) {
    const CriticalIcon = SeverityIcons.CRITICAL_VULNERABILITY_SEVERITY;
    const ImportantIcon = SeverityIcons.IMPORTANT_VULNERABILITY_SEVERITY;
    const ModerateIcon = SeverityIcons.MODERATE_VULNERABILITY_SEVERITY;
    const LowIcon = SeverityIcons.LOW_VULNERABILITY_SEVERITY;

    const isCriticalHidden = !!filteredSeverities && !filteredSeverities.includes('Critical');
    const isImportantHidden = !!filteredSeverities && !filteredSeverities.includes('Important');
    const isModerateHidden = !!filteredSeverities && !filteredSeverities.includes('Moderate');
    const isLowHidden = !!filteredSeverities && !filteredSeverities.includes('Low');

    const critical = isCriticalHidden ? undefined : criticalCount;
    const important = isImportantHidden ? undefined : importantCount;
    const moderate = isModerateHidden ? undefined : moderateCount;
    const low = isLowHidden ? undefined : lowCount;

    return (
        <Flex spaceItems={{ default: 'spaceItemsSm' }} flexWrap={{ default: 'nowrap' }}>
            <Tooltip content={getTooltipContent('critical', critical, entity)}>
                <Label
                    aria-label={getTooltipContent('critical', critical, entity)}
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
                        {!critical && critical !== 0 ? (
                            <EllipsisHIcon className="pf-u-my-xs" />
                        ) : (
                            critical
                        )}
                    </span>
                </Label>
            </Tooltip>
            <Tooltip content={getTooltipContent('important', important, entity)}>
                <Label
                    aria-label={getTooltipContent('important', important, entity)}
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
                        {!important && important !== 0 ? (
                            <EllipsisHIcon className="pf-u-my-xs" />
                        ) : (
                            important
                        )}
                    </span>
                </Label>
            </Tooltip>
            <Tooltip content={getTooltipContent('moderate', moderate, entity)}>
                <Label
                    aria-label={getTooltipContent('moderate', moderate, entity)}
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
                        {!moderate && moderate !== 0 ? (
                            <EllipsisHIcon className="pf-u-my-xs" />
                        ) : (
                            moderate
                        )}
                    </span>
                </Label>
            </Tooltip>
            <Tooltip content={getTooltipContent('low', low, entity)}>
                <Label
                    aria-label={getTooltipContent('low', low, entity)}
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
                        {!low && low !== 0 ? <EllipsisHIcon className="pf-u-my-xs" /> : low}
                    </span>
                </Label>
            </Tooltip>
        </Flex>
    );
}

export default SeverityCountLabels;

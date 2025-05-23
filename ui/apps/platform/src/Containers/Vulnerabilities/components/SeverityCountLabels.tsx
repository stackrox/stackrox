import React from 'react';
import { Flex, Label, Tooltip, capitalize } from '@patternfly/react-core';
import { EllipsisHIcon } from '@patternfly/react-icons';

import SeverityIcons from 'Components/PatternFly/SeverityIcons';
import { noViolationsClassName, noViolationsColor } from 'constants/severityColors';

import { VulnerabilitySeverityLabel } from '../types';

import './SeverityCountLabels.css';

type SeverityCountLabelsProps = {
    criticalCount: number;
    importantCount: number;
    moderateCount: number;
    lowCount: number;
    unknownCount: number;
    entity?: string;
    filteredSeverities?: VulnerabilitySeverityLabel[];
};

function getTooltipContent(severity: string, severityCount?: number, entity?: string) {
    if (!severityCount && severityCount !== 0) {
        return `${capitalize(severity)} severity is hidden by the applied filter`;
    }
    if (entity) {
        return `${severityCount} ${severity} severity cve count across this ${entity}`;
    }
    return `Image count with ${severity} severity`;
}

function getClassNameForCount(count?: number) {
    // Render non-zero count in normal black versus zero (or undefined) count in gray.
    return count ? '' : noViolationsClassName;
}

function SeverityCountLabels({
    criticalCount,
    importantCount,
    moderateCount,
    lowCount,
    unknownCount,
    entity,
    filteredSeverities,
}: SeverityCountLabelsProps) {
    const CriticalIcon = SeverityIcons.CRITICAL_VULNERABILITY_SEVERITY;
    const ImportantIcon = SeverityIcons.IMPORTANT_VULNERABILITY_SEVERITY;
    const ModerateIcon = SeverityIcons.MODERATE_VULNERABILITY_SEVERITY;
    const LowIcon = SeverityIcons.LOW_VULNERABILITY_SEVERITY;
    const UnknownIcon = SeverityIcons.UNKNOWN_VULNERABILITY_SEVERITY;

    const isCriticalHidden = !!filteredSeverities && !filteredSeverities.includes('Critical');
    const isImportantHidden = !!filteredSeverities && !filteredSeverities.includes('Important');
    const isModerateHidden = !!filteredSeverities && !filteredSeverities.includes('Moderate');
    const isLowHidden = !!filteredSeverities && !filteredSeverities.includes('Low');
    const isUnknownHidden = !!filteredSeverities && !filteredSeverities.includes('Unknown');

    const critical = isCriticalHidden ? undefined : criticalCount;
    const important = isImportantHidden ? undefined : importantCount;
    const moderate = isModerateHidden ? undefined : moderateCount;
    const low = isLowHidden ? undefined : lowCount;
    const unknown = isUnknownHidden ? undefined : unknownCount;

    return (
        <Flex
            className="severity-count-labels"
            spaceItems={{ default: 'spaceItemsSm' }}
            flexWrap={{ default: 'nowrap' }}
        >
            <Tooltip content={getTooltipContent('critical', critical, entity)}>
                <Label
                    aria-label={getTooltipContent('critical', critical, entity)}
                    variant="outline"
                    icon={<CriticalIcon color={critical ? undefined : noViolationsColor} />}
                >
                    <span className={getClassNameForCount(critical)}>
                        {!critical && critical !== 0 ? <EllipsisHIcon /> : critical}
                    </span>
                </Label>
            </Tooltip>
            <Tooltip content={getTooltipContent('important', important, entity)}>
                <Label
                    aria-label={getTooltipContent('important', important, entity)}
                    variant="outline"
                    icon={<ImportantIcon color={important ? undefined : noViolationsColor} />}
                >
                    <span className={getClassNameForCount(important)}>
                        {!important && important !== 0 ? <EllipsisHIcon /> : important}
                    </span>
                </Label>
            </Tooltip>
            <Tooltip content={getTooltipContent('moderate', moderate, entity)}>
                <Label
                    aria-label={getTooltipContent('moderate', moderate, entity)}
                    variant="outline"
                    icon={<ModerateIcon color={moderate ? undefined : noViolationsColor} />}
                >
                    <span className={getClassNameForCount(moderate)}>
                        {!moderate && moderate !== 0 ? <EllipsisHIcon /> : moderate}
                    </span>
                </Label>
            </Tooltip>
            <Tooltip content={getTooltipContent('low', low, entity)}>
                <Label
                    aria-label={getTooltipContent('low', low, entity)}
                    variant="outline"
                    icon={<LowIcon color={low ? undefined : noViolationsColor} />}
                >
                    <span className={getClassNameForCount(low)}>
                        {!low && low !== 0 ? <EllipsisHIcon /> : low}
                    </span>
                </Label>
            </Tooltip>
            <Tooltip content={getTooltipContent('unknown', unknown, entity)}>
                <Label
                    aria-label={getTooltipContent('unknown', unknown, entity)}
                    variant="outline"
                    icon={<UnknownIcon color={unknown ? undefined : noViolationsColor} />}
                >
                    <span className={getClassNameForCount(unknown)}>
                        {!unknown && unknown !== 0 ? <EllipsisHIcon /> : unknown}
                    </span>
                </Label>
            </Tooltip>
        </Flex>
    );
}

export default SeverityCountLabels;

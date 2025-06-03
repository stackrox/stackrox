import React from 'react';
import { Card, CardBody, CardTitle, Flex, Grid, GridItem, Text } from '@patternfly/react-core';

import SeverityIcons from 'Components/PatternFly/SeverityIcons';

import { VulnerabilitySeverity } from 'types/cve.proto';
import { vulnerabilitySeverityLabels } from 'messages/common';

const severitiesDescendingCriticality = [
    'CRITICAL_VULNERABILITY_SEVERITY',
    'IMPORTANT_VULNERABILITY_SEVERITY',
    'MODERATE_VULNERABILITY_SEVERITY',
    'LOW_VULNERABILITY_SEVERITY',
    'UNKNOWN_VULNERABILITY_SEVERITY',
] as const;

const severityToQuerySeverityKeys = {
    CRITICAL_VULNERABILITY_SEVERITY: 'critical',
    IMPORTANT_VULNERABILITY_SEVERITY: 'important',
    MODERATE_VULNERABILITY_SEVERITY: 'moderate',
    LOW_VULNERABILITY_SEVERITY: 'low',
    UNKNOWN_VULNERABILITY_SEVERITY: 'unknown',
} as const;

const severityToHiddenText = {
    CRITICAL_VULNERABILITY_SEVERITY: 'Critical hidden',
    IMPORTANT_VULNERABILITY_SEVERITY: 'Important hidden',
    MODERATE_VULNERABILITY_SEVERITY: 'Moderate hidden',
    LOW_VULNERABILITY_SEVERITY: 'Low hidden',
    UNKNOWN_VULNERABILITY_SEVERITY: 'Unknown hidden',
} as const;

const fadedTextColor = 'var(--pf-v5-global--Color--200)';

export type ResourceCountsByCveSeverity = {
    critical: { total: number };
    important: { total: number };
    moderate: { total: number };
    low: { total: number };
    unknown: { total: number };
};

export type BySeveritySummaryCardProps = {
    className?: string;
    title: string;
    severityCounts: ResourceCountsByCveSeverity;
    hiddenSeverities: Set<VulnerabilitySeverity>;
};

function BySeveritySummaryCard({
    className = '',
    title,
    severityCounts,
    hiddenSeverities,
}: BySeveritySummaryCardProps) {
    return (
        <Card className={className} isCompact isFlat isFullHeight>
            <CardTitle>{title}</CardTitle>
            <CardBody>
                <Grid className="pf-v5-u-pl-sm">
                    {severitiesDescendingCriticality.map((severity) => {
                        const querySeverityKey = severityToQuerySeverityKeys[severity];
                        const count = severityCounts[querySeverityKey];
                        const isHidden = hiddenSeverities.has(severity);
                        const textColor = isHidden ? fadedTextColor : '';
                        const text = isHidden
                            ? severityToHiddenText[severity]
                            : `${count.total} ${vulnerabilitySeverityLabels[severity]}`;
                        const Icon = SeverityIcons[severity];

                        return (
                            <GridItem key={severity} span={6}>
                                <Flex
                                    className="pf-v5-u-pt-sm"
                                    spaceItems={{ default: 'spaceItemsSm' }}
                                    alignItems={{ default: 'alignItemsCenter' }}
                                >
                                    <Icon
                                        title={vulnerabilitySeverityLabels[severity]}
                                        color={isHidden ? textColor : undefined}
                                    />
                                    <Text style={{ color: textColor }}>{text}</Text>
                                </Flex>
                            </GridItem>
                        );
                    })}
                </Grid>
            </CardBody>
        </Card>
    );
}

export default BySeveritySummaryCard;

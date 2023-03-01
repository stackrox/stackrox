import React from 'react';
import { Card, CardBody, CardTitle, Grid, GridItem } from '@patternfly/react-core';

import { VulnerabilitySeverity } from 'types/cve.proto';
import { vulnerabilitySeverityLabels } from 'messages/common';

export type BySeveritySummaryCardProps = {
    title: string;
    severityCounts: Record<VulnerabilitySeverity, number>;
    hiddenSeverities: Set<VulnerabilitySeverity>;
};

const severitiesCriticalToLow = [
    'CRITICAL_VULNERABILITY_SEVERITY',
    'IMPORTANT_VULNERABILITY_SEVERITY',
    'MODERATE_VULNERABILITY_SEVERITY',
    'LOW_VULNERABILITY_SEVERITY',
] as const;

function BySeveritySummaryCard({
    title,
    severityCounts,
    hiddenSeverities,
}: BySeveritySummaryCardProps) {
    return (
        <Card>
            <CardTitle>{title}</CardTitle>
            <CardBody>
                <Grid hasGutter>
                    {severitiesCriticalToLow.map((severity) => (
                        <GridItem key={severity} span={6}>
                            {hiddenSeverities.has(severity)
                                ? 'Results hidden'
                                : `${severityCounts[severity]} ${vulnerabilitySeverityLabels[severity]}`}
                        </GridItem>
                    ))}
                </Grid>
            </CardBody>
        </Card>
    );
}

export default BySeveritySummaryCard;

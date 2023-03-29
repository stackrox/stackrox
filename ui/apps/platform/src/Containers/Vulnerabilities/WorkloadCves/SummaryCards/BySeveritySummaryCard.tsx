import React from 'react';
import { Card, CardBody, CardTitle, Flex, Grid, GridItem, Text } from '@patternfly/react-core';

import SeverityIcons from 'Components/PatternFly/SeverityIcons';

import { VulnerabilitySeverity } from 'types/cve.proto';
import { vulnerabilitySeverityLabels } from 'messages/common';
import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';
import {
    ImageVulnerabilityCounter,
    ImageVulnerabilityCounterKey,
} from '../hooks/useImageVulnerabilities';

export type BySeveritySummaryCardProps = {
    title: string;
    severityCounts: ImageVulnerabilityCounter;
    hiddenSeverities: Set<VulnerabilitySeverity>;
};

const vulnCounterToSeverity: Record<ImageVulnerabilityCounterKey, VulnerabilitySeverity> = {
    low: 'LOW_VULNERABILITY_SEVERITY',
    moderate: 'MODERATE_VULNERABILITY_SEVERITY',
    important: 'IMPORTANT_VULNERABILITY_SEVERITY',
    critical: 'CRITICAL_VULNERABILITY_SEVERITY',
} as const;

const severitiesCriticalToLow = ['critical', 'important', 'moderate', 'low'] as const;

const disabledColor100 = 'var(--pf-global--disabled-color--100)';
const disabledColor200 = 'var(--pf-global--disabled-color--200)';

function BySeveritySummaryCard({
    title,
    severityCounts,
    hiddenSeverities,
}: BySeveritySummaryCardProps) {
    return (
        <Card isCompact>
            <CardTitle>{title}</CardTitle>
            <CardBody>
                <Grid className="pf-u-pl-sm">
                    {severitiesCriticalToLow.map((severity) => {
                        const count = severityCounts[severity];
                        const hasNoResults = count.total === 0;
                        const vulnSeverity = vulnCounterToSeverity[severity];
                        const isHidden = hiddenSeverities.has(vulnSeverity);

                        let textColor = '';
                        let text = `${count.total} ${vulnerabilitySeverityLabels[vulnSeverity]}`;

                        if (isHidden) {
                            textColor = disabledColor100;
                            text = 'Results hidden';
                        } else if (hasNoResults) {
                            textColor = disabledColor200;
                            text = 'No results';
                        }

                        const Icon: React.FC<SVGIconProps> | undefined =
                            SeverityIcons[vulnSeverity];

                        return (
                            <GridItem key={vulnSeverity} span={6}>
                                <Flex
                                    className="pf-u-pt-sm"
                                    spaceItems={{ default: 'spaceItemsSm' }}
                                    alignItems={{ default: 'alignItemsCenter' }}
                                >
                                    {Icon && <Icon color={hasNoResults ? textColor : undefined} />}
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

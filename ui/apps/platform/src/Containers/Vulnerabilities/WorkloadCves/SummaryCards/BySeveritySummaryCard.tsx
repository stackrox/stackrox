import React from 'react';
import { Card, CardBody, CardTitle, Flex, Grid, GridItem, Text } from '@patternfly/react-core';

import SeverityIcons from 'Components/PatternFly/SeverityIcons';

import { VulnerabilitySeverity } from 'types/cve.proto';
import { vulnerabilitySeverityLabels } from 'messages/common';
import { SVGIconProps } from '@patternfly/react-icons/dist/esm/createIcon';

export type BySeveritySummaryCardProps = {
    className?: string;
    title: string;
    severityCounts: Omit<Record<VulnerabilitySeverity, number>, 'UNKNOWN_VULNERABILITY_SEVERITY'>;
    hiddenSeverities: Set<VulnerabilitySeverity>;
};

const severitiesCriticalToLow = [
    'CRITICAL_VULNERABILITY_SEVERITY',
    'IMPORTANT_VULNERABILITY_SEVERITY',
    'MODERATE_VULNERABILITY_SEVERITY',
    'LOW_VULNERABILITY_SEVERITY',
] as const;

const disabledColor100 = 'var(--pf-global--disabled-color--100)';
const disabledColor200 = 'var(--pf-global--disabled-color--200)';

function BySeveritySummaryCard({
    className = '',
    title,
    severityCounts,
    hiddenSeverities,
}: BySeveritySummaryCardProps) {
    return (
        <Card className={className} isCompact>
            <CardTitle>{title}</CardTitle>
            <CardBody>
                <Grid className="pf-u-pl-sm">
                    {severitiesCriticalToLow.map((severity) => {
                        const count = severityCounts[severity];
                        const hasNoResults = count === 0;
                        const isHidden = hiddenSeverities.has(severity);

                        let textColor = '';
                        let text = `${count} ${vulnerabilitySeverityLabels[severity]}`;

                        if (isHidden) {
                            textColor = disabledColor100;
                            text = 'Results hidden';
                        } else if (hasNoResults) {
                            textColor = disabledColor200;
                            text = 'No results';
                        }

                        const Icon: React.FC<SVGIconProps> | undefined = SeverityIcons[severity];

                        return (
                            <GridItem key={severity} span={6}>
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

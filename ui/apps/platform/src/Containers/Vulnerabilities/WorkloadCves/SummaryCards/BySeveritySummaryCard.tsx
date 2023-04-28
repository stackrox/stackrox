import React from 'react';
import { Card, CardBody, CardTitle, Flex, Grid, GridItem, Text } from '@patternfly/react-core';

import SeverityIcons from 'Components/PatternFly/SeverityIcons';

import { VulnerabilitySeverity } from 'types/cve.proto';
import { vulnerabilitySeverityLabels } from 'messages/common';
import { graphql } from 'generated/graphql-codegen';
import { ResourceCountsByCveSeverityFragment } from 'generated/graphql-codegen/graphql';

const severitiesCriticalToLow = [
    'CRITICAL_VULNERABILITY_SEVERITY',
    'IMPORTANT_VULNERABILITY_SEVERITY',
    'MODERATE_VULNERABILITY_SEVERITY',
    'LOW_VULNERABILITY_SEVERITY',
] as const;

const severityToQuerySeverityKeys = {
    CRITICAL_VULNERABILITY_SEVERITY: 'critical',
    IMPORTANT_VULNERABILITY_SEVERITY: 'important',
    MODERATE_VULNERABILITY_SEVERITY: 'moderate',
    LOW_VULNERABILITY_SEVERITY: 'low',
} as const;

const fadedTextColor = 'var(--pf-global--Color--200)';

export const resourceCountByCveSeverityFragment = graphql(/* GraphQL */ `
    fragment ResourceCountsByCVESeverity on ResourceCountByCVESeverity {
        low {
            total
        }
        moderate {
            total
        }
        important {
            total
        }
        critical {
            total
        }
    }
`);

export type BySeveritySummaryCardProps = {
    className?: string;
    title: string;
    severityCounts: ResourceCountsByCveSeverityFragment;
    hiddenSeverities: Set<VulnerabilitySeverity>;
};

function BySeveritySummaryCard({
    className = '',
    title,
    severityCounts,
    hiddenSeverities,
}: BySeveritySummaryCardProps) {
    return (
        <Card className={className} isCompact isFlat>
            <CardTitle>{title}</CardTitle>
            <CardBody>
                <Grid className="pf-u-pl-sm">
                    {severitiesCriticalToLow.map((severity) => {
                        const querySeverityKey = severityToQuerySeverityKeys[severity];
                        const count = severityCounts[querySeverityKey];
                        const hasNoResults = count.total === 0;
                        const isHidden = hiddenSeverities.has(severity);

                        let textColor = '';
                        let text = `${count.total} ${vulnerabilitySeverityLabels[severity]}`;

                        if (isHidden) {
                            textColor = fadedTextColor;
                            text = 'Results hidden';
                        } else if (hasNoResults) {
                            textColor = fadedTextColor;
                            text = 'No results';
                        }

                        const Icon = SeverityIcons[severity];

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

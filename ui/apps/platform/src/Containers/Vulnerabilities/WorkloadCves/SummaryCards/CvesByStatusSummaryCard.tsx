import React from 'react';
import {
    Card,
    CardTitle,
    CardBody,
    Flex,
    Grid,
    GridItem,
    pluralize,
    Text,
} from '@patternfly/react-core';

import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';
import { gql } from '@apollo/client';
import { FixableStatus } from '../types';

export const resourceCountByCVESeverityKeys = ['low', 'moderate', 'important', 'critical'] as const;

export type ResourceCountByCveSeverityKey = (typeof resourceCountByCVESeverityKeys)[number];

export type ResourceCountByCveSeverity = Record<
    ResourceCountByCveSeverityKey,
    { total: number; fixable: number }
>;

export type CvesByStatusSummaryCardProps = {
    cveStatusCounts: ResourceCountByCveSeverity;
    hiddenStatuses: Set<FixableStatus>;
};

export const resourceCountByCveSeverityFragment = gql`
    fragment AllResourceCountsByCVESeverity on ResourceCountByCVESeverity {
        low {
            total
            fixable
        }
        moderate {
            total
            fixable
        }
        important {
            total
            fixable
        }
        critical {
            total
            fixable
        }
    }
`;

const statusDisplays = [
    {
        status: 'Fixable',
        Icon: CheckCircleIcon,
        iconColor: 'var(--pf-global--success-color--100)',
        text: (counts: ResourceCountByCveSeverity) => {
            let count = 0;
            resourceCountByCVESeverityKeys.forEach((key) => {
                count += counts[key].fixable;
            });
            return `${pluralize(count, 'vulnerability', 'vulnerabilities')} with available fixes`;
        },
    },
    {
        status: 'Not fixable',
        Icon: ExclamationCircleIcon,
        iconColor: 'var(--pf-global--danger-color--100)',
        text: (counts: ResourceCountByCveSeverity) => {
            let count = 0;
            resourceCountByCVESeverityKeys.forEach((key) => {
                count += counts[key].total - counts[key].fixable;
            });
            return `${count} vulnerabilities without fixes`;
        },
    },
] as const;

const disabledColor100 = 'var(--pf-global--disabled-color--100)';

function CvesByStatusSummaryCard({
    cveStatusCounts,
    hiddenStatuses,
}: CvesByStatusSummaryCardProps) {
    return (
        <Card isCompact>
            <CardTitle>CVEs by status</CardTitle>
            <CardBody>
                <Grid className="pf-u-pl-sm">
                    {statusDisplays.map(({ status, Icon, iconColor, text }) => {
                        const isHidden = hiddenStatuses.has(status);
                        return (
                            <GridItem key={status} span={12}>
                                <Flex
                                    className="pf-u-pt-sm"
                                    spaceItems={{ default: 'spaceItemsSm' }}
                                    alignItems={{ default: 'alignItemsCenter' }}
                                >
                                    <Icon color={iconColor} />
                                    <Text
                                        style={{
                                            color: isHidden ? disabledColor100 : 'inherit',
                                        }}
                                    >
                                        {isHidden ? 'Results hidden' : text(cveStatusCounts)}
                                    </Text>
                                </Flex>
                            </GridItem>
                        );
                    })}
                </Grid>
            </CardBody>
        </Card>
    );
}

export default CvesByStatusSummaryCard;

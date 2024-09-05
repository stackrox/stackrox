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
import { MinusIcon, WrenchIcon } from '@patternfly/react-icons';
import { gql } from '@apollo/client';

import { FixableStatus } from '../../types';

export type ResourceCountByCveSeverityAndStatus = {
    critical: { total: number; fixable: number };
    important: { total: number; fixable: number };
    moderate: { total: number; fixable: number };
    low: { total: number; fixable: number };
};

export const resourceCountByCveSeverityAndStatusFragment = gql`
    fragment ResourceCountsByCVESeverityAndStatus on ResourceCountByCVESeverity {
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
        Icon: WrenchIcon,
        text: (counts: ResourceCountByCveSeverityAndStatus) => {
            const { critical, important, moderate, low } = counts;
            const fixable = critical.fixable + important.fixable + moderate.fixable + low.fixable;
            return `${pluralize(fixable, 'vulnerability', 'vulnerabilities')} with available fixes`;
        },
    },
    {
        status: 'Not fixable',
        Icon: MinusIcon,
        text: (counts: ResourceCountByCveSeverityAndStatus) => {
            const { critical, important, moderate, low } = counts;
            const total = critical.total + important.total + moderate.total + low.total;
            const fixable = critical.fixable + important.fixable + moderate.fixable + low.fixable;
            return `${total - fixable} vulnerabilities without fixes`;
        },
    },
] as const;

const disabledColor100 = 'var(--pf-v5-global--disabled-color--100)';

const statusHiddenText = {
    Fixable: 'Fixable hidden',
    'Not fixable': 'Not fixable hidden',
} as const;

export type CvesByStatusSummaryCardProps = {
    cveStatusCounts: ResourceCountByCveSeverityAndStatus;
    hiddenStatuses: Set<FixableStatus>;
};

function CvesByStatusSummaryCard({
    cveStatusCounts,
    hiddenStatuses,
}: CvesByStatusSummaryCardProps) {
    return (
        <Card isCompact isFlat>
            <CardTitle>CVEs by status</CardTitle>
            <CardBody>
                <Grid className="pf-v5-u-pl-sm">
                    {statusDisplays.map(({ status, Icon, text }) => {
                        const isHidden = hiddenStatuses.has(status);
                        return (
                            <GridItem key={status} span={12}>
                                <Flex
                                    className="pf-v5-u-pt-sm"
                                    spaceItems={{ default: 'spaceItemsSm' }}
                                    alignItems={{ default: 'alignItemsCenter' }}
                                >
                                    <Icon />
                                    <Text
                                        style={{
                                            color: isHidden ? disabledColor100 : 'inherit',
                                        }}
                                    >
                                        {isHidden
                                            ? statusHiddenText[status]
                                            : text(cveStatusCounts)}
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

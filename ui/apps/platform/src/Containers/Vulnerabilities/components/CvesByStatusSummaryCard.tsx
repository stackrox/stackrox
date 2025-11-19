import {
    Card,
    CardBody,
    CardTitle,
    Content,
    Flex,
    Grid,
    GridItem,
    pluralize,
} from '@patternfly/react-core';
import { MinusIcon, WrenchIcon } from '@patternfly/react-icons';
import { gql } from '@apollo/client';
import sumBy from 'lodash/sumBy';

import type { FixableStatus } from '../types';

export type ResourceCountByCveSeverityAndStatus = {
    critical: { total: number; fixable: number };
    important: { total: number; fixable: number };
    moderate: { total: number; fixable: number };
    low: { total: number; fixable: number };
    unknown: { total: number; fixable: number };
};

export const resourceCountByCveSeverityAndStatusFragment = gql`
    fragment ResourceCountsByCVESeverityAndStatus on ResourceCountByCVESeverity {
        unknown {
            total
            fixable
        }
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

const severityKeys = ['critical', 'important', 'moderate', 'low', 'unknown'] as const;

const statusDisplays = [
    {
        status: 'Fixable',
        Icon: WrenchIcon,
        text: (counts: ResourceCountByCveSeverityAndStatus) => {
            const severityCounts = severityKeys.map((key) => counts[key]);
            const fixable = sumBy(severityCounts, 'fixable');
            return `${pluralize(fixable, 'vulnerability', 'vulnerabilities')} with available fixes`;
        },
    },
    {
        status: 'Not fixable',
        Icon: MinusIcon,
        text: (counts: ResourceCountByCveSeverityAndStatus) => {
            const severityCounts = severityKeys.map((key) => counts[key]);
            const total = sumBy(severityCounts, 'total');
            const fixable = sumBy(severityCounts, 'fixable');
            return `${total - fixable} vulnerabilities without fixes`;
        },
    },
] as const;

const disabledColor = 'var(--pf-t--global--text--color--disabled)';

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
        <Card isCompact isFullHeight>
            <CardTitle>CVEs by status</CardTitle>
            <CardBody>
                <Grid className="pf-v6-u-pl-sm">
                    {statusDisplays.map(({ status, Icon, text }) => {
                        const isHidden = hiddenStatuses.has(status);
                        return (
                            <GridItem key={status} span={12}>
                                <Flex
                                    className="pf-v6-u-pt-sm"
                                    spaceItems={{ default: 'spaceItemsSm' }}
                                    alignItems={{ default: 'alignItemsCenter' }}
                                >
                                    <Icon />
                                    <Content
                                        component="p"
                                        style={{
                                            color: isHidden ? disabledColor : 'inherit',
                                        }}
                                    >
                                        {isHidden
                                            ? statusHiddenText[status]
                                            : text(cveStatusCounts)}
                                    </Content>
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

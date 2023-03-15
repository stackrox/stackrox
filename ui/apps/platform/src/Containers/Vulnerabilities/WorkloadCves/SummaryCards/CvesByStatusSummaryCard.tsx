import React from 'react';
import { Card, CardTitle, CardBody, Flex, Text, Grid, GridItem } from '@patternfly/react-core';

import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';
import { FixableStatus } from '../types';

export type CvesByStatusSummaryCardProps = {
    cveStatusCounts: Record<FixableStatus, number | 'hidden'>;
    hiddenStatuses: Set<FixableStatus>;
};

const statusDisplays = [
    {
        status: 'Fixable',
        Icon: CheckCircleIcon,
        iconColor: 'var(--pf-global--success-color--100)',
        text: (counts: CvesByStatusSummaryCardProps['cveStatusCounts']) =>
            `${counts.Fixable} vulnerabilities with available fixes`,
    },
    {
        status: 'Not fixable',
        Icon: ExclamationCircleIcon,
        iconColor: 'var(--pf-global--danger-color--100)',
        text: (counts: CvesByStatusSummaryCardProps['cveStatusCounts']) =>
            `${counts['Not fixable']} vulnerabilities without fixes`,
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

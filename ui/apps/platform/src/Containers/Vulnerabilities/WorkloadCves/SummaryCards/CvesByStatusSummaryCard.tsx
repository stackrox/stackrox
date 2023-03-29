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
import { FixableStatus } from '../types';
import {
    ImageVulnerabilityCounter,
    imageVulnerabilityCounterKeys,
} from '../hooks/useImageVulnerabilities';

export type CvesByStatusSummaryCardProps = {
    cveStatusCounts: ImageVulnerabilityCounter;
    hiddenStatuses: Set<FixableStatus>;
};

const statusDisplays = [
    {
        status: 'Fixable',
        Icon: CheckCircleIcon,
        iconColor: 'var(--pf-global--success-color--100)',
        text: (counts: ImageVulnerabilityCounter) => {
            let count = 0;
            imageVulnerabilityCounterKeys.forEach((key) => {
                count += counts[key].fixable;
            });
            return `${pluralize(count, 'vulnerability', 'vulnerabilities')} with available fixes`;
        },
    },
    {
        status: 'Not fixable',
        Icon: ExclamationCircleIcon,
        iconColor: 'var(--pf-global--danger-color--100)',
        text: (counts: ImageVulnerabilityCounter) => {
            let count = 0;
            imageVulnerabilityCounterKeys.forEach((key) => {
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

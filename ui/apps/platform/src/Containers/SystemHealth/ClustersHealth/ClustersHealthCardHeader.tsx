import React, { ReactElement } from 'react';
import { CheckCircleIcon } from '@patternfly/react-icons';
import { Button, CardHeader, CardTitle, Flex, FlexItem, Spinner } from '@patternfly/react-core';

import IconText from 'Components/PatternFly/IconText/IconText';
import LinkShim from 'Components/PatternFly/LinkShim/LinkShim';
import { clustersBasePath } from 'routePaths';

import { ClusterStatusCounts } from './ClustersHealth.utils';

export type ClustersHealthCardHeaderProps = {
    counts: ClusterStatusCounts | null;
    isFetchingInitialRequest: boolean;
    title: string;
};

function ClustersHealthCardHeader({
    counts,
    isFetchingInitialRequest,
    title,
}: ClustersHealthCardHeaderProps): ReactElement {
    const isHealthy =
        counts !== null && counts.HEALTHY !== 0 && counts.UNHEALTHY === 0 && counts.DEGRADED === 0;

    return (
        <CardHeader>
            <Flex className="pf-u-flex-grow-1">
                <FlexItem>
                    <CardTitle component="h2">{title}</CardTitle>
                </FlexItem>
                {isFetchingInitialRequest && <Spinner isSVG size="md" />}
                {isHealthy && (
                    <IconText
                        icon={<CheckCircleIcon color="var(--pf-global--success-color--100)" />}
                        text="healthy"
                    />
                )}
                <FlexItem align={{ default: 'alignRight' }}>
                    <Button variant="link" isInline component={LinkShim} href={clustersBasePath}>
                        View clusters
                    </Button>
                </FlexItem>
            </Flex>
        </CardHeader>
    );
}

export default ClustersHealthCardHeader;

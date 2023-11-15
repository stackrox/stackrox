import React, { ReactElement } from 'react';
import { Button, CardHeader, CardTitle, Flex, FlexItem } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim/LinkShim';
import { clustersBasePath } from 'routePaths';

import { ErrorIcon, healthIconMap, SpinnerIcon } from '../CardHeaderIcons';

import {
    ClusterStatusCounts,
    getClustersHealthPhrase,
    getClustersHealthVariant,
} from './ClustersHealth.utils';

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
    /* eslint-disable no-nested-ternary */
    const icon = isFetchingInitialRequest
        ? SpinnerIcon
        : !counts
          ? ErrorIcon
          : healthIconMap[getClustersHealthVariant(counts)];
    /* eslint-enable no-nested-ternary */

    const phrase = counts === null ? '' : getClustersHealthPhrase(counts);

    return (
        <CardHeader>
            <Flex className="pf-u-flex-grow-1">
                <FlexItem>{icon}</FlexItem>
                <FlexItem>
                    <CardTitle component="h2">{title}</CardTitle>
                </FlexItem>
                {phrase && <FlexItem>{phrase}</FlexItem>}
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

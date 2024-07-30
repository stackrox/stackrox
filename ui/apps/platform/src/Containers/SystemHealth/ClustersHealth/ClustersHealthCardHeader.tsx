import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { CardHeader, CardTitle, Flex, FlexItem } from '@patternfly/react-core';

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
            <Flex className="pf-v5-u-flex-grow-1">
                <FlexItem>{icon}</FlexItem>
                <FlexItem>
                    <CardTitle component="h2">{title}</CardTitle>
                </FlexItem>
                {phrase && <FlexItem>{phrase}</FlexItem>}
                <FlexItem align={{ default: 'alignRight' }}>
                    <Link to={clustersBasePath}>View clusters</Link>
                </FlexItem>
            </Flex>
        </CardHeader>
    );
}

export default ClustersHealthCardHeader;

import React from 'react';
import { Flex, Skeleton } from '@patternfly/react-core';

export type HeaderLoadingSkeletonProps = {
    nameScreenreaderText: string;
    metadataScreenreaderText: string;
};

function HeaderLoadingSkeleton({
    nameScreenreaderText,
    metadataScreenreaderText,
}: HeaderLoadingSkeletonProps) {
    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXs' }}>
            <Skeleton screenreaderText={nameScreenreaderText} fontSize="2xl" />
            <Skeleton screenreaderText={metadataScreenreaderText} height="100px" />
        </Flex>
    );
}

export default HeaderLoadingSkeleton;

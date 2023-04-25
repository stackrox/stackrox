import React from 'react';
import { gql } from '@apollo/client';
import { Flex, Title, LabelGroup, Label, Skeleton } from '@patternfly/react-core';
import { getDateTime } from 'utils/dateUtils';

export type DeploymentMetadata = {
    id: string;
    name: string;
    namespace: string;
    clusterName: string;
    created: Date | null;
    imageCount: number;
};

export const deploymentMetadataFragment = gql`
    fragment DeploymentMetadata on Deployment {
        id
        name
        namespace
        clusterName
        created
        imageCount
    }
`;

export type DeploymentPageHeaderProps = {
    data: DeploymentMetadata | undefined;
};

function DeploymentPageHeader({ data }: DeploymentPageHeaderProps) {
    return data ? (
        <Flex direction={{ default: 'column' }} alignItems={{ default: 'alignItemsFlexStart' }}>
            <Title headingLevel="h1" className="pf-u-mb-sm">
                {data.name}
            </Title>
            <LabelGroup numLabels={3}>
                <Label isCompact>
                    In: {data.clusterName}/{data.namespace}
                </Label>
                <Label isCompact>Images: {data.imageCount}</Label>
                <Label isCompact>Created: {getDateTime(data.created)}</Label>
            </LabelGroup>
        </Flex>
    ) : (
        <Flex
            direction={{ default: 'column' }}
            spaceItems={{ default: 'spaceItemsXs' }}
            className="pf-u-w-50"
        >
            <Skeleton screenreaderText="Loading Deployment name" fontSize="2xl" />
            <Skeleton screenreaderText="Loading Deployment metadata" fontSize="sm" />
        </Flex>
    );
}

export default DeploymentPageHeader;

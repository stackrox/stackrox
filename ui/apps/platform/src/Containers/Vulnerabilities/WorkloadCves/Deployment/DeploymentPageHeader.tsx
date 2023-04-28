import React from 'react';
import { Flex, Title, LabelGroup, Label, Skeleton } from '@patternfly/react-core';
import { getDateTime } from 'utils/dateUtils';
import { graphql } from 'generated/graphql-codegen';
import { DeploymentMetadataFragment } from 'generated/graphql-codegen/graphql';

export const deploymentMetadataFragment = graphql(/* GraphQL */ `
    fragment DeploymentMetadata on Deployment {
        id
        name
        namespace
        clusterName
        created
        imageCount
    }
`);

export type DeploymentPageHeaderProps = {
    deployment: DeploymentMetadataFragment | null | undefined;
};

function DeploymentPageHeader({ deployment }: DeploymentPageHeaderProps) {
    return deployment ? (
        <Flex direction={{ default: 'column' }} alignItems={{ default: 'alignItemsFlexStart' }}>
            <Title headingLevel="h1" className="pf-u-mb-sm">
                {deployment.name}
            </Title>
            <LabelGroup numLabels={3}>
                <Label>
                    In: {deployment.clusterName}/{deployment.namespace}
                </Label>
                <Label>Images: {deployment.imageCount}</Label>
                {deployment.created && <Label>Created: {getDateTime(deployment.created)}</Label>}
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

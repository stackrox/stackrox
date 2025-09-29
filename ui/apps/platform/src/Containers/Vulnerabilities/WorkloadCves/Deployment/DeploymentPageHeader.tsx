import React from 'react';
import { gql } from '@apollo/client';
import { Flex, FlexItem, Title, LabelGroup, Label } from '@patternfly/react-core';
import { getDateTime } from 'utils/dateUtils';

import HeaderLoadingSkeleton from '../../components/HeaderLoadingSkeleton';

export type DeploymentMetadata = {
    id: string;
    name: string;
    namespace: string;
    clusterName: string;
    created: string | null;
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
    data: DeploymentMetadata | null | undefined;
    additionalActions?: React.ReactNode;
};

function DeploymentPageHeader({ data, additionalActions }: DeploymentPageHeaderProps) {
    return data ? (
        <Flex justifyContent={{ default: 'justifyContentSpaceBetween' }}>
            <Flex direction={{ default: 'column' }} alignItems={{ default: 'alignItemsFlexStart' }}>
                <Title headingLevel="h1" className="pf-v5-u-mb-sm">
                    {data.name}
                </Title>
                <LabelGroup numLabels={3}>
                    <Label>
                        In: {data.clusterName}/{data.namespace}
                    </Label>
                    <Label>Images: {data.imageCount}</Label>
                    {data.created && <Label>Created: {getDateTime(data.created)}</Label>}
                </LabelGroup>
            </Flex>
            {additionalActions && (
                <FlexItem alignSelf={{ default: 'alignSelfCenter' }}>{additionalActions}</FlexItem>
            )}
        </Flex>
    ) : (
        <HeaderLoadingSkeleton
            nameScreenreaderText="Loading Deployment name"
            metadataScreenreaderText="Loading Deployment metadata"
        />
    );
}

export default DeploymentPageHeader;

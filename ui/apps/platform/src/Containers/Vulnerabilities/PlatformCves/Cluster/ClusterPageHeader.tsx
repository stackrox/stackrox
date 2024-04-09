import React from 'react';
import { Flex, Title, LabelGroup, Label } from '@patternfly/react-core';
import { gql } from '@apollo/client';

import { getDateTime } from 'utils/dateUtils';

import HeaderLoadingSkeleton from '../../components/HeaderLoadingSkeleton';

export const clusterMetadataFragment = gql`
    fragment ClusterMetadata on Cluster {
        id
        name
        status {
            orchestratorMetadata {
                buildDate
                version
            }
        }
    }
`;

export type ClusterMetadata = {
    id: string;
    name: string;
    status?: {
        orchestratorMetadata?: {
            buildDate?: string; // ISO 8601 date format
            version: string;
        };
    };
};

export type ClusterPageHeaderProps = {
    data: ClusterMetadata | undefined;
};

function ClusterPageHeader({ data }: ClusterPageHeaderProps) {
    if (!data) {
        return (
            <HeaderLoadingSkeleton
                nameScreenreaderText="Loading Cluster name"
                metadataScreenreaderText="Loading Cluster metadata"
            />
        );
    }

    const buildDate = data.status?.orchestratorMetadata?.buildDate;
    const version = data.status?.orchestratorMetadata?.version;
    const numLabels = 0 + (buildDate ? 1 : 0) + (version ? 1 : 0);

    return (
        <Flex direction={{ default: 'column' }} alignItems={{ default: 'alignItemsFlexStart' }}>
            <Title headingLevel="h1" className="pf-v5-u-mb-sm">
                {data.name}
            </Title>
            {numLabels > 0 && (
                <LabelGroup numLabels={numLabels}>
                    {version && <Label>K8s version: {version}</Label>}
                    {buildDate && <Label>Build date: {getDateTime(buildDate)}</Label>}
                </LabelGroup>
            )}
        </Flex>
    );
}

export default ClusterPageHeader;

import React from 'react';
import {
    DescriptionList,
    DescriptionListGroup,
    DescriptionListTerm,
    DescriptionListDescription,
} from '@patternfly/react-core';

import type { ClusterStatus } from 'types/cluster.proto';

import { formatBuildDate, formatCloudProvider, formatKubernetesVersion } from '../cluster.helpers';

export type ClusterMetadataProps = {
    status: ClusterStatus;
};

function ClusterMetadata({ status }: ClusterMetadataProps) {
    return (
        <DescriptionList>
            <DescriptionListGroup>
                <DescriptionListTerm>Kubernetes version</DescriptionListTerm>
                <DescriptionListDescription>
                    {formatKubernetesVersion(status?.orchestratorMetadata)}
                </DescriptionListDescription>
            </DescriptionListGroup>
            <DescriptionListGroup>
                <DescriptionListTerm>Build date</DescriptionListTerm>
                <DescriptionListDescription>
                    {formatBuildDate(status?.orchestratorMetadata)}
                </DescriptionListDescription>
            </DescriptionListGroup>
            <DescriptionListGroup>
                <DescriptionListTerm>Cloud provider</DescriptionListTerm>
                <DescriptionListDescription>
                    {formatCloudProvider(status?.providerMetadata)}
                </DescriptionListDescription>
            </DescriptionListGroup>
        </DescriptionList>
    );
}

export default ClusterMetadata;

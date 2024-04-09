import React from 'react';
import { useParams } from 'react-router-dom';
import {
    PageSection,
    Breadcrumb,
    Divider,
    BreadcrumbItem,
    Skeleton,
    Bullseye,
} from '@patternfly/react-core';
import { gql, useQuery } from '@apollo/client';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import ClusterPageHeader, { ClusterMetadata, clusterMetadataFragment } from './ClusterPageHeader';
import { getOverviewPagePath } from '../../utils/searchUtils';

const platformCvesClusterOverviewPath = getOverviewPagePath('Platform', {
    entityTab: 'Cluster',
});

const clusterMetadataQuery = gql`
    ${clusterMetadataFragment}
    query getClusterMetadata($id: ID!) {
        cluster(id: $id) {
            ...ClusterMetadata
        }
    }
`;

// TODO - Update for PF5
function ClusterPage() {
    const { clusterId } = useParams() as { clusterId: string };

    const { data, error } = useQuery<{ cluster: ClusterMetadata }, { id: string }>(
        clusterMetadataQuery,
        {
            variables: { id: clusterId },
        }
    );

    const clusterName = data?.cluster?.name ?? '';

    return (
        <>
            <PageTitle title={`Platform CVEs - Cluster ${clusterName}`} />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={platformCvesClusterOverviewPath}>
                        Clusters
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>
                        {clusterName ?? (
                            <Skeleton screenreaderText="Loading cluster name" width="200px" />
                        )}
                    </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            {error ? (
                <PageSection variant="light">
                    <Bullseye>
                        <EmptyStateTemplate
                            title={getAxiosErrorMessage(error)}
                            headingLevel="h2"
                            icon={ExclamationCircleIcon}
                            iconClassName="pf-v5-u-danger-color-100"
                        />
                    </Bullseye>
                </PageSection>
            ) : (
                <PageSection variant="light">
                    <ClusterPageHeader data={data?.cluster} />
                </PageSection>
            )}
        </>
    );
}

export default ClusterPage;

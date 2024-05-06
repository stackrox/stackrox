import React from 'react';
import { useParams } from 'react-router-dom';
import {
    PageSection,
    Breadcrumb,
    Divider,
    BreadcrumbItem,
    Skeleton,
    Bullseye,
    Tab,
    Tabs,
    TabsComponent,
} from '@patternfly/react-core';
import { gql, useQuery } from '@apollo/client';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import { getOverviewPagePath } from '../../utils/searchUtils';
import { detailsTabValues } from '../../types';

import ClusterPageHeader, { ClusterMetadata, clusterMetadataFragment } from './ClusterPageHeader';
import ClusterPageDetails from './ClusterPageDetails';
import ClusterPageVulnerabilities from './ClusterPageVulnerabilities';

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

    const [activeTabKey, setActiveTabKey] = useURLStringUnion('detailsTab', detailsTabValues);

    const vulnTabKey = detailsTabValues[0];
    const detailTabKey = detailsTabValues[1];

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
                <>
                    <PageSection variant="light">
                        <ClusterPageHeader data={data?.cluster} />
                    </PageSection>
                    <PageSection padding={{ default: 'noPadding' }}>
                        <Tabs
                            activeKey={activeTabKey}
                            onSelect={(e, key) => {
                                setActiveTabKey(key);
                                // pagination.setPage(1);
                            }}
                            component={TabsComponent.nav}
                            className="pf-v5-u-pl-md pf-v5-u-background-color-100"
                            role="region"
                        >
                            <Tab eventKey={vulnTabKey} title={vulnTabKey} />
                            <Tab eventKey={detailTabKey} title={detailTabKey} />
                        </Tabs>
                    </PageSection>
                    <PageSection
                        isFilled
                        padding={{ default: 'noPadding' }}
                        className="pf-v5-u-display-flex pf-v5-u-flex-direction-column"
                    >
                        {activeTabKey === vulnTabKey && (
                            <ClusterPageVulnerabilities clusterId={clusterId} />
                        )}
                        {activeTabKey === detailTabKey && (
                            <ClusterPageDetails clusterId={clusterId} />
                        )}
                    </PageSection>
                </>
            )}
        </>
    );
}

export default ClusterPage;

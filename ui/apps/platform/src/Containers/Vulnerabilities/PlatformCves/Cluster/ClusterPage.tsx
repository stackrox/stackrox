import { useParams } from 'react-router-dom-v5-compat';
import {
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    PageSection,
    Skeleton,
    Tab,
    Tabs,
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

import ClusterPageHeader, { clusterMetadataFragment } from './ClusterPageHeader';
import type { ClusterMetadata } from './ClusterPageHeader';
import ClusterPageDetails from './ClusterPageDetails';
import ClusterPageVulnerabilities from './ClusterPageVulnerabilities';

const idDetails = 'ClusterPageDetails';
const idVulnerabilities = 'ClusterPageVulnerabilities';

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
            <PageSection>
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
            {error ? (
                <PageSection hasBodyWrapper={false}>
                    <Bullseye>
                        <EmptyStateTemplate
                            title={getAxiosErrorMessage(error)}
                            headingLevel="h2"
                            icon={ExclamationCircleIcon}
                            status="danger"
                        />
                    </Bullseye>
                </PageSection>
            ) : (
                <>
                    <PageSection hasBodyWrapper={false}>
                        <ClusterPageHeader data={data?.cluster} />
                    </PageSection>
                    <PageSection type="tabs">
                        <Tabs
                            activeKey={activeTabKey}
                            onSelect={(e, key) => {
                                setActiveTabKey(key);
                                // pagination.setPage(1);
                            }}
                            usePageInsets
                            mountOnEnter
                            unmountOnExit
                        >
                            <Tab
                                eventKey={vulnTabKey}
                                tabContentId={idVulnerabilities}
                                title={vulnTabKey}
                            >
                                <ClusterPageVulnerabilities clusterId={clusterId} />
                            </Tab>
                            <Tab
                                eventKey={detailTabKey}
                                tabContentId={idDetails}
                                title={detailTabKey}
                            >
                                <ClusterPageDetails clusterId={clusterId} />
                            </Tab>
                        </Tabs>
                    </PageSection>
                </>
            )}
        </>
    );
}

export default ClusterPage;

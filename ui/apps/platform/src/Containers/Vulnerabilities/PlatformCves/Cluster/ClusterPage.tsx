import { useParams } from 'react-router-dom-v5-compat';
import {
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Divider,
    PageSection,
    Skeleton,
    Tab,
    TabContent,
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
            <PageSection hasBodyWrapper={false} className="pf-v6-u-py-md">
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
                <PageSection hasBodyWrapper={false}>
                    <Bullseye>
                        <EmptyStateTemplate
                            title={getAxiosErrorMessage(error)}
                            headingLevel="h2"
                            icon={ExclamationCircleIcon}
                            iconClassName="pf-v6-u-danger-color-100"
                        />
                    </Bullseye>
                </PageSection>
            ) : (
                <>
                    <PageSection hasBodyWrapper={false}>
                        <ClusterPageHeader data={data?.cluster} />
                    </PageSection>
                    <PageSection hasBodyWrapper={false} padding={{ default: 'noPadding' }}>
                        <Tabs
                            activeKey={activeTabKey}
                            onSelect={(e, key) => {
                                setActiveTabKey(key);
                                // pagination.setPage(1);
                            }}
                            className="pf-v6-u-pl-md pf-v6-u-background-color-100"
                        >
                            <Tab
                                eventKey={vulnTabKey}
                                tabContentId={idVulnerabilities}
                                title={vulnTabKey}
                            />
                            <Tab
                                eventKey={detailTabKey}
                                tabContentId={idDetails}
                                title={detailTabKey}
                            />
                        </Tabs>
                    </PageSection>
                    <PageSection
                        hasBodyWrapper={false}
                        isFilled
                        padding={{ default: 'noPadding' }}
                        className="pf-v6-u-display-flex pf-v6-u-flex-direction-column"
                    >
                        {activeTabKey === vulnTabKey && (
                            <TabContent id={idVulnerabilities}>
                                <ClusterPageVulnerabilities clusterId={clusterId} />
                            </TabContent>
                        )}
                        {activeTabKey === detailTabKey && (
                            <TabContent id={idDetails}>
                                <ClusterPageDetails clusterId={clusterId} />
                            </TabContent>
                        )}
                    </PageSection>
                </>
            )}
        </>
    );
}

export default ClusterPage;

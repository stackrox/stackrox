import React from 'react';
import {
    PageSection,
    Breadcrumb,
    Divider,
    BreadcrumbItem,
    Skeleton,
    Tab,
    TabTitleText,
    Tabs,
    TabsComponent,
} from '@patternfly/react-core';
import { useParams } from 'react-router-dom';
import { gql, useQuery } from '@apollo/client';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import NotFoundMessage from 'Components/NotFoundMessage';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import useURLStringUnion from 'hooks/useURLStringUnion';
import useURLPagination from 'hooks/useURLPagination';

import DeploymentPageHeader, {
    DeploymentMetadata,
    deploymentMetadataFragment,
} from './DeploymentPageHeader';
import { getOverviewPagePath } from '../../utils/searchUtils';
import { detailsTabValues } from '../../types';
import DeploymentPageResources from './DeploymentPageResources';
import DeploymentPageVulnerabilities from './DeploymentPageVulnerabilities';
import DeploymentPageDetails from './DeploymentPageDetails';

const workloadCveOverviewDeploymentsPath = getOverviewPagePath('Workload', {
    vulnerabilityState: 'OBSERVED',
    entityTab: 'Deployment',
});

const deploymentMetadataQuery = gql`
    ${deploymentMetadataFragment}
    query getDeploymentMetadata($id: ID!) {
        deployment(id: $id) {
            ...DeploymentMetadata
        }
    }
`;

function DeploymentPage() {
    const { deploymentId } = useParams() as { deploymentId: string };
    const [activeTabKey, setActiveTabKey] = useURLStringUnion('detailsTab', detailsTabValues);

    const pagination = useURLPagination(20);

    const metadataRequest = useQuery<{ deployment: DeploymentMetadata | null }, { id: string }>(
        deploymentMetadataQuery,
        {
            variables: { id: deploymentId },
        }
    );

    const deploymentName = metadataRequest.data?.deployment?.name;
    const deploymentNotFound = metadataRequest.data && !metadataRequest.data.deployment;

    return (
        <>
            <PageTitle title={`Workload CVEs - Deployment ${deploymentName ?? ''}`} />
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={workloadCveOverviewDeploymentsPath}>
                        Deployments
                    </BreadcrumbItemLink>
                    {!metadataRequest.error && (
                        <BreadcrumbItem isActive>
                            {deploymentName ?? (
                                <Skeleton
                                    screenreaderText="Loading deployment name"
                                    width="200px"
                                />
                            )}
                        </BreadcrumbItem>
                    )}
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            {deploymentNotFound ? (
                <NotFoundMessage
                    title="404: We couldn't find that page"
                    message={`A deployment with ID ${deploymentId} could not be found.`}
                />
            ) : (
                <>
                    <PageSection variant="light">
                        {metadataRequest.error && (
                            <TableErrorComponent
                                error={metadataRequest.error}
                                message="The system was unable to load metadata for this deployment"
                            />
                        )}
                        <DeploymentPageHeader data={metadataRequest.data?.deployment} />
                    </PageSection>
                    <PageSection
                        className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
                        padding={{ default: 'noPadding' }}
                    >
                        <Tabs
                            activeKey={activeTabKey}
                            onSelect={(e, key) => {
                                setActiveTabKey(key);
                                pagination.setPage(1);
                            }}
                            component={TabsComponent.nav}
                            className="pf-u-pl-md pf-u-background-color-100"
                            mountOnEnter
                            unmountOnExit
                        >
                            <Tab
                                className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
                                eventKey="Vulnerabilities"
                                title={<TabTitleText>Vulnerabilities</TabTitleText>}
                            >
                                <DeploymentPageVulnerabilities
                                    deploymentId={deploymentId}
                                    pagination={pagination}
                                />
                            </Tab>
                            <Tab
                                className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
                                eventKey="Details"
                                title={<TabTitleText>Details</TabTitleText>}
                            >
                                <DeploymentPageDetails deploymentId={deploymentId} />
                            </Tab>
                            <Tab
                                className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
                                eventKey="Resources"
                                title={<TabTitleText>Resources</TabTitleText>}
                            >
                                <DeploymentPageResources
                                    deploymentId={deploymentId}
                                    pagination={pagination}
                                />
                            </Tab>
                        </Tabs>
                    </PageSection>
                </>
            )}
        </>
    );
}

export default DeploymentPage;

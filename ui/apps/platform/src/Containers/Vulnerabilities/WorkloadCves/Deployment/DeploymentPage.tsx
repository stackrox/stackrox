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
    Text,
} from '@patternfly/react-core';
import { useParams } from 'react-router-dom-v5-compat';
import { gql, useQuery } from '@apollo/client';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import NotFoundMessage from 'Components/NotFoundMessage';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import useURLStringUnion from 'hooks/useURLStringUnion';
import useURLPagination from 'hooks/useURLPagination';
import type { VulnerabilityState } from 'types/cve.proto';

import DeploymentPageHeader, {
    DeploymentMetadata,
    deploymentMetadataFragment,
} from './DeploymentPageHeader';
import { detailsTabValues } from '../../types';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import DeploymentPageResources from './DeploymentPageResources';
import DeploymentPageVulnerabilities from './DeploymentPageVulnerabilities';
import DeploymentPageDetails from './DeploymentPageDetails';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';

const deploymentMetadataQuery = gql`
    ${deploymentMetadataFragment}
    query getDeploymentMetadata($id: ID!) {
        deployment(id: $id) {
            ...DeploymentMetadata
        }
    }
`;
export type DeploymentPageProps = {
    showVulnerabilityStateTabs: boolean;
    vulnerabilityState: VulnerabilityState;
};

function DeploymentPage({ showVulnerabilityStateTabs, vulnerabilityState }: DeploymentPageProps) {
    const { deploymentId } = useParams() as { deploymentId: string };
    const { urlBuilder, pageTitle } = useWorkloadCveViewContext();
    const [activeTabKey, setActiveTabKey] = useURLStringUnion('detailsTab', detailsTabValues);

    const workloadCveOverviewDeploymentsPath = urlBuilder.workloadList('OBSERVED');

    const pagination = useURLPagination(DEFAULT_VM_PAGE_SIZE);

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
            <PageTitle title={`${pageTitle} - Deployment ${deploymentName ?? ''}`} />
            <PageSection variant="light" className="pf-v5-u-py-md">
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
                        className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                        padding={{ default: 'noPadding' }}
                    >
                        <Tabs
                            activeKey={activeTabKey}
                            onSelect={(e, key) => {
                                setActiveTabKey(key);
                                pagination.setPage(1);
                            }}
                            className="pf-v5-u-pl-md pf-v5-u-background-color-100"
                            mountOnEnter
                            unmountOnExit
                        >
                            <Tab
                                className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                                eventKey="Vulnerabilities"
                                title={<TabTitleText>Vulnerabilities</TabTitleText>}
                            >
                                <PageSection
                                    component="div"
                                    variant="light"
                                    className="pf-v5-u-py-md pf-v5-u-px-xl"
                                >
                                    <Text>
                                        Review and triage vulnerability data scanned for images
                                        within this deployment
                                    </Text>
                                </PageSection>
                                <Divider component="div" />
                                <DeploymentPageVulnerabilities
                                    deploymentId={deploymentId}
                                    pagination={pagination}
                                    showVulnerabilityStateTabs={showVulnerabilityStateTabs}
                                    vulnerabilityState={vulnerabilityState}
                                />
                            </Tab>
                            <Tab
                                className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                                eventKey="Details"
                                title={<TabTitleText>Details</TabTitleText>}
                            >
                                <DeploymentPageDetails deploymentId={deploymentId} />
                            </Tab>
                            <Tab
                                className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
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

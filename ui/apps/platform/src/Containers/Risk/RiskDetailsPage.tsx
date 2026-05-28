import type { ReactElement } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Button,
    Flex,
    FlexItem,
    PageBreadcrumb,
    PageSection,
    Skeleton,
    Spinner,
    Title,
} from '@patternfly/react-core';
import { useParams } from 'react-router-dom-v5-compat';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import type { FilteredWorkflowView } from 'Components/FilteredWorkflowViewSelector/types';
import useFilteredWorkflowViewURLState from 'Components/FilteredWorkflowViewSelector/useFilteredWorkflowViewURLState';
import LinkShim from 'Components/PatternFly/LinkShim/LinkShim';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import {
    getLinkToDeploymentInNetworkGraph,
    riskFullViewPath,
    riskPlatformViewPath,
    riskUserWorkloadsViewPath,
} from 'routePaths';
import useIsRouteEnabled from 'hooks/useIsRouteEnabled';

import RiskDetailTabs from './RiskDetailTabs';
import useDeploymentWithRisk from './useDeploymentWithRisk';

function getRiskBreadcrumb(filteredWorkflowView: FilteredWorkflowView) {
    // Note: We cannot exhaustively check for all possible values of filteredWorkflowView because
    // `FilteredWorkflowView` contains `Node view`, which is not a valid value for Risk.
    // Therefore, we only check for the valid values for Risk.
    if (filteredWorkflowView === 'Platform view') {
        return { title: 'Platform risk', url: riskPlatformViewPath };
    }
    if (filteredWorkflowView === 'Full view') {
        return { title: 'All deployment risk', url: riskFullViewPath };
    }
    return { title: 'User workload risk', url: riskUserWorkloadsViewPath };
}

function RiskDetailsPage(): ReactElement {
    const params = useParams();
    const { deploymentId } = params as { deploymentId: string };

    const { data, isLoading, error } = useDeploymentWithRisk(deploymentId);
    const deploymentName = data?.deployment.name;

    const { filteredWorkflowView } = useFilteredWorkflowViewURLState();

    const isRouteEnabled = useIsRouteEnabled();
    const isRouteEnabledForNetworkGraph = isRouteEnabled('network-graph');

    const { title: breadcrumbTitle, url: breadcrumbUrl } = getRiskBreadcrumb(filteredWorkflowView);

    return (
        <>
            <PageBreadcrumb>
                <Breadcrumb>
                    <BreadcrumbItemLink to={breadcrumbUrl}>{breadcrumbTitle}</BreadcrumbItemLink>
                    <BreadcrumbItem>{deploymentName ?? <Skeleton width="200px" />}</BreadcrumbItem>
                </Breadcrumb>
            </PageBreadcrumb>
            <PageSection>
                <Flex
                    justifyContent={{ default: 'justifyContentSpaceBetween' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                >
                    {deploymentName ? (
                        <Title headingLevel="h1">{deploymentName}</Title>
                    ) : (
                        <Skeleton width="25%" screenreaderText="Loading deployment information" />
                    )}
                    <FlexItem>
                        {isRouteEnabledForNetworkGraph && data && (
                            <Button
                                variant="link"
                                href={getLinkToDeploymentInNetworkGraph({
                                    cluster: data.deployment.clusterName,
                                    namespace: data.deployment.namespace,
                                    deploymentId: data.deployment.id,
                                })}
                                component={LinkShim}
                            >
                                View Deployment in Network Graph
                            </Button>
                        )}
                    </FlexItem>
                </Flex>
            </PageSection>
            {error && (
                <TableErrorComponent
                    error={error}
                    message="There was an error loading the deployment data"
                />
            )}
            {isLoading && !data && (
                <Bullseye>
                    <Spinner aria-label="Loading deployment information" />
                </Bullseye>
            )}
            {data && !error && <RiskDetailTabs deployment={data.deployment} risk={data.risk} />}
        </>
    );
}

export default RiskDetailsPage;

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
import LinkShim from 'Components/PatternFly/LinkShim/LinkShim';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import { getLinkToDeploymentInNetworkGraph, riskBasePath } from 'routePaths';
import useIsRouteEnabled from 'hooks/useIsRouteEnabled';

import RiskDetailTabs from './RiskDetailTabs';
import useDeploymentWithRisk from './useDeploymentWithRisk';

function RiskDetailsPage(): ReactElement {
    const params = useParams();
    const { deploymentId } = params as { deploymentId: string };

    const { data, isLoading, error } = useDeploymentWithRisk(deploymentId);
    const deploymentName = data?.deployment.name;

    const isRouteEnabled = useIsRouteEnabled();
    const isRouteEnabledForNetworkGraph = isRouteEnabled('network-graph');

    return (
        <>
            <PageBreadcrumb>
                <Breadcrumb>
                    <BreadcrumbItemLink to={riskBasePath}>Risk</BreadcrumbItemLink>
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

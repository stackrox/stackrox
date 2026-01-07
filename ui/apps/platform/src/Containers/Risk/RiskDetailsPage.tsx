import type { ReactElement } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    PageBreadcrumb,
    PageSection,
    Skeleton,
    Spinner,
    Title,
} from '@patternfly/react-core';
import { useParams } from 'react-router-dom-v5-compat';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import { riskBasePath } from 'routePaths';

import RiskSidePanel from './RiskSidePanel';
import useDeploymentWithRisk from './useDeploymentWithRisk';

function RiskDetailsPage(): ReactElement {
    const params = useParams();
    const { deploymentId } = params as { deploymentId: string };

    const { data, isLoading, error } = useDeploymentWithRisk(deploymentId);
    const deploymentName = data?.deployment.name;

    return (
        <>
            <PageBreadcrumb>
                <Breadcrumb>
                    <BreadcrumbItemLink to={riskBasePath}>Risk</BreadcrumbItemLink>
                    <BreadcrumbItem>{deploymentName ?? <Skeleton width="200px" />}</BreadcrumbItem>
                </Breadcrumb>
            </PageBreadcrumb>
            <PageSection variant="light">
                {deploymentName ? (
                    <Title headingLevel="h1" className="pf-v5-u-mb-sm">
                        {deploymentName}
                    </Title>
                ) : (
                    <Skeleton width="25%" screenreaderText="Loading deployment information" />
                )}
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
            {data && !error && <RiskSidePanel deploymentWithRisk={data} />}
        </>
    );
}

export default RiskDetailsPage;

import type { ReactElement } from 'react';
import { Alert, Breadcrumb, BreadcrumbItem, PageSection, Title } from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import { clustersBasePath, clustersInitBundlesPath } from 'routePaths';

import SecureClusterUsingOperator from './SecureClusterUsingOperator';

const title = 'Secure a cluster with an init bundle';
const headingLevel = 'h2';

function SecureClusterPage(): ReactElement {
    return (
        <>
            <PageTitle title="Secure a cluster" />
            <PageSection type="breadcrumb">
                <Breadcrumb>
                    <BreadcrumbItemLink to={clustersBasePath}>Clusters</BreadcrumbItemLink>
                    <BreadcrumbItemLink to={clustersInitBundlesPath}>
                        Cluster init bundles
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{title}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <PageSection>
                <Title headingLevel="h1">Secure a cluster with an init bundle</Title>
            </PageSection>
            <PageSection>
                <SecureClusterUsingOperator headingLevel={headingLevel} />
            </PageSection>
            <PageSection>
                <Alert
                    variant="info"
                    isInline
                    title="You can use a single init bundle to register multiple clusters."
                    component="p"
                />
            </PageSection>
        </>
    );
}

export default SecureClusterPage;

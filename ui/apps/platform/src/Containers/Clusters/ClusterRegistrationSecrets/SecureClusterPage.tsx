import type { ReactElement } from 'react';
import { Alert, Breadcrumb, BreadcrumbItem, PageSection, Title } from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import { clustersBasePath, clustersClusterRegistrationSecretsPath } from 'routePaths';

import SecureClusterUsingOperator from './SecureClusterUsingOperator';

const title = 'Secure a cluster with a cluster registration secret';
const headingLevel = 'h2';

function SecureClusterPage(): ReactElement {
    return (
        <>
            <PageTitle title="Secure a cluster" />
            <PageSection type="breadcrumb">
                <Breadcrumb>
                    <BreadcrumbItemLink to={clustersBasePath}>Clusters</BreadcrumbItemLink>
                    <BreadcrumbItemLink to={clustersClusterRegistrationSecretsPath}>
                        Cluster registration secrets
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{title}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <PageSection>
                <Title headingLevel="h1">Secure a cluster with a cluster registration secret</Title>
            </PageSection>
            <PageSection>
                <SecureClusterUsingOperator headingLevel={headingLevel} />
            </PageSection>
            <PageSection>
                <Alert
                    variant="info"
                    isInline
                    title="You can use a single cluster registration secret to register multiple clusters."
                    component="p"
                />
            </PageSection>
        </>
    );
}

export default SecureClusterPage;

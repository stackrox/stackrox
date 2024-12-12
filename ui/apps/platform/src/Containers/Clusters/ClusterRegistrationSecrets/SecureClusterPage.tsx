import React, { ReactElement } from 'react';
import { useLocation } from 'react-router-dom';
import qs from 'qs';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Title,
} from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import TabNav from 'Components/TabNav/TabNav';
import { clustersBasePath, clustersClusterRegistrationSecretsPath, clustersSecureClusterCrsPath } from 'routePaths';

import SecureClusterUsingHelmChart from './SecureClusterUsingHelmChart';
import SecureClusterUsingOperator from './SecureClusterUsingOperator';

const title = 'Secure a cluster with a cluster registration secret';
const headingLevel = 'h2';

const tabHelmChart = 'Helm-chart';
const titleOperator = 'Operator';
const titleHelmChart = 'Helm chart';
const tabLinks = [
    {
        href: `${clustersSecureClusterCrsPath}?tab=Operator`,
        title: titleOperator,
    },
    {
        href: `${clustersSecureClusterCrsPath}?tab=${tabHelmChart}`,
        title: titleHelmChart,
    },
];

function SecureClusterPage(): ReactElement {
    const { search } = useLocation();
    const { tab } = qs.parse(search, { ignoreQueryPrefix: true });
    const isOperator = tab !== tabHelmChart;

    return (
        <>
            <PageSection component="div" variant="light">
                <PageTitle title="Secure a cluster" />
                <Flex direction={{ default: 'column' }}>
                    <Flex direction={{ default: 'column' }}>
                        <Breadcrumb>
                            <BreadcrumbItemLink to={clustersBasePath}>Clusters</BreadcrumbItemLink>
                            <BreadcrumbItemLink to={clustersClusterRegistrationSecretsPath}>
                                Cluster registration secrets
                            </BreadcrumbItemLink>
                            <BreadcrumbItem isActive>{title}</BreadcrumbItem>
                        </Breadcrumb>
                        <Title headingLevel="h1">Secure a cluster with a cluster registration secret</Title>
                    </Flex>
                    <FlexItem>
                        <TabNav
                            currentTabTitle={isOperator ? titleOperator : titleHelmChart}
                            tabLinks={tabLinks}
                        />
                        <Divider component="div" />
                    </FlexItem>
                    {isOperator ? (
                        <SecureClusterUsingOperator headingLevel={headingLevel} />
                    ) : (
                        <SecureClusterUsingHelmChart headingLevel={headingLevel} />
                    )}
                    <Alert
                        variant="info"
                        isInline
                        title="You can use one cluster registration secret to secure at most one cluster."
                        component="p"
                    />
                </Flex>
            </PageSection>
        </>
    );
}

export default SecureClusterPage;

import type { ReactElement } from 'react';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    PageSection,
    Tab,
    TabTitleText,
    Tabs,
    Title,
} from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { clustersBasePath, clustersInitBundlesPath } from 'routePaths';

import SecureClusterUsingHelmChart from './SecureClusterUsingHelmChart';
import SecureClusterUsingOperator from './SecureClusterUsingOperator';

const title = 'Secure a cluster with an init bundle';
const headingLevel = 'h2';

const operatorTab = 'Operator';
const helmChartTab = 'Helm chart';

function SecureClusterPage(): ReactElement {
    const [activeTabKey, setActiveTabKey] = useURLStringUnion('tab', [operatorTab, helmChartTab]);

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
            <PageSection type="tabs">
                <Tabs
                    activeKey={activeTabKey}
                    onSelect={(_event, tabKey) => setActiveTabKey(tabKey)}
                    usePageInsets
                    mountOnEnter
                    unmountOnExit
                >
                    <Tab
                        eventKey={operatorTab}
                        title={<TabTitleText>{operatorTab}</TabTitleText>}
                        tabContentId={operatorTab}
                    >
                        <PageSection>
                            <SecureClusterUsingOperator headingLevel={headingLevel} />
                        </PageSection>
                    </Tab>
                    <Tab
                        eventKey={helmChartTab}
                        title={<TabTitleText>{helmChartTab}</TabTitleText>}
                        tabContentId={helmChartTab}
                    >
                        <PageSection>
                            <SecureClusterUsingHelmChart headingLevel={headingLevel} />
                        </PageSection>
                    </Tab>
                </Tabs>
            </PageSection>
            <PageSection>
                <Alert
                    variant="info"
                    isInline
                    title="You can use one bundle to secure multiple clusters that have the same installation method."
                    component="p"
                />
            </PageSection>
        </>
    );
}

export default SecureClusterPage;

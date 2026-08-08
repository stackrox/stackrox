import type { ReactElement } from 'react';
import { PageSection, Tab, TabContent, Tabs, Title } from '@patternfly/react-core';
import { useLocation, useNavigate } from 'react-router-dom-v5-compat';

import PageTitle from 'Components/PageTitle';
import usePermissions from 'hooks/usePermissions';
import type { ResourceName } from 'types/roleResources';
import {
    vulnerabilityConfigurationReportsPath,
    vulnerabilityViewBasedReportsPath,
} from 'routePaths';

import ConfigReportsTab from './ConfigReportsTab';
import ViewBasedReportsTab from './ViewBasedReportsTab';

type TabType = {
    id: string;
    title: string;
    path: string;
    resourceAccessRequirements: ResourceName[];
    tabContentElement: ReactElement;
};

// Assume resourceAccessRequirements for the route: ['Deployment', 'Image]
const tabs: TabType[] = [
    {
        id: 'report-configuration',
        title: 'Report configurations',
        path: vulnerabilityConfigurationReportsPath,
        resourceAccessRequirements: ['WorkflowAdministration'],
        tabContentElement: <ConfigReportsTab />,
    },
    {
        id: 'view-based-reports',
        title: 'View-based reports',
        path: vulnerabilityViewBasedReportsPath,
        resourceAccessRequirements: [],
        tabContentElement: <ViewBasedReportsTab />,
    },
];

function VulnReportingLayout() {
    const location = useLocation();
    const navigate = useNavigate();

    const { hasReadAccess } = usePermissions();
    const tabsEnabled = tabs.filter((tab) =>
        tab.resourceAccessRequirements.every((resourceName) => hasReadAccess(resourceName))
    );

    const tabIndexFound = tabsEnabled.findIndex((tab) => location.pathname.startsWith(tab.path));
    const activeTabIndex = tabIndexFound >= 0 ? tabIndexFound : 0;

    const onTabSelect = (_event, tabIndex) => {
        navigate(tabsEnabled[tabIndex].path);
    };

    return (
        <>
            <PageTitle title="Image vulnerability reports" />
            <PageSection>
                <Title headingLevel="h1">Image vulnerability reports</Title>
            </PageSection>
            <PageSection type="tabs">
                <Tabs
                    activeKey={activeTabIndex}
                    onSelect={onTabSelect}
                    usePageInsets
                    mountOnEnter
                    unmountOnExit
                >
                    {tabsEnabled.map((tab, index) => (
                        <Tab
                            key={tab.id}
                            eventKey={index}
                            title={tab.title}
                            tabContentId={tab.id}
                        />
                    ))}
                </Tabs>
            </PageSection>
            <TabContent id={tabsEnabled[activeTabIndex].id}>
                {tabsEnabled[activeTabIndex].tabContentElement}
            </TabContent>
        </>
    );
}

export default VulnReportingLayout;

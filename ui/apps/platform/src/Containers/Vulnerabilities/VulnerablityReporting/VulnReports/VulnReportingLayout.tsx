import { PageSection, Tab, Tabs, Title } from '@patternfly/react-core';
import { useLocation, useNavigate } from 'react-router-dom-v5-compat';

import PageTitle from 'Components/PageTitle';
import usePermissions from 'hooks/usePermissions';
import {
    vulnerabilityConfigurationReportsPath,
    vulnerabilityViewBasedReportsPath,
} from 'routePaths';

import ConfigReportsTab from './ConfigReportsTab';
import ViewBasedReportsTab from './ViewBasedReportsTab';

function VulnReportingLayout() {
    const location = useLocation();
    const navigate = useNavigate();

    const { hasReadAccess } = usePermissions();
    const isReportConfigurationEnabled = hasReadAccess('WorkflowAdministration');

    const tabs = [
        ...(isReportConfigurationEnabled
            ? [
                  {
                      id: 'report-configuration',
                      title: 'Report configurations',
                      path: vulnerabilityConfigurationReportsPath,
                      content: <ConfigReportsTab />,
                  },
              ]
            : []),
        {
            id: 'view-based-reports',
            title: 'View-based reports',
            path: vulnerabilityViewBasedReportsPath,
            content: <ViewBasedReportsTab />,
        },
    ];

    const tabIndexFound = tabs.findIndex((tab) => location.pathname.startsWith(tab.path));
    const activeTabIndex = tabIndexFound >= 0 ? tabIndexFound : 0;

    const onTabSelect = (_event, tabIndex) => {
        navigate(tabs[tabIndex].path);
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
                    {tabs.map((tab, index) => (
                        <Tab
                            key={tab.id}
                            eventKey={index}
                            title={tab.title}
                            tabContentId={`${tab.id}-tab-content`}
                        />
                    ))}
                </Tabs>
            </PageSection>
            {tabs[activeTabIndex].content}
        </>
    );
}

export default VulnReportingLayout;

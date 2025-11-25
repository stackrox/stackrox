import { PageSection, Tab, Tabs, Title } from '@patternfly/react-core';
import { Outlet, useLocation, useNavigate } from 'react-router-dom-v5-compat';

import PageTitle from 'Components/PageTitle';
import {
    vulnerabilityConfigurationReportsPath,
    vulnerabilityViewBasedReportsPath,
} from 'routePaths';

const tabs = [
    {
        id: 'report-configuration',
        title: 'Report configurations',
        path: vulnerabilityConfigurationReportsPath,
    },
    {
        id: 'view-based-reports',
        title: 'View-based reports',
        path: vulnerabilityViewBasedReportsPath,
    },
];

function VulnReportingLayout() {
    const location = useLocation();
    const navigate = useNavigate();

    const activeTabIndex = tabs.findIndex((tab) => location.pathname.startsWith(tab.path));

    const onTabSelect = (_event, tabIndex) => {
        navigate(tabs[tabIndex].path);
    };

    return (
        <>
            <PageTitle title="Vulnerability reporting" />
            <PageSection hasBodyWrapper={false}>
                <Title headingLevel="h1">Vulnerability reporting</Title>
            </PageSection>
            <PageSection
                hasBodyWrapper={false}
                padding={{ default: 'noPadding' }}
                className="pf-v6-u-pl-lg pf-v6-u-background-color-100"
            >
                <Tabs activeKey={activeTabIndex} onSelect={onTabSelect}>
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
            <PageSection hasBodyWrapper={false} padding={{ default: 'noPadding' }}>
                <Outlet />
            </PageSection>
        </>
    );
}

export default VulnReportingLayout;

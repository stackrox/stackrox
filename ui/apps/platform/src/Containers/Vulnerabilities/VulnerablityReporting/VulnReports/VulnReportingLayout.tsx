import { PageSection, Tab, Tabs, Title } from '@patternfly/react-core';
import { Outlet, useLocation, useNavigate } from 'react-router-dom-v5-compat';

import PageTitle from 'Components/PageTitle';
import useFeatureFlags from 'hooks/useFeatureFlags';
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

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isEnhancedFilteringEnabled = isFeatureFlagEnabled(
        'ROX_VULNERABILITY_REPORTS_ENHANCED_FILTERING'
    );

    const activeTabIndex = tabs.findIndex((tab) => location.pathname.startsWith(tab.path));

    const onTabSelect = (_event, tabIndex) => {
        navigate(tabs[tabIndex].path);
    };

    return (
        <>
            <PageTitle title="Vulnerability reporting" />
            <PageSection>
                <Title headingLevel="h1">
                    {isEnhancedFilteringEnabled
                        ? 'Image vulnerability reports'
                        : 'Vulnerability reporting'}
                </Title>
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
            <Outlet />
        </>
    );
}

export default VulnReportingLayout;

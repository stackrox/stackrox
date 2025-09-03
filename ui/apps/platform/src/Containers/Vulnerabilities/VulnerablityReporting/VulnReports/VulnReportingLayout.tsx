import React from 'react';
import { PageSection, Title, Tabs, Tab } from '@patternfly/react-core';
import { Outlet, useLocation, useNavigate } from 'react-router-dom-v5-compat';

import useFeatureFlags from 'hooks/useFeatureFlags';
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
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isViewBasedReportsEnabled = !isFeatureFlagEnabled('ROX_VULNERABILITY_VIEW_BASED_REPORTS');

    const location = useLocation();
    const navigate = useNavigate();

    const activeTabIndex = tabs.findIndex((tab) => location.pathname.startsWith(tab.path));

    const onTabSelect = (_event, tabIndex) => {
        navigate(tabs[tabIndex].path);
    };

    return (
        <>
            <PageTitle title="Vulnerability reporting" />
            <PageSection
                variant="light"
                className={`${!isViewBasedReportsEnabled && 'pf-v5-u-pb-0'}`}
            >
                <Title headingLevel="h1">Vulnerability reporting</Title>
            </PageSection>
            {isViewBasedReportsEnabled && (
                <PageSection
                    variant="light"
                    padding={{ default: 'noPadding' }}
                    className="pf-v5-u-pl-lg pf-v5-u-background-color-100"
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
            )}
            <PageSection padding={{ default: 'noPadding' }}>
                <Outlet />
            </PageSection>
        </>
    );
}

export default VulnReportingLayout;

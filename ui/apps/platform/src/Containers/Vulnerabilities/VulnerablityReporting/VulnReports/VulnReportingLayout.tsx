import React from 'react';
import { PageSection, Title, Tabs, Tab } from '@patternfly/react-core';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';

import useFeatureFlags from 'hooks/useFeatureFlags';
import PageTitle from 'Components/PageTitle';
import {
    vulnerabilityConfigurationReportsPath,
    vulnerabilityOnDemandReportsPath,
} from 'routePaths';

const tabs = [
    {
        id: 'report-configuration',
        title: 'Report configurations',
        path: vulnerabilityConfigurationReportsPath,
    },
    { id: 'on-demand-reports', title: 'On-demand reports', path: vulnerabilityOnDemandReportsPath },
];

function VulnReportingLayout() {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isOnDemandReportsEnabled = isFeatureFlagEnabled('ROX_VULNERABILITY_ON_DEMAND_REPORTS');

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
                className={`${!isOnDemandReportsEnabled && 'pf-v5-u-pb-0'}`}
            >
                <Title headingLevel="h1">Vulnerability reporting</Title>
            </PageSection>
            {isOnDemandReportsEnabled && (
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

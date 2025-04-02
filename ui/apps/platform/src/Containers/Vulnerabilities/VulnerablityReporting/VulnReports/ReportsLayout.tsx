import React from 'react';
import { PageSection, Title, Tabs, Tab } from '@patternfly/react-core';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';

import useFeatureFlags from 'hooks/useFeatureFlags';
import PageTitle from 'Components/PageTitle';
import {
    vulnerabilityConfigurationReportsPath,
    vulnerabilityOnDemandReportsPath,
} from 'routePaths';

export const tabStates = ['Report configurations', 'On-demand reports'] as const;

export type TabState = (typeof tabStates)[number];

function ReportsLayout() {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isOnDemandReportsEnabled = !isFeatureFlagEnabled('ROX_VULNERABILITY_ON_DEMAND_REPORTS');

    const location = useLocation();
    const navigate = useNavigate();

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
                    <Tabs
                        activeKey={location.pathname}
                        onSelect={(_e, tab) => {
                            navigate(String(tab));
                        }}
                    >
                        <Tab
                            eventKey={vulnerabilityConfigurationReportsPath}
                            title="Report configurations"
                            tabContentId="report-configurations-tab-content"
                        />
                        <Tab
                            eventKey={vulnerabilityOnDemandReportsPath}
                            title="On-demand reports"
                            tabContentId="on-demand-reports-tab-content"
                        />
                    </Tabs>
                </PageSection>
            )}
            <PageSection padding={{ default: 'noPadding' }}>
                <Outlet />
            </PageSection>
        </>
    );
}

export default ReportsLayout;

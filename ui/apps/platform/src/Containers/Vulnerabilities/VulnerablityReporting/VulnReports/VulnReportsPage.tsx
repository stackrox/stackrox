import React from 'react';
import { PageSection, Title, Tabs, Tab } from '@patternfly/react-core';

import useURLStringUnion from 'hooks/useURLStringUnion';
import useFeatureFlags from 'hooks/useFeatureFlags';
import PageTitle from 'Components/PageTitle';
import ConfigReportsTab from './ConfigReportsTab';
import OnDemandReportsTab from './OnDemandReportsTab';

export const tabStates = ['Report configurations', 'On-demand reports'] as const;

export type TabState = (typeof tabStates)[number];

function VulnReportsPage() {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isOnDemandReportsEnabled = isFeatureFlagEnabled('ROX_VULNERABILITY_ON_DEMAND_REPORTS');

    const [tabState, setTabState] = useURLStringUnion('tab', tabStates);

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
                        activeKey={tabState}
                        onSelect={(_e, tab) => {
                            setTabState(tab);
                        }}
                    >
                        <Tab
                            eventKey="Report configurations"
                            title="Report configurations"
                            tabContentId="report-configurations-tab-content"
                        />
                        <Tab
                            eventKey="On-demand reports"
                            title="On-demand reports"
                            tabContentId="on-demand-reports-tab-content"
                        />
                    </Tabs>
                </PageSection>
            )}
            <PageSection padding={{ default: 'noPadding' }}>
                {tabState === 'Report configurations' && <ConfigReportsTab />}
                {tabState === 'On-demand reports' && <OnDemandReportsTab />}
            </PageSection>
        </>
    );
}

export default VulnReportsPage;

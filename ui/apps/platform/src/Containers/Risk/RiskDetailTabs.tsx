import { PageSection, Tab, TabTitleText, Tabs } from '@patternfly/react-core';
import usePermissions from 'hooks/usePermissions';
import useURLStringUnion from 'hooks/useURLStringUnion';
import type { Deployment } from 'types/deployment.proto';
import type { Risk } from 'services/DeploymentsService';

import DeploymentDetails from './DeploymentDetails';
import RiskIndicators from './Indicators/RiskIndicators';
import ProcessDiscovery from './Process/ProcessDiscovery';

const riskIndicatorsTab = 'Risk Indicators';
const deploymentDetailsTab = 'Deployment Details';
const processDiscoveryTab = 'Process Discovery';

export type RiskSidePanelTabsProps = {
    deployment: Deployment;
    risk: Risk | null | undefined;
};

function RiskDetailTabs({ deployment, risk }) {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForDeploymentExtension = hasReadAccess('DeploymentExtension');

    const [activeTabKey, setActiveTabKey] = useURLStringUnion('contentTab', [
        riskIndicatorsTab,
        deploymentDetailsTab,
        processDiscoveryTab,
    ]);

    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Tabs
                    activeKey={activeTabKey}
                    onSelect={(_event, tabKey) => setActiveTabKey(tabKey)}
                    role="region"
                    inset={{ default: 'insetLg' }}
                >
                    <Tab
                        eventKey={riskIndicatorsTab}
                        title={<TabTitleText>Risk Indicators</TabTitleText>}
                        tabContentId={riskIndicatorsTab}
                    />
                    <Tab
                        eventKey={deploymentDetailsTab}
                        title={<TabTitleText>Deployment Details</TabTitleText>}
                        tabContentId={deploymentDetailsTab}
                    />
                    {hasReadAccessForDeploymentExtension && (
                        <Tab
                            eventKey={processDiscoveryTab}
                            title={<TabTitleText>Process Discovery</TabTitleText>}
                            tabContentId={processDiscoveryTab}
                        />
                    )}
                </Tabs>
            </PageSection>
            <PageSection variant="default" id={activeTabKey}>
                {activeTabKey === riskIndicatorsTab && (
                    <div className="flex flex-col">
                        <RiskIndicators deployment={deployment} risk={risk} />
                    </div>
                )}
                {activeTabKey === deploymentDetailsTab && (
                    <div className="flex flex-1 flex-col relative">
                        <div className="absolute w-full">
                            <DeploymentDetails deployment={deployment} />
                        </div>
                    </div>
                )}
                {activeTabKey === processDiscoveryTab && hasReadAccessForDeploymentExtension && (
                    <div className="flex flex-1 flex-col h-full relative">
                        <ProcessDiscovery deploymentId={deployment.id} />
                    </div>
                )}
            </PageSection>
        </>
    );
}

export default RiskDetailTabs;

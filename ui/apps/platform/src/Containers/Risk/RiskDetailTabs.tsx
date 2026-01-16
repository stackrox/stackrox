import { Flex, PageSection, Tab, TabTitleText, Tabs } from '@patternfly/react-core';

import usePermissions from 'hooks/usePermissions';
import useURLStringUnion from 'hooks/useURLStringUnion';
import type { Deployment } from 'types/deployment.proto';
import type { Risk } from 'services/DeploymentsService';

import DeploymentDetails from './DeploymentDetails';
import ProcessDiscovery from './Process/ProcessDiscovery';
import RiskIndicatorCard from './Indicators/RiskIndicatorCard';

const riskIndicatorsTab = 'Risk indicators';
const deploymentDetailsTab = 'Deployment details';
const processDiscoveryTab = 'Process discovery';

export type RiskDetailTabsProps = {
    deployment: Deployment;
    risk: Risk;
};

function RiskDetailTabs({ deployment, risk }: RiskDetailTabsProps) {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForDeploymentExtension = hasReadAccess('DeploymentExtension');

    const [activeTabKey, setActiveTabKey] = useURLStringUnion('contentTab', [
        riskIndicatorsTab,
        deploymentDetailsTab,
        processDiscoveryTab,
    ]);

    return (
        <>
            <PageSection type="tabs" variant="light" padding={{ default: 'noPadding' }}>
                <Tabs
                    activeKey={activeTabKey}
                    onSelect={(_event, tabKey) => setActiveTabKey(tabKey)}
                    role="region"
                    inset={{ default: 'insetLg' }}
                >
                    <Tab
                        eventKey={riskIndicatorsTab}
                        title={<TabTitleText>{riskIndicatorsTab}</TabTitleText>}
                        tabContentId={riskIndicatorsTab}
                    />
                    <Tab
                        eventKey={deploymentDetailsTab}
                        title={<TabTitleText>{deploymentDetailsTab}</TabTitleText>}
                        tabContentId={deploymentDetailsTab}
                    />
                    {hasReadAccessForDeploymentExtension && (
                        <Tab
                            eventKey={processDiscoveryTab}
                            title={<TabTitleText>{processDiscoveryTab}</TabTitleText>}
                            tabContentId={processDiscoveryTab}
                        />
                    )}
                </Tabs>
            </PageSection>
            <PageSection id={activeTabKey}>
                {activeTabKey === riskIndicatorsTab && (
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsMd' }}
                    >
                        {risk.results.map((result) => (
                            <RiskIndicatorCard key={result.name} result={result} />
                        ))}
                    </Flex>
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

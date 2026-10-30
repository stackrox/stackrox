import { Alert, Flex, Label, PageSection, Tab, TabTitleText, Tabs } from '@patternfly/react-core';

import usePermissions from 'hooks/usePermissions';
import useURLStringUnion from 'hooks/useURLStringUnion';
import type { Deployment } from 'types/deployment.proto';
import type { Risk } from 'services/DeploymentsService';
import type { RiskPriorityClassification } from 'types/risk.proto';

import DeploymentDetails from './DeploymentDetails';
import ProcessDiscovery from './Process/ProcessDiscovery';
import RiskIndicatorCard from './Indicators/RiskIndicatorCard';

const riskIndicatorsTab = 'Risk indicators';
const deploymentDetailsTab = 'Deployment details';
const processDiscoveryTab = 'Process discovery';

export type RiskDetailTabsProps = {
    deployment: Deployment;
    risk: Risk | null | undefined;
};

const riskClassLabels: Record<RiskPriorityClassification, string> = {
    RISK_PRIORITY_UNKNOWN: 'Unknown',
    RISK_PRIORITY_RESPOND_NOW: 'Respond now',
    RISK_PRIORITY_INVESTIGATE_NOW: 'Investigate now',
    RISK_PRIORITY_REMEDIATE_NOW: 'Remediate now',
    RISK_PRIORITY_REMEDIATE: 'Remediate',
    RISK_PRIORITY_IMPROVE_POSTURE: 'Improve posture',
};

const riskClassColors: Record<
    RiskPriorityClassification,
    'red' | 'orange' | 'yellow' | 'blue' | 'green' | 'grey'
> = {
    RISK_PRIORITY_UNKNOWN: 'grey',
    RISK_PRIORITY_RESPOND_NOW: 'red',
    RISK_PRIORITY_INVESTIGATE_NOW: 'orange',
    RISK_PRIORITY_REMEDIATE_NOW: 'red',
    RISK_PRIORITY_REMEDIATE: 'yellow',
    RISK_PRIORITY_IMPROVE_POSTURE: 'blue',
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
            <PageSection type="tabs" padding={{ default: 'noPadding' }}>
                <Tabs
                    activeKey={activeTabKey}
                    onSelect={(_event, tabKey) => setActiveTabKey(tabKey)}
                    role="region"
                    usePageInsets
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
                {activeTabKey === riskIndicatorsTab &&
                    (risk ? (
                        <Flex
                            direction={{ default: 'column' }}
                            spaceItems={{ default: 'spaceItemsMd' }}
                        >
                            {risk.priorityClassification && (
                                <Label color={riskClassColors[risk.priorityClassification]}>
                                    Risk class: {riskClassLabels[risk.priorityClassification]}
                                </Label>
                            )}
                            {risk.results.map((result) => (
                                <RiskIndicatorCard key={result.name} result={result} />
                            ))}
                        </Flex>
                    ) : (
                        <Alert variant="warning" isInline title="Risk not found" component="p">
                            Risk for selected deployment may not have been processed.
                        </Alert>
                    ))}
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

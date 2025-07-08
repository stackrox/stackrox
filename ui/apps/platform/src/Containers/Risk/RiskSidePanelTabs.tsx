import React from 'react';

import Tabs from 'Components/Tabs';
import Tab from 'Components/Tab';
import usePermissions from 'hooks/usePermissions';
import { Deployment } from 'types/deployment.proto';
import type { Risk } from 'types/risk.proto';

import DeploymentDetails from './DeploymentDetails';
import RiskIndicators from './Indicators/RiskIndicators';
import ProcessDiscovery from './Process/ProcessDiscovery';

export type RiskSidePanelTabsProps = {
    deployment: Deployment;
    risk: Risk | null | undefined;
};

function RiskSidePanelTabs({ deployment, risk }) {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForDeploymentExtension = hasReadAccess('DeploymentExtension');

    const riskPanelTabs = [{ text: 'Risk Indicators' }, { text: 'Deployment Details' }];
    if (hasReadAccessForDeploymentExtension) {
        riskPanelTabs.push({ text: 'Process Discovery' });
    }

    return (
        <Tabs headers={riskPanelTabs}>
            <Tab>
                <div className="flex flex-col pb-5">
                    <RiskIndicators deployment={deployment} risk={risk} />
                </div>
            </Tab>
            <Tab>
                <div className="flex flex-1 flex-col relative">
                    <div className="absolute w-full">
                        <DeploymentDetails deployment={deployment} />
                    </div>
                </div>
            </Tab>
            {hasReadAccessForDeploymentExtension && (
                <Tab>
                    <div className="flex flex-1 flex-col h-full relative">
                        <ProcessDiscovery deploymentId={deployment.id} />
                    </div>
                </Tab>
            )}
        </Tabs>
    );
}

export default RiskSidePanelTabs;

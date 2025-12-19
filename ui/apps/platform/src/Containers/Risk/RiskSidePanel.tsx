import { PanelBody, PanelNew } from 'Components/Panel';
import type { DeploymentWithRisk } from 'services/DeploymentsService';

import RiskSidePanelTabs from './RiskSidePanelTabs';

export type RiskSidePanelProps = {
    deploymentWithRisk: DeploymentWithRisk;
};

function RiskSidePanel({ deploymentWithRisk }: RiskSidePanelProps) {
    const { deployment, risk } = deploymentWithRisk;
    return (
        <PanelNew testid="panel">
            <PanelBody>
                <RiskSidePanelTabs deployment={deployment} risk={risk} />
            </PanelBody>
        </PanelNew>
    );
}

export default RiskSidePanel;

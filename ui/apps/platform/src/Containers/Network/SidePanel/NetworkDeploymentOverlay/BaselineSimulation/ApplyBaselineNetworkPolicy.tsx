import React, { ReactElement } from 'react';
import { SuccessButton } from '@stackrox/ui-components';

import { NetworkPolicyModification } from 'Containers/Network/networkTypes';
import { BaselineSimulationResult } from 'Containers/Network/useNetworkBaselineSimulation';
import { applyBaselineNetworkPolicy } from 'services/NetworkService';

export type ApplyBaselineNetworkPolicyProps = {
    deploymentId: string;
    networkPolicy: {
        modification: NetworkPolicyModification;
    };
    stopBaselineSimulation: BaselineSimulationResult['stopBaselineSimulation'];
};

function ApplyBaselineNetworkPolicy({
    deploymentId,
    networkPolicy,
    stopBaselineSimulation,
}: ApplyBaselineNetworkPolicyProps): ReactElement {
    async function applyNetworkPolicy() {
        const { modification } = networkPolicy;
        // TODO: Do proper error handling here
        await applyBaselineNetworkPolicy({
            deploymentId,
            modification,
        });
        stopBaselineSimulation();
        // TODO: Show proper feedback to user on success and error
    }

    return (
        <SuccessButton onClick={applyNetworkPolicy}>
            Apply Baseline As A Network Policy
        </SuccessButton>
    );
}

export default ApplyBaselineNetworkPolicy;

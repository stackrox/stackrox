import React from 'react';
import { Button } from '@patternfly/react-core';
import { useHistory } from 'react-router-dom';

import { networkBasePath } from 'routePaths';
import useURLParameter from 'hooks/useURLParameter';
import { Simulation } from '../utils/getSimulation';

type SimulateNetworkPolicyButtonProps = {
    simulation: Simulation;
    isDisabled: boolean;
};

function SimulateNetworkPolicyButton({ simulation, isDisabled }: SimulateNetworkPolicyButtonProps) {
    const history = useHistory();

    const [, setSimulationQueryValue] = useURLParameter('simulation', undefined);

    function enableNetworkPolicySimulation() {
        history.push(`${networkBasePath}${history.location.search as string}`);
        setSimulationQueryValue('networkPolicy');
    }

    return (
        <Button
            variant="secondary"
            isDisabled={isDisabled || simulation.isOn}
            onClick={enableNetworkPolicySimulation}
        >
            Network policy generator
        </Button>
    );
}

export default SimulateNetworkPolicyButton;

import React from 'react';
import { Button } from '@patternfly/react-core';
import { useHistory } from 'react-router-dom';

import useURLSearch from 'hooks/useURLSearch';
import { networkBasePathPF } from 'routePaths';
import { Simulation } from '../utils/getSimulation';

type SimulateNetworkPolicyButtonProps = {
    simulation: Simulation;
};

function SimulateNetworkPolicyButton({ simulation }: SimulateNetworkPolicyButtonProps) {
    const history = useHistory();
    const { searchFilter, setSearchFilter } = useURLSearch();

    function enableNetworkPolicySimulation() {
        const searchObject = {
            ...searchFilter,
            Simulation: 'network policy',
        };
        history.push(networkBasePathPF);
        setSearchFilter(searchObject);
    }

    return (
        <Button
            variant="secondary"
            isDisabled={simulation.isOn}
            onClick={enableNetworkPolicySimulation}
        >
            Simulate network policy
        </Button>
    );
}

export default SimulateNetworkPolicyButton;

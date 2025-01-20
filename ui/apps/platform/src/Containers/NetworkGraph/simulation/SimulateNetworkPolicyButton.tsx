import React from 'react';
import { Button } from '@patternfly/react-core';
import { useNavigate } from 'react-router-dom';

import { networkBasePath } from 'routePaths';
import useAnalytics, { CLUSTER_LEVEL_SIMULATOR_OPENED } from 'hooks/useAnalytics';
import useURLParameter from 'hooks/useURLParameter';
import useURLSearch from 'hooks/useURLSearch';
import { Simulation } from '../utils/getSimulation';
import { getPropertiesForAnalytics } from '../utils/networkGraphURLUtils';

type SimulateNetworkPolicyButtonProps = {
    simulation: Simulation;
    isDisabled: boolean;
};

function SimulateNetworkPolicyButton({ simulation, isDisabled }: SimulateNetworkPolicyButtonProps) {
    const { analyticsTrack } = useAnalytics();
    const navigate = useNavigate();
    const { searchFilter } = useURLSearch();

    const [, setSimulationQueryValue] = useURLParameter('simulation', undefined);

    function enableNetworkPolicySimulation() {
        const properties = getPropertiesForAnalytics(searchFilter);
        analyticsTrack({
            event: CLUSTER_LEVEL_SIMULATOR_OPENED,
            properties,
        });

        navigate(`${networkBasePath}${location.search}`);

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

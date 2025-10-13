import React from 'react';
import { Button } from '@patternfly/react-core';

import useSimulation from '../hooks/useSimulation';

function SimulateBaselinesButton() {
    const { simulation, setSimulation } = useSimulation();

    function enableBaselineSimulation() {
        setSimulation('baseline');
    }

    return (
        <Button variant="primary" isDisabled={simulation.isOn} onClick={enableBaselineSimulation}>
            Simulate baseline as network policy
        </Button>
    );
}

export default SimulateBaselinesButton;

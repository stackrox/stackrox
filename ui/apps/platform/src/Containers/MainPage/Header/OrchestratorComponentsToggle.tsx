import { useEffect, useState } from 'react';
import type { ReactElement } from 'react';
import { Switch } from '@patternfly/react-core';

import { ORCHESTRATOR_COMPONENTS_KEY } from 'utils/orchestratorComponents';

import './OrchestratorComponentsToggle.css';

const OrchestratorComponentsToggle = (): ReactElement => {
    const [showOrchestratorComponents, setShowOrchestratorComponents] = useState('false');

    useEffect(() => {
        const orchestratorComponentShowState = localStorage.getItem(ORCHESTRATOR_COMPONENTS_KEY);
        if (orchestratorComponentShowState) {
            setShowOrchestratorComponents(orchestratorComponentShowState);
        }
    }, []);

    function handleToggle(value) {
        const storedValue = value ? 'true' : 'false';
        localStorage.setItem(ORCHESTRATOR_COMPONENTS_KEY, storedValue);
        location.reload(); // TODO instead pages could re-render on change to Redux store.
    }

    return (
        <Switch
            id="orchestrator-components-toggle"
            aria-label="Toggle Showing Orchestrator Components"
            hasCheckIcon
            isChecked={showOrchestratorComponents === 'true'}
            label="Show orchestrator components"
            onChange={(_event, value) => handleToggle(value)}
        />
    );
};

export default OrchestratorComponentsToggle;

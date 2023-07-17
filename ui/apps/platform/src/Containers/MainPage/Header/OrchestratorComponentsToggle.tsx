import React, { ReactElement, useState, useEffect } from 'react';
import { Switch } from '@patternfly/react-core';

import { ORCHESTRATOR_COMPONENTS_KEY } from 'utils/orchestratorComponents';

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
        // eslint-disable-next-line no-restricted-globals
        location.reload(); // TODO instead pages could re-render on change to Redux store.
    }

    // TODO: update wrapper classes to PatternFly, like  `pf-u-background-color-100
    return (
        <div className="flex justify-center items-center pr-3 relative" style={{ top: '2px' }}>
            <Switch
                id="orchestrator-components-toggle"
                aria-label="Toggle Showing Orchestrator Components"
                isChecked={showOrchestratorComponents === 'true'}
                onChange={handleToggle}
            />
            <span className="p-2 text-base-600" aria-hidden="true">
                Show Orchestrator Components
            </span>
        </div>
    );
};

export default OrchestratorComponentsToggle;

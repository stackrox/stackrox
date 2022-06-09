import React, { ReactElement, useState, useEffect } from 'react';
import { connect } from 'react-redux';
import { Switch } from '@patternfly/react-core';

import { actions as graphActions } from 'reducers/network/graph';
import useCases from 'constants/useCaseTypes';
import { ORCHESTRATOR_COMPONENTS_KEY } from 'utils/orchestratorComponents';

type OrchestratorComponentsToggleProps = {
    useCase: string;
    updateNetworkNodes: () => void;
};

const OrchestratorComponentsToggle = ({
    useCase,
    updateNetworkNodes,
}: OrchestratorComponentsToggleProps): ReactElement => {
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
        if (useCase === useCases.NETWORK) {
            setShowOrchestratorComponents(storedValue);
            // we don't want to force reload on the network graph since search filters are not URL based
            updateNetworkNodes();
        } else {
            // eslint-disable-next-line no-restricted-globals
            location.reload();
        }
    }

    // TODO: update wrapper classes to PatternFly, like  `pf-u-background-color-100
    return (
        <div
            className="flex justify-center items-center pr-3 font-600 relative"
            style={{ top: '2px' }}
        >
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

const mapDispatchToProps = {
    updateNetworkNodes: graphActions.updateNetworkNodes,
};

export default connect(null, mapDispatchToProps)(OrchestratorComponentsToggle);

import React, { ReactElement, useState, useEffect } from 'react';
import { connect } from 'react-redux';

import { actions as graphActions } from 'reducers/network/graph';
import RadioButtonGroup from 'Components/RadioButtonGroup';
import useCases from 'constants/useCaseTypes';

export const orchestratorComponentOption = [
    {
        value: 'Orchestrator Component:',
        type: 'categoryOption',
    },
    {
        value: 'false',
    },
];

export const ORCHESTRATOR_COMPONENT_KEY = 'showOrchestratorComponents';

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
        const orchestratorComponentShowState = localStorage.getItem(ORCHESTRATOR_COMPONENT_KEY);
        if (orchestratorComponentShowState) {
            setShowOrchestratorComponents(orchestratorComponentShowState);
        }
    }, []);

    const buttons = [
        {
            text: 'Hide',
            value: 'false',
        },
        {
            text: 'Show',
            value: 'true',
        },
    ];
    function handleToggle(value) {
        localStorage.setItem(ORCHESTRATOR_COMPONENT_KEY, value);
        if (useCase === useCases.NETWORK) {
            setShowOrchestratorComponents(value);
            // we don't want to force reload on the network graph since search filters are not URL based
            updateNetworkNodes();
        } else {
            // eslint-disable-next-line no-restricted-globals
            location.reload();
        }
    }

    return (
        <div className="border-base-400 border-dashed border-r p-3 flex flex-col justify-center items-center">
            <RadioButtonGroup
                buttons={buttons}
                onClick={handleToggle}
                selected={showOrchestratorComponents}
                groupClassName="h-auto w-24 my-1"
                testId="orchestrator-components-toggle"
            />
            <div className="font-600 font-condensed uppercase text-base-500 flex justify-center pt-px">
                Orchestrator Components
            </div>
        </div>
    );
};

const mapDispatchToProps = {
    updateNetworkNodes: graphActions.updateNetworkNodes,
};

export default connect(null, mapDispatchToProps)(OrchestratorComponentsToggle);

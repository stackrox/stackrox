import React, { ReactElement, useState, useEffect } from 'react';

import RadioButtonGroup from 'Components/RadioButtonGroup';

export const ORCHESTRATOR_COMPONENT_KEY = 'showOrchestratorComponents';

const OrchestratorComponentsToggle = (): ReactElement => {
    const [showOrchestratorComponents, setShowOrchestratorComponents] = useState('false');

    useEffect(() => {
        const systemComponentShowState = localStorage.getItem(ORCHESTRATOR_COMPONENT_KEY);
        if (systemComponentShowState) {
            setShowOrchestratorComponents(systemComponentShowState);
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
        // eslint-disable-next-line no-restricted-globals
        location.reload();
    }

    return (
        <div className="border-base-400 border-dashed border-r p-3 flex flex-col justify-center items-center">
            <RadioButtonGroup
                buttons={buttons}
                onClick={handleToggle}
                selected={showOrchestratorComponents}
                groupClassName="h-auto w-24 my-1"
            />
            <div className="font-600 font-condensed uppercase text-base-500 flex justify-center pt-px">
                Orchestrator Components
            </div>
        </div>
    );
};

export default OrchestratorComponentsToggle;

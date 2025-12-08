import { useEffect, useState } from 'react';

export const ORCHESTRATOR_COMPONENT_KEY = 'showOrchestratorComponents';

function useOrchestratorComponentsState(): [string, (string) => void] {
    const [showOrchestratorComponents, setShowOrchestratorComponents] = useState('false');

    function setShowOrchestratorComponentsHandler(state) {
        localStorage.setItem(ORCHESTRATOR_COMPONENT_KEY, state);
        setShowOrchestratorComponents(state);
    }

    useEffect(() => {
        const systemComponentShowState = localStorage.getItem(ORCHESTRATOR_COMPONENT_KEY);
        if (systemComponentShowState) {
            setShowOrchestratorComponents(systemComponentShowState);
        }
    }, []);

    return [showOrchestratorComponents, setShowOrchestratorComponentsHandler];
}

export default useOrchestratorComponentsState;

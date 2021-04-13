import { useState, useCallback } from 'react';
import { useDispatch } from 'react-redux';

import { actions } from 'reducers/network/graph';
import { toggleAlertBaselineViolations } from 'services/NetworkService';

type ToggleAlertResult = { isToggling: boolean; error: string | null };

export type ToggleAlert = (enable: boolean) => void;

function useAlertBaselineViolations(deploymentId: string): ToggleAlert {
    const dispatch = useDispatch();
    const [, setResult] = useState<ToggleAlertResult>({ error: null, isToggling: false });

    const toggleAlert = useCallback(
        (enable: boolean) => {
            setResult((prevState) => ({ ...prevState, isToggling: true }));
            const promise = toggleAlertBaselineViolations({
                deploymentId,
                enable,
            });
            promise
                .then(() => {
                    dispatch(actions.updateNetworkNodes());
                    setResult({ isToggling: true, error: null });
                })
                .catch((error) => {
                    setResult({ isToggling: false, error });
                });
        },
        [deploymentId, dispatch]
    );
    return toggleAlert;
}

export default useAlertBaselineViolations;

import { useState } from 'react';
import { toggleAlertBaselineViolations } from 'services/NetworkService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

type Result = {
    isToggling: boolean;
    error: string;
};

type ToggleAlertingOnBaselineViolation = {
    toggleAlertingOnBaselineViolation: (enable: boolean, onSuccessCallback: () => void) => void;
} & Result;

const defaultResult = {
    isToggling: false,
    error: '',
};

function useToggleAlertingOnBaselineViolation(deploymentId): ToggleAlertingOnBaselineViolation {
    const [result, setResult] = useState<Result>(defaultResult);

    function toggleAlertingOnBaselineViolation(enable: boolean, onSuccessCallback: () => void) {
        setResult({ isToggling: true, error: '' });
        toggleAlertBaselineViolations({
            deploymentId,
            enable,
        })
            .then(() => {
                setResult({ isToggling: false, error: '' });
                onSuccessCallback();
            })
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                const errorMessage =
                    message || 'An unknown error occurred while getting the list of clusters';

                setResult({ isToggling: false, error: errorMessage });
            });
    }

    return {
        ...result,
        toggleAlertingOnBaselineViolation,
    };
}

export default useToggleAlertingOnBaselineViolation;

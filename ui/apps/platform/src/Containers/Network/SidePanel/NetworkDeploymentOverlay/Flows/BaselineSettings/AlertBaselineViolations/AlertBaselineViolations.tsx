import React, { ReactElement } from 'react';

import { HelpIcon } from '@stackrox/ui-components';
import ToggleSwitch from 'Components/ToggleSwitch';
import useAlertBaselineViolation from './useAlertBaselineViolations';

export type AlertBaselineViolationsProps = {
    deploymentId: string;
    isEnabled: boolean;
};

const helpDescription = "Trigger violations for network flows that aren't in the baseline.";

function AlertBaselineViolations({
    deploymentId,
    isEnabled,
}: AlertBaselineViolationsProps): ReactElement {
    const toggleAlert = useAlertBaselineViolation(deploymentId);

    function handleChange(): void {
        toggleAlert(!isEnabled);
    }

    return (
        <>
            <div className="flex items-center border border-base-400 rounded px-2">
                <label
                    htmlFor="baselineLock"
                    className="block py-2 text-base-600 font-700 uppercase"
                >
                    Alert On Baseline Violations
                </label>
                <ToggleSwitch
                    id="baselineLock"
                    toggleHandler={handleChange}
                    enabled={isEnabled}
                    small
                />
            </div>
            <div className="ml-2">
                <HelpIcon description={helpDescription} />
            </div>
        </>
    );
}

export default AlertBaselineViolations;

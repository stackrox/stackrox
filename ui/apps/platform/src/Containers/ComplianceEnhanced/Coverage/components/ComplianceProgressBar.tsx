import React from 'react';

import { Progress, ProgressMeasureLocation, Tooltip } from '@patternfly/react-core';

export type ComplianceProgressBarProps = {
    ariaLabel: string;
    isDisabled: boolean;
    passPercentage: number;
    progressBarId: string;
    tooltipText: string;
};

function ComplianceProgressBar({
    ariaLabel,
    isDisabled,
    passPercentage,
    progressBarId,
    tooltipText,
}: ComplianceProgressBarProps) {
    if (isDisabled) {
        return <div>—</div>;
    }
    return (
        <>
            <Progress
                id={progressBarId}
                value={passPercentage}
                measureLocation={ProgressMeasureLocation.outside}
                aria-label={`${ariaLabel} compliance percentage`}
            />
            <Tooltip
                content={<div>{tooltipText}</div>}
                triggerRef={() => document.getElementById(progressBarId) as HTMLButtonElement}
            />
        </>
    );
}

export default ComplianceProgressBar;

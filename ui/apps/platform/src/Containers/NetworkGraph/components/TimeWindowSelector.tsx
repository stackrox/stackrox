import React from 'react';
import { SelectOption } from '@patternfly/react-core';

import { timeWindows } from 'constants/timeWindows';
import SelectSingle from 'Components/SelectSingle/SelectSingle';

type TimeWindowSelectorProps = {
    setTimeWindow: (timeWindow: string) => void;
    timeWindow: string;
    isDisabled: boolean;
};

function TimeWindowSelector({ setTimeWindow, timeWindow, isDisabled }: TimeWindowSelectorProps) {
    const handleSelect = (_name: string, value: string) => {
        setTimeWindow(value);
    };

    return (
        <SelectSingle
            id="time-window-selector"
            toggleAriaLabel="Select time window"
            value={timeWindow}
            handleSelect={handleSelect}
            isDisabled={isDisabled}
        >
            {timeWindows.map((window) => (
                <SelectOption key={window} value={window}>
                    {window}
                </SelectOption>
            ))}
        </SelectSingle>
    );
}

export default TimeWindowSelector;

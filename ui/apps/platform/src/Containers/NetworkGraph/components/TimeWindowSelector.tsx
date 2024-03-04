import React from 'react';
import { Select, SelectOption } from '@patternfly/react-core';

import { timeWindows } from 'constants/timeWindows';
import useToggle from 'hooks/useToggle';

type TimeWindowSelectorProps = {
    setActiveTimeWindow: (timeWindow) => void;
    activeTimeWindow: string;
    isDisabled: boolean;
};

function TimeWindowSelector({
    setActiveTimeWindow,
    activeTimeWindow,
    isDisabled,
}: TimeWindowSelectorProps) {
    const { toggleOff: closeSelect, isOn: isOpen, onToggle } = useToggle();

    function selectTimeWindow(_event, selection) {
        closeSelect();
        setActiveTimeWindow(selection);
    }

    return (
        <Select
            isOpen={isOpen}
            onToggle={onToggle}
            onSelect={selectTimeWindow}
            selections={activeTimeWindow}
            isDisabled={isDisabled}
        >
            {timeWindows.map((window) => (
                <SelectOption key={window} value={window}>
                    {window}
                </SelectOption>
            ))}
        </Select>
    );
}

export default TimeWindowSelector;

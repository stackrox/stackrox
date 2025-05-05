import React from 'react';
import { Select, SelectOption } from '@patternfly/react-core/deprecated';

import { timeWindows } from 'constants/timeWindows';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';

type TimeWindowSelectorProps = {
    setTimeWindow: (timeWindow) => void;
    timeWindow: string;
    isDisabled: boolean;
};

function TimeWindowSelector({ setTimeWindow, timeWindow, isDisabled }: TimeWindowSelectorProps) {
    const { closeSelect, isOpen, onToggle } = useSelectToggle();

    function selectTimeWindow(_event, selection) {
        closeSelect();
        setTimeWindow(selection);
    }

    return (
        <Select
            isOpen={isOpen}
            onToggle={(_e, v) => onToggle(v)}
            onSelect={selectTimeWindow}
            selections={timeWindow}
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

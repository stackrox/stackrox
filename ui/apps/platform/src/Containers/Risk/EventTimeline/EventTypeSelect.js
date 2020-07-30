import React from 'react';

import { selectOptionEventTypes } from 'constants/timelineTypes';
import Select from 'Components/ReactSelect';

const options = [
    { label: 'Show All', value: selectOptionEventTypes.ALL },
    { label: 'Policy Violations', value: selectOptionEventTypes.POLICY_VIOLATION },
    { label: 'Process Activities', value: selectOptionEventTypes.PROCESS_ACTIVITY },
    { label: 'Restarts', value: selectOptionEventTypes.RESTART },
    { label: 'Terminations', value: selectOptionEventTypes.TERMINATION },
];

const EventTypeSelect = ({ selectedEventType, selectEventType }) => {
    return (
        <Select
            className="min-w-43"
            options={options}
            onChange={selectEventType}
            value={selectedEventType}
        />
    );
};

export default EventTypeSelect;

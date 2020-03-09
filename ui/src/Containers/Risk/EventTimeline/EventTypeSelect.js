import React from 'react';

import { eventTypes } from 'constants/timelineTypes';
import Select from 'Components/ReactSelect';

const options = [
    { label: 'Show All', value: eventTypes.ALL },
    { label: 'Policy Violations', value: eventTypes.POLICY_VIOLATION },
    { label: 'Process Activities', value: eventTypes.PROCESS_ACTIVITY },
    { label: 'Restarts', value: eventTypes.RESTART },
    { label: 'Terminations', value: eventTypes.TERMINATION }
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

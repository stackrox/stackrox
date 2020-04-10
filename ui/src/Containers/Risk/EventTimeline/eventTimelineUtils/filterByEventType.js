import { eventTypes } from 'constants/timelineTypes';

const filterByEventType = selectedEventType => event => {
    if (selectedEventType === eventTypes.ALL) return true;
    return event.type === selectedEventType;
};

export default filterByEventType;

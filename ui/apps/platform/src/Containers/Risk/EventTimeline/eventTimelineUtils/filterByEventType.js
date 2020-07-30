import { selectOptionEventTypes } from 'constants/timelineTypes';

const filterByEventType = (selectedEventType) => (event) => {
    if (selectedEventType === selectOptionEventTypes.ALL) return true;
    return event.type === selectedEventType;
};

export default filterByEventType;

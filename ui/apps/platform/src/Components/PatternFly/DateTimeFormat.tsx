import React from 'react';

import { getDateTime } from 'utils/dateUtils';

function DateTimeFormat({ time, isInline = false }) {
    const dateTime = getDateTime(time);

    return isInline ? <span>{dateTime}</span> : <div>{dateTime}</div>;
}

export default DateTimeFormat;

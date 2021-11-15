import React from 'react';

import { getDateTime } from 'utils/dateUtils';

function DateTimeFormat({ time }) {
    const dateTime = getDateTime(time);
    return <div>{dateTime}</div>;
}

export default DateTimeFormat;

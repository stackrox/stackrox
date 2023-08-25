import React from 'react';
import { isValid, parse } from 'date-fns';

import { getDateTime } from 'utils/dateUtils';

function DateTimeFormat({ time, isInline = false }) {
    if (!time || !isValid(parse(time))) {
        return isInline ? <span>—</span> : <div>—</div>;
    }
    const dateTime = getDateTime(time);

    return isInline ? <span>{dateTime}</span> : <div>{dateTime}</div>;
}

export default DateTimeFormat;

import React, { ReactElement } from 'react';
import { isValid, parse } from 'date-fns';
import { getDateTime } from 'utils/dateUtils';
import DateTimeUTCTooltip from 'Components/DateTimeWithUTCTooltip';

type DateTimeFieldProps = {
    date?: string; // ISO 8601 formatted date
};

function DateTimeField({ date = '' }: DateTimeFieldProps): ReactElement {
    if (!date || !isValid(parse(date))) {
        return <span>—</span>;
    }

    return <DateTimeUTCTooltip datetime={date}>{getDateTime(date)}</DateTimeUTCTooltip>;
}

export default DateTimeField;

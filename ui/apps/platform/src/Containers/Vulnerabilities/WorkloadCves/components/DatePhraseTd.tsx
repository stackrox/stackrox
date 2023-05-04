import React from 'react';
import { Tooltip } from '@patternfly/react-core';

import { getDateTime, getDistanceStrictAsPhrase } from 'utils/dateUtils';

export type DatePhraseTdProps = {
    date: string | number | Date | null | undefined;
};

function DatePhraseTd({ date }: DatePhraseTdProps) {
    if (!date) {
        return null;
    }
    return (
        <Tooltip content={getDateTime(date)}>
            <span>{getDistanceStrictAsPhrase(date, new Date())}</span>
        </Tooltip>
    );
}

export default DatePhraseTd;

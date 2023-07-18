import React from 'react';
import { Tooltip } from '@patternfly/react-core';

import { getDateTime, getDistanceStrict, getDistanceStrictAsPhrase } from 'utils/dateUtils';

export type DateDistanceTdProps = {
    date: string | number | Date | null | undefined;
    asPhrase?: boolean;
};

function DateDistanceTd({ date, asPhrase = true }: DateDistanceTdProps) {
    if (!date) {
        return null;
    }
    return (
        <Tooltip content={getDateTime(date)}>
            <span>
                {asPhrase
                    ? getDistanceStrictAsPhrase(date, new Date())
                    : getDistanceStrict(date, new Date())}
            </span>
        </Tooltip>
    );
}

export default DateDistanceTd;

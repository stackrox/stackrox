import React from 'react';

import { getDistanceStrict, getDistanceStrictAsPhrase } from 'utils/dateUtils';
import DateTimeUTCTooltip from './DateTimeWithUTCTooltip';

export type DateDistanceProps = {
    date: string | number | Date | null | undefined;
    asPhrase?: boolean;
};

function DateDistance({ date, asPhrase = true }: DateDistanceProps) {
    if (!date) {
        return null;
    }
    return (
        <DateTimeUTCTooltip datetime={date}>
            <span>
                {asPhrase
                    ? getDistanceStrictAsPhrase(date, new Date())
                    : getDistanceStrict(date, new Date())}
            </span>
        </DateTimeUTCTooltip>
    );
}

export default DateDistance;

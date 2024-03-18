import React from 'react';
import { Tooltip } from '@patternfly/react-core';

import { getDateTime, getDistanceStrict, getDistanceStrictAsPhrase } from 'utils/dateUtils';

export type DateDistanceProps = {
    date: string | number | Date | null | undefined;
    asPhrase?: boolean;
};

function DateDistance({ date, asPhrase = true }: DateDistanceProps) {
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

export default DateDistance;

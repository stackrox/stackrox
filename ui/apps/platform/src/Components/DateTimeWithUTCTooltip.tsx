import React, { ReactNode } from 'react';
import { Tooltip } from '@patternfly/react-core';
import { DateLike, displayDateTimeAsISO8601 } from 'utils/dateUtils';

export type DateTimeUTCTooltipProps = {
    datetime: DateLike;
    children: ReactNode;
};

export default function DateTimeUTCTooltip({ datetime, children }: DateTimeUTCTooltipProps) {
    return (
        <Tooltip content={`UTC: ${displayDateTimeAsISO8601(datetime)}`}>
            <span>{children}</span>
        </Tooltip>
    );
}

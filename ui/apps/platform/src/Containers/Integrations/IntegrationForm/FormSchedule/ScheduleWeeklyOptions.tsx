import React, { ReactElement } from 'react';
import { FormSelectOption } from '@patternfly/react-core';

export const daysOfWeek = [
    'Sunday',
    'Monday',
    'Tuesday',
    'Wednesday',
    'Thursday',
    'Friday',
    'Saturday',
];

function ScheduleWeeklyOptions(): ReactElement {
    return (
        <>
            <FormSelectOption label="Choose one..." value="" isDisabled />
            {daysOfWeek.map((day, i) => {
                return <FormSelectOption key={day} label={day} value={i} />;
            })}
        </>
    );
}

export default ScheduleWeeklyOptions;

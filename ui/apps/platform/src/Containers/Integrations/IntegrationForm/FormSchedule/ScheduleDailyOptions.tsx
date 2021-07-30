import React, { ReactElement } from 'react';
import { FormSelectOption } from '@patternfly/react-core';

const getTimes = () => {
    const times = ['12:00'];
    for (let i = 1; i <= 11; i += 1) {
        if (i < 10) {
            times.push(`0${i}:00`);
        } else {
            times.push(`${i}:00`);
        }
    }
    return times.map((x) => `${x}AM`).concat(times.map((x) => `${x}PM`));
};

export const times = getTimes();

function ScheduleDailyOptions(): ReactElement {
    return (
        <>
            {times.map((time, i) => {
                return <FormSelectOption key={time} label={`${time} UTC`} value={i} />;
            })}
        </>
    );
}

export default ScheduleDailyOptions;

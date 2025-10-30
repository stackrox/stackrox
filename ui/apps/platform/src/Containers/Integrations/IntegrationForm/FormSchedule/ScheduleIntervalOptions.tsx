import type { ReactElement } from 'react';
import { FormSelectOption } from '@patternfly/react-core';

const intervalOptions = [
    { label: 'Daily', value: 'DAILY' },
    { label: 'Weekly', value: 'WEEKLY' },
];

function ScheduleIntervalOptions(): ReactElement {
    return (
        <>
            {intervalOptions.map((option) => {
                return (
                    <FormSelectOption
                        key={option.label}
                        label={option.label}
                        value={option.value}
                    />
                );
            })}
        </>
    );
}

export default ScheduleIntervalOptions;

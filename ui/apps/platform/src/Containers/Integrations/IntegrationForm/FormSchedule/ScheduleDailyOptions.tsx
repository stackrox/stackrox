import React, { ReactElement } from 'react';
import { FormSelectOption } from '@patternfly/react-core';

import { timesOfDay } from '../../utils/integrationUtils';

function ScheduleDailyOptions(): ReactElement {
    return (
        <>
            {timesOfDay.map((time, i) => {
                return <FormSelectOption key={time} label={`${time} UTC`} value={i} />;
            })}
        </>
    );
}

export default ScheduleDailyOptions;

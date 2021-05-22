import React, { ReactElement } from 'react';

import { healthStatusStyles } from '../cluster.helpers';

const { bgColor, fgColor } = healthStatusStyles.UNINITIALIZED;

type HealthStatusNotApplicableProps = { testId: string; isList?: boolean };

function HealthStatusNotApplicable({
    testId,
    isList = false,
}: HealthStatusNotApplicableProps): ReactElement {
    return (
        <div className={`${isList ? 'inline' : ''} leading-normal`} data-testid={testId}>
            <span className={`${bgColor} ${fgColor}`}>Not applicable</span>
        </div>
    );
}

export default HealthStatusNotApplicable;

import React, { ReactElement } from 'react';

import { healthStatusStyles } from '../cluster.helpers';

const { bgColor, fgColor } = healthStatusStyles.UNINITIALIZED;

type HealthStatusNotApplicableProps = { testId: string };

function HealthStatusNotApplicable({ testId }: HealthStatusNotApplicableProps): ReactElement {
    return (
        <div className="leading-normal" data-testid={testId}>
            <span className={`${bgColor} ${fgColor}`}>Not applicable</span>
        </div>
    );
}

export default HealthStatusNotApplicable;

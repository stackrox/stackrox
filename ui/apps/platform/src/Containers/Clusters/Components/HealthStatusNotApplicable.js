import React from 'react';

import { healthStatusStyles } from '../cluster.helpers';

const { bgColor, fgColor } = healthStatusStyles.UNINITIALIZED;

const HealthStatusNotApplicable = () => (
    <div className="leading-normal">
        <span className={`${bgColor} ${fgColor}`}>Not applicable</span>
    </div>
);

export default HealthStatusNotApplicable;

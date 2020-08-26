import React from 'react';
import PropTypes from 'prop-types';

import { healthStatusStyles } from '../cluster.helpers';

const { bgColor, fgColor } = healthStatusStyles.UNINITIALIZED;

const HealthStatusNotApplicable = ({ testId }) => (
    <div className="leading-normal" data-testid={testId}>
        <span className={`${bgColor} ${fgColor}`}>Not applicable</span>
    </div>
);

HealthStatusNotApplicable.propTypes = {
    testId: PropTypes.string.isRequired,
};

export default HealthStatusNotApplicable;

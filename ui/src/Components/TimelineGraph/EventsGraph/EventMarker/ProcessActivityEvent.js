import React from 'react';
import PropTypes from 'prop-types';

const ProcessActivityEvent = ({ width, height }) => {
    const elementHeight = height || width;
    return (
        <polygon
            data-testid="process-activity-event"
            points={`${width / 2},0 0,${elementHeight / 2} ${width /
                2},${elementHeight} ${width},${elementHeight / 2}`}
            fill="var(--alert-600)"
            stroke="var(--alert-600)"
        />
    );
};

ProcessActivityEvent.propTypes = {
    width: PropTypes.number.isRequired,
    height: PropTypes.number
};

ProcessActivityEvent.defaultProps = {
    height: null
};

export default ProcessActivityEvent;

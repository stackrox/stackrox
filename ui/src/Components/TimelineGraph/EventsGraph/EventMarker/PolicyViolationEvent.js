import React from 'react';
import PropTypes from 'prop-types';

const PolicyViolationEvent = ({ width, height }) => {
    const elementHeight = height || width;
    return (
        <rect
            data-testid="policy-violation-event"
            fill="var(--primary-600)"
            width={width}
            height={elementHeight}
            rx={3}
        />
    );
};

PolicyViolationEvent.propTypes = {
    width: PropTypes.number.isRequired,
    height: PropTypes.number
};

PolicyViolationEvent.defaultProps = {
    height: null
};

export default PolicyViolationEvent;

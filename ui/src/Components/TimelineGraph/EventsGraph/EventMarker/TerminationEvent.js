import React from 'react';
import PropTypes from 'prop-types';

const TerminationEvent = ({ height, width }) => {
    const elementHeight = height || width;
    return (
        <polygon
            data-testid="termination-event"
            points={`0,0 ${width / 2},${elementHeight} ${width},0`}
            fill="var(--caution-600)"
            stroke="var(--caution-600)"
        />
    );
};

TerminationEvent.propTypes = {
    width: PropTypes.number.isRequired,
    height: PropTypes.number
};

TerminationEvent.defaultProps = {
    height: null
};

export default TerminationEvent;

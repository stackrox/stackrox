import React from 'react';
import PropTypes from 'prop-types';

const RestartEvent = ({ height, width }) => {
    const elementHeight = height || width;
    return (
        <polygon
            data-testid="restart-event"
            points={`0,${elementHeight} ${width / 2},0 ${width},${elementHeight}`}
            fill="var(--caution-600)"
            stroke="var(--caution-600)"
        />
    );
};

RestartEvent.propTypes = {
    width: PropTypes.number.isRequired,
    height: PropTypes.number
};

RestartEvent.defaultProps = {
    height: null
};

export default RestartEvent;

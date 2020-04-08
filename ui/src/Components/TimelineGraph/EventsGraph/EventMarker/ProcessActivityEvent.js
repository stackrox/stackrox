import React from 'react';
import PropTypes from 'prop-types';

import { eventTypes } from 'constants/timelineTypes';
import EventTooltip from '../EventTooltip';

const ProcessActivityEvent = ({ name, type, timestamp, width, height }) => {
    const elementHeight = height || width;
    return (
        // We wrap the tooltip within the specific event Components because the Tooltip Component
        // doesn't seem to work when wrapping it around the rendered html one level above. I suspect
        // it doesn't work because the D3Anchor renders a <g> while this renders an svg element
        <EventTooltip name={name} type={type} timestamp={timestamp}>
            <polygon
                data-testid="process-activity-event"
                points={`${width / 2},0 0,${elementHeight / 2} ${width /
                    2},${elementHeight} ${width},${elementHeight / 2}`}
                fill="var(--alert-600)"
                stroke="var(--alert-600)"
            />
        </EventTooltip>
    );
};

ProcessActivityEvent.propTypes = {
    name: PropTypes.string.isRequired,
    type: PropTypes.oneOf(Object.values(eventTypes)).isRequired,
    timestamp: PropTypes.string.isRequired,
    width: PropTypes.number.isRequired,
    height: PropTypes.number
};

ProcessActivityEvent.defaultProps = {
    height: null
};

export default ProcessActivityEvent;

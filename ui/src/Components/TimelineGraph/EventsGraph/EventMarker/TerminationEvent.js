import React from 'react';
import PropTypes from 'prop-types';

import { eventTypes } from 'constants/timelineTypes';
import EventTooltip from '../EventTooltip';

const TerminationEvent = ({ name, type, reason, timestamp, width, height }) => {
    const elementHeight = height || width;
    return (
        // We wrap the tooltip within the specific event Components because the Tooltip Component
        // doesn't seem to work when wrapping it around the rendered html one level above. I suspect
        // it doesn't work because the D3Anchor renders a <g> while this renders an svg element
        <EventTooltip name={name} type={type} reason={reason} timestamp={timestamp}>
            <polygon
                data-testid="termination-event"
                points={`0,0 ${width / 2},${elementHeight} ${width},0`}
                fill="var(--caution-600)"
                stroke="var(--caution-600)"
            />
        </EventTooltip>
    );
};

TerminationEvent.propTypes = {
    name: PropTypes.string.isRequired,
    type: PropTypes.oneOf(Object.values(eventTypes)).isRequired,
    reason: PropTypes.string,
    timestamp: PropTypes.string.isRequired,
    width: PropTypes.number.isRequired,
    height: PropTypes.number
};

TerminationEvent.defaultProps = {
    reason: null,
    height: null
};

export default TerminationEvent;

import React from 'react';
import PropTypes from 'prop-types';

import EventMarker from '../EventMarker';

const Events = ({ events, height, translateX, minTimeRange, maxTimeRange, margin, isZooming }) => {
    if (isZooming) {
        return [];
    }

    const eventMarkerSize = Math.max(0, height / 3);
    const eventMarkerOffsetY = Math.max(0, height / 2);

    return events.map(
        ({
            id,
            type,
            name,
            args,
            uid,
            parentName,
            parentUid,
            reason,
            inBaseline,
            differenceInMilliseconds,
            timestamp,
        }) => (
            <EventMarker
                key={id}
                id={id}
                name={name}
                args={args}
                uid={uid}
                parentName={parentName}
                parentUid={parentUid}
                reason={reason}
                type={type}
                timestamp={timestamp}
                inBaseline={inBaseline}
                differenceInMilliseconds={differenceInMilliseconds}
                translateX={translateX}
                translateY={eventMarkerOffsetY}
                size={eventMarkerSize}
                minTimeRange={minTimeRange}
                maxTimeRange={maxTimeRange}
                margin={margin}
            />
        )
    );
};

Events.propTypes = {
    minTimeRange: PropTypes.number.isRequired,
    maxTimeRange: PropTypes.number.isRequired,
    margin: PropTypes.number,
    height: PropTypes.number.isRequired,
    translateX: PropTypes.number,
    events: PropTypes.arrayOf(PropTypes.object),
    isZooming: PropTypes.bool,
};

Events.defaultProps = {
    margin: 0,
    translateX: 0,
    events: [],
    isZooming: false,
};

export default Events;

import React from 'react';
import { scaleLinear } from 'd3-scale';

import D3Anchor from 'Components/D3Anchor';
import { getZoomConfig } from 'Components/TimelineGraph/EventsGraph/ZoomableOverlay/zoomUtils';

const ZoomableOverlay = ({
    translateX,
    translateY,
    width,
    height,
    absoluteMinTimeRange,
    absoluteMaxTimeRange,
    onZoomChange,
    onZoomStart,
    onZoomEnd,
}) => {
    const xScale2 = scaleLinear()
        .domain([absoluteMinTimeRange, absoluteMaxTimeRange])
        .range([0, width]);
    const zoom = getZoomConfig(width, height)
        .on('zoom', zooming)
        .on('start', onZoomStart)
        .on('end', onZoomEnd);

    function zooming(event) {
        if (event.sourceEvent && event.sourceEvent.type === 'end') {
            return;
        }
        if (event.type === 'zoom' && event.sourceEvent && event.sourceEvent.type !== 'zoom') {
            const t = event.transform;
            const domain = t.rescaleX(xScale2).domain();
            const selection = {
                start: domain[0],
                end: domain[1],
            };
            onZoomChange(selection);
        }
    }

    // the "container" argument is a reference to the container for the D3-related element
    function onUpdate(container) {
        container.call(zoom);
    }

    return (
        <D3Anchor
            dataTestId="timeline-zoom-overlay"
            translateX={translateX}
            translateY={translateY}
            onUpdate={onUpdate}
        >
            <rect className="cursor-pointer" width={width} height={height} fill="transparent" />
        </D3Anchor>
    );
};

export default ZoomableOverlay;

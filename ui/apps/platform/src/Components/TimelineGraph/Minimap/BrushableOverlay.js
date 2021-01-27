import React from 'react';
import PropTypes from 'prop-types';
import { brushX } from 'd3-brush';
import { scaleLinear } from 'd3-scale';
import { zoomIdentity } from 'd3-zoom';

import D3Anchor from 'Components/D3Anchor';
import { updateZoomTransform } from 'Components/TimelineGraph/EventsGraph/ZoomableOverlay/zoomUtils';

const BrushableOverlay = ({
    translateX,
    translateY,
    width,
    height,
    minTimeRange,
    maxTimeRange,
    absoluteMinTimeRange,
    absoluteMaxTimeRange,
    onBrushSelectionChange,
    margin,
}) => {
    const xScale = scaleLinear()
        .domain([absoluteMinTimeRange, absoluteMaxTimeRange])
        .range([0, width]);

    function brushEnded(event) {
        if (!event.sourceEvent) {
            return;
        } // Only transition after input.

        // reset to view everything
        if (!event.selection) {
            const selection = {
                start: absoluteMinTimeRange,
                end: absoluteMaxTimeRange,
            };
            const transform = zoomIdentity.scale(1).translate(0, 0); // resets the transform back to the initial state
            updateZoomTransform(transform); // we want to sync the transform of the brush with the zoom transform since their internal states are disjoint
            onBrushSelectionChange(selection);
            return;
        }

        const selection = {
            start: xScale.invert(event.selection[0]),
            end: xScale.invert(event.selection[1]),
        };
        const transform = zoomIdentity
            .scale(width / (event.selection[1] - event.selection[0]))
            .translate(-event.selection[0], 0); // calculate the transform based on the selection
        updateZoomTransform(transform); // we want to sync the transform of the brush with the zoom transform since their internal states are disjoint
        onBrushSelectionChange(selection);
    }

    // the "container" argument is a reference to the container for the D3-related element
    function onUpdate(container) {
        const minHorizontalExtent = margin;
        const maxHorizontalExtent = width - margin;
        const brush = brushX()
            .extent([
                [minHorizontalExtent, 0],
                [maxHorizontalExtent, height],
            ])
            .on('end', brushEnded);
        container
            .call(brush)
            .call(brush.move, [xScale(minTimeRange), xScale(maxTimeRange)])
            .select('rect.selection')
            .style('fill', 'var(--accent-500)')
            .style('stroke', 'var(--accent-500)');
    }

    return (
        <D3Anchor
            dataTestId="timeline-brush"
            translateX={translateX}
            translateY={translateY}
            onUpdate={onUpdate}
        />
    );
};

BrushableOverlay.propTypes = {
    margin: PropTypes.number,
    height: PropTypes.number.isRequired,
    width: PropTypes.number.isRequired,
    translateX: PropTypes.number,
    translateY: PropTypes.number,
    onBrushSelectionChange: PropTypes.func.isRequired,
    minTimeRange: PropTypes.number.isRequired,
    maxTimeRange: PropTypes.number.isRequired,
    absoluteMinTimeRange: PropTypes.number.isRequired,
    absoluteMaxTimeRange: PropTypes.number.isRequired,
};

BrushableOverlay.defaultProps = {
    margin: 0,
    translateX: 0,
    translateY: 0,
};

export default BrushableOverlay;

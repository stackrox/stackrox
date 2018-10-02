import React from 'react';
import * as Icon from 'react-feather';
import { event as d3Event, select as d3Select, zoom as d3Zoom } from 'd3';
import {
    SCALE_EXTENT,
    SCALE_DURATION,
    SCALE_FACTOR
} from 'utils/environmentGraphUtils/environmentGraphConstants';

let zoom;

const zoomHandler = (svg, g) => {
    // Zoom functions
    function zoomed() {
        g.attr('transform', d3Event.transform);
    }

    zoom = d3Zoom()
        .scaleExtent(SCALE_EXTENT)
        .on('zoom', zoomed);

    svg.call(zoom);
};

const zoomIn = () => {
    const svg = d3Select('svg.environment-graph');
    zoom.scaleBy(svg.transition().duration(SCALE_DURATION), SCALE_FACTOR);
};

const zoomOut = () => {
    const svg = d3Select('svg.environment-graph');
    zoom.scaleBy(svg.transition().duration(SCALE_DURATION), 1 / SCALE_FACTOR);
};

const NetworkGraphZoom = () => {
    const svg = d3Select('svg.environment-graph');
    if (!svg) return null;
    // add pan+zoom functionality
    zoomHandler(svg, d3Select('svg.environment-graph g'));

    return (
        <div className="graph-zoom-buttons m-4">
            <button type="button" className="btn-icon btn-primary mb-2" onClick={zoomIn}>
                <Icon.Plus className="h-4 w-4" />
            </button>
            <button type="button" className="btn-icon btn-primary" onClick={zoomOut}>
                <Icon.Minus className="h-4 w-4" />
            </button>
        </div>
    );
};

export default NetworkGraphZoom;

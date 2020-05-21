import { select } from 'd3-selection';
import { zoom as d3Zoom } from 'd3-zoom';

import { getWidth, getHeight } from 'utils/d3Utils';
import mainViewSelector from 'Components/TimelineGraph/MainView/selectors';
import timelineZoomSelector from 'Components/TimelineGraph/EventsGraph/ZoomableOverlay/selectors';

/**
 * Initializes a d3 zoom object with the necessary pre-configured settings
 * @param {string} width - the width of the svg
 * @param {string} height - the height of the svg
 * @returns {Object}
 */
export function getZoomConfig(width, height) {
    const zoom = d3Zoom()
        .scaleExtent([1, 50])
        .translateExtent([
            [0, 0],
            [width, height],
        ])
        .extent([
            [0, 0],
            [width, height],
        ]);
    return zoom;
}

/**
 * This function is used to sync the transform of the brush with the zoom transform
 * since their internal states are disjoint
 * @param {Object} transform - The identity transform, where k = 1, x = y = 0 by default
 */
export function updateZoomTransform(transform) {
    const container = select(timelineZoomSelector);
    const width = getWidth(mainViewSelector);
    const height = getHeight(mainViewSelector);
    const zoom = getZoomConfig(width, height);
    container.call(zoom.transform, transform);
}

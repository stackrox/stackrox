import React from 'react';
import PropTypes from 'prop-types';

import * as Icon from 'react-feather';

const ZoomButtons = props => (
    <div className="graph-zoom-buttons m-4 absolute pin-b pin-network-zoom-buttons-left border-2 border-base-400">
        <button
            type="button"
            className="btn-icon btn-base border-b border-base-300"
            onClick={props.networkGraph && props.networkGraph.zoomIn}
        >
            <Icon.Plus className="h-4 w-4" />
        </button>
        <button
            type="button"
            className="btn-icon btn-base shadow"
            onClick={props.networkGraph && props.networkGraph.zoomOut}
        >
            <Icon.Minus className="h-4 w-4" />
        </button>
    </div>
);

ZoomButtons.propTypes = {
    networkGraph: PropTypes.shape({
        nodes: PropTypes.arrayOf(
            PropTypes.shape({
                id: PropTypes.string.isRequired
            })
        ),
        edges: PropTypes.arrayOf(
            PropTypes.shape({
                source: PropTypes.string.isRequired,
                target: PropTypes.string.isRequired
            })
        ),
        epoch: PropTypes.number,
        zoomIn: PropTypes.func.isRequired,
        zoomOut: PropTypes.func.isRequired
    })
};

ZoomButtons.defaultProps = {
    networkGraph: null
};

export default ZoomButtons;

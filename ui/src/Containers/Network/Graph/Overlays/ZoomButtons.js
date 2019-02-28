import React, { Component } from 'react';
import PropTypes from 'prop-types';

import * as Icon from 'react-feather';

class ZoomButtons extends Component {
    static propTypes = {
        getGraphRef: PropTypes.func.isRequired
    };

    zoomIn = () => {
        const graph = this.props.getGraphRef();
        if (graph) {
            graph.zoomIn();
        }
    };

    zoomOut = () => {
        const graph = this.props.getGraphRef();
        if (graph) {
            graph.zoomOut();
        }
    };

    render() {
        return (
            <div className="graph-zoom-buttons m-4 absolute pin-b pin-network-zoom-buttons-left border-2 border-base-400">
                <button
                    type="button"
                    className="btn-icon btn-base border-b border-base-300"
                    onClick={this.zoomIn}
                >
                    <Icon.Plus className="h-4 w-4" />
                </button>
                <button type="button" className="btn-icon btn-base shadow" onClick={this.zoomOut}>
                    <Icon.Minus className="h-4 w-4" />
                </button>
            </div>
        );
    }
}

export default ZoomButtons;

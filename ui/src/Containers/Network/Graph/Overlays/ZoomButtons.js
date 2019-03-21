import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Button from 'Components/Button';
import * as Icon from 'react-feather';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';

class ZoomButtons extends Component {
    static propTypes = {
        networkGraphRef: PropTypes.shape({
            zoomToFit: PropTypes.func,
            zoomIn: PropTypes.func,
            zoomOut: PropTypes.func
        })
    };

    static defaultProps = {
        networkGraphRef: null
    };

    zoomToFit = () => {
        const graph = this.props.networkGraphRef;
        if (graph) {
            graph.zoomToFit();
        }
    };

    zoomIn = () => {
        const graph = this.props.networkGraphRef;
        if (graph) {
            graph.zoomIn();
        }
    };

    zoomOut = () => {
        const graph = this.props.networkGraphRef;
        if (graph) {
            graph.zoomOut();
        }
    };

    render() {
        return (
            <div className="absolute pin-b pin-network-zoom-buttons-left">
                <div className="m-4 border-2 border-base-400 mb-4">
                    <Button
                        className="btn-icon btn-base border-b border-base-300"
                        icon={<Icon.Maximize className="h-4 w-4" />}
                        onClick={this.zoomToFit}
                    />
                </div>
                <div className="graph-zoom-buttons m-4 border-2 border-base-400">
                    <Button
                        className="btn-icon btn-base border-b border-base-300"
                        icon={<Icon.Plus className="h-4 w-4" />}
                        onClick={this.zoomIn}
                    />
                    <Button
                        className="btn-icon btn-base shadow"
                        icon={<Icon.Minus className="h-4 w-4" />}
                        onClick={this.zoomOut}
                    />
                </div>
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    networkGraphRef: selectors.getNetworkGraphRef
});

export default connect(mapStateToProps)(ZoomButtons);

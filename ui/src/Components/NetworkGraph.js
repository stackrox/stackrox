import React, { Component } from 'react';
import PropTypes from 'prop-types';
import NetworkGraphManager from 'webgl/NetworkGraph/Managers/NetworkGraphManager';

class NetworkGraph extends Component {
    static propTypes = {
        nodes: PropTypes.arrayOf(
            PropTypes.shape({
                deploymentId: PropTypes.string.isRequired
            })
        ).isRequired,
        networkFlowMapping: PropTypes.instanceOf(Map).isRequired,
        onNodeClick: PropTypes.func.isRequired,
        updateKey: PropTypes.number.isRequired
    };

    constructor(props) {
        super(props);
        this.manager = {};
    }

    componentDidMount() {
        if (this.isWebGLAvailable()) {
            this.manager = new NetworkGraphManager(this.networkGraph);
        }
    }

    shouldComponentUpdate(nextProps) {
        if (this.isWebGLAvailable()) {
            if (nextProps.updateKey !== this.props.updateKey) {
                const { nodes, networkFlowMapping } = nextProps;
                this.manager.setUpNetworkData({ nodes, networkFlowMapping });
                this.manager.setOnNodeClick(nextProps.onNodeClick);
            }
        }
        return false;
    }

    componentWillUnmount() {
        if (this.isWebGLAvailable()) {
            this.manager.unbindEventListeners();
        }
    }

    isWebGLAvailable = () => {
        try {
            const canvas = document.createElement('canvas');
            return !!(
                window.WebGLRenderingContext &&
                (canvas.getContext('webgl') || canvas.getContext('experimental-webgl'))
            );
        } catch (e) {
            return false;
        }
    };

    zoomIn = () => this.manager.zoomIn();

    zoomOut = () => this.manager.zoomOut();

    render() {
        return (
            <div className="h-full w-full relative">
                <div
                    className="network-graph network-grid-bg flex h-full w-full"
                    ref={ref => {
                        this.networkGraph = ref;
                    }}
                />
            </div>
        );
    }
}

export default NetworkGraph;

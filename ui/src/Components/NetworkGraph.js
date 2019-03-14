import React, { Component } from 'react';
import PropTypes from 'prop-types';
import NetworkGraphManager from 'webgl/NetworkGraph/Managers/NetworkGraphManager';
import { ALLOWED_STATE } from 'constants/networkGraph';
import { networkGraphWW, getBlobURL } from 'utils/webWorkers';

class NetworkGraph extends Component {
    static propTypes = {
        nodes: PropTypes.arrayOf(
            PropTypes.shape({
                entity: PropTypes.shape({
                    type: PropTypes.string.isRequired,
                    id: PropTypes.string.isRequired,
                    deployment: PropTypes.shape({
                        name: PropTypes.string.isRequired
                    })
                }).isRequired
            })
        ).isRequired,
        networkFlowMapping: PropTypes.shape({}).isRequired,
        onNodeClick: PropTypes.func.isRequired,
        updateKey: PropTypes.number.isRequired,
        filterState: PropTypes.number.isRequired
    };

    constructor(props) {
        super(props);
        this.manager = {};
        this.canvas = null;
        const blobUrl = getBlobURL(networkGraphWW);
        this.worker = new Worker(blobUrl);
    }

    componentDidMount() {
        this.canvas = document.createElement('canvas');
        if (this.isWebGLAvailable()) {
            this.manager = new NetworkGraphManager(this.networkGraph);
        }
    }

    shouldComponentUpdate(nextProps) {
        if (this.isWebGLAvailable()) {
            if (
                nextProps.updateKey !== this.props.updateKey ||
                nextProps.filterState !== this.props.filterState
            ) {
                const { nodes, networkFlowMapping, filterState } = nextProps;
                const filteredNetworkFlowMapping =
                    filterState === ALLOWED_STATE ? {} : networkFlowMapping;
                // Test, used in all examples:
                this.worker.onmessage = e => {
                    if (e.data) {
                        switch (e.data.type) {
                            case 'end':
                                this.manager.setNetworkNodes(e.data.nodes);
                                this.manager.setNetworkLinks(e.data.links);
                                this.manager.setNetworkNamespaces(e.data.namespaces);
                                this.manager.renderNetworkGraph();
                                break;
                            default:
                                break;
                        }
                    }
                };
                this.manager.setUpNetworkData({
                    nodes,
                    worker: this.worker,
                    networkFlowMapping: filteredNetworkFlowMapping
                });
                this.manager.setOnNodeClick(nextProps.onNodeClick);
            }
        }
        return false;
    }

    componentWillUnmount() {
        if (this.isWebGLAvailable()) {
            this.manager.unbindEventListeners();
        }
        this.worker.terminate();
    }

    isWebGLAvailable = () => {
        try {
            return !!(
                window.WebGLRenderingContext &&
                (this.canvas.getContext('webgl') || this.canvas.getContext('experimental-webgl'))
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

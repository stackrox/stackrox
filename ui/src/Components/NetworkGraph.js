import React, { Component } from 'react';
import PropTypes from 'prop-types';
import NetworkGraphManager from 'webgl/NetworkGraph/Managers/NetworkGraphManager';
import { networkGraphWW, getBlobURL } from 'utils/webWorkers';
import { ClipLoader as Loader } from 'react-spinners';

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
                }).isRequired,
                nonIsolatedIngress: PropTypes.bool,
                nonIsolatedEgress: PropTypes.bool
            })
        ).isRequired,
        networkFlowMapping: PropTypes.shape({}).isRequired,
        onNodeClick: PropTypes.func.isRequired,
        updateKey: PropTypes.number.isRequired,
        filterState: PropTypes.number.isRequired,
        isLoading: PropTypes.bool.isRequired,
        setNetworkGraphLoading: PropTypes.func.isRequired
    };

    constructor(props) {
        super(props);
        this.manager = {};
        this.canvas = null;
        this.setUpWebWorker();
    }

    componentDidMount() {
        this.canvas = document.createElement('canvas');
        if (this.isWebGLAvailable()) {
            this.manager = new NetworkGraphManager(this.networkGraph);
            const { nodes, networkFlowMapping, onNodeClick } = this.props;
            this.manager.setUpNetworkData({
                nodes,
                networkFlowMapping,
                worker: this.worker
            });
            this.manager.setOnNodeClick(onNodeClick);
        }
    }

    shouldComponentUpdate(nextProps) {
        this.setUp(nextProps);
        return nextProps.isLoading !== this.props.isLoading;
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

    setUp = nextProps => {
        if (this.isWebGLAvailable()) {
            if (
                nextProps.updateKey !== this.props.updateKey ||
                nextProps.filterState !== this.props.filterState
            ) {
                this.props.setNetworkGraphLoading(true);
                const { nodes, networkFlowMapping } = nextProps;
                this.manager.setUpNetworkData({
                    nodes,
                    networkFlowMapping,
                    worker: this.worker
                });
                this.manager.setOnNodeClick(nextProps.onNodeClick);
            }
        }
        return false;
    };

    setUpWebWorker = () => {
        const blobUrl = getBlobURL(networkGraphWW);
        this.worker = new Worker(blobUrl);
        this.worker.onmessage = ({ data }) => {
            if (!data) return;
            switch (data.type) {
                case 'forceSimulation.end':
                    this.manager.setNetworkLinks(data.links);
                    this.manager.setNetworkNamespaces(data.namespaces);
                    this.manager.renderNetworkGraph();
                    if (data.nodes.length) this.props.setNetworkGraphLoading(false);
                    break;
                default:
                    break;
            }
        };
    };

    zoomIn = () => this.manager.zoomIn();

    zoomOut = () => this.manager.zoomOut();

    renderLoader = () => {
        if (!this.props.isLoading) return null;
        return (
            <div className="flex flex-col items-center text-center">
                <div className="w-10 rounded-full p-2 bg-base-100 shadow-lg mb-4">
                    <Loader loading size={20} color="currentColor" />
                </div>
                <div className="uppercase text-sm tracking-widest font-700">
                    Generating Graph...
                </div>
            </div>
        );
    };

    render() {
        return (
            <div className="h-full w-full relative">
                <div
                    className={`network-graph network-grid-bg flex h-full w-full
                        ${this.props.isLoading ? 'invisible' : ''}`}
                    ref={ref => {
                        this.networkGraph = ref;
                    }}
                />
                <div className="absolute h-full w-full pin-t pointer-events-none">
                    <div className="flex flex-1 h-full items-center justify-center">
                        {this.renderLoader()}
                    </div>
                </div>
            </div>
        );
    }
}

export default NetworkGraph;

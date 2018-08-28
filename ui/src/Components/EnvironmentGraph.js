import React, { Component } from 'react';
import PropTypes from 'prop-types';
import {
    forceSimulation as d3ForceSimulation,
    forceManyBody as d3ForceManyBody,
    forceCenter as d3ForceCenter,
    forceLink as d3ForceLink,
    select as d3Select,
    timeout as d3Timeout,
    event as d3Event,
    zoom as d3Zoom
} from 'd3';
import {
    forceCluster,
    forceCollision,
    getBidirectionalEdges
} from 'utils/environmentGraphUtils/environmentGraphUtils';
import {
    enterNamespaceContainer,
    enterNode,
    updateNode,
    enterLink,
    updateLink,
    enterNamespaceLink,
    updateNamespaceLink,
    updateGraph
} from 'utils/environmentGraphUtils/selectionFunctions';
import {
    MAX_RADIUS,
    CLUSTER_INNER_PADDING,
    NAMESPACE_LABEL_OFFSET,
    SCALE_DURATION,
    SCALE_FACTOR,
    SCALE_EXTENT
} from 'utils/environmentGraphUtils/environmentGraphConstants';
import * as Icon from 'react-feather';
import uniqBy from 'lodash/uniqBy';

let width = 0;
let height = 0;

let namespaces;

let nodes = [];
let edges = [];
let namespaceEdges = [];

let force = d3ForceSimulation(nodes)
    .force('charge', d3ForceManyBody().strength(-50))
    // keep entire simulation balanced around screen center
    .force('center', d3ForceCenter(width / 2, height / 2))
    // cluster by section
    .force('cluster', forceCluster(namespaces).strength(0.9))
    .force('link', d3ForceLink(edges).id(d => d.id))
    .force('collide', forceCollision(nodes));

let zoom;

class EnvironmentGraph extends Component {
    static propTypes = {
        nodes: PropTypes.arrayOf(
            PropTypes.shape({
                id: PropTypes.string.isRequired
            })
        ).isRequired,
        edges: PropTypes.arrayOf(
            PropTypes.shape({
                source: PropTypes.string.isRequired,
                target: PropTypes.string.isRequired
            })
        ).isRequired,
        onNodeClick: PropTypes.func,
        updateKey: PropTypes.number.isRequired
    };

    static defaultProps = {
        onNodeClick: null
    };

    componentDidMount() {
        this.d3Graph = d3Select(this.graph);
    }

    shouldComponentUpdate(nextProps) {
        if (nextProps.updateKey !== this.props.updateKey) {
            this.d3Graph.selectAll('*').remove();

            nodes = this.setUpNodes(nextProps.nodes);
            edges = this.setUpEdges(nextProps.nodes, nextProps.edges);
            namespaceEdges = this.setUpNamespaceEdges(nextProps.nodes, nextProps.edges);

            this.setUpForceSimulation();

            this.setUpNamespaceGroups();

            this.setUpNamespaceContainers();

            this.setUpEdgeElements();

            this.setUpNodeElements();
        }

        return false;
    }

    setUpNodes = propNodes => {
        const namespacesMapping = {};
        const nodeIdToNodeMapping = {};

        nodes.forEach(d => {
            nodeIdToNodeMapping[d.id] = d;
        });

        const newNodes = propNodes.map(node => {
            const d = {
                ...node,
                radius: MAX_RADIUS,
                x: width / 2 + Math.random() * 500,
                y: height / 2 + Math.random() * 500
            };

            if (nodeIdToNodeMapping[d.id]) {
                // if the node already exists, maintain current position
                d.x = nodeIdToNodeMapping[d.id].x;
                d.y = nodeIdToNodeMapping[d.id].y;
            } else {
                // else assign it a random position near the center
                d.x = width / 2 + Math.random() * 500;
                d.y = height / 2 + Math.random() * 500;
            }

            if (!namespacesMapping[d.namespace] || d.internetAccess)
                namespacesMapping[d.namespace] = d;
            return d;
        });

        namespaces = Object.values(namespacesMapping);

        return newNodes;
    };

    setUpEdges = (propNodes, propEdges) => {
        const nodeIdToNodeMapping = {};

        propNodes.forEach(d => {
            nodeIdToNodeMapping[d.id] = d;
        });

        const newEdges = propEdges
            .filter(edge => {
                const sourceNamespace = nodeIdToNodeMapping[edge.source].namespace;
                const targetNamespace = nodeIdToNodeMapping[edge.target].namespace;
                return sourceNamespace === targetNamespace;
            })
            .map(edge => ({ ...edge }));

        return getBidirectionalEdges(newEdges);
    };

    setUpNamespaceEdges = (propNodes, propEdges) => {
        const nodeIdToNodeMapping = {};

        propNodes.forEach(d => {
            nodeIdToNodeMapping[d.id] = d;
        });

        let newNamespaceEdges = propEdges
            .filter(edge => {
                const sourceNamespace = nodeIdToNodeMapping[edge.source].namespace;
                const targetNamespace = nodeIdToNodeMapping[edge.target].namespace;
                return sourceNamespace !== targetNamespace;
            })
            .map(edge => ({
                source: nodeIdToNodeMapping[edge.source].namespace,
                target: nodeIdToNodeMapping[edge.target].namespace,
                id: `${nodeIdToNodeMapping[edge.source].namespace}-${
                    nodeIdToNodeMapping[edge.target].namespace
                }`
            }));

        newNamespaceEdges = uniqBy(newNamespaceEdges, 'id');

        return getBidirectionalEdges(newNamespaceEdges);
    };

    setUpForceSimulation = () => {
        const svg = d3Select('svg.environment-graph');

        width = +svg.node().clientWidth;
        height = +svg.node().clientHeight;

        // add pan+zoom functionality
        this.zoomHandler(svg, d3Select(this.graph));

        force = force
            .nodes(nodes, d => d.id)
            .force(
                'link',
                d3ForceLink(edges)
                    .id(d => d.id)
                    .strength(0)
            )
            .force('center', d3ForceCenter(width / 2, height / 2))
            .force('cluster', forceCluster(namespaces).strength(0.9))
            .force('collide', forceCollision(nodes))
            .on('tick', () => {
                // after force calculation starts, call updateGraph
                // which uses d3 to manipulate the attributes,
                // and React doesn't have to go through lifecycle on each tick
                this.d3Graph.call(selection => {
                    updateGraph(selection, this.d3Graph);
                });
                this.updateNamespaceContainers();
            })
            .alpha(1)
            .stop();

        // restart simulation
        let i = 0;
        const x = nodes.length * nodes.length;
        while (i < x) {
            force.tick();
            i += 1;
        }

        force.restart();
    };

    setUpNamespaceGroups = () => {
        const d3NamespaceGroups = this.d3Graph
            .selectAll('.namespace')
            .data(namespaces, n => n.namespace);
        // logic for creating namespace groups
        d3NamespaceGroups
            .enter()
            .insert('g')
            .call(selection => {
                selection.attr('class', d => `namespace namespace-${d.namespace}`);
            });
        // logic for removing namespace groups
        d3NamespaceGroups.exit().remove();
    };

    setUpNamespaceContainers = () => {
        const d3NamespaceContainer = this.d3Graph
            .selectAll('.container')
            .data(namespaces, n => n.namespace);
        // logic for creating namespace groups
        d3NamespaceContainer
            .enter()
            .insert('g', '.namespace')
            .call(enterNamespaceContainer(this.d3Graph));
        // logic for removing namespace groups
        d3NamespaceContainer.exit().remove();
    };

    setUpNodeElements = () => {
        this.d3Graph = d3Select(this.graph);
        namespaces.forEach(n => {
            const namespaceGroup = this.d3Graph.selectAll(`.namespace-${n.namespace}`);
            const d3Nodes = namespaceGroup
                .selectAll('.node')
                .data(nodes.filter(d => d.namespace === n.namespace));
            // logic for creating nodes
            d3Nodes
                .enter()
                .append('g')
                .call(enterNode(this.props.onNodeClick, this.d3Graph));
            // logic for remove nodes
            d3Nodes.exit().remove();
            // logic for updating nodes
            d3Nodes.call(updateNode);
        });
    };

    setUpEdgeElements = () => {
        // creates the arrow head for edges
        const svg = d3Select('svg.environment-graph');

        // creates svg:defs for the arrow heads
        svg
            .append('svg:defs')
            .selectAll('marker')
            .data(['start', 'end']) // Different link/path types can be defined here
            .enter()
            .append('svg:marker') // This section adds in the arrows
            .attr('id', String)
            .attr('viewBox', '0 -5 10 10')
            .attr('refX', d => (d === 'start' ? 0 : 15))
            .attr('refY', -1.5)
            .attr('markerWidth', 6)
            .attr('markerHeight', 6)
            .attr('orient', 'auto')
            .attr('fill', '#3f4983')
            .append('svg:path')
            .attr('d', d => (d === 'start' ? 'M10,-5L0,0L10,5' : 'M0,-5L10,0L0,5'));

        this.setUpServiceEdgeElements();

        this.setUpNamespaceEdgeElements();
    };

    setUpServiceEdgeElements = () => {
        const d3Links = this.d3Graph
            .selectAll('.link.service')
            .data(edges, link => `${link.source},${link.target}`);
        // logic for creating links
        d3Links
            .enter()
            .insert('path')
            .call(enterLink);
        // logic for removing links
        d3Links.exit().remove();
        // logic for updating links
        d3Links.call(updateLink);
    };

    setUpNamespaceEdgeElements = () => {
        const d3NamespaceLinks = this.d3Graph
            .selectAll('.link.namespace')
            .data(namespaceEdges, link => link.id);
        // logic for creating links
        d3NamespaceLinks
            .enter()
            .insert('path')
            .call(enterNamespaceLink);
        // logic for removing links
        d3NamespaceLinks.exit().remove();
        // logic for updating links
        d3NamespaceLinks.call(selection => {
            updateNamespaceLink(selection, this.d3Graph);
        });
    };

    updateNamespaceContainers = () => {
        d3Timeout(() => {
            const d3NamespaceGroups = this.d3Graph.selectAll('.container');
            d3NamespaceGroups.call(selection => {
                const boundingBoxMapping = {};

                this.d3Graph.selectAll('.namespace').each((d, i, items) => {
                    boundingBoxMapping[d.namespace] = d3Select(items[i])
                        .node()
                        .getBBox();
                });

                selection
                    .selectAll('rect')
                    .attr('x', d => boundingBoxMapping[d.namespace].x - CLUSTER_INNER_PADDING)
                    .attr('y', d => boundingBoxMapping[d.namespace].y - CLUSTER_INNER_PADDING)
                    .attr(
                        'height',
                        d => boundingBoxMapping[d.namespace].height + CLUSTER_INNER_PADDING * 2
                    )
                    .attr(
                        'width',
                        d => boundingBoxMapping[d.namespace].width + CLUSTER_INNER_PADDING * 2
                    );

                selection
                    .selectAll('text')
                    .attr(
                        'x',
                        d =>
                            boundingBoxMapping[d.namespace].x -
                            CLUSTER_INNER_PADDING +
                            (boundingBoxMapping[d.namespace].height + CLUSTER_INNER_PADDING * 2) / 2
                    )
                    .attr(
                        'y',
                        d =>
                            boundingBoxMapping[d.namespace].y -
                            CLUSTER_INNER_PADDING +
                            boundingBoxMapping[d.namespace].height +
                            CLUSTER_INNER_PADDING * 2 +
                            NAMESPACE_LABEL_OFFSET
                    )
                    .text(d => d.namespace);
            });
            this.setUpNamespaceEdgeElements();
        });
    };

    zoomHandler = (svg, g) => {
        // Zoom functions
        function zoomed() {
            g.attr('transform', d3Event.transform);
        }

        zoom = d3Zoom()
            .scaleExtent(SCALE_EXTENT)
            .on('zoom', zoomed);

        svg.call(zoom);
    };

    zoomIn = () => {
        const svg = d3Select('svg.environment-graph');
        zoom.scaleBy(svg.transition().duration(SCALE_DURATION), SCALE_FACTOR);
    };

    zoomOut = () => {
        const svg = d3Select('svg.environment-graph');
        zoom.scaleBy(svg.transition().duration(SCALE_DURATION), 1 / SCALE_FACTOR);
    };

    render() {
        return (
            <div className="h-full w-full relative">
                <svg className="environment-graph" width="100%" height="100%">
                    <g
                        ref={ref => {
                            this.graph = ref;
                        }}
                    />
                </svg>
                <div className="absolute pin-r pin-b m-4">
                    <button className="btn-icon btn-primary mb-2" onClick={this.zoomIn}>
                        <Icon.Plus className="h-4 w-4" />
                    </button>
                    <button className="btn-icon btn-primary" onClick={this.zoomOut}>
                        <Icon.Minus className="h-4 w-4" />
                    </button>
                </div>
            </div>
        );
    }
}

export default EnvironmentGraph;

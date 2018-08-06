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
import { forceCluster, forceCollision } from 'utils/environmentGraphUtils/environmentGraphUtils';
import {
    enterNamespaceContainer,
    enterNode,
    updateNode,
    enterLink,
    updateLink,
    updateGraph
} from 'utils/environmentGraphUtils/selectionFunctions';
import {
    MAX_RADIUS,
    CLUSTER_INNER_PADDING,
    NAMESPACE_LABEL_OFFSET
} from 'utils/environmentGraphUtils/environmentGraphConstants';

let width = 0;
let height = 0;

let namespaces;

let nodes = [];
let edges = [];

let force = d3ForceSimulation(nodes)
    .force('charge', d3ForceManyBody().strength(-50))
    // keep entire simulation balanced around screen center
    .force('center', d3ForceCenter(width / 2, height / 2))
    // cluster by section
    .force('cluster', forceCluster(namespaces).strength(0.9))
    .force('link', d3ForceLink(edges).id(d => d.id))
    .force('collide', forceCollision(nodes));

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
            nodes = this.getNodes(nextProps.nodes);
            edges = this.getEdges(nextProps.edges);

            this.setUpForceSimulation();

            this.setUpNamespaceGroups();

            this.setUpNamespaceContainers();

            this.setUpEdgeElements();

            this.setUpNodeElements();
        }

        return false;
    }

    getNodes = propNodes => {
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

    getEdges = propEdges => {
        const newEdges = propEdges.map(edge => ({ ...edge }));
        return newEdges;
    };

    setUpForceSimulation = () => {
        const svg = d3Select('svg.environment-graph');

        width = +svg.node().clientWidth;
        height = +svg.node().clientHeight;

        // add pan+zoom functionality
        this.zoomHandler(svg, d3Select(this.graph));

        force = force
            .nodes(nodes)
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
                this.d3Graph.call(updateGraph);
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
            .call(enterNamespaceContainer);
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
                .call(enterNode(this.props.onNodeClick));
            // logic for remove nodes
            d3Nodes.exit().remove();
            // logic for updating nodes
            d3Nodes.call(updateNode);
        });
    };

    setUpEdgeElements = () => {
        const d3Links = this.d3Graph
            .selectAll('.link')
            .data(edges, link => `${link.source},${link.target}`);
        // logic for creating links
        d3Links
            .enter()
            .insert('line', '.namespace')
            .call(enterLink);
        // logic for removing links
        d3Links.exit().remove();
        // logic for updating links
        d3Links.call(updateLink);
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
        });
    };

    zoomHandler = (svg, g) => {
        // Zoom functions
        function zoomed() {
            g.attr('transform', d3Event.transform);
        }
        const zoom = d3Zoom().on('zoom', zoomed);
        zoom(svg);
    };

    render() {
        return (
            <svg className="environment-graph" width="100%" height="100%">
                <g
                    ref={ref => {
                        this.graph = ref;
                    }}
                />
            </svg>
        );
    }
}

export default EnvironmentGraph;

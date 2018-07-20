import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as d3 from 'd3';

let width = 0;
let height = 0;
let force = d3
    .forceSimulation()
    .force('link', d3.forceLink().id(d => d.id))
    .force('charge', d3.forceManyBody())
    .force('center', d3.forceCenter(width / 2, height / 2));

const enterNode = selection => {
    selection.classed('node', true);

    selection
        .append('circle')
        .attr('fill', '#3F4884')
        .attr('r', 5);
};

const updateNode = selection => {
    selection.attr('transform', d => `translate(${d.x},${d.y})`);
};

const enterLink = selection => {
    selection
        .classed('link', true)
        .attr('stroke-width', '0.5px')
        .attr('stroke', '#3F4884');
};

const updateLink = selection => {
    selection
        .attr('x1', d => d.source.x)
        .attr('y1', d => d.source.y)
        .attr('x2', d => d.target.x)
        .attr('y2', d => d.target.y);
};

const updateGraph = selection => {
    selection.selectAll('.node').call(updateNode);
    selection.selectAll('.link').call(updateLink);
};

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
        ).isRequired
    };

    componentDidMount() {
        this.d3Graph = d3.select(this.graph);
        force.on('tick', () => {
            // after force calculation starts, call updateGraph
            // which uses d3 to manipulate the attributes,
            // and React doesn't have to go through lifecycle on each tick
            this.d3Graph.call(updateGraph);
        });
    }

    shouldComponentUpdate(nextProps) {
        this.setUpForceSimulation();

        const nodes = nextProps.nodes.map(n => ({ ...n }));
        const edges = nextProps.edges.map(e => ({ ...e }));

        this.setUpNodeElements(nodes);

        this.setUpEdgeElements(edges);

        this.updateForceSimulationData(nodes, edges);

        return false;
    }

    setUpForceSimulation = () => {
        const svg = d3.select('svg.environment-graph');

        width = +svg.node().clientWidth;
        height = +svg.node().clientHeight;

        force = d3
            .forceSimulation()
            .force('link', d3.forceLink().id(d => d.id))
            .force('charge', d3.forceManyBody())
            .force('center', d3.forceCenter(width / 2, height / 2));

        // add pan+zoom functionality
        this.zoomHandler(svg, d3.select(this.graph));
    };

    setUpNodeElements = nodes => {
        this.d3Graph = d3.select(this.graph);
        const d3Nodes = this.d3Graph.selectAll('.node').data(nodes, node => node.id);
        // logic for creating nodes
        d3Nodes
            .enter()
            .append('g')
            .call(enterNode);
        // logic for remove nodes
        d3Nodes.exit().remove();
        // logic for updating nodes
        d3Nodes.call(updateNode);
    };

    setUpEdgeElements = edges => {
        const d3Links = this.d3Graph
            .selectAll('.link')
            .data(edges, link => `${link.source},${link.target}`);
        // logic for creating links
        d3Links
            .enter()
            .insert('line', '.node')
            .call(enterLink);
        // logic for removing links
        d3Links.exit().remove();
        // logic for updating links
        d3Links.call(updateLink);
    };

    updateForceSimulationData = (nodes, edges) => {
        // update force nodes and links
        force.nodes(nodes);
        force.force('link').links(edges);

        // restart simulation
        force.alpha(1).restart();
    };

    zoomHandler = (svg, g) => {
        // Zoom functions
        function zoomed() {
            g.attr('transform', d3.event.transform);
        }
        const zoom = d3.zoom().on('zoom', zoomed);
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

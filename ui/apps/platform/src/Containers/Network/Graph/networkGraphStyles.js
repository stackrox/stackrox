import {
    NS_FONT_SIZE,
    TEXT_MAX_WIDTH,
    NODE_WIDTH,
    NODE_SOLID_BORDER_WIDTH,
    COLORS,
} from 'constants/networkGraph';

const deploymentStyle = {
    width: NODE_WIDTH,
    height: NODE_WIDTH,
    label: 'data(name)',
    'font-size': '6px',
    'text-max-width': TEXT_MAX_WIDTH,
    'text-wrap': 'ellipsis',
    'text-margin-y': '5px',
    'text-valign': 'bottom',
    'font-weight': 'bold',
    'font-family': 'Open Sans',
    'min-zoomed-font-size': '20px',
    color: COLORS.label,
    'z-compound-depth': 'top',
};

// Note: there is no specificity in cytoscape style
// the order of the styles in this array matters
const styles = [
    {
        selector: '.cluster',
        style: {
            'background-color': 'rgba(218, 226, 251)',
            'border-width': '1.5px',
            'border-color': COLORS.inactiveNS,
            events: 'no',
            shape: 'roundrectangle',
            'compound-sizing-wrt-labels': 'include',
            'font-family': 'stackrox, Open Sans',
            'text-margin-y': '8px',
            'text-valign': 'bottom',
            'font-size': NS_FONT_SIZE,
            color: COLORS.label,
            'font-weight': 700,
            label: 'data(name)',
            'background-opacity': '0.5',
            padding: '8px',
            'text-transform': 'uppercase',
            'z-compound-depth': 'auto',
        },
    },
    {
        selector: '.nsGroup',
        style: {
            'background-color': '#fff',
            'border-width': '1.5px',
            'border-color': COLORS.inactiveNS,
            shape: 'roundrectangle',
            'compound-sizing-wrt-labels': 'exclude',
            'font-family': 'stackrox, Open Sans',
            'text-margin-y': '8px',
            'text-valign': 'bottom',
            'font-size': NS_FONT_SIZE,
            color: COLORS.label,
            'font-weight': 700,
            label: 'data(name)',
            padding: '0px',
            'text-transform': 'uppercase',
            'z-compound-depth': 'auto',
        },
    },
    {
        selector: '.internet',
        style: {
            'background-color': '#fff',
            'border-width': '1.5px',
            'border-color': COLORS.inactiveNS,
            shape: 'cutrectangle',
            'compound-sizing-wrt-labels': 'include',
            'font-family': 'stackrox, Open Sans',
            'text-valign': 'center',
            'font-size': NS_FONT_SIZE * 1.8,
            color: COLORS.label,
            'font-weight': 700,
            label: 'External\n  Entities \u2b08',
            'line-height': 1.2,
            padding: '0px',
            'text-transform': 'uppercase',
            'text-wrap': 'wrap',
            width: 'label',
            'z-compound-depth': 'auto',
        },
    },
    {
        selector: '.cidrBlock',
        style: {
            'background-color': '#fff',
            'border-width': '1.5px',
            'border-color': COLORS.inactiveNS,
            shape: 'cutrectangle',
            'compound-sizing-wrt-labels': 'include',
            'font-family': 'stackrox, Open Sans',
            'text-valign': 'center',
            'font-size': NS_FONT_SIZE * 1.2,
            color: COLORS.label,
            'font-weight': 700,
            label: (ele) => {
                const address = ele.data()?.cidr || '';
                const name = ele.data()?.name || '';
                return `${address}\n${name}`;
            },
            'line-height': 1.5,
            padding: '0px',
            'text-transform': 'uppercase',
            'text-wrap': 'wrap',
            width: 'label',
            'z-compound-depth': 'auto',
        },
    },
    {
        selector: 'node.nsHovered',
        style: {
            opacity: 1,
            'border-style': 'solid',
            'border-color': COLORS.hovered,
            'overlay-padding': '3px',
            'overlay-color': 'hsla(227, 85%, 70%, 1)',
            'overlay-opacity': 0.05,
            'z-compound-depth': 'auto',
        },
    },
    {
        selector: 'node.nsSelected',
        style: {
            opacity: 1,
            'border-style': 'solid',
            'border-color': COLORS.selected,
            'overlay-padding': '3px',
            'overlay-color': 'hsla(227, 85%, 60%, 1)',
            'overlay-opacity': 0.05,
            'z-compound-depth': 'auto',
        },
    },
    {
        selector: 'node.nsActive',
        style: {
            'border-style': 'dashed',
            'border-color': COLORS.active,
        },
    },
    {
        selector: 'node.nsActive.nsHovered',
        style: {
            opacity: 1,
            'border-style': 'dashed',
            'border-color': COLORS.hoveredActive,
            'overlay-padding': '3px',
            'overlay-color': 'hsla(227, 85%, 60%, 1)',
            'overlay-opacity': 0.1,
            'z-compound-depth': 'auto',
        },
    },
    {
        selector: 'node.nsActive.nsSelected',
        style: {
            opacity: 1,
            'border-style': 'dashed',
            'border-color': COLORS.selectedActive,
            'overlay-padding': '3px',
            'overlay-color': 'hsla(227, 85%, 50%, 1)',
            'overlay-opacity': 0.1,
            'z-compound-depth': 'auto',
        },
    },
    {
        selector: ':parent > node.deployment',
        style: {
            'background-color': COLORS.inactive,
            ...deploymentStyle,
        },
    },
    {
        selector: 'node.active',
        style: {
            ...deploymentStyle,
            'background-color': COLORS.active,
            'border-style': 'double',
            'border-width': '1px',
            'border-color': COLORS.active,
        },
    },
    {
        selector: 'node.nonIsolated',
        style: {
            ...deploymentStyle,
            'background-color': COLORS.nonIsolated,
            'border-style': 'double',
            'border-width': '1px',
            'border-color': COLORS.nonIsolated,
        },
    },
    {
        selector: 'node.externallyConnected',
        style: {
            width: NODE_WIDTH + NODE_SOLID_BORDER_WIDTH,
            height: NODE_WIDTH + NODE_SOLID_BORDER_WIDTH,
            'background-color': COLORS.externallyConnectedNode,
            'border-style': 'solid',
            'border-width': NODE_SOLID_BORDER_WIDTH,
            'border-color': COLORS.externallyConnectedBorder,
            'text-margin-y': '4px',
        },
    },
    {
        selector: 'node.hovered',
        style: {
            opacity: 1,
            'overlay-padding': '3px',
            'overlay-color': 'hsla(227, 85%, 60%, 1)',
            'overlay-opacity': 0.1,
        },
    },
    {
        selector: 'node.selected',
        style: {
            opacity: 1,
            'overlay-padding': '3px',
            'overlay-color': 'hsla(227, 85%, 50%, 1)',
            'overlay-opacity': 0.1,
        },
    },
    {
        selector: ':parent > node.background',
        style: {
            opacity: 0.5,
            ...deploymentStyle,
        },
    },
    {
        selector: ':parent.background',
        style: {
            opacity: 0.5,
            'z-compound-depth': 'auto',
        },
    },
    {
        selector: ':parent > node.nsEdge',
        style: {
            width: 0.5,
            height: 0.5,
            padding: '0px',
            'background-color': 'white',
        },
    },
    {
        selector: ':parent > node.externalEntitiesEdge',
        style: {
            width: 0.5,
            height: 0.5,
            padding: '0px',
            'background-color': 'white',
        },
    },
    {
        selector: ':parent > node.cidrBlockEdge',
        style: {
            width: 0.5,
            height: 0.5,
            padding: '0px',
            'background-color': 'white',
        },
    },
    {
        selector: 'edge',
        style: {
            width: 1,
            'line-style': 'dashed',
            'line-color': COLORS.edge,
            'z-compound-depth': 'top',
        },
    },
    {
        selector: 'edge.namespace',
        style: {
            'curve-style': 'unbundled-bezier',
            'line-color': COLORS.edge,
            'edge-distances': 'node-position',
            label: 'data(count)',
            'font-size': '8px',
            color: COLORS.edge,
            'font-weight': 500,
            'text-background-opacity': 1,
            'text-background-color': 'white',
            'text-background-shape': 'roundrectangle',
            'text-background-padding': '3px',
            'text-border-opacity': 1,
            'text-border-color': COLORS.edge,
            'text-border-width': 1,
            width: 3,
        },
    },
    {
        selector: 'edge.namespace.simulated.added',
        style: {
            color: COLORS.simulatedStatus.ADDED,
            'text-border-color': COLORS.simulatedStatus.ADDED,
            'line-color': COLORS.simulatedStatus.ADDED,
        },
    },
    {
        selector: 'edge.namespace.simulated.removed',
        style: {
            color: COLORS.simulatedStatus.REMOVED,
            'text-border-color': COLORS.simulatedStatus.REMOVED,
            'line-color': COLORS.simulatedStatus.REMOVED,
        },
    },
    {
        selector: 'edge.namespace.simulated.modified',
        style: {
            color: COLORS.simulatedStatus.MODIFIED,
            'text-border-color': COLORS.simulatedStatus.MODIFIED,
            'line-color': COLORS.simulatedStatus.MODIFIED,
        },
    },
    {
        selector: 'edge.taxi-vertical',
        style: {
            'taxi-direction': 'vertical',
        },
    },
    {
        selector: 'edge.taxi-horizontal',
        style: {
            'taxi-direction': 'horizontal',
        },
    },
    {
        selector: 'edge.inner',
        style: {
            'curve-style': 'haystack',
            'line-style': 'dashed',
            'target-endpoint': 'inside-to-node',
        },
    },

    {
        selector: 'edge.nonIsolated',
        style: {
            display: 'none',
        },
    },
    {
        selector: 'edge.active',
        style: {
            'line-style': 'solid',
            'z-compound-depth': 'top',
        },
    },
    {
        selector: 'edge.unidirectional',
        style: {
            'mid-target-arrow-shape': 'triangle',
            'mid-target-arrow-fill': 'filled',
            'mid-target-arrow-color': COLORS.edge,
            'arrow-scale': 0.6,
        },
    },
    {
        selector: 'edge.bidirectional',
        style: {
            'mid-source-arrow-shape': 'triangle',
            'mid-source-arrow-fill': 'filled',
            'mid-source-arrow-color': COLORS.edge,
            'mid-target-arrow-shape': 'triangle',
            'mid-target-arrow-fill': 'filled',
            'mid-target-arrow-color': COLORS.edge,
            'arrow-scale': 0.6,
        },
    },
    {
        selector: 'edge.externalEdge.unidirectional',
        style: {
            'target-arrow-shape': 'triangle',
            'target-arrow-fill': 'filled',
            'target-arrow-color': COLORS.edge,
            'mid-source-arrow-shape': 'none',
            'mid-target-arrow-shape': 'none',
            'arrow-scale': 1,
        },
    },
    {
        selector: 'edge.externalEdge.bidirectional',
        style: {
            'source-arrow-shape': 'triangle',
            'source-arrow-fill': 'filled',
            'source-arrow-color': COLORS.edge,
            'target-arrow-shape': 'triangle',
            'target-arrow-fill': 'filled',
            'target-arrow-color': COLORS.edge,
            'mid-source-arrow-shape': 'none',
            'mid-target-arrow-shape': 'none',
            'arrow-scale': 1,
        },
    },
    {
        selector: 'edge.inner.withinNS',
        style: {
            'mid-target-arrow-shape': 'none',
            'mid-source-arrow-shape': 'none',
        },
    },
    {
        selector: 'edge.inner.hidden',
        style: {
            display: 'none',
        },
    },
    {
        selector: 'edge.hovered',
        style: {
            opacity: 1,
            color: 'hsl(228, 56%, 63%)',
            'line-color': COLORS.hoveredEdge,
            'text-border-color': COLORS.hoveredEdge,
            'overlay-padding': '3px',
            'mid-source-arrow-color': COLORS.hoveredEdge,
            'mid-target-arrow-color': COLORS.hoveredEdge,
        },
    },
    {
        selector: ':active',
        style: {
            'overlay-padding': '3px',
            'overlay-color': 'hsla(227, 85%, 50%, 1)',
            'overlay-opacity': 0.1,
        },
    },
    {
        selector: 'edge.simulated.added',
        style: {
            'line-color': COLORS.simulatedStatus.ADDED,
            'target-arrow-color': COLORS.simulatedStatus.ADDED,
            'mid-source-arrow-color': COLORS.simulatedStatus.ADDED,
            'mid-target-arrow-color': COLORS.simulatedStatus.ADDED,
        },
    },
    {
        selector: 'edge.simulated.added.hovered',
        style: {
            opacity: 1,
            color: 'hsl(228, 56%, 63%)',
            'line-color': COLORS.hoveredSimulatedStatus.ADDED,
            'text-border-color': COLORS.hoveredSimulatedStatus.ADDED,
            'overlay-padding': '3px',
            'mid-source-arrow-color': COLORS.hoveredSimulatedStatus.ADDED,
            'mid-target-arrow-color': COLORS.hoveredSimulatedStatus.ADDED,
        },
    },
    {
        selector: 'edge.simulated.removed',
        style: {
            'line-color': COLORS.simulatedStatus.REMOVED,
            'target-arrow-color': COLORS.simulatedStatus.REMOVED,
            'mid-source-arrow-color': COLORS.simulatedStatus.REMOVED,
            'mid-target-arrow-color': COLORS.simulatedStatus.REMOVED,
        },
    },
    {
        selector: 'edge.simulated.removed.hovered',
        style: {
            opacity: 1,
            color: 'hsl(228, 56%, 63%)',
            'line-color': COLORS.hoveredSimulatedStatus.REMOVED,
            'text-border-color': COLORS.hoveredSimulatedStatus.REMOVED,
            'overlay-padding': '3px',
            'mid-source-arrow-color': COLORS.hoveredSimulatedStatus.REMOVED,
            'mid-target-arrow-color': COLORS.hoveredSimulatedStatus.REMOVED,
        },
    },
    {
        selector: 'edge.simulated.modified',
        style: {
            'line-color': COLORS.simulatedStatus.MODIFIED,
            'target-arrow-color': COLORS.simulatedStatus.MODIFIED,
            'mid-source-arrow-color': COLORS.simulatedStatus.MODIFIED,
            'mid-target-arrow-color': COLORS.simulatedStatus.MODIFIED,
        },
    },
    {
        selector: 'edge.simulated.modified.hovered',
        style: {
            opacity: 1,
            color: 'hsl(228, 56%, 63%)',
            'line-color': COLORS.hoveredSimulatedStatus.MODIFIED,
            'text-border-color': COLORS.hoveredSimulatedStatus.MODIFIED,
            'overlay-padding': '3px',
            'mid-source-arrow-color': COLORS.hoveredSimulatedStatus.MODIFIED,
            'mid-target-arrow-color': COLORS.hoveredSimulatedStatus.MODIFIED,
        },
    },
];

export default styles;

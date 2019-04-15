import { TEXT_MAX_WIDTH, NODE_WIDTH } from 'constants/networkGraph';

const hoverColor = '#92bae5';
const nonIsolatedColor = 'hsla(2, 78%, 71%, 1)';
const activeColor = 'hsla(214, 74%, 68%, 1)';
const labelColor = 'hsla(231, 22%, 49%, 1)';

const styles = [
    {
        selector: ':parent > node.deployment',
        style: {
            'background-color': 'hsla(229, 24%, 59%, 1)',
            width: NODE_WIDTH,
            height: NODE_WIDTH,
            label: 'data(name)',
            'font-size': '8px',
            'text-max-width': TEXT_MAX_WIDTH,
            'text-wrap': 'ellipsis',
            'text-margin-y': '5px',
            'text-valign': 'bottom',
            'font-weight': 'bold',
            'font-family': 'Open Sans',
            'min-zoomed-font-size': '20px',
            color: labelColor
        }
    },
    {
        selector: 'node.nsEdge',
        style: {
            width: 1,
            height: 1,
            'background-color': 'white'
        }
    },

    {
        selector: 'node.nsHovered',
        style: {
            'background-color': hoverColor,
            'border-style': 'solid',
            'border-width': '1px',
            'border-color': hoverColor
        }
    },
    {
        selector: 'node.nsSelected',
        style: {
            'background-color': activeColor,
            'border-style': 'solid',
            'border-width': '1px',
            'border-color': activeColor
        }
    },
    {
        selector: 'node.nonIsolated',
        style: {
            'background-color': nonIsolatedColor,
            'border-style': 'double',
            'border-width': '1px',
            'border-color': nonIsolatedColor
        }
    },
    {
        selector: 'node.active',
        style: {
            'background-color': activeColor,
            'border-style': 'double',
            'border-width': '1px',
            'border-color': activeColor
        }
    },
    {
        selector: 'node.nsActive',
        style: {
            'background-color': activeColor,
            'border-style': 'dashed',
            'border-width': '2px',
            'border-color': activeColor
        }
    },

    {
        selector: ':parent',
        style: {
            'background-color': '#fff',
            shape: 'roundrectangle',
            'compound-sizing-wrt-labels': 'include',
            'font-family': 'stackrox, Open Sans',
            'text-margin-y': '8px',
            'text-valign': 'bottom',
            'font-size': '18px',
            color: labelColor,
            'text-outline-width': 1,
            'text-outline-opacity': 1,
            'text-outline-color': 'white',
            'font-weight': 600,
            label: 'data(name)',
            padding: 0
        }
    },

    {
        selector: 'edge',
        style: {
            width: 2,
            'line-style': 'dotted',
            'line-color': 'hsla(230, 68%, 87%, 1)'
        }
    },

    {
        selector: 'edge.namespace',
        style: {
            'curve-style': 'taxi',
            'edge-distances': 'node-position',
            'taxi-turn-min-distance': '10px',
            label: 'data(count)'
        }
    },
    {
        selector: 'edge.taxi-vertical',
        style: {
            'taxi-direction': 'vertical'
        }
    },
    {
        selector: 'edge.taxi-horizontal',
        style: {
            'taxi-direction': 'horizontal'
        }
    },

    {
        selector: 'edge.inner',
        style: {
            'curve-style': 'straight',
            'target-endpoint': 'inside-to-node'
        }
    },

    {
        selector: 'node.selected',
        style: {
            'background-color': 'red'
        }
    },

    {
        selector: 'edge.active',
        style: {
            'line-style': 'solid',
            'line-color': 'hsla(229°, 76%, 87%, 1)'
        }
    },

    {
        selector: 'edge.active',
        style: {
            'line-style': 'solid',
            'line-color': 'hsla(229°, 76%, 87%, 1)'
        }
    }
];

export default styles;

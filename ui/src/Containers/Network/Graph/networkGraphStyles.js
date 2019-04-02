const activeColor = 'hsla(214, 74%, 68%, 1)';
const hoverColor = '#92bae5';
const nonIsolatedColor = 'hsla(2, 78%, 71%, 1)';

const styles = [
    {
        selector: ':parent > node',
        style: {
            'background-color': 'hsla(229, 24%, 59%, 1)',
            width: 10,
            height: 10,
            label: 'data(name)',
            'font-size': '8px',
            'text-max-width': '35px',
            'text-wrap': 'ellipsis',
            'text-margin-y': '5px',
            'text-valign': 'bottom',
            'font-weight': 'bold',
            'font-family': 'Open Sans',
            'min-zoomed-font-size': '20px',
            color: 'hsla(231, 22%, 49%, 1)'
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
            'compound-sizing-wrt-labels': 'include'
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
        selector: 'edge.node',
        style: {
            'curve-style': 'unbundled-bezier'
        }
    },

    {
        selector: 'edge.namespace',
        style: {
            'curve-style': 'straight',
            label: 'data(count)'
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
    }
];

export default styles;

const styles = [
    {
        selector: 'node',
        style: {
            'background-color': 'hsla(225, 18%, 32%, 1)',
            width: 10,
            height: 10,
            label: 'data(name)',
            'font-size': '2px',
            'text-max-width': '12px',
            'text-wrap': 'ellipsis',
            'text-margin-y': '-1px'
        }
    },
    {
        selector: 'node.active',
        style: {
            'background-color': 'hsla(225, 65%, 68%, 1)',
            'border-style': 'double',
            'border-width': '1px',
            'border-color': 'hsla(225, 65%, 68%, 1)'
        }
    },

    {
        selector: 'node.nsActive',
        style: {
            'background-color': 'hsla(225, 65%, 68%, 1)',
            'border-style': 'dashed',
            'border-width': '2px',
            'border-color': 'hsla(225, 65%, 68%, 1)'
        }
    },

    {
        selector: ':parent',
        style: {
            'background-color': '#fff',
            'background-opacity': 0.333,
            label: 'data(id)',
            'text-max-width': '40px',
            'font-size': '3px',
            'text-halign': 'center',
            'text-valign': 'top',
            'text-margin-y': 5,
            'text-background-padding': 5
        }
    },

    {
        selector: 'edge',
        style: {
            width: 1,
            'line-color': 'hsla(225, 37%, 36%, 1)'
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
            'line-style': 'dotted'
        }
    }
];

export default styles;

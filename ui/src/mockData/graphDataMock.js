const horizontalBarData = [
    { y: 'PCI', x: 63, hint: { title: 'Tip Title', body: 'This is the body' } },
    { y: 'NIST', x: 84 },
    { y: 'HIPAA', x: 29, axisLink: 'https://google.com/search?q=HIPAA' },
    { y: 'CIS', x: 52 }
];

const horizontalBarDatum = [
    {
        y: 'PCI',
        x: 63,
        hint: { title: 'Tip Title', body: 'This is the body' },
        axisLink: 'https://google.com/search?q=PCI'
    }
];

// Vertical Bar data generator
const verticalBarData = [
    {
        PCIData: [
            {
                x: 'Docker Swarm Dev',
                y: 75,
                hint: { title: 'Hint Title', body: '12 controls failing in this cluster' }
            },
            { x: 'Kube Dev', y: 90 },
            { x: 'Kube Test', y: 60 }
        ],
        NISTData: [
            { x: 'Docker Swarm Dev', y: 50 },
            { x: 'Kube Dev', y: 10 },
            { x: 'Kube Test', y: 30 }
        ],
        HIPAAData: [
            { x: 'Docker Swarm Dev', y: 40 },
            { x: 'Kube Dev', y: 70 },
            { x: 'Kube Test', y: 20 }
        ],

        CISData: [
            { x: 'Docker Swarm Dev', y: 40 },
            { x: 'Kube Dev', y: 20 },
            { x: 'Kube Test', y: 50 }
        ]
    },
    {
        PCIData: [
            {
                x: 'Swarm Dev2',
                y: 10
            },
            { x: 'Kube Dev2', y: 30 },
            { x: 'Kube Test2', y: 50 }
        ],
        NISTData: [
            { x: 'Swarm Dev2', y: 90 },
            { x: 'Kube Dev2', y: 20 },
            { x: 'Kube Test2', y: 50 }
        ],
        HIPAAData: [
            { x: 'Swarm Dev2', y: 10 },
            { x: 'Kube Dev2', y: 50 },
            { x: 'Kube Test2', y: 60 }
        ],

        CISData: [{ x: 'Swarm Dev2', y: 40 }, { x: 'Kube Dev2', y: 20 }, { x: 'Kube Test2', y: 50 }]
    }
];

// Suburst Data Generator
function getSlice(sliceIndex) {
    // Inner slice
    const slice = {
        color: 'var(--base-500)',
        stroke: 2,
        radius: 20,
        radius0: 60,
        style: { stroke: 'var(--base-100)' },
        title: 'Inner Title',
        children: [],
        name: `inner-${sliceIndex}`
    };

    // Outer slice
    for (let i = 0; i < 4; i += 1) {
        slice.children.push({
            title: 'Outer Title',
            size: 1,
            radius: 60,
            radius0: 100,
            style: { stroke: 'white' },
            color: 'var(--base-500)',
            name: `Outer-${sliceIndex}-Inner-${i}`
        });
    }

    return slice;
}

function setSunburstColor(slice, color) {
    // eslint-disable-next-line
    slice.color = color;
    slice.children.forEach((child, i) => {
        // eslint-disable-next-line
        if (i !== 2) child.color = color;
    });
}

function getSunburstData() {
    const root = {
        title: 'Root Title',
        name: 'root',
        color: 'var(--base-100)',
        children: []
    };

    for (let i = 0; i < 12; i += 1) {
        root.children.push(getSlice(i));
    }

    setSunburstColor(root.children[0], 'var(--warning-500)');
    setSunburstColor(root.children[3], 'var(--alert-500)');

    return root;
}

const sunburstData = getSunburstData();
const sunburstLegendData = [
    { title: 'Passing', color: 'var(--base-400)' },
    { title: '< 10% Failing', color: 'var(--warning-400)' },
    { title: '> 10% Failing', color: 'var(--alert-400)' }
];

const listData = [
    {
        name: '1.2.2 - secure and synchronize router configuration file',
        link: 'https://google.com'
    },
    {
        name: '1.3.2 - limit inbound internet traffic to IP addresses with something something',
        link: 'https://google.com'
    },
    {
        name: '1.2.3 - secure and synchronize router configuration file',
        link: 'https://google.com'
    },
    {
        name: '1.3.3 - limit inbound internet traffic to IP addresses with something something',
        link: 'https://google.com'
    },
    {
        name: '1.2.4 - secure and synchronize router configuration file',
        link: 'https://google.com'
    },
    {
        name: '1.3.4 - limit inbound internet traffic to IP addresses with something something',
        link: 'https://google.com'
    }
];

export {
    verticalBarData,
    sunburstData,
    sunburstLegendData,
    horizontalBarData,
    horizontalBarDatum,
    listData
};

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
        PCI: [
            {
                x: 'Docker Swarm Dev',
                y: 75,
                hint: { title: 'PCI Standard', body: '25 controls failing in this cluster' },
                link: '/main/compliance2/PCI?Cluster=Docker Swarm Dev'
            },
            {
                x: 'Kube Dev',
                y: 100,
                link: '/main/compliance2/PCI?Cluster=Kube Dev',
                hint: { title: 'PCI Standard', body: '0 controls failing in this cluster' }
            },
            {
                x: 'Kube Test',
                y: 50,
                link: '/main/compliance2/PCI?Cluster=Kube Test',
                hint: { title: 'PCI Standard', body: '50 controls failing in this cluster' }
            }
        ],
        NIST: [
            {
                x: 'Docker Swarm Dev',
                y: 50,
                link: '/main/compliance2/NIST?Cluster=Docker Swarm Dev',
                hint: { title: 'NIST Standard', body: '50 controls failing in this cluster' }
            },
            {
                x: 'Kube Dev',
                y: 10,
                link: '/main/compliance2/NIST?Cluster=Kube Dev',
                hint: { title: 'NIST Standard', body: '90 controls failing in this cluster' }
            },
            {
                x: 'Kube Test',
                y: 30,
                link: '/main/compliance2/NIST?Cluster=Kube Test',
                hint: { title: 'NIST Standard', body: '70 controls failing in this cluster' }
            }
        ],
        HIPAA: [
            {
                x: 'Docker Swarm Dev',
                y: 40,
                link: '/main/compliance2/HIPAA?Cluster=Docker Swarm Dev',
                hint: { title: 'HIPAA Standard', body: '60 controls failing in this cluster' }
            },
            {
                x: 'Kube Dev',
                y: 70,
                link: '/main/compliance2/HIPAA?Cluster=Kube Dev',
                hint: { title: 'HIPAA Standard', body: '30 controls failing in this cluster' }
            },
            {
                x: 'Kube Test',
                y: 20,
                link: '/main/compliance2/HIPAA?Cluster=Kube Test',
                hint: { title: 'HIPAA Standard', body: '80 controls failing in this cluster' }
            }
        ],

        CIS: [
            {
                x: 'Docker Swarm Dev',
                y: 40,
                link: '/main/compliance2/CIS?Cluster=Docker Swarm Dev',
                hint: { title: 'CIS Standard', body: '60 controls failing in this cluster' }
            },
            {
                x: 'Kube Dev',
                y: 20,
                link: '/main/compliance2/CIS?Cluster=Kube Dev',
                hint: { title: 'CIS Standard', body: '80 controls failing in this cluster' }
            },
            {
                x: 'Kube Test',
                y: 50,
                link: '/main/compliance2/CIS?Cluster=Kube Test',
                hint: { title: 'CIS Standard', body: '50 controls failing in this cluster' }
            }
        ]
    }
];

const multiGaugeData = [
    { title: 'PCI', passing: 75, failing: 25 },
    { title: 'NIST', passing: 25, failing: 75 },
    { title: 'HIPAA', passing: 75, failing: 25 },
    { title: 'CIS', passing: 50, failing: 50 }
];

// Suburst Data Generator
function getSlice(sliceIndex) {
    // Inner slice
    const slice = {
        color: 'var(--tertiary-400)',
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
            color: 'var(--tertiary-400)',
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

    setSunburstColor(root.children[0], 'var(--warning-400)');
    setSunburstColor(root.children[3], 'var(--alert-400)');

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

const verticalSingleBarData = [
    { x: 'PCI', y: 80 },
    { x: 'NIST', y: 55 },
    { x: 'HIPAA', y: 78 },
    { x: 'CIS', y: 50 }
];

export {
    verticalSingleBarData,
    verticalBarData,
    sunburstData,
    sunburstLegendData,
    horizontalBarDatum,
    multiGaugeData,
    horizontalBarData,
    listData
    // arcData
};

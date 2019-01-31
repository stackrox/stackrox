const horizontalBarData = [
    {
        y: 'PCI',
        x: 50,
        axisLink: '/main/compliance2/PCI?GroupBy=Controls',
        barLink: '/main/compliance2/PCI?GroupBy=Clusters',
        hint: { title: 'PCI Standard - 50%', body: '50 controls failing across 3 clusters' }
    },
    {
        y: 'NIST',
        x: 100,
        axisLink: '/main/compliance2/NIST?GroupBy=Controls',
        barLink: '/main/compliance2/NIST?GroupBy=Clusters'
    },
    {
        y: 'HIPAA',
        x: 25,
        axisLink: '/main/compliance2/HIPAA?GroupBy=Controls',
        barLink: '/main/compliance2/HIPAA?GroupBy=Clusters'
    },
    {
        y: 'CIS',
        x: 75,
        axisLink: '/main/compliance2/CIS?GroupBy=Controls',
        barLink: '/main/compliance2/CIS?GroupBy=Clusters'
    }
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

const getSunburstData = () => {
    const numInnerRings = Math.floor(Math.random() * 7) + 3;
    const numOuterRings = Math.floor(Math.random() * 7) + 3;
    const colors = [
        'var(--tertiary-400)',
        'var(--warning-400)',
        'var(--caution-400)',
        'var(--alert-400)'
    ];
    const getColor = value => {
        if (value === 100) return colors[0];
        if (value >= 70) return colors[1];
        if (value >= 50) return colors[2];
        return colors[3];
    };
    const sunburstData = new Array(numInnerRings).fill(0).map((_, innerIndex) => {
        const data = {
            name: `${innerIndex}. Install an maintain a firewall configuration to protect data`,
            link: 'http://google.com',
            children: new Array(numOuterRings).fill(0).map((__, outerIndex) => {
                const value = Math.floor(Math.random() * 100);
                const childData = {
                    color: getColor(value),
                    name: `${innerIndex}.${outerIndex} - A formal process for approving and testing all network connections and changes to the firewall`,
                    value,
                    link: 'http://google.com'
                };
                return childData;
            })
        };
        const val = data.children.reduce(
            (acc, curr) => ({ total: acc.total + 100, passing: acc.passing + curr.value }),
            {
                total: 0,
                passing: 0
            }
        );
        data.value = Math.round((val.passing / val.total) * 100);
        data.color = getColor(data.value);
        return data;
    });
    return sunburstData;
};

const sunburstData = getSunburstData();

const sunburstLegendData = [
    { title: '100%', color: 'var(--base-400)' },
    { title: '> 70%', color: 'var(--warning-400)' },
    { title: '> 50%', color: 'var(--caution-400)' },
    { title: '< 50%', color: 'var(--alert-400)' }
];

const sunburstRootData = [
    {
        text: '4 Categories'
    },
    {
        text: '10 Controls'
    }
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
    sunburstRootData,
    sunburstLegendData,
    horizontalBarDatum,
    multiGaugeData,
    horizontalBarData,
    listData
    // arcData
};

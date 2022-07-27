import { filterModes } from 'constants/networkFilterModes';
import { Edge } from 'Containers/Network/networkTypes';

export const nodes = [
    {
        entity: {
            type: 'DEPLOYMENT',
            id: '6ff5049d-b70a-11ea-a716-025000000001',
            deployment: { name: 'kube-proxy', namespace: 'kube-system', cluster: 'remote' },
        },
        internetAccess: true,
        policyIds: [],
        nonIsolatedIngress: true,
        nonIsolatedEgress: true,
        outEdges: {
            '892424ba-b70d-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
            'e2f0275b-b70c-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
        },
    },
    {
        entity: {
            type: 'DEPLOYMENT',
            id: '8930c942-b70d-11ea-a716-025000000001',
            deployment: { name: 'collector', namespace: 'stackrox', cluster: 'remote' },
        },
        internetAccess: true,
        policyIds: ['88b16ad6-b70d-11ea-a716-025000000001'],
        nonIsolatedIngress: false,
        nonIsolatedEgress: true,
        outEdges: {
            '892424ba-b70d-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
            'e2f0275b-b70c-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
        },
    },
    {
        entity: {
            type: 'DEPLOYMENT',
            id: '96ce06cd-b70a-11ea-a716-025000000001',
            deployment: { name: 'compose-api', namespace: 'docker', cluster: 'remote' },
        },
        internetAccess: true,
        policyIds: [],
        nonIsolatedIngress: true,
        nonIsolatedEgress: true,
        outEdges: {
            '892424ba-b70d-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
            'e2f0275b-b70c-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
        },
    },
    {
        entity: {
            type: 'DEPLOYMENT',
            id: '96d38a65-b70a-11ea-a716-025000000001',
            deployment: { name: 'compose', namespace: 'docker', cluster: 'remote' },
        },
        internetAccess: true,
        policyIds: [],
        nonIsolatedIngress: true,
        nonIsolatedEgress: true,
        outEdges: {
            '892424ba-b70d-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
            'e2f0275b-b70c-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
        },
    },
    {
        entity: {
            type: 'DEPLOYMENT',
            id: '892424ba-b70d-11ea-a716-025000000001',
            deployment: { name: 'sensor', namespace: 'stackrox', cluster: 'remote' },
        },
        internetAccess: true,
        policyIds: ['88b05432-b70d-11ea-a716-025000000001'],
        nonIsolatedIngress: false,
        nonIsolatedEgress: true,
        outEdges: {
            'e2f0275b-b70c-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
        },
    },
    {
        entity: {
            type: 'DEPLOYMENT',
            id: 'e34b1fe2-b70c-11ea-a716-025000000001',
            deployment: { name: 'scanner-db', namespace: 'stackrox', cluster: 'remote' },
        },
        internetAccess: true,
        policyIds: ['e34e6cce-b70c-11ea-a716-025000000001'],
        nonIsolatedIngress: false,
        nonIsolatedEgress: true,
        outEdges: {
            '892424ba-b70d-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
            'e2f0275b-b70c-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
        },
    },
    {
        entity: {
            type: 'DEPLOYMENT',
            id: 'e349f5d4-b70c-11ea-a716-025000000001',
            deployment: { name: 'scanner', namespace: 'stackrox', cluster: 'remote' },
        },
        internetAccess: true,
        policyIds: ['e34dc4b5-b70c-11ea-a716-025000000001'],
        nonIsolatedIngress: false,
        nonIsolatedEgress: true,
        outEdges: {
            '892424ba-b70d-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
            'e34b1fe2-b70c-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
            'e2f0275b-b70c-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
        },
    },
    {
        entity: {
            type: 'DEPLOYMENT',
            id: '6fc26030-b70a-11ea-a716-025000000001',
            deployment: { name: 'coredns', namespace: 'kube-system', cluster: 'remote' },
        },
        internetAccess: true,
        policyIds: [],
        nonIsolatedIngress: true,
        nonIsolatedEgress: true,
        outEdges: {
            '892424ba-b70d-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
            'e2f0275b-b70c-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
        },
    },
    {
        entity: {
            type: 'DEPLOYMENT',
            id: 'e2f0275b-b70c-11ea-a716-025000000001',
            deployment: { name: 'central', namespace: 'stackrox', cluster: 'remote' },
        },
        internetAccess: true,
        policyIds: ['e2f3b506-b70c-11ea-a716-025000000001'],
        nonIsolatedIngress: false,
        nonIsolatedEgress: true,
        outEdges: {
            '892424ba-b70d-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
            'e349f5d4-b70c-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
        },
    },
    {
        entity: {
            type: 'DEPLOYMENT',
            id: '0b7e5849-3663-56e9-9321-27cbf6147418',
            deployment: {
                name: 'static-kube-controller-manager-pods',
                namespace: 'kube-system',
                cluster: 'remote',
            },
        },
        internetAccess: true,
        policyIds: [],
        nonIsolatedIngress: true,
        nonIsolatedEgress: true,
        outEdges: {
            '892424ba-b70d-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
            'e2f0275b-b70c-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
        },
    },
    {
        entity: {
            type: 'DEPLOYMENT',
            id: '9c55e10b-37c1-55ed-8480-60decad25167',
            deployment: {
                name: 'static-kube-apiserver-pods',
                namespace: 'kube-system',
                cluster: 'remote',
            },
        },
        internetAccess: true,
        policyIds: [],
        nonIsolatedIngress: true,
        nonIsolatedEgress: true,
        outEdges: {
            '892424ba-b70d-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
            'e2f0275b-b70c-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
        },
    },
    {
        entity: {
            type: 'DEPLOYMENT',
            id: '76c65318-a0aa-5991-8a32-dcf99d3c8d3b',
            deployment: {
                name: 'static-kube-scheduler-pods',
                namespace: 'kube-system',
                cluster: 'remote',
            },
        },
        internetAccess: true,
        policyIds: [],
        nonIsolatedIngress: true,
        nonIsolatedEgress: true,
        outEdges: {
            '892424ba-b70d-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
            'e2f0275b-b70c-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
        },
    },
    {
        entity: {
            type: 'DEPLOYMENT',
            id: '2e6290d8-4899-5216-9fab-e2e8767684c7',
            deployment: {
                name: 'static-etcd-pods',
                namespace: 'kube-system',
                cluster: 'remote',
            },
        },
        internetAccess: true,
        policyIds: [],
        nonIsolatedIngress: true,
        nonIsolatedEgress: true,
        outEdges: {
            '892424ba-b70d-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
            'e2f0275b-b70c-11ea-a716-025000000001': {
                properties: [
                    {
                        lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                        port: 443,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
            },
        },
    },
];

export const filteredData = nodes.filter((datum) => datum.entity && datum.entity.deployment);

export const links = [
    {
        source: '8930c942-b70d-11ea-a716-025000000001',
        target: '892424ba-b70d-11ea-a716-025000000001',
        sourceName: 'collector',
        targetName: 'sensor',
        sourceNS: 'stackrox',
        targetNS: 'stackrox',
        isActive: true,
        isBetweenNonIsolated: false,
    },
    {
        source: '892424ba-b70d-11ea-a716-025000000001',
        target: 'e2f0275b-b70c-11ea-a716-025000000001',
        sourceName: 'sensor',
        targetName: 'central',
        sourceNS: 'stackrox',
        targetNS: 'stackrox',
        isActive: true,
        isBetweenNonIsolated: false,
    },
    {
        source: 'e349f5d4-b70c-11ea-a716-025000000001',
        target: 'e34b1fe2-b70c-11ea-a716-025000000001',
        sourceName: 'scanner',
        targetName: 'scanner-db',
        sourceNS: 'stackrox',
        targetNS: 'stackrox',
        isActive: true,
        isBetweenNonIsolated: false,
    },
    {
        source: 'e349f5d4-b70c-11ea-a716-025000000001',
        target: '6fc26030-b70a-11ea-a716-025000000001',
        sourceName: 'scanner',
        targetName: 'coredns',
        sourceNS: 'stackrox',
        targetNS: 'kube-system',
        isActive: true,
        isBetweenNonIsolated: false,
    },
    {
        source: 'e349f5d4-b70c-11ea-a716-025000000001',
        target: 'e2f0275b-b70c-11ea-a716-025000000001',
        sourceName: 'scanner',
        targetName: 'central',
        sourceNS: 'stackrox',
        targetNS: 'stackrox',
        isActive: true,
        isBetweenNonIsolated: false,
    },
    {
        source: 'e2f0275b-b70c-11ea-a716-025000000001',
        target: '6fc26030-b70a-11ea-a716-025000000001',
        sourceName: 'central',
        targetName: 'coredns',
        sourceNS: 'stackrox',
        targetNS: 'kube-system',
        isActive: true,
        isBetweenNonIsolated: false,
    },
];

export const nodeSideMap = {
    'kube-system': {
        stackrox: {
            source: 'kube-system_right',
            target: 'stackrox_left',
            sourceSide: 'right',
            targetSide: 'left',
            distance: 294,
        },
        docker: {
            source: 'kube-system_right',
            target: 'docker_left',
            sourceSide: 'right',
            targetSide: 'left',
            distance: 162.9846618550347,
        },
    },
    stackrox: {
        'kube-system': {
            source: 'stackrox_left',
            target: 'kube-system_right',
            sourceSide: 'left',
            targetSide: 'right',
            distance: 294,
        },
        docker: {
            source: 'stackrox_left',
            target: 'docker_right',
            sourceSide: 'left',
            targetSide: 'right',
            distance: 184.62935844550833,
        },
    },
    docker: {
        'kube-system': {
            source: 'docker_left',
            target: 'kube-system_right',
            sourceSide: 'left',
            targetSide: 'right',
            distance: 162.9846618550347,
        },
        stackrox: {
            source: 'docker_right',
            target: 'stackrox_left',
            sourceSide: 'right',
            targetSide: 'left',
            distance: 184.62935844550833,
        },
    },
};

export const configObj = {
    nodes,
    unfilteredLinks: links,
    links,
    filterState: filterModes.active,
    nodeSideMap,
    hoveredNode: null,
    selectedNode: null,
    networkNodeMap: {},
    featureFlags: [],
};

export const namespaceEdges = [
    {
        classes: 'namespace active',
        data: {
            count: 2,
            source: 'kube-system_right',
            sourceNodeNamespace: 'kube-system',
            target: 'stackrox_left',
            targetNodeNamespace: 'stackrox',
            numBidirectionalLinks: 0,
            numUnidirectionalLinks: 2,
            numActiveBidirectionalLinks: 0,
            numActiveUnidirectionalLinks: 2,
            numAllowedBidirectionalLinks: 0,
            numAllowedUnidirectionalLinks: 0,
            portsAndProtocols: [
                {
                    port: '*',
                    protocol: 'L4_PROTOCOL_ANY',
                    traffic: 'ingress',
                },
                {
                    lastActiveTimestamp: '2020-07-31T06:36:29.194197900Z',
                    port: 443,
                    protocol: 'L4_PROTOCOL_TCP',
                    traffic: 'egress',
                },
            ],
            type: 'NAMESPACE_EDGE',
        },
    },
];

export const deploymentList = [
    {
        data: {
            internetAccess: false,
            policyIds: [],
            nonIsolatedIngress: false,
            nonIsolatedEgress: false,
            outEdges: {},
            isActive: false,
            type: 'DEPLOYMENT',
            id: '6ff5049d-b70a-11ea-a716-025000000001',
            name: 'kube-proxy',
            cluster: 'remote',
            parent: 'kube-system',
            edges: [],
            deploymentId: '6ff5049d-b70a-11ea-a716-025000000001',
        },
        classes: 'deployment',
    },
    {
        data: {
            internetAccess: false,
            policyIds: ['88b16ad6-b70d-11ea-a716-025000000001'],
            nonIsolatedIngress: false,
            nonIsolatedEgress: false,
            outEdges: {
                4: {
                    properties: [
                        {
                            port: 8443,
                            protocol: 'L4_PROTOCOL_TCP',
                            lastActiveTimestamp: '2020-07-10T06:42:30.749001500Z',
                        },
                    ],
                },
            },
            isActive: false,
            type: 'DEPLOYMENT',
            id: '8930c942-b70d-11ea-a716-025000000001',
            name: 'collector',
            cluster: 'remote',
            parent: 'stackrox',
            edges: [
                {
                    data: {
                        destNodeId: '892424ba-b70d-11ea-a716-025000000001',
                        destNodeNamespace: 'stackrox',
                        destNodeName: 'sensor',
                        source: '8930c942-b70d-11ea-a716-025000000001',
                        target: '892424ba-b70d-11ea-a716-025000000001',
                        sourceName: 'collector',
                        targetName: 'sensor',
                        sourceNS: 'stackrox',
                        targetNS: 'stackrox',
                        isActive: true,
                        isBetweenNonIsolated: false,
                    },
                    classes: 'edge active  ',
                },
            ],
            deploymentId: '8930c942-b70d-11ea-a716-025000000001',
        },
        classes: 'deployment',
    },
    {
        data: {
            internetAccess: false,
            policyIds: [],
            nonIsolatedIngress: false,
            nonIsolatedEgress: false,
            outEdges: {
                13: {
                    properties: [
                        {
                            port: 443,
                            protocol: 'L4_PROTOCOL_TCP',
                            lastActiveTimestamp: '2020-07-10T06:42:30.749018Z',
                        },
                    ],
                },
            },
            isActive: false,
            type: 'DEPLOYMENT',
            id: '96ce06cd-b70a-11ea-a716-025000000001',
            name: 'compose-api',
            cluster: 'remote',
            parent: 'docker',
            edges: [],
            deploymentId: '96ce06cd-b70a-11ea-a716-025000000001',
        },
        classes: 'deployment',
    },
    {
        data: {
            internetAccess: false,
            policyIds: [],
            nonIsolatedIngress: false,
            nonIsolatedEgress: false,
            outEdges: {
                13: {
                    properties: [
                        {
                            port: 443,
                            protocol: 'L4_PROTOCOL_TCP',
                            lastActiveTimestamp: '2020-07-10T06:42:30.749037500Z',
                        },
                    ],
                },
            },
            isActive: false,
            type: 'DEPLOYMENT',
            id: '96d38a65-b70a-11ea-a716-025000000001',
            name: 'compose',
            cluster: 'remote',
            parent: 'docker',
            edges: [],
            deploymentId: '96d38a65-b70a-11ea-a716-025000000001',
        },
        classes: 'deployment',
    },
    {
        data: {
            internetAccess: false,
            policyIds: ['88b05432-b70d-11ea-a716-025000000001'],
            nonIsolatedIngress: false,
            nonIsolatedEgress: false,
            outEdges: {
                8: {
                    properties: [
                        {
                            port: 8443,
                            protocol: 'L4_PROTOCOL_TCP',
                            lastActiveTimestamp: '2020-07-10T06:42:30.748966400Z',
                        },
                    ],
                },
                13: {
                    properties: [
                        {
                            port: 443,
                            protocol: 'L4_PROTOCOL_TCP',
                            lastActiveTimestamp: '2020-07-10T06:42:30.748984400Z',
                        },
                    ],
                },
            },
            isActive: false,
            type: 'DEPLOYMENT',
            id: '892424ba-b70d-11ea-a716-025000000001',
            name: 'sensor',
            cluster: 'remote',
            parent: 'stackrox',
            edges: [
                {
                    data: {
                        destNodeId: '8930c942-b70d-11ea-a716-025000000001',
                        destNodeNamespace: 'stackrox',
                        destNodeName: 'collector',
                        source: '8930c942-b70d-11ea-a716-025000000001',
                        target: '892424ba-b70d-11ea-a716-025000000001',
                        sourceName: 'collector',
                        targetName: 'sensor',
                        sourceNS: 'stackrox',
                        targetNS: 'stackrox',
                        isActive: true,
                        isBetweenNonIsolated: false,
                    },
                    classes: 'edge active  ',
                },
                {
                    data: {
                        destNodeId: 'e2f0275b-b70c-11ea-a716-025000000001',
                        destNodeNamespace: 'stackrox',
                        destNodeName: 'central',
                        source: '892424ba-b70d-11ea-a716-025000000001',
                        target: 'e2f0275b-b70c-11ea-a716-025000000001',
                        sourceName: 'sensor',
                        targetName: 'central',
                        sourceNS: 'stackrox',
                        targetNS: 'stackrox',
                        isActive: true,
                        isBetweenNonIsolated: false,
                    },
                    classes: 'edge active  ',
                },
            ],
            deploymentId: '892424ba-b70d-11ea-a716-025000000001',
        },
        classes: 'deployment',
    },
    {
        data: {
            internetAccess: false,
            policyIds: ['e34e6cce-b70c-11ea-a716-025000000001'],
            nonIsolatedIngress: false,
            nonIsolatedEgress: false,
            outEdges: {},
            isActive: false,
            type: 'DEPLOYMENT',
            id: 'e34b1fe2-b70c-11ea-a716-025000000001',
            name: 'scanner-db',
            cluster: 'remote',
            parent: 'stackrox',
            edges: [
                {
                    data: {
                        destNodeId: 'e349f5d4-b70c-11ea-a716-025000000001',
                        destNodeNamespace: 'stackrox',
                        destNodeName: 'scanner',
                        source: 'e349f5d4-b70c-11ea-a716-025000000001',
                        target: 'e34b1fe2-b70c-11ea-a716-025000000001',
                        sourceName: 'scanner',
                        targetName: 'scanner-db',
                        sourceNS: 'stackrox',
                        targetNS: 'stackrox',
                        isActive: true,
                        isBetweenNonIsolated: false,
                    },
                    classes: 'edge active  ',
                },
            ],
            deploymentId: 'e34b1fe2-b70c-11ea-a716-025000000001',
        },
        classes: 'deployment',
    },
    {
        data: {
            internetAccess: false,
            policyIds: ['e34dc4b5-b70c-11ea-a716-025000000001'],
            nonIsolatedIngress: false,
            nonIsolatedEgress: false,
            outEdges: {
                5: {
                    properties: [
                        {
                            port: 5432,
                            protocol: 'L4_PROTOCOL_TCP',
                            lastActiveTimestamp: '2020-07-10T06:42:30.749085900Z',
                        },
                    ],
                },
                7: {
                    properties: [
                        {
                            port: 53,
                            protocol: 'L4_PROTOCOL_UDP',
                            lastActiveTimestamp: '2020-07-10T06:40:30.928923Z',
                        },
                    ],
                },
                8: {
                    properties: [
                        {
                            port: 8443,
                            protocol: 'L4_PROTOCOL_TCP',
                            lastActiveTimestamp: '2020-07-10T06:42:30.749070500Z',
                        },
                    ],
                },
                13: {
                    properties: [
                        {
                            port: 443,
                            protocol: 'L4_PROTOCOL_TCP',
                            lastActiveTimestamp: '2020-07-10T06:42:30.749101400Z',
                        },
                        {
                            port: 9,
                            protocol: 'L4_PROTOCOL_UDP',
                            lastActiveTimestamp: '2020-07-10T06:40:30.929412Z',
                        },
                    ],
                },
            },
            isActive: false,
            type: 'DEPLOYMENT',
            id: 'e349f5d4-b70c-11ea-a716-025000000001',
            name: 'scanner',
            cluster: 'remote',
            parent: 'stackrox',
            edges: [
                {
                    data: {
                        destNodeId: 'e34b1fe2-b70c-11ea-a716-025000000001',
                        destNodeNamespace: 'stackrox',
                        destNodeName: 'scanner-db',
                        source: 'e349f5d4-b70c-11ea-a716-025000000001',
                        target: 'e34b1fe2-b70c-11ea-a716-025000000001',
                        sourceName: 'scanner',
                        targetName: 'scanner-db',
                        sourceNS: 'stackrox',
                        targetNS: 'stackrox',
                        isActive: true,
                        isBetweenNonIsolated: false,
                    },
                    classes: 'edge active  ',
                },
                {
                    data: {
                        source: 'e349f5d4-b70c-11ea-a716-025000000001',
                        target: 'stackrox_left',
                    },
                    classes: 'edge inner active  ',
                },
                {
                    data: {
                        source: '6fc26030-b70a-11ea-a716-025000000001',
                        target: 'kube-system_right',
                        destNodeId: '6fc26030-b70a-11ea-a716-025000000001',
                        destNodeName: 'coredns',
                        destNodeNamespace: 'kube-system',
                        isActive: true,
                    },
                    classes: 'edge inner active  ',
                },
                {
                    data: {
                        destNodeId: 'e2f0275b-b70c-11ea-a716-025000000001',
                        destNodeNamespace: 'stackrox',
                        destNodeName: 'central',
                        source: 'e349f5d4-b70c-11ea-a716-025000000001',
                        target: 'e2f0275b-b70c-11ea-a716-025000000001',
                        sourceName: 'scanner',
                        targetName: 'central',
                        sourceNS: 'stackrox',
                        targetNS: 'stackrox',
                        isActive: true,
                        isBetweenNonIsolated: false,
                    },
                    classes: 'edge active  ',
                },
            ],
            deploymentId: 'e349f5d4-b70c-11ea-a716-025000000001',
        },
        classes: 'deployment',
    },
    {
        data: {
            internetAccess: false,
            policyIds: [],
            nonIsolatedIngress: false,
            nonIsolatedEgress: false,
            outEdges: {
                13: {
                    properties: [
                        {
                            port: 443,
                            protocol: 'L4_PROTOCOL_TCP',
                            lastActiveTimestamp: '2020-07-10T06:42:30.748945900Z',
                        },
                        {
                            port: 8080,
                            protocol: 'L4_PROTOCOL_TCP',
                            lastActiveTimestamp: '2020-07-10T06:41:42.741887Z',
                        },
                        {
                            port: 53,
                            protocol: 'L4_PROTOCOL_UDP',
                            lastActiveTimestamp: '2020-07-10T06:40:46.822163Z',
                        },
                    ],
                },
            },
            isActive: false,
            type: 'DEPLOYMENT',
            id: '6fc26030-b70a-11ea-a716-025000000001',
            name: 'coredns',
            cluster: 'remote',
            parent: 'kube-system',
            edges: [
                {
                    data: {
                        source: 'e349f5d4-b70c-11ea-a716-025000000001',
                        target: 'stackrox_left',
                    },
                    classes: 'edge inner active  ',
                },
                {
                    data: {
                        source: '6fc26030-b70a-11ea-a716-025000000001',
                        target: 'kube-system_right',
                        destNodeId: 'e349f5d4-b70c-11ea-a716-025000000001',
                        destNodeName: 'scanner',
                        destNodeNamespace: 'stackrox',
                        isActive: true,
                    },
                    classes: 'edge inner active  ',
                },
                {
                    data: {
                        source: 'e2f0275b-b70c-11ea-a716-025000000001',
                        target: 'stackrox_left',
                    },
                    classes: 'edge inner active  ',
                },
                {
                    data: {
                        source: '6fc26030-b70a-11ea-a716-025000000001',
                        target: 'kube-system_right',
                        destNodeId: 'e2f0275b-b70c-11ea-a716-025000000001',
                        destNodeName: 'central',
                        destNodeNamespace: 'stackrox',
                        isActive: true,
                    },
                    classes: 'edge inner active  ',
                },
            ],
            deploymentId: '6fc26030-b70a-11ea-a716-025000000001',
        },
        classes: 'deployment',
    },
    {
        data: {
            internetAccess: false,
            policyIds: ['e2f3b506-b70c-11ea-a716-025000000001'],
            nonIsolatedIngress: false,
            nonIsolatedEgress: false,
            outEdges: {
                7: {
                    properties: [
                        {
                            port: 53,
                            protocol: 'L4_PROTOCOL_UDP',
                            lastActiveTimestamp: '2020-07-10T06:40:13.139662Z',
                        },
                    ],
                },
                13: {
                    properties: [
                        {
                            port: 443,
                            protocol: 'L4_PROTOCOL_TCP',
                            lastActiveTimestamp: '2020-07-10T06:42:30.749053800Z',
                        },
                        {
                            port: 9,
                            protocol: 'L4_PROTOCOL_UDP',
                            lastActiveTimestamp: '2020-07-10T06:40:13.139946Z',
                        },
                    ],
                },
            },
            isActive: false,
            type: 'DEPLOYMENT',
            id: 'e2f0275b-b70c-11ea-a716-025000000001',
            name: 'central',
            cluster: 'remote',
            parent: 'stackrox',
            edges: [
                {
                    data: {
                        destNodeId: '892424ba-b70d-11ea-a716-025000000001',
                        destNodeNamespace: 'stackrox',
                        destNodeName: 'sensor',
                        source: '892424ba-b70d-11ea-a716-025000000001',
                        target: 'e2f0275b-b70c-11ea-a716-025000000001',
                        sourceName: 'sensor',
                        targetName: 'central',
                        sourceNS: 'stackrox',
                        targetNS: 'stackrox',
                        isActive: true,
                        isBetweenNonIsolated: false,
                    },
                    classes: 'edge active  ',
                },
                {
                    data: {
                        destNodeId: 'e349f5d4-b70c-11ea-a716-025000000001',
                        destNodeNamespace: 'stackrox',
                        destNodeName: 'scanner',
                        source: 'e349f5d4-b70c-11ea-a716-025000000001',
                        target: 'e2f0275b-b70c-11ea-a716-025000000001',
                        sourceName: 'scanner',
                        targetName: 'central',
                        sourceNS: 'stackrox',
                        targetNS: 'stackrox',
                        isActive: true,
                        isBetweenNonIsolated: false,
                    },
                    classes: 'edge active  ',
                },
                {
                    data: {
                        source: 'e2f0275b-b70c-11ea-a716-025000000001',
                        target: 'stackrox_left',
                    },
                    classes: 'edge inner active  ',
                },
                {
                    data: {
                        source: '6fc26030-b70a-11ea-a716-025000000001',
                        target: 'kube-system_right',
                        destNodeId: '6fc26030-b70a-11ea-a716-025000000001',
                        destNodeName: 'coredns',
                        destNodeNamespace: 'kube-system',
                        isActive: true,
                    },
                    classes: 'edge inner active  ',
                },
            ],
            deploymentId: 'e2f0275b-b70c-11ea-a716-025000000001',
        },
        classes: 'deployment',
    },
    {
        data: {
            internetAccess: false,
            policyIds: [],
            nonIsolatedIngress: false,
            nonIsolatedEgress: false,
            outEdges: {},
            isActive: false,
            type: 'DEPLOYMENT',
            id: '0b7e5849-3663-56e9-9321-27cbf6147418',
            name: 'static-kube-controller-manager-pods',
            cluster: 'remote',
            parent: 'kube-system',
            edges: [],
            deploymentId: '0b7e5849-3663-56e9-9321-27cbf6147418',
        },
        classes: 'deployment',
    },
    {
        data: {
            internetAccess: false,
            policyIds: [],
            nonIsolatedIngress: false,
            nonIsolatedEgress: false,
            outEdges: {},
            isActive: false,
            type: 'DEPLOYMENT',
            id: '9c55e10b-37c1-55ed-8480-60decad25167',
            name: 'static-kube-apiserver-pods',
            cluster: 'remote',
            parent: 'kube-system',
            edges: [],
            deploymentId: '9c55e10b-37c1-55ed-8480-60decad25167',
        },
        classes: 'deployment',
    },
    {
        data: {
            internetAccess: false,
            policyIds: [],
            nonIsolatedIngress: false,
            nonIsolatedEgress: false,
            outEdges: {},
            isActive: false,
            type: 'DEPLOYMENT',
            id: '76c65318-a0aa-5991-8a32-dcf99d3c8d3b',
            name: 'static-kube-scheduler-pods',
            cluster: 'remote',
            parent: 'kube-system',
            edges: [],
            deploymentId: '76c65318-a0aa-5991-8a32-dcf99d3c8d3b',
        },
        classes: 'deployment',
    },
    {
        data: {
            internetAccess: false,
            policyIds: [],
            nonIsolatedIngress: false,
            nonIsolatedEgress: false,
            outEdges: {},
            isActive: false,
            type: 'DEPLOYMENT',
            id: '2e6290d8-4899-5216-9fab-e2e8767684c7',
            name: 'static-etcd-pods',
            cluster: 'remote',
            parent: 'kube-system',
            edges: [],
            deploymentId: '2e6290d8-4899-5216-9fab-e2e8767684c7',
        },
        classes: 'deployment',
    },
];

export const namespaceList = [
    {
        data: { id: 'kube-system', name: ' kube-system', active: false, type: 'NAMESPACE' },
        classes: '',
    },
    { data: { id: 'stackrox', name: ' stackrox', active: false, type: 'NAMESPACE' }, classes: '' },
    { data: { id: 'docker', name: ' docker', active: false, type: 'NAMESPACE' }, classes: '' },
];

export const namespaceEdgeNodes = [
    {
        data: {
            id: 'kube-system_top',
            parent: 'kube-system',
            side: 'top',
            category: 'NAMESPACE',
        },
        classes: 'nsEdge',
    },
    {
        data: {
            id: 'kube-system_left',
            parent: 'kube-system',
            side: 'left',
            category: 'NAMESPACE',
        },
        classes: 'nsEdge',
    },
    {
        data: {
            id: 'kube-system_right',
            parent: 'kube-system',
            side: 'right',
            category: 'NAMESPACE',
        },
        classes: 'nsEdge',
    },
    {
        data: {
            id: 'kube-system_bottom',
            parent: 'kube-system',
            side: 'bottom',
            category: 'NAMESPACE',
        },
        classes: 'nsEdge',
    },
    {
        data: { id: 'stackrox_top', parent: 'stackrox', side: 'top', category: 'NAMESPACE' },
        classes: 'nsEdge',
    },
    {
        data: {
            id: 'stackrox_left',
            parent: 'stackrox',
            side: 'left',
            category: 'NAMESPACE',
        },
        classes: 'nsEdge',
    },
    {
        data: {
            id: 'stackrox_right',
            parent: 'stackrox',
            side: 'right',
            category: 'NAMESPACE',
        },
        classes: 'nsEdge',
    },
    {
        data: {
            id: 'stackrox_bottom',
            parent: 'stackrox',
            side: 'bottom',
            category: 'NAMESPACE',
        },
        classes: 'nsEdge',
    },
    {
        data: { id: 'docker_top', parent: 'docker', side: 'top', category: 'NAMESPACE' },
        classes: 'nsEdge',
    },
    {
        data: { id: 'docker_left', parent: 'docker', side: 'left', category: 'NAMESPACE' },
        classes: 'nsEdge',
    },
    {
        data: { id: 'docker_right', parent: 'docker', side: 'right', category: 'NAMESPACE' },
        classes: 'nsEdge',
    },
    {
        data: {
            id: 'docker_bottom',
            parent: 'docker',
            side: 'bottom',
            category: 'NAMESPACE',
        },
        classes: 'nsEdge',
    },
];

export const deploymentEdges: Edge[] = [
    {
        data: {
            destNodeId: '1',
            destNodeName: 'node-1',
            destNodeNamespace: 'namespace-a',
            traffic: 'ingress',
            type: 'deployment',
            isActive: true,
            isAllowed: true,
            portsAndProtocols: [
                {
                    port: 111,
                    protocol: 'L4_PROTOCOL_TCP',
                    traffic: 'ingress',
                },
            ],
        },
    },
    {
        data: {
            destNodeId: '2',
            destNodeName: 'node-2',
            destNodeNamespace: 'namespace-a',
            traffic: 'egress',
            type: 'deployment',
            isActive: false,
            isAllowed: true,
            portsAndProtocols: [
                {
                    port: 222,
                    protocol: 'L4_PROTOCOL_UDP',
                    traffic: 'egress',
                },
            ],
        },
    },
    {
        data: {
            destNodeId: '3',
            destNodeName: 'node-3',
            destNodeNamespace: 'namespace-a',
            traffic: 'egress',
            type: 'deployment',
            isActive: false,
            isAllowed: true,
            portsAndProtocols: [
                {
                    port: 333,
                    protocol: 'L4_PROTOCOL_TCP',
                    traffic: 'egress',
                },
            ],
        },
    },
    {
        data: {
            destNodeId: '4',
            destNodeName: 'node-4',
            destNodeNamespace: 'namespace-a',
            traffic: 'bidirectional',
            type: 'deployment',
            isActive: true,
            isAllowed: true,
            portsAndProtocols: [
                {
                    port: 444,
                    protocol: 'L4_PROTOCOL_UDP',
                    traffic: 'ingress',
                },
                {
                    port: 555,
                    protocol: 'L4_PROTOCOL_TCP',
                    traffic: 'egress',
                },
            ],
        },
    },
];

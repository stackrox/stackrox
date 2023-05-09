import { AdvancedFlowsFilterType } from '../common/AdvancedFlowsFilter/types';
import { Flow, Peer } from '../types/flow.type';
import { CustomEdgeModel, CustomNodeModel } from '../types/topology.type';
import {
    filterNetworkFlows,
    getAllUniquePorts,
    getNetworkFlows,
    getNumExtraneousEgressFlows,
    getNumExtraneousIngressFlows,
    getNumFlows,
    getUniqueIdFromFlow,
    getUniqueIdFromPeer,
    transformFlowsToPeers,
} from './flowUtils';

describe('flowUtils', () => {
    describe('getAllUniquePorts', () => {
        it('should get all unique ports from flows', () => {
            const flows: Flow[] = [
                {
                    id: 'deployment1-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                    type: 'DEPLOYMENT',
                    entity: 'deployment1',
                    entityId: '1',
                    namespace: 'namespace1',
                    direction: 'Ingress',
                    port: '8443',
                    protocol: 'L4_PROTOCOL_TCP',
                    isAnomalous: true,
                    children: [],
                },
                {
                    id: 'deployment2-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                    type: 'DEPLOYMENT',
                    entity: 'deployment2',
                    entityId: '2',
                    namespace: 'namespace1',
                    direction: 'Ingress',
                    port: '8443',
                    protocol: 'L4_PROTOCOL_TCP',
                    isAnomalous: true,
                    children: [],
                },
                {
                    id: 'deployment3-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                    type: 'DEPLOYMENT',
                    entity: 'deployment3',
                    entityId: '3',
                    namespace: 'namespace1',
                    direction: 'Ingress',
                    port: '8080',
                    protocol: 'L4_PROTOCOL_TCP',
                    isAnomalous: true,
                    children: [],
                },
            ];

            const uniquePorts = getAllUniquePorts(flows);

            expect(uniquePorts).toEqual(['8443', '8080']);
        });
    });

    describe('getNumFlows', () => {
        it('should get number of flows for non-aggregated flows (no children)', () => {
            const flows: Flow[] = [
                {
                    id: 'deployment1-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                    type: 'DEPLOYMENT',
                    entity: 'deployment1',
                    entityId: '1',
                    namespace: 'namespace1',
                    direction: 'Ingress',
                    port: '8443',
                    protocol: 'L4_PROTOCOL_TCP',
                    isAnomalous: true,
                    children: [],
                },
                {
                    id: 'deployment2-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                    type: 'DEPLOYMENT',
                    entity: 'deployment2',
                    entityId: '2',
                    namespace: 'namespace1',
                    direction: 'Ingress',
                    port: '8443',
                    protocol: 'L4_PROTOCOL_TCP',
                    isAnomalous: true,
                    children: [],
                },
                {
                    id: 'deployment3-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                    type: 'DEPLOYMENT',
                    entity: 'deployment3',
                    entityId: '3',
                    namespace: 'namespace1',
                    direction: 'Ingress',
                    port: '8080',
                    protocol: 'L4_PROTOCOL_TCP',
                    isAnomalous: true,
                    children: [],
                },
            ];

            const numFlows = getNumFlows(flows);

            expect(numFlows).toEqual(3);
        });

        it('should get number of flows for aggregated flows (with children)', () => {
            const flows: Flow[] = [
                {
                    id: 'deployment1-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                    type: 'DEPLOYMENT',
                    entity: 'deployment1',
                    entityId: '1',
                    namespace: 'namespace1',
                    direction: 'Both ways',
                    port: '8443',
                    protocol: 'L4_PROTOCOL_TCP',
                    isAnomalous: true,
                    children: [
                        {
                            id: 'deployment1-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                            type: 'DEPLOYMENT',
                            entity: 'deployment1',
                            entityId: '1',
                            namespace: 'namespace1',
                            direction: 'Ingress',
                            port: '8443',
                            protocol: 'L4_PROTOCOL_TCP',
                            isAnomalous: true,
                        },
                        {
                            id: 'deployment1-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                            type: 'DEPLOYMENT',
                            entity: 'deployment1',
                            entityId: '1',
                            namespace: 'namespace1',
                            direction: 'Egress',
                            port: '8443',
                            protocol: 'L4_PROTOCOL_TCP',
                            isAnomalous: true,
                        },
                    ],
                },
                {
                    id: 'deployment2-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                    type: 'DEPLOYMENT',
                    entity: 'deployment2',
                    entityId: '2',
                    namespace: 'namespace1',
                    direction: 'Ingress',
                    port: '8443',
                    protocol: 'L4_PROTOCOL_TCP',
                    isAnomalous: true,
                    children: [],
                },
                {
                    id: 'deployment3-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                    type: 'DEPLOYMENT',
                    entity: 'deployment3',
                    entityId: '3',
                    namespace: 'namespace1',
                    direction: 'Ingress',
                    port: '8080',
                    protocol: 'L4_PROTOCOL_TCP',
                    isAnomalous: true,
                    children: [],
                },
            ];

            const numFlows = getNumFlows(flows);

            expect(numFlows).toEqual(4);
        });
    });

    describe('getUniqueIdFromFlows', () => {
        it('should get a unique id from a flow', () => {
            const flow: Flow = {
                id: 'scanner-stackrox-Ingress-8443-L4_PROTOCOL_TCP',
                type: 'DEPLOYMENT',
                entity: 'stackrox',
                entityId: '305cf89a-ea61-4804-bf09-e0c08f0a141f',
                namespace: 'stackrox',
                direction: 'Ingress',
                port: '8443',
                protocol: 'L4_PROTOCOL_TCP',
                isAnomalous: true,
                children: [],
            };

            const id = getUniqueIdFromFlow(flow);

            expect(id).toEqual('305cf89a-ea61-4804-bf09-e0c08f0a141f-Ingress-8443-L4_PROTOCOL_TCP');
        });
    });

    describe('getUniqueIdFromPeer', () => {
        it('should get a unique id from a peer', () => {
            const peer: Peer = {
                entity: {
                    id: '305cf89a-ea61-4804-bf09-e0c08f0a141f',
                    type: 'DEPLOYMENT',
                    name: 'test',
                    namespace: 'test',
                },
                ingress: true,
                port: 8443,
                protocol: 'L4_PROTOCOL_TCP',
            };

            const id = getUniqueIdFromPeer(peer);

            expect(id).toEqual('305cf89a-ea61-4804-bf09-e0c08f0a141f-Ingress-8443-L4_PROTOCOL_TCP');
        });
    });

    describe('getNetworkFlows', () => {
        it('should transform edges to network flows', () => {
            const nodes: CustomNodeModel[] = [
                {
                    id: 'deployment-1',
                    type: 'node',
                    label: 'deployment-1',
                    data: {
                        type: 'DEPLOYMENT',
                        id: 'deployment-1',
                        deployment: {
                            cluster: 'cluster-1',
                            listenPorts: [{ port: 8443, l4protocol: 'L4_PROTOCOL_TCP' }],
                            name: 'deployment-1',
                            namespace: 'namespace-1',
                        },
                        policyIds: ['policy-1', 'policy-2'],
                        networkPolicyState: 'both',
                        showPolicyState: true,
                        isExternallyConnected: false,
                        showExternalState: false,
                        isFadedOut: false,
                        labelIconClass: '',
                    },
                },
                {
                    id: 'deployment-2',
                    type: 'node',
                    label: 'deployment-2',
                    data: {
                        type: 'DEPLOYMENT',
                        id: 'deployment-2',
                        deployment: {
                            cluster: 'cluster-1',
                            listenPorts: [{ port: 8443, l4protocol: 'L4_PROTOCOL_TCP' }],
                            name: 'deployment-2',
                            namespace: 'namespace-2',
                        },
                        policyIds: ['policy-1', 'policy-2'],
                        networkPolicyState: 'both',
                        showPolicyState: true,
                        isExternallyConnected: false,
                        showExternalState: false,
                        isFadedOut: false,
                        labelIconClass: '',
                    },
                },
            ];
            const edges: CustomEdgeModel[] = [
                {
                    id: 'edge-1',
                    type: 'edge',
                    source: 'deployment-1',
                    target: 'deployment-2',
                    data: {
                        portProtocolLabel: '8443 TCP',
                        sourceToTargetProperties: [
                            {
                                port: 8443,
                                protocol: 'L4_PROTOCOL_TCP',
                                lastActiveTimestamp: null,
                            },
                        ],
                        isBidirectional: false,
                    },
                },
                {
                    id: 'edge-2',
                    type: 'edge',
                    source: 'deployment-1',
                    target: 'deployment-2',
                    data: {
                        portProtocolLabel: '8080 TCP',
                        sourceToTargetProperties: [
                            {
                                port: 8080,
                                protocol: 'L4_PROTOCOL_TCP',
                                lastActiveTimestamp: null,
                            },
                        ],
                        isBidirectional: false,
                    },
                },
            ];
            const id = 'deployment-1';

            const networkFlows = getNetworkFlows(nodes, edges, id);
            const expectedNetworkFlows: Flow[] = [
                {
                    id: 'deployment-2-namespace-2-Egress-8443-L4_PROTOCOL_TCP',
                    type: 'DEPLOYMENT',
                    entity: 'deployment-2',
                    entityId: 'deployment-2',
                    namespace: 'namespace-2',
                    port: '8443',
                    protocol: 'L4_PROTOCOL_TCP',
                    direction: 'Egress',
                    isAnomalous: true,
                    children: [],
                },
                {
                    id: 'deployment-2-namespace-2-Egress-8080-L4_PROTOCOL_TCP',
                    type: 'DEPLOYMENT',
                    entity: 'deployment-2',
                    entityId: 'deployment-2',
                    namespace: 'namespace-2',
                    port: '8080',
                    protocol: 'L4_PROTOCOL_TCP',
                    direction: 'Egress',
                    isAnomalous: true,
                    children: [],
                },
            ];

            expect(networkFlows).toEqual(expectedNetworkFlows);
        });
    });

    describe('filterNetworkFlows', () => {
        it('should filter network flows using the entity name and advanced filters', () => {
            const flows: Flow[] = [
                {
                    id: 'deployment1-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                    type: 'DEPLOYMENT',
                    entity: 'deployment1',
                    entityId: '1',
                    namespace: 'namespace1',
                    direction: 'Both ways',
                    port: '8443',
                    protocol: 'L4_PROTOCOL_TCP',
                    isAnomalous: true,
                    children: [
                        {
                            id: 'deployment1-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                            type: 'DEPLOYMENT',
                            entity: 'deployment1',
                            entityId: '1',
                            namespace: 'namespace1',
                            direction: 'Ingress',
                            port: '8443',
                            protocol: 'L4_PROTOCOL_TCP',
                            isAnomalous: true,
                        },
                        {
                            id: 'deployment1-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                            type: 'DEPLOYMENT',
                            entity: 'deployment1',
                            entityId: '1',
                            namespace: 'namespace1',
                            direction: 'Egress',
                            port: '8443',
                            protocol: 'L4_PROTOCOL_TCP',
                            isAnomalous: true,
                        },
                    ],
                },
                {
                    id: 'deployment2-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                    type: 'DEPLOYMENT',
                    entity: 'deployment2',
                    entityId: '2',
                    namespace: 'namespace1',
                    direction: 'Ingress',
                    port: '8443',
                    protocol: 'L4_PROTOCOL_TCP',
                    isAnomalous: false,
                    children: [],
                },
                {
                    id: 'deployment3-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                    type: 'DEPLOYMENT',
                    entity: 'deployment3',
                    entityId: '3',
                    namespace: 'namespace1',
                    direction: 'Egress',
                    port: '8080',
                    protocol: 'L4_PROTOCOL_TCP',
                    isAnomalous: false,
                    children: [],
                },
            ];
            const entityNameFilter = 'deployment2';
            const advancedFilters: AdvancedFlowsFilterType = {
                directionality: ['ingress'],
                protocols: ['L4_PROTOCOL_TCP'],
                ports: ['8443'],
            };

            const filteredFlows = filterNetworkFlows(flows, entityNameFilter, advancedFilters);
            const expectedFilteredFlows: Flow[] = [
                {
                    id: 'deployment2-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                    type: 'DEPLOYMENT',
                    entity: 'deployment2',
                    entityId: '2',
                    namespace: 'namespace1',
                    direction: 'Ingress',
                    port: '8443',
                    protocol: 'L4_PROTOCOL_TCP',
                    isAnomalous: false,
                    children: [],
                },
            ];

            expect(filteredFlows).toEqual(expectedFilteredFlows);
        });
    });

    describe('transformFlowsToPeers', () => {
        it('should transform flows to peers', () => {
            const flows: Flow[] = [
                {
                    id: 'deployment1-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                    type: 'DEPLOYMENT',
                    entity: 'deployment1',
                    entityId: '1',
                    namespace: 'namespace1',
                    direction: 'Both ways',
                    port: '8443',
                    protocol: 'L4_PROTOCOL_TCP',
                    isAnomalous: true,
                    children: [
                        {
                            id: 'deployment1-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                            type: 'DEPLOYMENT',
                            entity: 'deployment1',
                            entityId: '1',
                            namespace: 'namespace1',
                            direction: 'Ingress',
                            port: '8443',
                            protocol: 'L4_PROTOCOL_TCP',
                            isAnomalous: true,
                        },
                        {
                            id: 'deployment1-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                            type: 'DEPLOYMENT',
                            entity: 'deployment1',
                            entityId: '1',
                            namespace: 'namespace1',
                            direction: 'Egress',
                            port: '8443',
                            protocol: 'L4_PROTOCOL_TCP',
                            isAnomalous: true,
                        },
                    ],
                },
                {
                    id: 'deployment2-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                    type: 'DEPLOYMENT',
                    entity: 'deployment2',
                    entityId: '2',
                    namespace: 'namespace1',
                    direction: 'Ingress',
                    port: '8443',
                    protocol: 'L4_PROTOCOL_TCP',
                    isAnomalous: false,
                    children: [],
                },
                {
                    id: 'deployment3-namespace1-Ingress-8443-L4_PROTOCOL_TCP',
                    type: 'DEPLOYMENT',
                    entity: 'deployment3',
                    entityId: '3',
                    namespace: 'namespace1',
                    direction: 'Egress',
                    port: '8080',
                    protocol: 'L4_PROTOCOL_TCP',
                    isAnomalous: false,
                    children: [],
                },
            ];

            const peers = transformFlowsToPeers(flows);
            const expectedPeers: Peer[] = [
                {
                    entity: {
                        id: '1',
                        name: 'deployment1',
                        namespace: 'namespace1',
                        type: 'DEPLOYMENT',
                    },
                    ingress: false,
                    port: 8443,
                    protocol: 'L4_PROTOCOL_TCP',
                },
                {
                    entity: {
                        id: '2',
                        name: 'deployment2',
                        namespace: 'namespace1',
                        type: 'DEPLOYMENT',
                    },
                    ingress: true,
                    port: 8443,
                    protocol: 'L4_PROTOCOL_TCP',
                },
                {
                    entity: {
                        id: '3',
                        name: 'deployment3',
                        namespace: 'namespace1',
                        type: 'DEPLOYMENT',
                    },
                    ingress: false,
                    port: 8080,
                    protocol: 'L4_PROTOCOL_TCP',
                },
            ];

            expect(peers).toEqual(expectedPeers);
        });
    });

    describe('getNumExtraneousEgressFlows', () => {
        it('should get the number of extraneous egress flows', () => {
            const nodes: CustomNodeModel[] = [
                {
                    id: 'deployment-1',
                    type: 'node',
                    label: 'deployment-1',
                    data: {
                        type: 'DEPLOYMENT',
                        id: 'deployment-1',
                        deployment: {
                            cluster: 'cluster-1',
                            listenPorts: [{ port: 8443, l4protocol: 'L4_PROTOCOL_TCP' }],
                            name: 'deployment-1',
                            namespace: 'namespace-1',
                        },
                        policyIds: ['policy-1', 'policy-2'],
                        networkPolicyState: 'both',
                        showPolicyState: true,
                        isExternallyConnected: false,
                        showExternalState: false,
                        isFadedOut: false,
                        labelIconClass: '',
                    },
                },
                {
                    id: 'extraneous-egress-flows',
                    type: 'fakeGroup',
                    label: 'Egress flows',
                    data: {
                        collapsible: false,
                        numFlows: 50,
                        showContextMenu: false,
                        type: 'EXTRANEOUS',
                    },
                },
            ];

            const numExtraneousEgressFlows = getNumExtraneousEgressFlows(nodes);

            expect(numExtraneousEgressFlows).toEqual(50);
        });

        it('should get the number of extraneous egress flows when none exist', () => {
            const nodes: CustomNodeModel[] = [
                {
                    id: 'deployment-1',
                    type: 'node',
                    label: 'deployment-1',
                    data: {
                        type: 'DEPLOYMENT',
                        id: 'deployment-1',
                        deployment: {
                            cluster: 'cluster-1',
                            listenPorts: [{ port: 8443, l4protocol: 'L4_PROTOCOL_TCP' }],
                            name: 'deployment-1',
                            namespace: 'namespace-1',
                        },
                        policyIds: ['policy-1', 'policy-2'],
                        networkPolicyState: 'both',
                        showPolicyState: true,
                        isExternallyConnected: false,
                        showExternalState: false,
                        isFadedOut: false,
                        labelIconClass: '',
                    },
                },
            ];

            const numExtraneousEgressFlows = getNumExtraneousEgressFlows(nodes);

            expect(numExtraneousEgressFlows).toEqual(0);
        });
    });

    describe('getNumExtraneousIngressFlows', () => {
        it('should get the number of extraneous ingress flows', () => {
            const nodes: CustomNodeModel[] = [
                {
                    id: 'deployment-1',
                    type: 'node',
                    label: 'deployment-1',
                    data: {
                        type: 'DEPLOYMENT',
                        id: 'deployment-1',
                        deployment: {
                            cluster: 'cluster-1',
                            listenPorts: [{ port: 8443, l4protocol: 'L4_PROTOCOL_TCP' }],
                            name: 'deployment-1',
                            namespace: 'namespace-1',
                        },
                        policyIds: ['policy-1', 'policy-2'],
                        networkPolicyState: 'both',
                        showPolicyState: true,
                        isExternallyConnected: false,
                        showExternalState: false,
                        isFadedOut: false,
                        labelIconClass: '',
                    },
                },
                {
                    id: 'extraneous-ingress-flows',
                    type: 'fakeGroup',
                    label: 'Ingress flows',
                    data: {
                        collapsible: false,
                        numFlows: 100,
                        showContextMenu: false,
                        type: 'EXTRANEOUS',
                    },
                },
            ];

            const numExtraneousEgressFlows = getNumExtraneousIngressFlows(nodes);

            expect(numExtraneousEgressFlows).toEqual(100);
        });

        it('should get the number of extraneous ingress flows when none exist', () => {
            const nodes: CustomNodeModel[] = [
                {
                    id: 'deployment-1',
                    type: 'node',
                    label: 'deployment-1',
                    data: {
                        type: 'DEPLOYMENT',
                        id: 'deployment-1',
                        deployment: {
                            cluster: 'cluster-1',
                            listenPorts: [{ port: 8443, l4protocol: 'L4_PROTOCOL_TCP' }],
                            name: 'deployment-1',
                            namespace: 'namespace-1',
                        },
                        policyIds: ['policy-1', 'policy-2'],
                        networkPolicyState: 'both',
                        showPolicyState: true,
                        isExternallyConnected: false,
                        showExternalState: false,
                        isFadedOut: false,
                        labelIconClass: '',
                    },
                },
            ];

            const numExtraneousEgressFlows = getNumExtraneousIngressFlows(nodes);

            expect(numExtraneousEgressFlows).toEqual(0);
        });
    });
});

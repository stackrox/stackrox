import { Flow, Peer } from '../types/flow.type';
import {
    getAllUniquePorts,
    getNumFlows,
    getUniqueIdFromFlow,
    getUniqueIdFromPeer,
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
});

import { CustomEdgeModel } from '../types/topology.type';
import { getNumFlowsFromEdge } from './networkGraphUtils';

describe('networkGraphUtils', () => {
    describe('getNumFlowsFromEdge', () => {
        it('should get number of flows for a non-bidirectional edge with one port/protocol', () => {
            const edge: CustomEdgeModel = {
                id: 'edge-1',
                type: 'edge',
                visible: true,
                source: 'node-1',
                target: 'node-2',
                data: {
                    isBidirectional: false,
                    portProtocolLabel: '8080 TCP',
                    sourceToTargetProperties: [
                        {
                            port: 8080,
                            protocol: 'L4_PROTOCOL_TCP',
                            lastActiveTimestamp: '2023-02-08T18:24:50.470158762Z',
                        },
                    ],
                    tag: '8080 TCP',
                },
            };

            const numFlows = getNumFlowsFromEdge(edge);

            expect(numFlows).toEqual(1);
        });

        it('should get number of flows for a non-bidirectional edge with multiple ports/protocols', () => {
            const edge: CustomEdgeModel = {
                id: 'edge-1',
                type: 'edge',
                visible: true,
                source: 'node-1',
                target: 'node-2',
                data: {
                    isBidirectional: false,
                    portProtocolLabel: '2',
                    sourceToTargetProperties: [
                        {
                            port: 8080,
                            protocol: 'L4_PROTOCOL_TCP',
                            lastActiveTimestamp: '2023-02-08T18:24:50.470158762Z',
                        },
                        {
                            port: 80,
                            protocol: 'L4_PROTOCOL_TCP',
                            lastActiveTimestamp: '2023-02-08T18:24:50.470158762Z',
                        },
                    ],
                    tag: '2',
                },
            };

            const numFlows = getNumFlowsFromEdge(edge);

            expect(numFlows).toEqual(2);
        });

        it('should get number of flows for a bidirectional edge', () => {
            const edge: CustomEdgeModel = {
                id: 'edge-1',
                type: 'edge',
                visible: true,
                source: 'node-1',
                target: 'node-2',
                data: {
                    isBidirectional: true,
                    portProtocolLabel: '4',
                    sourceToTargetProperties: [
                        {
                            port: 8080,
                            protocol: 'L4_PROTOCOL_TCP',
                            lastActiveTimestamp: '2023-02-08T18:24:50.470158762Z',
                        },
                        {
                            port: 80,
                            protocol: 'L4_PROTOCOL_TCP',
                            lastActiveTimestamp: '2023-02-08T18:24:50.470158762Z',
                        },
                    ],
                    targetToSourceProperties: [
                        {
                            port: 8080,
                            protocol: 'L4_PROTOCOL_TCP',
                            lastActiveTimestamp: '2023-02-08T18:24:50.470158762Z',
                        },
                        {
                            port: 80,
                            protocol: 'L4_PROTOCOL_TCP',
                            lastActiveTimestamp: '2023-02-08T18:24:50.470158762Z',
                        },
                    ],
                    tag: '4',
                },
            };

            const numFlows = getNumFlowsFromEdge(edge);

            expect(numFlows).toEqual(4);
        });
    });
});

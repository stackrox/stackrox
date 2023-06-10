import { EdgeTerminalType } from '@patternfly/react-topology';
import { CustomEdgeModel } from '../types/topology.type';
import { removeDNSEdges } from './edgeUtils';

describe('edgeUtils', () => {
    describe('removeDNSEdges', () => {
        it('should not remove a one-directional non-dns edge', () => {
            const edges: CustomEdgeModel[] = [
                {
                    id: 'edge-1',
                    type: 'edge',
                    source: 'node-1',
                    target: 'node-2',
                    data: {
                        isBidirectional: false,
                        portProtocolLabel: '80 TCP',
                        tag: '80 TCP',
                        sourceToTargetProperties: [
                            {
                                port: 80,
                                protocol: 'L4_PROTOCOL_TCP',
                                lastActiveTimestamp: '',
                            },
                        ],
                        endTerminalType: 'directional' as EdgeTerminalType,
                    },
                },
            ];
            const edgesWithoutDNSFlows = removeDNSEdges(edges);
            const result: CustomEdgeModel[] = [
                {
                    id: 'edge-1',
                    type: 'edge',
                    source: 'node-1',
                    target: 'node-2',
                    data: {
                        isBidirectional: false,
                        portProtocolLabel: '80 TCP',
                        tag: '80 TCP',
                        sourceToTargetProperties: [
                            {
                                port: 80,
                                protocol: 'L4_PROTOCOL_TCP',
                                lastActiveTimestamp: '',
                            },
                        ],
                        targetToSourceProperties: [],
                        startTerminalType: 'none' as EdgeTerminalType,
                        endTerminalType: 'directional' as EdgeTerminalType,
                    },
                },
            ];
            expect(edgesWithoutDNSFlows).toEqual(result);
        });

        it('should remove a one-directional dns edge', () => {
            const edges: CustomEdgeModel[] = [
                {
                    id: 'edge-1',
                    type: 'edge',
                    source: 'node-1',
                    target: 'node-2',
                    data: {
                        isBidirectional: false,
                        portProtocolLabel: '53 UDP',
                        tag: '53 UDP',
                        sourceToTargetProperties: [
                            {
                                port: 53,
                                protocol: 'L4_PROTOCOL_UDP',
                                lastActiveTimestamp: '',
                            },
                        ],
                        endTerminalType: 'directional' as EdgeTerminalType,
                    },
                },
            ];
            const edgesWithoutDNSFlows = removeDNSEdges(edges);
            expect(edgesWithoutDNSFlows).toEqual([]);
        });

        it('should not remove a non-dns edge from a bi-directional edge', () => {
            const edges: CustomEdgeModel[] = [
                {
                    id: 'edge-1',
                    type: 'edge',
                    source: 'node-1',
                    target: 'node-2',
                    data: {
                        isBidirectional: true,
                        portProtocolLabel: '2',
                        tag: '2',
                        sourceToTargetProperties: [
                            {
                                port: 3000,
                                protocol: 'L4_PROTOCOL_TCP',
                                lastActiveTimestamp: '',
                            },
                        ],
                        targetToSourceProperties: [
                            {
                                port: 8080,
                                protocol: 'L4_PROTOCOL_TCP',
                                lastActiveTimestamp: '',
                            },
                        ],
                        endTerminalType: 'directional' as EdgeTerminalType,
                        startTerminalType: 'directional' as EdgeTerminalType,
                    },
                },
            ];
            const edgesWithoutDNSFlows = removeDNSEdges(edges);
            const result: CustomEdgeModel[] = [
                {
                    id: 'edge-1',
                    type: 'edge',
                    source: 'node-1',
                    target: 'node-2',
                    data: {
                        isBidirectional: true,
                        portProtocolLabel: '2',
                        tag: '2',
                        sourceToTargetProperties: [
                            {
                                port: 3000,
                                protocol: 'L4_PROTOCOL_TCP',
                                lastActiveTimestamp: '',
                            },
                        ],
                        targetToSourceProperties: [
                            {
                                port: 8080,
                                protocol: 'L4_PROTOCOL_TCP',
                                lastActiveTimestamp: '',
                            },
                        ],
                        endTerminalType: 'directional' as EdgeTerminalType,
                        startTerminalType: 'directional' as EdgeTerminalType,
                    },
                },
            ];
            expect(edgesWithoutDNSFlows).toEqual(result);
        });

        it('should remove a dns edge from a bi-directional edge', () => {
            const edges: CustomEdgeModel[] = [
                {
                    id: 'edge-1',
                    type: 'edge',
                    source: 'node-1',
                    target: 'node-2',
                    data: {
                        isBidirectional: true,
                        portProtocolLabel: '2',
                        tag: '2',
                        sourceToTargetProperties: [
                            {
                                port: 53,
                                protocol: 'L4_PROTOCOL_UDP',
                                lastActiveTimestamp: '',
                            },
                        ],
                        targetToSourceProperties: [
                            {
                                port: 8080,
                                protocol: 'L4_PROTOCOL_TCP',
                                lastActiveTimestamp: '',
                            },
                        ],
                        endTerminalType: 'directional' as EdgeTerminalType,
                        startTerminalType: 'directional' as EdgeTerminalType,
                    },
                },
            ];
            const edgesWithoutDNSFlows = removeDNSEdges(edges);
            const result: CustomEdgeModel[] = [
                {
                    id: 'edge-1',
                    type: 'edge',
                    source: 'node-1',
                    target: 'node-2',
                    data: {
                        isBidirectional: false,
                        portProtocolLabel: '8080 TCP',
                        tag: '8080 TCP',
                        sourceToTargetProperties: [],
                        targetToSourceProperties: [
                            {
                                port: 8080,
                                protocol: 'L4_PROTOCOL_TCP',
                                lastActiveTimestamp: '',
                            },
                        ],
                        endTerminalType: 'none' as EdgeTerminalType,
                        startTerminalType: 'directional' as EdgeTerminalType,
                    },
                },
            ];
            expect(edgesWithoutDNSFlows).toEqual(result);
        });
    });
});

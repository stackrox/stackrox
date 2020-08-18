import { filterModes } from 'constants/networkFilterModes';
import { getNetworkFlows } from './NetworkFlows';

const deploymentEdges = [
    {
        data: {
            destNodeId: '1',
            destNodeName: 'node-1',
            destNodeNamespace: 'namespace-a',
            traffic: 'ingress',
            isActive: true,
            portsAndProtocols: [
                {
                    port: 123,
                    protocol: 'L4_PROTOCOL_TCP',
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
            isActive: false,
            portsAndProtocols: [
                {
                    port: 456,
                    protocol: 'L4_PROTOCOL_TCP',
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
            isActive: false,
            portsAndProtocols: [
                {
                    port: 456,
                    protocol: 'L4_PROTOCOL_TCP',
                },
            ],
        },
    },
    {
        data: {
            destNodeId: '3',
            destNodeName: 'node-3',
            destNodeNamespace: 'namespace-a',
            traffic: 'bidirectional',
            isActive: true,
            portsAndProtocols: [
                {
                    port: 678,
                    protocol: 'L4_PROTOCOL_TCP',
                },
            ],
        },
    },
];

describe('getNetworkFlows', () => {
    it('should return all network flows', () => {
        const networkFlows = getNetworkFlows(deploymentEdges, filterModes.all);

        expect(networkFlows).toEqual([
            {
                connection: 'active',
                deploymentId: '1',
                deploymentName: 'node-1',
                namespace: 'namespace-a',
                portsAndProtocols: [
                    {
                        port: 123,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
                traffic: 'ingress',
            },
            {
                connection: 'allowed',
                deploymentId: '2',
                deploymentName: 'node-2',
                namespace: 'namespace-a',
                portsAndProtocols: [
                    {
                        port: 456,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
                traffic: 'egress',
            },
            {
                connection: 'active',
                deploymentId: '3',
                deploymentName: 'node-3',
                namespace: 'namespace-a',
                portsAndProtocols: [
                    {
                        port: 678,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
                traffic: 'bidirectional',
            },
        ]);
    });

    it('should return active network flows', () => {
        const networkFlows = getNetworkFlows(deploymentEdges, filterModes.active);

        expect(networkFlows).toEqual([
            {
                connection: 'active',
                deploymentId: '1',
                deploymentName: 'node-1',
                namespace: 'namespace-a',
                portsAndProtocols: [
                    {
                        port: 123,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
                traffic: 'ingress',
            },
            {
                connection: 'active',
                deploymentId: '3',
                deploymentName: 'node-3',
                namespace: 'namespace-a',
                portsAndProtocols: [
                    {
                        port: 678,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
                traffic: 'bidirectional',
            },
        ]);
    });

    it('should return allowed network flows', () => {
        const networkFlows = getNetworkFlows(deploymentEdges, filterModes.allowed);

        expect(networkFlows).toEqual([
            {
                connection: 'allowed',
                deploymentId: '2',
                deploymentName: 'node-2',
                namespace: 'namespace-a',
                portsAndProtocols: [
                    {
                        port: 456,
                        protocol: 'L4_PROTOCOL_TCP',
                    },
                ],
                traffic: 'egress',
            },
        ]);
    });
});

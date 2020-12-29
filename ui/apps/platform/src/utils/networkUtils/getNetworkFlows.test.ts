import { filterModes } from 'constants/networkFilterModes';
import { deploymentEdges } from 'utils/networkGraphUtils.test.constants';

import { getNetworkFlows } from './getNetworkFlows';

describe('getNetworkFlows', () => {
    it('should return all network flows', () => {
        const { networkFlows } = getNetworkFlows(deploymentEdges, filterModes.all);

        expect(networkFlows).toEqual([
            {
                connection: 'active',
                deploymentId: '1',
                entityName: 'node-1',
                namespace: 'namespace-a',
                portsAndProtocols: [
                    {
                        port: 111,
                        protocol: 'L4_PROTOCOL_TCP',
                        traffic: 'ingress',
                    },
                ],
                traffic: 'ingress',
                type: 'deployment',
            },
            {
                connection: 'allowed',
                deploymentId: '2',
                entityName: 'node-2',
                namespace: 'namespace-a',
                portsAndProtocols: [
                    {
                        port: 222,
                        protocol: 'L4_PROTOCOL_UDP',
                        traffic: 'egress',
                    },
                ],
                traffic: 'egress',
                type: 'deployment',
            },
            {
                connection: 'allowed',
                deploymentId: '3',
                entityName: 'node-3',
                namespace: 'namespace-a',
                portsAndProtocols: [
                    {
                        port: 333,
                        protocol: 'L4_PROTOCOL_TCP',
                        traffic: 'egress',
                    },
                ],
                traffic: 'egress',
                type: 'deployment',
            },
            {
                connection: 'active',
                deploymentId: '4',
                entityName: 'node-4',
                namespace: 'namespace-a',
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
                traffic: 'bidirectional',
                type: 'deployment',
            },
        ]);
    });

    it('should return active network flows', () => {
        const { networkFlows } = getNetworkFlows(deploymentEdges, filterModes.active);

        expect(networkFlows).toEqual([
            {
                connection: 'active',
                deploymentId: '1',
                entityName: 'node-1',
                namespace: 'namespace-a',
                portsAndProtocols: [
                    {
                        port: 111,
                        protocol: 'L4_PROTOCOL_TCP',
                        traffic: 'ingress',
                    },
                ],
                traffic: 'ingress',
                type: 'deployment',
            },
            {
                connection: 'active',
                deploymentId: '4',
                entityName: 'node-4',
                namespace: 'namespace-a',
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
                traffic: 'bidirectional',
                type: 'deployment',
            },
        ]);
    });

    it('should return allowed network flows', () => {
        const { networkFlows } = getNetworkFlows(deploymentEdges, filterModes.allowed);

        expect(networkFlows).toEqual([
            {
                connection: 'allowed',
                deploymentId: '1',
                entityName: 'node-1',
                namespace: 'namespace-a',
                portsAndProtocols: [
                    {
                        port: 111,
                        protocol: 'L4_PROTOCOL_TCP',
                        traffic: 'ingress',
                    },
                ],
                traffic: 'ingress',
                type: 'deployment',
            },
            {
                connection: 'allowed',
                deploymentId: '2',
                entityName: 'node-2',
                namespace: 'namespace-a',
                portsAndProtocols: [
                    {
                        port: 222,
                        protocol: 'L4_PROTOCOL_UDP',
                        traffic: 'egress',
                    },
                ],
                traffic: 'egress',
                type: 'deployment',
            },
            {
                connection: 'allowed',
                deploymentId: '3',
                entityName: 'node-3',
                namespace: 'namespace-a',
                portsAndProtocols: [
                    {
                        port: 333,
                        protocol: 'L4_PROTOCOL_TCP',
                        traffic: 'egress',
                    },
                ],
                traffic: 'egress',
                type: 'deployment',
            },
            {
                connection: 'allowed',
                deploymentId: '4',
                entityName: 'node-4',
                namespace: 'namespace-a',
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
                traffic: 'bidirectional',
                type: 'deployment',
            },
        ]);
    });

    it('should return the correct number of directional flows', () => {
        const { numIngressFlows, numEgressFlows } = getNetworkFlows(
            deploymentEdges,
            filterModes.all
        );

        expect(numIngressFlows).toEqual(2);
        expect(numEgressFlows).toEqual(3);
    });
});

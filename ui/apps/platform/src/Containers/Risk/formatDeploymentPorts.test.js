import { formatDeploymentPorts } from './DeploymentDetails';

describe('formatDeploymentPorts', () => {
    it('should set the appropriate exposure message', () => {
        const ports = getPorts();

        const formattedPorts = formatDeploymentPorts(ports);

        expect(formattedPorts[0].exposure).toEqual('ClusterIP');
        expect(formattedPorts[1].exposure).toEqual('LoadBalancer');
        expect(formattedPorts[2].exposure).toEqual('NodePort');
        expect(formattedPorts[3].exposure).toEqual('HostPort');
        expect(formattedPorts[4].exposure).toEqual('Exposure type is not set');
    });
});

function getPorts() {
    return [
        {
            name: 'dns-tcp',
            containerPort: 53,
            protocol: 'TCP',
            exposure: 'INTERNAL',
            exposedPort: 0,
            exposureInfos: [
                {
                    level: 'INTERNAL',
                    serviceName: 'kube-dns',
                    serviceId: 'dc2544dd-bbe9-47a2-9fa4-7c6c52cc3040',
                    serviceClusterIp: '10.11.240.10',
                    servicePort: 53,
                    nodePort: 0,
                    externalIps: [],
                    externalHostnames: [],
                },
            ],
        },
        {
            name: 'dns-local',
            containerPort: 10053,
            protocol: 'UDP',
            exposure: 'EXTERNAL',
            exposedPort: 0,
            exposureInfos: [],
        },
        {
            name: 'dns-tcp-local',
            containerPort: 10053,
            protocol: 'TCP',
            exposure: 'NODE',
            exposedPort: 0,
            exposureInfos: [],
        },
        {
            name: 'metrics',
            containerPort: 10054,
            protocol: 'TCP',
            exposure: 'HOST',
            exposedPort: 0,
            exposureInfos: [],
        },
        {
            name: 'metrics',
            containerPort: 10055,
            protocol: 'TCP',
            exposure: 'UNSET',
            exposedPort: 0,
            exposureInfos: [],
        },
    ];
}

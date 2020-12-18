const baselineStatuses = [
    {
        peer: {
            entity: {
                id: '07f8ff42-7b46-491b-9db6-bb36fde638ad',
                type: 'DEPLOYMENT',
                name: 'deployment-1',
                namespace: 'namespace-1',
            },
            port: '3000',
            protocol: 'L4_PROTOCOL_TCP',
            ingress: true,
            state: 'active',
        },
        status: 'BASELINE',
    },
    {
        peer: {
            entity: {
                id: '27f8ff42-7b46-491b-9db6-bb36fde638ad',
                type: 'DEPLOYMENT',
                name: 'deployment-2',
                namespace: 'namespace-1',
            },
            port: '4000',
            protocol: 'L4_PROTOCOL_TCP',
            ingress: true,
            state: 'active',
        },
        status: 'ANOMALOUS',
    },
    {
        peer: {
            entity: {
                id: '32f8ff42-7b46-491b-9db6-bb36fde638ad',
                type: 'DEPLOYMENT',
                name: 'deployment-3',
                namespace: 'namespace-1',
            },
            port: '5000',
            protocol: 'L4_PROTOCOL_TCP',
            ingress: false,
            state: 'active',
        },
        status: 'ANOMALOUS',
    },
    {
        peer: {
            entity: {
                id: '53f8ff42-7b46-491b-9db6-bb36fde638ad',
                type: 'DEPLOYMENT',
                name: 'deployment-4',
                namespace: 'namespace-1',
            },
            port: '6000',
            protocol: 'L4_PROTOCOL_UDP',
            ingress: true,
            state: 'active',
        },
        status: 'BASELINE',
    },
    {
        peer: {
            entity: {
                id: '43f8ff42-7b46-491b-9db6-bb36fde638ad',
                type: 'DEPLOYMENT',
                name: 'deployment-5',
                namespace: 'namespace-1',
            },
            port: '7000',
            protocol: 'L4_PROTOCOL_UDP',
            ingress: false,
            state: 'active',
        },
        status: 'BASELINE',
    },
];

export default baselineStatuses;

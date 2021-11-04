import React from 'react';
import { render } from '@testing-library/react';

import ObjectDescriptionList from './ObjectDescriptionList';

const ports = [
    {
        name: '',
        containerPort: 22,
        protocol: 'TCP',
        exposure: 'UNSET',
        exposedPort: 0,
        exposureInfos: [],
    },
    {
        name: '',
        containerPort: 8080,
        protocol: 'TCP',
        exposure: 'INTERNAL',
        exposedPort: 0,
        exposureInfos: [
            {
                level: 'INTERNAL',
                serviceName: 'visa-processor-service',
                serviceId: 'eb1af0b8-a6da-11ea-a6c6-42010a800049',
                serviceClusterIp: '10.19.243.95',
                servicePort: 8080,
                nodePort: 0,
                externalIps: [],
                externalHostnames: [],
            },
        ],
    },
];

describe('ObjectDescriptionList', () => {
    test('annotations', () => {
        const { container } = render(
            <ObjectDescriptionList data={{ 'deprecated.daemonset.template.generation': '1' }} />
        );

        expect(container).toMatchSnapshot();
    });

    test('labels', () => {
        const { container } = render(<ObjectDescriptionList data={{ 'k8s-app': 'kube-proxy' }} />);

        expect(container).toMatchSnapshot();
    });

    test('ports', () => {
        const { container } = render(
            <>
                {ports.map((port, index) => (
                    // eslint-disable-next-line react/no-array-index-key
                    <ObjectDescriptionList key={index} data={port} />
                ))}
            </>
        );

        expect(container).toMatchSnapshot();
    });
});

import React from 'react';
import { render } from '@testing-library/react';

import CveType from './CveType';

describe('CveType', () => {
    describe('default style', () => {
        test('should show an image CVE type in default style', async () => {
            const { asFragment } = render(<CveType types={['IMAGE_CVE']} />);

            expect(asFragment()).toMatchSnapshot();
        });

        test('should show a node CVE type in default style', async () => {
            const { asFragment } = render(<CveType types={['NODE_CVE']} />);

            expect(asFragment()).toMatchSnapshot();
        });

        test('should show a Kubernetes CVE type in default style', async () => {
            const { asFragment } = render(<CveType types={['K8S_CVE']} />);

            expect(asFragment()).toMatchSnapshot();
        });

        test('should show an OpenShift CVE type in default style', async () => {
            const { asFragment } = render(<CveType types={['OPENSHIFT_CVE']} />);

            expect(asFragment()).toMatchSnapshot();
        });

        test('should show an Istio CVE type in default style', async () => {
            const { asFragment } = render(<CveType types={['ISTIO_CVE']} />);

            expect(asFragment()).toMatchSnapshot();
        });

        test('should show an Image and Node CVE type in default style', async () => {
            const { asFragment } = render(<CveType types={['IMAGE_CVE', 'NODE_CVE']} />);

            expect(asFragment()).toMatchSnapshot();
        });

        test('should show a Node and Kubernetes CVE type in default style', async () => {
            const { asFragment } = render(<CveType types={['NODE_CVE', 'K8S_CVE']} />);

            expect(asFragment()).toMatchSnapshot();
        });

        test('should show a Kubernetes and OpenShift CVE type in default style', async () => {
            const { asFragment } = render(<CveType types={['K8S_CVE', 'OPENSHIFT_CVE']} />);

            expect(asFragment()).toMatchSnapshot();
        });
    });
});

import React from 'react';
import { render } from '@testing-library/react';

import CveType from './CveType';

describe('CveType', () => {
    describe('default style', () => {
        test('should show an image CVE type in default style', async () => {
            const { asFragment } = render(<CveType type="IMAGE_CVE" />);

            expect(asFragment()).toMatchSnapshot();
        });

        test('should show a Kubernetes CVE type in default style', async () => {
            const { asFragment } = render(<CveType type="K8S_CVE" />);

            expect(asFragment()).toMatchSnapshot();
        });

        test('should show an Istio CVE type in default style', async () => {
            const { asFragment } = render(<CveType type="ISTIO_CVE" />);

            expect(asFragment()).toMatchSnapshot();
        });
    });

    describe('callout style', () => {
        test('should show an image CVE type in callout style', async () => {
            const { asFragment } = render(<CveType context="callout" type="IMAGE_CVE" />);

            expect(asFragment()).toMatchSnapshot();
        });

        test('should show a Kubernetes CVE type in callout style', async () => {
            const { asFragment } = render(<CveType context="callout" type="K8S_CVE" />);

            expect(asFragment()).toMatchSnapshot();
        });

        test('should show an Istio CVE type in callout style', async () => {
            const { asFragment } = render(<CveType context="callout" type="ISTIO_CVE" />);

            expect(asFragment()).toMatchSnapshot();
        });
    });
});

import React from 'react';
import { MockedProvider } from '@apollo/client/testing';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/extend-expect';

import renderWithRouter from 'test-utils/renderWithRouter';
import ScopeBar, { namespacesQuery } from './ScopeBar';
import { Cluster, Namespace } from './types';

jest.setTimeout(10000);

const clusterNamespaces = {
    production: ['backend', 'default', 'frontend', 'kube-system', 'medical', 'payments'],
    security: ['default', ' kube-system', 'stackrox'],
};

const mockData: {
    clusters: Cluster[];
} = { clusters: [] };

Object.entries(clusterNamespaces).forEach(([mockCluster, mockNamespaces], clusterIndex) => {
    const namespaces = mockNamespaces.map((ns, nsIndex) => ({
        metadata: { id: `ns-${nsIndex}`, name: ns },
    }));
    mockData.clusters.push({ id: `cluster-${clusterIndex}`, name: mockCluster, namespaces });
});

const mocks = [
    {
        request: { query: namespacesQuery, variables: { query: '' } },
        result: { data: mockData },
    },
];

beforeEach(() => {
    jest.resetModules();
});

const setup = () => {
    const user = userEvent.setup();
    const utils = renderWithRouter(
        <MockedProvider mocks={mocks} addTypename={false}>
            <ScopeBar />
        </MockedProvider>
    );
    return { user, utils };
};

describe('Resource scope bar', () => {
    it('should default to all clusters and namespaces selected', async () => {
        const { user } = setup();

        const clusterDropdownToggle = screen.getByLabelText('Select clusters');
        const namespaceDropdownToggle = screen.getByLabelText('Select namespaces');
        await waitFor(() => expect(clusterDropdownToggle).not.toBeDisabled());

        // The default state is all clusters selected, with the ns dropdown disabled
        await user.click(clusterDropdownToggle);
        expect(await screen.findByLabelText('All clusters', { selector: 'input' })).toBeChecked();
        expect(namespaceDropdownToggle).toBeDisabled();
        await user.click(clusterDropdownToggle);
    });

    it('allows selection of multiple clusters and namespaces', async () => {
        const { user } = setup();

        const clusterDropdownToggle = screen.getByLabelText('Select clusters');
        const namespaceDropdownToggle = screen.getByLabelText('Select namespaces');
        await waitFor(() => expect(clusterDropdownToggle).not.toBeDisabled());

        // Selecting one or more clusters enables the ns dropdown
        await user.click(clusterDropdownToggle);
        await user.click(screen.getByLabelText('production'));
        await user.click(clusterDropdownToggle);
        expect(namespaceDropdownToggle).not.toBeDisabled();
        expect(clusterDropdownToggle).toHaveTextContent('Clusters1');

        // Enable some namespaces and check that the select badge updates
        await user.click(namespaceDropdownToggle);
        await user.click(screen.getByLabelText('frontend'));
        await user.click(screen.getByLabelText('backend'));
        await user.click(screen.getByLabelText('payments'));
        await user.click(namespaceDropdownToggle);
        expect(namespaceDropdownToggle).toHaveTextContent('Namespaces3');

        // Selecting another cluster when other namespaces are selected will explicitly select
        // all namespaces in that cluster
        await user.click(clusterDropdownToggle);
        await user.click(screen.getByLabelText('security'));
        await user.click(clusterDropdownToggle);
        expect(namespaceDropdownToggle).toHaveTextContent('Namespaces6');

        // Selecting "All Namespaces" and then a single namespaces will result in a single
        // namespace being selected
        await user.click(namespaceDropdownToggle);
        await user.click(screen.getByLabelText('All namespaces'));
        await user.click(screen.getByLabelText('frontend'));
        await user.click(namespaceDropdownToggle);
        expect(namespaceDropdownToggle).toHaveTextContent('Namespaces1');

        // Selecting "All Clusters" will clear the namespace selection and disable the dropdown
        await user.click(clusterDropdownToggle);
        await user.click(screen.getByLabelText('All clusters'));
        await user.click(clusterDropdownToggle);
        expect(clusterDropdownToggle).toHaveTextContent('All clusters');
        expect(namespaceDropdownToggle).toHaveTextContent('All namespaces');
        expect(namespaceDropdownToggle).toBeDisabled();
    });

    it('will track selected clusters and namespaces in the page URL', async () => {
        const {
            user,
            utils: { history },
        } = setup();

        // Check that the default state of "select all" results in empty URL search parameters
        const clusterDropdownToggle = screen.getByLabelText('Select clusters');
        const namespaceDropdownToggle = screen.getByLabelText('Select namespaces');
        await waitFor(() => expect(clusterDropdownToggle).not.toBeDisabled());
        expect(history.location.search).toBe('');

        // Select a cluster and verify it has been added to the URL
        await user.click(clusterDropdownToggle);
        await user.click(screen.getByLabelText('production'));
        await user.click(clusterDropdownToggle);
        expect(history.location.search).toMatch(new RegExp(`s\\[Cluster\\]\\[\\d\\]=production`));

        // Select multiple namespaces and verify they have been added to the URL
        const productionNamespaces =
            mockData.clusters.find((cs) => cs.name === 'production')?.namespaces ?? [];
        function findNsWithName(name) {
            return ({ metadata }) => metadata.name === name;
        }

        // Get a reference to the mock data object for each namespace so we can compare names against ids
        const frontend = productionNamespaces.find(findNsWithName('frontend')) as Namespace;
        const backend = productionNamespaces.find(findNsWithName('backend')) as Namespace;
        const payments = productionNamespaces.find(findNsWithName('payments')) as Namespace;
        await user.click(namespaceDropdownToggle);
        await user.click(screen.getByLabelText(frontend.metadata.name));
        await user.click(screen.getByLabelText(backend.metadata.name));
        await user.click(namespaceDropdownToggle);
        expect(history.location.search).toMatch(new RegExp(`s\\[Cluster\\]\\[\\d\\]=production`));
        // Namespaces are tracked _by id_ in the URL, not name
        expect(history.location.search).toMatch(
            new RegExp(`s\\[Namespace ID\\]\\[\\d\\]=${frontend.metadata.id}`)
        );
        expect(history.location.search).toMatch(
            new RegExp(`s\\[Namespace ID\\]\\[\\d\\]=${backend.metadata.id}`)
        );
        expect(history.location.search).not.toMatch(
            new RegExp(`s\\[Namespace ID\\]\\[\\d\\]=${payments.metadata.id}`)
        );
    });
});

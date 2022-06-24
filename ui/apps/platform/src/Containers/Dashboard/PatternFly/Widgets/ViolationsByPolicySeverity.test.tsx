import React from 'react';
import { MockedProvider } from '@apollo/client/testing';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/extend-expect';

import renderWithRouter from 'test-utils/renderWithRouter';
import ViolationsByPolicyCategory from 'Containers/Dashboard/ViolationsByPolicyCategory';
import { mostRecentAlertsQuery } from './ViolationsByPolicySeverity';

const mockAlerts = [
    {
        id: '1',
        time: '2022-06-24T02:20:00.703746138Z',
        deployment: { clusterName: 'security', namespace: 'stackrox', name: 'central' },
        policy: { name: 'Unauthorized Network Flow', severity: 'CRITICAL_SEVERITY' },
    },
    {
        id: '2',
        time: '2022-06-24T02:20:00.704383154Z',
        deployment: { clusterName: 'security', namespace: 'stackrox', name: 'scanner' },
        policy: { name: 'Unauthorized Network Flow', severity: 'CRITICAL_SEVERITY' },
    },
    {
        id: '3',
        time: '2022-06-24T00:35:42.299667447Z',
        deployment: { clusterName: 'production', namespace: 'kube-system', name: 'kube-proxy' },
        policy: { name: 'Ubuntu Package Manager in Image', severity: 'CRITICAL_SEVERITY' },
    },
];

const mocks = [
    {
        request: {
            query: mostRecentAlertsQuery,
            variables: {
                query: 'Severity:CRITICAL_SEVERITY',
            },
        },
        result: {
            data: {
                alerts: mockAlerts,
            },
        },
    },
];

jest.mock('hooks/useResizeObserver', () => ({
    __esModule: true,
    default: jest.fn().mockImplementation(jest.fn),
}));

beforeEach(() => {
    jest.resetModules();
});

function setup() {
    const user = userEvent.setup();
    const utils = renderWithRouter(
        <MockedProvider mocks={mocks} addTypename={false}>
            <ViolationsByPolicyCategory />
        </MockedProvider>
    );

    return { user, utils };
}

describe('Images at most risk dashboard widget', () => {
    it('should render the correct title based on selected options', async () => {
        const { user } = setup();

        // Default is display all images
        expect(
            await screen.findByRole('heading', {
                name: 'All images at most risk',
            })
        ).toBeInTheDocument();

        // Change to display only active images
        await user.click(await screen.findByRole('button', { name: `Options` }));
        await user.click(await screen.findByRole('button', { name: `Active images` }));

        expect(
            await screen.findByRole('heading', {
                name: 'Active images at most risk',
            })
        ).toBeInTheDocument();
    });
});

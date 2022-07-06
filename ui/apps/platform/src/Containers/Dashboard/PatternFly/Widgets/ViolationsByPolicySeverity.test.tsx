import React from 'react';
import { MockedProvider } from '@apollo/client/testing';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/extend-expect';

import renderWithRouter from 'test-utils/renderWithRouter';
import { violationsBasePath } from 'routePaths';
import ViolationsByPolicySeverity, { mostRecentAlertsQuery } from './ViolationsByPolicySeverity';

const mockAlerts = [
    {
        id: '1',
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

// Mock the hook that handles the data fetching of alert counts
jest.mock('Containers/Dashboard/PatternFly/hooks/useAlertGroups', () => ({
    __esModule: true,
    default: () => ({
        data: [
            {
                group: '',
                counts: [
                    { severity: 'LOW_SEVERITY', count: '220' },
                    { severity: 'MEDIUM_SEVERITY', count: '70' },
                    { severity: 'HIGH_SEVERITY', count: '140' },
                    { severity: 'CRITICAL_SEVERITY', count: '3' },
                ],
            },
        ],
        loading: false,
        error: undefined,
    }),
}));

beforeEach(() => {
    jest.resetModules();
});

function setup() {
    const user = userEvent.setup();
    const utils = renderWithRouter(
        <MockedProvider mocks={mocks} addTypename={false}>
            <ViolationsByPolicySeverity />
        </MockedProvider>
    );

    return { user, utils };
}

describe('Violations by policy severity widget', () => {
    it('should display total violations in the title that match the sum of the individual tiles', async () => {
        setup();

        // Find items on the screen that with text that contains -only- an integer
        const tiles = await screen.findAllByText(/^\d+$/);
        expect(tiles).toHaveLength(4);

        let alertCount = 0;
        tiles.forEach((tile) => {
            alertCount += parseInt(tile.textContent ?? '0', 10);
        });

        expect(
            await screen.findByRole('heading', {
                name: `${alertCount} policy violations by severity`,
            })
        ).toBeInTheDocument();
    });

    it('should link to the correct violations pages when clicking links in the widget', async () => {
        const {
            user,
            utils: { history },
        } = setup();

        expect(
            await screen.findByRole('heading', { name: /policy violations by severity/g })
        ).toBeInTheDocument();

        // Test the 'View all' violations link button
        await user.click(await screen.findByRole('link', { name: 'View all' }));
        expect(history.location.pathname).toBe(`${violationsBasePath}`);
        expect(history.location.search).toContain('sortOption[field]=Severity');

        // Test links from the violation count tiles
        await user.click(await screen.findByRole('link', { name: '220 Low' }));
        expect(history.location.pathname).toBe(`${violationsBasePath}`);
        expect(history.location.search).toContain('[Severity]=LOW_SEVERITY');
        await user.click(await screen.findByRole('link', { name: '3 Critical' }));
        expect(history.location.pathname).toBe(`${violationsBasePath}`);
        expect(history.location.search).toContain('[Severity]=CRITICAL_SEVERITY');

        // Test links from the 'most recent violations' section
        await user.click(await screen.findByRole('link', { name: /ubuntu package manager/gi }));
        expect(history.location.pathname).toBe(`${violationsBasePath}/${mockAlerts[0].id}`);
    });
});

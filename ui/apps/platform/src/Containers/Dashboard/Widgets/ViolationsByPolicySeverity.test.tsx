import React from 'react';
import { MockedProvider } from '@apollo/client/testing';
import { screen, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/extend-expect';

import renderWithRouter from 'test-utils/renderWithRouter';
import { withTextContent } from 'test-utils/queryUtils';
import { violationsBasePath } from 'routePaths';
import ViolationsByPolicySeverity, {
    alertsBySeverityQuery,
    mostRecentAlertsQuery,
} from './ViolationsByPolicySeverity';

const mockAlerts = [
    {
        id: '1',
        time: '2022-06-24T00:35:42.299667447Z',
        deployment: { clusterName: 'production', namespace: 'kube-system', name: 'kube-proxy' },
        resource: null,
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
    {
        request: {
            query: alertsBySeverityQuery,
            variables: {
                lowQuery: 'Severity:LOW_SEVERITY',
                medQuery: 'Severity:MEDIUM_SEVERITY',
                highQuery: 'Severity:HIGH_SEVERITY',
                critQuery: 'Severity:CRITICAL_SEVERITY',
            },
        },
        result: {
            data: {
                LOW_SEVERITY: 220,
                MEDIUM_SEVERITY: 70,
                HIGH_SEVERITY: 140,
                CRITICAL_SEVERITY: 3,
            },
        },
    },
];

jest.mock('hooks/useResizeObserver');

beforeEach(() => {
    jest.resetModules();
});

function setup() {
    // Ignore false positive, see: https://github.com/testing-library/eslint-plugin-testing-library/issues/800
    // eslint-disable-next-line testing-library/await-async-events
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

        // Ensure the data has loaded
        expect(await screen.findByText(withTextContent(/220 ?Low/))).toBeInTheDocument();

        const tiles = await screen.findAllByText(
            withTextContent(/^\d+ ?(Low|Medium|High|Critical)$/)
        );
        expect(tiles).toHaveLength(4);

        let alertCount = 0;
        tiles.forEach((tile) => {
            alertCount += parseInt(tile.textContent ?? '0', 10);
        });

        expect(
            await screen.findByText(`${alertCount} policy violations by severity`)
        ).toBeInTheDocument();
    });

    it('should link to the correct violations pages when clicking links in the widget', async () => {
        const {
            user,
            utils: { history },
        } = setup();

        // Ensure the data has loaded
        expect(await screen.findByText(withTextContent(/220 ?Low/))).toBeInTheDocument();

        // Test the 'View all' violations link button
        await act(() => user.click(screen.getByText('View all')));
        expect(history.location.pathname).toBe(`${violationsBasePath}`);
        expect(history.location.search).toContain('sortOption[field]=Severity');

        // Test links from the violation count tiles
        await act(() => user.click(screen.getByText('Low')));
        expect(history.location.pathname).toBe(`${violationsBasePath}`);
        expect(history.location.search).toContain('[Severity]=LOW_SEVERITY');
        await act(() => user.click(screen.getByText('Critical')));
        expect(history.location.pathname).toBe(`${violationsBasePath}`);
        expect(history.location.search).toContain('[Severity]=CRITICAL_SEVERITY');

        // Test links from the 'most recent violations' section
        await act(async () => user.click(await screen.findByText(/ubuntu package manager/i)));
        expect(history.location.pathname).toBe(`${violationsBasePath}/${mockAlerts[0].id}`);
    });
});

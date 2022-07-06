import React from 'react';
import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/extend-expect';

import renderWithRouter from 'test-utils/renderWithRouter';
import ViolationsByPolicyCategory from 'Containers/Dashboard/PatternFly/Widgets/ViolationsByPolicyCategory';

jest.mock('hooks/useResizeObserver', () => ({
    __esModule: true,
    default: jest.fn().mockImplementation(jest.fn),
}));

// Mock the hook that handles the data fetching of alert counts
jest.mock('Containers/Dashboard/PatternFly/hooks/useAlertGroups', () => {
    function makeFixtureCounts(crit: number, high: number, med: number, low: number) {
        return [
            { severity: 'CRITICAL_SEVERITY', count: `${crit}` },
            { severity: 'HIGH_SEVERITY', count: `${high}` },
            { severity: 'MEDIUM_SEVERITY', count: `${med}` },
            { severity: 'LOW_SEVERITY', count: `${low}` },
        ];
    }

    return {
        __esModule: true,
        default: () => ({
            data: [
                { counts: makeFixtureCounts(5, 20, 30, 10), group: 'Anomalous Activity' },
                { counts: makeFixtureCounts(5, 2, 30, 5), group: 'Docker CIS' },
                { counts: makeFixtureCounts(10, 20, 5, 5), group: 'Network Tools' },
                { counts: makeFixtureCounts(15, 2, 10, 5), group: 'Security Best Practices' },
                { counts: makeFixtureCounts(20, 10, 2, 10), group: 'Privileges' },
                { counts: makeFixtureCounts(15, 8, 10, 5), group: 'Vulnerability Management' },
            ],
            loading: false,
            error: undefined,
        }),
    };
});

const setup = () => {
    const user = userEvent.setup();
    const utils = renderWithRouter(<ViolationsByPolicyCategory />);

    return { user, utils };
};

// Extract the text from provided link elements
function getLinkCategories(links: HTMLElement[]) {
    return links.map((link) => link.textContent);
}

/**
 * Waits for the text in axis links of the chart to equal the provided array.
 *
 * @param linkText An array of string that should match the order of axis labels in the chart
 */
function waitForAxisLinksToBe(linkText: string[]) {
    return waitFor(() => {
        const chart = screen.getByLabelText(/policy violations by category/i);
        const links = within(chart).getAllByRole('link');
        const categories = getLinkCategories(links);
        expect(categories).toEqual(linkText);
    });
}

describe('Violations by policy category widget', () => {
    it('should sort a policy violations by category widget by severity and volume of violations', async () => {
        const { user } = setup();

        expect(
            await screen.findByRole('heading', { name: /Policy violations by category/g })
        ).toBeInTheDocument();

        // Default sorting should be by severity of critical and high Violations, with critical taking priority.
        await waitForAxisLinksToBe([
            'Anomalous Activity',
            'Network Tools',
            'Security Best Practices',
            'Vulnerability Management',
            'Privileges',
        ]);

        // Switch to sort-by-volume, which orders the chart by total violations per category
        await user.click(screen.getByRole('button', { name: 'Options' }));
        await user.click(screen.getByRole('button', { name: 'Volume' }));
        await user.click(screen.getByRole('button', { name: 'Options' }));

        await waitForAxisLinksToBe([
            'Security Best Practices',
            'Vulnerability Management',
            'Anomalous Activity',
            'Privileges',
            'Network Tools',
        ]);
    });

    it('should allow toggling of severities for a policy violations by category widget', async () => {
        const { user } = setup();

        expect(
            await screen.findByRole('heading', { name: /Policy violations by category/g })
        ).toBeInTheDocument();

        // Sort by volume, so that enabling lower severity bars changes the order of the chart
        await user.click(screen.getByRole('button', { name: 'Options' }));
        await user.click(screen.getByRole('button', { name: 'Volume' }));
        await user.click(screen.getByRole('button', { name: 'Options' }));

        // Toggle on low and medium violations, which are disabled by default
        await user.click(screen.getByText('Low'));
        await user.click(screen.getByText('Medium'));

        await waitForAxisLinksToBe([
            'Vulnerability Management',
            'Network Tools',
            'Privileges',
            'Docker CIS',
            'Anomalous Activity',
        ]);
    });
});

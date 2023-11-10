import React from 'react';
import { screen, waitFor, within, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/extend-expect';

import renderWithRouter from 'test-utils/renderWithRouter';
import { mockChartsWithoutAnimation } from 'test-utils/mocks/@patternfly/react-charts';
import ViolationsByPolicyCategory from './ViolationsByPolicyCategory';

jest.mock('@patternfly/react-charts', () => mockChartsWithoutAnimation);
jest.mock('hooks/useResizeObserver');

// Mock the hook that handles the data fetching of alert counts
jest.mock('Containers/Dashboard/hooks/useAlertGroups', () => {
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

beforeEach(() => {
    localStorage.clear();
});

const setup = () => {
    // Ignore false positive, see: https://github.com/testing-library/eslint-plugin-testing-library/issues/800
    // eslint-disable-next-line testing-library/await-async-events
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
        const chart = screen.getByRole('img');
        const links = within(chart).getAllByRole('link');
        const categories = getLinkCategories(links);
        expect(categories).toEqual(linkText);
    });
}

describe('Violations by policy category widget', () => {
    it('should sort a policy violations by category widget by severity and volume of violations', async () => {
        const { user } = setup();

        expect(await screen.findByText('Anomalous Activity')).toBeInTheDocument();

        // Default sorting should be by severity of critical and high Violations, with critical taking priority.
        await waitForAxisLinksToBe([
            'Anomalous Activity',
            'Network Tools',
            'Security Best Practices',
            'Vulnerability Management',
            'Privileges',
        ]);

        // Switch to sort-by-volume, which orders the chart by total violations per category
        await act(() => user.click(screen.getByLabelText('Options')));
        await act(() => user.click(screen.getByText('Total')));
        await act(() => user.click(screen.getByLabelText('Options')));

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

        expect(await screen.findByText('Anomalous Activity')).toBeInTheDocument();

        // Sort by volume, so that enabling lower severity bars changes the order of the chart
        await act(() => user.click(screen.getByLabelText('Options')));
        await act(() => user.click(screen.getByText('Total')));
        await act(() => user.click(screen.getByLabelText('Options')));

        // Toggle on low and medium violations, which are disabled by default
        await act(() => user.click(screen.getByText('Low')));
        await act(() => user.click(screen.getByText('Medium')));

        await waitForAxisLinksToBe([
            'Vulnerability Management',
            'Network Tools',
            'Privileges',
            'Docker CIS',
            'Anomalous Activity',
        ]);
    });

    it('should contain a button that resets the widget options to default', async () => {
        setup();
        const user = userEvent.setup({ skipHover: true });

        await act(() => user.click(screen.getByLabelText('Options')));
        const [severity, total, all, deploy, runtime] = await screen.findAllByRole('button', {
            name: /Severity|Total|All|Deploy|Runtime/,
        });

        // Defaults
        expect(severity).toHaveAttribute('aria-pressed', 'true');
        expect(total).toHaveAttribute('aria-pressed', 'false');
        expect(all).toHaveAttribute('aria-pressed', 'true');
        expect(deploy).toHaveAttribute('aria-pressed', 'false');
        expect(runtime).toHaveAttribute('aria-pressed', 'false');

        // Change some options
        await act(() => user.click(total));
        await act(() => user.click(runtime));

        expect(severity).toHaveAttribute('aria-pressed', 'false');
        expect(total).toHaveAttribute('aria-pressed', 'true');
        expect(all).toHaveAttribute('aria-pressed', 'false');
        expect(deploy).toHaveAttribute('aria-pressed', 'false');
        expect(runtime).toHaveAttribute('aria-pressed', 'true');

        const resetButton = await screen.findByLabelText('Revert to default options');
        await act(() => user.click(resetButton));

        expect(severity).toHaveAttribute('aria-pressed', 'true');
        expect(total).toHaveAttribute('aria-pressed', 'false');
        expect(all).toHaveAttribute('aria-pressed', 'true');
        expect(deploy).toHaveAttribute('aria-pressed', 'false');
        expect(runtime).toHaveAttribute('aria-pressed', 'false');
    });
});

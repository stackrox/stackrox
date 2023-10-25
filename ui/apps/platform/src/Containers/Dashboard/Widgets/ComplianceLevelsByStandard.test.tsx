import React from 'react';
import { MockedProvider } from '@apollo/client/testing';
import { screen, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/extend-expect';

import renderWithRouter from 'test-utils/renderWithRouter';
import { mockChartsWithoutAnimation } from 'test-utils/mocks/@patternfly/react-charts';
import { AGGREGATED_RESULTS_ACROSS_ENTITIES } from 'queries/controls';
import entityTypes, { standardEntityTypes } from 'constants/entityTypes';
import { complianceBasePath, urlEntityListTypes } from 'routePaths';
import ComplianceLevelsByStandard from './ComplianceLevelsByStandard';

/*
These standards have been formatted for easier verification of the expected ordering in
tests compared to a direct hard coding in the mocked response below.

id: [name, numFailing, numPassing]
*/
const standards = {
    CIS_Kubernetes_v1_5: ['CIS Kubernetes v1.5', 8, 2],
    HIPAA_164: ['HIPAA 164', 7, 3],
    NIST_800_190: ['NIST SP 800-190', 6, 4],
    NIST_SP_800_53_Rev_4: ['NIST SP 800-53', 5, 5],
    PCI_DSS_3_2: ['PCI DSS 3.2.1', 4, 6],
    'ocp4-cis': ['ocp4-cis', 3, 7],
    'ocp4-cis-node': ['ocp4-cis-node', 2, 8],
};

const mocks = [
    {
        request: {
            query: AGGREGATED_RESULTS_ACROSS_ENTITIES,
            variables: {
                groupBy: [entityTypes.STANDARD],
                where: 'Cluster:*',
            },
        },
        result: {
            data: {
                controls: {
                    results: Object.entries(standards).map(([id, [, numFailing, numPassing]]) => ({
                        aggregationKeys: [{ id, scope: 'STANDARD' }],
                        numFailing,
                        numPassing,
                        numSkipped: 0,
                        unit: 'CONTROL',
                    })),
                },
                complianceStandards: Object.entries(standards).map(([id, [name]]) => ({
                    id,
                    name,
                })),
            },
        },
    },
];

jest.mock('@patternfly/react-charts', () => mockChartsWithoutAnimation);
jest.mock('hooks/useResizeObserver');

beforeEach(() => {
    localStorage.clear();
});

const setup = () => {
    // Ignore false positive, see: https://github.com/testing-library/eslint-plugin-testing-library/issues/800
    const user = userEvent.setup();
    const utils = renderWithRouter(
        <MockedProvider mocks={mocks} addTypename={false}>
            <ComplianceLevelsByStandard />
        </MockedProvider>
    );

    return { user, utils };
};

describe('Compliance levels by standard dashboard widget', () => {
    it('should render graph bars correctly order by compliance percentage', async () => {
        const { user } = setup();

        // Allow graph to load
        await screen.findByLabelText('Compliance coverage by standard');

        async function getBarTitles() {
            const standardNames = Object.values(standards).map(([name]) => name);
            const titlesRegex = new RegExp(`${standardNames.join('|')}`);
            const titleElements = await screen.findAllByText(titlesRegex);
            const titles = titleElements.map((elem) => elem.innerHTML);
            // Note that we reverse here because the order in the DOM (bottom->top) is the opposite from
            // how that chart is displayed to the user (top->bottom)
            return titles.reverse();
        }

        // Default is ascending
        const ascendingData = await getBarTitles();
        expect(ascendingData).toHaveLength(6);
        expect(ascendingData).toStrictEqual(
            expect.arrayContaining([
                'CIS Kubernetes v1.5',
                'HIPAA 164',
                'NIST SP 800-190',
                'NIST SP 800-53',
                'PCI DSS 3.2.1',
            ])
        );

        // Sort by descending
        await act(() => user.click(screen.getByLabelText('Options')));
        await act(() => user.click(screen.getByText('Descending')));

        const descendingData = await getBarTitles();
        expect(descendingData).toHaveLength(6);
        expect(descendingData).toStrictEqual(
            expect.arrayContaining([
                'ocp4-cis-node',
                'ocp4-cis',
                'PCI DSS 3.2.1',
                'NIST SP 800-53',
                'NIST SP 800-190',
                'HIPAA 164',
            ])
        );
    });

    it('should visit the correct pages when widget links are clicked', async () => {
        const {
            user,
            utils: { history },
        } = setup();

        // Allow graph to load
        await screen.findByLabelText('Compliance coverage by standard');

        await act(() => user.click(screen.getByText('View all')));
        expect(history.location.pathname).toBe(complianceBasePath);

        const standard = 'CIS Kubernetes v1.5';
        await user.click(await screen.findByText(standard));
        expect(history.location.pathname).toBe(
            `${complianceBasePath}/${urlEntityListTypes[standardEntityTypes.CONTROL]}`
        );
        expect(history.location.search).toBe(
            `?s[Cluster]=%2A&s[standard]=${encodeURIComponent(standard)}`
        );
    });

    it('should contain a button that resets the widget options to default', async () => {
        setup();
        const user = userEvent.setup({ skipHover: true });

        await act(async () => user.click(await screen.findByLabelText('Options')));
        const [asc, desc] = await screen.findAllByRole('button', {
            name: /Ascending|Descending/,
        });

        // Defaults
        expect(asc).toHaveAttribute('aria-pressed', 'true');
        expect(desc).toHaveAttribute('aria-pressed', 'false');

        await act(() => user.click(desc));

        expect(asc).toHaveAttribute('aria-pressed', 'false');
        expect(desc).toHaveAttribute('aria-pressed', 'true');

        const resetButton = await screen.findByLabelText('Revert to default options');
        await act(() => user.click(resetButton));

        expect(asc).toHaveAttribute('aria-pressed', 'true');
        expect(desc).toHaveAttribute('aria-pressed', 'false');
    });
});

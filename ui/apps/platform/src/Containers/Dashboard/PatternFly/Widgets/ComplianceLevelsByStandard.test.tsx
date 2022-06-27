import React from 'react';
import { MockedProvider } from '@apollo/client/testing';
import { screen, within, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/extend-expect';

import renderWithRouter from 'test-utils/renderWithRouter';
import { AGGREGATED_RESULTS_ACROSS_ENTITIES } from 'queries/controls';
import entityTypes, { standardEntityTypes } from 'constants/entityTypes';
import { complianceBasePath, urlEntityListTypes } from 'routePaths';
import ComplianceLevelsByStandard from './ComplianceLevelsByStandard';

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
                    results: [
                        {
                            aggregationKeys: [{ id: 'CIS_Docker_v1_2_0', scope: 'STANDARD' }],
                            numFailing: 0,
                            numPassing: 1,
                            numSkipped: 0,
                            unit: 'CONTROL',
                        },
                        {
                            aggregationKeys: [{ id: 'CIS_Kubernetes_v1_5', scope: 'STANDARD' }],
                            numFailing: 0,
                            numPassing: 0,
                            numSkipped: 40,
                            unit: 'CONTROL',
                        },
                        {
                            aggregationKeys: [{ id: 'HIPAA_164', scope: 'STANDARD' }],
                            numFailing: 8,
                            numPassing: 10,
                            numSkipped: 0,
                            unit: 'CONTROL',
                        },
                        {
                            aggregationKeys: [{ id: 'NIST_800_190', scope: 'STANDARD' }],
                            numFailing: 9,
                            numPassing: 4,
                            numSkipped: 0,
                            unit: 'CONTROL',
                        },
                        {
                            aggregationKeys: [{ id: 'NIST_SP_800_53_Rev_4', scope: 'STANDARD' }],
                            numFailing: 10,
                            numPassing: 10,
                            numSkipped: 2,
                            unit: 'CONTROL',
                        },
                        {
                            aggregationKeys: [{ id: 'PCI_DSS_3_2', scope: 'STANDARD' }],
                            numFailing: 15,
                            numPassing: 8,
                            numSkipped: 1,
                            unit: 'CONTROL',
                        },
                    ],
                },
                complianceStandards: [
                    { id: 'CIS_Docker_v1_2_0', name: 'CIS Docker v1.2.0' },
                    { id: 'CIS_Kubernetes_v1_5', name: 'CIS Kubernetes v1.5' },
                    { id: 'HIPAA_164', name: 'HIPAA 164' },
                    { id: 'NIST_800_190', name: 'NIST SP 800-190' },
                    { id: 'NIST_SP_800_53_Rev_4', name: 'NIST SP 800-53' },
                    { id: 'PCI_DSS_3_2', name: 'PCI DSS 3.2.1' },
                ],
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

const setup = () => {
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
        await screen.findAllByRole('presentation');

        async function getBarPercentages() {
            // eslint-disable-next-line testing-library/no-node-access
            const svgBarsElement = document.querySelector('svg > g:nth-of-type(3)');
            const barPercentages = await within(svgBarsElement as HTMLElement).findAllByText(
                /\d+%/
            );
            expect(barPercentages).toHaveLength(6);
            // Note that we reverse here because the order in the DOM (bottom->top) is the opposite from
            // how that chart is displayed to the user (top->bottom)
            return barPercentages.map((elem) => parseInt(elem.innerHTML, 10)).reverse();
        }

        // Default is ascending
        await waitFor(async () => {
            const ascendingPercentages = await getBarPercentages();
            for (let i = 0; i < ascendingPercentages.length - 1; i += 1) {
                expect(ascendingPercentages[i]).toBeLessThan(ascendingPercentages[i + 1]);
            }
        });

        // Sort by descending
        await user.click(screen.getByRole('button', { name: 'Options' }));
        await user.click(screen.getByRole('button', { name: 'Descending' }));

        await waitFor(async () => {
            const descendingPercentages = await getBarPercentages();
            for (let i = 0; i < descendingPercentages.length - 1; i += 1) {
                expect(descendingPercentages[i]).toBeGreaterThan(descendingPercentages[i + 1]);
            }
        });
    });

    it('should visit the correct pages when widget links are clicked', async () => {
        const {
            user,
            utils: { history },
        } = setup();

        await user.click(await screen.findByRole('link', { name: 'View all' }));
        expect(history.location.pathname).toBe(complianceBasePath);

        const standard = 'CIS Docker v1.2.0';
        await act(async () => {
            await user.click(await screen.findByRole('link', { name: standard }));
        });
        expect(history.location.pathname).toBe(
            `${complianceBasePath}/${urlEntityListTypes[standardEntityTypes.CONTROL]}`
        );
        expect(history.location.search).toBe(
            `?s[Cluster]=%2A&s[standard]=${encodeURIComponent(standard)}`
        );
    });
});

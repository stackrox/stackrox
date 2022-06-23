import React from 'react';
import { MockedProvider } from '@apollo/client/testing';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/extend-expect';

import renderWithRouter from 'test-utils/renderWithRouter';
import ImagesAtMostRisk, { imagesQuery } from './ImagesAtMostRisk';

function makeMockImage(
    id: string,
    remote: string,
    fullName: string,
    priority: number,
    vulnCounter: VulnCounts
) {
    return {
        id,
        name: { remote, fullName },
        priority,
        vulnCounter,
    };
}

const totalImportant = 120;
const fixableImportant = 80;
const totalCritical = 100;
const fixableCritical = 60;

const vulnCounts = {
    important: {
        total: totalImportant,
        fixable: fixableImportant,
    },
    critical: {
        total: totalCritical,
        fixable: fixableCritical,
    },
};

type VulnCounts = typeof vulnCounts;

const mockImages = [1, 2, 3, 4, 5, 6].map((n) =>
    makeMockImage(`${n}`, `name-${n}`, `reg/name-${n}:tag`, n, vulnCounts)
);

const mocks = [
    {
        request: {
            query: imagesQuery,
            variables: {
                query: '',
            },
        },
        result: {
            data: {
                images: mockImages,
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
            <ImagesAtMostRisk />
        </MockedProvider>
    );

    return { user, utils };
}

describe('Images at most risk dashboard widget', () => {
    it('should render the correct title based on selected options', async () => {
        const { user } = setup();

        expect(
            await screen.findByRole('heading', {
                name: 'All images at most risk',
            })
        ).toBeInTheDocument();

        await user.click(await screen.findByRole('button', { name: `Options` }));
        await user.click(await screen.findByRole('button', { name: `Active images` }));

        expect(
            await screen.findByRole('heading', {
                name: 'Active images at most risk',
            })
        ).toBeInTheDocument();
    });

    it('should render the correct text and number of CVEs under each column', async () => {
        const { user } = setup();

        // Default should show fixable CVEs
        expect(await screen.findAllByText(`${fixableCritical} fixable`)).toHaveLength(6);
        expect(await screen.findAllByText(`${fixableImportant} fixable`)).toHaveLength(6);

        // Switch to display all CVEs
        await user.click(await screen.findByRole('button', { name: `Options` }));
        await user.click(await screen.findByRole('button', { name: `All CVEs` }));

        expect(await screen.findAllByText(`${totalCritical} CVEs`)).toHaveLength(6);
        expect(await screen.findAllByText(`${totalImportant} CVEs`)).toHaveLength(6);
    });

    it('should link to the appropriate pages in VulnMgmt', async () => {
        const {
            user,
            utils: { history },
        } = setup();

        await screen.findByRole('heading', { name: 'All images at most risk' });
        await user.click(screen.getByRole('link', { name: 'reg/name-2:tag' }));
        expect(history.location.pathname).toBe('/main/vulnerability-management/image/2');
        expect(history.location.hash).toBe('#image-findings');

        await history.goBack();

        await user.click(screen.getByRole('link', { name: 'View All' }));
        expect(history.location.pathname).toBe('/main/vulnerability-management/images');
    });
});

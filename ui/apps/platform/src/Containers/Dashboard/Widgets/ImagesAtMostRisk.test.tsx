import React from 'react';
import { MockedProvider } from '@apollo/client/testing';
import { screen, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/extend-expect';

import renderWithRouter from 'test-utils/renderWithRouter';
import { vulnManagementImagesPath, vulnManagementPath } from 'routePaths';
import ImagesAtMostRisk, { imagesAtMostRiskQuery } from './ImagesAtMostRisk';

function makeMockImage(
    id: string,
    remote: string,
    fullName: string,
    priority: number,
    imageVulnerabilityCounter: VulnCounts
) {
    return {
        id,
        name: { remote, fullName },
        priority,
        imageVulnerabilityCounter,
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
            query: imagesAtMostRiskQuery,
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

jest.mock('hooks/useResizeObserver');
jest.mock('hooks/useFeatureFlags', () => ({
    __esModule: true,
    default: () => ({
        isFeatureFlagEnabled: jest.fn(),
    }),
}));

beforeEach(() => {
    localStorage.clear();
});

function setup() {
    // Ignore false positive, see: https://github.com/testing-library/eslint-plugin-testing-library/issues/800
    // eslint-disable-next-line testing-library/await-async-events
    const user = userEvent.setup();
    const utils = renderWithRouter(
        <MockedProvider mocks={mocks} addTypename={false}>
            <ImagesAtMostRisk />
        </MockedProvider>
    );

    return { user, utils };
}

// Warning: The current testing environment is not configured to support act(...)
// eslint-disable-next-line jest/no-disabled-tests
describe.skip('Images at most risk dashboard widget', () => {
    it('should render the correct title based on selected options', async () => {
        const { user } = setup();

        // Default is display all images
        expect(await screen.findByText('Images at most risk')).toBeInTheDocument();

        // Change to display only active images
        await act(() => user.click(screen.getByLabelText('Options')));
        await act(() => user.click(screen.getByText('Active images')));

        expect(screen.getByText('Active images at most risk')).toBeInTheDocument();
    });

    it('should render the correct text and number of CVEs under each column', async () => {
        const { user } = setup();

        // Note that in this case the mock data uses the same number of CVEs for every image
        // so we will expect multiple elements matching the below text queries

        // Default should show fixable CVEs
        expect(await screen.findAllByText(`${fixableCritical} fixable`)).toHaveLength(
            mockImages.length
        );
        expect(await screen.findAllByText(`${fixableImportant} fixable`)).toHaveLength(
            mockImages.length
        );

        // Switch to show total CVEs
        await act(() => user.click(screen.getByLabelText('Options')));
        await act(() => user.click(screen.getByText('All CVEs')));

        expect(await screen.findAllByText(`${totalCritical} CVEs`)).toHaveLength(mockImages.length);
        expect(await screen.findAllByText(`${totalImportant} CVEs`)).toHaveLength(
            mockImages.length
        );
    });

    it('should link to the appropriate pages in VulnMgmt', async () => {
        const {
            user,
            utils: { history },
        } = setup();

        await screen.findByText('Images at most risk');
        // Click on the link matching the second image
        const secondImageInList = mockImages[1];
        await act(async () => user.click(await screen.findByText(secondImageInList.name?.remote)));
        expect(history.location.pathname).toBe(
            `${vulnManagementPath}/image/${secondImageInList.id}`
        );
        expect(history.location.hash).toBe('#image-findings');

        await history.goBack();

        await act(() => user.click(screen.getByText('View all')));
        expect(history.location.pathname).toBe(`${vulnManagementImagesPath}`);
    });

    it('should contain a button that resets the widget options to default', async () => {
        setup();
        const user = userEvent.setup({ skipHover: true });

        await act(() => user.click(screen.getByLabelText('Options')));
        const [fixableCves, allCves, activeImages, allImages] = await screen.findAllByRole(
            'button',
            {
                name: /Fixable CVEs|All CVEs|Active images|All images/,
            }
        );

        // Defaults
        expect(fixableCves).toHaveAttribute('aria-pressed', 'true');
        expect(allCves).toHaveAttribute('aria-pressed', 'false');
        expect(activeImages).toHaveAttribute('aria-pressed', 'false');
        expect(allImages).toHaveAttribute('aria-pressed', 'true');

        // Change some options
        await act(() => user.click(allCves));
        await act(() => user.click(activeImages));

        expect(fixableCves).toHaveAttribute('aria-pressed', 'false');
        expect(allCves).toHaveAttribute('aria-pressed', 'true');
        expect(activeImages).toHaveAttribute('aria-pressed', 'true');
        expect(allImages).toHaveAttribute('aria-pressed', 'false');

        const resetButton = await screen.findByLabelText('Revert to default options');
        await act(() => user.click(resetButton));

        expect(fixableCves).toHaveAttribute('aria-pressed', 'true');
        expect(allCves).toHaveAttribute('aria-pressed', 'false');
        expect(activeImages).toHaveAttribute('aria-pressed', 'false');
        expect(allImages).toHaveAttribute('aria-pressed', 'true');
    });
});

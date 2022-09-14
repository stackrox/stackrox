import React from 'react';
import { MockedProvider } from '@apollo/client/testing';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/extend-expect';

import renderWithRouter from 'test-utils/renderWithRouter';
import { mockChartsWithoutAnimation } from 'test-utils/mocks/@patternfly/react-charts';
import AgingImages, { imageCountQuery } from './AgingImages';

jest.setTimeout(10000);

const range0 = '30';
const range1 = '90';
const range2 = '180';
const range3 = '365';

const result0 = 8;
const result1 = 1;
const result2 = 13;
const result3 = 18;

const mocks = [
    {
        request: {
            query: imageCountQuery,
            variables: {
                query0: `Image Created Time:${range0}d-${range1}d`,
                query1: `Image Created Time:${range1}d-${range2}d`,
                query2: `Image Created Time:${range2}d-${range3}d`,
                query3: `Image Created Time:>${range3}d`,
            },
        },
        result: {
            data: {
                timeRange0: result0,
                timeRange1: result1,
                timeRange2: result2,
                timeRange3: result3,
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
    const user = userEvent.setup();
    const utils = renderWithRouter(
        <MockedProvider mocks={mocks} addTypename={false}>
            <AgingImages />
        </MockedProvider>
    );
    return { user, utils };
};

describe('AgingImages dashboard widget', () => {
    it('should render the correct number of images with default settings', async () => {
        setup();

        // When all items are selected, the total should be equal to the total of all buckets
        // returned by the server
        const cardHeading = await screen.findByText(
            `${result0 + result1 + result2 + result3} Aging images`
        );
        expect(cardHeading).toBeInTheDocument();

        // Each bar should display text that is specific to that time bucket, not
        // cumulative.
        expect(await screen.findByText(result0)).toBeInTheDocument();
        expect(await screen.findByText(result1)).toBeInTheDocument();
        expect(await screen.findByText(result2)).toBeInTheDocument();
        expect(await screen.findByText(result3)).toBeInTheDocument();
    });

    it('should render graph bars with the correct image counts when time buckets are toggled', async () => {
        const { user } = setup();

        expect(
            await screen.findByText(`${result0 + result1 + result2 + result3} Aging images`)
        ).toBeInTheDocument();

        await user.click(await screen.findByLabelText('Options'));
        const checkboxes = await screen.findAllByLabelText('Toggle image time range');
        expect(checkboxes).toHaveLength(4);

        // Disable the first bucket
        await user.click(checkboxes[0]);

        // With the first item deselected, aging images < 90 days should no longer be present
        // in the chart or the card header
        expect(
            await screen.findByText(`${result1 + result2 + result3} Aging images`)
        ).toBeInTheDocument();

        // Test values at top of each bar
        expect(() => screen.getByText(result0)).toThrow();
        expect(await screen.findByText(result1)).toBeInTheDocument();
        expect(await screen.findByText(result2)).toBeInTheDocument();
        expect(await screen.findByText(result3)).toBeInTheDocument();

        // Test display of x-axis
        expect(await screen.findByText(`${range1}-${range2} days`)).toBeInTheDocument();
        expect(await screen.findByText(`${range2}-${range3} days`)).toBeInTheDocument();
        expect(await screen.findByText(`>1 year`)).toBeInTheDocument();

        await user.click(checkboxes[0]);
        await user.click(checkboxes[2]);

        // With the first item re-selected (regardless of the other selected items), the heading total
        // should revert to the original value.
        expect(
            await screen.findByText(`${result0 + result1 + result2 + result3} Aging images`)
        ).toBeInTheDocument();

        expect(await screen.findByText(result0)).toBeInTheDocument();
        // The second bar in the chart should now contain values from the second and third buckets
        expect(await screen.findByText(result1 + result2)).toBeInTheDocument();
        expect(() => screen.getByText(result2)).toThrow();
        expect(await screen.findByText(result3)).toBeInTheDocument();

        // Test display of x-axis
        expect(await screen.findByText(`${range0}-${range1} days`)).toBeInTheDocument();
        expect(await screen.findByText(`${range1}-${range3} days`)).toBeInTheDocument();
        expect(await screen.findByText(`>1 year`)).toBeInTheDocument();
    });

    it('links users to the correct filtered image list', async () => {
        const {
            user,
            utils: { history },
        } = setup();

        // Check default links
        await user.click(await screen.findByText(`30-90 days`));
        expect(history.location.search).toContain('s[Image Created Time]=30d-90d');

        await user.click(await screen.findByText('90-180 days'));
        expect(history.location.search).toContain('s[Image Created Time]=90d-180d');

        await user.click(await screen.findByText('>1 year'));
        expect(history.location.search).toContain('s[Image Created Time]=>365d');

        // Deselect the second time range, merging the first and second time buckets
        await user.click(await screen.findByLabelText('Options'));
        const checkboxes = await screen.findAllByLabelText('Toggle image time range');
        await user.click(checkboxes[1]);
        await user.click(await screen.findByLabelText('Options'));

        await user.click(await screen.findByText('30-180 days'));
        expect(history.location.search).toContain('s[Image Created Time]=30d-180d');
    });

    it('should contain a button that resets the widget options to default', async () => {
        setup();
        const user = userEvent.setup({ skipHover: true });

        await user.click(await screen.findByLabelText('Options'));
        const checkboxes = await screen.findAllByLabelText('Toggle image time range');
        const inputs = (await screen.findAllByLabelText('Image age in days')) as HTMLInputElement[];

        // Defaults
        checkboxes.forEach((cb) => expect(cb).toBeChecked());
        expect(inputs.map(({ value }) => parseInt(value, 10))).toEqual(
            expect.arrayContaining([30, 90, 180, 365])
        );

        await user.click(checkboxes[0]);
        await user.click(checkboxes[1]);
        // Double clicking allows us to select the current input value and type over it
        await user.dblClick(inputs[1]);
        await user.type(inputs[1], '100', { skipClick: true });
        await user.dblClick(inputs[2]);
        await user.type(inputs[2], '200', { skipClick: true });

        expect(checkboxes[0]).not.toBeChecked();
        expect(checkboxes[1]).not.toBeChecked();
        expect(checkboxes[2]).toBeChecked();
        expect(checkboxes[3]).toBeChecked();
        expect(inputs.map(({ value }) => parseInt(value, 10))).toEqual(
            expect.arrayContaining([30, 100, 200, 365])
        );

        const resetButton = await screen.findByLabelText('Revert to default options');
        await user.click(resetButton);

        checkboxes.forEach((cb) => expect(cb).toBeChecked());
        expect(inputs.map(({ value }) => parseInt(value, 10))).toEqual(
            expect.arrayContaining([30, 90, 180, 365])
        );
    });
});

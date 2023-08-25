import React from 'react';
import * as PFReactCharts from '@patternfly/react-charts';

const { Chart, ...rest } = jest.requireActual('@patternfly/react-charts');

/**
 * Overrides props to the `Chart` component and globally disables animation. This is to avoid asynchronous
 * state updates that result in `act()` errors, as well as for a performance boost.
 */
// eslint-disable-next-line import/prefer-default-export
export const mockChartsWithoutAnimation = {
    ...rest,
    Chart: (props) => <Chart {...props} animate={undefined} />,
} as jest.Mock<typeof PFReactCharts>;

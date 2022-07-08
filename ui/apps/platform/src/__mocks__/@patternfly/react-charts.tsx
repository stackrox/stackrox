import React from 'react';

const { Chart, ...rest } = jest.requireActual('@patternfly/react-charts');

module.exports = {
    ...rest,
    // Disables animation on Charts during testing. Enabling animation on charts causes React.setState calls
    // to fire asynchronously while the animation updates, often causing a `setState` call to happen outside
    // of React's `act()` call, which causes intermittent errors during testing.
    Chart: (props) => <Chart {...props} animate={undefined} />,
};

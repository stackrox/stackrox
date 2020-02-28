import React from 'react';
import { render } from '@testing-library/react';
import DeploymentEventTimeline from './DeploymentEventTimeline';

test('Should display total events across pods in a deployment', async () => {
    const deploymentId = '5039c79f-5294-11ea-84f7-025000000001';
    const { findByText } = render(<DeploymentEventTimeline deploymentId={deploymentId} />);

    findByText('12 EVENTS ACROSS 3 PODS');
});

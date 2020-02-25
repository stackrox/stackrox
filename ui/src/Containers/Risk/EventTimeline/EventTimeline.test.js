import React from 'react';
import { render, fireEvent } from '@testing-library/react';
import EventTimeline from './EventTimeline';

test('Shows the Event Timeline Overview Information', async () => {
    const deploymentId = '5039c79f-5294-11ea-84f7-025000000001';
    const { getByText, findByText } = render(<EventTimeline deploymentId={deploymentId} />);

    getByText('12 EVENTS');
    getByText('3');
    getByText('Policy Violations');
    getByText('5');
    getByText('Process Activities');
    getByText('4');
    getByText('Restarts / Failures');

    // open the modal
    fireEvent.click(getByText('View Graph'));

    findByText(`Show Event Timeline Graph for Deployment: ${deploymentId}`);
});

import React from 'react';
import { render, fireEvent, waitForElement } from '@testing-library/react';
import EventTimeline from './EventTimeline';

test('Should be able to switch between the Deployment and Pod Event Timeline', async () => {
    const id = '5039c79f-5294-11ea-84f7-025000000001';
    const { findByText, getByTestId, getAllByTestId } = render(<EventTimeline deploymentId={id} />);

    findByText('12 EVENTS ACROSS 3 PODS');

    const buttonExpander = getAllByTestId('timeline-name-list-item-expander');

    fireEvent.click(buttonExpander[0]);

    await waitForElement(() => findByText('Pod with Container Events'));

    const backButton = getByTestId('timeline-back-button');

    fireEvent.click(backButton);

    findByText('12 EVENTS ACROSS 3 PODS');
});

import React from 'react';
import { render, screen } from '@testing-library/react';

import EventsRow from './EventsRow';

const MockedEventsRow = () => {
    const entityName = 'Test';
    // we want to create 10 events between 0-9000ms equally spaced by 1000ms
    const events = Array.from(Array(10).keys()).map((index) => {
        return {
            id: `event-${index}`,
            name: `event-${index}`,
            type: 'PolicyViolationEvent',
            timestamp: '2020-04-20T20:20:20.358227916Z',
            differenceInMilliseconds: 1000 * index,
        };
    });
    const height = 100;
    const width = 500;
    return (
        <svg data-testid="timeline-main-view" width={width} height={height}>
            <EventsRow
                entityName={entityName}
                events={events}
                height={height}
                width={width}
                translateX={0}
                translateY={0}
                minTimeRange={0}
                // we want the window to be between 0-4500ms
                maxTimeRange={4500}
            />
        </svg>
    );
};

test('should only render events in the view', () => {
    render(<MockedEventsRow />);
    const elements = screen.getAllByTestId('timeline-event-marker');
    // we should only see 5 events between 0-4500ms
    expect(elements.length).toBe(5);
});

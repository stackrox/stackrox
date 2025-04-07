import React from 'react';
import { Tooltip } from '@patternfly/react-core';

import Button from 'Components/Button';
import PolicyViolationEvent from 'Components/TimelineGraph/EventsGraph/EventMarker/PolicyViolationEvent';
import ProcessActivityEvent from 'Components/TimelineGraph/EventsGraph/EventMarker/ProcessActivityEvent';
import RestartEvent from 'Components/TimelineGraph/EventsGraph/EventMarker/RestartEvent';
import TerminationEvent from 'Components/TimelineGraph/EventsGraph/EventMarker/TerminationEvent';

const ICON_SIZE = 15;

const TimelineLegend = () => {
    const content = (
        <div data-testid="timeline-legend-items">
            <div className="flex items-center mb-2">
                <ProcessActivityEvent size={ICON_SIZE} />
                <span className="ml-2">Process Activity</span>
            </div>
            <div className="flex items-center mb-2">
                <PolicyViolationEvent size={ICON_SIZE} />
                <span className="ml-2">Process Activity with Violation</span>
            </div>
            <div className="flex items-center mb-2">
                <ProcessActivityEvent size={ICON_SIZE} inBaseline />
                <span className="ml-2">Baseline Process Activity</span>
            </div>
            <div className="flex items-center mb-2">
                <RestartEvent size={ICON_SIZE} />
                <span className="ml-2">Container Restart</span>
            </div>
            <div className="flex items-center">
                <TerminationEvent size={ICON_SIZE} />
                <span className="ml-2">Container Termination</span>
            </div>
        </div>
    );
    return (
        <Tooltip content={content}>
            <div>
                <Button className="btn btn-base" dataTestId="timeline-legend" text="Show Legend" />
            </div>
        </Tooltip>
    );
};

export default TimelineLegend;

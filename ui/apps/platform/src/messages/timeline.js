import { graphTypes, eventTypes } from 'constants/timelineTypes';

export const graphLabels = {
    [graphTypes.POD]: 'Pod',
    [graphTypes.CONTAINER]: 'Container',
};

export const eventLabels = {
    [eventTypes.POLICY_VIOLATION]: 'Policy Violation',
    [eventTypes.PROCESS_ACTIVITY]: 'Process Activity',
    [eventTypes.RESTART]: 'Container Restart',
    [eventTypes.TERMINATION]: 'Container Termination',
};

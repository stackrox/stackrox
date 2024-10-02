import React from 'react';
import { Switch } from '@patternfly/react-core';

export type MyJobsFilterProps = {
    isViewingOnlyMyJobs: boolean;
    onMyJobsFilterChange: (checked: boolean) => void;
};

function MyJobsFilter({ isViewingOnlyMyJobs, onMyJobsFilterChange }: MyJobsFilterProps) {
    // We're using the same label for both "label" and "labelOff" because changing the label between "on" and "off" states was causing confusion.
    // When the label changes (e.g., from "View only my jobs" to "View all jobs"), users found it unclear what state the switch was in and what they were actually viewing.
    // By keeping the label consistent, it avoids this confusion and maintains clarity on what the switch controls.
    return (
        <Switch
            id="view-only-my-jobs"
            label="View only my jobs"
            labelOff="View only my jobs"
            isChecked={isViewingOnlyMyJobs}
            onChange={(_event, checked: boolean) => onMyJobsFilterChange(checked)}
        />
    );
}

export default MyJobsFilter;

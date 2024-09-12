import React from 'react';
import { Switch } from '@patternfly/react-core';

export type MyJobsFilterProps = {
    showOnlyMyJobs: boolean;
    onToggle: (checked: boolean) => void;
};

function MyJobsFilter({ showOnlyMyJobs, onToggle }: MyJobsFilterProps) {
    return (
        <Switch
            id="view-only-my-jobs"
            label="View only my jobs"
            labelOff="View only my jobs"
            isChecked={showOnlyMyJobs}
            onChange={(_event, checked: boolean) => onToggle(checked)}
        />
    );
}

export default MyJobsFilter;

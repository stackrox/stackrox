import React from 'react';
import { SelectOption } from '@patternfly/react-core';

import SelectSingle from 'Components/SelectSingle/SelectSingle';

export const edgeStates = ['active', 'inactive'] as const;

export type EdgeState = (typeof edgeStates)[number];

type EdgeStateSelectProps = {
    edgeState: EdgeState;
    setEdgeState: (state: EdgeState) => void;
    isDisabled: boolean;
};

function EdgeStateSelect({ edgeState, setEdgeState, isDisabled }: EdgeStateSelectProps) {
    function handleSelect(_name: string, value: string) {
        setEdgeState(value as EdgeState);
    }

    return (
        <SelectSingle
            id="edge-state-select"
            value={edgeState}
            handleSelect={handleSelect}
            isDisabled={isDisabled}
        >
            <SelectOption
                value="active"
                description="Flows where traffic has been observed during your selected time window."
            >
                Active flows
            </SelectOption>
            <SelectOption
                value="inactive"
                description="Possible flows allowed by your Kubernetes network policies, although they carried no traffic in your selected time window.  In a well-isolated implementation, this view will be empty."
            >
                Inactive flows
            </SelectOption>
        </SelectSingle>
    );
}

export default EdgeStateSelect;

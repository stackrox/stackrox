import React from 'react';
import { Select, SelectOption } from '@patternfly/react-core';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';

export type EdgeState = 'active' | 'inactive';

type EdgeStateSelectProps = {
    edgeState: EdgeState;
    setEdgeState: (state) => void;
    isDisabled: boolean;
};

function EdgeStateSelect({ edgeState, setEdgeState, isDisabled }: EdgeStateSelectProps) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();

    function onSelect(_event, selection) {
        closeSelect();
        setEdgeState(selection);
    }

    return (
        <Select
            variant="single"
            isOpen={isOpen}
            onToggle={onToggle}
            onSelect={onSelect}
            selections={edgeState}
            isDisabled={isDisabled}
            id="edge-state-select"
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
        </Select>
    );
}

export default EdgeStateSelect;

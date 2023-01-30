import React, { useState } from 'react';
import { Select, SelectOption } from '@patternfly/react-core';

export type EdgeState = 'active' | 'extraneous';
type EdgeStateSelectProps = {
    edgeState: EdgeState;
    setEdgeState: (state) => void;
    isDisabled: boolean;
};

function EdgeStateSelect({ edgeState, setEdgeState, isDisabled }: EdgeStateSelectProps) {
    const [isOpen, setIsOpen] = useState(false);

    function onToggle() {
        setIsOpen(!isOpen);
    }

    function onSelect() {
        const newEdgeState = edgeState === 'active' ? 'extraneous' : 'active';
        setEdgeState(newEdgeState);
    }

    return (
        <Select
            variant="single"
            isOpen={isOpen}
            onToggle={onToggle}
            onSelect={onSelect}
            selections={edgeState}
            isDisabled={isDisabled}
        >
            <SelectOption
                value="active"
                description="Traffic observed in your selected time window."
            >
                Active traffic
            </SelectOption>
            <SelectOption
                value="extraneous"
                description="Inactive flows allowed by your network policies in your selected time window."
            >
                Extraneous flows
            </SelectOption>
        </Select>
    );
}

export default EdgeStateSelect;

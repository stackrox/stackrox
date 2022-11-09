import React, { useState } from 'react';
import { Select, SelectOption } from '@patternfly/react-core';

export type FlowsState = 'active' | 'extraneous';
type FlowsSelectProps = {
    flowsState: FlowsState;
    setFlowsState: (state) => void;
};

function FlowsSelect({ flowsState, setFlowsState }: FlowsSelectProps) {
    const [isOpen, setIsOpen] = useState(false);

    function onToggle() {
        setIsOpen(!isOpen);
    }

    function onSelect() {
        const newFlowsState = flowsState === 'active' ? 'extraneous' : 'active';
        setFlowsState(newFlowsState);
    }

    return (
        <Select
            variant="single"
            isOpen={isOpen}
            onToggle={onToggle}
            onSelect={onSelect}
            selections={flowsState}
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

export default FlowsSelect;

import React, { useState } from 'react';
import {
    Flex,
    Icon,
    MenuToggleElement,
    MenuToggle,
    Select,
    SelectList,
    SelectOption,
} from '@patternfly/react-core';
import { CubesIcon } from '@patternfly/react-icons';

import { FilteredWorkflowState, SetFilteredWorkflowState } from './types';

const width = '330px';

export type FilteredWorkflowSelectorProps = {
    filteredWorkflowState: FilteredWorkflowState;
    setFilteredWorkflowState: SetFilteredWorkflowState;
};

function FilteredWorkflowSelector({
    filteredWorkflowState,
    setFilteredWorkflowState,
}: FilteredWorkflowSelectorProps) {
    const [isSelectOpen, setIsSelectOpen] = useState(false);

    return (
        <Select
            isOpen={isSelectOpen}
            selected={filteredWorkflowState}
            onSelect={(_, value) => {
                setFilteredWorkflowState(value);
                setIsSelectOpen(false);
            }}
            onOpenChange={(isOpen) => setIsSelectOpen(isOpen)}
            toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                <MenuToggle
                    style={{ width }}
                    aria-label="Filtered workflow select"
                    ref={toggleRef}
                    onClick={() => setIsSelectOpen(!isSelectOpen)}
                    isExpanded={isSelectOpen}
                >
                    <Flex
                        spaceItems={{ default: 'spaceItemsSm' }}
                        alignItems={{ default: 'alignItemsCenter' }}
                    >
                        <Icon>{<CubesIcon />}</Icon>
                        <span>{filteredWorkflowState}</span>
                    </Flex>
                </MenuToggle>
            )}
            shouldFocusToggleOnSelect
        >
            <SelectList style={{ width }}>
                <SelectOption
                    value="Application view"
                    description="Display findings for application workloads."
                    isDisabled
                >
                    Application view
                </SelectOption>
                <SelectOption
                    value="Platform view"
                    description="Display findings for platform components in OpenShift and layered services."
                    isDisabled
                >
                    Platform view
                </SelectOption>
                <SelectOption
                    value="Full view"
                    description="Display findings for application workloads and platform components simultaneously."
                >
                    Full view
                </SelectOption>
            </SelectList>
        </Select>
    );
}

export default FilteredWorkflowSelector;

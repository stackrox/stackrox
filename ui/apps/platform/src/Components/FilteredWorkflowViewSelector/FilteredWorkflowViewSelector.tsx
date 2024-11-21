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

import { FilteredWorkflowView } from './types';

const width = '330px';

export type FilteredWorkflowViewSelectorProps = {
    filteredWorkflowView: FilteredWorkflowView;
    onChangeFilteredWorkflowView: (value: string | number | undefined) => void;
};

function FilteredWorkflowViewSelector({
    filteredWorkflowView,
    onChangeFilteredWorkflowView,
}: FilteredWorkflowViewSelectorProps) {
    const [isSelectOpen, setIsSelectOpen] = useState(false);

    return (
        <Select
            isOpen={isSelectOpen}
            selected={filteredWorkflowView}
            onSelect={(_, value) => {
                onChangeFilteredWorkflowView(value);
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
                        <span>{filteredWorkflowView}</span>
                    </Flex>
                </MenuToggle>
            )}
            shouldFocusToggleOnSelect
        >
            <SelectList style={{ width }} aria-label="Filtered workflow select options">
                <SelectOption
                    value="Applications view"
                    description="Display findings for application workloads."
                >
                    Applications view
                </SelectOption>
                <SelectOption
                    value="Platform view"
                    description="Display findings for platform components in OpenShift."
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

export default FilteredWorkflowViewSelector;

import React from 'react';
import {
    Select,
    SelectOption,
    SelectGroup,
    MenuToggle,
    MenuToggleElement,
    SelectList,
    Flex,
    FlexItem,
} from '@patternfly/react-core';

import TypeaheadCheckboxSelect from 'Components/PatternFly/TypeaheadCheckboxSelect';
import { AdvancedFlowsFilterType } from './types';
import { filtersToSelections, selectionsToFilters } from './advancedFlowsFilterUtils';

export type AdvancedFlowsFilterProps = {
    filters: AdvancedFlowsFilterType;
    setFilters: React.Dispatch<React.SetStateAction<AdvancedFlowsFilterType>>;
    allUniquePorts: string[];
};

export const defaultAdvancedFlowsFilters: AdvancedFlowsFilterType = {
    directionality: [],
    protocols: [],
    ports: [],
};

function AdvancedFlowsFilter({
    filters,
    setFilters,
    allUniquePorts,
}: AdvancedFlowsFilterProps): React.ReactElement {
    // derived state
    const selections = filtersToSelections(filters);

    // component state
    const [isFilterDropdownOpen, setIsFilterDropdownOpen] = React.useState(false);

    // setters
    const onTrafficFilterSelect = (
        _: React.MouseEvent | undefined,
        selection: string | number | undefined
    ) => {
        if (!selection || typeof selection !== 'string') {
            return;
        }

        // Handle directionality and protocol selections
        if (selections.includes(selection)) {
            setFilters((prevFilters) => {
                const prevSelection = filtersToSelections(prevFilters);
                const newSelection = prevSelection.filter((item) => item !== selection);
                const newFilters = selectionsToFilters(newSelection);
                return newFilters;
            });
        } else {
            setFilters((prevFilters) => {
                const prevSelection = filtersToSelections(prevFilters);
                const newSelection = [...prevSelection, selection] as string[];
                const newFilters = selectionsToFilters(newSelection);
                return newFilters;
            });
        }
    };

    const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            onClick={() => setIsFilterDropdownOpen(!isFilterDropdownOpen)}
            isExpanded={isFilterDropdownOpen}
            aria-label="Advanced flows filter"
            className="advanced-flows-filters-select"
        >
            Advanced
        </MenuToggle>
    );

    const onPortsChange = (newPorts: string[]) => {
        setFilters((prevFilters) => ({
            ...prevFilters,
            ports: newPorts,
        }));
    };

    return (
        <Flex spaceItems={{ default: 'spaceItemsSm' }} flexWrap={{ default: 'nowrap' }}>
            <FlexItem>
                <Select
                    className="advanced-flows-filters-select"
                    isOpen={isFilterDropdownOpen}
                    selected={selections}
                    onSelect={onTrafficFilterSelect}
                    onOpenChange={(nextOpen: boolean) => setIsFilterDropdownOpen(nextOpen)}
                    toggle={toggle}
                    popperProps={{
                        direction: 'down',
                        position: 'right',
                    }}
                >
                    <SelectList style={{ minWidth: '200px' }}>
                        <SelectGroup label="Flow directionality">
                            <SelectOption
                                hasCheckbox
                                value="ingress"
                                isSelected={selections.includes('ingress')}
                            >
                                Ingress (inbound)
                            </SelectOption>
                            <SelectOption
                                hasCheckbox
                                value="egress"
                                isSelected={selections.includes('egress')}
                            >
                                Egress (outbound)
                            </SelectOption>
                        </SelectGroup>
                        <SelectGroup label="Protocols">
                            <SelectOption
                                hasCheckbox
                                value="L4_PROTOCOL_TCP"
                                isSelected={selections.includes('L4_PROTOCOL_TCP')}
                            >
                                TCP
                            </SelectOption>
                            <SelectOption
                                hasCheckbox
                                value="L4_PROTOCOL_UDP"
                                isSelected={selections.includes('L4_PROTOCOL_UDP')}
                            >
                                UDP
                            </SelectOption>
                        </SelectGroup>
                    </SelectList>
                </Select>
            </FlexItem>
            <FlexItem>
                <TypeaheadCheckboxSelect
                    id="ports-filter-select"
                    selections={filters.ports}
                    onChange={onPortsChange}
                    options={allUniquePorts.map((port) => ({ value: port }))}
                    placeholder="Filter by ports..."
                    toggleAriaLabel="Ports filter"
                />
            </FlexItem>
        </Flex>
    );
}

export default AdvancedFlowsFilter;

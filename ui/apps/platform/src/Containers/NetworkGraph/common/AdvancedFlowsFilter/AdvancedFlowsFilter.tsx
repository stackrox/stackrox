import React from 'react';
import {
    Select,
    SelectOption,
    SelectGroup,
    MenuToggle,
    MenuToggleElement,
    SelectList,
} from '@patternfly/react-core';

import useMultiSelect from 'hooks/useMultiSelect';
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
    const {
        isOpen: isPortsSelectOpen,
        onToggle: onTogglePortsSelect,
        onSelect: onSelectPorts,
    } = useMultiSelect(handlePortsSelect, filters.ports, false);

    // setters
    const onTrafficFilterSelect = (_: React.MouseEvent | undefined, selection: string | undefined) => {
        if (!selection) return;
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
    function handlePortsSelect(selection) {
        setFilters((prevFilters) => {
            const newFilters = { ...prevFilters };
            newFilters.ports = selection;
            return newFilters;
        });
    }

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

    return (
        <Select
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
            <SelectList>
                <SelectGroup label="Flow directionality">
                    <SelectOption hasCheckbox value="ingress" isSelected={selections.includes('ingress')}>
                        Ingress (inbound)
                    </SelectOption>
                    <SelectOption hasCheckbox value="egress" isSelected={selections.includes('egress')}>
                        Egress (outbound)
                    </SelectOption>
                </SelectGroup>
                <SelectGroup label="Protocols">
                    <SelectOption hasCheckbox value="L4_PROTOCOL_TCP" isSelected={selections.includes('L4_PROTOCOL_TCP')}>
                        TCP
                    </SelectOption>
                    <SelectOption hasCheckbox value="L4_PROTOCOL_UDP" isSelected={selections.includes('L4_PROTOCOL_UDP')}>
                        UDP
                    </SelectOption>
                </SelectGroup>
                <SelectGroup label="Ports">
                    <div className="pf-v5-u-px-md">
                        <Select
                            isOpen={isPortsSelectOpen}
                            selected={filters.ports}
                            onSelect={onSelectPorts}
                            onOpenChange={(nextOpen: boolean) => onTogglePortsSelect()}
                            toggle={(toggleRef) => (
                                <MenuToggle
                                    ref={toggleRef}
                                    onClick={() => onTogglePortsSelect()}
                                    isExpanded={isPortsSelectOpen}
                                    aria-label="Select ports"
                                    variant="typeahead"
                                >
                                    {filters.ports.length > 0 ? `${filters.ports.length} selected` : 'Select ports'}
                                </MenuToggle>
                            )}
                            popperProps={{
                                appendTo: () => document.body,
                            }}
                        >
                            <SelectList>
                                {allUniquePorts.map((port) => {
                                    return (
                                        <SelectOption hasCheckbox value={port} key={port} isSelected={filters.ports.includes(port)}>
                                            {port}
                                        </SelectOption>
                                    );
                                })}
                            </SelectList>
                        </Select>
                    </div>
                </SelectGroup>
            </SelectList>
        </Select>
    );
}

export default AdvancedFlowsFilter;

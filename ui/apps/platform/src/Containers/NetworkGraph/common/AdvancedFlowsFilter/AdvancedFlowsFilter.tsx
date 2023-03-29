import React from 'react';
import {
    Select,
    SelectGroup,
    SelectOption,
    SelectPosition,
    SelectVariant,
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
    const onFilterDropdownToggle = (isOpen: boolean) => {
        setIsFilterDropdownOpen(isOpen);
    };
    const onTrafficFilterSelect = (_, selection) => {
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

    return (
        <Select
            className="advanced-flows-filters-select"
            variant={SelectVariant.checkbox}
            onToggle={onFilterDropdownToggle}
            onSelect={onTrafficFilterSelect}
            selections={selections}
            isOpen={isFilterDropdownOpen}
            placeholderText="Advanced"
            aria-labelledby="advanced-flows-filters-select"
            isGrouped
            position={SelectPosition.right}
        >
            <SelectGroup label="Flow directionality">
                <SelectOption value="ingress">Ingress (inbound)</SelectOption>
                <SelectOption value="egress">Egress (outbound)</SelectOption>
            </SelectGroup>
            <SelectGroup label="Protocols">
                <SelectOption value="L4_PROTOCOL_TCP">TCP</SelectOption>
                <SelectOption value="L4_PROTOCOL_UDP">UDP</SelectOption>
            </SelectGroup>
            <SelectGroup label="Ports">
                <Select
                    className="pf-u-px-md"
                    variant={SelectVariant.typeaheadMulti}
                    toggleAriaLabel="Select ports"
                    onToggle={onTogglePortsSelect}
                    onSelect={onSelectPorts}
                    selections={filters.ports}
                    isOpen={isPortsSelectOpen}
                    placeholderText="Select ports"
                    menuAppendTo="parent"
                >
                    {allUniquePorts.map((port) => {
                        return (
                            <SelectOption value={port} key={port}>
                                {port}
                            </SelectOption>
                        );
                    })}
                </Select>
            </SelectGroup>
        </Select>
    );
}

export default AdvancedFlowsFilter;

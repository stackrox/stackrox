import { useState } from 'react';
import type { Dispatch, FormEvent, MouseEvent, ReactElement, SetStateAction } from 'react';
import {
    Badge,
    Divider,
    Flex,
    FlexItem,
    MenuToggle,
    Select,
    SelectGroup,
    SelectList,
    SelectOption,
    TextInputGroup,
    TextInputGroupMain,
} from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';

import type { AdvancedFlowsFilterType } from './types';
import { filtersToSelections, selectionsToFilters } from './advancedFlowsFilterUtils';
import { toggleItemInArray } from 'utils/arrayUtils';

export type AdvancedFlowsFilterProps = {
    filters: AdvancedFlowsFilterType;
    setFilters: Dispatch<SetStateAction<AdvancedFlowsFilterType>>;
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
}: AdvancedFlowsFilterProps): ReactElement {
    // derived state
    const selections = filtersToSelections(filters);

    // component state
    const [isFilterDropdownOpen, setIsFilterDropdownOpen] = useState(false);
    const [portsFilterValue, setPortsFilterValue] = useState('');

    // Calculate total filter count for badge
    const totalFilterCount =
        filters.directionality.length + filters.protocols.length + filters.ports.length;

    // setters
    const onFilterDropdownToggle = () => {
        setIsFilterDropdownOpen((prev) => !prev);
    };

    const onSelect = (
        _event: MouseEvent<Element, globalThis.MouseEvent> | undefined,
        selection: string | number | undefined
    ) => {
        const value = String(selection);

        // Handle port selection
        if (allUniquePorts.includes(value)) {
            setFilters((prevFilters) => {
                const newPorts = toggleItemInArray(prevFilters.ports, value);
                return { ...prevFilters, ports: newPorts };
            });
            setPortsFilterValue('');
            return;
        }

        // Handle traffic filter selection (directionality and protocols)
        if (selections.includes(value)) {
            setFilters((prevFilters) => {
                const prevSelection = filtersToSelections(prevFilters);
                const newSelection = prevSelection.filter((item) => item !== value);
                const newFilters = selectionsToFilters(newSelection);
                return newFilters;
            });
        } else {
            setFilters((prevFilters) => {
                const prevSelection = filtersToSelections(prevFilters);
                const newSelection = [...prevSelection, value] as string[];
                const newFilters = selectionsToFilters(newSelection);
                return newFilters;
            });
        }
    };

    const onPortsFilterChange = (_event: FormEvent<HTMLInputElement>, value: string) => {
        setPortsFilterValue(value);
    };

    // Filter and sort ports based on search input
    const filtered = allUniquePorts.filter((port) =>
        port.toLowerCase().includes(portsFilterValue.toLowerCase())
    );
    const filteredPorts = filtered.sort((a, b) => parseInt(a, 10) - parseInt(b, 10));

    return (
        <Select
            isOpen={isFilterDropdownOpen}
            onOpenChange={setIsFilterDropdownOpen}
            onSelect={onSelect}
            toggle={(toggleRef) => (
                <MenuToggle
                    ref={toggleRef}
                    onClick={onFilterDropdownToggle}
                    isExpanded={isFilterDropdownOpen}
                    className="advanced-flows-filters-select"
                >
                    <Flex
                        alignItems={{ default: 'alignItemsCenter' }}
                        spaceItems={{ default: 'spaceItemsSm' }}
                    >
                        <FlexItem>Advanced</FlexItem>
                        {totalFilterCount > 0 && <Badge isRead>{totalFilterCount}</Badge>}
                    </Flex>
                </MenuToggle>
            )}
            popperProps={{ position: 'right' }}
        >
            <SelectList style={{ maxHeight: '50vh', overflow: 'auto' }}>
                <SelectGroup label="Flow directionality">
                    <SelectOption
                        value="ingress"
                        hasCheckbox
                        isSelected={selections.includes('ingress')}
                    >
                        Ingress (inbound)
                    </SelectOption>
                    <SelectOption
                        value="egress"
                        hasCheckbox
                        isSelected={selections.includes('egress')}
                    >
                        Egress (outbound)
                    </SelectOption>
                </SelectGroup>
                <SelectGroup label="Protocols">
                    <SelectOption
                        value="L4_PROTOCOL_TCP"
                        hasCheckbox
                        isSelected={selections.includes('L4_PROTOCOL_TCP')}
                    >
                        TCP
                    </SelectOption>
                    <SelectOption
                        value="L4_PROTOCOL_UDP"
                        hasCheckbox
                        isSelected={selections.includes('L4_PROTOCOL_UDP')}
                    >
                        UDP
                    </SelectOption>
                </SelectGroup>
                <SelectGroup label="Ports">
                    <div className="pf-v6-u-p-md">
                        <TextInputGroup>
                            <TextInputGroupMain
                                value={portsFilterValue}
                                onChange={onPortsFilterChange}
                                placeholder="Filter by port"
                                aria-label="Filter ports"
                                icon={<SearchIcon />}
                            />
                        </TextInputGroup>
                    </div>
                    {filteredPorts.length > 0 && <Divider />}
                    {filteredPorts.length > 0 ? (
                        filteredPorts.map((port) => (
                            <SelectOption
                                key={port}
                                value={port}
                                hasCheckbox
                                isSelected={filters.ports.includes(port)}
                            >
                                {port}
                            </SelectOption>
                        ))
                    ) : (
                        <SelectOption isDisabled>No matching ports</SelectOption>
                    )}
                </SelectGroup>
            </SelectList>
        </Select>
    );
}

export default AdvancedFlowsFilter;

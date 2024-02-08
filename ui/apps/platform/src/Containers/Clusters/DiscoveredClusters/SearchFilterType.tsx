import React, { useState } from 'react';
import { Divider, Select, SelectOption } from '@patternfly/react-core';

import { DiscoveredClusterType, isType, types } from 'services/DiscoveredClusterService';

import { getTypeText } from './DiscoveredCluster';

const optionAll = 'All_types';

type SearchFilterTypeProps = {
    typesSelected: DiscoveredClusterType[] | undefined;
    isDisabled: boolean;
    setTypesSelected: (types: DiscoveredClusterType[] | undefined) => void;
};

// TODO for multiselect rename as SearchFilterTypes (that is, plural).
function SearchFilterType({ typesSelected, isDisabled, setTypesSelected }: SearchFilterTypeProps) {
    const [isOpen, setIsOpen] = useState(false);

    function onSelect(_event, selection) {
        if (isType(selection)) {
            // TODO for multiselect, replace set with either spread in or filter out.
            setTypesSelected([selection]);
        } else {
            setTypesSelected(undefined);
        }
        setIsOpen(false);
    }

    const options = types.map((type) => (
        <SelectOption key={type} value={type}>
            {getTypeText(type)}
        </SelectOption>
    ));
    options.push(
        <Divider key="Divider" />,
        <SelectOption key="All" value={optionAll} isPlaceholder>
            All types
        </SelectOption>
    );

    // TODO replace single with checkbox for multiple selections.
    return (
        <Select
            variant="single"
            aria-label="Type filter menu items"
            toggleAriaLabel="Type filter menu toggle"
            onToggle={setIsOpen}
            onSelect={onSelect}
            selections={
                Array.isArray(typesSelected) && typesSelected.length !== 0
                    ? typesSelected[0]
                    : optionAll
            }
            isDisabled={isDisabled}
            isOpen={isOpen}
        >
            {options}
        </Select>
    );
}

export default SearchFilterType;

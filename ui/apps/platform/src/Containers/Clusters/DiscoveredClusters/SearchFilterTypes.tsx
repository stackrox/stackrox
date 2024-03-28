import React, { useState } from 'react';
import { Divider } from '@patternfly/react-core';
import { Select, SelectOption } from '@patternfly/react-core/deprecated';

import { DiscoveredClusterType, isType, types } from 'services/DiscoveredClusterService';

import { getTypeText } from './DiscoveredCluster';

const optionAll = 'All_types';

type SearchFilterTypesProps = {
    typesSelected: DiscoveredClusterType[] | undefined;
    isDisabled: boolean;
    setTypesSelected: (types: DiscoveredClusterType[] | undefined) => void;
};

function SearchFilterTypes({
    typesSelected,
    isDisabled,
    setTypesSelected,
}: SearchFilterTypesProps) {
    const [isOpen, setIsOpen] = useState(false);

    function onSelect(_event, selection) {
        const previousTypes = typesSelected ?? [];
        if (isType(selection)) {
            setTypesSelected(
                previousTypes.includes(selection)
                    ? previousTypes.filter((type) => type !== selection)
                    : [...previousTypes, selection]
            );
        } else {
            setTypesSelected(undefined);
        }
    }

    const options = types.map((type) => (
        <SelectOption key={type} value={type}>
            {getTypeText(type)}
        </SelectOption>
    ));
    options.push(
        <Divider key="Divider" />,
        <SelectOption key="All" value={optionAll}>
            All types
        </SelectOption>
    );

    return (
        <Select
            variant="checkbox"
            placeholderText="Filter by type"
            aria-label="Type filter menu items"
            toggleAriaLabel="Type filter menu toggle"
            onToggle={(_event, val) => setIsOpen(val)}
            onSelect={onSelect}
            selections={typesSelected ?? optionAll}
            isDisabled={isDisabled}
            isOpen={isOpen}
        >
            {options}
        </Select>
    );
}

export default SearchFilterTypes;

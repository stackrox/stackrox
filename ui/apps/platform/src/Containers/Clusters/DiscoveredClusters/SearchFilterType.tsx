import React, { useState } from 'react';
import { Divider, Select, SelectOption } from '@patternfly/react-core';

const optionAll = 'All';

type SearchFilterTypeProps = {
    types: string[]; // TODO
    isDisabled: boolean;
    setTypes: (types: string[]) => void; // TODO
};

function SearchFilterType({ types, isDisabled, setTypes }: SearchFilterTypeProps) {
    const [isOpen, setIsOpen] = useState(false);

    function onSelect(_event, selection) {
        setTypes(selection === optionAll ? [] : [selection]); // TODO add versus remove
        setIsOpen(false);
    }

    const options = types.map((typeArg) => (
        <SelectOption key={typeArg} value={typeArg}>
            {typeArg}
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
            selections={types[0] ?? optionAll}
            isDisabled={isDisabled}
            isOpen={isOpen}
        >
            {options}
        </Select>
    );
}

export default SearchFilterType;

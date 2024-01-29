import React, { ReactElement } from 'react';
import { Select, SelectOption } from '@patternfly/react-core';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { SearchOption } from '../searchOptions';

export type SearchOptionsDropdownProps = {
    setSearchOption: (selection) => void;
    searchOption: SearchOption;
    children: ReactElement<typeof SelectOption>[];
};

function SearchOptionsDropdown({
    setSearchOption,
    searchOption,
    children,
}: SearchOptionsDropdownProps) {
    const { isOpen, onToggle } = useSelectToggle();

    function onSearchOptionSelect(e, selection) {
        setSearchOption(selection);
    }

    return (
        <Select
            variant="single"
            toggleAriaLabel="search options filter menu toggle"
            aria-label="search options filter menu items"
            onToggle={onToggle}
            onSelect={onSearchOptionSelect}
            selections={searchOption.value}
            isOpen={isOpen}
            className="pf-u-flex-basis-0"
        >
            {children}
        </Select>
    );
}

export default SearchOptionsDropdown;

// @TODO: Replace the usage of FilterResourceDropdown with this

import React, { ReactElement } from 'react';
import { Select, SelectOption } from '@patternfly/react-core';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { SearchCategory } from 'services/SearchService';

export type SearchOption = { label: string; value: string; category: SearchCategory };

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
            toggleAriaLabel="exception request filter menu toggle"
            aria-label="exception request filter menu items"
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

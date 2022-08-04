import React, { useState } from 'react';
import { Select, SelectOption, SelectVariant } from '@patternfly/react-core';

export type CategoryFilter = 'Default categories' | 'Custom categories';

type PolicyCategoriesFilterSelectProps = {
    selectedFilters: CategoryFilter[];
    setSelectedFilters: (selectedFilters: CategoryFilter[]) => void;
    isDisabled: boolean;
};

function PolicyCategoriesFilterSelect({
    selectedFilters,
    setSelectedFilters,
    isDisabled,
}: PolicyCategoriesFilterSelectProps) {
    const [isOpen, setIsOpen] = useState(false);

    function onSelect(e, selection) {
        if (selectedFilters.includes(selection)) {
            setSelectedFilters(selectedFilters.filter((item) => item !== selection));
        } else {
            setSelectedFilters([...selectedFilters, selection]);
        }
    }

    return (
        <Select
            variant={SelectVariant.checkbox}
            onToggle={setIsOpen}
            onSelect={onSelect}
            isOpen={isOpen}
            selections={selectedFilters}
            isCheckboxSelectionBadgeHidden
            isDisabled={isDisabled}
            placeholderText={
                selectedFilters.length === 1 ? selectedFilters[0] : 'Show all categories'
            }
        >
            <SelectOption key={0} value="Default categories" />
            <SelectOption key={1} value="Custom categories" />
        </Select>
    );
}

export default PolicyCategoriesFilterSelect;

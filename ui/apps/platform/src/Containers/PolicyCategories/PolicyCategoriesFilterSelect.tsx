import React from 'react';
import { SelectOption } from '@patternfly/react-core';
import SelectSingle from 'Components/SelectSingle/SelectSingle';

export type CategoryFilter = 'All categories' | 'Default categories' | 'Custom categories';

type PolicyCategoriesFilterSelectProps = {
    selectedFilter: CategoryFilter;
    setSelectedFilter: (selectedFilter: CategoryFilter) => void;
    isDisabled: boolean;
};

function PolicyCategoriesFilterSelect({
    selectedFilter,
    setSelectedFilter,
    isDisabled,
}: PolicyCategoriesFilterSelectProps) {
    const handleChange = (_name: string, value: string) => {
        setSelectedFilter(value as CategoryFilter);
    };

    return (
        <SelectSingle
            id="policy-categories-filter"
            value={selectedFilter}
            handleSelect={handleChange}
            isDisabled={isDisabled}
            placeholderText="Select category filter"
            toggleAriaLabel="Policy categories filter"
            maxWidth="100%"
        >
            <SelectOption key={0} value="All categories">
                All categories
            </SelectOption>
            <SelectOption key={1} value="Default categories">
                Default categories
            </SelectOption>
            <SelectOption key={2} value="Custom categories">
                Custom categories
            </SelectOption>
        </SelectSingle>
    );
}

export default PolicyCategoriesFilterSelect;

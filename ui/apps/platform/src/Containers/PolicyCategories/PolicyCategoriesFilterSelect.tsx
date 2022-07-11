import React, { useEffect, useState } from 'react';
import { Select, SelectOption, SelectVariant } from '@patternfly/react-core';

import { PolicyCategory } from 'types/policy.proto';

type PolicyCategoriesFilterSelectProps = {
    setCurrentPolicyCategories: (policyCategories: PolicyCategory[]) => void;
    defaultPolicyCategories: PolicyCategory[];
    customPolicyCategories: PolicyCategory[];
};

function PolicyCategoriesFilterSelect({
    setCurrentPolicyCategories,
    defaultPolicyCategories,
    customPolicyCategories,
}: PolicyCategoriesFilterSelectProps) {
    const [isOpen, setIsOpen] = useState(false);
    const [selectedFilters, setSelectedFilters] = useState<string[]>([
        'Default categories',
        'Custom categories',
    ]);

    function onSelect(e, selection) {
        if (selectedFilters.includes(selection)) {
            setSelectedFilters(selectedFilters.filter((item) => item !== selection));
        } else {
            setSelectedFilters([...selectedFilters, selection]);
        }
    }

    useEffect(() => {
        if (selectedFilters.length === 1) {
            if (selectedFilters[0] === 'Default categories') {
                setCurrentPolicyCategories(defaultPolicyCategories);
            }
            if (selectedFilters[0] === 'Custom categories') {
                setCurrentPolicyCategories(customPolicyCategories);
            }
        } else {
            setCurrentPolicyCategories([...defaultPolicyCategories, ...customPolicyCategories]);
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [selectedFilters]);

    return (
        <Select
            variant={SelectVariant.checkbox}
            onToggle={setIsOpen}
            onSelect={onSelect}
            isOpen={isOpen}
            selections={selectedFilters}
            isCheckboxSelectionBadgeHidden
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

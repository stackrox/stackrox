import React from 'react';
import { SelectOption } from '@patternfly/react-core';

import CheckboxSelect from 'Components/PatternFly/CheckboxSelect';
import { searchValueAsArray } from 'utils/searchUtils';
import { SearchFilter } from 'types/search';

type CVEStatusDropdownProps<FilterField> = {
    filterField: FilterField;
    searchFilter: SearchFilter;
    onSelect: (filterType: FilterField, checked: boolean, selection: string) => void;
};

function CVEStatusDropdown<FilterField extends 'FIXABLE' | 'CLUSTER CVE FIXABLE'>({
    filterField,
    searchFilter,
    onSelect,
}: CVEStatusDropdownProps<FilterField>) {
    const selections = searchValueAsArray(searchFilter[filterField]);

    function handleItemSelect(selection: string, checked: boolean) {
        onSelect(filterField, checked, selection);
    }

    return (
        <CheckboxSelect
            id="vm-filter-toolbar-dropdown"
            selections={selections}
            onItemSelect={handleItemSelect}
            ariaLabel="CVE status filter menu items"
            placeholderText="CVE status"
        >
            <SelectOption value="Fixable">Fixable</SelectOption>
            <SelectOption value="Not fixable">Not fixable</SelectOption>
        </CheckboxSelect>
    );
}

export default CVEStatusDropdown;

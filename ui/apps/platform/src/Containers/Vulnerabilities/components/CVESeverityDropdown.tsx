import React from 'react';
import { SelectOption } from '@patternfly/react-core';

import CheckboxSelect from 'Components/PatternFly/CheckboxSelect';
import { searchValueAsArray } from 'utils/searchUtils';
import { SearchFilter } from 'types/search';

type CVESeverityDropdownProps = {
    searchFilter: SearchFilter;
    onSelect: (filterType: 'SEVERITY', checked: boolean, selection: string) => void;
};

function CVESeverityDropdown({ searchFilter, onSelect }: CVESeverityDropdownProps) {
    const selections = searchValueAsArray(searchFilter.SEVERITY);

    function handleItemSelect(selection: string, checked: boolean) {
        onSelect('SEVERITY', checked, selection);
    }

    return (
        <CheckboxSelect
            id="vm-filter-toolbar-dropdown cve-severity-select"
            selections={selections}
            onItemSelect={handleItemSelect}
            ariaLabel="CVE severity filter menu items"
            toggleAriaLabel="CVE severity filter menu toggle"
            placeholderText="CVE severity"
        >
            <SelectOption value="Critical">Critical</SelectOption>
            <SelectOption value="Important">Important</SelectOption>
            <SelectOption value="Moderate">Moderate</SelectOption>
            <SelectOption value="Low">Low</SelectOption>
            <SelectOption value="Unknown">Unknown</SelectOption>
        </CheckboxSelect>
    );
}

export default CVESeverityDropdown;

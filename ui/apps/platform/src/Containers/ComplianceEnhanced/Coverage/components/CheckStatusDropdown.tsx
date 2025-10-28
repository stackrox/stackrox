import React from 'react';
import { SelectOption } from '@patternfly/react-core';

import CheckboxSelect from 'Components/PatternFly/CheckboxSelect';
import { searchValueAsArray } from 'utils/searchUtils';
import type { SearchFilter } from 'types/search';

type CheckStatusDropdownProps = {
    searchFilter: SearchFilter;
    onSelect: (filterType: 'Compliance Check Status', checked: boolean, selection: string) => void;
};

function CheckStatusDropdown({ searchFilter, onSelect }: CheckStatusDropdownProps) {
    const selections = searchValueAsArray(searchFilter['Compliance Check Status']);

    function handleItemSelect(selection: string, checked: boolean) {
        onSelect('Compliance Check Status', checked, selection);
    }

    return (
        <CheckboxSelect
            selections={selections}
            onItemSelect={handleItemSelect}
            ariaLabel="Check status filter menu items"
            toggleAriaLabel="Check status filter menu toggle"
            placeholderText="Compliance status"
        >
            <SelectOption value="Pass">Pass</SelectOption>
            <SelectOption value="Fail">Fail</SelectOption>
            <SelectOption value="Error">Error</SelectOption>
            <SelectOption value="Info">Info</SelectOption>
            <SelectOption value="Manual">Manual</SelectOption>
            <SelectOption value="Not Applicable">Not Applicable</SelectOption>
            <SelectOption value="Inconsistent">Inconsistent</SelectOption>
        </CheckboxSelect>
    );
}

export default CheckStatusDropdown;

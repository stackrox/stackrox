import { SelectOption } from '@patternfly/react-core';

import CheckboxSelect from 'Components/PatternFly/CheckboxSelect';
import { searchValueAsArray } from 'utils/searchUtils';
import type { SearchFilter } from 'types/search';

type ComponentScannableStatusDropdownProps = {
    searchFilter: SearchFilter;
    onSelect: (filterType: 'SCANNABLE', checked: boolean, selection: string) => void;
};

function ComponentScannableStatusDropdown({
    searchFilter,
    onSelect,
}: ComponentScannableStatusDropdownProps) {
    const selections = searchValueAsArray(searchFilter.SCANNABLE);

    function handleItemSelect(selection: string, checked: boolean) {
        onSelect('SCANNABLE', checked, selection);
    }

    return (
        <CheckboxSelect
            id="vm-filter-toolbar-dropdown"
            selections={selections}
            onItemSelect={handleItemSelect}
            ariaLabel="Component scannable status filter menu items"
            toggleAriaLabel="Component scannable status filter menu toggle"
            placeholderText="Scan status"
        >
            <SelectOption value="Scanned">Scanned</SelectOption>
            <SelectOption value="Not scanned">Not scanned</SelectOption>
        </CheckboxSelect>
    );
}

export default ComponentScannableStatusDropdown;

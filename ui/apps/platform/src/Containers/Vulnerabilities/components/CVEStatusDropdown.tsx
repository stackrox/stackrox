import React, { useState } from 'react';
import { Select, SelectOption } from '@patternfly/react-core/deprecated';

import { SearchFilter } from 'types/search';

import './FilterDropdowns.css';

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
    const [cveStatusIsOpen, setCveStatusIsOpen] = useState(false);

    function onCveStatusToggle(isOpen: boolean) {
        setCveStatusIsOpen(isOpen);
    }

    return (
        <Select
            className="vm-filter-toolbar-dropdown"
            variant="checkbox"
            aria-label="CVE status filter menu items"
            toggleAriaLabel="CVE status filter menu toggle"
            onToggle={(_event, isOpen: boolean) => onCveStatusToggle(isOpen)}
            onSelect={(e, selection) => {
                onSelect(filterField, (e.target as HTMLInputElement).checked, selection as string);
            }}
            selections={searchFilter[filterField]}
            isOpen={cveStatusIsOpen}
            placeholderText="CVE status"
        >
            <SelectOption key="Fixable" value="Fixable" />
            <SelectOption key="NotFixable" value="Not fixable" />
        </Select>
    );
}

export default CVEStatusDropdown;

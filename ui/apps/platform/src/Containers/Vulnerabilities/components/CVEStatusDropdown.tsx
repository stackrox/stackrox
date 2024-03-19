import React, { useState } from 'react';
import { Select, SelectOption } from '@patternfly/react-core';

import { SearchFilter } from 'types/search';

import './FilterDropdowns.css';

type CVEStatusDropdownProps = {
    searchFilter: SearchFilter;
    onSelect: (filterType: 'FIXABLE', checked: boolean, selection: string) => void;
};

function CVEStatusDropdown({ searchFilter, onSelect }: CVEStatusDropdownProps) {
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
            onToggle={onCveStatusToggle}
            onSelect={(e, selection) => {
                onSelect('FIXABLE', (e.target as HTMLInputElement).checked, selection as string);
            }}
            selections={searchFilter.FIXABLE}
            isOpen={cveStatusIsOpen}
            placeholderText="CVE status"
        >
            <SelectOption key="Fixable" value="Fixable" />
            <SelectOption key="Important" value="Not fixable" />
        </Select>
    );
}

export default CVEStatusDropdown;

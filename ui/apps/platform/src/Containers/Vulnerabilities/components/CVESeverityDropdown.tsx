import React, { useState } from 'react';
import { Select, SelectOption } from '@patternfly/react-core';

import { SearchFilter } from 'types/search';

import './FilterDropdowns.css';

type CVESeverityDropdownProps = {
    searchFilter: SearchFilter;
    onSelect: (filterType: 'SEVERITY', checked: boolean, selection: string) => void;
};

function CVESeverityDropdown({ searchFilter, onSelect }: CVESeverityDropdownProps) {
    const [cveSeverityIsOpen, setCveSeverityIsOpen] = useState(false);

    function onCveSeverityToggle(isOpen: boolean) {
        setCveSeverityIsOpen(isOpen);
    }

    return (
        <Select
            variant="checkbox"
            aria-label="CVE severity filter menu items"
            toggleAriaLabel="CVE severity filter menu toggle"
            onToggle={onCveSeverityToggle}
            onSelect={(e, selection) => {
                onSelect('SEVERITY', (e.target as HTMLInputElement).checked, selection as string);
            }}
            selections={searchFilter.SEVERITY}
            isOpen={cveSeverityIsOpen}
            placeholderText="CVE severity"
            className="vm-filter-toolbar-dropdown cve-severity-select"
        >
            <SelectOption key="Critical" value="Critical" />
            <SelectOption key="Important" value="Important" />
            <SelectOption key="Moderate" value="Moderate" />
            <SelectOption key="Low" value="Low" />
        </Select>
    );
}

export default CVESeverityDropdown;

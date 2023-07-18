import React, { useState } from 'react';
import { Select, SelectOption } from '@patternfly/react-core';

import { SearchFilter } from 'types/search';

type CVESeverityDropdownProps = {
    searchFilter: SearchFilter;
    onSelect: (filterType, e, selection) => void;
};

function CVESeverityDropdown({ searchFilter, onSelect }: CVESeverityDropdownProps) {
    const [cveSeverityIsOpen, setCveSeverityIsOpen] = useState(false);

    function onCveSeverityToggle(isOpen: boolean) {
        setCveSeverityIsOpen(isOpen);
    }

    function onCveSeveritySelect(e, selection) {
        onSelect('Severity', e, selection);
    }

    return (
        <Select
            variant="checkbox"
            aria-label="CVE severity filter menu items"
            toggleAriaLabel="CVE severity filter menu toggle"
            onToggle={onCveSeverityToggle}
            onSelect={onCveSeveritySelect}
            selections={searchFilter.Severity}
            isOpen={cveSeverityIsOpen}
            placeholderText="CVE severity"
            className="cve-severity-select"
        >
            <SelectOption key="Critical" value="Critical" />
            <SelectOption key="Important" value="Important" />
            <SelectOption key="Moderate" value="Moderate" />
            <SelectOption key="Low" value="Low" />
        </Select>
    );
}

export default CVESeverityDropdown;

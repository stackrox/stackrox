import React, { useState } from 'react';
import { Select, SelectOption } from '@patternfly/react-core';

import { SearchFilter } from 'types/search';

type CVEStatusDropdownProps = {
    searchFilter: SearchFilter;
    onSelect: (filterType, e, selection) => void;
};

function CVEStatusDropdown({ searchFilter, onSelect }: CVEStatusDropdownProps) {
    const [cveStatusIsOpen, setCveStatusIsOpen] = useState(false);

    function onCveStatusToggle(isOpen: boolean) {
        setCveStatusIsOpen(isOpen);
    }
    function onCveStatusSelect(e, selection) {
        onSelect('Fixable', e, selection);
    }

    return (
        <Select
            variant="checkbox"
            aria-label="CVE status filter menu items"
            toggleAriaLabel="CVE status filter menu toggle"
            onToggle={onCveStatusToggle}
            onSelect={onCveStatusSelect}
            selections={searchFilter.Fixable}
            isOpen={cveStatusIsOpen}
            placeholderText="CVE status"
        >
            <SelectOption key="Fixable" value="Fixable" />
            <SelectOption key="Important" value="Not fixable" />
        </Select>
    );
}

export default CVEStatusDropdown;

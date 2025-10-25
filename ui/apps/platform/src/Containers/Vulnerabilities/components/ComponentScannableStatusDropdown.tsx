import React, { useState } from 'react';
import { Select, SelectOption } from '@patternfly/react-core/deprecated';

import { SearchFilter } from 'types/search';

import './FilterDropdowns.css';

type ComponentScannableStatusDropdownProps = {
    searchFilter: SearchFilter;
    onSelect: (filterType: 'SCANNABLE', checked: boolean, selection: string) => void;
};

function ComponentScannableStatusDropdown({
    searchFilter,
    onSelect,
}: ComponentScannableStatusDropdownProps) {
    const [componentScannableStatusIsOpen, setComponentScannableStatusIsOpen] = useState(false);

    function onComponentScannableStatusToggle(isOpen: boolean) {
        setComponentScannableStatusIsOpen(isOpen);
    }

    return (
        <Select
            variant="checkbox"
            aria-label="Component scannable status filter menu items"
            toggleAriaLabel="Component scannable status filter menu toggle"
            onToggle={(_event, isOpen: boolean) => onComponentScannableStatusToggle(isOpen)}
            onSelect={(e, selection) => {
                onSelect('SCANNABLE', (e.target as HTMLInputElement).checked, selection as string);
            }}
            selections={searchFilter.SCANNABLE}
            isOpen={componentScannableStatusIsOpen}
            placeholderText="Scan status"
            className="vm-filter-toolbar-dropdown"
        >
            <SelectOption key="Scanned" value="Scanned" />
            <SelectOption key="NotScanned" value="Not scanned" />
        </Select>
    );
}

export default ComponentScannableStatusDropdown;

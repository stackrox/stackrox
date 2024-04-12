import React, { useState } from 'react';
import { Select, SelectOption } from '@patternfly/react-core/deprecated';

import { SearchFilter } from 'types/search';

type CheckStatusDropdownProps = {
    searchFilter: SearchFilter;
    onSelect: (filterType: 'Compliance Check Status', checked: boolean, selection: string) => void;
};

function CheckStatusDropdown({ searchFilter, onSelect }: CheckStatusDropdownProps) {
    const [checkStatusIsOpen, setCheckStatusIsOpen] = useState(false);

    function onCheckStatusToggle(isOpen: boolean) {
        setCheckStatusIsOpen(isOpen);
    }

    return (
        <Select
            variant="checkbox"
            aria-label="Check status filter menu items"
            toggleAriaLabel="Check status filter menu toggle"
            onToggle={(_event, isOpen: boolean) => onCheckStatusToggle(isOpen)}
            onSelect={(event, selection) => {
                const { checked } = event.target as HTMLInputElement;
                onSelect('Compliance Check Status', checked, selection.toString());
            }}
            selections={searchFilter['Compliance Check Status']}
            isOpen={checkStatusIsOpen}
            placeholderText="Compliance status"
        >
            <SelectOption key="PASS" value="Pass" />
            <SelectOption key="FAIL" value="Fail" />
            <SelectOption key="ERROR" value="Error" />
            <SelectOption key="INFO" value="Info" />
            <SelectOption key="MANUAL" value="Manual" />
            <SelectOption key="NOT_APPLICABLE" value="Not Applicable" />
            <SelectOption key="INCONSISTENT" value="Inconsistent" />
        </Select>
    );
}

export default CheckStatusDropdown;

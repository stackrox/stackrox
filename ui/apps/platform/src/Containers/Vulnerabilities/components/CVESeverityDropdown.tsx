import React, { useState } from 'react';
import type { MouseEvent as ReactMouseEvent, Ref } from 'react';
import {
    Badge,
    Flex,
    FlexItem,
    MenuToggle,
    Select,
    SelectList,
    SelectOption,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';

import { searchValueAsArray } from 'utils/searchUtils';
import { SearchFilter } from 'types/search';

type CVESeverityDropdownProps = {
    searchFilter: SearchFilter;
    onSelect: (filterType: 'SEVERITY', checked: boolean, selection: string) => void;
};

function CVESeverityDropdown({ searchFilter, onSelect }: CVESeverityDropdownProps) {
    const [cveSeverityIsOpen, setCveSeverityIsOpen] = useState(false);

    const selections = searchValueAsArray(searchFilter.SEVERITY);

    function onToggle() {
        setCveSeverityIsOpen((prev) => !prev);
    }

    function handleSelect(
        _event: ReactMouseEvent<Element, MouseEvent> | undefined,
        selection: string | number | undefined
    ) {
        if (typeof selection !== 'string') {
            return;
        }

        const isSelected = selections.includes(selection);
        onSelect('SEVERITY', !isSelected, selection);
    }

    const toggle = (toggleRef: Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            onClick={onToggle}
            isExpanded={cveSeverityIsOpen}
            aria-label="CVE severity filter menu toggle"
        >
            <Flex
                alignItems={{ default: 'alignItemsCenter' }}
                spaceItems={{ default: 'spaceItemsSm' }}
                flexWrap={{ default: 'nowrap' }}
            >
                <FlexItem>CVE severity</FlexItem>
                {selections.length > 0 && <Badge isRead>{selections.length}</Badge>}
            </Flex>
        </MenuToggle>
    );

    return (
        <Select
            className="vm-filter-toolbar-dropdown cve-severity-select"
            aria-label="CVE severity filter menu items"
            isOpen={cveSeverityIsOpen}
            selected={selections}
            onSelect={handleSelect}
            onOpenChange={(nextOpen: boolean) => setCveSeverityIsOpen(nextOpen)}
            toggle={toggle}
            shouldFocusToggleOnSelect
        >
            <SelectList>
                <SelectOption
                    value="Critical"
                    hasCheckbox
                    isSelected={selections.includes('Critical')}
                >
                    Critical
                </SelectOption>
                <SelectOption
                    value="Important"
                    hasCheckbox
                    isSelected={selections.includes('Important')}
                >
                    Important
                </SelectOption>
                <SelectOption
                    value="Moderate"
                    hasCheckbox
                    isSelected={selections.includes('Moderate')}
                >
                    Moderate
                </SelectOption>
                <SelectOption value="Low" hasCheckbox isSelected={selections.includes('Low')}>
                    Low
                </SelectOption>
                <SelectOption
                    value="Unknown"
                    hasCheckbox
                    isSelected={selections.includes('Unknown')}
                >
                    Unknown
                </SelectOption>
            </SelectList>
        </Select>
    );
}

export default CVESeverityDropdown;

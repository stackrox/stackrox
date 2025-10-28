import React, { useState } from 'react';
import type { MouseEvent as ReactMouseEvent, Ref } from 'react';
import {
    Select,
    SelectOption,
    SelectList,
    MenuToggle,
    Badge,
    Flex,
    FlexItem,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';

import { searchValueAsArray } from 'utils/searchUtils';
import { SearchFilter } from 'types/search';

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

    const selections = searchValueAsArray(searchFilter[filterField]);

    function onToggle() {
        setCveStatusIsOpen((prev) => !prev);
    }

    function handleSelect(
        _event: ReactMouseEvent<Element, MouseEvent> | undefined,
        selection: string | number | undefined
    ) {
        if (typeof selection !== 'string') {
            return;
        }

        const isSelected = selections.includes(selection);
        onSelect(filterField, !isSelected, selection);
    }

    const toggle = (toggleRef: Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            onClick={onToggle}
            isExpanded={cveStatusIsOpen}
            aria-label="CVE status filter menu toggle"
        >
            <Flex
                alignItems={{ default: 'alignItemsCenter' }}
                spaceItems={{ default: 'spaceItemsSm' }}
                flexWrap={{ default: 'nowrap' }}
            >
                <FlexItem>CVE status</FlexItem>
                {selections.length > 0 && <Badge isRead>{selections.length}</Badge>}
            </Flex>
        </MenuToggle>
    );

    return (
        <Select
            className="vm-filter-toolbar-dropdown"
            aria-label="CVE status filter menu items"
            isOpen={cveStatusIsOpen}
            selected={selections}
            onSelect={handleSelect}
            onOpenChange={(nextOpen: boolean) => setCveStatusIsOpen(nextOpen)}
            toggle={toggle}
            shouldFocusToggleOnSelect
        >
            <SelectList>
                <SelectOption
                    value="Fixable"
                    hasCheckbox
                    isSelected={selections.includes('Fixable')}
                >
                    Fixable
                </SelectOption>
                <SelectOption
                    value="Not fixable"
                    hasCheckbox
                    isSelected={selections.includes('Not fixable')}
                >
                    Not fixable
                </SelectOption>
            </SelectList>
        </Select>
    );
}

export default CVEStatusDropdown;

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

import type { SearchFilter } from 'types/search';
import { searchValueAsArray } from 'utils/searchUtils';

type CheckStatusDropdownProps = {
    searchFilter: SearchFilter;
    onSelect: (filterType: 'Compliance Check Status', checked: boolean, selection: string) => void;
};

function CheckStatusDropdown({ searchFilter, onSelect }: CheckStatusDropdownProps) {
    const [checkStatusIsOpen, setCheckStatusIsOpen] = useState(false);

    const selections = searchValueAsArray(searchFilter['Compliance Check Status']);

    function onToggle() {
        setCheckStatusIsOpen((prev) => !prev);
    }

    function handleSelect(
        _event: ReactMouseEvent<Element, MouseEvent> | undefined,
        selection: string | number | undefined
    ) {
        if (typeof selection !== 'string') {
            return;
        }

        const isSelected = selections.includes(selection);
        onSelect('Compliance Check Status', !isSelected, selection);
    }

    const toggle = (toggleRef: Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            onClick={onToggle}
            isExpanded={checkStatusIsOpen}
            aria-label="Check status filter menu toggle"
        >
            <Flex
                alignItems={{ default: 'alignItemsCenter' }}
                spaceItems={{ default: 'spaceItemsSm' }}
            >
                <FlexItem>Compliance status</FlexItem>
                {selections.length > 0 && <Badge isRead>{selections.length}</Badge>}
            </Flex>
        </MenuToggle>
    );

    return (
        <Select
            aria-label="Check status filter menu items"
            isOpen={checkStatusIsOpen}
            selected={selections}
            onSelect={handleSelect}
            onOpenChange={(nextOpen: boolean) => setCheckStatusIsOpen(nextOpen)}
            toggle={toggle}
            shouldFocusToggleOnSelect
        >
            <SelectList>
                <SelectOption value="Pass" hasCheckbox isSelected={selections.includes('Pass')}>
                    Pass
                </SelectOption>
                <SelectOption value="Fail" hasCheckbox isSelected={selections.includes('Fail')}>
                    Fail
                </SelectOption>
                <SelectOption value="Error" hasCheckbox isSelected={selections.includes('Error')}>
                    Error
                </SelectOption>
                <SelectOption value="Info" hasCheckbox isSelected={selections.includes('Info')}>
                    Info
                </SelectOption>
                <SelectOption value="Manual" hasCheckbox isSelected={selections.includes('Manual')}>
                    Manual
                </SelectOption>
                <SelectOption
                    value="Not Applicable"
                    hasCheckbox
                    isSelected={selections.includes('Not Applicable')}
                >
                    Not Applicable
                </SelectOption>
                <SelectOption
                    value="Inconsistent"
                    hasCheckbox
                    isSelected={selections.includes('Inconsistent')}
                >
                    Inconsistent
                </SelectOption>
            </SelectList>
        </Select>
    );
}

export default CheckStatusDropdown;

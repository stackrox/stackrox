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

type ComponentScannableStatusDropdownProps = {
    searchFilter: SearchFilter;
    onSelect: (filterType: 'SCANNABLE', checked: boolean, selection: string) => void;
};

function ComponentScannableStatusDropdown({
    searchFilter,
    onSelect,
}: ComponentScannableStatusDropdownProps) {
    const [componentScannableStatusIsOpen, setComponentScannableStatusIsOpen] = useState(false);

    const selections = searchValueAsArray(searchFilter.SCANNABLE);

    function onToggle() {
        setComponentScannableStatusIsOpen((prev) => !prev);
    }

    function handleSelect(
        _event: ReactMouseEvent<Element, MouseEvent> | undefined,
        selection: string | number | undefined
    ) {
        if (typeof selection !== 'string') {
            return;
        }

        const isSelected = selections.includes(selection);
        onSelect('SCANNABLE', !isSelected, selection);
    }

    const toggle = (toggleRef: Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            onClick={onToggle}
            isExpanded={componentScannableStatusIsOpen}
            aria-label="Component scannable status filter menu toggle"
        >
            <Flex
                alignItems={{ default: 'alignItemsCenter' }}
                spaceItems={{ default: 'spaceItemsSm' }}
                flexWrap={{ default: 'nowrap' }}
            >
                <FlexItem>Scan status</FlexItem>
                {selections.length > 0 && <Badge isRead>{selections.length}</Badge>}
            </Flex>
        </MenuToggle>
    );

    return (
        <Select
            className="vm-filter-toolbar-dropdown"
            aria-label="Component scannable status filter menu items"
            isOpen={componentScannableStatusIsOpen}
            selected={selections}
            onSelect={handleSelect}
            onOpenChange={(nextOpen: boolean) => setComponentScannableStatusIsOpen(nextOpen)}
            toggle={toggle}
            shouldFocusToggleOnSelect
        >
            <SelectList>
                <SelectOption
                    value="Scanned"
                    hasCheckbox
                    isSelected={selections.includes('Scanned')}
                >
                    Scanned
                </SelectOption>
                <SelectOption
                    value="Not scanned"
                    hasCheckbox
                    isSelected={selections.includes('Not scanned')}
                >
                    Not scanned
                </SelectOption>
            </SelectList>
        </Select>
    );
}

export default ComponentScannableStatusDropdown;

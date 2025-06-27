import React, { ReactElement, ReactNode, useState } from 'react';
import { Dropdown, DropdownList, MenuToggle, MenuToggleElement } from '@patternfly/react-core';

type MenuDropdownProps = {
    children: ReactNode;
    toggleText: string;
    toggleId?: string;
    isDisabled?: boolean;
};

// TODO: Reuse this for the Violations Page Bulk Actions
function MenuDropdown({
    children,
    toggleText,
    toggleId = 'menu-dropdown',
    isDisabled = false,
}: MenuDropdownProps): ReactElement {
    const [isOpen, setIsOpen] = useState(false);

    function onToggleClick() {
        setIsOpen(!isOpen);
    }

    function onSelect() {
        setIsOpen(false);
    }

    return (
        <Dropdown
            isOpen={isOpen}
            onSelect={onSelect}
            onOpenChange={(isOpen: boolean) => setIsOpen(isOpen)}
            toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                <MenuToggle
                    id={toggleId}
                    ref={toggleRef}
                    onClick={onToggleClick}
                    isExpanded={isOpen}
                    isDisabled={isDisabled}
                >
                    {toggleText}
                </MenuToggle>
            )}
            shouldFocusToggleOnSelect
        >
            <DropdownList>{children}</DropdownList>
        </Dropdown>
    );
}

export default MenuDropdown;

import React, { ReactElement, ReactNode, useState } from 'react';
import {
    Dropdown,
    DropdownList,
    DropdownPopperProps,
    MenuToggle,
    MenuToggleElement,
    MenuToggleProps,
} from '@patternfly/react-core';

type MenuDropdownProps = {
    children: ReactNode;
    toggleText: string;
    toggleId?: string;
    toggleClassName?: string;
    toggleVariant?: MenuToggleProps['variant'];
    onSelect?: (event?: React.MouseEvent<Element, MouseEvent>, value?: string | number) => void;
    isDisabled?: boolean;
    popperProps?: DropdownPopperProps;
};

// TODO: Reuse this for the Violations Page Bulk Actions
function MenuDropdown({
    children,
    toggleText,
    toggleId = 'menu-dropdown',
    toggleClassName,
    toggleVariant = 'default',
    onSelect,
    isDisabled = false,
    popperProps,
}: MenuDropdownProps): ReactElement {
    const [isOpen, setIsOpen] = useState(false);

    function onToggleClick() {
        setIsOpen(!isOpen);
    }

    function onSelectHandler(event) {
        setIsOpen(false);
        onSelect?.call(event);
    }

    return (
        <Dropdown
            isOpen={isOpen}
            onSelect={onSelectHandler}
            onOpenChange={(isOpen: boolean) => setIsOpen(isOpen)}
            toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                <MenuToggle
                    className={toggleClassName}
                    id={toggleId}
                    ref={toggleRef}
                    onClick={onToggleClick}
                    isExpanded={isOpen}
                    isDisabled={isDisabled}
                    variant={toggleVariant}
                >
                    {toggleText}
                </MenuToggle>
            )}
            shouldFocusToggleOnSelect
            popperProps={popperProps}
        >
            <DropdownList>{children}</DropdownList>
        </Dropdown>
    );
}

export default MenuDropdown;

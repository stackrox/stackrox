import React, { useState } from 'react';
import type { MouseEvent as ReactMouseEvent, ReactElement, ReactNode, Ref } from 'react';
import { Bullseye, Dropdown, DropdownList, MenuToggle, Spinner } from '@patternfly/react-core';
import type {
    DropdownPopperProps,
    MenuToggleElement,
    MenuToggleProps,
} from '@patternfly/react-core';

type MenuDropdownProps = {
    children: ReactNode;
    toggleText: string;
    toggleId?: string;
    toggleClassName?: string;
    toggleVariant?: MenuToggleProps['variant'];
    toggleIcon?: ReactNode;
    ariaLabel?: string;
    onSelect?: (event?: ReactMouseEvent<Element, MouseEvent>, value?: string | number) => void;
    isDisabled?: boolean;
    isPlain?: boolean;
    isLoading?: boolean;
    popperProps?: DropdownPopperProps;
};

// TODO: Reuse this for the Violations Page Bulk Actions
function MenuDropdown({
    children,
    toggleText,
    toggleId = 'menu-dropdown',
    toggleClassName,
    toggleVariant = 'default',
    toggleIcon,
    ariaLabel,
    onSelect,
    isDisabled = false,
    isPlain = false,
    isLoading = false,
    popperProps,
}: MenuDropdownProps): ReactElement {
    const [isOpen, setIsOpen] = useState(false);

    function onToggleClick() {
        setIsOpen(!isOpen);
    }

    function onSelectHandler(event, value) {
        setIsOpen(false);
        if (onSelect) {
            onSelect(event, value);
        }
    }

    return (
        <Dropdown
            isOpen={isOpen}
            isPlain={isPlain}
            onSelect={onSelectHandler}
            onOpenChange={(isOpen: boolean) => setIsOpen(isOpen)}
            toggle={(toggleRef: Ref<MenuToggleElement>) => (
                <MenuToggle
                    className={toggleClassName}
                    icon={toggleIcon}
                    id={toggleId}
                    ref={toggleRef}
                    onClick={onToggleClick}
                    isExpanded={isOpen}
                    isDisabled={isDisabled}
                    variant={toggleVariant}
                    aria-label={ariaLabel}
                >
                    {isLoading ? (
                        <Bullseye>
                            <Spinner size="md" />
                        </Bullseye>
                    ) : (
                        toggleText
                    )}
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

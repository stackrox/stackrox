import React, { ReactElement, ReactNode, useState } from 'react';
import { Dropdown, DropdownList, MenuToggle, MenuToggleElement } from '@patternfly/react-core';

type BulkActionsDropdownProps = {
    children: ReactNode;
    isDisabled?: boolean;
};

// TODO: Connect this to the APIs
// TODO: Reuse this for the Violations Page Bulk Actions
function BulkActionsDropdown({
    children,
    isDisabled = false,
}: BulkActionsDropdownProps): ReactElement {
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
                    id="bulk-actions-dropdown"
                    ref={toggleRef}
                    onClick={onToggleClick}
                    isExpanded={isOpen}
                    isDisabled={isDisabled}
                >
                    Bulk actions
                </MenuToggle>
            )}
            shouldFocusToggleOnSelect
        >
            <DropdownList>{children}</DropdownList>
        </Dropdown>
    );
}

export default BulkActionsDropdown;

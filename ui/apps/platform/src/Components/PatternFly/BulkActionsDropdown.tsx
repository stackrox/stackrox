import React, { ReactElement, ReactNode, useState } from 'react';
import { Dropdown, DropdownToggle } from '@patternfly/react-core/deprecated';
import { CaretDownIcon } from '@patternfly/react-icons';

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

    function onToggle(value) {
        setIsOpen(value);
    }

    function onFocus() {
        const element = document.getElementById('bulk-actions-dropdown');
        element?.focus();
    }

    // @TODO: Use "event.target.id" to figure out what was clicked and then do an action
    function onSelect() {
        setIsOpen((prevValue) => !prevValue);
        onFocus();
    }

    const dropdownItems = React.Children.toArray(children);

    return (
        <Dropdown
            onSelect={onSelect}
            toggle={
                <DropdownToggle
                    id="bulk-actions-dropdown"
                    onToggle={(_event, value) => onToggle(value)}
                    toggleIndicator={CaretDownIcon}
                    isDisabled={isDisabled}
                >
                    Bulk actions
                </DropdownToggle>
            }
            isOpen={isOpen}
            dropdownItems={dropdownItems}
        />
    );
}

export default BulkActionsDropdown;

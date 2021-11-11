import React, { useState } from 'react';
import {
    Dropdown,
    DropdownToggle,
    DropdownItem,
    Button,
    Tooltip,
    Flex,
    FlexItem,
} from '@patternfly/react-core';
import { CaretDownIcon } from '@patternfly/react-icons';

type PoliciesTablePageActionButtonsProps = {
    onClickImportPolicy: () => void;
    onClickReassessPolicies: () => void;
};

function PoliciesTablePageActionButtons({
    onClickImportPolicy,
    onClickReassessPolicies,
}: PoliciesTablePageActionButtonsProps): React.ReactElement {
    const [isDropdownOpen, setIsDropdownOpen] = useState(false);

    function onToggleDropdown(toggleDropdown) {
        setIsDropdownOpen(toggleDropdown);
    }

    function handleOnClickImportPolicy() {
        setIsDropdownOpen(false);
        onClickImportPolicy();
    }

    const dropdownItems = [
        // TODO: add link to create form
        <DropdownItem key="link">Create policy</DropdownItem>,
        <DropdownItem key="action" component="button" onClick={handleOnClickImportPolicy}>
            Import policy
        </DropdownItem>,
    ];

    return (
        <Flex>
            <FlexItem>
                <Dropdown
                    toggle={
                        <DropdownToggle
                            onToggle={onToggleDropdown}
                            toggleIndicator={CaretDownIcon}
                            isPrimary
                            id="add-policy-dropdown-toggle"
                        >
                            Add Policy
                        </DropdownToggle>
                    }
                    isOpen={isDropdownOpen}
                    dropdownItems={dropdownItems}
                />
            </FlexItem>
            <FlexItem>
                <Tooltip content="Manually enrich external data">
                    <Button variant="secondary" onClick={onClickReassessPolicies}>
                        Reassess all
                    </Button>
                </Tooltip>
            </FlexItem>
        </Flex>
    );
}

export default PoliciesTablePageActionButtons;

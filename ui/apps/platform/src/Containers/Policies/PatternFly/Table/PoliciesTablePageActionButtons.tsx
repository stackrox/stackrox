import React, { useState } from 'react';
import { Link } from 'react-router-dom';
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

import { policiesBasePath } from 'routePaths';

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
        <DropdownItem
            key="routerlink"
            component={<Link to={`${policiesBasePath}/?action=create`}>Create policy</Link>}
        />,
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

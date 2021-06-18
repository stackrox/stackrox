/* eslint-disable react/jsx-no-bind */
import React, { ReactElement, useState } from 'react';
import { Dropdown, DropdownItem, DropdownToggle } from '@patternfly/react-core';

import { LabelSelectorOperator } from 'services/RolesService';

export type AddRequirementDropdownProps = {
    isDisabled: boolean;
    onAddRequirement: (op: LabelSelectorOperator) => void;
};

function AddRequirementDropdown({
    isDisabled,
    onAddRequirement,
}: AddRequirementDropdownProps): ReactElement {
    const [isOpen, setIsOpen] = useState(false);

    // TODO Solve problem with text color and then move isDisabled from items to toggle.
    const dropdownItems = [
        <DropdownItem
            component="button"
            isDisabled={isDisabled}
            onClick={() => onAddRequirement('IN')}
        >
            key in set of values
        </DropdownItem>,
        <DropdownItem
            component="button"
            isDisabled={isDisabled}
            onClick={() => onAddRequirement('NOT_IN')}
        >
            key not in set of values
        </DropdownItem>,
        <DropdownItem
            component="button"
            isDisabled={isDisabled}
            onClick={() => onAddRequirement('EXISTS')}
        >
            key exists
        </DropdownItem>,
        <DropdownItem
            component="button"
            isDisabled={isDisabled}
            onClick={() => onAddRequirement('NOT_EXISTS')}
        >
            key not exists
        </DropdownItem>,
    ];

    return (
        <Dropdown
            className="pf-m-small"
            dropdownItems={dropdownItems}
            isOpen={isOpen}
            onSelect={() => setIsOpen(false)}
            toggle={
                <DropdownToggle
                    isPrimary
                    isDisabled={false}
                    onToggle={(isOpenToggle) => setIsOpen(isOpenToggle)}
                >
                    Add requirement
                </DropdownToggle>
            }
        />
    );
}

export default AddRequirementDropdown;

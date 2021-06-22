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

    const dropdownItems = [
        <DropdownItem component="button" onClick={() => onAddRequirement('EXISTS')}>
            key <strong>exists</strong>
        </DropdownItem>,
        <DropdownItem component="button" onClick={() => onAddRequirement('NOT_EXISTS')}>
            key <strong>not exists</strong>
        </DropdownItem>,
        <DropdownItem component="button" onClick={() => onAddRequirement('IN')}>
            key <strong>in</strong> set of values (also for key = value)
        </DropdownItem>,
        <DropdownItem component="button" onClick={() => onAddRequirement('NOT_IN')}>
            key <strong>not in</strong> set of values (also for key != value)
        </DropdownItem>,
    ];

    return (
        <Dropdown
            className="pf-m-smaller"
            dropdownItems={dropdownItems}
            isOpen={isOpen}
            onSelect={() => setIsOpen(false)}
            toggle={
                <DropdownToggle
                    isPrimary
                    isDisabled={isDisabled}
                    onToggle={(isOpenToggle) => setIsOpen(isOpenToggle)}
                >
                    Add requirement
                </DropdownToggle>
            }
        />
    );
}

export default AddRequirementDropdown;

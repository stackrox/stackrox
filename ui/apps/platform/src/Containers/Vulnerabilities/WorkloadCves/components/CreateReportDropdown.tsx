import React from 'react';
import {
    Dropdown,
    DropdownItem,
    DropdownList,
    MenuToggle,
    MenuToggleElement,
} from '@patternfly/react-core';

function CreateReportDropdown({ isOpen, setIsOpen, onSelect }) {
    const onToggleClick = () => {
        setIsOpen(!isOpen);
    };

    const onSelectHandler = (
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined
    ) => {
        console.log('selected', value);
        onSelect(value);
        setIsOpen(false);
    };

    return (
        <>
            <Dropdown
                isOpen={isOpen}
                onSelect={onSelectHandler}
                onOpenChange={(isOpen: boolean) => setIsOpen(isOpen)}
                toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                    <MenuToggle ref={toggleRef} onClick={onToggleClick} isExpanded={isOpen}>
                        Create report
                    </MenuToggle>
                )}
                shouldFocusToggleOnSelect
                popperProps={{ position: 'right' }}
            >
                <DropdownList>
                    <DropdownItem
                        value="Export report as CSV"
                        key="Export report as CSV"
                        description="Export a one time CSV report from this view using the filters you've applied."
                    >
                        Export report as CSV
                    </DropdownItem>
                </DropdownList>
            </Dropdown>
        </>
    );
}

export default CreateReportDropdown;

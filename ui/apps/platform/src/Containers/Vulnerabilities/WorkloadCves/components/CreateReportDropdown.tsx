import React from 'react';
import {
    Dropdown,
    DropdownItem,
    DropdownList,
    MenuToggle,
    MenuToggleElement,
} from '@patternfly/react-core';

export type CreateReportDropdownProps = {
    isOpen: boolean;
    setIsOpen: (value: boolean) => void;
    onSelect: (value: string | number | undefined) => void;
};

function CreateReportDropdown({ isOpen, setIsOpen, onSelect }: CreateReportDropdownProps) {
    const onToggleClick = () => {
        setIsOpen(!isOpen);
    };

    const onSelectHandler = (
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined
    ) => {
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
                        description="Export an on-demand CSV report from this view using the filters you've applied."
                    >
                        Export report as CSV
                    </DropdownItem>
                </DropdownList>
            </Dropdown>
        </>
    );
}

export default CreateReportDropdown;

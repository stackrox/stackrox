import React, { ReactElement } from 'react';
import {
    Select,
    SelectOption,
    SelectList,
    MenuToggle,
    MenuToggleElement,
} from '@patternfly/react-core';

export type SimpleSelectProps = {
    value: string | number | undefined;
    onChange: (value: string) => void;
    children: ReactElement<typeof SelectOption>[];
    id: string;
    isDisabled?: boolean;
    ariaLabelMenu?: string;
    ariaLabelToggle?: string;
};

function SimpleSelect({
    value,
    onChange,
    children,
    id,
    isDisabled = false,
    ariaLabelMenu = '',
    ariaLabelToggle = '',
}: SimpleSelectProps) {
    const [isOpen, setIsOpen] = React.useState(false);

    const onToggleClick = () => {
        setIsOpen(!isOpen);
    };

    const onSelect = (
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        newValue: string | number | undefined
    ) => {
        onChange(newValue as string);
        setIsOpen(false);
    };

    const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
        <MenuToggle
            aria-label={ariaLabelToggle}
            ref={toggleRef}
            onClick={onToggleClick}
            isExpanded={isOpen}
            isDisabled={isDisabled}
        >
            {value}
        </MenuToggle>
    );

    return (
        <Select
            id={id}
            aria-label={ariaLabelMenu}
            isOpen={isOpen}
            selected={value}
            onSelect={onSelect}
            onOpenChange={(isOpen) => setIsOpen(isOpen)}
            toggle={toggle}
            shouldFocusToggleOnSelect
        >
            <SelectList>{children}</SelectList>
        </Select>
    );
}

export default SimpleSelect;

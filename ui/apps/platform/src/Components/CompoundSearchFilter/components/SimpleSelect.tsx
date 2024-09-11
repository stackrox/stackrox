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
    onChange: (value: string | number | undefined) => void;
    children: ReactElement<typeof SelectOption>[];
    isDisabled?: boolean;
    ariaLabelMenu?: string;
    ariaLabelToggle?: string;
    menuToggleClassName?: string;
};

function SimpleSelect({
    value,
    onChange,
    children,
    isDisabled = false,
    ariaLabelMenu,
    ariaLabelToggle,
    menuToggleClassName,
}: SimpleSelectProps) {
    const [isOpen, setIsOpen] = React.useState(false);

    const onToggleClick = () => {
        setIsOpen(!isOpen);
    };

    const onSelect = (
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        newValue: string | number | undefined
    ) => {
        onChange(newValue);
        setIsOpen(false);
    };

    const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
        <MenuToggle
            className={menuToggleClassName}
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
            isOpen={isOpen}
            selected={value}
            onSelect={onSelect}
            onOpenChange={(isOpen) => setIsOpen(isOpen)}
            toggle={toggle}
            shouldFocusToggleOnSelect
        >
            <SelectList aria-label={ariaLabelMenu}>{children}</SelectList>
        </Select>
    );
}

export default SimpleSelect;

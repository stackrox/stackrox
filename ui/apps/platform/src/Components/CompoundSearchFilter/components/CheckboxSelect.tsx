import React, { ReactElement } from 'react';
import {
    Select,
    SelectOption,
    SelectList,
    MenuToggle,
    MenuToggleElement,
    Badge,
} from '@patternfly/react-core';

type CheckboxSelectProps = {
    selection: string[];
    onChange: (value: string[]) => void;
    children: ReactElement<typeof SelectOption> | ReactElement<typeof SelectOption>[];
    isDisabled?: boolean;
    ariaLabelMenu?: string;
    toggleLabel?: string;
};

function CheckboxSelect({
    selection,
    onChange,
    children,
    isDisabled = false,
    ariaLabelMenu,
    toggleLabel,
}: CheckboxSelectProps) {
    const [isOpen, setIsOpen] = React.useState(false);

    const onToggleClick = () => {
        setIsOpen(!isOpen);
    };

    const onSelect = (
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined
    ) => {
        // @TODO: Consider what to do if the value is an invalid value
        if (selection.includes(String(value))) {
            onChange(selection.filter((id) => id !== value));
        } else {
            onChange([...selection, String(value)]);
        }
    };

    const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
        <MenuToggle
            aria-label={toggleLabel}
            ref={toggleRef}
            onClick={onToggleClick}
            isExpanded={isOpen}
            isDisabled={isDisabled}
        >
            {toggleLabel}
            {selection && selection.length > 0 && (
                <Badge className="pf-v5-u-ml-sm" isRead>
                    {selection.length}
                </Badge>
            )}
        </MenuToggle>
    );

    return (
        <Select
            role="menu"
            aria-label={ariaLabelMenu}
            isOpen={isOpen}
            selected={selection}
            onSelect={onSelect}
            onOpenChange={(nextOpen: boolean) => setIsOpen(nextOpen)}
            toggle={toggle}
            shouldFocusToggleOnSelect
        >
            <SelectList>{children}</SelectList>
        </Select>
    );
}

export default CheckboxSelect;

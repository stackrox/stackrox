import React, { ReactElement } from 'react';
import {
    Select,
    SelectOption,
    SelectList,
    MenuToggle,
    MenuToggleElement,
    Badge,
} from '@patternfly/react-core';
import { ensureString } from '../utils/utils';

type CheckboxSelectProps = {
    selection: string[];
    onChange: (checked: boolean, value: string) => void;
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
        event: React.MouseEvent<Element, MouseEvent> | undefined,
        value: string | number | undefined
    ) => {
        if (event) {
            const { checked } = event.target as HTMLInputElement;
            onChange(checked, ensureString(value));
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
            {selection && selection.length > 0 && <Badge isRead>{selection.length}</Badge>}
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

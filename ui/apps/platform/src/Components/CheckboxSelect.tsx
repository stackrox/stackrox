import React, { ReactElement } from 'react';
import { Select, SelectOption, MenuToggle, MenuToggleElement, Badge } from '@patternfly/react-core';

import { ensureString } from 'utils/ensure';

type CheckboxSelectProps = {
    selection: string[];
    onChange: (checked: boolean, value: string) => void;
    children: ReactElement<typeof SelectOption> | ReactElement<typeof SelectOption>[];
    isDisabled?: boolean;
    ariaLabelMenu?: string;
    toggleLabel?: string;
    toggleIcon?: React.ReactNode;
};

function CheckboxSelect({
    selection,
    onChange,
    children,
    isDisabled = false,
    ariaLabelMenu,
    toggleLabel,
    toggleIcon,
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
            icon={toggleIcon}
        >
            {toggleLabel}
            {selection && selection.length > 0 && (
                <Badge isRead className="pf-v5-u-ml-sm">
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
            {children}
        </Select>
    );
}

export default CheckboxSelect;

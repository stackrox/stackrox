import React, { ReactElement, useState } from 'react';
import {
    Select,
    MenuToggle,
    MenuToggleElement,
    SelectList,
    MenuFooter,
    SelectOptionProps,
} from '@patternfly/react-core';

export type SelectSingleProps = {
    toggleIcon?: ReactElement;
    toggleAriaLabel?: string;
    id: string;
    value: string;
    handleSelect: (name: string, value: string) => void;
    isDisabled?: boolean;
    children: ReactElement<SelectOptionProps>[];
    direction?: 'up' | 'down';
    placeholderText?: string;
    onBlur?: React.FocusEventHandler<HTMLDivElement>;
    menuAppendTo?: () => HTMLElement;
    footer?: React.ReactNode;
    maxHeight?: string;
};

function SelectSingle({
    toggleIcon,
    toggleAriaLabel,
    id,
    value,
    handleSelect,
    isDisabled = false,
    children,
    direction = 'down',
    placeholderText = '',
    onBlur,
    menuAppendTo = undefined,
    footer,
    maxHeight = '300px',
}: SelectSingleProps): ReactElement {
    const [isOpen, setIsOpen] = useState(false);

    function onSelect(
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        selection: string | number | undefined
    ) {
        if (typeof selection === 'string') {
            setIsOpen(false);
            handleSelect(id, selection);
        }
    }

    function onToggle() {
        setIsOpen(!isOpen);
    }

    // Find the display text for the selected value
    const getDisplayText = (): string => {
        if (!value) {
            return placeholderText;
        }

        const selectedChild = children.find((child) => {
            return child.props.value === value;
        });

        return (selectedChild?.props.children as string) || value;
    };

    const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            onClick={onToggle}
            isExpanded={isOpen}
            isDisabled={isDisabled}
            icon={toggleIcon}
            aria-label={toggleAriaLabel}
            id={id}
            variant="default"
            className="pf-v5-u-w-100"
        >
            {getDisplayText()}
        </MenuToggle>
    );

    return (
        <Select
            aria-label={toggleAriaLabel}
            isOpen={isOpen}
            selected={value}
            onSelect={onSelect}
            onOpenChange={(nextOpen: boolean) => setIsOpen(nextOpen)}
            toggle={toggle}
            shouldFocusToggleOnSelect
            popperProps={{
                appendTo: menuAppendTo,
                direction,
                minWidth: 'trigger',
            }}
            onBlur={onBlur}
        >
            <SelectList style={{ maxHeight, overflowY: 'auto' }}>{children}</SelectList>
            {footer && <MenuFooter>{footer}</MenuFooter>}
        </Select>
    );
}

export default SelectSingle;

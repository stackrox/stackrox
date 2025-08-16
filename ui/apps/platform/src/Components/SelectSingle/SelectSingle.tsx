import React, { ReactElement } from 'react';
import {
    Select,
    MenuToggle,
    MenuToggleElement,
    SelectList,
    MenuFooter,
    SelectOptionProps,
    MenuToggleProps,
} from '@patternfly/react-core';

import useSelectState from './useSelectState';

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
    maxWidth?: string;
    variant?: MenuToggleProps['variant'];
    className?: string;
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
    maxWidth = '30ch',
    variant = 'default',
    className,
}: SelectSingleProps): ReactElement {
    const { isOpen, setIsOpen, onSelect, onToggle } = useSelectState((selection) =>
        handleSelect(id, selection)
    );

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
            aria-label={toggleAriaLabel}
            id={id}
            variant={variant}
            className="pf-v5-u-w-100"
        >
            <span className="pf-v5-u-display-flex pf-v5-u-align-items-center">
                {toggleIcon && <span className="pf-v5-u-mr-sm">{toggleIcon}</span>}
                <span>{getDisplayText()}</span>
            </span>
        </MenuToggle>
    );

    return (
        <Select
            className={className}
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
            <SelectList style={{ maxHeight, maxWidth, overflowY: 'auto' }}>{children}</SelectList>
            {footer && <MenuFooter>{footer}</MenuFooter>}
        </Select>
    );
}

export default SelectSingle;

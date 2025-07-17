import React, { ReactElement, useState } from 'react';
import { Select, SelectList, MenuToggle, MenuToggleElement } from '@patternfly/react-core';

export type SelectSingleProps = {
    toggleIcon?: ReactElement;
    toggleAriaLabel?: string;
    id: string;
    value: string;
    handleSelect: (name: string, value: string) => void;
    isDisabled?: boolean;
    children: ReactElement[];
    direction?: 'up' | 'down';
    isCreatable?: boolean;
    variant?: 'typeahead' | null;
    placeholderText?: string;
    onBlur?: React.FocusEventHandler<HTMLButtonElement | HTMLDivElement | HTMLTextAreaElement>;
    menuAppendTo?: (() => HTMLElement) | 'inline' | 'parent';
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
    isCreatable: _isCreatable = false, // eslint-disable-line @typescript-eslint/no-unused-vars
    variant: _variant = null, // eslint-disable-line @typescript-eslint/no-unused-vars
    placeholderText = '',
    onBlur,
    menuAppendTo,
    footer,
    maxHeight: _maxHeight = '300px', // eslint-disable-line @typescript-eslint/no-unused-vars
}: SelectSingleProps): ReactElement {
    const [isOpen, setIsOpen] = useState(false);

    function onSelect(
        _event: React.MouseEvent<Element, MouseEvent> | undefined,
        selection: string | number | undefined
    ) {
        if (selection !== undefined) {
            setIsOpen(false);
            handleSelect(id, String(selection));
        }
    }

    function onToggleClick() {
        setIsOpen(!isOpen);
    }

    function onOpenChange(nextOpen: boolean) {
        setIsOpen(nextOpen);
    }

    // Find the display text for the selected value
    const getSelectedDisplayText = (): string => {
        if (!value) {
            return placeholderText;
        }

        // Find the matching SelectOption child and extract its text content
        const selectedOption = React.Children.toArray(children).find((child) => {
            return (
                React.isValidElement(child) &&
                child.props &&
                typeof child.props === 'object' &&
                'value' in child.props &&
                child.props.value === value
            );
        });

        if (selectedOption && React.isValidElement(selectedOption) && selectedOption.props) {
            // Return the text content of the SelectOption
            const childProps = selectedOption.props as { children?: React.ReactNode };
            return (typeof childProps.children === 'string' ? childProps.children : value) || value;
        }

        return value || placeholderText;
    };

    const toggle = (toggleRef: React.Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            onClick={onToggleClick}
            isExpanded={isOpen}
            isDisabled={isDisabled}
            icon={toggleIcon}
            aria-label={toggleAriaLabel}
            id={id}
            onBlur={onBlur}
        >
            {getSelectedDisplayText()}
        </MenuToggle>
    );

    return (
        <Select
            id={`${id}-select`}
            isOpen={isOpen}
            selected={value}
            onSelect={onSelect}
            onOpenChange={onOpenChange}
            toggle={toggle}
            isScrollable
            shouldFocusToggleOnSelect
            popperProps={{
                direction: direction === 'up' ? 'up' : 'down',
                appendTo: menuAppendTo === 'parent' ? undefined : menuAppendTo,
            }}
        >
            <SelectList>
                {children}
                {footer}
            </SelectList>
        </Select>
    );
}

export default SelectSingle;

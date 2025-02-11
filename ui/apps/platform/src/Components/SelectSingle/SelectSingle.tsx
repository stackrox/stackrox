import React, { ReactElement, useState } from 'react';
import { Select } from '@patternfly/react-core/deprecated';

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
    onBlur?: React.FocusEventHandler<HTMLTextAreaElement>;
    menuAppendTo?: (() => HTMLElement) | 'inline' | 'parent';
    footer?: React.ReactNode;
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
    isCreatable = false,
    variant = null,
    placeholderText = '',
    onBlur,
    menuAppendTo,
    footer,
}: SelectSingleProps): ReactElement {
    const [isOpen, setIsOpen] = useState(false);

    function onSelect(_event, selection) {
        // The mouse event is not useful.
        setIsOpen(false);
        handleSelect(id, selection);
    }

    return (
        <Select
            variant={variant === 'typeahead' ? 'typeahead' : 'single'}
            toggleIcon={toggleIcon}
            toggleAriaLabel={toggleAriaLabel}
            id={id}
            isDisabled={isDisabled}
            isOpen={isOpen}
            onSelect={onSelect}
            onToggle={(_event, val) => setIsOpen(val)}
            selections={value}
            direction={direction}
            isCreatable={isCreatable}
            placeholderText={placeholderText}
            toggleId={id}
            onBlur={onBlur}
            menuAppendTo={menuAppendTo}
            footer={footer}
        >
            {children}
        </Select>
    );
}

export default SelectSingle;

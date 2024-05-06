import React, { ReactElement, useState } from 'react';
import { Select, SelectVariant } from '@patternfly/react-core/deprecated';

export type SelectSingleProps = {
    className?: string;
    toggleIcon?: ReactElement;
    toggleAriaLabel?: string;
    ariaLabel?: string;
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
    menuAppendTo?: () => HTMLElement;
    footer?: React.ReactNode;
};

function SelectSingle({
    className,
    toggleIcon,
    toggleAriaLabel,
    ariaLabel,
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

    const isTypeahead = variant === 'typeahead' ? SelectVariant.typeahead : SelectVariant.single;

    function onSelect(_event, selection) {
        // The mouse event is not useful.
        setIsOpen(false);
        handleSelect(id, selection);
    }

    return (
        <Select
            variant={isTypeahead}
            toggleIcon={toggleIcon}
            toggleAriaLabel={toggleAriaLabel}
            aria-label={ariaLabel}
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
            className={className}
        >
            {children}
        </Select>
    );
}

export default SelectSingle;

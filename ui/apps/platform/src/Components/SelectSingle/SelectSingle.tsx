import React, { ReactElement, useState } from 'react';
import { Select, SelectVariant } from '@patternfly/react-core';

export type SelectSingleProps = {
    toggleIcon?: ReactElement;
    id: string;
    value: string;
    handleSelect: (name: string, value: string) => void;
    isDisabled?: boolean;
    children: ReactElement[];
    direction?: 'up' | 'down';
    isCreatable?: boolean;
    variant?: 'typeahead' | null;
    placeholderText?: string;
};

function SelectSingle({
    toggleIcon,
    id,
    value,
    handleSelect,
    isDisabled = false,
    children,
    direction = 'down',
    isCreatable = false,
    variant = null,
    placeholderText = '',
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
            id={id}
            isDisabled={isDisabled}
            isOpen={isOpen}
            onSelect={onSelect}
            onToggle={setIsOpen}
            selections={value}
            direction={direction}
            isCreatable={isCreatable}
            placeholderText={placeholderText}
            toggleId={id}
        >
            {children}
        </Select>
    );
}

export default SelectSingle;

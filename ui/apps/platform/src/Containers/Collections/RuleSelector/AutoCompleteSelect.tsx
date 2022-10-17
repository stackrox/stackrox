import React from 'react';
import { Select, ValidatedOptions } from '@patternfly/react-core';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';

export type AutoCompleteSelectProps = {
    selectedOption: string;
    className?: string;
    typeAheadAriaLabel?: string;
    onChange: (value: string) => void;
    validated: ValidatedOptions;
};

/* TODO Implement autocompletion */
export function AutoCompleteSelect({
    selectedOption,
    className = '',
    typeAheadAriaLabel,
    onChange,
    validated,
}: AutoCompleteSelectProps) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();

    function onSelect(_, value) {
        onChange(value);
        closeSelect();
    }

    return (
        <>
            <Select
                validated={validated}
                typeAheadAriaLabel={typeAheadAriaLabel}
                className={className}
                variant="typeahead"
                isCreatable
                isOpen={isOpen}
                onToggle={onToggle}
                selections={selectedOption}
                onSelect={onSelect}
            />
        </>
    );
}

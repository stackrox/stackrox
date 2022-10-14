import React from 'react';
import { Select } from '@patternfly/react-core';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';

export type AutoCompleteSelectorProps = {
    selectedOption: string;
    className?: string;
    onChange: (value: string) => void;
};

/* TODO Implement autocompletion */
export function AutoCompleteSelector({
    selectedOption,
    className = '',
    onChange,
}: AutoCompleteSelectorProps) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();

    function onSelect(_, value) {
        onChange(value);
        closeSelect();
    }

    return (
        <>
            <Select
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

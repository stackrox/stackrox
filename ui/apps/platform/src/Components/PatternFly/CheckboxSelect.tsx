import React, { ReactElement, useState } from 'react';
import {
    Select,
    SelectOptionObject,
    SelectOptionProps,
    SelectVariant,
} from '@patternfly/react-core';

export type CheckboxSelectProps = {
    id?: string;
    name?: string;
    selections: string[];
    onChange: (selection: string[]) => void;
    onBlur?: React.FocusEventHandler<HTMLTextAreaElement>;
    ariaLabel: string;
    children: ReactElement<SelectOptionProps>[];
    placeholderText?: string;
    toggleIcon?: ReactElement;
    toggleId?: string;
    menuAppendTo?: () => HTMLElement;
};

function CheckboxSelect({
    id,
    name,
    selections,
    onChange,
    onBlur,
    ariaLabel,
    children,
    placeholderText = 'Filter by value',
    toggleIcon,
    toggleId,
    menuAppendTo,
}: CheckboxSelectProps): ReactElement {
    const [isOpen, setIsOpen] = useState(false);

    function onToggle(isExpanded: boolean) {
        setIsOpen(isExpanded);
    }

    function onSelect(
        event: React.MouseEvent | React.ChangeEvent,
        selection: string | SelectOptionObject
    ) {
        if (typeof selection !== 'string' || !selections || !onChange) {
            return;
        }
        if (selections.includes(selection)) {
            onChange(selections.filter((item) => item !== selection));
        } else {
            onChange([...selections, selection]);
        }
    }

    return (
        <Select
            id={id}
            name={name}
            variant={SelectVariant.checkbox}
            toggleIcon={toggleIcon}
            onToggle={onToggle}
            onSelect={onSelect}
            onBlur={onBlur}
            selections={selections}
            isOpen={isOpen}
            placeholderText={placeholderText}
            aria-label={ariaLabel}
            toggleId={toggleId}
            menuAppendTo={menuAppendTo}
        >
            {children}
        </Select>
    );
}

export default CheckboxSelect;

import React, { ReactElement, useState } from 'react';
import {
    Select,
    SelectOptionObject,
    SelectOptionProps,
    SelectVariant,
} from '@patternfly/react-core';

export type CheckboxSelectProps = {
    selections: string[];
    onChange: (selection: string[]) => void;
    ariaLabel: string;
    children: ReactElement<SelectOptionProps>[];
};

function CheckboxSelect({
    selections,
    onChange,
    ariaLabel,
    children,
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
            variant={SelectVariant.checkbox}
            onToggle={onToggle}
            onSelect={onSelect}
            selections={selections}
            isOpen={isOpen}
            placeholderText="Filter by value"
            aria-label={ariaLabel}
        >
            {children}
        </Select>
    );
}

export default CheckboxSelect;

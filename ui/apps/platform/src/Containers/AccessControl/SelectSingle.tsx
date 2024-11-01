import React, { ReactElement, useState } from 'react';
import { Select } from '@patternfly/react-core/deprecated';

export type SelectSingleProps = {
    id: string;
    value: string;
    setFieldValue: (name: string, value: string) => void;
    isDisabled: boolean;
    children: ReactElement[];
};

function SelectSingle({
    id,
    value,
    setFieldValue,
    isDisabled,
    children,
}: SelectSingleProps): ReactElement {
    const [isOpen, setIsOpen] = useState(false);

    function onSelect(_event, selection) {
        // The mouse event is not useful.
        setIsOpen(false);
        setFieldValue(id, selection);
    }

    return (
        <Select
            variant="single"
            id={id}
            isDisabled={isDisabled}
            isOpen={isOpen}
            onSelect={onSelect}
            onToggle={(_event, val) => setIsOpen(val)}
            selections={value}
        >
            {children}
        </Select>
    );
}

export default SelectSingle;

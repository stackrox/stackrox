import React, { ReactElement, useState } from 'react';
import { Select, SelectVariant } from '@patternfly/react-core';

export type FormSelectProps = {
    id: string;
    value: string;
    onChange: (selection, id) => void;
    children: ReactElement[];
};

const FormSelect = ({ id, value, onChange, children }: FormSelectProps): ReactElement => {
    const [isOpen, setIsOpen] = useState(false);

    function onToggle(toggleOpen) {
        setIsOpen(toggleOpen);
    }

    function onSelect(event, selection) {
        onChange(selection, id);
        setIsOpen(false);
    }

    return (
        <Select
            id={id}
            variant={SelectVariant.single}
            selections={value}
            onToggle={onToggle}
            onSelect={onSelect}
            isOpen={isOpen}
            placeholderText="UNSET"
        >
            {children}
        </Select>
    );
};

export default FormSelect;

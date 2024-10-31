import React, { ReactElement, useState } from 'react';
import { Select } from '@patternfly/react-core/deprecated';

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
            variant="single"
            selections={value}
            onToggle={(_event, toggleOpen) => onToggle(toggleOpen)}
            onSelect={onSelect}
            isOpen={isOpen}
            placeholderText="UNSET"
        >
            {children}
        </Select>
    );
};

export default FormSelect;

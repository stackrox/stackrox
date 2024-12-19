/* PatternFly Component */
import React, { ReactElement, useState } from 'react';
import { Select } from '@patternfly/react-core/deprecated';

export type FormMultiSelectProps = {
    id: string;
    values: string[];
    onChange: (id: string, selection: string[]) => void;
    children: ReactElement[];
    isDisabled?: boolean;
};

const FormMultiSelect = ({
    id,
    values = [],
    onChange,
    children,
    isDisabled = false,
}: FormMultiSelectProps): ReactElement => {
    const [isOpen, setIsOpen] = useState(false);

    function onToggle(toggleOpen) {
        setIsOpen(toggleOpen);
    }

    function onSelect(_event, selection) {
        if (values.includes(selection)) {
            const newSelection = values.filter((item) => item !== selection);
            onChange(id, newSelection);
            setIsOpen(false);
        } else {
            const newSelection = [...values, selection];
            onChange(id, newSelection);
            setIsOpen(false);
        }
    }

    function onClearHandler() {
        onChange(id, []);
    }

    return (
        <Select
            id={id}
            variant="typeaheadmulti"
            selections={values}
            onToggle={(_event, toggleOpen) => onToggle(toggleOpen)}
            onSelect={onSelect}
            onClear={onClearHandler}
            isOpen={isOpen}
            isDisabled={isDisabled}
            toggleId={id}
        >
            {children}
        </Select>
    );
};

export default FormMultiSelect;

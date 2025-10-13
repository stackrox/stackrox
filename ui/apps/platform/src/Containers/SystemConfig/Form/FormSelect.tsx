import React, { ReactElement } from 'react';
import { SelectOptionProps } from '@patternfly/react-core';

import SelectSingle from 'Components/SelectSingle';

export type FormSelectProps = {
    id: string;
    value: string;
    onChange: (selection, id) => void;
    children: ReactElement<SelectOptionProps>[];
};

const FormSelect = ({ id, value, onChange, children }: FormSelectProps): ReactElement => {
    function handleSelect(fieldId: string, selection: string) {
        onChange(selection, fieldId);
    }

    return (
        <SelectSingle id={id} value={value} handleSelect={handleSelect} placeholderText="UNSET">
            {children}
        </SelectSingle>
    );
};

export default FormSelect;

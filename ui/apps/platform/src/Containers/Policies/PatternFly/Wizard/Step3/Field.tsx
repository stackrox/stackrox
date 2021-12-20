import React from 'react';
import { useField } from 'formik';
import { TextInput, Checkbox } from '@patternfly/react-core';

import { Descriptor, SubComponent } from 'Containers/Policies/Wizard/Form/descriptors';

type FieldProps = {
    descriptor: Descriptor | SubComponent;
    readOnly: boolean;
    name: string;
};

function Field({ descriptor, readOnly, name }: FieldProps) {
    const [field, meta] = useField(name);
    if (field === undefined) {
        return null;
    }
    const { value } = field;
    console.log('Field value', field, name);

    // this is to accomodate for recursive Fields (when type is 'group')
    // const path = descriptor.subpath ? name : `${name}.value`;

    function handleChangeValue(_, e) {
        field.onChange(e);
    }

    // function

    switch (descriptor.type) {
        case 'text':
            return (
                <TextInput
                    value={value.value}
                    type="text"
                    isDisabled={readOnly}
                    onChange={handleChangeValue}
                />
            );
        case 'checkbox':
            return (
                <Checkbox
                    isChecked={value.value}
                    id={name}
                    isDisabled={readOnly}
                    onChange={handleChangeValue}
                />
            );
        case 'number':
            return (
                <TextInput
                    value={value.value}
                    type="number"
                    isDisabled={readOnly}
                    onChange={handleChangeValue}
                    placeholder={descriptor.placeholder}
                />
            );
        default:
            throw new Error(`Unknown field type: ${descriptor.type}`);
    }
}

export default Field;

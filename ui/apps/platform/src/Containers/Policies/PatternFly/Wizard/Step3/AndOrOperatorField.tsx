import React from 'react';
import { useField } from 'formik';
import { Button } from '@patternfly/react-core';

type AndOrOperatorFieldProps = {
    name: string;
    readOnly?: boolean;
};

function AndOrOperatorField({ name, readOnly = false }: AndOrOperatorFieldProps) {
    const [field, , helpers] = useField(name);

    function handleBooleanOperator(e) {
        // const newBooleanValue = group.booleanOperator === 'AND' ? 'OR' : 'AND';
        // helpers.setValue({})
        console.log(field.value);
    }

    return (
        <Button variant="plain" onClick={handleBooleanOperator} isDisabled={readOnly}>
            — and —
        </Button>
    );
}

export default AndOrOperatorField;

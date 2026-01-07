import { useField } from 'formik';
import { Button } from '@patternfly/react-core';

interface AndOrOperatorFieldProps {
    name: string;
    readOnly?: boolean;
}

function AndOrOperatorField({ name, readOnly = false }: AndOrOperatorFieldProps) {
    const [field, , helpers] = useField(name);

    function handleBooleanOperator() {
        const newBooleanValue = field.value.booleanOperator === 'AND' ? 'OR' : 'AND';
        helpers.setValue({ ...field.value, booleanOperator: newBooleanValue });
    }

    return (
        <Button
            icon={<>— {field.value.booleanOperator.toLowerCase()} —</>}
            variant="plain"
            onClick={handleBooleanOperator}
            isDisabled={readOnly}
            data-testid="policy-criteria-boolean-operator"
        />
    );
}

export default AndOrOperatorField;

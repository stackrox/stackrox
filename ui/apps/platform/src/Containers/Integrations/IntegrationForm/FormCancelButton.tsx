import React, { ReactElement } from 'react';
import { Button } from '@patternfly/react-core';

export type FormCancelButtonProps = {
    children: ReactElement | ReactElement[] | string;
    onCancel: () => void;
};

function FormCancelButton({ children, onCancel }: FormCancelButtonProps): ReactElement {
    return (
        <Button variant="link" onClick={onCancel}>
            {children}
        </Button>
    );
}

export default FormCancelButton;

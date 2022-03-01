import React, { ReactElement } from 'react';
import { Button } from '@patternfly/react-core';

export type FormCancelButtonProps = {
    children: ReactElement | ReactElement[] | string;
    onCancel: () => void;
    isDisabled?: boolean;
};

function FormCancelButton({
    children,
    onCancel,
    isDisabled = false,
}: FormCancelButtonProps): ReactElement {
    return (
        <Button variant="link" onClick={onCancel} isDisabled={isDisabled}>
            {children}
        </Button>
    );
}

export default FormCancelButton;

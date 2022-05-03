import React, { ReactElement } from 'react';
import { Button } from '@patternfly/react-core';

export type FormSaveButtonProps = {
    children: ReactElement | ReactElement[] | string;
    onSave: () => void;
    isSubmitting: boolean;
    isTesting: boolean;
    isDisabled?: boolean;
    type?: 'button' | 'submit' | 'reset';
};

function FormSaveButton({
    children,
    onSave,
    isSubmitting,
    isTesting,
    isDisabled = false,
    type = 'button',
}: FormSaveButtonProps): ReactElement {
    return (
        <Button
            type={type}
            variant="primary"
            onClick={onSave}
            data-testid="create-btn"
            isDisabled={isDisabled || isSubmitting}
            isLoading={isSubmitting && !isTesting}
        >
            {children}
        </Button>
    );
}

export default FormSaveButton;

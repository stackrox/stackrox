import React, { ReactElement } from 'react';
import { Button } from '@patternfly/react-core';

export type FormSaveButtonProps = {
    children: ReactElement | ReactElement[] | string;
    onSave: () => void;
    isSubmitting: boolean;
    isTesting: boolean;
    isDisabled?: boolean;
};

function FormSaveButton({
    children,
    onSave,
    isSubmitting,
    isTesting,
    isDisabled = false,
}: FormSaveButtonProps): ReactElement {
    return (
        <Button
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

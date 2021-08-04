import React, { ReactElement } from 'react';
import { Button } from '@patternfly/react-core';

export type FormTestButtonProps = {
    children: ReactElement | ReactElement[] | string;
    onTest: () => void;
    isValid?: boolean;
    isSubmitting: boolean;
    isTesting: boolean;
};

function FormTestButton({
    children,
    onTest,
    isValid = false,
    isSubmitting,
    isTesting,
}: FormTestButtonProps): ReactElement {
    return (
        <Button
            variant="secondary"
            onClick={onTest}
            data-testid="test-btn"
            isDisabled={isSubmitting || isTesting || !isValid}
            isLoading={isSubmitting && isTesting}
        >
            {children}
        </Button>
    );
}

export default FormTestButton;

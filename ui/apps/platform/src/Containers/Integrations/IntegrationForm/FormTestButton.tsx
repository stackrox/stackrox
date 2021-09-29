import React, { ReactElement } from 'react';
import { Button } from '@patternfly/react-core';

export type FormTestButtonProps = {
    children: ReactElement | ReactElement[] | string;
    onTest: () => void;
    isSubmitting: boolean;
    isTesting: boolean;
    isDisabled?: boolean;
};

function FormTestButton({
    children,
    onTest,
    isSubmitting,
    isTesting,
    isDisabled = false,
}: FormTestButtonProps): ReactElement {
    return (
        <Button
            variant="secondary"
            onClick={onTest}
            data-testid="test-btn"
            isDisabled={isDisabled || isSubmitting}
            isLoading={isSubmitting && isTesting}
        >
            {children}
        </Button>
    );
}

export default FormTestButton;

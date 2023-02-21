import React, { ReactElement } from 'react';
import { Alert, AlertVariant } from '@patternfly/react-core';

export type FormResponseMessage = {
    message: string;
    isError: boolean;
} | null;

export type FormMessageProps = {
    message: FormResponseMessage;
};

function FormMessage({ message }: FormMessageProps): ReactElement {
    const title = message?.isError ? 'Failure' : 'Success';
    const variant = message?.isError ? AlertVariant.danger : AlertVariant.success;
    return (
        <div id="form-message-alert">
            {message && (
                <Alert className="pf-u-mt-md pf-u-mb-md" title={title} variant={variant} isInline>
                    <p>{message?.message}</p>
                </Alert>
            )}
        </div>
    );
}

export default FormMessage;

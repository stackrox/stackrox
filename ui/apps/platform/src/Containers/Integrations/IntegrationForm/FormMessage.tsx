import React, { ReactElement } from 'react';
import { Alert, AlertVariant } from '@patternfly/react-core';

import { FormResponseMessage } from './useIntegrationForm';

export type FormMessageProps = {
    message: FormResponseMessage;
};

function FormMessage({ message }: FormMessageProps): ReactElement {
    const title = message?.isError ? 'Could not save the integration' : 'Success';
    const variant = message?.isError ? AlertVariant.danger : AlertVariant.success;
    return (
        <Alert className="pf-u-mt-md pf-u-mb-md" title={title} variant={variant} isInline>
            <p>{message?.message}</p>
        </Alert>
    );
}

export default FormMessage;

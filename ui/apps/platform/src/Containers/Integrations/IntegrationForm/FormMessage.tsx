import React, { ReactElement } from 'react';
import { Alert, AlertVariant } from '@patternfly/react-core';

import { FormResponseMessage } from './useIntegrationForm';

export type FormMessageProps = {
    message: FormResponseMessage;
};

function FormMessage({ message }: FormMessageProps): ReactElement {
    return (
        <Alert
            className="pf-u-mt-md pf-u-mb-md"
            title="Could not save the integration"
            variant={AlertVariant.danger}
            isInline
        >
            <p>{message?.message}</p>
        </Alert>
    );
}

export default FormMessage;

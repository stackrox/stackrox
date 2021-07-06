import React, { ReactElement } from 'react';
import { Alert, AlertVariant } from '@patternfly/react-core';

import AccessControlHeading from './AccessControlHeading';

function AccessControlNoPermission(): ReactElement {
    return (
        <>
            <AccessControlHeading />
            <Alert
                title="You do not have permission to view Access Control"
                variant={AlertVariant.info}
                isInline
            />
        </>
    );
}

export default AccessControlNoPermission;

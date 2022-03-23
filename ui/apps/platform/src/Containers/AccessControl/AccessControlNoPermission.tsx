import React, { ReactElement } from 'react';
import { Alert, AlertVariant, PageSection, PageSectionVariants } from '@patternfly/react-core';

import AccessControlHeading from './AccessControlHeading';

function AccessControlNoPermission(): ReactElement {
    return (
        <>
            <AccessControlHeading isNavHidden />
            <PageSection variant={PageSectionVariants.light}>
                <Alert
                    className="pf-u-mt-md"
                    title="You do not have permission to view Access Control"
                    variant={AlertVariant.info}
                    isInline
                />
            </PageSection>
        </>
    );
}

export default AccessControlNoPermission;

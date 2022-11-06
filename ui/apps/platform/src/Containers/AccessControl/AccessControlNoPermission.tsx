import React, { ReactElement } from 'react';
import { Alert, AlertVariant, PageSection, PageSectionVariants } from '@patternfly/react-core';

import AccessControlHeading from './AccessControlHeading';
import { AccessControlEntityType } from '../../constants/entityTypes';

type AccessControlNoPermissionProps = {
    subPage: string;
    entityType: AccessControlEntityType;
};

function AccessControlNoPermission({
    subPage,
    entityType,
}: AccessControlNoPermissionProps): ReactElement {
    return (
        <>
            <AccessControlHeading entityType={entityType} />
            <PageSection variant={PageSectionVariants.light}>
                <Alert
                    className="pf-u-mt-md"
                    title={`You do not have permission to view ${subPage}`}
                    variant={AlertVariant.info}
                    isInline
                />
            </PageSection>
        </>
    );
}

export default AccessControlNoPermission;

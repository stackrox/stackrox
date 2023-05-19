import React, { ReactElement } from 'react';
import { Alert, AlertVariant, PageSection, PageSectionVariants } from '@patternfly/react-core';

import AccessControlHeading from './AccessControlHeading';
import { AccessControlEntityType } from '../../constants/entityTypes';

type AccessControlNoPermissionProps = {
    subPage: string;
    entityType?: AccessControlEntityType;
    isNavHidden?: boolean;
};

function AccessControlNoPermission({
    subPage,
    entityType,
    isNavHidden = false,
}: AccessControlNoPermissionProps): ReactElement {
    return (
        <>
            <AccessControlHeading isNavHidden={isNavHidden} entityType={entityType} />
            <PageSection variant={PageSectionVariants.light}>
                <Alert
                    className="pf-u-mt-md"
                    title={`You do not have permission to view ${subPage}. To access this page, you should have READ permission to the Access resource.`}
                    variant={AlertVariant.info}
                    isInline
                />
            </PageSection>
        </>
    );
}

export default AccessControlNoPermission;

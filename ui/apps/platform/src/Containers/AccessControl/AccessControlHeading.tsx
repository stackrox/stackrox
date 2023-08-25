import React, { ReactElement } from 'react';
import { Divider, PageSection, Title } from '@patternfly/react-core';

import { AccessControlEntityType } from 'constants/entityTypes';
import AccessControlNav from './AccessControlNav';

export type AccessControlHeadingProps = {
    /** The AccessControl Entity managed on this page, used to highlight the current navigation item. */
    entityType?: AccessControlEntityType;
    /** Whether or not to hide the tab navigation component */
    isNavHidden?: boolean;
};

/**
 * Render title h1 and tab navigation at top of page.
 */
function AccessControlHeading({
    entityType,
    isNavHidden = false,
}: AccessControlHeadingProps): ReactElement {
    return (
        <>
            <PageSection variant="light">
                <Title headingLevel="h1">Access Control</Title>
            </PageSection>
            {isNavHidden || (
                <PageSection variant="light" className="pf-u-px-sm pf-u-py-0">
                    <AccessControlNav entityType={entityType} />
                </PageSection>
            )}
            <Divider component="div" />
        </>
    );
}

export default AccessControlHeading;

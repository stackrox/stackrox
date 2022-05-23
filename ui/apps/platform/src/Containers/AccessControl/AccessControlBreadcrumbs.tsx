import React, { ReactElement } from 'react';
import { Breadcrumb, BreadcrumbItem, Divider, PageSection } from '@patternfly/react-core';
import pluralize from 'pluralize';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { AccessControlEntityType } from 'constants/entityTypes';
import { accessControlLabels } from 'messages/common';

import { getEntityPath } from './accessControlPaths';

export type AccessControlBreadcrumbsProps = {
    entityType: AccessControlEntityType;
    entityName?: string;
};

function AccessControlBreadcrumbs({
    entityType,
    entityName,
}: AccessControlBreadcrumbsProps): ReactElement {
    const entityTypeLabel = entityType ? pluralize(accessControlLabels[entityType]) : null;

    return (
        <>
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={getEntityPath(entityType)}>
                        {entityTypeLabel}
                    </BreadcrumbItemLink>
                    {entityName && <BreadcrumbItem>{entityName}</BreadcrumbItem>}
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
        </>
    );
}

export default AccessControlBreadcrumbs;

import type { ReactElement } from 'react';
import { Breadcrumb, BreadcrumbItem, PageSection } from '@patternfly/react-core';
import pluralize from 'pluralize';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import type { AccessControlEntityType } from 'constants/entityTypes';
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
        <PageSection type="breadcrumb">
            <Breadcrumb>
                <BreadcrumbItemLink to={getEntityPath(entityType)}>
                    {entityTypeLabel}
                </BreadcrumbItemLink>
                {entityName && <BreadcrumbItem>{entityName}</BreadcrumbItem>}
            </Breadcrumb>
        </PageSection>
    );
}

export default AccessControlBreadcrumbs;

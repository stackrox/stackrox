import React, { ReactElement } from 'react';
import { Breadcrumb, BreadcrumbItem, Divider, PageSection } from '@patternfly/react-core';
import pluralize from 'pluralize';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { AccessControlEntityType } from 'constants/entityTypes';
import { accessControlLabels } from 'messages/common';

import { getEntityPath } from './accessControlPaths';

export type AccessControlBreadcrumbsProps = {
    entityType?: AccessControlEntityType;
    entityName?: string;
    isDisabled?: boolean;
    isList?: boolean;
};

function AccessControlBreadcrumbs({
    entityType,
    entityName,
    isDisabled,
    isList,
}: AccessControlBreadcrumbsProps): ReactElement {
    let entityTypeBreadcrumb;
    const entityTypeLabel = entityType ? pluralize(accessControlLabels[entityType]) : null;
    if (entityType) {
        entityTypeBreadcrumb =
            isDisabled || isList ? (
                <BreadcrumbItem isActive>{entityTypeLabel}</BreadcrumbItem>
            ) : (
                <BreadcrumbItemLink to={getEntityPath(entityType)}>
                    {entityTypeLabel}
                </BreadcrumbItemLink>
            );
    }

    return (
        <>
            <PageSection variant="light">
                <Breadcrumb>
                    <BreadcrumbItem isActive>Access Control</BreadcrumbItem>
                    {entityTypeBreadcrumb}
                    {entityName && <BreadcrumbItem isActive>{entityName}</BreadcrumbItem>}
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
        </>
    );
}

export default AccessControlBreadcrumbs;

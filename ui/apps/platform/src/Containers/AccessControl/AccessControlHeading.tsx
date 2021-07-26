import React, { ReactElement } from 'react';
import { Breadcrumb, BreadcrumbItem, Title } from '@patternfly/react-core';
import pluralize from 'pluralize';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { AccessControlEntityType } from 'constants/entityTypes';
import { accessControlLabels } from 'messages/common';

import { getEntityPath } from './accessControlPaths';

export type AccessControlHeadingProps = {
    entityType?: AccessControlEntityType;
    entityName?: string;
    isDisabled?: boolean;
};

/*
 * Render breadcrumb and title h1 at the top of Access Control page.
 *
 * The isActive prop renders a breadcrumb item as text.
 * BreadcrumbItemLink renders a React Router link.
 */
function AccessControlHeading({
    entityType,
    entityName,
    isDisabled,
}: AccessControlHeadingProps): ReactElement {
    let entityTypeBreadcrumb;
    if (entityType) {
        const entityTypeLabel = pluralize(accessControlLabels[entityType]);
        entityTypeBreadcrumb =
            isDisabled || !entityName ? (
                <BreadcrumbItem isActive>{entityTypeLabel}</BreadcrumbItem>
            ) : (
                <BreadcrumbItemLink to={getEntityPath(entityType)}>
                    {entityTypeLabel}
                </BreadcrumbItemLink>
            );
    }

    return (
        <>
            <Breadcrumb>
                <BreadcrumbItem isActive>Access Control</BreadcrumbItem>
                {entityTypeBreadcrumb}
                {entityName && <BreadcrumbItem isActive>{entityName}</BreadcrumbItem>}
            </Breadcrumb>
            <Title headingLevel="h1">Access Control</Title>
        </>
    );
}

export default AccessControlHeading;

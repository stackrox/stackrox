import React, { ReactElement } from 'react';
import { ButtonVariant } from '@patternfly/react-core';
import pluralize from 'pluralize';

import ButtonLink from 'Components/PatternFly/ButtonLink';
import { AccessControlEntityType } from 'constants/entityTypes';
import { Role } from 'services/RolesService';

import { getEntityPath } from './accessControlPaths';

export type AccessControlEntityLinkProps = {
    entityType: AccessControlEntityType;
    entityId: string;
    entityName: string;
};

export function AccessControlEntityLink({
    entityType,
    entityId,
    entityName,
}: AccessControlEntityLinkProps): ReactElement {
    return (
        <ButtonLink variant={ButtonVariant.link} isInline to={getEntityPath(entityType, entityId)}>
            {entityName}
        </ButtonLink>
    );
}

export type RolesLinkProps = {
    roles: Role[];
    entityType: AccessControlEntityType;
    entityId: string;
};

export function RolesLink({ roles, entityType, entityId }: RolesLinkProps): ReactElement {
    if (roles.length === 0) {
        return <span>No roles</span>;
    }

    if (roles.length === 1) {
        const { name } = roles[0];
        // The name is the id for a role.
        return <AccessControlEntityLink entityType="ROLE" entityId={name} entityName={name} />;
    }

    const count = roles.length;
    const url = getEntityPath('ROLE', '', { s: { [entityType]: entityId } });
    const text = `${count} ${pluralize('role', count)}`;
    return (
        <ButtonLink variant={ButtonVariant.link} isInline to={url}>
            {text}
        </ButtonLink>
    );
}

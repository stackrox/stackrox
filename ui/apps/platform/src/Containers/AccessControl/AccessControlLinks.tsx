import React, { ReactElement } from 'react';
import { Button, ButtonVariant } from '@patternfly/react-core';
import pluralize from 'pluralize';

import LinkShim from 'Components/PatternFly/LinkShim';
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
        <Button
            variant={ButtonVariant.link}
            isInline
            component={LinkShim}
            href={getEntityPath(entityType, entityId)}
        >
            {entityName}
        </Button>
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
        <Button variant={ButtonVariant.link} isInline component={LinkShim} href={url}>
            {text}
        </Button>
    );
}

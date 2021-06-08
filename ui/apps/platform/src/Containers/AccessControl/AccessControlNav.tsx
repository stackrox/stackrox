import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { Nav, NavItem, NavList } from '@patternfly/react-core';
import pluralize from 'pluralize';

import { AccessControlEntityType } from 'constants/entityTypes';
import { accessControlLabels } from 'messages/common';

import { entityPathSegment, getEntityPath } from './accessControlPaths';

export type AccessControlNavProps = {
    entityType?: AccessControlEntityType;
};

function AccessControlNav({ entityType }: AccessControlNavProps): ReactElement {
    return (
        <Nav variant="tertiary">
            <NavList>
                {Object.entries(entityPathSegment).map(([itemType, itemSegment]) => (
                    <NavItem key={itemSegment} isActive={itemType === entityType}>
                        <Link to={getEntityPath(itemType as AccessControlEntityType)}>
                            {pluralize(accessControlLabels[itemType])}
                        </Link>
                    </NavItem>
                ))}
            </NavList>
        </Nav>
    );
}

export default AccessControlNav;

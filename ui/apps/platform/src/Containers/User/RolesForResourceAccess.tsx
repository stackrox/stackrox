import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { Button, Flex } from '@patternfly/react-core';
import { CheckIcon, TimesIcon } from '@patternfly/react-icons';

import { userPath } from 'routePaths';

const forbiddenIcon = (
    <TimesIcon aria-label="forbidden" color="var(--pf-global--danger-color--100)" size="sm" />
);
const permittedIcon = (
    <CheckIcon aria-label="permitted" color="var(--pf-global--success-color--100)" size="sm" />
);

const getUserRolePath = (roleName: string) => `${userPath}/roles/${roleName}`;

export type RolesForResourceAccessProps = {
    roleNames: string[];
};

function RolesForResourceAccess({ roleNames }: RolesForResourceAccessProps): ReactElement {
    if (roleNames.length === 0) {
        return (
            <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                {forbiddenIcon}
                <span className="pf-u-text-nowrap">No roles</span>
            </Flex>
        );
    }

    return (
        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
            {permittedIcon}
            {roleNames.map((roleName) => (
                <Button key={roleName} variant="link" isInline>
                    <Link to={getUserRolePath(roleName)}>{roleName}</Link>
                </Button>
            ))}
        </Flex>
    );
}

export default RolesForResourceAccess;

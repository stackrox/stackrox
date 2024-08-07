import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { Flex, Icon } from '@patternfly/react-core';
import { CheckIcon, TimesIcon } from '@patternfly/react-icons';

import { userBasePath } from 'routePaths';

const forbiddenIcon = (
    <Icon size="sm">
        <TimesIcon color="var(--pf-v5-global--danger-color--100)" aria-label="forbidden" />
    </Icon>
);
const permittedIcon = (
    <Icon size="sm">
        <CheckIcon color="var(--pf-v5-global--success-color--100)" aria-label="permitted" />
    </Icon>
);

const getUserRolePath = (roleName: string) => `${userBasePath}/roles/${roleName}`;

export type RolesForResourceAccessProps = {
    roleNames: string[];
};

function RolesForResourceAccess({ roleNames }: RolesForResourceAccessProps): ReactElement {
    if (roleNames.length === 0) {
        return (
            <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                {forbiddenIcon}
                <span className="pf-v5-u-text-nowrap">No roles</span>
            </Flex>
        );
    }

    return (
        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
            {permittedIcon}
            {roleNames.map((roleName) => (
                <Link key={roleName} to={getUserRolePath(roleName)}>
                    {roleName}
                </Link>
            ))}
        </Flex>
    );
}

export default RolesForResourceAccess;

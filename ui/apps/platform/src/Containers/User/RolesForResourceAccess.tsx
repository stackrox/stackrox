import React, { ReactElement } from 'react';
import { ButtonVariant, Flex } from '@patternfly/react-core';
import { CheckIcon, TimesIcon } from '@patternfly/react-icons';

import ButtonLink from 'Components/PatternFly/ButtonLink';
import { userBasePath } from 'routePaths';

const forbiddenIcon = (
    <TimesIcon aria-label="forbidden" color="var(--pf-global--danger-color--100)" size="sm" />
);
const permittedIcon = (
    <CheckIcon aria-label="permitted" color="var(--pf-global--success-color--100)" size="sm" />
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
                <span className="pf-u-text-nowrap">No roles</span>
            </Flex>
        );
    }

    return (
        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
            {permittedIcon}
            {roleNames.map((roleName) => (
                <ButtonLink
                    key={roleName}
                    variant={ButtonVariant.link}
                    isInline
                    to={getUserRolePath(roleName)}
                >
                    {roleName}
                </ButtonLink>
            ))}
        </Flex>
    );
}

export default RolesForResourceAccess;

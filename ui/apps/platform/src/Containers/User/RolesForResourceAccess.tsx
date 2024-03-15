import React, { ReactElement } from 'react';
import { Button, ButtonVariant, Flex, Icon } from '@patternfly/react-core';
import { CheckIcon, TimesIcon } from '@patternfly/react-icons';

import LinkShim from 'Components/PatternFly/LinkShim';
import { userBasePath } from 'routePaths';

const forbiddenIcon = (
    <Icon color="var(--pf-global--danger-color--100)" size="sm">
        <TimesIcon aria-label="forbidden" />
    </Icon>
);
const permittedIcon = (
    <Icon color="var(--pf-global--success-color--100)" size="sm">
        <CheckIcon aria-label="permitted" />
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
                <span className="pf-u-text-nowrap">No roles</span>
            </Flex>
        );
    }

    return (
        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
            {permittedIcon}
            {roleNames.map((roleName) => (
                <Button
                    key={roleName}
                    variant={ButtonVariant.link}
                    isInline
                    component={LinkShim}
                    href={getUserRolePath(roleName)}
                >
                    {roleName}
                </Button>
            ))}
        </Flex>
    );
}

export default RolesForResourceAccess;

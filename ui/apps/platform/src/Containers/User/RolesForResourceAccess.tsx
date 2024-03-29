import React, { ReactElement } from 'react';
import { Button, ButtonVariant, Flex, Icon } from '@patternfly/react-core';
import { CheckIcon, TimesIcon } from '@patternfly/react-icons';

import LinkShim from 'Components/PatternFly/LinkShim';
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

import React, { ReactElement } from 'react';
import { CheckIcon, TimesIcon } from '@patternfly/react-icons';

import { AccessLevel } from 'services/RolesService';

import { Icon } from '@patternfly/react-core';
import { getIsReadAccess, getIsWriteAccess } from './permissionSets.utils';

const forbiddenIcon = (
    <Icon color="var(--pf-v5-global--danger-color--100)" size="sm">
        <TimesIcon aria-label="forbidden" />
    </Icon>
);
const permittedIcon = (
    <Icon color="var(--pf-v5-global--success-color--100)" size="sm">
        <CheckIcon aria-label="permitted" />
    </Icon>
);

export type AccessIconProps = {
    accessLevel: AccessLevel;
};

export function ReadAccessIcon({ accessLevel }: AccessIconProps): ReactElement {
    return getIsReadAccess(accessLevel) ? permittedIcon : forbiddenIcon;
}

export function WriteAccessIcon({ accessLevel }: AccessIconProps): ReactElement {
    return getIsWriteAccess(accessLevel) ? permittedIcon : forbiddenIcon;
}

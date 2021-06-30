import React, { ReactElement } from 'react';
import { CheckIcon, TimesIcon } from '@patternfly/react-icons';

import { AccessLevel } from 'services/RolesService';

import { getIsReadAccess, getIsWriteAccess } from './permissionSets.utils';

const forbiddenIcon = (
    <TimesIcon aria-label="forbidden" color="var(--pf-global--danger-color--100)" size="sm" />
);
const permittedIcon = (
    <CheckIcon aria-label="permitted" color="var(--pf-global--success-color--100)" size="sm" />
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

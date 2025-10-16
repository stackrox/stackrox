import type { ReactElement } from 'react';
import { CheckIcon, TimesIcon } from '@patternfly/react-icons';

import type { AccessLevel } from 'services/RolesService';

import { Icon } from '@patternfly/react-core';
import { getIsReadAccess, getIsWriteAccess } from './permissionSets.utils';

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

export type AccessIconProps = {
    accessLevel: AccessLevel;
};

export function ReadAccessIcon({ accessLevel }: AccessIconProps): ReactElement {
    return getIsReadAccess(accessLevel) ? permittedIcon : forbiddenIcon;
}

export function WriteAccessIcon({ accessLevel }: AccessIconProps): ReactElement {
    return getIsWriteAccess(accessLevel) ? permittedIcon : forbiddenIcon;
}
